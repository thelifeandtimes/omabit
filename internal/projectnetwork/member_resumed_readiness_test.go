package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func resumedReadinessFixture(t *testing.T) (Plan, MemberResume) {
	t.Helper()
	p, s, b := resumeFixture(t)
	now := b.CapturedAt.Add(time.Second)
	d := &resumeDriver{t: t, retained: cloneMemberResume(s), before: b, now: &now}
	result, err := ExecuteMemberResume(context.Background(), p, s, d, func(_ string, v MemberResume) error { d.retained = cloneMemberResume(v); return nil }, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return p, result
}

type resumedReadinessDriver struct {
	t                            *testing.T
	saved                        MemberResume
	before, after                MemberResumeObservation
	reads, verifies, inspections int
	fault                        string
	cancel                       context.CancelFunc
}

func (d *resumedReadinessDriver) VerifyEngine(context.Context, string) error {
	d.verifies++
	if d.fault == "engine" || d.fault == "late-engine" && d.verifies == 4 {
		return errors.New("private engine detail")
	}
	return nil
}
func (d *resumedReadinessDriver) Inspect(_ context.Context, s MemberResume) (MemberResumeObservation, error) {
	if !reflect.DeepEqual(s, d.saved) {
		d.t.Fatal("changed selection")
	}
	s.After.Runtime.Members[0].InstanceID = "driver-alias"
	s.Retirement.Exit.Stop.Binding.Members[0].InstanceID = "driver-alias"
	d.inspections++
	if d.fault == "inspect" || d.fault == "late-inspect" && d.inspections == 2 {
		return MemberResumeObservation{}, errors.New("private inspect detail")
	}
	if d.inspections == 1 {
		return d.before, nil
	}
	return d.after, nil
}
func (d *resumedReadinessDriver) ReadHostIdentity(_ context.Context, endpoint string) (string, error) {
	d.reads++
	if endpoint != "http://127.0.0.1:18081/~/host" {
		d.t.Fatal("unexpected endpoint", endpoint)
	}
	if d.fault == "http" {
		return "", errors.New("private HTTP detail")
	}
	if d.fault == "cancel-http" {
		d.cancel()
	}
	if d.fault == "identity" {
		return "~nec", nil
	}
	// A driver retaining its returned buffers must not mutate collected evidence.
	d.before.Runtime.Members[0].InstanceID = "buffer-alias"
	return "~zod", nil
}
func TestResumedReadinessBracketsExactReplacement(t *testing.T) {
	cases := []string{"pass", "engine", "late-engine", "inspect", "late-inspect", "http", "identity", "cancel-before", "cancel-http", "stale-before", "stale-after", "changed-peer", "changed-container", "changed-process", "changed-pier", "changed-namespace", "incomplete", "old-generation", "target-stopped", "reversed-time"}
	for _, fault := range cases {
		t.Run(fault, func(t *testing.T) {
			p, s := resumedReadinessFixture(t)
			original := cloneMemberResume(s)
			before := cloneResumeObservation(*s.After)
			before.CapturedAt = before.CapturedAt.Add(time.Second)
			before.Runtime.ObservedAt = before.CapturedAt
			after := cloneResumeObservation(before)
			after.CapturedAt = after.CapturedAt.Add(time.Second)
			after.Runtime.ObservedAt = after.CapturedAt
			now := after.CapturedAt
			switch fault {
			case "stale-before":
				before.Runtime.ObservedAt = before.Runtime.ObservedAt.Add(-time.Minute)
			case "stale-after":
				after.CapturedAt = after.CapturedAt.Add(-time.Minute)
			case "changed-peer":
				after.Runtime.Members[0].Running = !after.Runtime.Members[0].Running
			case "changed-container":
				after.Runtime.Members[1].Resource.ID = strings.Repeat("7", 64)
			case "changed-process":
				after.StartedAt = after.StartedAt.Add(time.Second)
			case "changed-pier":
				after.Storage.Identity.Inode++
			case "changed-namespace":
				after.Runtime.Anchor.Namespace.Inode++
			case "incomplete":
				after.Runtime.Complete = false
			case "old-generation":
				after.Runtime.Generation++
			case "target-stopped":
				after.Runtime.Members[1].Running = false
				after.Runtime.Members[1].Namespace = NamespaceIdentity{}
			case "reversed-time":
				after.CapturedAt = before.CapturedAt.Add(-time.Second)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancel-before" {
				cancel()
			}
			d := &resumedReadinessDriver{t: t, saved: cloneMemberResume(s), before: before, after: after, fault: fault, cancel: cancel}
			r, err := CollectResumedMemberReadiness(ctx, p, s, d, func() time.Time { return now })
			if fault == "pass" {
				if err != nil || r.Validate(p, s) != nil || d.reads != 1 || d.inspections != 2 {
					t.Fatal("readiness failed", err, r)
				}
				if r.Binding.Generation != s.Retirement.Exit.Stop.Binding.Generation || r.ContainerGeneration != 2 || r.Binding.Members[1].Container != s.Container {
					t.Fatal("project/member generation confused")
				}
				r.Binding.Members[0].InstanceID = "caller-alias"
			} else if err == nil || !reflect.DeepEqual(r, ResumedMemberReadiness{}) || strings.Contains(err.Error(), "private") {
				t.Fatal("failure leaked evidence", r, err)
			}
			if !reflect.DeepEqual(s, original) {
				t.Fatal("resume history mutated")
			}
		})
	}
}
func TestResumedReadinessRejectsIncompleteHistory(t *testing.T) {
	for _, mode := range []string{"unfinished", "nil-after", "operation", "generation", "binding", "endpoint", "identity", "capture", "storage"} {
		t.Run(mode, func(t *testing.T) {
			p, s := resumedReadinessFixture(t)
			b, err := ResumedRuntimeBinding(p, s)
			if err != nil {
				t.Fatal(err)
			}
			r := ResumedMemberReadiness{InstanceID: "zod", ResumeOperationID: s.OperationID, ContainerGeneration: 2, Binding: b, Storage: s.After.Storage, Ship: "~zod", Endpoint: "http://127.0.0.1:18081/~/host", Before: cloneResumeObservation(*s.After), After: cloneResumeObservation(*s.After), CapturedAt: s.After.CapturedAt}
			if r.Validate(p, s) != nil {
				t.Fatal("bad fixture")
			}
			switch mode {
			case "unfinished":
				s.Phase = "started"
				s.After = nil
			case "nil-after":
				s.After = nil
			case "operation":
				r.ResumeOperationID = "another-resume"
			case "generation":
				r.ContainerGeneration++
			case "binding":
				r.Binding.Members[1].Container = s.Retirement.Exit.Stop.Binding.Members[1].Container
			case "endpoint":
				r.Endpoint = "http://localhost:18081/~/host"
			case "identity":
				r.Ship = "~nec"
			case "capture":
				r.CapturedAt = time.Time{}
			case "storage":
				r.Storage.Identity.Inode++
			}
			if r.Validate(p, s) == nil {
				t.Fatal("changed history accepted")
			}
		})
	}
}
