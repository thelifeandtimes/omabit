package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func retirementFixture(t *testing.T) (Plan, MemberRetirement, MemberStopObservation, MemberAbsenceObservation) {
	t.Helper()
	p, s, b, a := initialStopFixture(t)
	dispatched, err := s.Advance(p, "planned", "stop-dispatched", &b)
	if err != nil {
		t.Fatal(err)
	}
	proof := MemberStopExitEvidence{OperationID: "observed-exit", Stop: dispatched, Observation: a}
	retired, err := NewMemberRetirement(p, proof, "retire-zod")
	if err != nil {
		t.Fatal(err)
	}
	before := cloneMemberStopObservation(a)
	before.CapturedAt = before.CapturedAt.Add(time.Second)
	before.Runtime.ObservedAt = before.CapturedAt
	after := MemberAbsenceObservation{Runtime: cloneReadinessObservation(before.Runtime), Storage: before.Storage, CapturedAt: before.CapturedAt.Add(time.Second)}
	after.Runtime.ObservedAt = after.CapturedAt
	after.Runtime.Members = nil
	for _, m := range before.Runtime.Members {
		if m.InstanceID != s.InstanceID {
			after.Runtime.Members = append(after.Runtime.Members, m)
		}
	}
	return p, retired, before, after
}

type retirementDriver struct {
	t                                  *testing.T
	retained                           MemberRetirement
	before                             MemberStopObservation
	after                              MemberAbsenceObservation
	fault                              string
	verifies, reads, removes, absences int
	cancel                             context.CancelFunc
	now                                *time.Time
}

