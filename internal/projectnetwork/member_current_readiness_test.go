package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func currentReadinessFixture(t *testing.T, instance string) (Plan, []MemberLaunch, []MemberResume, InitialMemberReadiness, MemberContext, MemberStopObservation) {
	t.Helper()
	p, launches, original := memberContextReady(t, instance)
	_, first := resumedReadinessFixture(t)
	// Keep the original peer running throughout the replacement history, not just
	// in the final read. No previous peer Stop or unrecorded restart is implied.
	peerRunning := func(o *Observation) {
		for i := range o.Members {
			if o.Members[i].InstanceID == "nec" {
				o.Members[i].Running = true
				o.Members[i].Namespace = o.Anchor.Namespace
			}
		}
	}
	peerRunning(&first.Retirement.Exit.Stop.Before.Runtime)
	if first.Retirement.Exit.Stop.After != nil {
		peerRunning(&first.Retirement.Exit.Stop.After.Runtime)
	}
	peerRunning(&first.Retirement.Exit.Observation.Runtime)
	peerRunning(&first.Retirement.Before.Runtime)
	peerRunning(&first.Retirement.After.Runtime)
	peerRunning(&first.Before.Runtime)
	peerRunning(&first.Created.Runtime)
	peerRunning(&first.After.Runtime)
	if err := first.Validate(p); err != nil {
		t.Fatal(err)
	}
	history := []MemberResume{first}
	c, err := SelectMemberContext(p, launches, history, original)
	if err != nil {
		t.Fatal(err)
	}
	now := first.After.CapturedAt.Add(time.Second)
	started := c.KnownStartedAt
	if started.IsZero() {
		started = time.Unix(900, 0)
	}
	o := MemberStopObservation{Runtime: cloneReadinessObservation(first.After.Runtime), Storage: c.Storage, StartedAt: started, CapturedAt: now}
	o.Runtime.ObservedAt = now
	return p, launches, history, original, c, o
}

type currentReadinessDriver struct {
	t                            *testing.T
	expected                     MemberContext
	before, after                MemberStopObservation
	ship, endpoint, fault        string
	inspections, reads, verifies int
	cancel                       context.CancelFunc
}

func (d *currentReadinessDriver) VerifyEngine(_ context.Context, id string) error {
	d.verifies++
	if id != d.expected.Binding.EngineID {
		d.t.Fatal("wrong engine")
	}
	if d.fault == "engine" || d.fault == "late-engine" && d.verifies == 4 {
		return errors.New("private engine detail")
	}
	return nil
}
func (d *currentReadinessDriver) Inspect(_ context.Context, c MemberContext) (MemberStopObservation, error) {
	d.inspections++
	if !reflect.DeepEqual(c, d.expected) {
		d.t.Fatal("changed selection")
	}
	c.Binding.Members[0].InstanceID = "driver-alias"
	if d.fault == "inspect" || d.fault == "late-inspect" && d.inspections == 2 {
		return MemberStopObservation{}, errors.New("private inspection detail")
	}
	if d.inspections == 1 {
		return d.before, nil
	}
	return d.after, nil
}
func (d *currentReadinessDriver) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	d.reads++
	if endpoint != d.endpoint {
		d.t.Fatal("wrong endpoint")
	}
	if d.fault == "http" {
		return "", errors.New("private HTTP detail")
	}
	if d.fault == "identity" {
		return "~bus", nil
	}
	if d.fault == "cancel-http" {
		d.cancel()
	}
	d.before.Runtime.Members[0].InstanceID = "buffer-alias"
	return d.ship, nil
}

func TestCurrentMemberReadinessBracketsLatestHistory(t *testing.T) {
	for _, instance := range []string{"nec", "zod"} {
		t.Run(instance, func(t *testing.T) {
			for _, fault := range []string{"pass", "engine", "late-engine", "inspect", "late-inspect", "http", "identity", "cancel-before", "cancel-http", "stale", "future", "predates-history", "changed-process", "unknown-process", "changed-pier", "changed-peer", "old-binding", "changed-namespace", "attachment", "target-stopped", "reversed-time"} {
				t.Run(fault, func(t *testing.T) {
					p, launches, history, original, c, before := currentReadinessFixture(t, instance)
					after := cloneMemberStopObservation(before)
					after.CapturedAt = after.CapturedAt.Add(time.Second)
					after.Runtime.ObservedAt = after.CapturedAt
					now := after.CapturedAt
					target := 0
					if instance == "zod" {
						target = 1
					}
					switch fault {
					case "stale":
						before.Runtime.ObservedAt = before.CapturedAt.Add(-time.Minute)
					case "future":
						after.CapturedAt = now.Add(time.Minute)
					case "predates-history":
						before.CapturedAt = history[0].After.CapturedAt.Add(-time.Second)
						before.Runtime.ObservedAt = before.CapturedAt
					case "changed-process":
						after.StartedAt = after.StartedAt.Add(time.Second)
					case "unknown-process":
						before.StartedAt = time.Time{}
					case "changed-pier":
						after.Storage.Identity.Inode++
					case "changed-peer":
						after.Runtime.Members[1-target].Running = false
						after.Runtime.Members[1-target].Namespace = NamespaceIdentity{}
					case "old-binding":
						after.Runtime.Members[1].Resource = original.Binding.Members[1].Container
					case "changed-namespace":
						after.Runtime.Anchor.Namespace.Inode++
					case "attachment":
						after.Runtime.BridgeAttachments = append(after.Runtime.BridgeAttachments, strings.Repeat("9", 64))
					case "target-stopped":
						after.Runtime.Members[target].Running = false
						after.Runtime.Members[target].Namespace = NamespaceIdentity{}
					case "reversed-time":
						after.CapturedAt = before.CapturedAt.Add(-time.Second)
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					if fault == "cancel-before" {
						cancel()
					}
					_, endpoint, _ := currentReadinessEndpoint(p, instance)
					d := &currentReadinessDriver{t: t, expected: cloneMemberContext(c), before: before, after: after, ship: original.Ship, endpoint: endpoint, fault: fault, cancel: cancel}
					r, err := CollectCurrentMemberReadiness(ctx, p, launches, history, original, d, func() time.Time { return now })
					if fault == "pass" {
						if err != nil || r.Validate(p, launches, history, original) != nil || d.reads != 1 || d.inspections != 2 || reflect.DeepEqual(r.Binding, original.Binding) {
							t.Fatal("current readiness failed", err)
						}
						if !r.Before.StartedAt.Equal(r.After.StartedAt) || r.CreatorOperationID != c.Member.OperationID {
							t.Fatal("wrong process or creator")
						}
					} else if err == nil || !reflect.DeepEqual(r, CurrentMemberReadiness{}) || strings.Contains(err.Error(), "private") {
						t.Fatal("failure returned evidence or private details", err)
					}
					if original.Validate(p, launches) != nil || history[0].Validate(p) != nil {
						t.Fatal("history changed")
					}
				})
			}
		})
	}
}

func TestCurrentMemberReadinessRefusesPendingHistoryBeforeIO(t *testing.T) {
	p, launches, history, original, c, before := currentReadinessFixture(t, "nec")
	history[0].Phase = "started"
	history[0].After = nil
	d := &currentReadinessDriver{t: t, expected: c, before: before, after: before}
	_, err := CollectCurrentMemberReadiness(context.Background(), p, launches, history, original, d, time.Now)
	if err == nil || d.verifies+d.reads+d.inspections != 0 {
		t.Fatal("incomplete history reached driver")
	}
}
