//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// OpenResumedProjectMemberStopExit observes a retained Stop of the current
// replacement without retrying its signal or completing the original receipt.
func OpenResumedProjectMemberStopExit(ctx context.Context, s ProjectResumedMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return openResumedProjectMemberStopExit(ctx, s, defaults())
}
func openResumedProjectMemberStopExit(ctx context.Context, s ProjectResumedMemberStopSelection, d dependencies) (projectnetwork.MemberStopExitDriver, error) {
	retained := s.Stop
	if retained.Validate(s.Project.Plan) != nil || retained.Before == nil {
		return nil, errors.New("replacement Stop exit requires retained dispatch and process evidence")
	}
	s.Stop.Phase, s.Stop.Before, s.Stop.After = "planned", nil, nil
	raw, err := openResumedProjectMemberStop(ctx, s, d)
	if err != nil {
		return nil, err
	}
	return memberStopExitFromSource(s.Project.Plan, raw.(*projectMemberStopEngine), retained)
}

type ProjectResumedMemberRetirementSelection struct {
	Stop        ProjectResumedMemberStopSelection
	Exit        projectnetwork.MemberStopExitEvidence
	OperationID string
}

// Retirement still removes only an exactly observed exited container; its pier,
// peer processes and all earlier receipts remain retained.
func OpenResumedProjectMemberRetirement(ctx context.Context, s ProjectResumedMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return openResumedProjectMemberRetirement(ctx, s, defaults())
}
func openResumedProjectMemberRetirement(ctx context.Context, s ProjectResumedMemberRetirementSelection, d dependencies) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	initial, err := projectnetwork.NewMemberRetirement(s.Stop.Project.Plan, s.Exit, s.OperationID)
	if err != nil || !reflect.DeepEqual(s.Exit.Stop, s.Stop.Stop) {
		return projectnetwork.MemberRetirement{}, nil, errors.New("replacement retirement requires its exact retained clean exit")
	}
	raw, err := openResumedProjectMemberStopExit(ctx, s.Stop, d)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	frozen, err := projectnetwork.NewMemberRetirement(s.Stop.Project.Plan, s.Exit, s.OperationID)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	return initial, &projectMemberRetirementEngine{source: raw.(*projectMemberStopExitEngine).source, initial: frozen}, nil
}
