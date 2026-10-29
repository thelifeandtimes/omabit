package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func initialStopFixture(t *testing.T) (Plan, MemberStop, MemberStopObservation, MemberStopObservation) {
	t.Helper()
	plan, launches, o, location := initialReadinessFixture(t)
	binding, err := InitialRuntimeBinding(plan, launches)
	if err != nil {
		t.Fatal(err)
	}
	ready := InitialMemberReadiness{InstanceID: "zod", LaunchOperationID: launches[0].OperationID, Binding: binding, Storage: location, Ship: "~zod", Endpoint: "http://127.0.0.1:18081/~/host", Before: o, After: o, CapturedAt: time.Unix(1001, 0)}
	s, err := NewInitialMemberStop(plan, launches, ready, "observe-zod", "stop-zod")
	if err != nil {
		t.Fatal(err)
	}
	before := MemberStopObservation{Runtime: cloneReadinessObservation(o), Storage: location, StartedAt: time.Unix(900, 0), CapturedAt: time.Unix(1001, 0)}
	after := cloneMemberStopObservation(before)
	after.CapturedAt = time.Unix(1002, 0)
	after.Runtime.ObservedAt = after.CapturedAt
	after.FinishedAt = after.CapturedAt
	for i := range after.Runtime.Members {
		if after.Runtime.Members[i].InstanceID == s.InstanceID {
			after.Runtime.Members[i].Running = false
			after.Runtime.Members[i].Namespace = NamespaceIdentity{}
		}
	}
	return plan, s, before, after
}

type stopFixtureDriver struct {
	t                      *testing.T
	retained               MemberStop
	before, after          MemberStopObservation
	fault                  string
	cancel                 context.CancelFunc
	verify, signals, waits int
}

