package projectnetwork

import (
	"reflect"
	"slices"
	"testing"
)

func TestAssembleRecoveryHistoryKeepsEveryOrdinaryGeneration(t *testing.T) {
	firstSource, first, between := recoveredHistoryFixture(t)
	secondSource, err := AppendAnchorRecoveryEpoch(firstSource, first)
	if err != nil {
		t.Fatal(err)
	}
	secondSource.Resumes = between
	second := secondRecoveryFixture(t, secondSource)
	complete, err := AppendAnchorRecoveryEpoch(secondSource, second)
	if err != nil {
		t.Fatal(err)
	}
	instance := second.Members[0].InstanceID
	c, err := SelectRecoveryMemberContext(complete, instance)
	if err != nil {
		t.Fatal(err)
	}
	last := ordinaryResumeAfterRecovery(t, complete.Plan, c, second.After.Runtime, "records-final", 901)
	complete.Resumes = []MemberResume{last}
	root := firstSource.Clone()
	root.Resumes = nil
	root.Current = nil
	all := append(slices.Clone(firstSource.Resumes), between...)
	all = append(all, last)
	records := []CompletedRecoveryRecord{{firstSource, first}, {secondSource, second}}
	for _, mode := range []string{"pass", "reverse", "missing-first", "missing-middle-resume", "changed-middle-resume", "duplicate-record", "duplicate-resume", "altered-earlier-receipt", "branched-record", "unknown-final-generation"} {
		t.Run(mode, func(t *testing.T) {
			input := slices.Clone(all)
			selected := make([]CompletedRecoveryRecord, len(records))
			for i, r := range records {
				selected[i] = CompletedRecoveryRecord{r.Source.Clone(), r.Readiness.Clone()}
			}
			switch mode {
			case "reverse":
				slices.Reverse(input)
				slices.Reverse(selected)
			case "missing-first":
				selected = selected[1:]
			case "missing-middle-resume":
				input = slices.DeleteFunc(input, func(r MemberResume) bool { return r.OperationID == between[0].OperationID })
			case "changed-middle-resume":
				for i, r := range input {
					if r.OperationID == between[0].OperationID {
						input[i] = cloneMemberResume(r)
						input[i].Retirement.Exit.Stop.ReadinessOperationID = "changed"
					}
				}
			case "duplicate-record":
				selected = append(selected, selected[0])
			case "duplicate-resume":
				input = append(input, input[0])
			case "altered-earlier-receipt":
				selected[1].Source.Epochs[0].Readiness.Members[0].Ship = "~bus"
			case "branched-record":
				selected[1].Source.Epochs = nil
			case "unknown-final-generation":
				input[len(input)-1] = cloneMemberResume(last)
				input[len(input)-1].PreviousGeneration++
			}
			got, err := AssembleRecoveryHistory(root, input, selected)
			if mode == "pass" || mode == "reverse" {
				if err != nil || !reflect.DeepEqual(got, complete) {
					t.Fatal("complete record history changed", err)
				}
				got.Epochs[0].Readiness.Replacement.Members[0].Creation.Storage.Path = "caller-alias"
				if !reflect.DeepEqual(selected[0].Source.Plan, root.Plan) || records[0].Readiness.Replacement.Members[0].Creation.Storage.Path == "caller-alias" {
					t.Fatal("history aliases records")
				}
			} else if err == nil {
				t.Fatal("incomplete/altered records accepted")
			}
		})
	}
}
