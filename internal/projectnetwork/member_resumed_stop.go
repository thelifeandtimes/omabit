package projectnetwork

import (
	"errors"
	"reflect"
	"slices"
)

// NewResumedMemberStop binds new Stop intent to the latest causally resolved
// member creation and its separate readiness receipt. Historical success does not
// grant live authority: ExecuteMemberStop still requires fresh inspection before
// dispatch. The caller must retain original uncertain Stop/exit/retirement receipts.
func NewResumedMemberStop(plan Plan, launches []MemberLaunch, resumes []MemberResume, ready ResumedMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	fail := func() (MemberStop, error) {
		return MemberStop{}, errors.New("member Stop requires current replacement history and matching readiness")
	}
	current, err := ResolveMemberGenerations(plan, launches, resumes)
	if err != nil || !reflect.DeepEqual(current.Binding, ready.Binding) {
		return fail()
	}
	return resumedMemberStop(plan, current, resumes, ready, readinessOperation, operation)
}

// ValidateResumedMemberStop checks an old receipt at its historical binding even
// when later completed replacements exist. It does not authorize another Stop.
func ValidateResumedMemberStop(plan Plan, launches []MemberLaunch, resumes []MemberResume, ready ResumedMemberReadiness, retained MemberStop) error {
	if retained.Validate(plan) != nil {
		return errors.New("invalid retained replacement Stop")
	}
	prefix, err := MemberGenerationsAtBinding(plan, launches, resumes, ready.Binding)
	if err != nil {
		return err
	}
	expected, err := resumedMemberStop(plan, prefix, resumes, ready, retained.ReadinessOperationID, retained.OperationID)
	if err != nil {
		return err
	}
	baseline := retained
	baseline.Phase = "planned"
	baseline.Before = nil
	baseline.After = nil
	if !reflect.DeepEqual(expected, baseline) {
		return errors.New("retained replacement Stop lost its readiness binding")
	}
	return nil
}
func resumedMemberStop(plan Plan, current MemberGenerationSet, resumes []MemberResume, ready ResumedMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	fail := func() (MemberStop, error) {
		return MemberStop{}, errors.New("member Stop requires matching replacement history and readiness")
	}
	var generation MemberGeneration
	for _, m := range current.Members {
		if m.InstanceID == ready.InstanceID {
			generation = m
		}
	}
	if generation.Mode != "resume" || generation.OperationID != ready.ResumeOperationID || generation.Generation != ready.ContainerGeneration || generation.Storage != ready.Storage {
		return fail()
	}
	found := false
	for _, s := range resumes {
		if s.OperationID == generation.OperationID {
			if ready.Validate(plan, s) != nil {
				return fail()
			}
			found = true
		}
	}
	if !found {
		return fail()
	}
	result := MemberStop{OperationID: operation, LaunchOperationID: generation.OperationID, ReadinessOperationID: readinessOperation, InstanceID: generation.InstanceID, Binding: current.Binding, Storage: ready.Storage, Phase: "planned"}
	result.Binding.Members = slices.Clone(result.Binding.Members)
	if result.Validate(plan) != nil {
		return fail()
	}
	return result, nil
}
