package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func anchorRecoveryReviewFixture(t *testing.T, stopped bool) (Plan, []MemberLaunch, []MemberResume, []InitialMemberReadiness, []CurrentMemberReadiness, AnchorRecoveryContext, AnchorRecoveryObservation) {
	t.Helper()
	p, launches, resumes, original, ready, _ := currentStopModelFixture(t, "nec")
	p2, l2, r2, o2, ready2, _ := currentStopModelFixture(t, "zod")
	if !reflect.DeepEqual(p, p2) || !reflect.DeepEqual(launches, l2) || !reflect.DeepEqual(resumes, r2) {
		t.Fatal("fixture histories diverge")
	}
	originals := []InitialMemberReadiness{original, o2}
	fresh := []CurrentMemberReadiness{ready, ready2}
	c, err := SelectAnchorRecoveryContext(p, launches, resumes, originals, fresh)
	if err != nil {
		t.Fatal(err)
	}
	runtime := cloneReadinessObservation(ready.After.Runtime)
	runtime.ObservedAt = ready.CapturedAt.Add(time.Second)
	if stopped {
		runtime.Anchor.Running = false
		runtime.Anchor.Namespace = NamespaceIdentity{}
		runtime.Anchor.Active = nil
		runtime.BridgeAttachments = nil
	} else {
		runtime.Anchor.Namespace.Inode++
	}
	o := AnchorRecoveryObservation{Runtime: runtime, CapturedAt: runtime.ObservedAt}
	for _, m := range c.Members {
		o.Processes = append(o.Processes, AnchorRecoveryProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, StartedAt: m.StartedAt})
	}
	return p, launches, resumes, originals, fresh, c, o
}

type recoveryObserverFixture struct {
	t                     *testing.T
	expected              AnchorRecoveryContext
	before, after         AnchorRecoveryObservation
	fault                 string
	inspections, verifies int
	cancel                context.CancelFunc
}

