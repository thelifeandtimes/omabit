//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectAnchorReplacementSelection struct {
	Source      ProjectAnchorRecoveryReviewSelection
	Replacement projectnetwork.AnchorReplacement
}
type projectAnchorReplacementEngine struct {
	review                                             *projectAnchorRecoveryReviewEngine
	history                                            projectnetwork.AnchorRecoveryHistory
	initial                                            projectnetwork.AnchorReplacement
	contracts                                          []projectMemberContract
	index                                              int
	before, created                                    *projectnetwork.AnchorRetirementObservation
	beforeSnapshot, createdSnapshot                    projectMemberSnapshot
	createdID                                          string
	createAttempted, startAttempted, startAcknowledged bool
	continueCreated                                    bool
}

// OpenProjectAnchorReplacement requires accepted retained intent under the common
// project fence. It cannot adopt old/new containers or bootstrap a pier. Reopened
// pending Starts are observation-only; uncertain Create requires separate review.
func OpenProjectAnchorReplacement(ctx context.Context, s ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementDriver, error) {
	return openProjectAnchorReplacement(ctx, s, defaults())
}
func openProjectAnchorReplacement(ctx context.Context, s ProjectAnchorReplacementSelection, d dependencies) (projectnetwork.AnchorReplacementDriver, error) {
	return openProjectAnchorReplacementSelected(ctx, s, d, false)
}

