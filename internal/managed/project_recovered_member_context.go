package managed

import (
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// No complete recovery means the existing ordinary lifecycle path remains exact.
// A completed recovery instead requires the full verified causal history, even
// when the requested operation belongs to an earlier, ordinary generation.
func completedProjectRecoveryHistory(q querier, project string) (*projectnetwork.AnchorRecoveryHistory, error) {
	jobs, err := projectAnchorRecoveryJobs(q)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.ProjectID == project && j.Journal.Phase == "complete" {
			h, _, _, err := projectRecoveryHistory(q, project, false)
			if err != nil {
				return nil, err
			}
			return &h, nil
		}
	}
	return nil, nil
}
func recoveryGenerationSelection(q querier, p storedProject, launches []projectnetwork.MemberLaunch, resumes []projectnetwork.MemberResume, binding projectnetwork.RuntimeBinding, current bool) (projectnetwork.MemberGenerationSet, error) {
	h, err := completedProjectRecoveryHistory(q, p.ID)
	if err != nil {
		return projectnetwork.MemberGenerationSet{}, err
	}
	if h != nil {
		if !current {
			prefix, err := projectnetwork.AnchorRecoveryHistoryAtBinding(*h, binding)
			if err != nil {
				return projectnetwork.MemberGenerationSet{}, err
			}
			h = &prefix
		}
		return projectnetwork.ResolveAnchorRecoveryHistory(*h)
	}
	if current {
		return projectnetwork.ResolveMemberGenerations(p.Plan, launches, resumes)
	}
	return projectnetwork.MemberGenerationsAtBinding(p.Plan, launches, resumes, binding)
}
func contextReadinessProofValid(src projectContextReadinessPrerequisites, r projectnetwork.CurrentMemberReadiness) error {
	if src.RecoveryHistory != nil {
		return r.ValidateRecovered(*src.RecoveryHistory)
	}
	return r.Validate(src.Project.Plan, src.Launches, src.Resumes, src.Original)
}
func recoveredContextSource(q querier, p storedProject, launches []projectnetwork.MemberLaunch, original projectnetwork.InitialMemberReadiness, binding *projectnetwork.RuntimeBinding) (*projectContextReadinessPrerequisites, error) {
	h, err := completedProjectRecoveryHistory(q, p.ID)
	if err != nil || h == nil {
		return nil, err
	}
	if binding != nil {
		prefix, err := projectnetwork.AnchorRecoveryHistoryAtBinding(*h, *binding)
		if err != nil {
			return nil, err
		}
		h = &prefix
	}
	src := projectContextReadinessPrerequisites{Project: p, Launches: launches, Resumes: h.Resumes, Original: original}
	if len(h.Epochs) == 0 {
		src.Context, err = projectnetwork.SelectMemberContext(p.Plan, launches, h.Resumes, original)
	} else {
		found := false
		for _, r := range h.Originals {
			if reflect.DeepEqual(r, original) {
				found = true
			}
		}
		if !found {
			return nil, errors.New("recovered member requires its retained original readiness reference")
		}
		src.Context, err = projectnetwork.SelectRecoveryMemberContext(*h, original.InstanceID)
		src.RecoveryHistory = h
	}
	return &src, err
}
func recoveryOrdinaryHistory(h projectnetwork.AnchorRecoveryHistory) []projectnetwork.MemberResume {
	var out []projectnetwork.MemberResume
	for _, e := range h.Epochs {
		out = append(out, e.Resumes...)
	}
	return append(out, h.Resumes...)
}
