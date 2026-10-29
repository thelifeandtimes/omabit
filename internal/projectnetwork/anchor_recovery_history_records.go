package projectnetwork

import (
	"errors"
	"reflect"
	"slices"
	"strings"
)

// CompletedRecoveryRecord is the exact immutable source and completed receipt
// retained by a future owning journal. The store must additionally verify each
// receipt's actor, operation ownership and database references before using it.
type CompletedRecoveryRecord struct {
	Source    AnchorRecoveryHistory
	Readiness AnchorRecoveryReadiness
}

// AssembleRecoveryHistory partitions all completed ordinary replacements around
// recorded recovery boundaries. It selects by exact retained ancestry and causal
// bindings, never receipt time, SQL row order or the largest member generation.
// Pending recovery and ordinary effects must remain fenced by the caller.
func AssembleRecoveryHistory(root AnchorRecoveryHistory, resumes []MemberResume, records []CompletedRecoveryRecord) (AnchorRecoveryHistory, error) {
	fail := func() (AnchorRecoveryHistory, error) {
		return AnchorRecoveryHistory{}, errors.New("completed recovery records do not form one complete causal history")
	}
	if len(root.Epochs) != 0 || len(root.Resumes) != 0 || len(root.Current) != 0 {
		return fail()
	}
	if _, err := SelectAnchorRecoveryHistoryContext(root); err != nil {
		return fail()
	}
	root = root.Clone()
	remaining := map[string]MemberResume{}
	for _, r := range resumes {
		if r.Validate(root.Plan) != nil || r.Phase != "running" {
			return fail()
		}
		if _, ok := remaining[r.OperationID]; ok {
			return fail()
		}
		remaining[r.OperationID] = cloneMemberResume(r)
	}
	ordered := make([]CompletedRecoveryRecord, len(records))
	for i, r := range records {
		ordered[i] = CompletedRecoveryRecord{r.Source.Clone(), r.Readiness.Clone()}
	}
	slices.SortFunc(ordered, func(a, b CompletedRecoveryRecord) int {
		if len(a.Source.Epochs) != len(b.Source.Epochs) {
			return len(a.Source.Epochs) - len(b.Source.Epochs)
		}
		return strings.Compare(a.Readiness.Replacement.OperationID, b.Readiness.Replacement.OperationID)
	})
	h := root
	for i, r := range ordered {
		source := r.Source
		if len(source.Epochs) != i || !reflect.DeepEqual(source.Plan, root.Plan) || !reflect.DeepEqual(source.Launches, root.Launches) || !reflect.DeepEqual(source.Originals, root.Originals) || !reflect.DeepEqual(source.Epochs, h.Epochs) {
			return fail()
		}
		for _, s := range source.Resumes {
			actual, ok := remaining[s.OperationID]
			if !ok || !reflect.DeepEqual(actual, s) {
				return fail()
			}
			delete(remaining, s.OperationID)
		}
		if r.Readiness.Validate(source) != nil {
			return fail()
		}
		var err error
		h, err = AppendAnchorRecoveryEpoch(source, r.Readiness)
		if err != nil {
			return fail()
		}
	}
	h.Resumes = nil
	for _, r := range remaining {
		h.Resumes = append(h.Resumes, r)
	}
	slices.SortFunc(h.Resumes, func(a, b MemberResume) int { return strings.Compare(a.OperationID, b.OperationID) })
	if _, err := ResolveAnchorRecoveryHistory(h); err != nil {
		return fail()
	}
	return h.Clone(), nil
}
