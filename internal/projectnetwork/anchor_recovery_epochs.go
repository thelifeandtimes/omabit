package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"time"
)

// AnchorRecoveryEpoch is a completed causal boundary. Resumes and Current are
// the ordinary replacements and review prerequisites preceding this recovery,
// after the previous boundary. The containing history's Resumes/Current are the
// current suffix. No epoch rewrites an original creator or namespace receipt.
// These values remain private source models, not a new persistent format.
type AnchorRecoveryEpoch struct {
	Resumes   []MemberResume           `json:"resumes"`
	Current   []CurrentMemberReadiness `json:"current"`
	Readiness AnchorRecoveryReadiness  `json:"readiness"`
}

func recoveryEpochSource(h AnchorRecoveryHistory, index int) AnchorRecoveryHistory {
	e := h.Epochs[index]
	return (AnchorRecoveryHistory{Plan: h.Plan, Launches: h.Launches, Originals: h.Originals, Epochs: h.Epochs[:index], Resumes: e.Resumes, Current: e.Current}).Clone()
}
func AppendAnchorRecoveryEpoch(h AnchorRecoveryHistory, r AnchorRecoveryReadiness) (AnchorRecoveryHistory, error) {
	if r.Validate(h) != nil {
		return AnchorRecoveryHistory{}, errors.New("recovery boundary requires complete validated readiness")
	}
	out := h.Clone()
	out.Epochs = append(out.Epochs, AnchorRecoveryEpoch{Resumes: out.Resumes, Current: out.Current, Readiness: r.Clone()})
	out.Resumes = nil
	out.Current = nil
	return out, nil
}
func ResolveAnchorRecoveryHistory(h AnchorRecoveryHistory) (MemberGenerationSet, error) {
	if len(h.Epochs) == 0 {
		return ResolveMemberGenerations(h.Plan, h.Launches, h.Resumes)
	}
	i := len(h.Epochs) - 1
	return ResolveRecoveredMemberGenerations(recoveryEpochSource(h, i), h.Epochs[i].Readiness, h.Resumes)
}
func SelectAnchorRecoveryHistoryContext(h AnchorRecoveryHistory) (AnchorRecoveryContext, error) {
	if len(h.Epochs) == 0 {
		return SelectAnchorRecoveryContext(h.Plan, h.Launches, h.Resumes, h.Originals, h.Current)
	}
	fail := func() (AnchorRecoveryContext, error) {
		return AnchorRecoveryContext{}, errors.New("recovery history lost exact member ownership or prior readiness")
	}
	current, err := ResolveAnchorRecoveryHistory(h)
	if err != nil || len(h.Originals) != len(h.Plan.Members) || len(h.Current) > len(h.Plan.Members) {
		return fail()
	}
	i := len(h.Epochs) - 1
	source := recoveryEpochSource(h, i)
	boundary := h.Epochs[i].Readiness
	result := AnchorRecoveryContext{Binding: current.Binding}
	result.Binding.Members = slices.Clone(result.Binding.Members)
	used := map[string]bool{}
	for _, m := range current.Members {
		c, err := SelectRecoveredMemberContext(source, boundary, h.Resumes, m.InstanceID)
		if err != nil {
			return fail()
		}
		started, notBefore := c.KnownStartedAt, c.NotBefore
		for _, r := range h.Current {
			if r.InstanceID != m.InstanceID {
				continue
			}
			if used[r.InstanceID] || r.CreatorOperationID != m.OperationID || r.ContainerGeneration != m.Generation {
				return fail()
			}
			used[r.InstanceID] = true
			prefix, err := RecoveredResumesAtBinding(source, boundary, h.Resumes, r.Binding)
			if err != nil {
				return fail()
			}
			historical, err := SelectRecoveredMemberContext(source, boundary, prefix, m.InstanceID)
			if err != nil || r.validateContext(h.Plan, historical) != nil || !r.After.StartedAt.Equal(started) {
				return fail()
			}
			if r.CapturedAt.After(notBefore) {
				notBefore = r.CapturedAt
			}
		}
		result.Members = append(result.Members, AnchorRecoveryMember{Context: c, StartedAt: started, NotBefore: notBefore})
	}
	if len(used) != len(h.Current) {
		return fail()
	}
	return result, nil
}

func CollectAnchorRecoveryHistoryReview(ctx context.Context, h AnchorRecoveryHistory, d AnchorRecoveryObserver, now func() time.Time) (AnchorRecoveryReview, error) {
	h = h.Clone()
	c, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return AnchorRecoveryReview{}, err
	}
	return collectAnchorRecoveryReviewContext(ctx, h.Plan, c, d, now, h.validateReview)
}

func recoveryEpochOperationExists(h AnchorRecoveryHistory, id string) bool {
	for _, e := range h.Epochs {
		for _, s := range e.Resumes {
			for _, old := range recoveredLifecycleOperations(s) {
				if old == id {
					return true
				}
			}
		}
		r := e.Readiness.Replacement
		for _, old := range []string{r.OperationID, r.Retirement.OperationID, r.Retirement.Namespace.OperationID, r.Retirement.Namespace.Drain.OperationID} {
			if old == id {
				return true
			}
		}
		for _, m := range r.Members {
			if m.Creation.OperationID == id {
				return true
			}
		}
	}
	return false
}
func recoveryHistoryContainerIDs(h AnchorRecoveryHistory) map[string]bool {
	out := map[string]bool{}
	for _, l := range h.Launches {
		out[l.Container.ID] = true
	}
	for _, s := range h.Resumes {
		out[s.Container.ID] = true
	}
	for _, e := range h.Epochs {
		for _, s := range e.Resumes {
			out[s.Container.ID] = true
		}
		for _, m := range e.Readiness.Replacement.Members {
			out[m.Container.ID] = true
		}
	}
	return out
}

// AnchorRecoveryHistoryAtBinding validates the complete history before returning
// an exact causal prefix, including prefixes before earlier recovery boundaries.
// A historical prefix is evidence only; current admission resolves the full tail.
func AnchorRecoveryHistoryAtBinding(h AnchorRecoveryHistory, binding RuntimeBinding) (AnchorRecoveryHistory, error) {
	if _, err := ResolveAnchorRecoveryHistory(h); err != nil {
		return AnchorRecoveryHistory{}, err
	}
	if binding.Validate(h.Plan) != nil {
		return AnchorRecoveryHistory{}, errors.New("invalid historical recovery binding")
	}
	for i := 0; i <= len(h.Epochs); i++ {
		source := h.Clone()
		if i < len(h.Epochs) {
			source = recoveryEpochSource(h, i)
		}
		var prefix []MemberResume
		var err error
		if len(source.Epochs) == 0 {
			prefix, err = MemberResumesAtBinding(h.Plan, h.Launches, source.Resumes, binding)
		} else {
			j := len(source.Epochs) - 1
			prefix, err = RecoveredResumesAtBinding(recoveryEpochSource(source, j), source.Epochs[j].Readiness, source.Resumes, binding)
		}
		if err != nil {
			continue
		}
		source.Resumes = prefix
		source.Current = nil
		got, err := ResolveAnchorRecoveryHistory(source)
		if err == nil && reflect.DeepEqual(got.Binding, binding) {
			return source, nil
		}
	}
	return AnchorRecoveryHistory{}, errors.New("binding is not a prefix of complete recovery history")
}
