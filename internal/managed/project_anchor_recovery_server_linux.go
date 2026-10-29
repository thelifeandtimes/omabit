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

type projectAnchorRecoveryBackend interface {
	OpenProjectAnchorRecoveryReview(context.Context, runtimequalify.ProjectAnchorRecoveryReviewSelection) (projectnetwork.AnchorRecoveryObserver, error)
	OpenProjectAnchorDrain(context.Context, runtimequalify.ProjectAnchorDrainSelection) (projectnetwork.AnchorDrainDriver, error)
	OpenProjectAnchorNamespace(context.Context, runtimequalify.ProjectAnchorNamespaceSelection) (projectnetwork.AnchorNamespaceDriver, error)
	OpenProjectAnchorRetirement(context.Context, runtimequalify.ProjectAnchorRetirementSelection) (projectnetwork.AnchorRetirementDriver, error)
	PlanProjectAnchorReplacement(context.Context, runtimequalify.ProjectAnchorReplacementPlanSelection) (projectnetwork.AnchorReplacement, error)
	OpenProjectAnchorReplacement(context.Context, runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementDriver, error)
	OpenProjectAnchorRecoveryReadiness(context.Context, runtimequalify.ProjectAnchorRecoveryReadinessSelection) (projectnetwork.AnchorRecoveryReadinessDriver, error)
}

func (realBackend) OpenProjectAnchorRecoveryReview(ctx context.Context, s runtimequalify.ProjectAnchorRecoveryReviewSelection) (projectnetwork.AnchorRecoveryObserver, error) {
	return runtimequalify.OpenProjectAnchorRecoveryReview(ctx, s)
}
func (realBackend) OpenProjectAnchorDrain(ctx context.Context, s runtimequalify.ProjectAnchorDrainSelection) (projectnetwork.AnchorDrainDriver, error) {
	return runtimequalify.OpenProjectAnchorDrain(ctx, s)
}
func (realBackend) OpenProjectAnchorNamespace(ctx context.Context, s runtimequalify.ProjectAnchorNamespaceSelection) (projectnetwork.AnchorNamespaceDriver, error) {
	return runtimequalify.OpenProjectAnchorNamespace(ctx, s)
}
func (realBackend) OpenProjectAnchorRetirement(ctx context.Context, s runtimequalify.ProjectAnchorRetirementSelection) (projectnetwork.AnchorRetirementDriver, error) {
	return runtimequalify.OpenProjectAnchorRetirement(ctx, s)
}
func (realBackend) PlanProjectAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementPlanSelection) (projectnetwork.AnchorReplacement, error) {
	return runtimequalify.PlanProjectAnchorReplacement(ctx, s)
}
func (realBackend) OpenProjectAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementDriver, error) {
	return runtimequalify.OpenProjectAnchorReplacement(ctx, s)
}
func (realBackend) OpenProjectAnchorRecoveryReadiness(ctx context.Context, s runtimequalify.ProjectAnchorRecoveryReadinessSelection) (projectnetwork.AnchorRecoveryReadinessDriver, error) {
	return runtimequalify.OpenProjectAnchorRecoveryReadiness(ctx, s)
}

// Shared public-command and qualification queue. Acceptance/replay shares management-update
// and worker admission. Neither startup nor actor replay schedules recovery.
func (r *Server) queueProjectAnchorRecovery(actor, project string, req projectAnchorRecoveryRequest) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is stopping or state storage needs review")
	}
	if r.updateFence.Held {
		return protocol.Operation{}, false, conflict("management_quiesced", "management update holds admission; resume the activation journal")
	}
	b, ok := r.b.(projectAnchorRecoveryBackend)
	if !ok {
		return protocol.Operation{}, false, bad("coordinated project recovery is unavailable on this runner")
	}
	op, replayed, err := r.s.acceptProjectAnchorRecovery(actor, project, req)
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
			runErr = r.executeProjectAnchorRecovery(b, op)
		case <-r.ctx.Done():
			runErr = r.ctx.Err()
		}
		if runErr != nil {
			j, err := loadProjectAnchorRecovery(r.s.db, op.ID)
			if err == nil {
				err = r.s.interruptAnchorRecovery(op.ID, j.Journal.Phase == "review-planned" || j.Journal.Phase == "reviewing")
			}
			if err != nil {
				r.failed.Store(true)
			}
		}
	}()
	return op, false, nil
}
func recoveryRuntimeSource(s *store, j projectAnchorRecoveryJob) runtimequalify.ProjectAnchorRecoveryReviewSelection {
	h := j.Source.Clone()
	a := h.Launches[0].Anchor.Prepared
	return runtimequalify.ProjectAnchorRecoveryReviewSelection{Project: runtimequalify.ProjectPreparationSelection{RunnerID: s.runnerID, ProjectID: j.ProjectID, OperationID: a.OperationID, Revision: a.ProjectRevision, Generation: a.Generation, Plan: h.Plan}, Epochs: h.Epochs, Launches: h.Launches, Resumes: h.Resumes, Originals: h.Originals, Current: h.Current}
}
func (r *Server) executeProjectAnchorRecovery(b projectAnchorRecoveryBackend, op protocol.Operation) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if err := projectMaintenanceClear(r.s); err != nil {
		return err
	}
	j, err := loadProjectAnchorRecovery(r.s.db, op.ID)
	if err != nil {
		return err
	}
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		return err
	}
	if actual.Kind != "recover-project-anchor" || actual.ProjectID != op.ProjectID || actual.Actor != op.Actor || actual.Status != "queued" || j.Journal.Phase != "review-planned" {
		return errors.New("recovery is not freshly queued; retained uncertainty requires separate observation")
	}
	return r.driveProjectAnchorRecovery(b, nil, j, func(expected uint64, phase string, u projectnetwork.AnchorRecoveryJournal) (projectAnchorRecoveryJob, error) {
		return r.s.advanceProjectAnchorRecovery(op.ID, expected, phase, u)
	})
}

