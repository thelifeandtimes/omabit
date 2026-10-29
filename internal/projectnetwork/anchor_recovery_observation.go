package projectnetwork

import (
	"context"
	"errors"
	"time"
)

// These interfaces expose observation only. Resolving a retained checkpoint never
// authorizes the next mutation; an owning store must admit continuation separately.
type AnchorRecoveryObservers struct {
	Drain       AnchorDrainObserver
	Namespace   AnchorNamespaceObserver
	Retirement  AnchorRetirementObserver
	Replacement AnchorReplacementObserver
	Readiness   AnchorRecoveryReadinessDriver
}
type AnchorRecoveryCommit func(expected uint64, next AnchorRecoveryJournal) error

// ObserveAnchorRecoveryCheckpoint resolves at most one uncertain checkpoint.
// Unknown Create outcomes, fresh planned work and completed records are refused.
// The complete source and journal must already be bound to actual store ownership.
func ObserveAnchorRecoveryCheckpoint(ctx context.Context, h AnchorRecoveryHistory, j AnchorRecoveryJournal, d AnchorRecoveryObservers, commit AnchorRecoveryCommit, now func() time.Time) (AnchorRecoveryJournal, error) {
	if j.Validate(h) != nil || commit == nil || now == nil {
		return j, errors.New("recovery observation requires retained authority and durable commit")
	}
	h = h.Clone()
	current := j.Clone()
	save := func(phase string, u AnchorRecoveryJournal) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		next, err := current.Advance(h, current.Sequence, phase, u)
		if err != nil {
			return err
		}
		if err = commit(current.Sequence, next.Clone()); err != nil {
			return err
		}
		current = next
		return nil
	}
	var err error
	switch current.Phase {
	case "draining":
		_, err = ObserveAnchorDrainExit(ctx, h, *current.Drain, d.Drain, func(expected uint64, next AnchorDrain) error {
			if expected != current.Drain.Sequence {
				return errors.New("stale observed drain")
			}
			return save("draining", AnchorRecoveryJournal{Drain: &next})
		}, now)
	case "namespace":
		_, err = ObserveAnchorNamespace(ctx, h, *current.Namespace, d.Namespace, func(expected uint64, next AnchorNamespace) error {
			if expected != current.Namespace.Sequence {
				return errors.New("stale observed namespace")
			}
			return save("namespace", AnchorRecoveryJournal{Namespace: &next})
		}, now)
	case "retiring":
		_, err = ObserveAnchorRetirementAbsence(ctx, h, *current.Retirement, d.Retirement, func(expected uint64, next AnchorRetirement) error {
			if expected != current.Retirement.Sequence {
				return errors.New("stale observed retirement")
			}
			return save("retiring", AnchorRecoveryJournal{Retirement: &next})
		}, now)
	case "replacing":
		if current.Replacement.Running() {
			var ready AnchorRecoveryReadiness
			ready, err = CollectAnchorRecoveryReadiness(ctx, h, *current.Replacement, d.Readiness, now)
			if err == nil {
				err = save("complete", AnchorRecoveryJournal{Readiness: &ready})
			}
		} else {
			_, err = ObserveAnchorReplacementRunning(ctx, h, *current.Replacement, d.Replacement, func(expected uint64, next AnchorReplacement) error {
				if expected != current.Replacement.Sequence {
					return errors.New("stale observed replacement")
				}
				return save("replacing", AnchorRecoveryJournal{Replacement: &next})
			}, now)
		}
	default:
		err = errors.New("recovery checkpoint has no pending read-only outcome; preserve intent and review separately")
	}
	return current.Clone(), err
}
