package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
)

type anchorNamespaceFixture struct {
	t                       *testing.T
	history                 AnchorRecoveryHistory
	saved                   AnchorNamespace
	state                   AnchorDrainObservation
	now                     time.Time
	starts, reads, verifies int
	mode                    string
	cancel                  context.CancelFunc
}

func newAnchorNamespaceFixture(t *testing.T, stopped bool) *anchorNamespaceFixture {
	t.Helper()
	drain := newAnchorDrainFixture(t, stopped)
	for !drain.saved.Clean() {
		if _, err := ExecuteNextAnchorDrain(context.Background(), drain.history, drain.saved, drain, drain.commit, drain.clock); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewAnchorNamespace(drain.history, drain.saved, "successor-namespace")
	if err != nil {
		t.Fatal(err)
	}
	return &anchorNamespaceFixture{t: t, history: drain.history, saved: s, state: cloneAnchorDrainObservation(drain.state), now: drain.now}
}
func (f *anchorNamespaceFixture) clock() time.Time { return f.now }
func (f *anchorNamespaceFixture) VerifyEngine(ctx context.Context, id string) error {
	f.verifies++
	if id != f.saved.Drain.Review.Context.Binding.EngineID {
		f.t.Fatal("wrong engine")
	}
	if f.mode == "engine" || f.mode == "late-engine" && f.starts > 0 {
		return errors.New("private engine detail")
	}
	return ctx.Err()
}
func (f *anchorNamespaceFixture) InspectNamespace(_ context.Context, s AnchorNamespace) (AnchorDrainObservation, error) {
	f.reads++
	f.now = f.now.Add(time.Second)
	if !reflect.DeepEqual(s, f.saved) {
		f.t.Fatal("changed namespace selection")
	}
	o := cloneAnchorDrainObservation(f.state)
	o.CapturedAt = f.now
	o.Runtime.ObservedAt = f.now
	if f.mode == "mutate-driver" {
		s.Drain.Members[0].InstanceID = "changed"
		s.Drain.Review.Context.Binding.Members[0].InstanceID = "changed"
	}
	if f.mode == "after-inspection" && f.reads > 1 {
		return o, errors.New("private inspect detail")
	}
	return o, nil
}
func (f *anchorNamespaceFixture) StartRecoveryAnchor(ctx context.Context, s AnchorNamespace) error {
	if !reflect.DeepEqual(s, f.saved) || s.Phase != "start-dispatched" {
		f.t.Fatal("Start before durable intent")
	}
	before := cloneAnchorDrainObservation(f.state)
	before.CapturedAt = f.now
	before.Runtime.ObservedAt = f.now
	if err := s.ValidateObservation(f.history, before, false, f.now); err != nil {
		return err
	}
	if f.mode == "start-error-before" {
		return errors.New("private Start failure")
	}
	f.starts++
	f.state.Runtime.Anchor.Running = true
	f.state.Runtime.Anchor.Namespace = s.Drain.Review.Context.Binding.Namespace
	f.state.Runtime.Anchor.Namespace.Inode += 500
	f.state.Runtime.Anchor.Active = slices.Clone(f.state.Runtime.Anchor.Requested)
	f.state.Runtime.BridgeAttachments = []string{s.Drain.Review.Context.Binding.Anchor.ID}
	f.state.AnchorProcess = RecoveryProcessState{StartedAt: f.now, FinishedAt: f.state.AnchorProcess.FinishedAt}
	if f.mode == "lost-reply" {
		return errors.New("private reply lost")
	}
	if f.mode == "cancel-after-start" {
		f.cancel()
	}
	return nil
}
func (f *anchorNamespaceFixture) commit(seq uint64, next AnchorNamespace) error {
	if seq != f.saved.Sequence || next.Sequence != seq+1 || next.Validate(f.history) != nil {
		f.t.Fatal("stale or invalid commit")
	}
	if f.mode == "commit-"+next.Phase {
		return errors.New("failed commit")
	}
	f.saved = next.Clone()
	if f.mode == "mutate-commit" {
		next.Drain.Members[0].InstanceID = "changed"
		next.Before.Processes[0].InstanceID = "changed"
	}
	if f.mode == "pre-start-peer" {
		f.state.Processes[0].Process.StartedAt = f.now
	}
	if f.mode == "pre-start-pier" {
		f.state.Processes[0].Storage.Path = "changed"
	}
	return nil
}
func TestAnchorNamespacePreservesHistoryAndAvoidsRedundantStart(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, mode := range []string{"", "mutate-driver", "mutate-commit"} {
			t.Run(map[bool]string{false: "restarted", true: "stopped"}[stopped]+"/"+mode, func(t *testing.T) {
				f := newAnchorNamespaceFixture(t, stopped)
				f.mode = mode
				history := f.history.Clone()
				original := f.saved.Clone()
				got, err := ExecuteAnchorNamespace(context.Background(), f.history, f.saved, f, f.commit, f.clock)
				if err != nil || got.Validate(f.history) != nil || got.Phase != "observed" || !reflect.DeepEqual(got.Drain, original.Drain) || !reflect.DeepEqual(history, f.history) {
					t.Fatal("succession failed/history changed", err)
				}
				expected := 0
				if stopped {
					expected = 1
				}
				if f.starts != expected || got.StartAcknowledged != stopped {
					t.Fatal("redundant Start or wrong ack")
				}
				ns, err := got.Namespace(f.history)
				if err != nil || ns == *history.Launches[0].Anchor.Namespace {
					t.Fatal("missing successor", err)
				}
				if _, err = ExecuteAnchorNamespace(context.Background(), f.history, got, f, f.commit, f.clock); err == nil || f.starts != expected {
					t.Fatal("completed Start replay")
				}
				got.After.Processes[0].InstanceID = "caller-change"
				got.Drain.Members[0].InstanceID = "caller-change"
				if f.saved.Validate(f.history) != nil || original.Validate(f.history) != nil {
					t.Fatal("returned evidence aliases retained state")
				}
			})
		}
	}
}

