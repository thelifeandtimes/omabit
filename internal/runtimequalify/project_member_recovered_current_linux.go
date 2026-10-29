//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func selectedCurrentRecoveryHistory(s ProjectCurrentReadinessSelection) (projectnetwork.AnchorRecoveryHistory, error) {
	if s.RecoveryHistory == nil {
		return projectnetwork.AnchorRecoveryHistory{}, errors.New("no explicit recovery history")
	}
	h := s.RecoveryHistory.Clone()
	if len(h.Epochs) == 0 || !reflect.DeepEqual(h.Plan, s.Project.Plan) || !reflect.DeepEqual(h.Launches, s.Launches) || !reflect.DeepEqual(h.Resumes, s.Resumes) {
		return h, errors.New("current recovery selection disagrees with retained source")
	}
	found := false
	for _, r := range h.Originals {
		if reflect.DeepEqual(r, s.Original) {
			found = true
		}
	}
	if !found {
		return h, errors.New("current recovery lost original initialized-pier evidence")
	}
	return h, nil
}
func openRecoveredCurrentProjectReadiness(ctx context.Context, s ProjectCurrentReadinessSelection, d dependencies) (projectnetwork.CurrentReadinessDriver, error) {
	h, err := selectedCurrentRecoveryHistory(s)
	if err != nil {
		return nil, err
	}
	c, err := projectnetwork.SelectRecoveryMemberContext(h, s.Original.InstanceID)
	if err != nil {
		return nil, err
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, ProjectAnchorRecoveryReviewSelection{Project: s.Project, Epochs: h.Epochs, Launches: h.Launches, Resumes: h.Resumes, Originals: h.Originals, Current: h.Current}, d)
	if err != nil {
		return nil, err
	}
	review := raw.(*projectAnchorRecoveryReviewEngine)
	e := &projectCurrentReadinessEngine{recovery: review, source: review.source, expected: c}
	for _, m := range h.Plan.Members {
		if m.InstanceID == c.Member.InstanceID {
			e.ship = m.Ship
			e.endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func newSelectedCurrentMemberStop(s ProjectCurrentReadinessSelection, ready projectnetwork.CurrentMemberReadiness, readyID, operation string) (projectnetwork.MemberStop, error) {
	if s.RecoveryHistory == nil {
		return projectnetwork.NewCurrentMemberStop(s.Project.Plan, s.Launches, s.Resumes, s.Original, ready, readyID, operation)
	}
	h, err := selectedCurrentRecoveryHistory(s)
	if err != nil {
		return projectnetwork.MemberStop{}, err
	}
	if ready.InstanceID != s.Original.InstanceID {
		return projectnetwork.MemberStop{}, errors.New("recovered Stop target changed")
	}
	return projectnetwork.NewRecoveredCurrentMemberStop(h, ready, readyID, operation)
}
func validateSelectedCurrentMemberStop(s ProjectCurrentReadinessSelection, ready projectnetwork.CurrentMemberReadiness, stop projectnetwork.MemberStop) error {
	if s.RecoveryHistory == nil {
		return projectnetwork.ValidateCurrentMemberStop(s.Project.Plan, s.Launches, s.Resumes, s.Original, ready, stop)
	}
	h, err := selectedCurrentRecoveryHistory(s)
	if err != nil {
		return err
	}
	if ready.InstanceID != s.Original.InstanceID {
		return errors.New("recovered Stop target changed")
	}
	return projectnetwork.ValidateRecoveredCurrentMemberStop(h, ready, stop)
}
func recoveredCurrentMemberPeerExits(h projectnetwork.AnchorRecoveryHistory, expected projectnetwork.MemberStop, peers []projectnetwork.MemberStop) (map[string]projectMemberExit, error) {
	exits := map[string]projectMemberExit{}
	current, err := projectnetwork.ResolveAnchorRecoveryHistory(h)
	if err != nil {
		return nil, err
	}
	for _, peer := range peers {
		if peer.Validate(h.Plan) != nil || peer.Phase != "stopped" || peer.InstanceID == expected.InstanceID || peer.OperationID == expected.OperationID {
			return nil, errors.New("invalid recovered peer Stop")
		}
		if _, ok := exits[peer.InstanceID]; ok {
			return nil, errors.New("duplicate recovered peer Stop")
		}
		prefix, err := projectnetwork.AnchorRecoveryHistoryAtBinding(h, peer.Binding)
		if err != nil {
			return nil, err
		}
		historical, err := projectnetwork.SelectRecoveryMemberContext(prefix, peer.InstanceID)
		if err != nil {
			return nil, err
		}
		matched := false
		for _, m := range current.Members {
			if m == historical.Member && m.InstanceID == peer.InstanceID && m.OperationID == peer.LaunchOperationID && m.Storage == peer.Storage {
				matched = true
			}
		}
		if !matched || !peer.After.StartedAt.Equal(historical.KnownStartedAt) {
			return nil, errors.New("recovered peer Stop lost current process or creator")
		}
		o := peer.After
		exits[peer.InstanceID] = projectMemberExit{o.StartedAt, o.FinishedAt, o.ExitCode, o.OOMKilled, peer.Storage}
	}
	return exits, nil
}
func (e *projectMemberStopEngine) inspectMembers(ctx context.Context, known []projectObservedMember, absent ...string) (projectMemberSnapshot, error) {
	if e.recovery == nil {
		return e.source.anchor.inspectProjectMemberSnapshot(ctx, known)
	}
	return e.recovery.inspectMembers(ctx, known, absent)
}

// Recovery-aware ordinary operations preserve all earlier retired identities and
// the current retirement's absence, including containers outside this network.
func (e *projectAnchorRecoveryReviewEngine) inspectMembers(ctx context.Context, known []projectObservedMember, absent []string) (projectMemberSnapshot, error) {
	fail := func(err error) (projectMemberSnapshot, error) { return projectMemberSnapshot{}, err }
	anchor := e.source.anchor
	ids, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if err = e.checkRetiredIDs(ids); err != nil {
		return fail(err)
	}
	for _, id := range absent {
		if slices.Contains(ids, id) {
			return fail(errors.New("retired member remains in engine inventory"))
		}
	}
	snapshot, err := anchor.inspectProjectMemberSnapshotMode(ctx, known, true)
	if err != nil {
		return fail(err)
	}
	after, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if !slices.Equal(ids, after) {
		return fail(errors.New("recovered member inventory changed"))
	}
	return snapshot, nil
}
