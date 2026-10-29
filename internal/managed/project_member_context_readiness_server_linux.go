//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectMemberContextReadinessBackend interface {
	OpenCurrentProjectReadiness(context.Context, runtimequalify.ProjectCurrentReadinessSelection) (projectnetwork.CurrentReadinessDriver, error)
}

func (realBackend) OpenCurrentProjectReadiness(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection) (projectnetwork.CurrentReadinessDriver, error) {
	return runtimequalify.OpenCurrentProjectReadiness(ctx, s)
}

// Explicit public current observation has no automatic startup dispatch.
// This shares admission with
// runtime mutations to avoid inspecting across an in-process maintenance action.
func (r *Server) queueProjectMemberContextReadiness(actor, project, instance string, req projectMemberContextReadinessRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberContextReadinessBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member readiness is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberContextReadiness(actor, project, instance, req)
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
			runErr = r.executeProjectMemberContextReadiness(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberContextReadiness(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberContextReadiness(b projectMemberContextReadinessBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberContextReadiness(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberContextReadiness(r.s.db, op.ID)
	if err != nil {
		return err
	}
	src, err := currentContextUnchanged(r.s.db, j)
	if err != nil {
		return err
	}
	p := src.Project
	if actual.Status != "queued" || actual.Kind != "observe-current-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.Phase != "planned" || j.Evidence != nil || !reflect.DeepEqual(j.Context, src.Context) {
		return errors.New("current readiness is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	a := src.Launches[0].Anchor.Prepared
	selection := runtimequalify.ProjectCurrentReadinessSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, OperationID: a.OperationID, Revision: p.Revision, Generation: a.Generation, Plan: p.Plan}, Launches: src.Launches, Resumes: src.Resumes, Original: src.Original, RecoveryHistory: src.RecoveryHistory}

	driver, err := b.OpenCurrentProjectReadiness(r.ctx, selection)
	if err != nil {
		return err
	}
	return r.s.runProjectMemberContextReadiness(r.ctx, op.ID, driver)
}