func (d *retirementDriver) check(s MemberRetirement, phase string) {
	d.t.Helper()
	if s.Phase != phase || !reflect.DeepEqual(s, d.retained) {
		d.t.Fatal("effect preceded committed retirement", phase)
	}
	s.Exit.Stop.Binding.Members[0].InstanceID = "mutated"
	if s.Before != nil {
		s.Before.Runtime.Anchor.Active[0].HostPort++
	}
}
func (d *retirementDriver) VerifyEngine(_ context.Context, id string) error {
	d.verifies++
	if id != d.retained.Exit.Stop.Binding.EngineID {
		d.t.Fatal("foreign engine")
	}
	if d.fault == "engine-before" || d.fault == "engine-dispatch" && d.verifies == 3 || d.fault == "engine-after" && d.verifies == 5 {
		return errors.New("private engine failure")
	}
	return nil
}
func (d *retirementDriver) InspectStopped(_ context.Context, s MemberRetirement) (MemberStopObservation, error) {
	d.check(s, "planned")
	d.reads++
	if d.fault == "read" {
		return d.before, errors.New("private read failure")
	}
	return d.before, nil
}
func (d *retirementDriver) RemoveStopped(_ context.Context, s MemberRetirement) error {
	d.check(s, "remove-dispatched")
	d.removes++
	if d.fault == "remove" {
		return errors.New("private lost removal reply")
	}
	if d.fault == "cancel-remove" {
		d.cancel()
	}
	if d.fault == "late-absence" {
		*d.now = d.now.Add(60 * time.Second)
	}
	return nil
}
func (d *retirementDriver) InspectAbsent(_ context.Context, s MemberRetirement) (MemberAbsenceObservation, error) {
	d.check(s, "removed")
	d.absences++
	if d.fault == "absence" {
		return d.after, errors.New("private absence failure")
	}
	if d.fault == "cancel-absence" {
		d.cancel()
	}
	return d.after, nil
}
func TestMemberRetirementWriteAheadAndUncertainty(t *testing.T) {
	cases := []struct {
		fault, phase string
		removes      int
	}{
		{"", "absent", 1}, {"engine-before", "planned", 0}, {"read", "planned", 0}, {"cancel-before", "planned", 0},
		{"commit-remove-dispatched", "planned", 0}, {"engine-dispatch", "remove-dispatched", 0}, {"stale-dispatch", "remove-dispatched", 0},
		{"remove", "remove-dispatched", 1}, {"commit-removed", "remove-dispatched", 1}, {"cancel-remove", "removed", 1},
		{"absence", "removed", 1}, {"late-absence", "removed", 1}, {"cancel-absence", "removed", 1}, {"engine-after", "removed", 1}, {"commit-absent", "removed", 1},
	}
	for _, c := range cases {
		t.Run(c.fault, func(t *testing.T) {
			p, s, b, a := retirementFixture(t)
			original := cloneMemberRetirement(s)
			now := a.CapturedAt
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &retirementDriver{t: t, retained: cloneMemberRetirement(s), before: b, after: a, fault: c.fault, cancel: cancel, now: &now}
			if c.fault == "cancel-before" {
				cancel()
			}
			commit := func(expected string, next MemberRetirement) error {
				if d.retained.Phase != expected {
					t.Fatal("commit lost expected phase")
				}
				if c.fault == "commit-"+next.Phase {
					return errors.New("commit failed")
				}
				d.retained = cloneMemberRetirement(next)
				next.Exit.Stop.Before.Runtime.Anchor.Active[0].HostPort++
				if c.fault == "stale-dispatch" {
					now = now.Add(60 * time.Second)
				}
				return nil
			}
			got, err := ExecuteMemberRetirement(ctx, p, s, d, commit, func() time.Time { return now })
			if (err == nil) != (c.fault == "") || got.Phase != c.phase || d.removes != c.removes || !reflect.DeepEqual(got, d.retained) || !reflect.DeepEqual(s, original) {
				t.Fatal(c, got.Phase, d.removes, err)
			}
			if err != nil && strings.Contains(err.Error(), "private") {
				t.Fatal("driver detail leaked")
			}
			if got.Phase != "planned" {
				calls := d.verifies
				if _, err = ExecuteMemberRetirement(context.Background(), p, got, d, commit, func() time.Time { return now }); err == nil || d.verifies != calls || d.removes != c.removes {
					t.Fatal("uncertain/completed retirement replayed")
				}
			}
			if c.fault == "" {
				d.before.Runtime.Members[0].InstanceID = "changed"
				d.after.Runtime.Anchor.Active[0].HostPort++
				if got.Validate(p) != nil || !reflect.DeepEqual(got.Exit, original.Exit) {
					t.Fatal("driver mutated retained evidence")
				}
			}
		})
	}
}
func TestMemberRetirementRejectsChangedExitStorageAndProject(t *testing.T) {
	for _, fault := range []string{"unclean", "same-operation", "unknown-phase", "running", "start", "finish", "exit-code", "storage", "peer", "namespace", "stale", "still-present", "after-storage", "after-peer", "after-namespace", "incomplete", "old-absence", "future-absence"} {
		t.Run(fault, func(t *testing.T) {
			p, s, b, a := retirementFixture(t)
			now := a.CapturedAt
			switch fault {
			case "unclean":
				s.Exit.Observation.ExitCode = 137
			case "same-operation":
				s.OperationID = s.Exit.OperationID
			case "unknown-phase":
				s.Phase = "unknown"
			case "running":
				for i := range b.Runtime.Members {
					if b.Runtime.Members[i].InstanceID == s.Exit.Stop.InstanceID {
						b.Runtime.Members[i].Running = true
						b.Runtime.Members[i].Namespace = s.Exit.Stop.Binding.Namespace
					}
				}
			case "start":
				b.StartedAt = b.StartedAt.Add(time.Second)
			case "finish":
				b.FinishedAt = b.FinishedAt.Add(time.Second)
			case "exit-code":
				b.ExitCode = 137
			case "storage":
				b.Storage.Identity.Inode++
			case "peer":
				for i := range b.Runtime.Members {
					if b.Runtime.Members[i].InstanceID != s.Exit.Stop.InstanceID {
						b.Runtime.Members[i].Running = false
						b.Runtime.Members[i].Namespace = NamespaceIdentity{}
					}
				}
			case "namespace":
				b.Runtime.Anchor.Namespace.Inode++
			case "stale":
				now = now.Add(31 * time.Second)
			case "still-present":
				a.Runtime.Members = cloneReadinessObservation(b.Runtime).Members
			case "after-storage":
				a.Storage.Identity.UrbInode++
			case "after-peer":
				a.Runtime.Members[0].Resource.ID = strings.Repeat("a", 64)
			case "after-namespace":
				a.Runtime.Anchor.Namespace.Inode++
			case "incomplete":
				a.Runtime.Complete = false
			case "old-absence":
				a.CapturedAt = b.CapturedAt.Add(-time.Second)
			case "future-absence":
				a.Runtime.ObservedAt = now.Add(6 * time.Second)
			}
			d := &retirementDriver{t: t, retained: cloneMemberRetirement(s), before: b, after: a}
			commit := func(_ string, n MemberRetirement) error { d.retained = cloneMemberRetirement(n); return nil }
			got, err := ExecuteMemberRetirement(context.Background(), p, s, d, commit, func() time.Time { return now })
			if err == nil || got.Phase == "absent" {
				t.Fatal("changed retirement accepted", fault, got)
			}
			afterFault := strings.HasPrefix(fault, "after-") || fault == "still-present" || fault == "incomplete" || fault == "old-absence" || fault == "future-absence"
			if !afterFault && d.removes != 0 {
				t.Fatal("invalid preflight reached removal")
			}
		})
	}
}

func TestMemberRetirementAbsenceFreshnessAndEmptyMembership(t *testing.T) {
	_, s, b, a := retirementFixture(t)
	s.Before = &b
	a.Runtime.ObservedAt = b.Runtime.ObservedAt
	if s.checkAbsent(a, a.CapturedAt.Add(30*time.Second)) == nil {
		t.Fatal("runtime observation exceeded independent freshness bound")
	}
	b.Runtime.Members = nil
	for _, m := range s.Exit.Observation.Runtime.Members {
		if m.InstanceID == s.Exit.Stop.InstanceID {
			b.Runtime.Members = append(b.Runtime.Members, m)
		}
	}
	a.Runtime = cloneReadinessObservation(b.Runtime)
	a.Runtime.ObservedAt = a.CapturedAt
	for _, members := range [][]MemberObservation{nil, {}} {
		a.Runtime.Members = members
		if err := s.checkAbsent(a, a.CapturedAt); err != nil {
			t.Fatal("empty namespace membership representation changed absence", err)
		}
	}
}
