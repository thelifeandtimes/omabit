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

type projectMemberReadinessBackend interface {
	OpenInitialProjectReadiness(context.Context, runtimequalify.ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error)
}

func (realBackend) OpenInitialProjectReadiness(ctx context.Context, s runtimequalify.ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error) {
	return runtimequalify.AwaitInitialProjectReadiness(ctx, s)
}

// The explicit public initial observation shares admission with runtime mutations
// to avoid inspecting across an in-process maintenance action. No startup retry.
func (r *Server) queueProjectMemberReadiness(actor, project, instance string, req projectMemberReadinessRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberReadinessBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member readiness is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberReadiness(actor, project, instance, req)
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
			runErr = r.executeProjectMemberReadiness(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberReadiness(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberReadiness(b projectMemberReadinessBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberReadiness(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberReadiness(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, launches, binding, target, err := projectReadinessPrerequisites(r.s.db, op.ProjectID, op.InstanceID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "observe-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.Phase != "planned" || j.Evidence != nil || j.LaunchOperationID != target || !reflect.DeepEqual(j.Binding, binding) {
		return errors.New("member readiness is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	a := launches[0].Anchor.Prepared
	selection := runtimequalify.ProjectInitialReadinessSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, OperationID: a.OperationID, Revision: p.Revision, Generation: a.Generation, Plan: p.Plan}, Launches: launches}
	driver, err := b.OpenInitialProjectReadiness(r.ctx, selection)
	if err != nil {
		return err
	}
	return r.s.runProjectMemberReadiness(r.ctx, op.ID, driver)
}
