package projectnetwork

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestRecoveredCurrentReadinessAndStop(t *testing.T) {
	base, first, tail := recoveredHistoryFixture(t)
	history, err := AppendAnchorRecoveryEpoch(base, first)
	if err != nil {
		t.Fatal(err)
	}
	history.Resumes = tail
	second := secondRecoveryFixture(t, history)
	history, err = AppendAnchorRecoveryEpoch(history, second)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range history.Plan.Members {
		t.Run(member.InstanceID, func(t *testing.T) {
			c, err := SelectRecoveryMemberContext(history, member.InstanceID)
			if err != nil {
				t.Fatal(err)
			}
			now := second.CapturedAt.Add(time.Second)
			before := MemberStopObservation{Runtime: cloneReadinessObservation(second.After.Runtime), Storage: c.Storage, StartedAt: c.KnownStartedAt, CapturedAt: now}
			before.Runtime.ObservedAt = now
			after := cloneMemberStopObservation(before)
			after.CapturedAt = now.Add(time.Second)
			after.Runtime.ObservedAt = after.CapturedAt
			_, endpoint, _ := currentReadinessEndpoint(history.Plan, member.InstanceID)
			for _, mode := range []string{"pass", "changed-process", "changed-namespace", "changed-ship", "predates-recovery", "incomplete-history"} {
				t.Run(mode, func(t *testing.T) {
					h := history.Clone()
					b := cloneMemberStopObservation(before)
					a := cloneMemberStopObservation(after)
					d := &currentReadinessDriver{t: t, expected: cloneMemberContext(c), before: b, after: a, ship: member.Ship, endpoint: endpoint}
					switch mode {
					case "changed-process":
						d.after.StartedAt = d.after.StartedAt.Add(time.Second)
					case "changed-namespace":
						d.after.Runtime.Anchor.Namespace.Inode++
					case "changed-ship":
						d.ship = "~bus"
					case "predates-recovery":
						d.before.CapturedAt = second.CapturedAt.Add(-time.Second)
						d.before.Runtime.ObservedAt = d.before.CapturedAt
					case "incomplete-history":
						h.Epochs[1].Readiness.Replacement.Members[0].Phase = "created"
					}
					r, err := CollectRecoveredCurrentMemberReadiness(context.Background(), h, member.InstanceID, d, func() time.Time { return after.CapturedAt })
					if mode != "pass" {
						if err == nil || !reflect.DeepEqual(r, CurrentMemberReadiness{}) {
							t.Fatal("invalid recovered readiness accepted", mode)
						}
						if mode == "incomplete-history" && (d.verifies != 0 || d.reads != 0 || d.inspections != 0) {
							t.Fatal("invalid history reached driver")
						}
						return
					}
					if err != nil || r.ValidateRecovered(h) != nil || d.reads != 1 || d.inspections != 2 {
						t.Fatal("recovered readiness failed", err)
					}
					s, err := NewRecoveredCurrentMemberStop(h, r, "current-ready-"+member.InstanceID, "current-stop-"+member.InstanceID)
					if err != nil {
						t.Fatal(err)
					}
					observed := cloneMemberStopObservation(r.After)
					observed.CapturedAt = observed.CapturedAt.Add(time.Second)
					observed.Runtime.ObservedAt = observed.CapturedAt
					s, err = s.Advance(h.Plan, "planned", "stop-dispatched", &observed)
					if err != nil || ValidateRecoveredCurrentMemberStop(h, r, s) != nil {
						t.Fatal("recovered Stop lost readiness", err)
					}
					bad := cloneMemberStop(s)
					bad.Before.StartedAt = bad.Before.StartedAt.Add(time.Nanosecond)
					if ValidateRecoveredCurrentMemberStop(h, r, bad) == nil {
						t.Fatal("changed Stop process accepted")
					}
					// Later ordinary replacement of this member invalidates fresh admission,
					// while the earlier readiness/Stop retain their exact historical prefix.
					laterContext := cloneMemberContext(c)
					laterContext.NotBefore = observed.CapturedAt
					resumed := ordinaryResumeAfterRecovery(t, h.Plan, laterContext, observed.Runtime, "after-second-"+member.InstanceID, 900)
					h.Resumes = append(h.Resumes, resumed)
					if r.ValidateRecovered(h) == nil || r.ValidateRecoveredHistorical(h) != nil || ValidateRecoveredCurrentMemberStop(h, r, s) != nil {
						t.Fatal("current/historical recovery authority conflated")
					}
					h.Epochs[0].Readiness.Binding.Namespace.Inode++
					if r.ValidateRecoveredHistorical(h) == nil {
						t.Fatal("corrupt full history accepted at historical prefix")
					}
				})
			}
		})
	}
}
