//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"reflect"
)

// A new current-context Stop retains its own process evidence through exit and
// retirement. These adapters never retry a Stop or rewrite its original receipt.
func OpenCurrentProjectMemberStopExit(ctx context.Context, s ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return openCurrentProjectMemberStopExit(ctx, s, defaults())
}
func openCurrentProjectMemberStopExit(ctx context.Context, s ProjectCurrentMemberStopSelection, d dependencies) (projectnetwork.MemberStopExitDriver, error) {
	retained := s.Stop
	input := s.ReadinessSource
	if retained.Before == nil || validateSelectedCurrentMemberStop(input, s.Readiness, retained) != nil {
		return nil, errors.New("current Stop exit requires retained matching dispatch and process evidence")
	}
	s.Stop.Phase, s.Stop.Before, s.Stop.After = "planned", nil, nil
	raw, err := openCurrentProjectMemberStop(ctx, s, d)
	if err != nil {
		return nil, err
	}
	return memberStopExitFromSource(input.Project.Plan, raw.(*projectMemberStopEngine), retained)
}

type ProjectCurrentMemberRetirementSelection struct {
	Stop        ProjectCurrentMemberStopSelection
	Exit        projectnetwork.MemberStopExitEvidence
	OperationID string
}

func OpenCurrentProjectMemberRetirement(ctx context.Context, s ProjectCurrentMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return openCurrentProjectMemberRetirement(ctx, s, defaults())
}
func openCurrentProjectMemberRetirement(ctx context.Context, s ProjectCurrentMemberRetirementSelection, d dependencies) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	plan := s.Stop.ReadinessSource.Project.Plan
	initial, err := projectnetwork.NewMemberRetirement(plan, s.Exit, s.OperationID)
	if err != nil || !reflect.DeepEqual(s.Exit.Stop, s.Stop.Stop) {
		return projectnetwork.MemberRetirement{}, nil, errors.New("current retirement requires its exact retained clean exit")
	}
	raw, err := openCurrentProjectMemberStopExit(ctx, s.Stop, d)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	frozen, err := projectnetwork.NewMemberRetirement(plan, s.Exit, s.OperationID)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	return initial, &projectMemberRetirementEngine{source: raw.(*projectMemberStopExitEngine).source, initial: frozen}, nil
}
