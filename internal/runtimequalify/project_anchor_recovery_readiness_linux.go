//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorRecoveryReadinessSelection struct {
	Source      ProjectAnchorRecoveryReviewSelection
	Replacement projectnetwork.AnchorReplacement
}
type projectAnchorRecoveryReadinessEngine struct {
	review      *projectAnchorRecoveryReviewEngine
	history     projectnetwork.AnchorRecoveryHistory
	expected    projectnetwork.AnchorReplacement
	members     []projectObservedMember
	endpoints   map[string]string
	inspectedAt time.Time
	first       map[string]container
}

// OpenProjectAnchorRecoveryReadiness is an observation-only adapter for a complete
// replacement set. Callers retain the recovery fence and all causal history; this
// does not persist completion, adopt resources, or authorize later mutations.
func OpenProjectAnchorRecoveryReadiness(ctx context.Context, s ProjectAnchorRecoveryReadinessSelection) (projectnetwork.AnchorRecoveryReadinessDriver, error) {
	return openProjectAnchorRecoveryReadiness(ctx, s, defaults())
}
func openProjectAnchorRecoveryReadiness(ctx context.Context, s ProjectAnchorRecoveryReadinessSelection, d dependencies) (projectnetwork.AnchorRecoveryReadinessDriver, error) {
	copy := cloneProjectAnchorDrainSelection(ProjectAnchorDrainSelection{Source: s.Source, Drain: s.Replacement.Retirement.Namespace.Drain})
	h := anchorDrainHistory(copy.Source)
	intent := s.Replacement.Clone()
	if _, err := projectnetwork.RecoveredRuntimeBinding(h, intent); err != nil {
		return nil, err
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, copy.Source, d)
	if err != nil {
		return nil, err
	}
	e := &projectAnchorRecoveryReadinessEngine{review: raw.(*projectAnchorRecoveryReviewEngine), history: h, expected: intent, endpoints: map[string]string{}}
	last := intent.Members[len(intent.Members)-1].After
	for _, m := range intent.Members {
		path, err := pierstore.Absolute(d.home, m.Creation.Storage.Path)
		if err != nil {
			return nil, err
		}
		c, err := newRecoveryProjectMemberContract(copy.Source.Project, h, intent.Retirement, m.Creation.InstanceID, m.Creation.OperationID, d.home, filepath.Dir(path), e.review.source.anchor.base.contract.ImageID)
		if err != nil || c.digest != m.Container.ContractDigest || !reflect.DeepEqual(*c.recovery, m.Creation) {
			return nil, errors.New("recovery readiness contract differs from retained replacement")
		}
		observed := projectObservedMember{contract: c, id: m.Container.ID}
		for _, p := range last.Processes {
			if p.InstanceID == m.Creation.InstanceID {
				observed.processStartedAt = p.Process.StartedAt
			}
		}
		if observed.processStartedAt.IsZero() {
			return nil, errors.New("recovery readiness lacks exact process evidence")
		}
		e.members = append(e.members, observed)
	}
	for _, m := range h.Plan.Members {
		e.endpoints[fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)] = m.Ship
	}
	if err = e.review.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func (e *projectAnchorRecoveryReadinessEngine) VerifyEngine(ctx context.Context, id string) error {
	err := e.review.VerifyEngine(ctx, id)
	if err != nil {
		e.inspectedAt = time.Time{}
	}
	return err
}
func (e *projectAnchorRecoveryReadinessEngine) InspectRecoveryReadiness(ctx context.Context, s projectnetwork.AnchorReplacement) (projectnetwork.AnchorRetirementObservation, error) {
	e.inspectedAt = time.Time{}
	fail := func(err error) (projectnetwork.AnchorRetirementObservation, error) {
		return projectnetwork.AnchorRetirementObservation{}, err
	}
	if ctx.Err() != nil || !reflect.DeepEqual(s, e.expected) {
		return fail(errors.New("recovery readiness selection changed"))
	}
	if err := e.review.storage(ctx); err != nil {
		return fail(err)
	}
	anchor := e.review.source.anchor
	ids, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if err = e.review.checkRetiredIDs(ids); err != nil {
		return fail(err)
	}
	for _, m := range s.Retirement.Namespace.Drain.Review.Context.Members {
		if slices.Contains(ids, m.Context.Member.Container.ID) {
			return fail(errors.New("retired old container reappeared during readiness"))
		}
	}
	snapshot, err := anchor.inspectProjectMemberSnapshotMode(ctx, e.members, true)
	if err != nil {
		return fail(err)
	}
	after, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if !slices.Equal(ids, after) {
		return fail(errors.New("recovery readiness inventory changed"))
	}
	if err = e.review.storage(ctx); err != nil {
		return fail(err)
	}
	o := projectnetwork.AnchorRetirementObservation{Runtime: snapshot.Observation, CapturedAt: time.Now().UTC(), Piers: slices.Clone(s.Members[len(s.Members)-1].After.Piers)}
	o.AnchorProcess, err = recoveryProcess(snapshot.Containers[s.Retirement.Namespace.Drain.Review.Context.Binding.Anchor.ID])
	if err != nil {
		return fail(err)
	}
	for _, m := range e.members {
		state, err := recoveryProcess(snapshot.Containers[m.id])
		if err != nil {
			return fail(err)
		}
		o.Processes = append(o.Processes, projectnetwork.AnchorDrainProcess{InstanceID: m.contract.selection.InstanceID, Storage: m.contract.selection.Storage, Process: state})
	}
	if err = projectnetwork.ValidateRecoveryReadinessObservation(e.history, s, o, time.Now().UTC()); err != nil {
		return fail(err)
	}
	if e.first != nil && !reflect.DeepEqual(e.first, snapshot.Containers) {
		return fail(errors.New("recovery readiness container metadata changed"))
	}
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	e.first = snapshot.Containers
	e.inspectedAt = snapshot.Observation.ObservedAt
	return o, nil
}
func (e *projectAnchorRecoveryReadinessEngine) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if ctx.Err() != nil || e.endpoints[endpoint] == "" || e.inspectedAt.IsZero() || time.Since(e.inspectedAt) > 30*time.Second || time.Until(e.inspectedAt) > 5*time.Second {
		return "", errors.New("recovery readiness HTTP requires fresh selected publication evidence")
	}
	return e.review.source.anchor.base.q.d.hostIdentity(ctx, endpoint)
}