func (d *stopFixtureDriver) VerifyEngine(context.Context, string) error {
	d.verify++
	if d.fault == "engine" || d.fault == "late-engine" && d.verify == 5 {
		return errors.New("private engine detail")
	}
	return nil
}
func (d *stopFixtureDriver) check(s MemberStop, phase string) {
	d.t.Helper()
	if s.Phase != phase || !reflect.DeepEqual(s, d.retained) {
		d.t.Fatal("effect preceded durable state", phase)
	}
	s.Binding.Members[0].InstanceID = "edited"
	if s.Before != nil {
		s.Before.Runtime.Members[0].InstanceID = "edited"
	}
}
func (d *stopFixtureDriver) Inspect(_ context.Context, s MemberStop) (MemberStopObservation, error) {
	d.check(s, "planned")
	if d.fault == "inspect" {
		return MemberStopObservation{}, errors.New("secret fixture detail")
	}
	return d.before, nil
}
func (d *stopFixtureDriver) SignalStop(_ context.Context, s MemberStop) error {
	d.check(s, "stop-dispatched")
	d.signals++
	if d.fault == "signal" {
		return errors.New("private signal detail")
	}
	if d.fault == "cancel-signal" {
		d.cancel()
	}
	return nil
}
func (d *stopFixtureDriver) AwaitStopped(_ context.Context, s MemberStop) (MemberStopObservation, error) {
	d.check(s, "stop-signalled")
	d.waits++
	if d.fault == "wait" {
		return MemberStopObservation{}, errors.New("secret wait detail")
	}
	if d.fault == "cancel-wait" {
		d.cancel()
	}
	if d.fault == "late-wait" {
		return d.after, errors.New("private late result")
	}
	return d.after, nil
}
func TestExecuteMemberStopRetainsEveryEffectBoundary(t *testing.T) {
	cases := map[string]string{"pass": "stopped", "engine": "planned", "inspect": "planned", "signal": "stop-dispatched", "cancel-signal": "stop-signalled", "wait": "stop-signalled", "late-engine": "stop-signalled", "cancel-wait": "stop-signalled", "late-wait": "stopped", "unclean": "stopped", "oom": "stopped", "commit-dispatch": "planned", "commit-signal": "stop-dispatched", "commit-exit": "stop-signalled", "stale-dispatch": "stop-dispatched"}
	for fault, want := range cases {
		t.Run(fault, func(t *testing.T) {
			plan, initial, before, after := initialStopFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "unclean" {
				after.ExitCode = 137
			}
			if fault == "oom" {
				after.OOMKilled = true
			}
			d := &stopFixtureDriver{t: t, retained: cloneMemberStop(initial), before: before, after: after, fault: fault, cancel: cancel}
			now := time.Unix(1002, 0)
			commits := 0
			commit := func(expected string, next MemberStop) error {
				if d.retained.Phase != expected {
					t.Fatal("stale compare and commit")
				}
				if fault == "commit-dispatch" && next.Phase == "stop-dispatched" || fault == "commit-signal" && next.Phase == "stop-signalled" || fault == "commit-exit" && next.Phase == "stopped" {
					return errors.New("synthetic commit failure")
				}
				d.retained = cloneMemberStop(next)
				commits++
				next.Binding.Members[0].InstanceID = "edited"
				if fault == "stale-dispatch" {
					now = time.Unix(1033, 0)
				}
				return nil
			}
			got, err := ExecuteMemberStop(ctx, plan, initial, d, commit, func() time.Time { return now })
			if got.Phase != want || !reflect.DeepEqual(got, d.retained) || got.Validate(plan) != nil {
				t.Fatal(got.Phase, want, err)
			}
			if (fault == "pass") != (err == nil) {
				t.Fatal(fault, err)
			}
			if err != nil && (strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret")) {
				t.Fatal("leaked adapter details")
			}
			if want == "planned" || fault == "stale-dispatch" {
				if d.signals != 0 {
					t.Fatal("signalled before valid dispatch")
				}
			} else if d.signals != 1 {
				t.Fatal("signal count", d.signals)
			}
			if fault == "unclean" || fault == "oom" {
				if got.Clean() || got.After == nil {
					t.Fatal("unclean exit lost or relabeled")
				}
			}
			if fault == "pass" && (!got.Clean() || commits != 3 || d.waits != 1) {
				t.Fatal("missing clean stop sequence")
			}
			d.before.Runtime.Members[0].InstanceID = "edited"
			d.after.Runtime.Members[0].InstanceID = "edited"
			if got.Validate(plan) != nil || initial.Binding.Members[0].InstanceID == "edited" {
				t.Fatal("retained state aliases adapter or caller")
			}
			if got.Phase != "planned" {
				signals := d.signals
				if _, err = ExecuteMemberStop(context.Background(), plan, got, d, commit, func() time.Time { return now }); err == nil || d.signals != signals {
					t.Fatal("resumed uncertain or completed Stop")
				}
			}
		})
	}
}
func TestMemberStopRejectsChangedOwnershipAndExitEvidence(t *testing.T) {
	faults := []string{"before-missing", "before-stale", "before-stopped", "parent", "pier", "urb", "unknown-member", "anchor", "member", "peer-stopped", "restarted", "not-exited", "old-exit", "future-exit", "negative-exit", "large-exit", "after-stale", "after-running", "after-missing", "reverse-time"}
	for _, fault := range faults {
		t.Run(fault, func(t *testing.T) {
			plan, s, before, after := initialStopFixture(t)
			badBefore := strings.HasPrefix(fault, "before-")
			switch fault {
			case "before-missing":
				before.Runtime.Complete = false
			case "before-stale":
				before.CapturedAt = time.Unix(960, 0)
				before.Runtime.ObservedAt = before.CapturedAt
			case "before-stopped":
				for i := range before.Runtime.Members {
					if before.Runtime.Members[i].InstanceID == s.InstanceID {
						before.Runtime.Members[i].Running = false
						before.Runtime.Members[i].Namespace = NamespaceIdentity{}
					}
				}
			case "parent":
				after.Storage.MountIdentity.Inode++
			case "pier":
				after.Storage.Identity.Inode++
			case "urb":
				after.Storage.Identity.UrbInode++
			case "unknown-member":
				after.Runtime.Members = append(after.Runtime.Members, after.Runtime.Members[0])
			case "anchor":
				after.Runtime.Anchor.Namespace.Inode++
			case "member":
				after.Runtime.Members[0].Resource.ID = strings.Repeat("f", 64)
			case "peer-stopped":
				for i := range after.Runtime.Members {
					if after.Runtime.Members[i].InstanceID != s.InstanceID {
						after.Runtime.Members[i].Running = false
						after.Runtime.Members[i].Namespace = NamespaceIdentity{}
					}
				}
			case "restarted":
				after.StartedAt = after.StartedAt.Add(time.Second)
			case "not-exited":
				after.FinishedAt = time.Time{}
			case "old-exit":
				after.FinishedAt = time.Unix(950, 0)
			case "future-exit":
				after.FinishedAt = time.Unix(1010, 0)
			case "negative-exit":
				after.ExitCode = -1
			case "large-exit":
				after.ExitCode = 256
			case "after-stale":
				after.CapturedAt = time.Unix(1033, 0)
			case "after-running":
				after.Runtime = cloneReadinessObservation(before.Runtime)
			case "after-missing":
				after.Runtime.Complete = false
			case "reverse-time":
				after.CapturedAt = time.Unix(999, 0)
				after.Runtime.ObservedAt = after.CapturedAt
			}
			d := &stopFixtureDriver{t: t, retained: cloneMemberStop(s), before: before, after: after}
			commit := func(_ string, next MemberStop) error { d.retained = cloneMemberStop(next); return nil }
			got, err := ExecuteMemberStop(context.Background(), plan, s, d, commit, func() time.Time { return time.Unix(1002, 0) })
			if err == nil || got.After != nil {
				t.Fatal("accepted incomplete exit", fault, got)
			}
			if badBefore {
				if got.Phase != "planned" || d.signals != 0 {
					t.Fatal("signalled invalid ownership")
				}
			} else if got.Phase != "stop-signalled" || d.signals != 1 {
				t.Fatal("lost dispatched outcome", got.Phase)
			}
		})
	}
}
func TestMemberStopRejectsMalformedAndSkippedTransitions(t *testing.T) {
	plan, s, before, after := initialStopFixture(t)
	for _, phase := range []string{"stop-signalled", "stopped", "bogus"} {
		if _, err := s.Advance(plan, "planned", phase, &after); err == nil {
			t.Fatal("skipped dispatch", phase)
		}
	}
	for _, fault := range []string{"operation", "readiness", "launch", "member", "binding", "storage", "phase"} {
		v := cloneMemberStop(s)
		switch fault {
		case "operation":
			v.OperationID = v.ReadinessOperationID
		case "readiness":
			v.ReadinessOperationID = v.LaunchOperationID
		case "launch":
			v.LaunchOperationID = ""
		case "member":
			v.InstanceID = "missing"
		case "binding":
			v.Binding.Namespace.Inode = 0
		case "storage":
			v.Storage.Uncertain = true
		case "phase":
			v.Phase = "stopped"
		}
		if v.Validate(plan) == nil {
			t.Fatal(fault)
		}
	}
	dispatched, err := s.Advance(plan, "planned", "stop-dispatched", &before)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dispatched.Advance(plan, "stop-dispatched", "stop-signalled", &before); err == nil {
		t.Fatal("spurious observation at acknowledgement")
	}
	if _, err = dispatched.Advance(plan, "planned", "stop-dispatched", &before); err == nil {
		t.Fatal("repeated stale transition")
	}
}

