//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorRecoveryReviewSelection struct {
	Epochs    []projectnetwork.AnchorRecoveryEpoch
	Project   ProjectPreparationSelection
	Launches  []projectnetwork.MemberLaunch
	Resumes   []projectnetwork.MemberResume
	Originals []projectnetwork.InitialMemberReadiness
	Current   []projectnetwork.CurrentMemberReadiness
}
type projectAnchorRecoveryReviewEngine struct {
	retired  []string
	source   *projectInitialReadinessEngine
	expected projectnetwork.AnchorRecoveryContext
	first    map[string]container
}

// OpenProjectAnchorRecoveryReview is read-only. The caller holds the common
// project fence, excludes pending/uncertain work and supplies validated history
// with every known current-process receipt. This observes a fault without
// changing existing lifecycle authority or adopting a new namespace.
func OpenProjectAnchorRecoveryReview(ctx context.Context, s ProjectAnchorRecoveryReviewSelection) (projectnetwork.AnchorRecoveryObserver, error) {
	return openProjectAnchorRecoveryReview(ctx, s, defaults())
}
func openProjectAnchorRecoveryReview(ctx context.Context, s ProjectAnchorRecoveryReviewSelection, d dependencies) (projectnetwork.AnchorRecoveryObserver, error) {
	h := anchorDrainHistory(s).Clone()
	project := s.Project
	project.Plan = h.Plan
	c, err := projectnetwork.SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return nil, err
	}
	current, err := projectnetwork.ResolveAnchorRecoveryHistory(h)
	if err != nil {
		return nil, err
	}
	source, err := openProjectRecoveryMemberGenerations(ctx, project, h, current, d)
	if err != nil {
		return nil, err
	}
	for i, m := range source.members {
		for _, selected := range c.Members {
			if m.contract.selection.InstanceID == selected.Context.Member.InstanceID {
				source.members[i].processStartedAt = selected.StartedAt
			}
		}
	}
	e := &projectAnchorRecoveryReviewEngine{source: source, expected: c, retired: retiredRecoveryHistoryIDs(h, current)}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func (e *projectAnchorRecoveryReviewEngine) storage(ctx context.Context) error {
	if err := e.source.parents(ctx); err != nil {
		return err
	}
	for _, m := range e.expected.Members {
		if _, err := pierstore.Check(e.source.anchor.base.q.d.home, m.Context.Storage); err != nil {
			return errors.New("anchor recovery review initialized pier changed")
		}
	}
	return ctx.Err()
}
func (e *projectAnchorRecoveryReviewEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.source.VerifyEngine(ctx, id)
}
func (e *projectAnchorRecoveryReviewEngine) InspectAnchorRecovery(ctx context.Context, c projectnetwork.AnchorRecoveryContext) (projectnetwork.AnchorRecoveryObservation, error) {
	refuse := func() (projectnetwork.AnchorRecoveryObservation, error) {
		return projectnetwork.AnchorRecoveryObservation{}, errors.New("anchor recovery review ownership or process changed")
	}
	if ctx.Err() != nil || !reflect.DeepEqual(c, e.expected) || e.storage(ctx) != nil {
		return refuse()
	}

	var inventory []string
	if len(e.retired) > 0 {
		var err error
		inventory, err = e.source.anchor.listIDs(ctx)
		if err != nil {
			return refuse()
		}
		if e.checkRetiredIDs(inventory) != nil {
			return refuse()
		}
	}
	snapshot, err := e.source.anchor.inspectProjectMemberSnapshotMode(ctx, e.source.members, true)
	if err != nil || e.storage(ctx) != nil {
		return refuse()
	}

	if len(e.retired) > 0 {
		after, err := e.source.anchor.listIDs(ctx)
		if err != nil || !slices.Equal(inventory, after) {
			return refuse()
		}
	}
	if e.first != nil && !reflect.DeepEqual(e.first, snapshot.Containers) {
		return refuse()
	}
	now := time.Now().UTC()
	result := projectnetwork.AnchorRecoveryObservation{Runtime: snapshot.Observation, CapturedAt: now}
	for _, m := range c.Members {
		v := snapshot.Containers[m.Context.Member.Container.ID]
		if v.State == nil || v.State.Running == nil || !*v.State.Running {
			return refuse()
		}
		started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		if err != nil || started.IsZero() || started.After(now) || !m.StartedAt.IsZero() && !started.Equal(m.StartedAt) {
			return refuse()
		}
		result.Processes = append(result.Processes, projectnetwork.AnchorRecoveryProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, StartedAt: started})
	}
	if ctx.Err() != nil {
		return refuse()
	}
	e.first = snapshot.Containers
	return result, nil
}

// Retired identities are excluded globally, even if a stale container moved to
// another network. Later recovery phases reuse their complete inventory read.
func (e *projectAnchorRecoveryReviewEngine) checkRetiredIDs(ids []string) error {
	for _, id := range e.retired {
		if slices.Contains(ids, id) {
			return errors.New("historically retired project container reappeared")
		}
	}
	return nil
}
