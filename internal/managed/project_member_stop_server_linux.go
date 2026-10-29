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

type projectMemberStopBackend interface {
	OpenProjectMemberStop(context.Context, runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error)
}

type projectResumedMemberStopBackend interface {
	OpenResumedProjectMemberStop(context.Context, runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopDriver, error)
}

func (realBackend) OpenResumedProjectMemberStop(ctx context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return runtimequalify.OpenResumedProjectMemberStop(ctx, s)
}

func (realBackend) OpenProjectMemberStop(ctx context.Context, s runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return runtimequalify.OpenProjectMemberStop(ctx, s)
}

// Explicit public Stop uses the common work/update gates to serialize its
// observation and signal. No startup task schedules or retries a Stop.
func (r *Server) queueProjectMemberStop(actor, project, instance string, req projectMemberStopRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberStopBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member Stop is unavailable on this runner")
	}
	if _, err := loadProjectMemberResumedReadiness(r.s.db, req.ReadinessOperationID); err == nil {
		if _, ok := b.(projectResumedMemberStopBackend); !ok {
			return protocol.Operation{}, false, bad("replacement member Stop is unavailable on this runner")
		}
	}
	if _, err := loadProjectMemberContextReadiness(r.s.db, req.ReadinessOperationID); err == nil {
		if _, ok := b.(projectCurrentMemberStopBackend); !ok {
			return protocol.Operation{}, false, bad("current member Stop is unavailable on this runner")
		}
	}
	op, replayed, err := r.s.acceptProjectMemberStop(actor, project, instance, req)
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
			runErr = r.executeProjectMemberStop(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptMemberStop(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberStop(b projectMemberStopBackend, op protocol.Operation) error {
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
	j, err := loadProjectMemberStop(r.s.db, op.ID)
	if err != nil {
		return err
	}
	p, expected, container, err := memberStopSelection(r.s.db, op.ProjectID, op.InstanceID, j.Stop.ReadinessOperationID, op.ID, true)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.Kind != "stop-project-member" || actual.ProjectID != p.ID || actual.InstanceID != op.InstanceID || actual.ConfigRevision != p.Revision || j.ContainerID != container || !reflect.DeepEqual(j.Stop, expected) {
		return errors.New("member Stop is no longer freshly queued")
	}
	_, launches, _, _, err := projectReadinessPrerequisites(r.s.db, p.ID, op.InstanceID)
	if err != nil {
		return err
	}
	resumes, err := completedProjectMemberResumes(r.s.db, p.ID)
	if err != nil {
		return err
	}
	jobs, err := projectMemberStopJobs(r.s.db)
	if err != nil {
		return err
	}
	var peers []projectnetwork.MemberStop
	for _, peer := range jobs {
		if peer.ProjectID != p.ID || peer.OperationID == op.ID {
			continue
		}
		current := false
		for _, m := range expected.Binding.Members {
			if m.InstanceID == peer.InstanceID && m.Container.ID == peer.ContainerID {
				current = true
			}
		}
		if !current {
			if stopWasReplaced(resumes, peer.Stop) {
				continue
			}
			return errors.New("member Stop history contains an unaccounted old container")
		}
		if peer.Stop.Phase != "stopped" {
			return errors.New("another member Stop remains unresolved")
		}
		peers = append(peers, peer.Stop)
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	a := launches[0].Anchor.Prepared
	projectSelection := runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, OperationID: a.OperationID, Revision: p.Revision, Generation: a.Generation, Plan: p.Plan}
	var driver projectnetwork.MemberStopDriver
	ready, readyErr := loadProjectMemberReadiness(r.s.db, j.Stop.ReadinessOperationID)
	if currentReady, e := loadProjectMemberContextReadiness(r.s.db, j.Stop.ReadinessOperationID); e == nil {
		current, ok := b.(projectCurrentMemberStopBackend)
		if !ok {
			return errors.New("current member Stop backend unavailable")
		}
		src, _, e := contextReadinessJobSource(r.s.db, currentReady)
		if e != nil {
			return e
		}
		driver, err = current.OpenCurrentProjectMemberStop(r.ctx, runtimequalify.ProjectCurrentMemberStopSelection{ReadinessSource: runtimequalify.ProjectCurrentReadinessSelection{Project: projectSelection, Launches: launches, Resumes: src.Resumes, Original: src.Original, RecoveryHistory: src.RecoveryHistory}, Readiness: *currentReady.Evidence, Stop: j.Stop, StoppedPeers: peers})
	} else if readyErr == nil {
		selection := runtimequalify.ProjectMemberStopSelection{Project: projectSelection, Launches: launches, Readiness: *ready.Evidence, Stop: j.Stop, StoppedPeers: peers}
		driver, err = b.OpenProjectMemberStop(r.ctx, selection)
	} else {
		currentBackend, ok := b.(projectResumedMemberStopBackend)
		if !ok {
			return errors.New("replacement Stop backend unavailable")
		}
		ready, loadErr := loadProjectMemberResumedReadiness(r.s.db, j.Stop.ReadinessOperationID)
		if loadErr != nil || ready.Evidence == nil {
			return errors.New("replacement Stop readiness missing")
		}
		driver, err = currentBackend.OpenResumedProjectMemberStop(r.ctx, runtimequalify.ProjectResumedMemberStopSelection{Project: projectSelection, Launches: launches, Resumes: resumes, Readiness: *ready.Evidence, Stop: j.Stop, StoppedPeers: peers})
	}
	if err != nil {
		return err
	}
	return r.s.runProjectMemberStop(r.ctx, op.ID, driver)
}
