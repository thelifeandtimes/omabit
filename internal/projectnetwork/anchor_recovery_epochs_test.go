package projectnetwork

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func secondRecoveryFixture(t *testing.T, h AnchorRecoveryHistory) AnchorRecoveryReadiness {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := SelectAnchorRecoveryHistoryContext(h)
	must(err)
	runtime := cloneReadinessObservation(h.Epochs[len(h.Epochs)-1].Readiness.After.Runtime)
	if len(h.Resumes) > 0 {
		runtime = cloneReadinessObservation(h.Resumes[len(h.Resumes)-1].After.Runtime)
	}
	now := c.Members[0].NotBefore
	for _, m := range c.Members {
		if m.NotBefore.After(now) {
			now = m.NotBefore
		}
	}
	now = now.Add(time.Second)
	runtime.ObservedAt = now
	runtime.Anchor.Namespace.Inode++
	b := AnchorRecoveryObservation{Runtime: runtime, CapturedAt: now}
	for _, m := range c.Members {
		b.Processes = append(b.Processes, AnchorRecoveryProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, StartedAt: m.StartedAt})
	}
	a := cloneAnchorRecoveryObservation(b)
	a.CapturedAt = now.Add(time.Second)
	a.Runtime.ObservedAt = a.CapturedAt
	observer := &recoveryObserverFixture{t: t, expected: c, before: b, after: a}
	review, err := CollectAnchorRecoveryHistoryReview(context.Background(), h, observer, func() time.Time { return a.CapturedAt })
	must(err)
	drain, err := NewAnchorDrain(h, review, "drain-second")
	must(err)
	d := &anchorDrainFixture{t: t, history: h, saved: drain, now: a.CapturedAt, state: AnchorDrainObservation{Runtime: cloneReadinessObservation(a.Runtime), CapturedAt: a.CapturedAt, AnchorProcess: RecoveryProcessState{StartedAt: now.Add(-time.Second), FinishedAt: now.Add(-2 * time.Second)}}}
	for _, p := range a.Processes {
		d.state.Processes = append(d.state.Processes, AnchorDrainProcess{InstanceID: p.InstanceID, Storage: p.Storage, Process: RecoveryProcessState{StartedAt: p.StartedAt}})
	}
	for !d.saved.Clean() {
		_, err = ExecuteNextAnchorDrain(context.Background(), h, d.saved, d, d.commit, d.clock)
		must(err)
	}
	ns, err := NewAnchorNamespace(h, d.saved, "namespace-second")
	must(err)
	n := &anchorNamespaceFixture{t: t, history: h, saved: ns, now: d.now, state: cloneAnchorDrainObservation(d.state)}
	ns, err = ExecuteAnchorNamespace(context.Background(), h, n.saved, n, n.commit, n.clock)
	must(err)
	retired, err := NewAnchorRetirement(h, ns, "retire-second")
	must(err)
	state := AnchorRetirementObservation{Runtime: cloneReadinessObservation(ns.After.Runtime), AnchorProcess: ns.After.AnchorProcess, Processes: slices.Clone(ns.After.Processes), CapturedAt: ns.After.CapturedAt}
	for _, p := range ns.After.Processes {
		state.Piers = append(state.Piers, RecoveryPier{InstanceID: p.InstanceID, Storage: p.Storage})
	}
	rt := &anchorRetirementFixture{t: t, history: h, saved: retired, now: n.now, state: state}
	for !rt.saved.Complete() {
		_, err = ExecuteNextAnchorRetirement(context.Background(), h, rt.saved, rt, rt.commit, rt.clock)
		must(err)
	}
	var members []RecoveredMember
	for _, m := range rt.saved.Members {
		creation, err := SelectRecoveryMemberCreation(h, rt.saved, m.InstanceID, "second-"+m.InstanceID)
		must(err)
		members = append(members, RecoveredMember{Creation: creation, Container: ResourceIdentity{ContractDigest: strings.Repeat("d", 64)}, Phase: "planned"})
	}
	replaced, err := NewAnchorReplacement(h, rt.saved, "replace-second", members)
	must(err)
	f := &anchorReplacementFixture{t: t, h: h, saved: replaced, state: cloneAnchorRetirementObservation(*rt.saved.Members[len(rt.saved.Members)-1].After), now: rt.now, idBase: 300}
	for !f.saved.Running() {
		_, err = ExecuteNextAnchorReplacement(context.Background(), h, f.saved, f, f.commit, f.clock)
		must(err)
	}
	ready := &recoveryReadinessFixture{t: t, h: h, s: f.saved, now: f.now}
	result, err := CollectAnchorRecoveryReadiness(context.Background(), h, f.saved, ready, ready.clock)
	must(err)
	return result
}
func TestAnchorRecoveryEpochsSecondRecovery(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	original := h.Clone()
	first := r.Clone()
	next, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	next.Resumes = tail
	c, err := SelectAnchorRecoveryHistoryContext(next)
	if err != nil {
		t.Fatal(err)
	}
	last, err := ResolveRecoveredMemberGenerations(h, r, tail)
	if err != nil || !reflect.DeepEqual(c.Binding, last.Binding) {
		t.Fatal("lost ordinary suffix", err)
	}
	second := secondRecoveryFixture(t, next)
	completed, err := AppendAnchorRecoveryEpoch(next, second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveAnchorRecoveryHistory(completed)
	if err != nil {
		t.Fatal(err)
	}
	if len(completed.Epochs) != 2 || got.Binding.Namespace == r.Binding.Namespace || got.Binding.Generation != r.Binding.Generation {
		t.Fatal("recovery boundaries collapsed")
	}
	for i, m := range got.Members {
		if m.Generation != last.Members[i].Generation+1 {
			t.Fatal("lost prior member generation")
		}
	}
	initial, err := InitialRuntimeBinding(h.Plan, h.Launches)
	if err != nil {
		t.Fatal(err)
	}
	for _, binding := range []RuntimeBinding{initial, r.Binding, last.Binding, second.Binding} {
		prefix, err := AnchorRecoveryHistoryAtBinding(completed, binding)
		if err != nil {
			t.Fatal("missing historical prefix", err)
		}
		selected, err := ResolveAnchorRecoveryHistory(prefix)
		if err != nil || !reflect.DeepEqual(selected.Binding, binding) {
			t.Fatal("prefix changed", err)
		}
	}
	context, err := SelectRecoveredMemberContext(next, second, nil, got.Members[0].InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	ordinary := ordinaryResumeAfterRecovery(t, h.Plan, context, second.After.Runtime, "after-second", 500)
	completed.Resumes = []MemberResume{ordinary}
	if _, err = ResolveAnchorRecoveryHistory(completed); err != nil {
		t.Fatal("ordinary replacement after second recovery", err)
	}
	if !reflect.DeepEqual(h, original) || !reflect.DeepEqual(r, first) {
		t.Fatal("rewrote original or previous recovery")
	}
}

func TestAnchorRecoveryEpochsRejectChangedHistory(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	next, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	next.Resumes = tail
	second := secondRecoveryFixture(t, next)
	completed, err := AppendAnchorRecoveryEpoch(next, second)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := InitialRuntimeBinding(h.Plan, h.Launches)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*AnchorRecoveryHistory){
		"missing-initial-resume": func(h *AnchorRecoveryHistory) { h.Epochs[0].Resumes = nil },
		"missing-middle-resume":  func(h *AnchorRecoveryHistory) { h.Epochs[1].Resumes = h.Epochs[1].Resumes[1:] },
		"reordered-boundaries":   func(h *AnchorRecoveryHistory) { h.Epochs[0], h.Epochs[1] = h.Epochs[1], h.Epochs[0] },
		"incomplete-second": func(h *AnchorRecoveryHistory) {
			h.Epochs[1].Readiness.Replacement.Members[0].Phase = "start-dispatched"
		},
		"wrong-second-identity": func(h *AnchorRecoveryHistory) { h.Epochs[1].Readiness.Members[0].Ship = "~bus" },
		"duplicate-boundary":    func(h *AnchorRecoveryHistory) { h.Epochs = append(h.Epochs, h.Epochs[1]) },
		"second-process-restarted": func(h *AnchorRecoveryHistory) {
			h.Epochs[1].Readiness.After.Processes[0].Process.StartedAt = h.Epochs[1].Readiness.After.Processes[0].Process.StartedAt.Add(time.Nanosecond)
		},
		"ordinary-tail-in-previous-namespace": func(h *AnchorRecoveryHistory) { h.Resumes = tail },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			v := completed.Clone()
			mutate(&v)
			if _, err := ResolveAnchorRecoveryHistory(v); err == nil {
				t.Fatal("invalid causal history accepted")
			}
			if _, err := AnchorRecoveryHistoryAtBinding(v, initial); err == nil {
				t.Fatal("old prefix concealed invalid future history")
			}
			if _, err := ResolveAnchorRecoveryHistory(completed); err != nil {
				t.Fatal("clone changed original", err)
			}
		})
	}
	source := next.Clone()
	review := second.Replacement.Retirement.Namespace.Drain.Review
	if source.validateReview(review) != nil {
		t.Fatal("second review not valid before collision tests")
	}
	if _, err := NewAnchorDrain(source, review, r.Replacement.Retirement.Namespace.Drain.OperationID); err == nil {
		t.Fatal("reused previous drain operation")
	}
	if _, err := NewAnchorNamespace(source, second.Replacement.Retirement.Namespace.Drain, r.Replacement.Retirement.Namespace.OperationID); err == nil {
		t.Fatal("reused previous namespace operation")
	}
	if _, err := NewAnchorRetirement(source, second.Replacement.Retirement.Namespace, r.Replacement.Retirement.OperationID); err == nil {
		t.Fatal("reused previous retirement operation")
	}
	if _, err := SelectRecoveryMemberCreation(source, second.Replacement.Retirement, second.Replacement.Members[0].Creation.InstanceID, r.Replacement.Members[0].Creation.OperationID); err == nil {
		t.Fatal("reused previous member creator")
	}
	// The resource inventory includes retired containers from every earlier epoch.
	for _, m := range r.Replacement.Members {
		if !recoveryHistoryContainerIDs(source)[m.Container.ID] {
			t.Fatal("forgot retired prior-epoch ID")
		}
	}
	if _, err := AppendAnchorRecoveryEpoch(next, r); err == nil {
		t.Fatal("earlier completion replayed in new namespace")
	}
}

