package projectnetwork

import (
	"errors"
	"reflect"
	"slices"
)

// ValidateHistorical checks a read-only receipt at its exact causal prefix. It
// validates the entire supplied completed history but grants no current authority.
func (r CurrentMemberReadiness) ValidateHistorical(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness) error {
	prefix, err := MemberResumesAtBinding(plan, launches, resumes, r.Binding)
	if err != nil {
		return err
	}
	return r.Validate(plan, launches, prefix, original)
}

// NewCurrentMemberStop binds Stop intent to new readiness at the latest complete
// history, including an original member whose peer was replaced. No older Stop,
// readiness or process evidence is rewritten. Persistence is a separate gate.
func NewCurrentMemberStop(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness, ready CurrentMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	if ready.Validate(plan, launches, resumes, original) != nil {
		return MemberStop{}, errors.New("member Stop requires fresh current-context readiness")
	}
	return newCurrentMemberStop(plan, ready, readinessOperation, operation)
}
func newCurrentMemberStop(plan Plan, ready CurrentMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	s := MemberStop{OperationID: operation, LaunchOperationID: ready.CreatorOperationID, ReadinessOperationID: readinessOperation, InstanceID: ready.InstanceID, Binding: ready.Binding, Storage: ready.Storage, Phase: "planned"}
	s.Binding.Members = slices.Clone(s.Binding.Members)
	if err := s.Validate(plan); err != nil {
		return MemberStop{}, err
	}
	return s, nil
}

// ValidateCurrentMemberStop preserves historical validity after later peer or
// target replacements. Any captured Stop process must match the readiness bracket;
// fresh selection is still mandatory before dispatch, never this historical read.
func ValidateCurrentMemberStop(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness, ready CurrentMemberReadiness, retained MemberStop) error {
	fail := func() error { return errors.New("current-context Stop lost its readiness or process binding") }
	if retained.Validate(plan) != nil || ready.ValidateHistorical(plan, launches, resumes, original) != nil {
		return fail()
	}
	prefix, err := MemberResumesAtBinding(plan, launches, resumes, ready.Binding)
	if err != nil {
		return fail()
	}
	expected, err := NewCurrentMemberStop(plan, launches, prefix, original, ready, retained.ReadinessOperationID, retained.OperationID)
	baseline := cloneMemberStop(retained)
	baseline.Phase = "planned"
	baseline.Before = nil
	baseline.After = nil
	if err != nil || !reflect.DeepEqual(expected, baseline) {
		return fail()
	}
	if retained.Before != nil && (!retained.Before.StartedAt.Equal(ready.After.StartedAt) || retained.Before.CapturedAt.Before(ready.After.CapturedAt) || retained.Before.Runtime.ObservedAt.Before(ready.After.Runtime.ObservedAt)) {
		return fail()
	}
	return nil
}
