//go:build linux && amd64

package managed

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectPreparationBackend interface {
	OpenProjectPreparation(context.Context, runtimequalify.ProjectPreparationSelection) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error)
}

func (realBackend) OpenProjectPreparation(ctx context.Context, selection runtimequalify.ProjectPreparationSelection) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
	return runtimequalify.OpenProjectPreparation(ctx, selection)
}

// The explicit public preparation route supplies its authenticated operator.
// This method serializes receipt/worker admission; no startup path schedules it.
func (r *Server) queueProjectPreparation(actor, id string, req projectPrepareRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectPreparationBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project preparation is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectPreparation(actor, id, req)
	if err != nil || replayed {
		return op, replayed, err
	}
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { r.mu.Lock(); r.updateWorkers--; r.mu.Unlock() }()
		var runErr error
		select {
		case r.workGate <- struct{}{}:
			defer func() { <-r.workGate }()
			runErr = r.executeProjectPreparation(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			// Setup failure or cancellation before begin is also durable uncertainty.
			// Known success is never changed by a late failure callback.
			if err := r.s.interruptProjectPreparation(op.ProjectID, op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}

func (r *Server) executeProjectPreparation(b projectPreparationBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	// Check retained work again after acquiring the runner-wide resource gate.
	if err := validateProjectJobs(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectJob(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if actual.ID != op.ID || actual.Status != "queued" || actual.ProjectID != p.ID || actual.ConfigRevision != p.Revision || j.OperationID != op.ID || j.Preparation != nil {
		return errors.New("project preparation is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	selection := runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, Revision: p.Revision, OperationID: op.ID, Generation: j.Generation, Plan: p.Plan}
	prepared, engine, err := b.OpenProjectPreparation(r.ctx, selection)
	if err != nil {
		return err
	}
	return r.s.runProjectPreparation(r.ctx, p.ID, op.ID, prepared, engine)
}
