//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorDrainSelection struct {
	Source ProjectAnchorRecoveryReviewSelection
	Drain  projectnetwork.AnchorDrain
}
type projectAnchorDrainEngine struct {
	review                              *projectAnchorRecoveryReviewEngine
	history                             projectnetwork.AnchorRecoveryHistory
	initial                             projectnetwork.AnchorDrain
	target                              projectObservedMember
	index                               int
	before                              *projectnetwork.AnchorDrainObservation
	snapshot                            projectMemberSnapshot
	signalAttempted, signalAcknowledged bool
}

func anchorDrainHistory(s ProjectAnchorRecoveryReviewSelection) projectnetwork.AnchorRecoveryHistory {
	return projectnetwork.AnchorRecoveryHistory{Epochs: s.Epochs, Plan: s.Project.Plan, Launches: s.Launches, Resumes: s.Resumes, Originals: s.Originals, Current: s.Current}
}

// OpenProjectAnchorDrain requires validated history and an accepted intent under
// the common project fence. The caller persists every phase before dispatch.
// This adapter never starts an anchor/member, removes a container or bootstraps.
func OpenProjectAnchorDrain(ctx context.Context, s ProjectAnchorDrainSelection) (projectnetwork.AnchorDrainDriver, error) {
	return openProjectAnchorDrain(ctx, s, defaults())
}
func openProjectAnchorDrain(ctx context.Context, s ProjectAnchorDrainSelection, d dependencies) (projectnetwork.AnchorDrainDriver, error) {
	h := anchorDrainHistory(s.Source)
	if s.Drain.Validate(h) != nil {
		return nil, errors.New("anchor drain requires validated retained intent")
	}
	index := -1
	for i, m := range s.Drain.Members {
		if m.Phase != "exited" {
			index = i
			break
		}
		if m.After == nil {
			return nil, errors.New("anchor drain previous exit missing")
		}
		for _, p := range m.After.Processes {
			if p.InstanceID == m.InstanceID && (p.Process.ExitCode != 0 || p.Process.OOMKilled) {
				return nil, errors.New("anchor drain previous exit requires review")
			}
		}
	}
	if index < 0 {
		return nil, errors.New("anchor drain has no pending member")
	}
	// Detach retained history before constructing any observer or contract.
	cloned := cloneProjectAnchorDrainSelection(s)
	h = anchorDrainHistory(cloned.Source)
	if cloned.Drain.Validate(h) != nil {
		return nil, errors.New("anchor drain source copy changed")
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, cloned.Source, d)
	if err != nil {
		return nil, err
	}
	review := raw.(*projectAnchorRecoveryReviewEngine)
	e := &projectAnchorDrainEngine{review: review, history: h, initial: cloned.Drain, index: index}
	for i, m := range review.source.members {
		for _, p := range cloned.Drain.Review.After.Processes {
			if p.InstanceID == m.contract.selection.InstanceID {
				review.source.members[i].processStartedAt = p.StartedAt
			}
		}
		if m.contract.selection.InstanceID == cloned.Drain.Members[index].InstanceID {
			e.target = review.source.members[i]
		}
	}
	if e.target.id == "" {
		return nil, errors.New("anchor drain target is missing")
	}
	return e, nil
}
func (e *projectAnchorDrainEngine) selected(ctx context.Context, s projectnetwork.AnchorDrain) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Validate(e.history) != nil || s.OperationID != e.initial.OperationID || !reflect.DeepEqual(s.Review, e.initial.Review) || len(s.Members) != len(e.initial.Members) {
		return errors.New("anchor drain selection changed")
	}
	for i, m := range s.Members {
		if i != e.index && !reflect.DeepEqual(m, e.initial.Members[i]) {
			return errors.New("anchor drain peer history changed")
		}
	}
	return nil
}
func (e *projectAnchorDrainEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.review.VerifyEngine(ctx, id)
}
func recoveryProcess(v container) (projectnetwork.RecoveryProcessState, error) {
	if v.State == nil || v.State.ExitCode == nil || v.State.OOMKilled == nil {
		return projectnetwork.RecoveryProcessState{}, errUntrustedContainer
	}
	started, se := time.Parse(time.RFC3339Nano, v.State.StartedAt)
	finished, fe := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
	if se != nil || fe != nil {
		return projectnetwork.RecoveryProcessState{}, errUntrustedContainer
	}
	return projectnetwork.RecoveryProcessState{StartedAt: started, FinishedAt: finished, ExitCode: *v.State.ExitCode, OOMKilled: *v.State.OOMKilled}, nil
}
func (e *projectAnchorDrainEngine) observe(ctx context.Context, s projectnetwork.AnchorDrain, mayExit bool) (projectnetwork.AnchorDrainObservation, projectMemberSnapshot, error) {
	fail := func(err error) (projectnetwork.AnchorDrainObservation, projectMemberSnapshot, error) {
		return projectnetwork.AnchorDrainObservation{}, projectMemberSnapshot{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	if err := e.review.storage(ctx); err != nil {
		return fail(err)
	}
	known := slices.Clone(e.review.source.members)
	for i := range known {
		for j, m := range s.Members {
			if m.InstanceID == known[i].contract.selection.InstanceID {
				known[i].allowExited = j < e.index || j == e.index && mayExit
			}
		}
	}
	var inventory []string
	if len(e.review.retired) > 0 {
		var err error
		inventory, err = e.review.source.anchor.listIDs(ctx)
		if err != nil {
			return fail(err)
		}
		if err = e.review.checkRetiredIDs(inventory); err != nil {
			return fail(err)
		}
	}
	snapshot, err := e.review.source.anchor.inspectProjectMemberSnapshotMode(ctx, known, true)
	if err != nil {
		return fail(err)
	}
	if err = e.review.storage(ctx); err != nil {
		return fail(err)
	}
	if len(e.review.retired) > 0 {
		after, err := e.review.source.anchor.listIDs(ctx)
		if err != nil {
			return fail(err)
		}
		if !slices.Equal(inventory, after) {
			return fail(errors.New("anchor drain history inventory changed"))
		}
	}
	o := projectnetwork.AnchorDrainObservation{Runtime: snapshot.Observation, CapturedAt: time.Now().UTC()}
	o.AnchorProcess, err = recoveryProcess(snapshot.Containers[s.Review.Context.Binding.Anchor.ID])
	if err != nil {
		return fail(err)
	}
	for _, m := range s.Review.Context.Members {
		state, err := recoveryProcess(snapshot.Containers[m.Context.Member.Container.ID])
		if err != nil {
			return fail(err)
		}
		o.Processes = append(o.Processes, projectnetwork.AnchorDrainProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, Process: state})
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return o, snapshot, nil
}
func (e *projectAnchorDrainEngine) InspectDrain(ctx context.Context, s projectnetwork.AnchorDrain) (projectnetwork.AnchorDrainObservation, error) {
	if err := e.selected(ctx, s); err != nil {
		return projectnetwork.AnchorDrainObservation{}, err
	}
	initialPhase := e.initial.Members[e.index].Phase
	if !reflect.DeepEqual(s, e.initial) {
		return projectnetwork.AnchorDrainObservation{}, errors.New("anchor drain inspection differs from selected intent")
	}
	if initialPhase == "planned" {
		if e.before != nil || e.signalAttempted {
			return projectnetwork.AnchorDrainObservation{}, errors.New("anchor drain cannot replace its accepted process")
		}
		o, snapshot, err := e.observe(ctx, s, false)
		if err != nil {
			return projectnetwork.AnchorDrainObservation{}, err
		}
		next, err := s.Advance(e.history, e.index, s.Sequence, "stop-dispatched", &o)
		if err != nil {
			return projectnetwork.AnchorDrainObservation{}, err
		}
		e.before = next.Members[e.index].Before
		e.snapshot = snapshot
		return o, nil
	}
	if initialPhase != "stop-dispatched" && initialPhase != "stop-signalled" {
		return projectnetwork.AnchorDrainObservation{}, errors.New("anchor drain has no unresolved signal")
	}
	o, _, err := e.observe(ctx, s, true)
	if err == nil {
		err = s.ValidateObservation(e.history, e.index, o, true, time.Now().UTC())
	}
	return o, err
}
func (e *projectAnchorDrainEngine) SignalDrainMember(ctx context.Context, s projectnetwork.AnchorDrain, index int) error {
	if err := e.selected(ctx, s); err != nil {
		return err
	}
	if index != e.index || e.initial.Members[index].Phase != "planned" || s.Members[index].Phase != "stop-dispatched" || e.before == nil || !reflect.DeepEqual(s.Members[index].Before, e.before) || e.signalAttempted {
		return errors.New("anchor drain signal cannot dispatch or replay")
	}
	e.signalAttempted = true
	if err := s.ValidateObservation(e.history, index, *e.before, false, time.Now().UTC()); err != nil {
		return err
	}
	o, snapshot, err := e.observe(ctx, s, false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e.snapshot.Containers, snapshot.Containers) {
		return errors.New("anchor drain process metadata changed before signal")
	}
	if err = s.ValidateObservation(e.history, index, o, false, time.Now().UTC()); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	q := e.review.source.anchor.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "kill", "--signal", "SIGTERM", e.target.id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.target.id {
		return errors.New("anchor drain signal acknowledgement unresolved")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	e.signalAcknowledged = true
	return nil
}
func (e *projectAnchorDrainEngine) AwaitDrainExit(ctx context.Context, s projectnetwork.AnchorDrain, index int) (projectnetwork.AnchorDrainObservation, error) {
	fail := func(err error) (projectnetwork.AnchorDrainObservation, error) {
		return projectnetwork.AnchorDrainObservation{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	if index != e.index || !e.signalAcknowledged || e.before == nil || s.Members[index].Phase != "stop-signalled" || !reflect.DeepEqual(s.Members[index].Before, e.before) {
		return fail(errors.New("anchor drain has no acknowledged dispatch"))
	}
	q := e.review.source.anchor.base.q
	ctx, cancel := context.WithTimeout(ctx, q.d.times.stop)
	defer cancel()
	for {
		rows, err := e.review.source.anchor.inspectProjectContainers(ctx, []string{e.target.id})
		if err != nil {
			return fail(err)
		}
		v := rows[e.target.id]
		if !*v.State.Running {
			if err = e.target.contract.validateExitedContainer(v, e.target.id, e.target.processStartedAt); err != nil {
				return fail(err)
			}
			o, snapshot, err := e.observe(ctx, s, true)
			if err != nil {
				return fail(err)
			}
			expected := make(map[string]container, len(e.snapshot.Containers))
			for id, v := range e.snapshot.Containers {
				expected[id] = v
			}
			target := expected[e.target.id]
			target.State = snapshot.Containers[e.target.id].State
			expected[e.target.id] = target
			if !reflect.DeepEqual(expected, snapshot.Containers) {
				return fail(errors.New("anchor drain peer or anchor process changed during exit"))
			}
			if err = s.ValidateObservation(e.history, index, o, true, time.Now().UTC()); err != nil {
				return fail(err)
			}
			return o, nil
		}
		if err = e.target.contract.validateContainer(v, e.target.id, true); err != nil {
			return fail(err)
		}
		started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		if err != nil || !started.Equal(e.target.processStartedAt) {
			return fail(errors.New("anchor drain target restarted while awaiting exit"))
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fail(ctx.Err())
		case <-timer.C:
		}
	}
}

func cloneProjectAnchorDrainSelection(s ProjectAnchorDrainSelection) ProjectAnchorDrainSelection {
	h := anchorDrainHistory(s.Source).Clone()
	s.Source.Project.Plan = h.Plan
	s.Source.Epochs = h.Epochs
	s.Source.Launches = h.Launches
	s.Source.Resumes = h.Resumes
	s.Source.Originals = h.Originals
	s.Source.Current = h.Current
	s.Drain = s.Drain.Clone()
	return s
}
