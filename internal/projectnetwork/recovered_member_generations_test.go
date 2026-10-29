package projectnetwork

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func ordinaryResumeAfterRecovery(t *testing.T, p Plan, c MemberContext, o Observation, label string, id int) MemberResume {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	now := c.NotBefore.Add(time.Second)
	old := MemberStop{OperationID: "stop-" + label, ReadinessOperationID: "ready-" + label, LaunchOperationID: c.Member.OperationID, InstanceID: c.Member.InstanceID, Binding: c.Binding, Storage: c.Storage, Phase: "planned"}
	before := MemberStopObservation{Runtime: cloneReadinessObservation(o), Storage: c.Storage, StartedAt: c.KnownStartedAt, CapturedAt: now}
	before.Runtime.ObservedAt = now
	var err error
	old, err = old.Advance(p, "planned", "stop-dispatched", &before)
	must(err)
	exit := cloneMemberStopObservation(before)
	exit.CapturedAt = now.Add(time.Second)
	exit.Runtime.ObservedAt = exit.CapturedAt
	exit.FinishedAt = exit.CapturedAt
	for i := range exit.Runtime.Members {
		if exit.Runtime.Members[i].InstanceID == c.Member.InstanceID {
			exit.Runtime.Members[i].Running = false
			exit.Runtime.Members[i].Namespace = NamespaceIdentity{}
		}
	}
	retired, err := NewMemberRetirement(p, MemberStopExitEvidence{OperationID: "exit-" + label, Stop: old, Observation: exit}, "retire-"+label)
	must(err)
	retired, err = retired.Advance(p, "planned", "remove-dispatched", &exit, nil)
	must(err)
	retired, err = retired.Advance(p, "remove-dispatched", "removed", nil, nil)
	must(err)
	absence := MemberAbsenceObservation{Runtime: cloneReadinessObservation(exit.Runtime), Storage: c.Storage, CapturedAt: exit.CapturedAt.Add(time.Second)}
	absence.Runtime.ObservedAt = absence.CapturedAt
	absence.Runtime.Members = slices.DeleteFunc(absence.Runtime.Members, func(m MemberObservation) bool { return m.InstanceID == c.Member.InstanceID })
	retired, err = retired.Advance(p, "removed", "absent", nil, &absence)
	must(err)
	s, err := NewMemberResume(p, retired, c.Member.Generation, "resume-"+label, strings.Repeat("c", 64))
	must(err)
	absence.CapturedAt = absence.CapturedAt.Add(time.Second)
	absence.Runtime.ObservedAt = absence.CapturedAt
	s, err = s.Advance(p, "planned", "create-dispatched", MemberResumeResult{Before: &absence})
	must(err)
	s, err = s.Advance(p, "create-dispatched", "created", MemberResumeResult{ContainerID: fmt.Sprintf("%064x", id)})
	must(err)
	created := MemberResumeObservation{Runtime: cloneReadinessObservation(absence.Runtime), Storage: c.Storage, CapturedAt: absence.CapturedAt.Add(time.Second)}
	created.Runtime.ObservedAt = created.CapturedAt
	created.Runtime.Members = append(created.Runtime.Members, MemberObservation{InstanceID: c.Member.InstanceID, ResourceObservation: ResourceObservation{Resource: s.Container, Present: true}, AnchorID: c.Binding.Anchor.ID})
	slices.SortFunc(created.Runtime.Members, func(a, b MemberObservation) int { return strings.Compare(a.InstanceID, b.InstanceID) })
	s, err = s.Advance(p, "created", "start-dispatched", MemberResumeResult{Observation: &created})
	must(err)
	s, err = s.Advance(p, "start-dispatched", "started", MemberResumeResult{})
	must(err)
	running := cloneResumeObservation(created)
	running.CapturedAt = created.CapturedAt.Add(time.Second)
	running.Runtime.ObservedAt = running.CapturedAt
	running.StartedAt = running.CapturedAt
	for i := range running.Runtime.Members {
		if running.Runtime.Members[i].InstanceID == c.Member.InstanceID {
			running.Runtime.Members[i].Running = true
			running.Runtime.Members[i].Namespace = c.Binding.Namespace
		}
	}
	s, err = s.Advance(p, "started", "running", MemberResumeResult{Observation: &running})
	must(err)
	return s
}
func recoveredHistoryFixture(t *testing.T) (AnchorRecoveryHistory, AnchorRecoveryReadiness, []MemberResume) {
	t.Helper()
	h, s, now := completedReplacementForReadiness(t, false)
	f := &recoveryReadinessFixture{t: t, h: h, s: s, now: now}
	r, err := CollectAnchorRecoveryReadiness(context.Background(), h, s, f, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	var tail []MemberResume
	o := r.After.Runtime
	for i, index := range []int{0, 1, 0} {
		c, err := SelectRecoveredMemberContext(h, r, tail, s.Members[index].Creation.InstanceID)
		if err != nil {
			t.Fatal(err)
		}
		next := ordinaryResumeAfterRecovery(t, h.Plan, c, o, fmt.Sprintf("after-recovery-%d", i), 200+i)
		tail = append(tail, next)
		o = next.After.Runtime
	}
	return h, r, tail
}
func TestRecoveredMemberGenerationsRetainCausalPrefixes(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	original := r.Clone()
	prior := h.Clone()
	for _, n := range []int{0, 1, 2, 3} {
		history := slices.Clone(tail[:n])
		slices.Reverse(history)
		got, err := ResolveRecoveredMemberGenerations(h, r, history)
		if err != nil {
			t.Fatal(err)
		}
		want := r.Binding
		if n > 0 {
			want, err = ResumedRuntimeBinding(h.Plan, tail[n-1])
			if err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(got.Binding, want) || got.Binding.Generation != r.Binding.Generation || got.Binding.Namespace != r.Binding.Namespace {
			t.Fatal("lost successor or causal membership")
		}
		for i, m := range got.Members {
			c, err := SelectRecoveredMemberContext(h, r, history, m.InstanceID)
			if err != nil {
				t.Fatal(err)
			}
			expected := r.Replacement.Members[i].Creation.Generation
			for _, s := range tail[:n] {
				if s.Retirement.Exit.Stop.InstanceID == m.InstanceID {
					expected++
				}
			}
			if c.Member.Generation != expected || c.Storage != r.Replacement.Members[i].Creation.Storage || c.KnownStartedAt.IsZero() {
				t.Fatal("wrong recovered ownership")
			}
			if n > 0 && c.NotBefore != tail[n-1].After.CapturedAt {
				t.Fatal("peer update omitted from time floor")
			}
		}
		prefix, err := RecoveredResumesAtBinding(h, r, []MemberResume{tail[2], tail[0], tail[1]}, want)
		if err != nil || len(prefix) != n {
			t.Fatal("historical prefix lost", err)
		}
		if n > 0 {
			prefix[0].Retirement.Exit.Stop.Storage.Path = "changed"
		}
	}
	if !reflect.DeepEqual(h, prior) || !reflect.DeepEqual(r, original) {
		t.Fatal("rewrote recovery or initial history")
	}
}
func TestRecoveredMemberGenerationsRejectAmbiguityAndReuse(t *testing.T) {
	h, r, tail := recoveredHistoryFixture(t)
	for _, mode := range []string{"unfinished", "duplicate", "missing-predecessor", "branch", "wrong-creator", "wrong-generation", "changed-process", "early-stop", "operation-reuse", "retired-resource-reuse"} {
		t.Run(mode, func(t *testing.T) {
			values := []MemberResume{cloneMemberResume(tail[0]), cloneMemberResume(tail[1]), cloneMemberResume(tail[2])}
			switch mode {
			case "unfinished":
				values[2].Phase = "started"
				values[2].After = nil
			case "duplicate":
				values = append(values, cloneMemberResume(tail[0]))
			case "missing-predecessor":
				values = values[1:]
			case "branch":
				other := cloneMemberResume(tail[0])
				other.OperationID = "branch-resume"
				other.Retirement.OperationID = "branch-retire"
				values = append(values, other)
			case "wrong-creator":
				values[0].Retirement.Exit.Stop.LaunchOperationID = "other-creator"
			case "wrong-generation":
				values[0].PreviousGeneration++
			case "changed-process":
				v := &values[0]
				started := v.Retirement.Exit.Stop.Before.StartedAt.Add(time.Nanosecond)
				v.Retirement.Exit.Stop.Before.StartedAt = started
				v.Retirement.Exit.Observation.StartedAt = started
				v.Retirement.Before.StartedAt = started
			case "early-stop":
				v := values[0].Retirement.Exit.Stop.Before
				v.CapturedAt = r.CapturedAt.Add(-time.Nanosecond)
				v.Runtime.ObservedAt = v.CapturedAt
			case "operation-reuse":
				values[0].OperationID = r.Replacement.Retirement.Namespace.OperationID
			case "retired-resource-reuse":
				v := &values[0]
				v.Container.ID = h.Launches[0].Container.ID
				for _, o := range []*MemberResumeObservation{v.Created, v.After} {
					for i := range o.Runtime.Members {
						if o.Runtime.Members[i].InstanceID == v.Retirement.Exit.Stop.InstanceID {
							o.Runtime.Members[i].Resource = v.Container
						}
					}
				}
			}
			for _, v := range values {
				if err := v.Validate(h.Plan); err != nil {
					t.Fatal("fixture failed before cross-history boundary", err)
				}
			}
			if _, err := ResolveRecoveredMemberGenerations(h, r, values); err == nil {
				t.Fatal("invalid history accepted")
			}
			if _, err := RecoveredResumesAtBinding(h, r, values, r.Binding); err == nil {
				t.Fatal("early prefix hid invalid suffix")
			}
		})
	}
	foreign := r.Binding
	foreign.Members = slices.Clone(foreign.Members)
	foreign.Namespace = r.Replacement.Retirement.Namespace.Drain.Review.Context.Binding.Namespace
	if _, err := RecoveredResumesAtBinding(h, r, tail, foreign); err == nil {
		t.Fatal("pre-recovery binding accepted in successor tail")
	}
}
