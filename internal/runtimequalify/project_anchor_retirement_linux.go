//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorRetirementSelection struct {
	Source     ProjectAnchorRecoveryReviewSelection
	Retirement projectnetwork.AnchorRetirement
}
type projectAnchorRetirementEngine struct {
	review                  *projectAnchorRecoveryReviewEngine
	history                 projectnetwork.AnchorRecoveryHistory
	initial                 projectnetwork.AnchorRetirement
	index                   int
	target                  projectObservedMember
	before                  *projectnetwork.AnchorRetirementObservation
	snapshot                projectMemberSnapshot
	attempted, acknowledged bool
}

// OpenProjectAnchorRetirement binds one old container's retained clean exit to
// its removal under the successor namespace. The caller holds the project fence
// and commits dispatch before removal. Reopened pending intents can only observe
// absence. This never removes a pier or starts a container.
func OpenProjectAnchorRetirement(ctx context.Context, s ProjectAnchorRetirementSelection) (projectnetwork.AnchorRetirementDriver, error) {
	return openProjectAnchorRetirement(ctx, s, defaults())
}
func openProjectAnchorRetirement(ctx context.Context, s ProjectAnchorRetirementSelection, d dependencies) (projectnetwork.AnchorRetirementDriver, error) {
	copy := cloneProjectAnchorDrainSelection(ProjectAnchorDrainSelection{Source: s.Source, Drain: s.Retirement.Namespace.Drain})
	h := anchorDrainHistory(copy.Source)
	intent := s.Retirement.Clone()
	if intent.Validate(h) != nil || intent.Complete() {
		return nil, errors.New("retirement requires validated incomplete intent")
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, copy.Source, d)
	if err != nil {
		return nil, err
	}
	review := raw.(*projectAnchorRecoveryReviewEngine)
	index := 0
	for intent.Members[index].Phase == "absent" {
		index++
	}
	e := &projectAnchorRetirementEngine{review: review, history: h, initial: intent, index: index}
	for i, m := range review.source.members {
		for _, p := range intent.Namespace.Drain.Review.After.Processes {
			if p.InstanceID == m.contract.selection.InstanceID {
				review.source.members[i].processStartedAt = p.StartedAt
			}
		}
		review.source.members[i].allowExited = true
		if m.contract.selection.InstanceID == intent.Members[index].InstanceID {
			e.target = review.source.members[i]
		}
	}
	if e.target.id == "" {
		return nil, errors.New("retirement exact target is missing")
	}
	return e, nil
}
func (e *projectAnchorRetirementEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.review.VerifyEngine(ctx, id)
}
func (e *projectAnchorRetirementEngine) selected(ctx context.Context, s projectnetwork.AnchorRetirement) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Validate(e.history) != nil || s.OperationID != e.initial.OperationID || !reflect.DeepEqual(s.Namespace, e.initial.Namespace) || len(s.Members) != len(e.initial.Members) {
		return errors.New("retirement authority changed")
	}
	for i, m := range s.Members {
		if i != e.index && !reflect.DeepEqual(m, e.initial.Members[i]) {
			return errors.New("retirement peer history changed")
		}
	}
	if e.initial.Members[e.index].Phase != "planned" && !reflect.DeepEqual(s, e.initial) {
		return errors.New("reopened retirement intent changed")
	}
	return nil
}
func (e *projectAnchorRetirementEngine) storage(ctx context.Context) error {
	if err := e.review.storage(ctx); err != nil {
		return err
	}
	for _, p := range e.initial.Namespace.After.Processes {
		path, err := pierstore.Check(e.review.source.anchor.base.q.d.home, p.Storage)
		if err != nil {
			return err
		}
		if pierstore.Busy(path) {
			return errors.New("retirement pier still has a Vere lock")
		}
		if err = pierstore.CheckStateFiles(ctx, path); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (e *projectAnchorRetirementEngine) observe(ctx context.Context, s projectnetwork.AnchorRetirement, absent bool) (projectnetwork.AnchorRetirementObservation, projectMemberSnapshot, error) {
	fail := func(err error) (projectnetwork.AnchorRetirementObservation, projectMemberSnapshot, error) {
		return projectnetwork.AnchorRetirementObservation{}, projectMemberSnapshot{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	if err := e.storage(ctx); err != nil {
		return fail(err)
	}
	excluded := map[string]bool{}
	for i, m := range s.Members {
		if i < e.index || i == e.index && absent {
			excluded[m.InstanceID] = true
		}
	}
	known := []projectObservedMember{}
	absentIDs := []string{}
	for _, m := range e.review.source.members {
		if excluded[m.contract.selection.InstanceID] {
			absentIDs = append(absentIDs, m.id)
		} else {
			known = append(known, m)
		}
	}
	anchor := e.review.source.anchor
	ids, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if err = e.review.checkRetiredIDs(ids); err != nil {
		return fail(err)
	}
	for _, id := range absentIDs {
		if slices.Contains(ids, id) {
			return fail(errors.New("retired container remains present"))
		}
	}
	snapshot, err := anchor.inspectProjectMemberSnapshotMode(ctx, known, true)
	if err != nil {
		return fail(err)
	}
	afterIDs, err := anchor.listIDs(ctx)
	if err != nil {
		return fail(err)
	}
	if !slices.Equal(ids, afterIDs) {
		return fail(errors.New("retirement inventory changed"))
	}
	if err = e.storage(ctx); err != nil {
		return fail(err)
	}
	o := projectnetwork.AnchorRetirementObservation{Runtime: snapshot.Observation, CapturedAt: time.Now().UTC()}
	o.AnchorProcess, err = recoveryProcess(snapshot.Containers[s.Namespace.Drain.Review.Context.Binding.Anchor.ID])
	if err != nil {
		return fail(err)
	}
	for _, p := range s.Namespace.After.Processes {
		o.Piers = append(o.Piers, projectnetwork.RecoveryPier{InstanceID: p.InstanceID, Storage: p.Storage})
	}
	for _, m := range known {
		p, err := recoveryProcess(snapshot.Containers[m.id])
		if err != nil {
			return fail(err)
		}
		for _, pier := range o.Piers {
			if pier.InstanceID == m.contract.selection.InstanceID {
				o.Processes = append(o.Processes, projectnetwork.AnchorDrainProcess{InstanceID: pier.InstanceID, Storage: pier.Storage, Process: p})
			}
		}
	}
	if err = s.ValidateObservation(e.history, e.index, o, absent, time.Now().UTC()); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return o, snapshot, nil
}
func (e *projectAnchorRetirementEngine) InspectRetirement(ctx context.Context, s projectnetwork.AnchorRetirement) (projectnetwork.AnchorRetirementObservation, error) {
	fail := func(err error) (projectnetwork.AnchorRetirementObservation, error) {
		return projectnetwork.AnchorRetirementObservation{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	first := e.initial.Members[e.index].Phase == "planned" && e.before == nil
	if first && !reflect.DeepEqual(s, e.initial) {
		return fail(errors.New("retirement preflight differs from selected intent"))
	}
	if !first && e.initial.Members[e.index].Phase == "planned" {
		if s.Members[e.index].Phase != "remove-acknowledged" || !e.acknowledged || !reflect.DeepEqual(s.Members[e.index].Before, e.before) {
			return fail(errors.New("retirement lacks this dispatch acknowledgement"))
		}
	}
	o, snapshot, err := e.observe(ctx, s, !first)
	if err != nil {
		return fail(err)
	}
	if first {
		next, err := s.Advance(e.history, e.index, s.Sequence, "remove-dispatched", &o)
		if err != nil {
			return fail(err)
		}
		e.before = next.Members[e.index].Before
		e.snapshot = snapshot
	} else if e.before != nil {
		expected := map[string]container{}
		for id, c := range e.snapshot.Containers {
			if id != e.target.id {
				expected[id] = c
			}
		}
		if !reflect.DeepEqual(expected, snapshot.Containers) {
			return fail(errors.New("retirement changed anchor or peer metadata"))
		}
	}
	return o, nil
}
func (e *projectAnchorRetirementEngine) RemoveRetiredMember(ctx context.Context, s projectnetwork.AnchorRetirement, index int) error {
	if err := e.selected(ctx, s); err != nil {
		return err
	}
	if index != e.index || e.initial.Members[index].Phase != "planned" || s.Members[index].Phase != "remove-dispatched" || e.before == nil || !reflect.DeepEqual(s.Members[index].Before, e.before) || e.attempted {
		return errors.New("retirement cannot dispatch or replay removal")
	}
	e.attempted = true
	if err := s.ValidateObservation(e.history, index, *e.before, false, time.Now().UTC()); err != nil {
		return err
	}
	o, snapshot, err := e.observe(ctx, s, false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e.snapshot.Containers, snapshot.Containers) {
		return errors.New("retirement process metadata changed before removal")
	}
	if err = s.ValidateObservation(e.history, index, *e.before, false, time.Now().UTC()); err != nil {
		return err
	}
	if err = s.ValidateObservation(e.history, index, o, false, time.Now().UTC()); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	q := e.review.source.anchor.base.q
	// No force, volume deletion, signal, Start or pier mutation.
	out, err := q.docker(ctx, q.d.times.command, "container", "rm", e.target.id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.target.id {
		return errors.New("retirement removal acknowledgement unresolved")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	e.acknowledged = true
	return nil
}
