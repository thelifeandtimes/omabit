//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectRecoveryObservationSelection struct {
	Source     runtimequalify.ProjectAnchorRecoveryReviewSelection
	History    projectnetwork.AnchorRecoveryHistory
	Checkpoint projectnetwork.AnchorRecoveryJournal
	Target     projectnetwork.RecoveryObservationTarget
}
type projectRecoveryObservationBackend interface {
	OpenProjectRecoveryObservation(context.Context, projectRecoveryObservationSelection) (projectnetwork.AnchorRecoveryObservers, error)
}

// Return only the observation interfaces. Constructing these existing adapters
// does not dispatch lifecycle effects; the worker cannot invoke their mutations.
func (realBackend) OpenProjectRecoveryObservation(ctx context.Context, s projectRecoveryObservationSelection) (projectnetwork.AnchorRecoveryObservers, error) {
	var out projectnetwork.AnchorRecoveryObservers
	target, err := projectnetwork.RecoveryCheckpointObservationTarget(s.History, s.Checkpoint)
	if err != nil || target != s.Target {
		return out, errors.New("recovery observation target changed")
	}
	switch target.Kind {
	case "member-exit":
		d, err := runtimequalify.OpenProjectAnchorDrain(ctx, runtimequalify.ProjectAnchorDrainSelection{Source: s.Source, Drain: s.Checkpoint.Drain.Clone()})
		if err != nil {
			return out, err
		}
		out.Drain = struct {
			projectnetwork.AnchorDrainObserver
		}{d}
	case "namespace":
		d, err := runtimequalify.OpenProjectAnchorNamespace(ctx, runtimequalify.ProjectAnchorNamespaceSelection{Source: s.Source, Namespace: s.Checkpoint.Namespace.Clone()})
		if err != nil {
			return out, err
		}
		out.Namespace = struct {
			projectnetwork.AnchorNamespaceObserver
		}{d}
	case "member-absence":
		d, err := runtimequalify.OpenProjectAnchorRetirement(ctx, runtimequalify.ProjectAnchorRetirementSelection{Source: s.Source, Retirement: s.Checkpoint.Retirement.Clone()})
		if err != nil {
			return out, err
		}
		out.Retirement = struct {
			projectnetwork.AnchorRetirementObserver
		}{d}
	case "member-running":
		d, err := runtimequalify.OpenProjectAnchorReplacement(ctx, runtimequalify.ProjectAnchorReplacementSelection{Source: s.Source, Replacement: s.Checkpoint.Replacement.Clone()})
		if err != nil {
			return out, err
		}
		out.Replacement = struct {
			projectnetwork.AnchorReplacementObserver
		}{d}
	case "readiness":
		d, err := runtimequalify.OpenProjectAnchorRecoveryReadiness(ctx, runtimequalify.ProjectAnchorRecoveryReadinessSelection{Source: s.Source, Replacement: s.Checkpoint.Replacement.Clone()})
		if err != nil {
			return out, err
		}
		out.Readiness = d
	default:
		return out, errors.New("unsupported recovery observation")
	}
	return out, nil
}

// Shared public-command and qualification queue. Replay and startup never schedule a read or
// mutation. Each fresh request observes one exact retained checkpoint, sharing
// the lifecycle worker and management-update fence.
func (r *Server) queueProjectRecoveryObservation(actor, project string, req projectRecoveryObservationRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectRecoveryObservationBackend)
	if !ok {
		return protocol.Operation{}, false, bad("recovery observation is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectRecoveryObservation(actor, project, req)
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
			runErr = r.executeProjectRecoveryObservation(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			if err := r.s.interruptRecoveryObservation(op.ID); err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func (r *Server) executeProjectRecoveryObservation(b projectRecoveryObservationBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := projectMaintenanceClear(r.s); err != nil {
		return err
	}
	if err := validateProjectAnchorRecoveries(r.s.db); err != nil {
		return err
	}
	if err := validateProjectRecoveryObservations(r.s.db); err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	if actual.Kind != "observe-project-recovery" || actual.ProjectID != op.ProjectID || actual.Actor != op.Actor || actual.Status != "queued" {
		return errors.New("recovery observation is not freshly queued")
	}
	j, err := r.s.beginProjectRecoveryObservation(op.ID)
	if err != nil {
		return err
	}
	parent, err := loadProjectAnchorRecovery(r.s.db, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	if err = r.ctx.Err(); err != nil {
		return err
	}
	d, err := b.OpenProjectRecoveryObservation(r.ctx, projectRecoveryObservationSelection{Source: recoveryRuntimeSource(r.s, parent), History: parent.Source.Clone(), Checkpoint: j.Before.Clone(), Target: j.Target})
	if err != nil {
		return err
	}
	_, err = projectnetwork.ObserveAnchorRecoveryCheckpoint(r.ctx, parent.Source, j.Before, d, func(expected uint64, next projectnetwork.AnchorRecoveryJournal) error {
		if err := r.ctx.Err(); err != nil {
			return err
		}
		return r.s.completeProjectRecoveryObservation(op.ID, expected, next)
	}, func() time.Time { return time.Now().UTC() })
	return err
}
