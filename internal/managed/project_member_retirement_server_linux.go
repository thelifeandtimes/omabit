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

type projectMemberRetirementBackend interface {
	OpenProjectMemberRetirement(context.Context, runtimequalify.ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error)
}

func (realBackend) OpenProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return runtimequalify.OpenProjectMemberRetirement(ctx, s)
}

type projectResumedMemberRetirementBackend interface {
	OpenResumedProjectMemberRetirement(context.Context, runtimequalify.ProjectResumedMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error)
}

func (realBackend) OpenResumedProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectResumedMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return runtimequalify.OpenResumedProjectMemberRetirement(ctx, s)
}

// Explicit public retirement uses the common resource/update gate. No startup
// action or implicit removal follows completion of an exit observation.
func (r *Server) queueProjectMemberRetirement(actor, project, instance string, req projectMemberRetirementRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberRetirementBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member retirement is unavailable on this runner")
	}
	if exit, e := loadProjectMemberStopExit(r.s.db, req.ExitOperationID); e == nil {
		if _, e := loadProjectMemberContextReadiness(r.s.db, exit.Stop.ReadinessOperationID); e == nil {
			if _, ok := b.(projectCurrentMemberRetirementBackend); !ok {
				return protocol.Operation{}, false, bad("current retirement is unavailable on this runner")
			}
		}
		if _, e := loadProjectMemberResumedReadiness(r.s.db, exit.Stop.ReadinessOperationID); e == nil {
			if _, ok := b.(projectResumedMemberRetirementBackend); !ok {
				return protocol.Operation{}, false, bad("replacement retirement is unavailable on this runner")
			}
		}
	}
	op, replayed, err := r.s.acceptProjectMemberRetirement(actor, project, instance, req)
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
			runErr = r.executeProjectMemberRetirement(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberRetirement(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberRetirement(b projectMemberRetirementBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberRetirements(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	j, err := loadProjectMemberRetirement(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, expected, container, err := initialMemberRetirementSelection(r.s.db, op.ProjectID, op.InstanceID, j.ExitOperationID, op.ID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "retire-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.ContainerID != container || !reflect.DeepEqual(j.Retirement, expected) {
		return errors.New("member retirement is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, p.ID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	source, err := projectMemberLifecycleSource(r.s, p, j.Retirement.Exit.Stop)
	if err != nil {
		return err
	}
	var initial projectnetwork.MemberRetirement
	var driver projectnetwork.MemberRetirementDriver
	if source.CurrentStop != nil {
		current, ok := b.(projectCurrentMemberRetirementBackend)
		if !ok {
			return errors.New("current retirement backend unavailable")
		}
		initial, driver, err = current.OpenCurrentProjectMemberRetirement(r.ctx, runtimequalify.ProjectCurrentMemberRetirementSelection{Stop: *source.CurrentStop, Exit: j.Retirement.Exit, OperationID: op.ID})
	} else if source.ResumedStop != nil {
		current, ok := b.(projectResumedMemberRetirementBackend)
		if !ok {
			return errors.New("replacement retirement backend unavailable")
		}
		initial, driver, err = current.OpenResumedProjectMemberRetirement(r.ctx, runtimequalify.ProjectResumedMemberRetirementSelection{Stop: *source.ResumedStop, Exit: j.Retirement.Exit, OperationID: op.ID})
	} else {
		initial, driver, err = b.OpenProjectMemberRetirement(r.ctx, runtimequalify.ProjectMemberRetirementSelection{Stop: source.Stop, Exit: j.Retirement.Exit, OperationID: op.ID})
	}

	if err != nil {
		return err
	}
	if !reflect.DeepEqual(initial, j.Retirement) {
		return errors.New("retirement adapter changed accepted intent")
	}
	return r.s.runProjectMemberRetirement(r.ctx, op.ID, driver)
}
