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

type projectMemberStopExitBackend interface {
	OpenProjectMemberStopExit(context.Context, runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error)
}

func (realBackend) OpenProjectMemberStopExit(ctx context.Context, s runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return runtimequalify.OpenProjectMemberStopExit(ctx, s)
}

type projectResumedMemberStopExitBackend interface {
	OpenResumedProjectMemberStopExit(context.Context, runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopExitDriver, error)
}

func (realBackend) OpenResumedProjectMemberStopExit(ctx context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return runtimequalify.OpenResumedProjectMemberStopExit(ctx, s)
}

// Explicit read-only recovery observation. No automatic startup job is
// introduced. The common resource/update fence protects the fresh observation;
// neither this queue nor its driver can replay the original Stop signal.
func (r *Server) queueProjectMemberStopExit(actor, project, instance string, req projectMemberStopExitRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberStopExitBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member Stop exit is unavailable on this runner")
	}
	if stop, e := loadProjectMemberStop(r.s.db, req.StopOperationID); e == nil {
		if _, e := loadProjectMemberContextReadiness(r.s.db, stop.Stop.ReadinessOperationID); e == nil {
			if _, ok := b.(projectCurrentMemberStopExitBackend); !ok {
				return protocol.Operation{}, false, bad("current Stop exit is unavailable on this runner")
			}
		}
		if _, e := loadProjectMemberResumedReadiness(r.s.db, stop.Stop.ReadinessOperationID); e == nil {
			if _, ok := b.(projectResumedMemberStopExitBackend); !ok {
				return protocol.Operation{}, false, bad("replacement Stop exit is unavailable on this runner")
			}
		}
	}
	op, replayed, err := r.s.acceptProjectMemberStopExit(actor, project, instance, req)
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
			runErr = r.executeProjectMemberStopExit(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberStopExit(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberStopExit(b projectMemberStopExitBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberStopExit(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberStopExit(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, retained, err := projectStopExitPrerequisites(r.s.db, op.ProjectID, op.InstanceID, j.StopOperationID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "observe-project-member-stop-exit" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.Phase != "planned" || j.Evidence != nil || !reflect.DeepEqual(j.Stop, retained) {
		return errors.New("Stop exit observation is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, p.ID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	source, err := projectMemberLifecycleSource(r.s, p, retained)
	if err != nil {
		return err
	}
	var driver projectnetwork.MemberStopExitDriver
	if source.CurrentStop != nil {
		current, ok := b.(projectCurrentMemberStopExitBackend)
		if !ok {
			return errors.New("current Stop exit backend unavailable")
		}
		driver, err = current.OpenCurrentProjectMemberStopExit(r.ctx, *source.CurrentStop)
	} else if source.ResumedStop != nil {
		current, ok := b.(projectResumedMemberStopExitBackend)
		if !ok {
			return errors.New("replacement Stop exit backend unavailable")
		}
		driver, err = current.OpenResumedProjectMemberStopExit(r.ctx, *source.ResumedStop)
	} else {
		driver, err = b.OpenProjectMemberStopExit(r.ctx, source.Stop)
	}

	if err != nil {
		return err
	}
	return r.s.runProjectMemberStopExit(r.ctx, op.ID, driver)
}
