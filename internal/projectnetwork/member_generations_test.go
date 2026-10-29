package projectnetwork

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func nextGenerationFixture(t *testing.T, p Plan, previous MemberResume) MemberResume {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	b, err := ResumedRuntimeBinding(p, previous)
	must(err)
	now := previous.After.CapturedAt.Add(time.Second)
	old := MemberStop{OperationID: "stop-next", LaunchOperationID: previous.OperationID, ReadinessOperationID: "ready-next", InstanceID: previous.Retirement.Exit.Stop.InstanceID, Binding: b, Storage: previous.After.Storage, Phase: "planned"}
	before := MemberStopObservation{Runtime: cloneReadinessObservation(previous.After.Runtime), Storage: old.Storage, StartedAt: previous.After.StartedAt, CapturedAt: now}
	before.Runtime.ObservedAt = now
	old, err = old.Advance(p, "planned", "stop-dispatched", &before)
	must(err)
	exit := cloneMemberStopObservation(before)
	exit.CapturedAt = now.Add(time.Second)
	exit.Runtime.ObservedAt = exit.CapturedAt
	exit.FinishedAt = exit.CapturedAt
	for i := range exit.Runtime.Members {
		if exit.Runtime.Members[i].InstanceID == old.InstanceID {
			exit.Runtime.Members[i].Running = false
			exit.Runtime.Members[i].Namespace = NamespaceIdentity{}
		}
	}
	r, err := NewMemberRetirement(p, MemberStopExitEvidence{OperationID: "exit-next", Stop: old, Observation: exit}, "retire-next")
	must(err)
	r, err = r.Advance(p, "planned", "remove-dispatched", &exit, nil)
	must(err)
	r, err = r.Advance(p, "remove-dispatched", "removed", nil, nil)
	must(err)
	absence := MemberAbsenceObservation{Runtime: cloneReadinessObservation(exit.Runtime), Storage: old.Storage, CapturedAt: exit.CapturedAt.Add(time.Second)}
	absence.Runtime.Members = slices.DeleteFunc(absence.Runtime.Members, func(m MemberObservation) bool { return m.InstanceID == old.InstanceID })
	absence.Runtime.ObservedAt = absence.CapturedAt
	r, err = r.Advance(p, "removed", "absent", nil, &absence)
	must(err)
	s, err := NewMemberResume(p, r, previous.PreviousGeneration+1, "resume-next", strings.Repeat("6", 64))
	must(err)
	absence.CapturedAt = absence.CapturedAt.Add(time.Second)
	absence.Runtime.ObservedAt = absence.CapturedAt
	s, err = s.Advance(p, "planned", "create-dispatched", MemberResumeResult{Before: &absence})
	must(err)
	s, err = s.Advance(p, "create-dispatched", "created", MemberResumeResult{ContainerID: strings.Repeat("7", 64)})
	must(err)
	created := MemberResumeObservation{Runtime: cloneReadinessObservation(absence.Runtime), Storage: old.Storage, CapturedAt: absence.CapturedAt.Add(time.Second)}
	created.Runtime.ObservedAt = created.CapturedAt
	target := MemberObservation{InstanceID: old.InstanceID, ResourceObservation: ResourceObservation{Resource: s.Container, Present: true}, AnchorID: b.Anchor.ID}
	created.Runtime.Members = append(created.Runtime.Members, target)
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
		if running.Runtime.Members[i].InstanceID == old.InstanceID {
			running.Runtime.Members[i].Running = true
			running.Runtime.Members[i].Namespace = b.Namespace
		}
	}
	s, err = s.Advance(p, "started", "running", MemberResumeResult{Observation: &running})
	must(err)
	return s
}
func TestMemberGenerationSelectionUsesCausalHistory(t *testing.T) {
	p, launches, _, _ := initialReadinessFixture(t)
	_, first := resumedReadinessFixture(t)
	second := nextGenerationFixture(t, p, first)
	baseline, err := InitialRuntimeBinding(p, launches)
	if err != nil {
		t.Fatal(err)
	}
	for _, history := range [][]MemberResume{nil, {first}, {first, second}, {second, first}} {
		original := make([]MemberResume, len(history))
		for i, s := range history {
			original[i] = cloneMemberResume(s)
		}
		result, err := ResolveMemberGenerations(p, launches, history)
		if err != nil {
			t.Fatal(err)
		}
		if result.Binding.Generation != baseline.Generation || result.Binding.Namespace != baseline.Namespace || result.Members[0].Generation != 1 || result.Members[1].Generation != uint64(1+len(history)) {
			t.Fatal("wrong generation selection", result)
		}
		if len(history) > 0 && result.Members[1].Mode != "resume" {
			t.Fatal("lost resume mode")
		}
		result.Binding.Members[0].InstanceID = "caller-edit"
		for i := range history {
			if !reflect.DeepEqual(history[i], original[i]) {
				t.Fatal("mutated immutable history")
			}
		}
		if len(history) == 2 && result.Members[1].Container != second.Container {
			t.Fatal("selected by input order")
		}
	}
}
func TestMemberGenerationSelectionRefusesAmbiguousHistory(t *testing.T) {
	for _, mode := range []string{"partial-launch", "unfinished", "duplicate", "retirement-reuse", "resource-reuse", "operation-reuse", "wrong-previous", "wrong-creator", "missing-predecessor", "branch"} {
		t.Run(mode, func(t *testing.T) {
			p, launches, _, _ := initialReadinessFixture(t)
			_, first := resumedReadinessFixture(t)
			second := nextGenerationFixture(t, p, first)
			history := []MemberResume{first, second}
			switch mode {
			case "partial-launch":
				launches = launches[:1]
			case "unfinished":
				second.Phase = "started"
				second.After = nil
				history[1] = second
			case "duplicate":
				history = append(history, first)
			case "retirement-reuse":
				history[1].Retirement.OperationID = first.Retirement.OperationID
			case "resource-reuse":
				id := first.Retirement.Exit.Stop.Binding.Members[1].Container.ID
				history[1].Container.ID = id
				for _, o := range []*MemberResumeObservation{history[1].Created, history[1].After} {
					for i := range o.Runtime.Members {
						if o.Runtime.Members[i].InstanceID == "zod" {
							o.Runtime.Members[i].Resource.ID = id
						}
					}
				}
			case "operation-reuse":
				history[1].OperationID = launches[0].Storage.OperationID
			case "wrong-previous":
				history[1].PreviousGeneration++
			case "wrong-creator":
				history[1].Retirement.Exit.Stop.LaunchOperationID = "wrong-creator"
			case "missing-predecessor":
				history = history[1:]
			case "branch":
				other := cloneMemberResume(first)
				other.OperationID = "branched-resume"
				other.Retirement.OperationID = "branched-retirement"
				history = []MemberResume{first, other}
			}
			if mode != "unfinished" {
				for _, receipt := range history {
					if err := receipt.Validate(p); err != nil {
						t.Fatal("fixture must reach cross-history validation", mode, err)
					}
				}
			}
			if r, err := ResolveMemberGenerations(p, launches, history); err == nil || !reflect.DeepEqual(r, MemberGenerationSet{}) {
				t.Fatal("ambiguous history selected", r, err)
			}
		})
	}
}

