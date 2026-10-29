package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func generationReadinessFixture(t *testing.T, p Plan, s MemberResume) ResumedMemberReadiness {
	t.Helper()
	b, err := ResumedRuntimeBinding(p, s)
	if err != nil {
		t.Fatal(err)
	}
	r := ResumedMemberReadiness{InstanceID: "zod", ResumeOperationID: s.OperationID, ContainerGeneration: s.PreviousGeneration + 1, Binding: b, Storage: s.After.Storage, Ship: "~zod", Endpoint: "http://127.0.0.1:18081/~/host", Before: cloneResumeObservation(*s.After), After: cloneResumeObservation(*s.After), CapturedAt: s.After.CapturedAt}
	if err = r.Validate(p, s); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestResumedMemberStopTargetsLatestGeneration(t *testing.T) {
	for _, third := range []bool{false, true} {
		p, launches, _, _ := initialReadinessFixture(t)
		_, first := resumedReadinessFixture(t)
		history := []MemberResume{first}
		latest := first
		if third {
			latest = nextGenerationFixture(t, p, first)
			history = append(history, latest)
		}
		ready := generationReadinessFixture(t, p, latest)
		for _, fault := range []string{"pass", "lost-signal", "uncommitted-dispatch"} {
			t.Run(latest.OperationID+"/"+fault, func(t *testing.T) {
				initial, err := NewResumedMemberStop(p, launches, history, ready, "ready-stop-current", "stop-current")
				if err != nil {
					t.Fatal(err)
				}
				if initial.LaunchOperationID != latest.OperationID || initial.Binding.Members[1].Container != latest.Container {
					t.Fatal("selected retired container")
				}
				before := MemberStopObservation{Runtime: cloneReadinessObservation(latest.After.Runtime), Storage: ready.Storage, StartedAt: latest.After.StartedAt, CapturedAt: latest.After.CapturedAt.Add(time.Second)}
				before.Runtime.ObservedAt = before.CapturedAt
				after := cloneMemberStopObservation(before)
				after.CapturedAt = before.CapturedAt.Add(time.Second)
				after.Runtime.ObservedAt = after.CapturedAt
				after.FinishedAt = after.CapturedAt
				for i := range after.Runtime.Members {
					if after.Runtime.Members[i].InstanceID == initial.InstanceID {
						after.Runtime.Members[i].Running = false
						after.Runtime.Members[i].Namespace = NamespaceIdentity{}
					}
				}
				d := &stopFixtureDriver{t: t, retained: cloneMemberStop(initial), before: before, after: after}
				if fault == "lost-signal" {
					d.fault = "signal"
				}
				got, err := ExecuteMemberStop(context.Background(), p, initial, d, func(expected string, next MemberStop) error {
					if expected != d.retained.Phase {
						t.Fatal("stale dispatch")
					}
					if fault == "uncommitted-dispatch" {
						return errors.New("fixture commit failure")
					}
					d.retained = cloneMemberStop(next)
					return nil
				}, func() time.Time { return after.CapturedAt })
				switch fault {
				case "pass":
					if err != nil || !got.Clean() || d.signals != 1 {
						t.Fatal(got, err)
					}
				case "lost-signal":
					if err == nil || got.Phase != "stop-dispatched" || d.signals != 1 {
						t.Fatal(got, err)
					}
				case "uncommitted-dispatch":
					if err == nil || got.Phase != "planned" || d.signals != 0 {
						t.Fatal(got, err)
					}
				}
				if ready.Validate(p, latest) != nil || history[0].Validate(p) != nil {
					t.Fatal("rewrote prior evidence")
				}
			})
		}
	}
}
func TestResumedMemberStopRejectsStaleSelection(t *testing.T) {
	for _, fault := range []string{"older-readiness", "missing-predecessor", "wrong-creator", "wrong-member", "wrong-generation", "same-operation", "old-binding"} {
		t.Run(fault, func(t *testing.T) {
			p, launches, _, _ := initialReadinessFixture(t)
			_, first := resumedReadinessFixture(t)
			second := nextGenerationFixture(t, p, first)
			history := []MemberResume{first, second}
			ready := generationReadinessFixture(t, p, second)
			operation := "stop-current"
			switch fault {
			case "older-readiness":
				ready = generationReadinessFixture(t, p, first)
			case "missing-predecessor":
				history = history[1:]
			case "wrong-creator":
				ready.ResumeOperationID = "wrong-creator"
			case "wrong-member":
				ready.InstanceID = "nec"
			case "wrong-generation":
				ready.ContainerGeneration--
			case "same-operation":
				operation = second.OperationID
			case "old-binding":
				ready.Binding = second.Retirement.Exit.Stop.Binding
			}
			result, err := NewResumedMemberStop(p, launches, history, ready, "ready-stop-current", operation)
			if err == nil || !reflect.DeepEqual(result, MemberStop{}) {
				t.Fatal("stale selection accepted", result, err)
			}
		})
	}
}

func TestResumedMemberStopHistoricalValidationIsNotAdmission(t *testing.T) {
	p, launches, _, _ := initialReadinessFixture(t)
	_, first := resumedReadinessFixture(t)
	second := nextGenerationFixture(t, p, first)
	ready := generationReadinessFixture(t, p, first)
	old, err := NewResumedMemberStop(p, launches, []MemberResume{first}, ready, "ready-old", "stop-old")
	if err != nil {
		t.Fatal(err)
	}
	history := []MemberResume{second, first}
	if err = ValidateResumedMemberStop(p, launches, history, ready, old); err != nil {
		t.Fatal("lost historical Stop", err)
	}
	if _, err = NewResumedMemberStop(p, launches, history, ready, "ready-old", "another-stop"); err == nil {
		t.Fatal("old receipt authorized new Stop")
	}
	changed := old
	changed.LaunchOperationID = second.OperationID
	if err = ValidateResumedMemberStop(p, launches, history, ready, changed); err == nil {
		t.Fatal("changed old receipt accepted")
	}
}