func TestMemberStopPreservesAlreadyStoppedPeer(t *testing.T) {
	plan, s, before, after := initialStopFixture(t)
	for _, o := range []*MemberStopObservation{&before, &after} {
		for i := range o.Runtime.Members {
			if o.Runtime.Members[i].InstanceID != s.InstanceID {
				o.Runtime.Members[i].Running = false
				o.Runtime.Members[i].Namespace = NamespaceIdentity{}
			}
		}
	}
	d := &stopFixtureDriver{t: t, retained: cloneMemberStop(s), before: before, after: after}
	got, err := ExecuteMemberStop(context.Background(), plan, s, d, func(_ string, next MemberStop) error { d.retained = cloneMemberStop(next); return nil }, func() time.Time { return time.Unix(1002, 0) })
	if err != nil || !got.Clean() || d.signals != 1 {
		t.Fatal(got.Phase, err)
	}
}
func TestMemberStopCancellationBeforeObservationDoesNothing(t *testing.T) {
	plan, s, before, after := initialStopFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := &stopFixtureDriver{t: t, retained: cloneMemberStop(s), before: before, after: after}
	got, err := ExecuteMemberStop(ctx, plan, s, d, func(string, MemberStop) error { t.Fatal("committed canceled request"); return nil }, time.Now)
	if err == nil || got.Phase != "planned" || d.verify != 0 || d.signals != 0 || d.waits != 0 {
		t.Fatal(got.Phase, err, d)
	}
}