type projectCreatedAnchorReplacementBackend interface {
	OpenProjectCreatedAnchorReplacement(context.Context, runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementStartDriver, error)
}

func (realBackend) OpenProjectCreatedAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementStartDriver, error) {
	return runtimequalify.OpenProjectCreatedAnchorReplacement(ctx, s)
}

// The continuation worker supplies its own atomic parent/receipt committer.
// A nil created backend preserves ordinary fresh-queue admission.
func (r *Server) driveProjectAnchorRecovery(b projectAnchorRecoveryBackend, created projectCreatedAnchorReplacementBackend, j projectAnchorRecoveryJob, advance func(uint64, string, projectnetwork.AnchorRecoveryJournal) (projectAnchorRecoveryJob, error)) error {
	var err error
	save := func(phase string, u projectnetwork.AnchorRecoveryJournal) error {
		next, err := advance(j.Journal.Sequence, phase, u)
		if err == nil {
			j = next
		}
		return err
	}
	for {
		if err = r.ctx.Err(); err != nil {
			return err
		}
		source := recoveryRuntimeSource(r.s, j)
		switch j.Journal.Phase {
		case "review-planned":
			if err = save("reviewing", projectnetwork.AnchorRecoveryJournal{}); err != nil {
				return err
			}
		case "reviewing":
			d, err := b.OpenProjectAnchorRecoveryReview(r.ctx, source)
			if err != nil {
				return err
			}
			proof, err := projectnetwork.CollectAnchorRecoveryHistoryReview(r.ctx, j.Source, d, time.Now)
			if err != nil {
				return err
			}
			if err = save("reviewed", projectnetwork.AnchorRecoveryJournal{Review: &proof}); err != nil {
				return err
			}
		case "reviewed":
			if err = save("draining", projectnetwork.AnchorRecoveryJournal{}); err != nil {
				return err
			}
		case "draining":
			if j.Journal.Drain.Clean() {
				if err = save("namespace", projectnetwork.AnchorRecoveryJournal{}); err != nil {
					return err
				}
				continue
			}
			d, err := b.OpenProjectAnchorDrain(r.ctx, runtimequalify.ProjectAnchorDrainSelection{Source: source, Drain: j.Journal.Drain.Clone()})
			if err != nil {
				return err
			}
			_, err = projectnetwork.ExecuteNextAnchorDrain(r.ctx, j.Source, *j.Journal.Drain, d, func(expected uint64, next projectnetwork.AnchorDrain) error {
				if j.Journal.Drain == nil || expected != j.Journal.Drain.Sequence {
					return errors.New("stale drain callback")
				}
				return save("draining", projectnetwork.AnchorRecoveryJournal{Drain: &next})
			}, time.Now)
			if err != nil {
				return err
			}
		case "namespace":
			if j.Journal.Namespace.Phase == "observed" {
				if err = save("retiring", projectnetwork.AnchorRecoveryJournal{}); err != nil {
					return err
				}
				continue
			}
			d, err := b.OpenProjectAnchorNamespace(r.ctx, runtimequalify.ProjectAnchorNamespaceSelection{Source: source, Namespace: j.Journal.Namespace.Clone()})
			if err != nil {
				return err
			}
			_, err = projectnetwork.ExecuteAnchorNamespace(r.ctx, j.Source, *j.Journal.Namespace, d, func(expected uint64, next projectnetwork.AnchorNamespace) error {
				if j.Journal.Namespace == nil || expected != j.Journal.Namespace.Sequence {
					return errors.New("stale namespace callback")
				}
				return save("namespace", projectnetwork.AnchorRecoveryJournal{Namespace: &next})
			}, time.Now)
			if err != nil {
				return err
			}
		case "retiring":
			if j.Journal.Retirement.Complete() {
				planned, err := b.PlanProjectAnchorReplacement(r.ctx, runtimequalify.ProjectAnchorReplacementPlanSelection{Source: source, Retirement: j.Journal.Retirement.Clone(), OperationID: j.OperationID, Members: append([]projectnetwork.RecoveryJournalMemberID(nil), j.Journal.IDs.Members...)})
				if err != nil {
					return err
				}
				if err = save("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &planned}); err != nil {
					return err
				}
				continue
			}
			d, err := b.OpenProjectAnchorRetirement(r.ctx, runtimequalify.ProjectAnchorRetirementSelection{Source: source, Retirement: j.Journal.Retirement.Clone()})
			if err != nil {
				return err
			}
			_, err = projectnetwork.ExecuteNextAnchorRetirement(r.ctx, j.Source, *j.Journal.Retirement, d, func(expected uint64, next projectnetwork.AnchorRetirement) error {
				if j.Journal.Retirement == nil || expected != j.Journal.Retirement.Sequence {
					return errors.New("stale retirement callback")
				}
				return save("retiring", projectnetwork.AnchorRecoveryJournal{Retirement: &next})
			}, time.Now)
			if err != nil {
				return err
			}
		case "replacing":
			if j.Journal.Replacement.Running() {
				proof, err := awaitProjectAnchorRecoveryReadiness(r.ctx, b, source, j)
				if err != nil {
					return err
				}
				if err = save("complete", projectnetwork.AnchorRecoveryJournal{Readiness: &proof}); err != nil {
					return err
				}
				continue
			}
			for index, member := range j.Journal.Replacement.Members {
				if member.Phase == "running" {
					continue
				}
				if member.Phase != "created" {
					break
				}
				if created == nil {
					return errors.New("retained Create reply needs explicit continuation authority")
				}
				d, err := created.OpenProjectCreatedAnchorReplacement(r.ctx, runtimequalify.ProjectAnchorReplacementSelection{Source: source, Replacement: j.Journal.Replacement.Clone()})
				if err != nil {
					return err
				}
				_, err = projectnetwork.StartCreatedAnchorReplacement(r.ctx, j.Source, *j.Journal.Replacement, d, func(expected uint64, next projectnetwork.AnchorReplacement) error {
					if j.Journal.Replacement == nil || expected != j.Journal.Replacement.Sequence || j.Journal.Replacement.Members[index].Phase == "running" {
						return errors.New("stale continued replacement callback")
					}
					return save("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
				}, time.Now)
				if err != nil {
					return err
				}
				// Continue the outer driver only after the retained exact ID is running.
				break
			}
			if j.Journal.Replacement.Running() {
				continue
			}
			d, err := b.OpenProjectAnchorReplacement(r.ctx, runtimequalify.ProjectAnchorReplacementSelection{Source: source, Replacement: j.Journal.Replacement.Clone()})
			if err != nil {
				return err
			}
			_, err = projectnetwork.ExecuteNextAnchorReplacement(r.ctx, j.Source, *j.Journal.Replacement, d, func(expected uint64, next projectnetwork.AnchorReplacement) error {
				if j.Journal.Replacement == nil || expected != j.Journal.Replacement.Sequence {
					return errors.New("stale replacement callback")
				}
				return save("replacing", projectnetwork.AnchorRecoveryJournal{Replacement: &next})
			}, time.Now)
			if err != nil {
				return err
			}
		case "complete":
			return nil
		default:
			return errors.New("unsupported recovery worker phase")
		}
	}
}

// A running process may not have opened Eyre yet. Retry only the whole read-only
// readiness bracket, with a new driver and fresh evidence each time. The retained
// replacement checkpoint remains fenced; no Stop/Create/Start can be replayed.
func awaitProjectAnchorRecoveryReadiness(ctx context.Context, b projectAnchorRecoveryBackend, source runtimequalify.ProjectAnchorRecoveryReviewSelection, j projectAnchorRecoveryJob) (projectnetwork.AnchorRecoveryReadiness, error) {
	child, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	var last error
	for {
		if err := child.Err(); err != nil {
			return projectnetwork.AnchorRecoveryReadiness{}, errors.Join(last, err)
		}
		driver, err := b.OpenProjectAnchorRecoveryReadiness(child, runtimequalify.ProjectAnchorRecoveryReadinessSelection{Source: source, Replacement: j.Journal.Replacement.Clone()})
		if err == nil {
			var proof projectnetwork.AnchorRecoveryReadiness
			proof, err = projectnetwork.CollectAnchorRecoveryReadiness(child, j.Source, *j.Journal.Replacement, driver, time.Now)
			if err == nil {
				return proof, nil
			}
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return projectnetwork.AnchorRecoveryReadiness{}, err
		}
		last = err
		timer := time.NewTimer(time.Second)
		select {
		case <-child.Done():
			timer.Stop()
			return projectnetwork.AnchorRecoveryReadiness{}, errors.Join(last, child.Err())
		case <-timer.C:
		}
	}
}