func (d *recoveryObserverFixture) VerifyEngine(_ context.Context, id string) error {
	d.verifies++
	if id != d.expected.Binding.EngineID {
		d.t.Fatal("wrong engine")
	}
	if d.fault == "engine" || d.fault == "late-engine" && d.verifies == 3 {
		return errors.New("private engine diagnostic")
	}
	return nil
}
func (d *recoveryObserverFixture) InspectAnchorRecovery(_ context.Context, c AnchorRecoveryContext) (AnchorRecoveryObservation, error) {
	d.inspections++
	if !reflect.DeepEqual(c, d.expected) {
		d.t.Fatal("context changed")
	}
	if d.fault == "inspect" {
		return AnchorRecoveryObservation{}, errors.New("private container detail")
	}
	if d.fault == "cancel" {
		d.cancel()
	}
	if d.fault == "mutate-context" {
		c.Binding.Members[0].InstanceID = "foreign"
		c.Members[0].Context.Binding.Members[0].InstanceID = "foreign"
	}
	if d.inspections == 1 {
		return d.before, nil
	}
	if d.fault == "mutate-prior-buffer" {
		d.before.Processes[0].InstanceID = "foreign"
		d.before.Runtime.Members[0].InstanceID = "foreign"
	}
	return d.after, nil
}
func TestAnchorRecoveryReviewPreservesOldMembershipAndProcess(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		p, launches, resumes, originals, ready, c, b := anchorRecoveryReviewFixture(t, stopped)
		after := cloneAnchorRecoveryObservation(b)
		after.CapturedAt = after.CapturedAt.Add(time.Second)
		after.Runtime.ObservedAt = after.CapturedAt
		for _, fault := range []string{"", "mutate-context", "mutate-prior-buffer"} {
			d := &recoveryObserverFixture{t: t, expected: c, before: cloneAnchorRecoveryObservation(b), after: cloneAnchorRecoveryObservation(after), fault: fault}
			got, err := CollectAnchorRecoveryReview(context.Background(), p, launches, resumes, originals, ready, d, func() time.Time { return after.CapturedAt })
			if err != nil || got.Validate(p, launches, resumes, originals, ready) != nil || !reflect.DeepEqual(got.Context, c) || d.inspections != 2 || d.verifies != 3 {
				t.Fatal(got.Assessment, err)
			}
			want := "namespace-changed"
			if stopped {
				want = "anchor-unavailable"
			}
			if got.Assessment.State != want || !got.Assessment.RecoveryRequired || !reflect.DeepEqual(got.Before, b) {
				t.Fatal("wrong fault review")
			}
			got.Context.Binding.Members[0].InstanceID = "caller-change"
			got.Context.Members[0].Context.Binding.Members[0].InstanceID = "caller-change"
			if !reflect.DeepEqual(d.expected, c) || ready[0].InstanceID != "nec" {
				t.Fatal("review aliases source")
			}
		}
	}
}
func TestAnchorRecoveryReviewRejectsUnsafeOrIncompleteObservation(t *testing.T) {
	faults := map[string]func(*AnchorRecoveryObservation){
		"healthy": func(o *AnchorRecoveryObservation) { o.Runtime.Anchor.Namespace.Inode-- },
		"stale": func(o *AnchorRecoveryObservation) {
			o.CapturedAt = o.CapturedAt.Add(-time.Minute)
			o.Runtime.ObservedAt = o.CapturedAt
		},
		"future": func(o *AnchorRecoveryObservation) {
			o.CapturedAt = o.CapturedAt.Add(time.Minute)
			o.Runtime.ObservedAt = o.CapturedAt
		},
		"runtime-future":    func(o *AnchorRecoveryObservation) { o.Runtime.ObservedAt = o.CapturedAt.Add(time.Second) },
		"incomplete":        func(o *AnchorRecoveryObservation) { o.Runtime.Complete = false },
		"engine":            func(o *AnchorRecoveryObservation) { o.Runtime.EngineID = "different" },
		"missing-process":   func(o *AnchorRecoveryObservation) { o.Processes = o.Processes[1:] },
		"duplicate-process": func(o *AnchorRecoveryObservation) { o.Processes[1] = o.Processes[0] },
		"foreign-process":   func(o *AnchorRecoveryObservation) { o.Processes[0].InstanceID = "foreign" },
		"changed-process": func(o *AnchorRecoveryObservation) {
			o.Processes[0].StartedAt = o.Processes[0].StartedAt.Add(time.Second)
		},
		"changed-pier":   func(o *AnchorRecoveryObservation) { o.Processes[0].Storage.Identity.Inode++ },
		"changed-parent": func(o *AnchorRecoveryObservation) { o.Processes[0].Storage.MountIdentity.Inode++ },
		"member-exited": func(o *AnchorRecoveryObservation) {
			o.Runtime.Members[0].Running = false
			o.Runtime.Members[0].Namespace = NamespaceIdentity{}
		},
		"member-migrated": func(o *AnchorRecoveryObservation) { o.Runtime.Members[0].Namespace = o.Runtime.Anchor.Namespace },
		"other-boot": func(o *AnchorRecoveryObservation) {
			o.Runtime.Anchor.Namespace.BootID = "11234567-0123-0123-0123-0123456789ab"
		},
		"foreign-attachment": func(o *AnchorRecoveryObservation) {
			o.Runtime.BridgeAttachments = append(o.Runtime.BridgeAttachments, "unknown")
		},
		"missing-publication": func(o *AnchorRecoveryObservation) { o.Runtime.Anchor.Active = nil },
		"foreign-publication": func(o *AnchorRecoveryObservation) { o.Runtime.Anchor.Active[0].HostAddress = "0.0.0.0" },
		"missing-anchor": func(o *AnchorRecoveryObservation) {
			o.Runtime.Anchor = AnchorObservation{ResourceObservation: ResourceObservation{Resource: ResourceIdentity{ID: o.Runtime.Anchor.Resource.ID}}}
			o.Runtime.BridgeAttachments = nil
		},
	}
	for name, mutate := range faults {
		t.Run(name, func(t *testing.T) {
			p, l, r, originals, ready, c, b := anchorRecoveryReviewFixture(t, false)
			a := cloneAnchorRecoveryObservation(b)
			mutate(&a)
			d := &recoveryObserverFixture{t: t, expected: c, before: b, after: a}
			got, err := CollectAnchorRecoveryReview(context.Background(), p, l, r, originals, ready, d, func() time.Time { return b.CapturedAt })
			if err == nil || !reflect.DeepEqual(got, AnchorRecoveryReview{}) {
				t.Fatal("unsafe review accepted", got, err)
			}
		})
	}
}
func TestAnchorRecoveryReviewRejectsChangedContextOrUncertainRead(t *testing.T) {
	for _, fault := range []string{"missing-original", "duplicate-ready", "later-member", "old-binding", "bad-readiness", "engine", "late-engine", "inspect", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			p, l, r, originals, ready, c, b := anchorRecoveryReviewFixture(t, false)
			switch fault {
			case "missing-original":
				originals = originals[1:]
			case "duplicate-ready":
				ready[1] = ready[0]
			case "later-member":
				r = append(r, nextGenerationFixture(t, p, r[0]))
			case "old-binding":
				ready[0].Binding = originals[0].Binding
			case "bad-readiness":
				ready[0].After.StartedAt = ready[0].After.StartedAt.Add(time.Second)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &recoveryObserverFixture{t: t, expected: c, before: b, after: b, fault: fault, cancel: cancel}
			got, err := CollectAnchorRecoveryReview(ctx, p, l, r, originals, ready, d, func() time.Time { return b.CapturedAt })
			if err == nil || !reflect.DeepEqual(got, AnchorRecoveryReview{}) {
				t.Fatal("uncertain read accepted")
			}
			if fault == "cancel" && d.inspections != 1 {
				t.Fatal("continued after cancellation")
			}
		})
	}
}

