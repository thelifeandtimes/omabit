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

type projectMemberResumedReadinessBackend interface {
	OpenResumedProjectReadiness(context.Context, runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error)
}

func (realBackend) OpenResumedProjectReadiness(ctx context.Context, s runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	return runtimequalify.OpenResumedProjectReadiness(ctx, s)
}

// Explicit public replacement observation has no automatic startup dispatch.
// This shares admission with
// runtime mutations to avoid inspecting across an in-process maintenance action.
func (r *Server) queueProjectMemberResumedReadiness(actor, project, instance string, req projectMemberResumedReadinessRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberResumedReadinessBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member readiness is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberResumedReadiness(actor, project, instance, req)
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
			runErr = r.executeProjectMemberResumedReadiness(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberResumedReadiness(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberResumedReadiness(b projectMemberResumedReadinessBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberResumedReadiness(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberResumedReadiness(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, resumed, binding, err := projectResumedReadinessPrerequisites(r.s.db, op.ProjectID, op.InstanceID, j.ResumeOperationID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "observe-resumed-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.Phase != "planned" || j.Evidence != nil || j.ResumeOperationID != resumed.OperationID || !reflect.DeepEqual(j.Binding, binding) {
		return errors.New("member readiness is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	source, err := projectMemberLifecycleSource(r.s, p, resumed.Retirement.Exit.Stop)
	if err != nil {
		return err
	}
	source.Retirement, source.OperationID = resumed.Retirement, resumed.OperationID
	selection := runtimequalify.ProjectResumedReadinessSelection{Resume: source, Completed: resumed}

	driver, err := b.OpenResumedProjectReadiness(r.ctx, selection)
	if err != nil {
		return err
	}
	return r.s.runProjectMemberResumedReadiness(r.ctx, op.ID, driver)
}
