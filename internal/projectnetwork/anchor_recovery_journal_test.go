package projectnetwork

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func journalReadinessFixture(t *testing.T, stopped bool) (AnchorRecoveryHistory, AnchorRecoveryReadiness) {
	t.Helper()
	h, s, now := completedReplacementForReadiness(t, stopped)
	f := &recoveryReadinessFixture{t: t, h: h, s: s, now: now}
	ready, err := CollectAnchorRecoveryReadiness(context.Background(), h, s, f, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	return h, ready
}
func recoveryJournalIDs(ready AnchorRecoveryReadiness) RecoveryJournalIDs {
	r := ready.Replacement
	ids := RecoveryJournalIDs{Drain: r.Retirement.Namespace.Drain.OperationID, Namespace: r.Retirement.Namespace.OperationID, Retirement: r.Retirement.OperationID, Replacement: r.OperationID}
	for _, m := range r.Members {
		ids.Members = append(ids.Members, RecoveryJournalMemberID{m.Creation.InstanceID, m.Creation.OperationID})
	}
	return ids
}
func replayRecoveryJournal(t *testing.T, h AnchorRecoveryHistory, ready AnchorRecoveryReadiness, ack bool) []AnchorRecoveryJournal {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	ids := recoveryJournalIDs(ready)
	j, err := NewAnchorRecoveryJournal(h, ids)
	must(err)
	original := h.Clone()
	trace := []AnchorRecoveryJournal{j.Clone()}
	step := func(phase string, u AnchorRecoveryJournal) {
		t.Helper()
		// Round-trip every boundary as a future store reopen would, then require an
		// exact serial next step. Duplicate receipts cannot be dispatch authority.
		body, err := json.Marshal(j)
		must(err)
		var reopened AnchorRecoveryJournal
		must(json.Unmarshal(body, &reopened))
		must(reopened.Validate(h))
		next, err := reopened.Advance(h, reopened.Sequence, phase, u)
		must(err)
		if _, err = reopened.Advance(h, reopened.Sequence+1, phase, u); err == nil {
			t.Fatal("stale sequence accepted")
		}
		if _, err = next.Advance(h, reopened.Sequence, phase, u); err == nil {
			t.Fatal("committed step replayed")
		}
		if next.Sequence != j.Sequence+1 {
			t.Fatal("nonserial recovery step")
		}
		j = next
		trace = append(trace, j.Clone())
	}
	step("reviewing", AnchorRecoveryJournal{})
	review := ready.Replacement.Retirement.Namespace.Drain.Review
	step("reviewed", AnchorRecoveryJournal{Review: &review})
	step("draining", AnchorRecoveryJournal{})
	finalDrain := ready.Replacement.Retirement.Namespace.Drain
	for i, m := range finalDrain.Members {
		next, err := j.Drain.Advance(h, i, j.Drain.Sequence, "stop-dispatched", m.Before)
		must(err)
		step("draining", AnchorRecoveryJournal{Drain: &next})
		if ack {
			next, err = j.Drain.Advance(h, i, j.Drain.Sequence, "stop-signalled", nil)
			must(err)
			step("draining", AnchorRecoveryJournal{Drain: &next})
		}
		next, err = j.Drain.Advance(h, i, j.Drain.Sequence, "exited", m.After)
		must(err)
		step("draining", AnchorRecoveryJournal{Drain: &next})
	}
	step("namespace", AnchorRecoveryJournal{})
	finalNS := ready.Replacement.Retirement.Namespace
	phase := "observing"
	if !finalNS.Before.Runtime.Anchor.Running {
		phase = "start-dispatched"
	}
	nextNS, err := j.Namespace.Advance(h, j.Namespace.Sequence, phase, finalNS.Before)
	must(err)
	step("namespace", AnchorRecoveryJournal{Namespace: &nextNS})
	if phase == "start-dispatched" && ack {
		nextNS, err = j.Namespace.Advance(h, j.Namespace.Sequence, "start-acknowledged", nil)
		must(err)
		step("namespace", AnchorRecoveryJournal{Namespace: &nextNS})
	}
	nextNS, err = j.Namespace.Advance(h, j.Namespace.Sequence, "observed", finalNS.After)
	must(err)
	step("namespace", AnchorRecoveryJournal{Namespace: &nextNS})
	step("retiring", AnchorRecoveryJournal{})
	for i, m := range ready.Replacement.Retirement.Members {
		next, err := j.Retirement.Advance(h, i, j.Retirement.Sequence, "remove-dispatched", m.Before)
		must(err)
		step("retiring", AnchorRecoveryJournal{Retirement: &next})
		if ack {
			next, err = j.Retirement.Advance(h, i, j.Retirement.Sequence, "remove-acknowledged", nil)
			must(err)
			step("retiring", AnchorRecoveryJournal{Retirement: &next})
		}
		next, err = j.Retirement.Advance(h, i, j.Retirement.Sequence, "absent", m.After)
		must(err)
		step("retiring", AnchorRecoveryJournal{Retirement: &next})
	}
	var members []RecoveredMember
	for _, m := range ready.Replacement.Members {
		members = append(members, RecoveredMember{Creation: m.Creation.Clone(), Container: ResourceIdentity{ContractDigest: m.Container.ContractDigest}, Phase: "planned"})
	}
	nextR, err := NewAnchorReplacement(h, *j.Retirement, j.IDs.Replacement, members)
	must(err)
	step("replacing", AnchorRecoveryJournal{Replacement: &nextR})
	for i, m := range ready.Replacement.Members {
		for _, phase := range []string{"create-dispatched", "created", "start-dispatched", "start-acknowledged", "running"} {
			if phase == "start-acknowledged" && !ack {
				continue
			}
			result := AnchorReplacementResult{}
			switch phase {
			case "create-dispatched":
				result.Observation = m.Before
			case "created":
				result.ContainerID = m.Container.ID
			case "start-dispatched":
				result.Observation = m.Created
			case "running":
				result.Observation = m.After
			}
			next, err := j.Replacement.Advance(h, i, j.Replacement.Sequence, phase, result)
			must(err)
			step("replacing", AnchorRecoveryJournal{Replacement: &next})
		}
	}
	final := ready.Clone()
	final.Replacement = j.Replacement.Clone()
	step("complete", AnchorRecoveryJournal{Readiness: &final})
	if !reflect.DeepEqual(h, original) {
		t.Fatal("journal rewrote source history")
	}
	return trace
}
func TestAnchorRecoveryJournalExactSequence(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		for _, ack := range []bool{false, true} {
			t.Run(fmt.Sprintf("stopped-%t/ack-%t", stopped, ack), func(t *testing.T) {
				h, r := journalReadinessFixture(t, stopped)
				trace := replayRecoveryJournal(t, h, r, ack)
				last := trace[len(trace)-1]
				next, err := AppendAnchorRecoveryEpoch(h, *last.Readiness)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = NewAnchorRecoveryJournal(next, last.IDs); err == nil {
					t.Fatal("prior recovery IDs reused")
				}
				for _, j := range trace {
					body, err := json.Marshal(j)
					if err != nil {
						t.Fatal(err)
					}
					var changed AnchorRecoveryJournal
					if err = json.Unmarshal(body, &changed); err != nil {
						t.Fatal(err)
					}
					changed.Sequence++
					if changed.Validate(h) == nil {
						t.Fatal("changed journal sequence accepted", j.Phase)
					}
					changed = j.Clone()
					changed.IDs.Members[0].OperationID = changed.IDs.Drain
					if changed.Validate(h) == nil {
						t.Fatal("overlapping identities accepted")
					}
					changed = j.Clone()
					changed.Review = &r.Replacement.Retirement.Namespace.Drain.Review
					if j.Phase != "reviewed" && changed.Validate(h) == nil {
						t.Fatal("ambiguous phase payload accepted", j.Phase)
					}
					if j.Phase != "complete" {
						if _, err = j.Advance(h, j.Sequence, "complete", AnchorRecoveryJournal{Readiness: &r}); err == nil && !(j.Phase == "replacing" && j.Replacement.Running()) {
							t.Fatal("skipped recovery steps")
						}
					}
				}
				if last.Readiness.Replacement.Members[0].StartAcknowledged != ack || last.Readiness.Replacement.Retirement.Members[0].RemoveAcknowledged != ack || last.Readiness.Replacement.Retirement.Namespace.Drain.Members[0].SignalAcknowledged != ack {
					t.Fatal("uncertain dispatch became acknowledged")
				}
				// Mutating a returned nested receipt never changes the preceding checkpoint.
				prev := trace[len(trace)-2].Clone()
				last.Readiness.Replacement.Members[0].Creation.Storage.Path = "changed"
				if !reflect.DeepEqual(trace[len(trace)-2], prev) {
					t.Fatal("journal checkpoints alias")
				}
			})
		}
	}
}
func TestAnchorRecoveryJournalSecondBoundary(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	h, err := AppendAnchorRecoveryEpoch(h, r)
	if err != nil {
		t.Fatal(err)
	}
	h.Resumes = tail
	second := secondRecoveryFixture(t, h)
	trace := replayRecoveryJournal(t, h, second, false)
	final, err := AppendAnchorRecoveryEpoch(h, *trace[len(trace)-1].Readiness)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Epochs) != 2 || len(final.Epochs[1].Resumes) != len(tail) {
		t.Fatal("second journal lost earlier ordinary suffix")
	}
}
func TestAnchorRecoveryJournalRejectsRewrittenChild(t *testing.T) {
	h, r := journalReadinessFixture(t, false)
	trace := replayRecoveryJournal(t, h, r, true)
	checked := map[string]bool{}
	for i := 1; i < len(trace); i++ {
		old, next := trace[i-1], trace[i]
		if old.Phase != next.Phase || checked[old.Phase] {
			continue
		}
		candidate := AnchorRecoveryJournal{}
		switch old.Phase {
		case "draining":
			v := next.Drain.Clone()
			v.Review.After.Processes[0].StartedAt = v.Review.After.Processes[0].StartedAt.Add(1)
			candidate.Drain = &v
		case "namespace":
			v := next.Namespace.Clone()
			v.Drain.OperationID = "foreign-drain"
			candidate.Namespace = &v
		case "retiring":
			v := next.Retirement.Clone()
			v.Namespace.OperationID = "foreign-namespace"
			candidate.Retirement = &v
		case "replacing":
			v := next.Replacement.Clone()
			v.Retirement.OperationID = "foreign-retirement"
			candidate.Replacement = &v
		default:
			continue
		}
		if _, err := old.Advance(h, old.Sequence, old.Phase, candidate); err == nil {
			t.Fatal("rewritten prior child accepted", old.Phase)
		}
		checked[old.Phase] = true
	}
	if len(checked) != 4 {
		t.Fatal("missing phase corruption check")
	}
}