func TestAnchorRecoveryReviewDoesNotRequireProactiveHealthyRefresh(t *testing.T) {
	p, l, r, originals, _, _, b := anchorRecoveryReviewFixture(t, false)
	c, err := SelectAnchorRecoveryContext(p, l, r, originals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Members[0].StartedAt.IsZero() || c.Members[1].StartedAt.IsZero() {
		t.Fatal("initial versus replacement process history lost")
	}
	for _, fault := range []string{"", "original-restarted-between-reads", "replacement-restarted"} {
		before := cloneAnchorRecoveryObservation(b)
		after := cloneAnchorRecoveryObservation(b)
		after.CapturedAt = after.CapturedAt.Add(time.Second)
		after.Runtime.ObservedAt = after.CapturedAt
		if fault == "original-restarted-between-reads" {
			after.Processes[0].StartedAt = after.Processes[0].StartedAt.Add(time.Second)
		}
		if fault == "replacement-restarted" {
			before.Processes[1].StartedAt = before.Processes[1].StartedAt.Add(time.Second)
			after.Processes[1].StartedAt = before.Processes[1].StartedAt
		}
		d := &recoveryObserverFixture{t: t, expected: c, before: before, after: after}
		got, err := CollectAnchorRecoveryReview(context.Background(), p, l, r, originals, nil, d, func() time.Time { return after.CapturedAt })
		if fault == "" {
			if err != nil || got.Validate(p, l, r, originals, nil) != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("changed process accepted", fault)
		}
	}
}

func TestAnchorRecoveryReviewCanRetainSameCreatorEvidenceAcrossPeerChange(t *testing.T) {
	p, l, r, originals, ready, _, b := anchorRecoveryReviewFixture(t, false)
	r = append(r, nextGenerationFixture(t, p, r[0]))
	// The original nec creator still matches. Its healthy proof's old peer
	// generation remains historical evidence, not a reason to lose its start time.
	ready = ready[:1]
	c, err := SelectAnchorRecoveryContext(p, l, r, originals, ready)
	if err != nil {
		t.Fatal(err)
	}
	if c.Members[0].StartedAt != ready[0].After.StartedAt || c.Members[1].Context.Member.Generation != 3 {
		t.Fatal("wrong current ownership")
	}
	latest := c.Members[0].NotBefore
	for _, m := range c.Members {
		if m.NotBefore.After(latest) {
			latest = m.NotBefore
		}
	}
	b.CapturedAt = latest.Add(time.Second)
	b.Runtime.ObservedAt = b.CapturedAt
	b.Processes = nil
	for i, m := range c.Members {
		b.Runtime.Members[i].Resource = m.Context.Member.Container
		b.Processes = append(b.Processes, AnchorRecoveryProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, StartedAt: m.StartedAt})
	}
	d := &recoveryObserverFixture{t: t, expected: c, before: b, after: cloneAnchorRecoveryObservation(b)}
	got, err := CollectAnchorRecoveryReview(context.Background(), p, l, r, originals, ready, d, func() time.Time { return b.CapturedAt })
	if err != nil || got.Validate(p, l, r, originals, ready) != nil {
		t.Fatal(err)
	}
}