func TestAnchorRecoveryEpochsRetainHistoricalReviewPrerequisites(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	next, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	next.Resumes = tail
	// This peer's current creator survives the later replacement of the other ship.
	instance := r.Replacement.Members[1].Creation.InstanceID
	c, err := SelectRecoveredMemberContext(h, r, tail[:2], instance)
	if err != nil {
		t.Fatal(err)
	}
	before := MemberStopObservation{Runtime: cloneReadinessObservation(tail[1].After.Runtime), Storage: c.Storage, StartedAt: c.KnownStartedAt, CapturedAt: c.NotBefore.Add(time.Millisecond)}
	before.Runtime.ObservedAt = before.CapturedAt
	after := cloneMemberStopObservation(before)
	after.CapturedAt = after.CapturedAt.Add(time.Millisecond)
	after.Runtime.ObservedAt = after.CapturedAt
	member, endpoint, err := currentReadinessEndpoint(h.Plan, instance)
	if err != nil {
		t.Fatal(err)
	}
	proof := CurrentMemberReadiness{InstanceID: instance, CreatorOperationID: c.Member.OperationID, ContainerGeneration: c.Member.Generation, Binding: c.Binding, Storage: c.Storage, Ship: member.Ship, Endpoint: endpoint, Before: before, After: after, CapturedAt: after.CapturedAt}
	if err = proof.validateContext(h.Plan, c); err != nil {
		t.Fatal("invalid prerequisite fixture", err)
	}
	next.Current = []CurrentMemberReadiness{proof}
	selected, err := SelectAnchorRecoveryHistoryContext(next)
	if err != nil {
		t.Fatal("valid historical prerequisite refused", err)
	}
	if selected.Binding.Namespace != r.Binding.Namespace {
		t.Fatal("wrong namespace")
	}
	for _, mode := range []string{"duplicate", "unknown-member", "wrong-process", "old-namespace", "wrong-creator"} {
		t.Run(mode, func(t *testing.T) {
			v := next.Clone()
			switch mode {
			case "duplicate":
				v.Current = append(v.Current, v.Current[0])
			case "unknown-member":
				v.Current[0].InstanceID = "foreign"
			case "wrong-process":
				v.Current[0].After.StartedAt = v.Current[0].After.StartedAt.Add(time.Nanosecond)
			case "old-namespace":
				v.Current[0] = h.Current[0]
			case "wrong-creator":
				v.Current[0].CreatorOperationID = "foreign-creator"
			}
			if _, err = SelectAnchorRecoveryHistoryContext(v); err == nil {
				t.Fatal("invalid prerequisite accepted")
			}
		})
	}
	if _, err = SelectAnchorRecoveryHistoryContext(next); err != nil {
		t.Fatal("clone changed prerequisites", err)
	}
}
