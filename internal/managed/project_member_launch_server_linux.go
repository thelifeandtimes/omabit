//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"fmt"
	"log"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectMemberLaunchBackend interface {
	OpenProjectMemberLaunch(context.Context, runtimequalify.ProjectMemberLaunchSelection) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error)
}

func (realBackend) OpenProjectMemberLaunch(ctx context.Context, s runtimequalify.ProjectMemberLaunchSelection) (projectnetwork.MemberLaunch, projectnetwork.MemberLaunchDriver, error) {
	initial, driver, err := runtimequalify.OpenProjectMemberLaunch(ctx, s)
	if err != nil {
		return initial, nil, err
	}
	return initial, &memberLaunchDiagnostic{MemberLaunchDriver: driver, project: s.Project.ProjectID, instance: s.Storage.InstanceID, operation: s.OperationID}, nil
}

// The explicit public launch action uses this queue. Startup never schedules it.
func (r *Server) queueProjectMemberLaunch(actor, project, instance string, req projectMemberLaunchRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectMemberLaunchBackend)
	if !ok {
		return protocol.Operation{}, false, bad("project member launch is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectMemberLaunch(actor, project, instance, req)
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
			runErr = r.executeProjectMemberLaunch(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			message := runErr.Error()
			if len(message) > 512 {
				message = message[:512]
			}
			log.Printf("project-member-launch project=%s instance=%s operation=%s failed=%q", op.ProjectID, op.InstanceID, op.ID, message)
			if err := r.s.interruptMemberLaunch(op.InstanceID, op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectMemberLaunch(b projectMemberLaunchBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := validateProjectMemberLaunches(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	job, err := loadProjectMemberLaunch(r.s.db, op.InstanceID)
	if err != nil {
		return err
	}
	p, a, storage, err := memberLaunchPrerequisites(r.s.db, op.ProjectID, op.InstanceID)
	if err != nil {
		return err
	}
	if actual.Status != "queued" || actual.ProjectID != p.ID || actual.InstanceID != storage.InstanceID || actual.ConfigRevision != p.Revision || job.OperationID != op.ID || job.Launch != nil {
		return errors.New("project member launch is no longer freshly queued")
	}
	if err = projectRetirementClear(r.s.db, op.ProjectID, op.ID); err != nil {
		return err
	}
	if err = projectMaintenanceClear(r.s); err != nil {
		return err
	}
	peerBindings, err := projectLaunchPeers(r.s.db, p.ID, op.InstanceID)
	if err != nil {
		return err
	}
	var peers []projectnetwork.MemberLaunch
	for _, binding := range peerBindings {
		j, err := loadProjectMemberLaunch(r.s.db, binding.InstanceID)
		if err != nil {
			return err
		}
		if j.Launch == nil {
			return errors.New("member peer launch is unresolved")
		}
		peers = append(peers, *j.Launch)
	}
	selection := runtimequalify.ProjectMemberLaunchSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: p.ID, OperationID: a.Prepared.OperationID, Revision: p.Revision, Generation: a.Prepared.Generation, Plan: p.Plan}, Anchor: a, Storage: storage, OperationID: op.ID, Peers: peers}
	initial, driver, err := b.OpenProjectMemberLaunch(r.ctx, selection)
	if err != nil {
		return fmt.Errorf("open member launch: %w", err)
	}
	return r.s.runProjectMemberLaunch(r.ctx, op.InstanceID, op.ID, initial, driver)
}
