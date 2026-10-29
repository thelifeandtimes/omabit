//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorNamespaceSelection struct {
	Source    ProjectAnchorRecoveryReviewSelection
	Namespace projectnetwork.AnchorNamespace
}
type projectAnchorNamespaceEngine struct {
	observation                       *projectAnchorDrainEngine
	initial                           projectnetwork.AnchorNamespace
	before                            *projectnetwork.AnchorDrainObservation
	snapshot                          projectMemberSnapshot
	startAttempted, startAcknowledged bool
}

// OpenProjectAnchorNamespace requires retained clean-drain history under the
// project fence. It may Start only the exact stopped anchor after a durable
// dispatch; reopened intents are observation-only. No member is started or removed.
func OpenProjectAnchorNamespace(ctx context.Context, s ProjectAnchorNamespaceSelection) (projectnetwork.AnchorNamespaceDriver, error) {
	return openProjectAnchorNamespace(ctx, s, defaults())
}
func openProjectAnchorNamespace(ctx context.Context, s ProjectAnchorNamespaceSelection, d dependencies) (projectnetwork.AnchorNamespaceDriver, error) {
	copy := cloneProjectAnchorDrainSelection(ProjectAnchorDrainSelection{Source: s.Source, Drain: s.Namespace.Drain})
	h := anchorDrainHistory(copy.Source)
	intent := s.Namespace.Clone()
	if intent.Validate(h) != nil || intent.Phase == "observed" {
		return nil, errors.New("namespace succession requires valid incomplete intent")
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, copy.Source, d)
	if err != nil {
		return nil, err
	}
	review := raw.(*projectAnchorRecoveryReviewEngine)
	for i, m := range review.source.members {
		for _, p := range intent.Drain.Review.After.Processes {
			if p.InstanceID == m.contract.selection.InstanceID {
				review.source.members[i].processStartedAt = p.StartedAt
			}
		}
	}
	// An index beyond the complete drain selects no signal target. The shared
	// observation routine permits every recorded member's already proven exit.
	observer := &projectAnchorDrainEngine{review: review, history: h, initial: intent.Drain.Clone(), index: len(intent.Drain.Members)}
	return &projectAnchorNamespaceEngine{observation: observer, initial: intent}, nil
}
func (e *projectAnchorNamespaceEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.observation.VerifyEngine(ctx, id)
}
func (e *projectAnchorNamespaceEngine) selected(ctx context.Context, s projectnetwork.AnchorNamespace) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Validate(e.observation.history) != nil || s.OperationID != e.initial.OperationID || !reflect.DeepEqual(s.Drain, e.initial.Drain) {
		return errors.New("namespace succession selection changed")
	}
	if e.initial.Phase != "planned" && !reflect.DeepEqual(s, e.initial) {
		return errors.New("reopened namespace intent changed")
	}
	return nil
}
func (e *projectAnchorNamespaceEngine) InspectNamespace(ctx context.Context, s projectnetwork.AnchorNamespace) (projectnetwork.AnchorDrainObservation, error) {
	fail := func(err error) (projectnetwork.AnchorDrainObservation, error) {
		return projectnetwork.AnchorDrainObservation{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	first := e.initial.Phase == "planned" && e.before == nil
	if first && !reflect.DeepEqual(s, e.initial) {
		return fail(errors.New("namespace preflight is not the selected intent"))
	}
	if !first && e.initial.Phase == "planned" {
		if s.Before == nil || !reflect.DeepEqual(s.Before, e.before) || s.Phase != "observing" && s.Phase != "start-acknowledged" || s.Phase == "start-acknowledged" && !e.startAcknowledged {
			return fail(errors.New("namespace observation lacks committed preflight or acknowledged Start"))
		}
	}
	o, snapshot, err := e.observation.observe(ctx, s.Drain, true)
	if err != nil {
		return fail(err)
	}
	if err = s.ValidateObservation(e.observation.history, o, !first, time.Now().UTC()); err != nil {
		return fail(err)
	}
	if first {
		phase := "observing"
		if !o.Runtime.Anchor.Running {
			phase = "start-dispatched"
		}
		next, err := s.Advance(e.observation.history, s.Sequence, phase, &o)
		if err != nil {
			return fail(err)
		}
		e.before = next.Before
		e.snapshot = snapshot
	} else if e.before != nil {
		expected := make(map[string]container, len(e.snapshot.Containers))
		for id, v := range e.snapshot.Containers {
			expected[id] = v
		}
		if !e.before.Runtime.Anchor.Running {
			id := s.Drain.Review.Context.Binding.Anchor.ID
			v := expected[id]
			// Network endpoint state legitimately changes on Start. Its immutable
			// requested publications/bridge and complete inventory were revalidated.
			v.State = snapshot.Containers[id].State
			v.NetworkSettings = snapshot.Containers[id].NetworkSettings
			expected[id] = v
		}
		if !reflect.DeepEqual(expected, snapshot.Containers) {
			return fail(errors.New("namespace succession process metadata changed"))
		}
	}
	return o, nil
}
func (e *projectAnchorNamespaceEngine) StartRecoveryAnchor(ctx context.Context, s projectnetwork.AnchorNamespace) error {
	if err := e.selected(ctx, s); err != nil {
		return err
	}
	if e.initial.Phase != "planned" || e.before == nil || e.before.Runtime.Anchor.Running || !reflect.DeepEqual(s.Before, e.before) || s.Phase != "start-dispatched" || e.startAttempted {
		return errors.New("recovery anchor cannot dispatch or replay Start")
	}
	e.startAttempted = true
	if err := s.ValidateObservation(e.observation.history, *e.before, false, time.Now().UTC()); err != nil {
		return err
	}
	o, snapshot, err := e.observation.observe(ctx, s.Drain, true)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e.snapshot.Containers, snapshot.Containers) {
		return errors.New("recovery anchor or member metadata changed before Start")
	}
	if err = s.ValidateObservation(e.observation.history, o, false, time.Now().UTC()); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	q := e.observation.review.source.anchor.base.q
	id := s.Drain.Review.Context.Binding.Anchor.ID
	out, err := q.docker(ctx, q.d.times.command, "container", "start", id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != id {
		return errors.New("recovery anchor Start acknowledgement unresolved")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	e.startAcknowledged = true
	return nil
}
