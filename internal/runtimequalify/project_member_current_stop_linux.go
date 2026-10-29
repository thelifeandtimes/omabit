//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectCurrentMemberStopSelection struct {
	ReadinessSource ProjectCurrentReadinessSelection
	Readiness       projectnetwork.CurrentMemberReadiness
	Stop            projectnetwork.MemberStop
	StoppedPeers    []projectnetwork.MemberStop
}

// OpenCurrentProjectMemberStop binds the existing single-TERM driver to new
// current-context readiness. Later peer exits require exact retained Stop receipts;
// durable journal admission and public scheduling are not yet integrated.
func OpenCurrentProjectMemberStop(ctx context.Context, s ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return openCurrentProjectMemberStop(ctx, s, defaults())
}
func openCurrentProjectMemberStop(ctx context.Context, s ProjectCurrentMemberStopSelection, d dependencies) (projectnetwork.MemberStopDriver, error) {
	input := s.ReadinessSource
	expected, err := newSelectedCurrentMemberStop(input, s.Readiness, s.Stop.ReadinessOperationID, s.Stop.OperationID)
	if err != nil || !reflect.DeepEqual(expected, s.Stop) {
		return nil, errors.New("member Stop requires matching current-context readiness intent")
	}
	raw, err := openCurrentProjectReadiness(ctx, input, d)
	if err != nil {
		return nil, err
	}
	reader := raw.(*projectCurrentReadinessEngine)
	source := reader.source
	e := &projectMemberStopEngine{recovery: reader.recovery, source: source, initial: expected, exits: map[string]projectMemberExit{}}
	if input.RecoveryHistory != nil {
		e.exits, err = recoveredCurrentMemberPeerExits(*input.RecoveryHistory, expected, s.StoppedPeers)
	} else {
		current, er := projectnetwork.ResolveMemberGenerations(input.Project.Plan, input.Launches, input.Resumes)
		if er != nil {
			return nil, er
		}
		e.exits, err = currentMemberPeerExits(input.Project.Plan, input.Launches, input.Resumes, current, expected, s.StoppedPeers)
	}
	if err != nil {
		return nil, err
	}

	for i, m := range source.members {
		if m.contract.selection.InstanceID == expected.InstanceID {
			// Initial launch receipts have no process timestamp. The new readiness
			// bracket supplies it, so a subsequent same-container restart must refuse.
			source.members[i].processStartedAt = s.Readiness.After.StartedAt
			e.target = source.members[i]
		}
	}
	if e.target.id == "" {
		return nil, errors.New("current Stop target is missing")
	}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
