package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type recoveryReadinessFixture struct {
	t                            *testing.T
	h                            AnchorRecoveryHistory
	s                            AnchorReplacement
	now                          time.Time
	observations, reads, engines int
	mode                         string
	mutate                       func(int, *AnchorRetirementObservation)
	cancel                       context.CancelFunc
}

func completedReplacementForReadiness(t *testing.T, stopped bool) (AnchorRecoveryHistory, AnchorReplacement, time.Time) {
	t.Helper()
	f := newAnchorReplacementFixture(t, stopped)
	s := f.saved.Clone()
	for !s.Running() {
		var err error
		s, err = ExecuteNextAnchorReplacement(context.Background(), f.h, s, f, f.commit, f.clock)
		if err != nil {
			t.Fatal(err)
		}
	}
	return f.h, s, f.now
}
func (f *recoveryReadinessFixture) clock() time.Time { return f.now }
func (f *recoveryReadinessFixture) VerifyEngine(ctx context.Context, id string) error {
	f.engines++
	if id != f.s.Retirement.Namespace.Drain.Review.Context.Binding.EngineID {
		f.t.Fatal("wrong engine")
	}
	if f.mode == "engine" || f.mode == "late-engine" && f.reads > 0 {
		return errors.New("engine failure")
	}
	return ctx.Err()
}
func (f *recoveryReadinessFixture) InspectRecoveryReadiness(ctx context.Context, s AnchorReplacement) (AnchorRetirementObservation, error) {
	if !reflect.DeepEqual(s, f.s) {
		f.t.Fatal("changed authority")
	}
	f.observations++
	f.now = f.now.Add(time.Second)
	o := cloneAnchorRetirementObservation(*s.Members[len(s.Members)-1].After)
	o.CapturedAt = f.now
	o.Runtime.ObservedAt = f.now
	if f.mutate != nil {
		f.mutate(f.observations, &o)
	}
	if f.mode == "alias" {
		s.Members[0].Creation.Storage.Path = "changed"
		s.Retirement.Namespace.Drain.Members[0].InstanceID = "changed"
	}
	if f.mode == "inspect" || f.mode == "late-inspect" && f.observations == 2 {
		return o, errors.New("inspection failure")
	}
	return o, ctx.Err()
}
func (f *recoveryReadinessFixture) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if f.observations != 1 {
		f.t.Fatal("HTTP outside observation bracket")
	}
	f.reads++
	f.now = f.now.Add(time.Second)
	if f.mode == "slow" {
		f.now = f.now.Add(time.Minute)
	}
	if f.mode == "cancel" {
		f.cancel()
	}
	for _, m := range f.h.Plan.Members {
		_, want, _ := currentReadinessEndpoint(f.h.Plan, m.InstanceID)
		if endpoint == want {
			if f.mode == "wrong-ship" || f.mode == "wrong-last-ship" && f.reads == len(f.s.Members) {
				return "~bus", nil
			}
			if f.mode == "http" {
				return "", errors.New("HTTP failure")
			}
			return m.Ship, ctx.Err()
		}
	}
	f.t.Fatal("unselected HTTP endpoint")
	return "", nil
}
func TestAnchorRecoveryReadinessAllIdentitiesAndExactProcesses(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		h, s, now := completedReplacementForReadiness(t, stopped)
		for _, mode := range []string{"", "alias"} {
			f := &recoveryReadinessFixture{t: t, h: h, s: s.Clone(), now: now, mode: mode}
			original := s.Clone()
			history := h.Clone()
			r, err := CollectAnchorRecoveryReadiness(context.Background(), h, s, f, f.clock)
			if err != nil || r.Validate(h) != nil || f.reads != len(s.Members) || f.observations != 2 {
				t.Fatal("complete readiness failed", err)
			}
			if !reflect.DeepEqual(h, history) || !reflect.DeepEqual(s, original) || !reflect.DeepEqual(r.Replacement, original) {
				t.Fatal("rewrote history")
			}
			if r.Binding.Namespace != s.Retirement.Namespace.After.Runtime.Anchor.Namespace || r.Binding.Generation != s.Retirement.Namespace.Drain.Review.Context.Binding.Generation {
				t.Fatal("changed resource generation or wrong namespace")
			}
			r.Replacement.Members[0].Creation.Storage.Path = "changed"
			r.Before.Piers[0].Storage.Path = "changed"
			if s.Validate(h) != nil {
				t.Fatal("result aliases original")
			}
		}
	}
}
func TestAnchorRecoveryReadinessFailuresPreserveFence(t *testing.T) {
	h, s, now := completedReplacementForReadiness(t, false)
	for _, mode := range []string{"engine", "late-engine", "inspect", "late-inspect", "wrong-ship", "wrong-last-ship", "http", "slow", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := &recoveryReadinessFixture{t: t, h: h, s: s.Clone(), now: now, mode: mode}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			if _, err := CollectAnchorRecoveryReadiness(ctx, h, s, f, f.clock); err == nil {
				t.Fatal("failed readiness accepted")
			}
			if mode == "wrong-last-ship" && f.reads != len(s.Members) || mode == "late-inspect" && f.observations != 2 {
				t.Fatal("did not reach intended boundary")
			}
			if s.Validate(h) != nil {
				t.Fatal("failure changed history")
			}
		})
	}
	mutations := map[string]func(*AnchorRetirementObservation){
		"first-process-restarted": func(o *AnchorRetirementObservation) {
			o.Processes[0].Process.StartedAt = o.Processes[0].Process.StartedAt.Add(time.Nanosecond)
		},
		"last-process-restarted": func(o *AnchorRetirementObservation) {
			i := len(o.Processes) - 1
			o.Processes[i].Process.StartedAt = o.Processes[i].Process.StartedAt.Add(time.Nanosecond)
		},
		"anchor-restarted": func(o *AnchorRetirementObservation) {
			o.AnchorProcess.StartedAt = o.AnchorProcess.StartedAt.Add(time.Nanosecond)
		},
		"anchor-finish-changed": func(o *AnchorRetirementObservation) {
			o.AnchorProcess.FinishedAt = o.AnchorProcess.FinishedAt.Add(time.Nanosecond)
		},
		"namespace":        func(o *AnchorRetirementObservation) { o.Runtime.Anchor.Namespace.Inode++ },
		"member-namespace": func(o *AnchorRetirementObservation) { o.Runtime.Members[0].Namespace.Inode++ },
		"pier":             func(o *AnchorRetirementObservation) { o.Piers[0].Storage.Identity.Inode++ },
		"publication":      func(o *AnchorRetirementObservation) { o.Runtime.Anchor.Active[0].HostPort++ },
		"missing-process":  func(o *AnchorRetirementObservation) { o.Processes = o.Processes[:1] },
		"incomplete":       func(o *AnchorRetirementObservation) { o.Runtime.Complete = false },
		"stale":            func(o *AnchorRetirementObservation) { o.Runtime.ObservedAt = o.Runtime.ObservedAt.Add(-time.Minute) },
	}
	for name, mutate := range mutations {
		for _, at := range []int{1, 2} {
			t.Run(name+string(rune('0'+at)), func(t *testing.T) {
				f := &recoveryReadinessFixture{t: t, h: h, s: s.Clone(), now: now, mutate: func(n int, o *AnchorRetirementObservation) {
					if n == at {
						mutate(o)
					}
				}}
				if _, err := CollectAnchorRecoveryReadiness(context.Background(), h, s, f, f.clock); err == nil {
					t.Fatal("changed evidence accepted")
				}
				if f.observations != at {
					t.Fatal("wrong refusal boundary")
				}
			})
		}
	}
}
func TestAnchorRecoveryReadinessRejectsForgedReceipts(t *testing.T) {
	h, s, now := completedReplacementForReadiness(t, false)
	f := &recoveryReadinessFixture{t: t, h: h, s: s.Clone(), now: now}
	r, err := CollectAnchorRecoveryReadiness(context.Background(), h, s, f, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*AnchorRecoveryReadiness){
		"missing-identity":            func(r *AnchorRecoveryReadiness) { r.Members = r.Members[:1] },
		"duplicate-identity":          func(r *AnchorRecoveryReadiness) { r.Members[1] = r.Members[0] },
		"creator":                     func(r *AnchorRecoveryReadiness) { r.Members[0].CreatorOperationID = "other" },
		"generation":                  func(r *AnchorRecoveryReadiness) { r.Members[0].ContainerGeneration++ },
		"endpoint":                    func(r *AnchorRecoveryReadiness) { r.Members[0].Endpoint = "http://foreign/~/host" },
		"identity-before-observation": func(r *AnchorRecoveryReadiness) { r.Members[0].CapturedAt = r.Before.CapturedAt.Add(-time.Nanosecond) },
		"identity-after-observation": func(r *AnchorRecoveryReadiness) {
			r.Members[1].CapturedAt = r.After.Runtime.ObservedAt.Add(time.Nanosecond)
		},
		"old-namespace": func(r *AnchorRecoveryReadiness) {
			r.Binding.Namespace = s.Retirement.Namespace.Drain.Review.Context.Binding.Namespace
		},
		"resource-generation": func(r *AnchorRecoveryReadiness) { r.Binding.Generation++ },
		"pending-replacement": func(r *AnchorRecoveryReadiness) { r.Replacement.Members[1].Phase = "start-dispatched" },
		"capture-stale":       func(r *AnchorRecoveryReadiness) { r.CapturedAt = r.CapturedAt.Add(time.Minute) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			v := r.Clone()
			mutate(&v)
			if v.Validate(h) == nil {
				t.Fatal("forged receipt accepted")
			}
			if r.Validate(h) != nil {
				t.Fatal("clone aliases original")
			}
		})
	}
}
