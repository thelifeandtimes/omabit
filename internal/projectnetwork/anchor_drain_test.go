package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type anchorDrainFixture struct {
	t                                    *testing.T
	history                              AnchorRecoveryHistory
	state                                AnchorDrainObservation
	saved                                AnchorDrain
	now                                  time.Time
	signals, inspects, verifies, commits int
	mode                                 string
	mutate                               func(*AnchorDrainObservation)
	cancel                               context.CancelFunc
}

func newAnchorDrainFixture(t *testing.T, stopped bool) *anchorDrainFixture {
	t.Helper()
	p, l, r, o, current, c, b := anchorRecoveryReviewFixture(t, stopped)
	after := cloneAnchorRecoveryObservation(b)
	after.CapturedAt = after.CapturedAt.Add(time.Second)
	after.Runtime.ObservedAt = after.CapturedAt
	obs := &recoveryObserverFixture{t: t, expected: c, before: b, after: after}
	review, err := CollectAnchorRecoveryReview(context.Background(), p, l, r, o, current, obs, func() time.Time { return after.CapturedAt })
	if err != nil {
		t.Fatal(err)
	}
	h := AnchorRecoveryHistory{Plan: p, Launches: l, Resumes: r, Originals: o, Current: current}
	drain, err := NewAnchorDrain(h, review, "drain-old-members")
	if err != nil {
		t.Fatal(err)
	}
	f := &anchorDrainFixture{t: t, history: h, saved: drain, now: after.CapturedAt}
	f.state = AnchorDrainObservation{Runtime: cloneReadinessObservation(after.Runtime), CapturedAt: after.CapturedAt, AnchorProcess: RecoveryProcessState{StartedAt: after.CapturedAt.Add(-time.Hour)}}
	f.state.AnchorProcess.FinishedAt = f.state.AnchorProcess.StartedAt.Add(-time.Second)
	if stopped {
		f.state.AnchorProcess.FinishedAt = after.CapturedAt.Add(-time.Second)
		f.state.AnchorProcess.ExitCode = 143
	}
	for _, v := range after.Processes {
		f.state.Processes = append(f.state.Processes, AnchorDrainProcess{InstanceID: v.InstanceID, Storage: v.Storage, Process: RecoveryProcessState{StartedAt: v.StartedAt}})
	}
	return f
}
func (f *anchorDrainFixture) clock() time.Time { return f.now }
func (f *anchorDrainFixture) VerifyEngine(ctx context.Context, id string) error {
	f.verifies++
	if id != f.saved.Review.Context.Binding.EngineID {
		f.t.Fatal("changed engine selection")
	}
	if f.mode == "engine" || f.mode == "late-engine" && f.signals > 0 {
		return errors.New("private engine detail")
	}
	return ctx.Err()
}
func (f *anchorDrainFixture) InspectDrain(_ context.Context, s AnchorDrain) (AnchorDrainObservation, error) {
	f.inspects++
	f.now = f.now.Add(time.Second)
	if s.OperationID != f.saved.OperationID {
		f.t.Fatal("changed operation")
	}
	o := cloneAnchorDrainObservation(f.state)
	o.CapturedAt = f.now
	o.Runtime.ObservedAt = f.now
	if f.mutate != nil {
		f.mutate(&o)
	}
	if f.mode == "inspect" {
		return o, errors.New("private inspection")
	}
	if f.mode == "cancel-inspect" {
		f.cancel()
	}
	if f.mode == "mutate-driver-copy" {
		s.Members[0].InstanceID = "foreign"
		s.Review.Context.Binding.Members[0].InstanceID = "foreign"
	}
	return o, nil
}
func (f *anchorDrainFixture) SignalDrainMember(ctx context.Context, s AnchorDrain, index int) error {
	if !reflect.DeepEqual(f.saved, s) || s.Members[index].Phase != "stop-dispatched" {
		f.t.Fatal("signal preceded exact durable intent")
	}
	o, err := f.InspectDrain(ctx, cloneAnchorDrain(s))
	if err != nil || s.checkObservation(f.history, index, o, false, f.now) != nil {
		return errors.New("dispatch reinspection refused")
	}
	if f.mode == "signal-before-effect" {
		return errors.New("private signal detail")
	}
	f.signals++
	member := s.Members[index].InstanceID
	for i := range f.state.Runtime.Members {
		if f.state.Runtime.Members[i].InstanceID == member {
			f.state.Runtime.Members[i].Running = false
			f.state.Runtime.Members[i].Namespace = NamespaceIdentity{}
		}
	}
	for i := range f.state.Processes {
		if f.state.Processes[i].InstanceID == member {
			v := &f.state.Processes[i].Process
			v.FinishedAt = f.now
			if f.mode == "oom" {
				v.OOMKilled = true
				v.ExitCode = 137
			}
			if f.mode == "nonzero" {
				v.ExitCode = 1
			}
		}
	}
	if f.mode == "lost-reply" {
		return errors.New("private signal response lost")
	}
	return nil
}
func (f *anchorDrainFixture) AwaitDrainExit(ctx context.Context, s AnchorDrain, index int) (AnchorDrainObservation, error) {
	if !reflect.DeepEqual(f.saved, s) || s.Members[index].Phase != "stop-signalled" {
		f.t.Fatal("await preceded signal receipt")
	}
	o, err := f.InspectDrain(ctx, cloneAnchorDrain(s))
	if f.mode == "wait-error" {
		err = errors.New("private wait error")
	}
	return o, err
}
func (f *anchorDrainFixture) commit(expected uint64, next AnchorDrain) error {
	f.commits++
	if expected != f.saved.Sequence || next.Sequence != expected+1 || next.Validate(f.history) != nil {
		f.t.Fatal("invalid commit")
	}
	index := f.saved.next()
	phase := next.Members[index].Phase
	if f.mode == "commit-"+phase {
		return errors.New("private commit failed")
	}
	f.saved = cloneAnchorDrain(next)
	if f.mode == "mutate-commit-copy" {
		next.Members[0].InstanceID = "foreign"
		next.Review.Context.Binding.Members[0].InstanceID = "foreign"
	}
	if f.mode == "change-before-signal" && phase == "stop-dispatched" {
		f.state.Processes[index].Process.StartedAt = f.now
	}
	return nil
}
func TestAnchorDrainSerialCleanExitAndImmutableHistory(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, mode := range []string{"", "mutate-driver-copy", "mutate-commit-copy"} {
			t.Run(map[bool]string{true: "stopped", false: "restarted"}[stopped]+"/"+mode, func(t *testing.T) {
				f := newAnchorDrainFixture(t, stopped)
				f.mode = mode
				history := cloneAnchorRecoveryHistory(f.history)
				review := cloneAnchorRecoveryReview(f.saved.Review)
				original := cloneAnchorDrain(f.saved)
				for index := range f.saved.Members {
					prior := cloneAnchorDrain(f.saved)
					got, err := ExecuteNextAnchorDrain(context.Background(), f.history, f.saved, f, f.commit, f.clock)
					if err != nil || got.Validate(f.history) != nil || !reflect.DeepEqual(got, f.saved) || f.signals != index+1 {
						t.Fatal("drain failed", err)
					}
					if got.Members[index].Phase != "exited" || !got.Members[index].SignalAcknowledged || got.Sequence != uint64(3*(index+1)) {
						t.Fatal("wrong committed sequence")
					}
					for k := 0; k < index; k++ {
						if !reflect.DeepEqual(got.Members[k], prior.Members[k]) {
							t.Fatal("previous exit rewritten")
						}
					}
					if !reflect.DeepEqual(f.history, history) || !reflect.DeepEqual(got.Review, review) {
						t.Fatal("historical evidence changed")
					}
				}
				if !f.saved.Clean() || original.Sequence != 0 {
					t.Fatal("clean drain or original intent changed")
				}
				reads := f.inspects
				signals := f.signals
				if _, err := ExecuteNextAnchorDrain(context.Background(), f.history, f.saved, f, f.commit, f.clock); err == nil || f.inspects != reads || f.signals != signals {
					t.Fatal("complete drain replayed")
				}
			})
		}
	}
}
func TestAnchorDrainPreservesUncertaintyAndObservedExit(t *testing.T) {
	for _, mode := range []string{"lost-reply", "signal-before-effect", "commit-stop-dispatched", "commit-stop-signalled", "commit-exited", "wait-error", "nonzero", "oom", "engine", "late-engine", "inspect", "cancel-inspect", "change-before-signal"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorDrainFixture(t, false)
			f.mode = mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			got, err := ExecuteNextAnchorDrain(ctx, f.history, f.saved, f, f.commit, f.clock)
			if err == nil || got.Validate(f.history) != nil || !reflect.DeepEqual(got, f.saved) {
				t.Fatal("uncertain outcome lost", err)
			}
			if mode == "nonzero" || mode == "oom" {
				if got.Members[0].Phase != "exited" {
					t.Fatal("exit evidence discarded")
				}
			}
			if mode == "nonzero" || mode == "oom" {
				if got.Clean() || got.next() != -1 {
					t.Fatal("unclean exit advanced recovery")
				}
			}
			signals := f.signals
			reads := f.inspects
			phase := got.Members[0].Phase
			if phase != "planned" {
				if _, err := ExecuteNextAnchorDrain(context.Background(), f.history, got, f, f.commit, f.clock); err == nil || f.signals != signals || f.inspects != reads {
					t.Fatal("pending/failed signal replayed")
				}
			}
			if phase == "stop-dispatched" || phase == "stop-signalled" {
				f.mode = ""
				f.mutate = nil
				f.now = f.now.Add(time.Minute)
				observed, err := ObserveAnchorDrainExit(context.Background(), f.history, got, f, f.commit, f.clock)
				if signals == 0 {
					if err == nil || !reflect.DeepEqual(observed, got) {
						t.Fatal("running target called stopped")
					}
				} else {
					if err != nil || observed.Members[0].Phase != "exited" || observed.Members[0].SignalAcknowledged != (phase == "stop-signalled") || observed.Validate(f.history) != nil {
						t.Fatal("read-only reconciliation failed", err)
					}
					if !reflect.DeepEqual(got.Review, observed.Review) {
						t.Fatal("reconciliation changed review")
					}
				}
				if f.signals != signals {
					t.Fatal("observation sent signal")
				}
			}
		})
	}
}
func TestAnchorDrainObservationRefusals(t *testing.T) {
	faults := map[string]func(*AnchorDrainObservation){
		"stale": func(o *AnchorDrainObservation) {
			o.CapturedAt = o.CapturedAt.Add(-time.Minute)
			o.Runtime.ObservedAt = o.CapturedAt
		},
		"future": func(o *AnchorDrainObservation) {
			o.CapturedAt = o.CapturedAt.Add(time.Minute)
			o.Runtime.ObservedAt = o.CapturedAt
		},
		"runtime-future":    func(o *AnchorDrainObservation) { o.Runtime.ObservedAt = o.CapturedAt.Add(time.Second) },
		"incomplete":        func(o *AnchorDrainObservation) { o.Runtime.Complete = false },
		"missing-member":    func(o *AnchorDrainObservation) { o.Runtime.Members = o.Runtime.Members[1:] },
		"missing-process":   func(o *AnchorDrainObservation) { o.Processes = o.Processes[1:] },
		"duplicate-process": func(o *AnchorDrainObservation) { o.Processes[1] = o.Processes[0] },
		"changed-start": func(o *AnchorDrainObservation) {
			o.Processes[0].Process.StartedAt = o.Processes[0].Process.StartedAt.Add(time.Second)
		},
		"changed-pier":             func(o *AnchorDrainObservation) { o.Processes[0].Storage.Identity.Inode++ },
		"changed-parent":           func(o *AnchorDrainObservation) { o.Processes[0].Storage.MountIdentity.Inode++ },
		"migrated-member":          func(o *AnchorDrainObservation) { o.Runtime.Members[0].Namespace = o.Runtime.Anchor.Namespace },
		"changed-anchor-namespace": func(o *AnchorDrainObservation) { o.Runtime.Anchor.Namespace.Inode++ },
		"unknown-bridge": func(o *AnchorDrainObservation) {
			o.Runtime.BridgeAttachments = append(o.Runtime.BridgeAttachments, "foreign")
		},
		"missing-publication": func(o *AnchorDrainObservation) { o.Runtime.Anchor.Active = nil },
		"unexpected-exit": func(o *AnchorDrainObservation) {
			o.Runtime.Members[1].Running = false
			o.Runtime.Members[1].Namespace = NamespaceIdentity{}
		},
		"false-running-exit": func(o *AnchorDrainObservation) { o.Processes[0].Process.FinishedAt = o.CapturedAt },
	}
	for name, mutate := range faults {
		t.Run(name, func(t *testing.T) {
			f := newAnchorDrainFixture(t, false)
			f.mutate = mutate
			if _, err := ExecuteNextAnchorDrain(context.Background(), f.history, f.saved, f, f.commit, f.clock); err == nil || f.signals != 0 || f.commits != 0 {
				t.Fatal("invalid initial observation accepted")
			}
		})
	}
}
func TestAnchorDrainRejectsChangedRetainedSteps(t *testing.T) {
	f := newAnchorDrainFixture(t, true)
	complete, err := ExecuteNextAnchorDrain(context.Background(), f.history, f.saved, f, f.commit, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	faults := map[string]func(*AnchorDrain){
		"sequence":         func(s *AnchorDrain) { s.Sequence++ },
		"member-order":     func(s *AnchorDrain) { s.Members[0], s.Members[1] = s.Members[1], s.Members[0] },
		"forged-ack":       func(s *AnchorDrain) { s.Members[0].SignalAcknowledged = false },
		"lost-exit":        func(s *AnchorDrain) { s.Members[0].After = nil },
		"review-namespace": func(s *AnchorDrain) { s.Review.Context.Binding.Namespace.Inode++ },
		"anchor-process": func(s *AnchorDrain) {
			s.Members[0].After.AnchorProcess.StartedAt = s.Members[0].After.AnchorProcess.StartedAt.Add(time.Second)
		},
		"peer-exit": func(s *AnchorDrain) {
			s.Members[0].After.Processes[1].Process.FinishedAt = s.Members[0].After.CapturedAt
		},
		"backwards-exit": func(s *AnchorDrain) { s.Members[0].After.CapturedAt = s.Members[0].Before.CapturedAt.Add(-time.Second) },
	}
	for name, mutate := range faults {
		t.Run(name, func(t *testing.T) {
			s := cloneAnchorDrain(complete)
			mutate(&s)
			if s.Validate(f.history) == nil {
				t.Fatal("altered receipt accepted")
			}
		})
	}
	// Later inspections must retain the exact old anchor process and previous exit.
	for _, which := range []string{"anchor", "previous-exit"} {
		t.Run(which, func(t *testing.T) {
			q := newAnchorDrainFixture(t, true)
			if _, err := ExecuteNextAnchorDrain(context.Background(), q.history, q.saved, q, q.commit, q.clock); err != nil {
				t.Fatal(err)
			}
			if which == "anchor" {
				q.state.AnchorProcess.StartedAt = q.state.AnchorProcess.StartedAt.Add(time.Second)
			} else {
				q.state.Processes[0].Process.FinishedAt = q.state.Processes[0].Process.FinishedAt.Add(time.Second)
			}
			signals := q.signals
			if _, err := ExecuteNextAnchorDrain(context.Background(), q.history, q.saved, q, q.commit, q.clock); err == nil || q.signals != signals {
				t.Fatal("changed predecessor accepted")
			}
		})
	}
	if _, err := complete.Advance(f.history, 1, complete.Sequence+1, "stop-dispatched", &f.state); err == nil {
		t.Fatal("stale sequence accepted")
	}
	if _, err := complete.Advance(f.history, 0, complete.Sequence, "stop-dispatched", &f.state); err == nil {
		t.Fatal("completed signal replayed")
	}
}

func TestAnchorDrainObservationRequiresValidHistoryAndIntent(t *testing.T) {
	f := newAnchorDrainFixture(t, false)
	o, err := f.InspectDrain(context.Background(), f.saved)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.saved.ValidateObservation(f.history, 0, o, false, f.clock()); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"history", "sequence", "index"} {
		t.Run(fault, func(t *testing.T) {
			h := cloneAnchorRecoveryHistory(f.history)
			s := cloneAnchorDrain(f.saved)
			i := 0
			switch fault {
			case "history":
				h.Originals = nil
			case "sequence":
				s.Sequence++
			case "index":
				i = len(s.Members)
			}
			if s.ValidateObservation(h, i, o, false, f.clock()) == nil {
				t.Fatal("unbound dispatch observation accepted")
			}
		})
	}
}