func TestAnchorNamespaceUncertainDispatchAndReadOnlyRecovery(t *testing.T) {
	for _, mode := range []string{"engine", "start-error-before", "lost-reply", "late-engine", "cancel-after-start", "after-inspection", "commit-start-dispatched", "commit-start-acknowledged", "commit-observed", "pre-start-peer", "pre-start-pier"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorNamespaceFixture(t, true)
			f.mode = mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			got, err := ExecuteAnchorNamespace(ctx, f.history, f.saved, f, f.commit, f.clock)
			if err == nil || !reflect.DeepEqual(got, f.saved) || got.Validate(f.history) != nil {
				t.Fatal("uncertainty not retained", err)
			}
			if _, err = got.Namespace(f.history); err == nil {
				t.Fatal("uncertain state exposes successor")
			}
			n := f.starts
			if got.Phase != "planned" {
				if _, err = ExecuteAnchorNamespace(context.Background(), f.history, got, f, f.commit, f.clock); err == nil || f.starts != n {
					t.Fatal("uncertain Start replayed")
				}
			}
			f.mode = ""
			if got.Phase == "planned" {
				return
			}
			recovered, err := ObserveAnchorNamespace(context.Background(), f.history, got, f, f.commit, f.clock)
			if n == 0 {
				if err == nil {
					t.Fatal("missing Start treated as success")
				}
			} else {
				if err != nil || recovered.Phase != "observed" || recovered.StartAcknowledged != got.StartAcknowledged {
					t.Fatal("read-only recovery failed or forged ack", err)
				}
			}
			if f.starts != n {
				t.Fatal("observation issued Start")
			}
		})
	}
}
func TestAnchorNamespaceRejectsChangedEvidence(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		f := newAnchorNamespaceFixture(t, stopped)
		got, err := ExecuteAnchorNamespace(context.Background(), f.history, f.saved, f, f.commit, f.clock)
		if err != nil {
			t.Fatal(err)
		}
		faults := map[string]func(*AnchorDrainObservation){
			"old-namespace": func(o *AnchorDrainObservation) {
				o.Runtime.Anchor.Namespace = got.Drain.Review.Context.Binding.Namespace
			},
			"host-boot": func(o *AnchorDrainObservation) {
				o.Runtime.Anchor.Namespace.BootID = "22345678-1234-1234-1234-123456789abc"
			},
			"anchor-id": func(o *AnchorDrainObservation) { o.Runtime.Anchor.Resource.ID = "changed" },
			"contract":  func(o *AnchorDrainObservation) { o.Runtime.Anchor.Resource.ContractDigest = "changed" },
			"foreign-bridge": func(o *AnchorDrainObservation) {
				o.Runtime.BridgeAttachments = append(o.Runtime.BridgeAttachments, "foreign")
			},
			"member-running":    func(o *AnchorDrainObservation) { o.Runtime.Members[0].Running = true },
			"member-namespace":  func(o *AnchorDrainObservation) { o.Runtime.Members[0].Namespace = o.Runtime.Anchor.Namespace },
			"member-start":      func(o *AnchorDrainObservation) { o.Processes[0].Process.StartedAt = f.now },
			"member-exit":       func(o *AnchorDrainObservation) { o.Processes[0].Process.ExitCode = 1 },
			"pier":              func(o *AnchorDrainObservation) { o.Processes[0].Storage.Path = "foreign" },
			"missing-process":   func(o *AnchorDrainObservation) { o.Processes = o.Processes[:1] },
			"duplicate-process": func(o *AnchorDrainObservation) { o.Processes[1] = o.Processes[0] },
			"publication":       func(o *AnchorDrainObservation) { o.Runtime.Anchor.Active = nil },
			"oom":               func(o *AnchorDrainObservation) { o.AnchorProcess.OOMKilled = true },
			"stale":             func(o *AnchorDrainObservation) { o.CapturedAt = f.now.Add(-time.Minute) },
			"future":            func(o *AnchorDrainObservation) { o.CapturedAt = f.now.Add(time.Minute) },
			"before-dispatch":   func(o *AnchorDrainObservation) { o.Runtime.ObservedAt = got.Before.CapturedAt.Add(-time.Second) },
			"incomplete":        func(o *AnchorDrainObservation) { o.Runtime.Complete = false },
		}
		if !stopped {
			faults["anchor-restarted-again"] = func(o *AnchorDrainObservation) { o.AnchorProcess.StartedAt = f.now }
		}
		for name, change := range faults {
			t.Run(map[bool]string{false: "restarted", true: "stopped"}[stopped]+"/"+name, func(t *testing.T) {
				o := cloneAnchorDrainObservation(*got.After)
				change(&o)
				if err := got.ValidateObservation(f.history, o, true, f.now); err == nil {
					t.Fatal("changed succession evidence accepted")
				}
			})
		}
	}
}
func TestAnchorNamespaceRequiresCompleteCleanDrain(t *testing.T) {
	for _, mode := range []string{"incomplete", "unclean", "same-operation", "history-changed", "sequence", "false-ack", "missing-before", "missing-after"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorNamespaceFixture(t, true)
			s := f.saved.Clone()
			h := f.history.Clone()
			switch mode {
			case "incomplete":
				s.Drain.Members[1] = AnchorDrainMember{InstanceID: s.Drain.Members[1].InstanceID, Phase: "planned"}
				s.Drain.Sequence -= 3
			case "unclean":
				s.Drain.Members[1].After.Processes[1].Process.ExitCode = 8
			case "same-operation":
				s.OperationID = s.Drain.OperationID
			case "history-changed":
				h.Resumes = nil
			default:
				var err error
				s, err = ExecuteAnchorNamespace(context.Background(), h, s, f, f.commit, f.clock)
				if err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "sequence":
					s.Sequence++
				case "false-ack":
					s.StartAcknowledged = false
				case "missing-before":
					s.Before = nil
				case "missing-after":
					s.After = nil
				}
			}
			if s.Validate(h) == nil {
				t.Fatal("forged succession accepted")
			}
		})
	}
}

func TestAnchorNamespaceInterruptedObservationDoesNotStartRunningAnchor(t *testing.T) {
	f := newAnchorNamespaceFixture(t, false)
	f.mode = "commit-observed"
	got, err := ExecuteAnchorNamespace(context.Background(), f.history, f.saved, f, f.commit, f.clock)
	if err == nil || got.Phase != "observing" || f.starts != 0 {
		t.Fatal("failed commit lost observation intent", err)
	}
	f.mode = ""
	recovered, err := ObserveAnchorNamespace(context.Background(), f.history, got, f, f.commit, f.clock)
	if err != nil || recovered.Phase != "observed" || recovered.StartAcknowledged || f.starts != 0 {
		t.Fatal("observation recovery started anchor", err)
	}
}
