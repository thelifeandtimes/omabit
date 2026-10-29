package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type stopExitDriver struct {
	t               *testing.T
	original        MemberStop
	observed        MemberStopObservation
	verifies, reads int
	fault           string
	cancel          context.CancelFunc
}

func (d *stopExitDriver) VerifyEngine(ctx context.Context, id string) error {
	d.verifies++
	if id != d.original.Binding.EngineID {
		d.t.Fatal("foreign engine")
	}
	if d.fault == "engine-before" || d.fault == "engine-after" && d.verifies > 1 {
		return errors.New("secret engine failure")
	}
	return nil
}
func (d *stopExitDriver) InspectExit(ctx context.Context, s MemberStop) (MemberStopObservation, error) {
	d.reads++
	if !reflect.DeepEqual(s, d.original) {
		d.t.Fatal("lost original Stop")
	}
	// Mutation of the callback's argument must not rewrite the retained receipt.
	s.Binding.Members[0].Container.ID = strings.Repeat("f", 64)
	s.Before.Runtime.Members[0].Running = false
	if d.fault == "cancel" {
		d.cancel()
	}
	if d.fault == "read" {
		return d.observed, errors.New("secret read failure")
	}
	return d.observed, nil
}
func TestCollectMemberStopExitPreservesOriginalOutcome(t *testing.T) {
	for _, phase := range []string{"stop-dispatched", "stop-signalled", "stopped"} {
		for _, unclean := range []bool{false, true} {
			t.Run(phase+map[bool]string{false: "-clean", true: "-unclean"}[unclean], func(t *testing.T) {
				plan, initial, before, after := initialStopFixture(t)
				s, err := initial.Advance(plan, "planned", "stop-dispatched", &before)
				if err != nil {
					t.Fatal(err)
				}
				if phase != "stop-dispatched" {
					s, err = s.Advance(plan, "stop-dispatched", "stop-signalled", nil)
					if err != nil {
						t.Fatal(err)
					}
				}
				if unclean {
					after.ExitCode = 137
					after.OOMKilled = true
				}
				if phase == "stopped" {
					s, err = s.Advance(plan, "stop-signalled", "stopped", &after)
					if err != nil {
						t.Fatal(err)
					}
				}
				original := cloneMemberStop(s)
				driver := &stopExitDriver{t: t, original: cloneMemberStop(s), observed: cloneMemberStopObservation(after)}
				evidence, err := CollectMemberStopExit(context.Background(), plan, s, "observe-stop", driver, func() time.Time { return after.CapturedAt })
				if err != nil || evidence.Validate(plan) != nil || evidence.Clean() == unclean || driver.verifies != 2 || driver.reads != 1 {
					t.Fatal(evidence, err)
				}
				if !reflect.DeepEqual(s, original) || !reflect.DeepEqual(evidence.Stop, original) {
					t.Fatal("historical Stop was rewritten")
				}
				driver.observed.Runtime.Members[0].Resource.ID = strings.Repeat("a", 64)
				s.Before.Runtime.Anchor.Active[0].HostPort++
				if !reflect.DeepEqual(evidence.Stop, original) || evidence.Validate(plan) != nil {
					t.Fatal("caller changed retained exit evidence")
				}
			})
		}
	}
}
func TestCollectMemberStopExitRefusesMissingStaleAndChangedEvidence(t *testing.T) {
	for _, fault := range []string{"planned", "same-operation", "engine-before", "engine-after", "read", "cancel", "already-cancelled", "running", "missing", "restarted", "storage", "peer", "stale", "future", "old-known-exit", "changed-known-exit"} {
		t.Run(fault, func(t *testing.T) {
			plan, initial, before, after := initialStopFixture(t)
			s, err := initial.Advance(plan, "planned", "stop-dispatched", &before)
			if err != nil {
				t.Fatal(err)
			}
			operation := "observe-stop"
			now := after.CapturedAt
			switch fault {
			case "planned":
				s = initial
			case "same-operation":
				operation = s.OperationID
			case "running":
				after = before
			case "missing":
				for i := range after.Runtime.Members {
					if after.Runtime.Members[i].InstanceID == s.InstanceID {
						after.Runtime.Members[i].Present = false
					}
				}
			case "restarted":
				after.StartedAt = after.StartedAt.Add(time.Second)
			case "storage":
				after.Storage.Identity.Inode++
			case "peer":
				for i := range after.Runtime.Members {
					if after.Runtime.Members[i].InstanceID != s.InstanceID {
						after.Runtime.Members[i].Running = false
						after.Runtime.Members[i].Namespace = NamespaceIdentity{}
					}
				}
			case "stale":
				now = now.Add(31 * time.Second)
			case "future":
				now = now.Add(-6 * time.Second)
			case "old-known-exit", "changed-known-exit":
				s, err = s.Advance(plan, "stop-dispatched", "stop-signalled", nil)
				if err != nil {
					t.Fatal(err)
				}
				s, err = s.Advance(plan, "stop-signalled", "stopped", &after)
				if err != nil {
					t.Fatal(err)
				}
				if fault == "old-known-exit" {
					after.CapturedAt = after.CapturedAt.Add(-time.Millisecond)
					after.Runtime.ObservedAt = after.Runtime.ObservedAt.Add(-time.Millisecond)
				} else {
					after.ExitCode = 137
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "already-cancelled" {
				cancel()
			}
			d := &stopExitDriver{t: t, original: cloneMemberStop(s), observed: after, fault: fault, cancel: cancel}
			got, err := CollectMemberStopExit(ctx, plan, s, operation, d, func() time.Time { return now })
			if err == nil || !reflect.DeepEqual(got, MemberStopExitEvidence{}) || strings.Contains(err.Error(), "secret") {
				t.Fatal("unresolved exit accepted", fault, got, err)
			}
			if (fault == "planned" || fault == "same-operation" || fault == "already-cancelled") && (d.verifies != 0 || d.reads != 0) {
				t.Fatal("invalid input reached driver")
			}
		})
	}
}
