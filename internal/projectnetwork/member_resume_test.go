package projectnetwork

import (
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func resumeFixture(t *testing.T) (Plan, MemberResume, MemberAbsenceObservation) {
	t.Helper()
	p, r, b, a := retirementFixture(t)
	var err error
	for _, phase := range []string{"remove-dispatched", "removed", "absent"} {
		var before *MemberStopObservation
		var after *MemberAbsenceObservation
		if phase == "remove-dispatched" {
			before = &b
		}
		if phase == "absent" {
			after = &a
		}
		r, err = r.Advance(p, r.Phase, phase, before, after)
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewMemberResume(p, r, 1, "resume-zod", strings.Repeat("9", 64))
	if err != nil {
		t.Fatal(err)
	}
	a.Runtime = cloneReadinessObservation(a.Runtime)
	a.CapturedAt = a.CapturedAt.Add(time.Second)
	a.Runtime.ObservedAt = a.CapturedAt
	return p, s, a
}

type resumeDriver struct {
	t                                *testing.T
	retained                         MemberResume
	before                           MemberAbsenceObservation
	fault                            string
	now                              *time.Time
	cancel                           context.CancelFunc
	creates, starts, reads, verifies int
}

func (d *resumeDriver) check(s MemberResume, phase string) {
	d.t.Helper()
	if s.Phase != phase || !reflect.DeepEqual(s, d.retained) {
		d.t.Fatal("resume effect preceded committed intent", phase)
	}
	s.Retirement.Exit.Stop.Binding.Members[0].InstanceID = "alias-edit"
	if s.Before != nil {
		s.Before.Runtime.Anchor.Active[0].HostPort++
	}
}
func (d *resumeDriver) VerifyEngine(_ context.Context, id string) error {
	d.verifies++
	if id != d.retained.Retirement.Exit.Stop.Binding.EngineID {
		d.t.Fatal("foreign engine")
	}
	if d.fault == "engine" || d.fault == "engine-dispatch" && d.verifies == 3 {
		return errors.New("private engine detail")
	}
	return nil
}
func (d *resumeDriver) InspectAbsent(_ context.Context, s MemberResume) (MemberAbsenceObservation, error) {
	d.check(s, "planned")
	o := d.before
	o.Runtime = cloneReadinessObservation(o.Runtime)
	switch d.fault {
	case "absence":
		return o, errors.New("private absence detail")
	case "stale-absence":
		o.CapturedAt = o.CapturedAt.Add(-time.Minute)
	case "changed-peer":
		o.Runtime.Members[0].Running = !o.Runtime.Members[0].Running
	case "changed-pier":
		o.Storage.Identity.Inode++
	case "changed-anchor":
		o.Runtime.Anchor.Namespace.Inode++
	}
	return o, nil
}
func (d *resumeDriver) Create(_ context.Context, s MemberResume) (string, error) {
	d.check(s, "create-dispatched")
	d.creates++
	id := strings.Repeat("8", 64)
	switch d.fault {
	case "no-id":
		return "", nil
	case "create-lost":
		return "", errors.New("private Create reply lost")
	case "create-late":
		return id, errors.New("private late error")
	case "duplicate-id":
		return s.Retirement.Exit.Stop.Binding.Members[0].Container.ID, nil
	case "cancel-create":
		d.cancel()
	}
	return id, nil
}
func (d *resumeDriver) Inspect(_ context.Context, s MemberResume) (MemberResumeObservation, error) {
	phase := "created"
	if d.reads > 0 {
		phase = "started"
	}
	d.check(s, phase)
	d.reads++
	running := phase == "started"
	o := MemberResumeObservation{Runtime: cloneReadinessObservation(d.before.Runtime), Storage: d.before.Storage, CapturedAt: d.before.CapturedAt.Add(time.Second)}
	o.Runtime.ObservedAt = o.CapturedAt
	m := MemberObservation{InstanceID: s.Retirement.Exit.Stop.InstanceID, ResourceObservation: ResourceObservation{Resource: s.Container, Present: true}, AnchorID: s.Retirement.Exit.Stop.Binding.Anchor.ID, Running: running}
	if running {
		m.Namespace = s.Retirement.Exit.Stop.Binding.Namespace
		o.StartedAt = o.CapturedAt
	}
	o.Runtime.Members = append(o.Runtime.Members, m)
	slices.SortFunc(o.Runtime.Members, func(a, b MemberObservation) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	if d.fault == "inspect-created" && !running || d.fault == "inspect-running" && running {
		return o, errors.New("private inspect detail")
	}
	if d.fault == "stale-created" && !running || d.fault == "stale-running" && running {
		o.Runtime.ObservedAt = o.Runtime.ObservedAt.Add(-time.Minute)
	}
	if d.fault == "early-start" && !running {
		o.StartedAt = o.CapturedAt
	}
	if d.fault == "wrong-namespace" && running {
		for i := range o.Runtime.Members {
			if o.Runtime.Members[i].InstanceID == m.InstanceID {
				o.Runtime.Members[i].Namespace.Inode++
			}
		}
	}
	if d.fault == "old-process" && running {
		o.StartedAt = o.CapturedAt.Add(-time.Hour)
	}
	return o, nil
}
func (d *resumeDriver) Start(_ context.Context, s MemberResume) error {
	d.check(s, "start-dispatched")
	d.starts++
	if d.fault == "start-lost" {
		return errors.New("private Start reply lost")
	}
	if d.fault == "cancel-start" {
		d.cancel()
	}
	return nil
}
func TestMemberResumeRetainsDispatchAndExactPriorEvidence(t *testing.T) {
	cases := []struct {
		fault, phase    string
		creates, starts int
	}{
		{"", "running", 1, 1}, {"cancel-before", "planned", 0, 0}, {"engine", "planned", 0, 0}, {"absence", "planned", 0, 0}, {"stale-absence", "planned", 0, 0}, {"changed-peer", "planned", 0, 0}, {"changed-pier", "planned", 0, 0}, {"changed-anchor", "planned", 0, 0},
		{"commit-create-dispatched", "planned", 0, 0}, {"engine-dispatch", "create-dispatched", 0, 0}, {"stale-create-dispatched", "create-dispatched", 0, 0}, {"no-id", "create-dispatched", 1, 0}, {"create-lost", "create-dispatched", 1, 0}, {"duplicate-id", "create-dispatched", 1, 0}, {"commit-created", "create-dispatched", 1, 0}, {"create-late", "created", 1, 0}, {"cancel-create", "created", 1, 0},
		{"inspect-created", "created", 1, 0}, {"stale-created", "created", 1, 0}, {"early-start", "created", 1, 0}, {"commit-start-dispatched", "created", 1, 0}, {"stale-start-dispatched", "start-dispatched", 1, 0}, {"start-lost", "start-dispatched", 1, 1}, {"commit-started", "start-dispatched", 1, 1}, {"cancel-start", "started", 1, 1}, {"inspect-running", "started", 1, 1}, {"stale-running", "started", 1, 1}, {"wrong-namespace", "started", 1, 1}, {"old-process", "started", 1, 1}, {"commit-running", "started", 1, 1},
	}
	for _, c := range cases {
		t.Run(c.fault, func(t *testing.T) {
			p, s, b := resumeFixture(t)
			old := cloneMemberResume(s)
			now := b.CapturedAt.Add(time.Second)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &resumeDriver{t: t, retained: cloneMemberResume(s), before: b, fault: c.fault, now: &now, cancel: cancel}
			if c.fault == "cancel-before" {
				cancel()
			}
			commit := func(expected string, next MemberResume) error {
				if expected != d.retained.Phase {
					t.Fatal("stale expected phase")
				}
				if c.fault == "commit-"+next.Phase {
					return errors.New("commit failed")
				}
				d.retained = cloneMemberResume(next)
				if c.fault == "stale-"+next.Phase {
					now = now.Add(time.Minute)
				}
				next.Retirement.Exit.Stop.Binding.Members[0].InstanceID = "mutated"
				if next.Before != nil {
					next.Before.Runtime.Anchor.Active[0].HostPort++
				}
				return nil
			}
			got, err := ExecuteMemberResume(ctx, p, s, d, commit, func() time.Time { return now })
			if (err == nil) != (c.fault == "") || got.Phase != c.phase || d.creates != c.creates || d.starts != c.starts || !reflect.DeepEqual(got, d.retained) {
				t.Fatal(c, got.Phase, err, d.creates, d.starts)
			}
			if !reflect.DeepEqual(s, old) || !reflect.DeepEqual(got.Retirement, old.Retirement) {
				t.Fatal("resume changed original evidence")
			}
			if got.Validate(p) != nil {
				t.Fatal("returned invalid retained state")
			}
			if got.Phase != "planned" {
				v := d.verifies
				if _, err = ExecuteMemberResume(context.Background(), p, got, d, commit, func() time.Time { return now }); err == nil || v != d.verifies {
					t.Fatal("resumed an old dispatch")
				}
			}
		})
	}
}
func TestMemberResumeRefusesInvalidIntentAndAliasing(t *testing.T) {
	p, s, b := resumeFixture(t)
	cases := map[string]func(*MemberResume){
		"zero-generation": func(s *MemberResume) { s.PreviousGeneration = 0 }, "overflow": func(s *MemberResume) { s.PreviousGeneration = math.MaxUint64 }, "unfinished-retirement": func(s *MemberResume) { s.Retirement.Phase = "removed"; s.Retirement.After = nil }, "unclean": func(s *MemberResume) { s.Retirement.Exit.Observation.ExitCode = 137 }, "old-operation": func(s *MemberResume) { s.OperationID = s.Retirement.Exit.Stop.OperationID }, "unknown-phase": func(s *MemberResume) { s.Phase = "resume" }, "premature-id": func(s *MemberResume) { s.Container.ID = strings.Repeat("8", 64) }, "premature-before": func(s *MemberResume) { s.Before = &b }, "bad-contract": func(s *MemberResume) { s.Container.ContractDigest = "unknown" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			v := cloneMemberResume(s)
			mutate(&v)
			if v.Validate(p) == nil {
				t.Fatal("accepted invalid resume")
			}
		})
	}
	v, err := s.Advance(p, "planned", "create-dispatched", MemberResumeResult{Before: &b})
	if err != nil {
		t.Fatal(err)
	}
	b.Runtime.Anchor.Active[0].HostPort++
	if v.Validate(p) != nil {
		t.Fatal("input observation aliases journal")
	}
	if _, err = s.Advance(p, "planned", "created", MemberResumeResult{ContainerID: strings.Repeat("8", 64)}); err == nil {
		t.Fatal("skipped durable dispatch")
	}
	if _, err = v.Advance(p, "planned", "created", MemberResumeResult{ContainerID: strings.Repeat("8", 64)}); err == nil {
		t.Fatal("accepted stale phase")
	}
	if _, err = v.Advance(p, "create-dispatched", "created", MemberResumeResult{ContainerID: s.Retirement.Exit.Stop.Binding.Anchor.ID}); err == nil {
		t.Fatal("adopted anchor")
	}
	r := cloneMemberRetirement(s.Retirement)
	created, err := NewMemberResume(p, r, 1, "other-resume", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	r.Exit.Stop.Binding.Members[0].InstanceID = "changed"
	if created.Validate(p) != nil {
		t.Fatal("constructor aliases retirement")
	}
}