// OpenProjectCreatedAnchorReplacement is only for separately admitted continuation
// of a retained exact Create reply. The returned interface cannot create a container.
func OpenProjectCreatedAnchorReplacement(ctx context.Context, s ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementStartDriver, error) {
	e, err := openProjectAnchorReplacementSelected(ctx, s, defaults(), true)
	if err != nil {
		return nil, err
	}
	return struct {
		projectnetwork.AnchorReplacementStartDriver
	}{e}, nil
}
func openProjectAnchorReplacementSelected(ctx context.Context, s ProjectAnchorReplacementSelection, d dependencies, continueCreated bool) (*projectAnchorReplacementEngine, error) {
	copy := cloneProjectAnchorDrainSelection(ProjectAnchorDrainSelection{Source: s.Source, Drain: s.Replacement.Retirement.Namespace.Drain})
	h := anchorDrainHistory(copy.Source)
	intent := s.Replacement.Clone()
	if intent.Validate(h) != nil || intent.Running() {
		return nil, errors.New("replacement requires valid incomplete intent")
	}
	index := 0
	for intent.Members[index].Phase == "running" {
		index++
	}
	phase := intent.Members[index].Phase
	if continueCreated && phase != "created" || !continueCreated && phase != "planned" && phase != "start-dispatched" && phase != "start-acknowledged" {
		return nil, errors.New("uncertain replacement Create requires review of its retained ID")
	}
	raw, err := openProjectAnchorRecoveryReview(ctx, copy.Source, d)
	if err != nil {
		return nil, err
	}
	e := &projectAnchorReplacementEngine{review: raw.(*projectAnchorRecoveryReviewEngine), history: h, initial: intent, index: index, continueCreated: continueCreated}
	for _, m := range intent.Members {
		path, err := pierstore.Absolute(d.home, m.Creation.Storage.Path)
		if err != nil {
			return nil, err
		}
		c, err := newRecoveryProjectMemberContract(copy.Source.Project, h, intent.Retirement, m.Creation.InstanceID, m.Creation.OperationID, d.home, filepath.Dir(path), e.review.source.anchor.base.contract.ImageID)
		if err != nil || c.digest != m.Container.ContractDigest || !reflect.DeepEqual(*c.recovery, m.Creation) {
			return nil, errors.New("replacement contract differs from retained intent")
		}
		e.contracts = append(e.contracts, c)
	}
	if phase != "planned" {
		e.createdID = intent.Members[index].Container.ID
	}
	if continueCreated {
		e.before = intent.Members[index].Before
	}
	return e, nil
}
func (e *projectAnchorReplacementEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.review.VerifyEngine(ctx, id)
}
func (e *projectAnchorReplacementEngine) selected(ctx context.Context, s projectnetwork.AnchorReplacement) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Validate(e.history) != nil || s.OperationID != e.initial.OperationID || !reflect.DeepEqual(s.Retirement, e.initial.Retirement) || len(s.Members) != len(e.initial.Members) {
		return errors.New("replacement authority changed")
	}
	for i, m := range s.Members {
		if i != e.index && !reflect.DeepEqual(m, e.initial.Members[i]) || i == e.index && (!reflect.DeepEqual(m.Creation, e.initial.Members[i].Creation) || m.Container.ContractDigest != e.contracts[i].digest) {
			return errors.New("replacement member selection changed")
		}
	}
	m := s.Members[e.index]
	if e.initial.Members[e.index].Phase != "planned" && !e.continueCreated {
		if !reflect.DeepEqual(s, e.initial) {
			return errors.New("reopened replacement changed")
		}
		return nil
	}
	if m.Container.ID != "" && m.Container.ID != e.createdID || m.Before != nil && !reflect.DeepEqual(m.Before, e.before) || m.Created != nil && !reflect.DeepEqual(m.Created, e.created) {
		return errors.New("replacement dispatch evidence changed")
	}
	return nil
}
func (e *projectAnchorReplacementEngine) storage(ctx context.Context, running bool) error {
	if err := e.review.storage(ctx); err != nil {
		return err
	}
	for i := e.index; i < len(e.contracts); i++ {
		if i == e.index && running {
			continue
		}
		if err := e.contracts[i].checkStorageBeforeCreate(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (e *projectAnchorReplacementEngine) observe(ctx context.Context, s projectnetwork.AnchorReplacement, stage string) (projectnetwork.AnchorRetirementObservation, projectMemberSnapshot, error) {
	fail := func(err error) (projectnetwork.AnchorRetirementObservation, projectMemberSnapshot, error) {
		return projectnetwork.AnchorRetirementObservation{}, projectMemberSnapshot{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	if err := e.storage(ctx, stage == "running"); err != nil {
		return fail(err)
	}
	known := []projectObservedMember{}
	for i, m := range s.Members {
		if i > e.index || i == e.index && stage == "absent" {
			continue
		}
		member := projectObservedMember{contract: e.contracts[i], id: m.Container.ID}
		if i < e.index {
			for _, p := range m.After.Processes {
				if p.InstanceID == m.Creation.InstanceID {
					member.processStartedAt = p.Process.StartedAt
				}
			}
		}
		known = append(known, member)
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
			return fail(errors.New("retired old container reappeared"))
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
		return fail(errors.New("replacement inventory changed"))
	}
	if err = e.storage(ctx, stage == "running"); err != nil {
		return fail(err)
	}
	o := projectnetwork.AnchorRetirementObservation{Runtime: snapshot.Observation, CapturedAt: time.Now().UTC()}
	o.AnchorProcess, err = recoveryProcess(snapshot.Containers[s.Retirement.Namespace.Drain.Review.Context.Binding.Anchor.ID])
	if err != nil {
		return fail(err)
	}
	last := s.Retirement.Members[len(s.Retirement.Members)-1].After
	o.Piers = slices.Clone(last.Piers)
	for _, m := range known {
		p, err := recoveryProcess(snapshot.Containers[m.id])
		if err != nil {
			return fail(err)
		}
		o.Processes = append(o.Processes, projectnetwork.AnchorDrainProcess{InstanceID: m.contract.selection.InstanceID, Storage: m.contract.selection.Storage, Process: p})
	}
	if err = s.ValidateObservation(e.history, e.index, o, stage, time.Now().UTC()); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return o, snapshot, nil
}
func withoutReplacementContainer(rows map[string]container, id string) map[string]container {
	out := map[string]container{}
	for k, v := range rows {
		if k != id {
			out[k] = v
		}
	}
	return out
}
func (e *projectAnchorReplacementEngine) InspectReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, index int, stage string) (projectnetwork.AnchorRetirementObservation, error) {
	fail := func(err error) (projectnetwork.AnchorRetirementObservation, error) {
		return projectnetwork.AnchorRetirementObservation{}, err
	}
	if err := e.selected(ctx, s); err != nil {
		return fail(err)
	}
	if index != e.index {
		return fail(errors.New("replacement selected wrong member"))
	}
	m := s.Members[index]
	fresh := e.initial.Members[index].Phase == "planned" || e.continueCreated
	switch stage {
	case "absent":
		if !fresh || m.Phase != "planned" || e.before != nil || e.createAttempted {
			return fail(errors.New("replacement absence cannot replay"))
		}
	case "created":
		if !fresh || m.Phase != "created" || e.created != nil || e.createdID == "" {
			return fail(errors.New("replacement created observation has no exact dispatch"))
		}
	case "running":
		if fresh && (m.Phase != "start-acknowledged" || !e.startAcknowledged) {
			return fail(errors.New("replacement running observation lacks acknowledged dispatch"))
		}
	default:
		return fail(errors.New("unsupported replacement observation"))
	}
	if stage != "running" {
		own := ""
		if stage == "created" {
			own = e.createdID
		}
		if err := e.review.source.anchor.rejectConflictingProjectMounts(ctx, e.contracts[index], own); err != nil {
			return fail(err)
		}
	}
	o, snapshot, err := e.observe(ctx, s, stage)
	if err != nil {
		return fail(err)
	}
	if stage == "absent" {
		next, err := s.Advance(e.history, index, s.Sequence, "create-dispatched", projectnetwork.AnchorReplacementResult{Observation: &o})
		if err != nil {
			return fail(err)
		}
		e.before = next.Members[index].Before
		e.beforeSnapshot = snapshot
	} else if stage == "created" {
		if !e.continueCreated && !reflect.DeepEqual(e.beforeSnapshot.Containers, withoutReplacementContainer(snapshot.Containers, e.createdID)) {
			return fail(errors.New("replacement Create changed peer metadata"))
		}
		next, err := s.Advance(e.history, index, s.Sequence, "start-dispatched", projectnetwork.AnchorReplacementResult{Observation: &o})
		if err != nil {
			return fail(err)
		}
		e.created = next.Members[index].Created
		e.createdSnapshot = snapshot
	} else if fresh {
		expected := map[string]container{}
		for id, c := range e.createdSnapshot.Containers {
			expected[id] = c
		}
		target := expected[e.createdID]
		target.State = snapshot.Containers[e.createdID].State
		target.NetworkSettings = snapshot.Containers[e.createdID].NetworkSettings
		expected[e.createdID] = target
		if !reflect.DeepEqual(expected, snapshot.Containers) {
			return fail(errors.New("replacement Start changed immutable or peer metadata"))
		}
	}
	return o, nil
}
func (e *projectAnchorReplacementEngine) CreateReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, index int) (string, error) {
	if err := e.selected(ctx, s); err != nil {
		return "", err
	}
	if index != e.index || e.initial.Members[index].Phase != "planned" || s.Members[index].Phase != "create-dispatched" || e.before == nil || e.createAttempted {
		return "", errors.New("replacement Create cannot dispatch or replay")
	}
	e.createAttempted = true
	if err := s.ValidateObservation(e.history, index, *e.before, "absent", time.Now().UTC()); err != nil {
		return "", err
	}
	if err := e.review.source.anchor.rejectConflictingProjectMounts(ctx, e.contracts[index], ""); err != nil {
		return "", err
	}
	o, snapshot, err := e.observe(ctx, s, "absent")
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(e.beforeSnapshot.Containers, snapshot.Containers) {
		return "", errors.New("replacement absence changed before Create")
	}
	if err = s.ValidateObservation(e.history, index, *e.before, "absent", time.Now().UTC()); err != nil {
		return "", err
	}
	if err = s.ValidateObservation(e.history, index, o, "absent", time.Now().UTC()); err != nil {
		return "", err
	}
	q := e.review.source.anchor.base.q
	out, err := q.dockerCreateID(ctx, q.d.times.command, e.contracts[index].createArgs()...)
	id := strings.TrimSpace(out)
	if !containerIDPattern.MatchString(id) {
		return "", errors.New("replacement Create returned no exact ID")
	}
	e.createdID = id
	return id, err
}
func (e *projectAnchorReplacementEngine) StartReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, index int) error {
	if err := e.selected(ctx, s); err != nil {
		return err
	}
	if index != e.index || e.initial.Members[index].Phase != "planned" && !e.continueCreated || s.Members[index].Phase != "start-dispatched" || e.created == nil || e.startAttempted {
		return errors.New("replacement Start cannot dispatch or replay")
	}
	e.startAttempted = true
	if err := s.ValidateObservation(e.history, index, *e.created, "created", time.Now().UTC()); err != nil {
		return err
	}
	if err := e.review.source.anchor.rejectConflictingProjectMounts(ctx, e.contracts[index], e.createdID); err != nil {
		return err
	}
	o, snapshot, err := e.observe(ctx, s, "created")
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e.createdSnapshot.Containers, snapshot.Containers) {
		return errors.New("replacement changed before Start")
	}
	if err = s.ValidateObservation(e.history, index, *e.created, "created", time.Now().UTC()); err != nil {
		return err
	}
	if err = s.ValidateObservation(e.history, index, o, "created", time.Now().UTC()); err != nil {
		return err
	}
	q := e.review.source.anchor.base.q
	out, err := q.docker(ctx, q.d.times.command, "container", "start", e.createdID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.createdID {
		return errors.New("replacement Start acknowledgement unresolved")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	e.startAcknowledged = true
	return nil
}
