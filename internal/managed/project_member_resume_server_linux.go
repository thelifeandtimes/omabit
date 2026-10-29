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

type projectMemberResumeBackend interface {
	OpenProjectMemberResume(context.Context, runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error)
}

func (realBackend) OpenProjectMemberResume(ctx context.Context, s runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	return runtimequalify.OpenProjectMemberResume(ctx, s)
}

// Explicit public resume uses the common resource/update gate. No startup
// action or implicit restart follows completion of retirement.
func (r *Server) queueProjectMemberResume(actor, project, instance string, req projectMemberResumeRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberResumeBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member resume is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberResume(actor, project, instance, req)
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
			runErr = r.executeProjectMemberResume(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberResume(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberResume(b projectMemberResumeBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberResumes(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberResume(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, retired, err := memberResumePrerequisites(r.s.db, op.ProjectID, op.InstanceID, j.RetirementOperationID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "resume-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.Resume != nil || !reflect.DeepEqual(j.Retirement, retired) {
		return errors.New("member resume is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, p.ID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	selection, err := projectMemberLifecycleSource(r.s, p, j.Retirement.Exit.Stop)
	if err != nil {
		return err
	}
	selection.Retirement, selection.OperationID = j.Retirement, op.ID

	initial, driver, err := b.OpenProjectMemberResume(r.ctx, selection)
	if err != nil {
		return err
	}
	return r.s.runProjectMemberResume(r.ctx, op.ID, initial, driver)
}
