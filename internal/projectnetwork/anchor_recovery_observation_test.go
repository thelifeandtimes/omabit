package projectnetwork

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type journalObservationFixture struct {
	h              AnchorRecoveryHistory
	j              AnchorRecoveryJournal
	ready          AnchorRecoveryReadiness
	calls          int
	readinessReads int
}

func (f *journalObservationFixture) VerifyEngine(_ context.Context, id string) error {
	f.calls++
	if id != f.ready.Binding.EngineID {
		return errors.New("wrong engine")
	}
	return nil
}
func (f *journalObservationFixture) InspectDrain(context.Context, AnchorDrain) (AnchorDrainObservation, error) {
	f.calls++
	return cloneAnchorDrainObservation(*f.ready.Replacement.Retirement.Namespace.Drain.Members[f.j.Drain.next()].After), nil
}
func (f *journalObservationFixture) InspectNamespace(context.Context, AnchorNamespace) (AnchorDrainObservation, error) {
	f.calls++
	return cloneAnchorDrainObservation(*f.ready.Replacement.Retirement.Namespace.After), nil
}
func (f *journalObservationFixture) InspectRetirement(context.Context, AnchorRetirement) (AnchorRetirementObservation, error) {
	f.calls++
	return cloneAnchorRetirementObservation(*f.ready.Replacement.Retirement.Members[f.j.Retirement.next()].After), nil
}
func (f *journalObservationFixture) InspectReplacement(_ context.Context, _ AnchorReplacement, i int, stage string) (AnchorRetirementObservation, error) {
	f.calls++
	if stage != "running" {
		return AnchorRetirementObservation{}, errors.New("unexpected replacement observation")
	}
	return cloneAnchorRetirementObservation(*f.ready.Replacement.Members[i].After), nil
}
func (f *journalObservationFixture) InspectRecoveryReadiness(context.Context, AnchorReplacement) (AnchorRetirementObservation, error) {
	f.calls++
	f.readinessReads++
	if f.readinessReads == 1 {
		return cloneAnchorRetirementObservation(f.ready.Before), nil
	}
	return cloneAnchorRetirementObservation(f.ready.After), nil
}
func (f *journalObservationFixture) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	f.calls++
	for _, m := range f.ready.Members {
		if m.Endpoint == endpoint {
			return m.Ship, nil
		}
	}
	return "", errors.New("unselected endpoint")
}
func (f *journalObservationFixture) drivers() AnchorRecoveryObservers {
	return AnchorRecoveryObservers{Drain: f, Namespace: f, Retirement: f, Replacement: f, Readiness: f}
}
func observationCut(j AnchorRecoveryJournal, ready AnchorRecoveryReadiness) (bool, time.Time) {
	switch j.Phase {
	case "draining":
		if i := j.Drain.next(); i >= 0 {
			p := j.Drain.Members[i].Phase
			if p == "stop-dispatched" || p == "stop-signalled" {
				return true, ready.Replacement.Retirement.Namespace.Drain.Members[i].After.CapturedAt
			}
		}
	case "namespace":
		p := j.Namespace.Phase
		if p == "observing" || p == "start-dispatched" || p == "start-acknowledged" {
			return true, ready.Replacement.Retirement.Namespace.After.CapturedAt
		}
	case "retiring":
		if i := j.Retirement.next(); i >= 0 {
			p := j.Retirement.Members[i].Phase
			if p == "remove-dispatched" || p == "remove-acknowledged" {
				return true, ready.Replacement.Retirement.Members[i].After.CapturedAt
			}
		}
	case "replacing":
		if j.Replacement.Running() {
			return true, ready.CapturedAt
		}
		i := j.Replacement.next()
		p := j.Replacement.Members[i].Phase
		if p == "start-dispatched" || p == "start-acknowledged" {
			return true, ready.Replacement.Members[i].After.CapturedAt
		}
	}
	return false, ready.CapturedAt
}
func TestRecoveryJournalObservationResolvesOnlyOneRetainedOutcome(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		h, ready := journalReadinessFixture(t, stopped)
		for _, ack := range []bool{false, true} {
			trace := replayRecoveryJournal(t, h, ready, ack)
			for _, j := range trace {
				t.Run(fmt.Sprintf("stopped-%v/ack-%v/seq-%d-%s", stopped, ack, j.Sequence, j.Phase), func(t *testing.T) {
					f := &journalObservationFixture{h: h, j: j, ready: ready}
					allowed, now := observationCut(j, ready)
					before, source := j.Clone(), h.Clone()
					commits := 0
					result, err := ObserveAnchorRecoveryCheckpoint(context.Background(), h, j, f.drivers(), func(expected uint64, next AnchorRecoveryJournal) error {
						commits++
						if expected != j.Sequence || next.Sequence != expected+1 {
							t.Fatal("nonserial observation")
						}
						if e := next.Validate(h); e != nil {
							t.Fatal(e)
						}
						next.IDs.Members[0].OperationID = "caller-mutation"
						return nil
					}, func() time.Time { return now })
					if !reflect.DeepEqual(j, before) || !reflect.DeepEqual(h, source) {
						t.Fatal("observer changed retained authority")
					}
					target, targetErr := RecoveryCheckpointObservationTarget(h, j)
					if allowed && (targetErr != nil || target.Kind == "") || !allowed && targetErr == nil {
						t.Fatal("incorrect observation classification", target, targetErr)
					}
					if allowed {
						if err != nil || commits != 1 || f.calls == 0 || result.Sequence != j.Sequence+1 || result.Validate(h) != nil {
							t.Fatal("observation failed", err, commits, f.calls)
						}
						if e := RecoveryJournalObservedTransition(h, j, result); e != nil {
							t.Fatal(e)
						}
						if j.Phase == "replacing" && j.Replacement.Running() {
							if result.Phase != "complete" {
								t.Fatal("missing complete readiness")
							}
						} else if result.Phase != j.Phase {
							t.Fatal("observation advanced to another effect phase")
						}
					} else if err == nil || commits != 0 || f.calls != 0 || !reflect.DeepEqual(result, before) {
						t.Fatal("observation admitted planned/unknown/completed work", err, commits, f.calls)
					}
				})
			}
		}
	}
}
func TestRecoveryJournalObservationFailuresPreserveOriginal(t *testing.T) {
	h, ready := journalReadinessFixture(t, false)
	trace := replayRecoveryJournal(t, h, ready, false)
	var pending AnchorRecoveryJournal
	for _, j := range trace {
		if j.Phase == "draining" && j.Drain.Members[0].Phase == "stop-dispatched" {
			pending = j
			break
		}
	}
	for _, fault := range []string{"commit", "cancelled", "stale-observation", "foreign-history", "missing-observer"} {
		t.Run(fault, func(t *testing.T) {
			source := h.Clone()
			f := &journalObservationFixture{h: h, j: pending, ready: ready}
			_, now := observationCut(pending, ready)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			drivers := f.drivers()
			switch fault {
			case "cancelled":
				cancel()
			case "stale-observation":
				now = now.Add(time.Hour)
			case "foreign-history":
				source.Plan.Members[0].HTTPPort++
			case "missing-observer":
				drivers.Drain = nil
			}
			commits := 0
			result, err := ObserveAnchorRecoveryCheckpoint(ctx, source, pending, drivers, func(uint64, AnchorRecoveryJournal) error { commits++; return errors.New("synthetic commit failure") }, func() time.Time { return now })
			if err == nil || !reflect.DeepEqual(result, pending) {
				t.Fatal("uncertain observation changed checkpoint", err)
			}
			if fault == "commit" {
				if commits != 1 {
					t.Fatal("commit was not attempted")
				}
			} else if commits != 0 {
				t.Fatal("invalid observation reached commit")
			}
		})
	}
}

func TestRecoveryObservedTransitionCannotInventAcknowledgement(t *testing.T) {
	h, ready := journalReadinessFixture(t, false)
	trace := replayRecoveryJournal(t, h, ready, true)
	for i, j := range trace {
		if i+1 == len(trace) {
			break
		}
		next := trace[i+1]
		_, err := RecoveryCheckpointObservationTarget(h, j)
		if err != nil {
			continue
		}
		observed := j.Phase == "draining" && next.Drain != nil && next.Drain.Members[j.Drain.next()].Phase == "exited" || j.Phase == "namespace" && next.Namespace != nil && next.Namespace.Phase == "observed" || j.Phase == "retiring" && next.Retirement != nil && next.Retirement.Members[j.Retirement.next()].Phase == "absent" || j.Phase == "replacing" && (next.Phase == "complete" || next.Replacement != nil && next.Replacement.Members[j.Replacement.next()].Phase == "running")
		if (RecoveryJournalObservedTransition(h, j, next) == nil) != observed {
			t.Fatal("observation confused with acknowledgement", j.Phase, j.Sequence)
		}
	}
}