func TestMemberGenerationHistoricalPrefixPreservesOldBindings(t *testing.T) {
	p, launches, _, _ := initialReadinessFixture(t)
	_, first := resumedReadinessFixture(t)
	second := nextGenerationFixture(t, p, first)
	history := []MemberResume{second, first}
	initial, err := InitialRuntimeBinding(p, launches)
	if err != nil {
		t.Fatal(err)
	}
	middle, err := ResumedRuntimeBinding(p, first)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := ResumedRuntimeBinding(p, second)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range []RuntimeBinding{initial, middle, latest} {
		got, err := MemberGenerationsAtBinding(p, launches, history, b)
		if err != nil || got.Members[1].Generation != uint64(i+1) || !reflect.DeepEqual(got.Binding, b) {
			t.Fatal("lost historical prefix", i, got, err)
		}
		got.Binding.Members[0].InstanceID = "caller-edit"
		got.Members[0].InstanceID = "caller-edit"
		if first.Validate(p) != nil || second.Validate(p) != nil || b.Validate(p) != nil {
			t.Fatal("prefix aliases history")
		}
	}
	foreign := latest
	foreign.Members = slices.Clone(latest.Members)
	foreign.Members[1].Container.ContractDigest = strings.Repeat("5", 64)
	if _, err = MemberGenerationsAtBinding(p, launches, history, foreign); err == nil {
		t.Fatal("unrecorded historical binding accepted")
	}
	// A matching early prefix must not hide an invalid later branch.
	broken := cloneMemberResume(second)
	broken.PreviousGeneration++
	if _, err = MemberGenerationsAtBinding(p, launches, []MemberResume{first, broken}, initial); err == nil {
		t.Fatal("bad suffix hidden by old binding")
	}
}

func TestMemberResumeHistoryPrefixIsOrderedAndIndependent(t *testing.T) {
	p, launches, _, _ := initialReadinessFixture(t)
	_, first := resumedReadinessFixture(t)
	second := nextGenerationFixture(t, p, first)
	initial, err := InitialRuntimeBinding(p, launches)
	if err != nil {
		t.Fatal(err)
	}
	middle, err := ResumedRuntimeBinding(p, first)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := ResumedRuntimeBinding(p, second)
	if err != nil {
		t.Fatal(err)
	}
	for count, binding := range []RuntimeBinding{initial, middle, latest} {
		got, err := MemberResumesAtBinding(p, launches, []MemberResume{second, first}, binding)
		if err != nil || len(got) != count {
			t.Fatal("prefix not found", count, err)
		}
		if count > 0 && !reflect.DeepEqual(got[0], first) {
			t.Fatal("input order used instead of causal order")
		}
		if count > 1 && !reflect.DeepEqual(got[1], second) {
			t.Fatal("lost later prefix")
		}
		if count > 0 {
			got[0].After.Runtime.Members[0].InstanceID = "caller-edit"
			if first.Validate(p) != nil {
				t.Fatal("prefix aliases original history")
			}
		}
	}
	broken := cloneMemberResume(second)
	broken.PreviousGeneration++
	if _, err = MemberResumesAtBinding(p, launches, []MemberResume{first, broken}, initial); err == nil {
		t.Fatal("early prefix hid invalid later history")
	}
}
