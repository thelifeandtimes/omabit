package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
)

type anchorRetirementFixture struct {
	t                       *testing.T
	history                 AnchorRecoveryHistory
	saved                   AnchorRetirement
	state                   AnchorRetirementObservation
	now                     time.Time
	removes, reads, commits int
	mode                    string
	cancel                  context.CancelFunc
	mutate                  func(*AnchorRetirementObservation)
}

func newAnchorRetirementFixture(t *testing.T, stopped bool) *anchorRetirementFixture {
	t.Helper()
	ns := newAnchorNamespaceFixture(t, stopped)
	successor, err := ExecuteAnchorNamespace(context.Background(), ns.history, ns.saved, ns, ns.commit, ns.clock)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewAnchorRetirement(ns.history, successor, "retire-old-members")
	if err != nil {
		t.Fatal(err)
	}
	o := AnchorRetirementObservation{Runtime: cloneReadinessObservation(successor.After.Runtime), AnchorProcess: successor.After.AnchorProcess, Processes: slices.Clone(successor.After.Processes), CapturedAt: successor.After.CapturedAt}
	for _, p := range successor.After.Processes {
		o.Piers = append(o.Piers, RecoveryPier{InstanceID: p.InstanceID, Storage: p.Storage})
	}
	return &anchorRetirementFixture{t: t, history: ns.history, saved: s, state: o, now: ns.now}
}
func (f *anchorRetirementFixture) clock() time.Time { return f.now }
func (f *anchorRetirementFixture) VerifyEngine(ctx context.Context, id string) error {
	if id != f.saved.Namespace.Drain.Review.Context.Binding.EngineID {
		f.t.Fatal("wrong engine selection")
	}
	if f.mode == "engine" || f.mode == "late-engine" && f.removes > 0 {
		return errors.New("private engine failure")
	}
	return ctx.Err()
}
func (f *anchorRetirementFixture) InspectRetirement(_ context.Context, s AnchorRetirement) (AnchorRetirementObservation, error) {
	if !reflect.DeepEqual(s, f.saved) {
		f.t.Fatal("changed retirement selection")
	}
	f.reads++
	f.now = f.now.Add(time.Second)
	o := cloneAnchorRetirementObservation(f.state)
	o.CapturedAt = f.now
	o.Runtime.ObservedAt = f.now
	if f.mutate != nil {
		f.mutate(&o)
	}
	if f.mode == "mutate-driver" {
		s.Namespace.Drain.Members[0].InstanceID = "foreign"
		s.Members[0].InstanceID = "foreign"
	}
	if f.mode == "inspect" || f.mode == "inspect-after" && f.removes > 0 {
		return o, errors.New("private inspection failure")
	}
	return o, nil
}
func (f *anchorRetirementFixture) RemoveRetiredMember(ctx context.Context, s AnchorRetirement, index int) error {
	if !reflect.DeepEqual(s, f.saved) || s.Members[index].Phase != "remove-dispatched" {
		f.t.Fatal("removal preceded committed intent")
	}
	before, err := f.InspectRetirement(ctx, s.Clone())
	if err != nil || s.ValidateObservation(f.history, index, before, false, f.now) != nil {
		return errors.New("removal recheck refused")
	}
	if f.mode == "remove-before-effect" {
		return errors.New("private removal failure")
	}
	f.removes++
	id := s.Members[index].InstanceID
	if f.mode != "false-ack" {
		f.state.Runtime.Members = slices.DeleteFunc(f.state.Runtime.Members, func(m MemberObservation) bool { return m.InstanceID == id })
		f.state.Processes = slices.DeleteFunc(f.state.Processes, func(p AnchorDrainProcess) bool { return p.InstanceID == id })
	}
	if f.mode == "lost-reply" {
		return errors.New("private reply lost")
	}
	if f.mode == "cancel-after" {
		f.cancel()
	}
	return nil
}
func (f *anchorRetirementFixture) commit(seq uint64, next AnchorRetirement) error {
	if seq != f.saved.Sequence || next.Sequence != seq+1 || next.Validate(f.history) != nil {
		f.t.Fatal("stale or invalid retirement commit")
	}
	f.commits++
	index := f.saved.next()
	phase := next.Members[index].Phase
	if f.mode == "commit-"+phase {
		return errors.New("failed commit")
	}
	f.saved = next.Clone()
	if f.mode == "mutate-commit" {
		next.Members[0].InstanceID = "foreign"
		next.Namespace.Drain.Review.Context.Binding.Members[0].InstanceID = "foreign"
	}
	if phase == "remove-dispatched" {
		if f.mode == "stale-dispatch" {
			f.now = f.now.Add(time.Minute)
		}
		if f.mode == "changed-pier" {
			f.state.Piers[0].Storage.Path = "changed"
		}
	}
	return nil
}
func TestAnchorRetirementSerialAbsencePreservesPiersAndHistory(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, mode := range []string{"", "mutate-driver", "mutate-commit"} {
			t.Run(map[bool]string{true: "stopped", false: "restarted"}[stopped]+"/"+mode, func(t *testing.T) {
				f := newAnchorRetirementFixture(t, stopped)
				f.mode = mode
				original := f.saved.Clone()
				history := f.history.Clone()
				piers := slices.Clone(f.state.Piers)
				first, err := ExecuteNextAnchorRetirement(context.Background(), f.history, f.saved, f, f.commit, f.clock)
				if err != nil || first.Members[0].Phase != "absent" || first.Members[1].Phase != "planned" || f.removes != 1 || first.Complete() {
					t.Fatal("first retirement failed", err)
				}
				retained := first.Clone()
				got, err := ExecuteNextAnchorRetirement(context.Background(), f.history, first, f, f.commit, f.clock)
				if err != nil || !got.Complete() || got.Validate(f.history) != nil || f.removes != 2 || len(f.state.Processes) != 0 || len(f.state.Runtime.Members) != 0 {
					t.Fatal("serial retirement failed", err)
				}
				if !reflect.DeepEqual(got.Namespace, original.Namespace) || !reflect.DeepEqual(history, f.history) || !reflect.DeepEqual(piers, f.state.Piers) || !reflect.DeepEqual(got.Members[0], retained.Members[0]) {
					t.Fatal("retirement rewrote history or pier identity")
				}
				if _, err = ExecuteNextAnchorRetirement(context.Background(), f.history, got, f, f.commit, f.clock); err == nil || f.removes != 2 {
					t.Fatal("completed removal replayed")
				}
				got.Namespace.Drain.Members[0].InstanceID = "caller-change"
				got.Members[0].After.Piers[0].InstanceID = "caller-change"
				if f.saved.Validate(f.history) != nil || original.Validate(f.history) != nil {
					t.Fatal("result aliases source")
				}
			})
		}
	}
}
func TestAnchorRetirementUncertaintyNeverRepeatsRemoval(t *testing.T) {
	for _, mode := range []string{"engine", "inspect", "commit-remove-dispatched", "stale-dispatch", "changed-pier", "remove-before-effect", "lost-reply", "commit-remove-acknowledged", "false-ack", "inspect-after", "late-engine", "cancel-after", "commit-absent"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorRetirementFixture(t, false)
			f.mode = mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			got, err := ExecuteNextAnchorRetirement(ctx, f.history, f.saved, f, f.commit, f.clock)
			if err == nil || got.Validate(f.history) != nil || !reflect.DeepEqual(got, f.saved) {
				t.Fatal("failure not retained", err)
			}
			n := f.removes
			if got.Members[0].Phase != "planned" {
				if _, err = ExecuteNextAnchorRetirement(context.Background(), f.history, got, f, f.commit, f.clock); err == nil || f.removes != n {
					t.Fatal("uncertain removal replayed")
				}
				f.mode = ""
				recovered, err := ObserveAnchorRetirementAbsence(context.Background(), f.history, got, f, f.commit, f.clock)
				actualAbsent := len(f.state.Processes) == 1
				if actualAbsent {
					if err != nil || recovered.Members[0].Phase != "absent" || recovered.Members[0].RemoveAcknowledged != got.Members[0].RemoveAcknowledged {
						t.Fatal("read-only recovery changed ack or failed", err)
					}
				} else if err == nil {
					t.Fatal("absence inferred from acknowledgement")
				}
				if f.removes != n {
					t.Fatal("read-only absence repeated removal")
				}
			}
		})
	}
}
func TestAnchorRetirementRequiresExactAllPierAndProcessEvidence(t *testing.T) {
	faults := map[string]func(*AnchorRetirementObservation){
		"missing-pier":     func(o *AnchorRetirementObservation) { o.Piers = o.Piers[:1] },
		"duplicate-pier":   func(o *AnchorRetirementObservation) { o.Piers[1] = o.Piers[0] },
		"changed-pier":     func(o *AnchorRetirementObservation) { o.Piers[0].Storage.Path = "changed" },
		"changed-identity": func(o *AnchorRetirementObservation) { o.Piers[0].Storage.Identity.Inode++ },
		"member-restarted": func(o *AnchorRetirementObservation) { o.Processes[0].Process.StartedAt = o.CapturedAt },
		"unclean-exit":     func(o *AnchorRetirementObservation) { o.Processes[0].Process.ExitCode = 1 },
		"foreign-process":  func(o *AnchorRetirementObservation) { o.Processes = append(o.Processes, o.Processes[0]) },
		"missing-process":  func(o *AnchorRetirementObservation) { o.Processes = nil },
		"member-running":   func(o *AnchorRetirementObservation) { o.Runtime.Members[0].Running = true },
		"foreign-container": func(o *AnchorRetirementObservation) {
			o.Runtime.Members = append(o.Runtime.Members, o.Runtime.Members[0])
		},
		"anchor-restarted": func(o *AnchorRetirementObservation) { o.AnchorProcess.StartedAt = o.CapturedAt },
		"namespace":        func(o *AnchorRetirementObservation) { o.Runtime.Anchor.Namespace.Inode++ },
		"anchor-stopped":   func(o *AnchorRetirementObservation) { o.Runtime.Anchor.Running = false },
		"bridge":           func(o *AnchorRetirementObservation) { o.Runtime.BridgeAttachments = nil },
		"publications":     func(o *AnchorRetirementObservation) { o.Runtime.Anchor.Active = nil },
		"incomplete":       func(o *AnchorRetirementObservation) { o.Runtime.Complete = false },
		"stale":            func(o *AnchorRetirementObservation) { o.CapturedAt = o.CapturedAt.Add(-time.Minute) },
		"future":           func(o *AnchorRetirementObservation) { o.CapturedAt = o.CapturedAt.Add(time.Minute) },
	}
	for name, mutate := range faults {
		t.Run(name, func(t *testing.T) {
			f := newAnchorRetirementFixture(t, false)
			f.mutate = mutate
			if _, err := ExecuteNextAnchorRetirement(context.Background(), f.history, f.saved, f, f.commit, f.clock); err == nil || f.removes != 0 || f.commits != 0 {
				t.Fatal("changed preflight accepted")
			}
		})
	}
	// Even after the container disappears, its pier must still be freshly observed.
	f := newAnchorRetirementFixture(t, false)
	f.mode = "lost-reply"
	pending, err := ExecuteNextAnchorRetirement(context.Background(), f.history, f.saved, f, f.commit, f.clock)
	if err == nil {
		t.Fatal("lost reply unexpectedly complete")
	}
	f.mode = ""
	f.state.Piers = f.state.Piers[1:]
	if _, err = ObserveAnchorRetirementAbsence(context.Background(), f.history, pending, f, f.commit, f.clock); err == nil || f.removes != 1 {
		t.Fatal("missing retired pier accepted")
	}
}
func TestAnchorRetirementRejectsForgedOrderAndAuthority(t *testing.T) {
	for _, mode := range []string{"history", "namespace-pending", "operation-reused", "drain-reused", "namespace-reused", "sequence", "skip-member", "skip-phase", "changed-order", "false-ack", "missing-before", "missing-after", "changed-old-step"} {
		t.Run(mode, func(t *testing.T) {
			f := newAnchorRetirementFixture(t, true)
			s := f.saved.Clone()
			h := f.history.Clone()
			if mode == "skip-member" {
				if _, err := s.Advance(h, 1, s.Sequence, "remove-dispatched", &f.state); err == nil {
					t.Fatal("skipped member")
				}
				return
			}
			if mode == "skip-phase" {
				if _, err := s.Advance(h, 0, s.Sequence, "absent", &f.state); err == nil {
					t.Fatal("skipped dispatch")
				}
				return
			}
			switch mode {
			case "history":
				h.Resumes = nil
			case "namespace-pending":
				s.Namespace.Phase = "start-dispatched"
			case "operation-reused":
				s.OperationID = h.Resumes[0].Retirement.OperationID
			case "drain-reused":
				s.OperationID = s.Namespace.Drain.OperationID
			case "namespace-reused":
				s.OperationID = s.Namespace.OperationID
			case "sequence":
				s.Sequence++
			case "changed-order":
				s.Members[0], s.Members[1] = s.Members[1], s.Members[0]
			default:
				var err error
				s, err = ExecuteNextAnchorRetirement(context.Background(), h, s, f, f.commit, f.clock)
				if err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "false-ack":
					s.Members[0].RemoveAcknowledged = false
				case "missing-before":
					s.Members[0].Before = nil
				case "missing-after":
					s.Members[0].After = nil
				case "changed-old-step":
					s.Members[0].After.AnchorProcess.StartedAt = f.now
				}
			}
			if s.Validate(h) == nil {
				t.Fatal("forged retirement accepted")
			}
		})
	}
}
