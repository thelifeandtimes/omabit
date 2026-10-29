//go:build linux && amd64

package managed

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectMemberStorageBackend interface {
	OpenProjectMemberStorage(context.Context, projectnetwork.Plan, projectnetwork.MemberStorage) (projectnetwork.MemberStorageDriver, error)
}

func (realBackend) OpenProjectMemberStorage(ctx context.Context, p projectnetwork.Plan, s projectnetwork.MemberStorage) (projectnetwork.MemberStorageDriver, error) {
	return runtimequalify.OpenProjectMemberStorage(ctx, p, s)
}

// The explicit public storage action uses this queue. Startup never schedules it.
func (r *Server) queueProjectMemberStorage(actor, project, instance string, req projectMemberStorageRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberStorageBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member storage preparation is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberStorage(actor, project, instance, req)
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
			runErr = r.executeProjectMemberStorage(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberStorage(op.InstanceID, op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberStorage(b projectMemberStorageBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if _, err := projectReservations(r.s.db); err != nil {
		return err
	}
	if err := validateProjectMemberStorage(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	state, err := loadProjectMemberStorage(r.s.db, op.InstanceID)
	if err != nil {
		return err
	}
	p, err := loadProject(r.s.db, op.ProjectID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.ProjectID != p.ID || actual.InstanceID != state.InstanceID || actual.ConfigRevision != p.Revision || state.OperationID != op.ID || state.Phase != "planned" {
		return errors.New("project member storage is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	driver, err := b.OpenProjectMemberStorage(r.ctx, p.Plan, state)
	if err != nil {
		return err
	}
	return r.s.runProjectMemberStorage(r.ctx, state.InstanceID, op.ID, driver)
}
