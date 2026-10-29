package projectnetwork

import (
	"context"
	"testing"
	"time"
)

func currentStopModelFixture(t *testing.T, instance string) (Plan, []MemberLaunch, []MemberResume, InitialMemberReadiness, CurrentMemberReadiness, MemberStop) {
	t.Helper()
	p, launches, history, original, c, b := currentReadinessFixture(t, instance)
	a := cloneMemberStopObservation(b)
	a.CapturedAt = a.CapturedAt.Add(time.Second)
	a.Runtime.ObservedAt = a.CapturedAt
	_, endpoint, _ := currentReadinessEndpoint(p, instance)
	d := &currentReadinessDriver{t: t, expected: c, before: b, after: a, ship: original.Ship, endpoint: endpoint}
	ready, err := CollectCurrentMemberReadiness(context.Background(), p, launches, history, original, d, func() time.Time { return a.CapturedAt })
	if err != nil {
		t.Fatal(err)
	}
	stop, err := NewCurrentMemberStop(p, launches, history, original, ready, "current-ready", "current-stop")
	if err != nil {
		t.Fatal(err)
	}
	return p, launches, history, original, ready, stop
}
func TestCurrentMemberStopAfterPeerReplacement(t *testing.T) {
	for _, instance := range []string{"nec", "zod"} {
		t.Run(instance, func(t *testing.T) {
			p, launches, history, original, ready, stop := currentStopModelFixture(t, instance)
			if err := ValidateCurrentMemberStop(p, launches, history, original, ready, stop); err != nil {
				t.Fatal(err)
			}
			b := cloneMemberStopObservation(ready.After)
			b.CapturedAt = b.CapturedAt.Add(time.Second)
			b.Runtime.ObservedAt = b.CapturedAt
			stopped, err := stop.Advance(p, "planned", "stop-dispatched", &b)
			if err != nil {
				t.Fatal(err)
			}
			if err = ValidateCurrentMemberStop(p, launches, history, original, ready, stopped); err != nil {
				t.Fatal(err)
			}
			// Later completed target history does not rewrite earlier ownership. This is
			// a historical-read test, not permission to perform the stale Stop again.
			second := nextGenerationFixture(t, p, history[0])
			later := append(history, second)
			if ready.Validate(p, launches, later, original) == nil {
				t.Fatal("old readiness treated as current")
			}
			if err = ValidateCurrentMemberStop(p, launches, later, original, ready, stopped); err != nil {
				t.Fatal("lost historical validity", err)
			}
			if _, err = NewCurrentMemberStop(p, launches, later, original, ready, "another-ready", "another-stop"); err == nil {
				t.Fatal("stale Stop admitted")
			}
			for _, fault := range []string{"creator", "process", "chronology", "binding", "storage"} {
				changed := cloneMemberStop(stopped)
				switch fault {
				case "creator":
					changed.LaunchOperationID = "different-creator"
				case "process":
					changed.Before.StartedAt = changed.Before.StartedAt.Add(time.Second)
				case "chronology":
					changed.Before.CapturedAt = ready.After.CapturedAt.Add(-time.Second)
					changed.Before.Runtime.ObservedAt = changed.Before.CapturedAt
				case "binding":
					changed.Binding = original.Binding
				case "storage":
					changed.Storage.Identity.Inode++
				}
				if ValidateCurrentMemberStop(p, launches, later, original, ready, changed) == nil {
					t.Fatal("changed Stop accepted", fault)
				}
			}
		})
	}
}
