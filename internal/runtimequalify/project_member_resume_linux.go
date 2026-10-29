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

// Resume uses exactly one Stop source. Initial Stop retains generation-one launch
// history; ResumedStop or CurrentStop supplies the corresponding causal readiness.
// Neither path adopts a container or rewrites an earlier lifecycle receipt.
type ProjectMemberResumeSelection struct {
	Stop        ProjectMemberStopSelection
	ResumedStop *ProjectResumedMemberStopSelection
	CurrentStop *ProjectCurrentMemberStopSelection
	Retirement  projectnetwork.MemberRetirement
	OperationID string
}
type projectMemberResumeEngine struct {
	source                                             *projectMemberStopEngine
	contract                                           projectMemberContract
	initial                                            projectnetwork.MemberResume
	peers                                              []projectObservedMember
	before                                             *projectnetwork.MemberAbsenceObservation
	created                                            *projectnetwork.MemberResumeObservation
	beforeSnapshot, createdSnapshot                    projectMemberSnapshot
	createdID                                          string
	createAttempted, startAttempted, startAcknowledged bool
}

func OpenProjectMemberResume(ctx context.Context, s ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	return openProjectMemberResume(ctx, s, defaults())
}
func openProjectMemberResume(ctx context.Context, s ProjectMemberResumeSelection, d dependencies) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	e, err := newProjectMemberResumeEngine(ctx, s, d)
	if err != nil {
		return projectnetwork.MemberResume{}, nil, err
	}
	if err = e.storage(ctx, false); err != nil {
		return projectnetwork.MemberResume{}, nil, err
	}
	// Return an independent selection; the engine retains its own immutable copy.
	initial, err := projectnetwork.NewMemberResume(e.contract.selection.Project.Plan, e.initial.Retirement, e.initial.PreviousGeneration, s.OperationID, e.contract.digest)
	if err != nil {
		return projectnetwork.MemberResume{}, nil, err
	}
	return initial, e, nil
}
func newProjectMemberResumeEngine(ctx context.Context, s ProjectMemberResumeSelection, d dependencies) (*projectMemberResumeEngine, error) {
	fail := func(err error) (*projectMemberResumeEngine, error) {
		return nil, err
	}
	project, stop, err := memberResumeStopSelection(s)
	if err != nil {
		return fail(err)
	}
	if s.Retirement.Validate(project.Plan) != nil || s.Retirement.Phase != "absent" || !reflect.DeepEqual(s.Retirement.Exit.Stop, stop) {
		return fail(errors.New("resume requires its exact completed retirement"))
	}
	var raw projectnetwork.MemberStopExitDriver
	if s.CurrentStop != nil {
		raw, err = openCurrentProjectMemberStopExit(ctx, *s.CurrentStop, d)
	} else if s.ResumedStop == nil {
		raw, err = openProjectMemberStopExit(ctx, s.Stop, d)
	} else {
		raw, err = openResumedProjectMemberStopExit(ctx, *s.ResumedStop, d)
	}
	if err != nil {
		return fail(err)
	}
	source := raw.(*projectMemberStopExitEngine).source
	previous := source.target.contract
	if s.CurrentStop == nil && (s.ResumedStop == nil && (previous.selection.ContainerGeneration != 1 || previous.selection.Mode != "boot") || s.ResumedStop != nil && (previous.selection.ContainerGeneration < 2 || previous.selection.Mode != "resume")) {
		return fail(errors.New("resume source does not match its recorded creation mode"))
	}
	selected := previous.selection
	selected.Mode = "resume"
	selected.ContainerGeneration = previous.selection.ContainerGeneration + 1
	selected.OperationID = s.OperationID
	selected.Storage = s.Retirement.Exit.Stop.Storage
	var c projectMemberContract
	if s.CurrentStop != nil && s.CurrentStop.ReadinessSource.RecoveryHistory != nil {
		c, err = newRecoveredResumeProjectMemberContract(project, *s.CurrentStop.ReadinessSource.RecoveryHistory, s.Retirement, s.OperationID, d.home, previous.mountSource, previous.imageID)
	} else {
		c, err = newProjectMemberContract(selected, d.home, previous.mountSource, previous.imageID)
	}
	if err != nil {
		return fail(err)
	}
	frozen, err := projectnetwork.NewMemberResume(project.Plan, s.Retirement, previous.selection.ContainerGeneration, s.OperationID, c.digest)
	if err != nil {
		return fail(err)
	}
	e := &projectMemberResumeEngine{source: source, contract: c, initial: frozen}
	for _, m := range source.source.members {
		if m.id == source.target.id {
			continue
		}
		if exit, ok := source.exits[m.contract.selection.InstanceID]; ok {
			m.processStartedAt, m.allowExited = exit.started, true
		}
		e.peers = append(e.peers, m)
	}
	return e, nil
}
func memberResumeStopSelection(s ProjectMemberResumeSelection) (ProjectPreparationSelection, projectnetwork.MemberStop, error) {
	if s.CurrentStop != nil {
		if s.ResumedStop != nil || !reflect.DeepEqual(s.Stop, ProjectMemberStopSelection{}) {
			return ProjectPreparationSelection{}, projectnetwork.MemberStop{}, errors.New("resume has conflicting Stop sources")
		}
		return s.CurrentStop.ReadinessSource.Project, s.CurrentStop.Stop, nil
	}

	if s.ResumedStop != nil {
		if !reflect.DeepEqual(s.Stop, ProjectMemberStopSelection{}) {
			return ProjectPreparationSelection{}, projectnetwork.MemberStop{}, errors.New("resume has conflicting Stop sources")
		}
		return s.ResumedStop.Project, s.ResumedStop.Stop, nil
	}
	return s.Stop.Project, s.Stop.Stop, nil
}

func (e *projectMemberResumeEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.source.VerifyEngine(ctx, id)
}
func (e *projectMemberResumeEngine) selected(ctx context.Context, s projectnetwork.MemberResume, phase string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Phase != phase || s.Validate(e.contract.selection.Project.Plan) != nil {
		return errors.New("resume phase changed")
	}
	baseline := s
	baseline.Phase = "planned"
	baseline.Container.ID = ""
	baseline.Before, baseline.Created, baseline.After = nil, nil, nil
	if !reflect.DeepEqual(baseline, e.initial) || s.Container.ID != "" && s.Container.ID != e.createdID || phase != "planned" && (e.before == nil || !reflect.DeepEqual(s.Before, e.before)) || s.Created != nil && (e.created == nil || !reflect.DeepEqual(s.Created, e.created)) {
		return errors.New("resume evidence or dispatch binding changed")
	}
	return nil
}
func (e *projectMemberResumeEngine) storage(ctx context.Context, running bool) error {
	if err := e.source.storage(ctx); err != nil {
		return err
	}
	if !running {
		return e.contract.checkStorageBeforeCreate(ctx)
	}
	return ctx.Err()
}
func (e *projectMemberResumeEngine) fresh(ctx context.Context, captured, observed time.Time) error {
	now := time.Now()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if captured.IsZero() || observed.IsZero() || now.Sub(captured) > 30*time.Second || captured.Sub(now) > 5*time.Second || now.Sub(observed) > 30*time.Second || observed.Sub(now) > 5*time.Second {
		return errors.New("resume requires fresh complete ownership evidence")
	}
	return nil
}
func (e *projectMemberResumeEngine) snapshot(ctx context.Context, present, running bool) (projectMemberSnapshot, error) {
	fail := func(err error) (projectMemberSnapshot, error) { return projectMemberSnapshot{}, err }
	if err := e.storage(ctx, running); err != nil {
		return fail(err)
	}
	known := slices.Clone(e.peers)
	for i, m := range known {
		if old, ok := e.beforeSnapshot.Containers[m.id]; ok {
			started, err := time.Parse(time.RFC3339Nano, old.State.StartedAt)
			if err != nil {
				return fail(err)
			}
			known[i].processStartedAt = started
		}
	}
	if present {
		known = append(known, projectObservedMember{contract: e.contract, id: e.createdID})
	}
	snapshot, err := e.source.inspectMembers(ctx, known, e.source.target.id)
	if err != nil {
		return fail(err)
	}
	if err = e.storage(ctx, running); err != nil {
		return fail(err)
	}
	wantNamespace := e.initial.Retirement.Exit.Stop.Binding.Namespace
	if snapshot.Observation.Anchor.Namespace != wantNamespace {
		return fail(errors.New("resume anchor namespace changed before dispatch"))
	}
	for _, member := range snapshot.Observation.Members {
		if member.Running && member.Namespace != wantNamespace {
			return fail(errors.New("resume member namespace changed before dispatch"))
		}
	}
	for _, m := range e.peers {
		c := snapshot.Containers[m.id]
		if exit, ok := e.source.exits[m.contract.selection.InstanceID]; ok {
			start, se := time.Parse(time.RFC3339Nano, c.State.StartedAt)
			finish, fe := time.Parse(time.RFC3339Nano, c.State.FinishedAt)
			if se != nil || fe != nil || *c.State.Running || !start.Equal(exit.started) || !finish.Equal(exit.finished) || *c.State.ExitCode != exit.code || *c.State.OOMKilled != exit.oom {
				return fail(errors.New("resume stopped-peer process changed"))
			}
		}
	}
	if e.beforeSnapshot.Containers != nil {
		for id, c := range e.beforeSnapshot.Containers {
			if !reflect.DeepEqual(c, snapshot.Containers[id]) {
				return fail(errors.New("resume project or peer metadata changed"))
			}
		}
	}
	if present {
		c := snapshot.Containers[e.createdID]
		if *c.State.Running != running {
			return fail(errors.New("resume target process does not match dispatch phase"))
		}
	}
	return snapshot, nil
}
func (e *projectMemberResumeEngine) InspectAbsent(ctx context.Context, s projectnetwork.MemberResume) (projectnetwork.MemberAbsenceObservation, error) {
	fail := func(err error) (projectnetwork.MemberAbsenceObservation, error) {
		return projectnetwork.MemberAbsenceObservation{}, err
	}
	if err := e.selected(ctx, s, "planned"); err != nil {
		return fail(err)
	}
	if e.before != nil || e.createAttempted {
		return fail(errors.New("resume absence cannot replace retained evidence"))
	}
	if err := e.source.source.anchor.rejectConflictingProjectMounts(ctx, e.contract, ""); err != nil {
		return fail(err)
	}
	snapshot, err := e.snapshot(ctx, false, false)
	if err != nil {
		return fail(err)
	}
	o := projectnetwork.MemberAbsenceObservation{Runtime: snapshot.Observation, Storage: e.contract.selection.Storage, CapturedAt: time.Now().UTC()}
	next, err := s.Advance(e.contract.selection.Project.Plan, "planned", "create-dispatched", projectnetwork.MemberResumeResult{Before: &o})
	if err != nil {
		return fail(err)
	}
	if err = e.fresh(ctx, o.CapturedAt, o.Runtime.ObservedAt); err != nil {
		return fail(err)
	}
	e.before, e.beforeSnapshot = next.Before, snapshot
	return o, nil
}
func (e *projectMemberResumeEngine) Create(ctx context.Context, s projectnetwork.MemberResume) (string, error) {
	if err := e.selected(ctx, s, "create-dispatched"); err != nil {
		return "", err
	}
	if e.createAttempted {
		return "", errors.New("resume Create cannot replay")
	}
	e.createAttempted = true
	if err := e.fresh(ctx, e.before.CapturedAt, e.before.Runtime.ObservedAt); err != nil {
		return "", err
	}
	if err := e.source.source.anchor.rejectConflictingProjectMounts(ctx, e.contract, ""); err != nil {
		return "", err
	}
	snapshot, err := e.snapshot(ctx, false, false)
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(e.beforeSnapshot.Containers, snapshot.Containers) {
		return "", errors.New("resume absence changed before Create")
	}
	if err = e.storage(ctx, false); err != nil {
		return "", err
	}
	if err = e.fresh(ctx, e.before.CapturedAt, e.before.Runtime.ObservedAt); err != nil {
		return "", err
	}
	if err = e.fresh(ctx, time.Now(), snapshot.Observation.ObservedAt); err != nil {
		return "", err
	}
	q := e.source.source.anchor.base.q
	if err := q.admitStart(e.contract.mountSource, true, "existing project ship create"); err != nil {
		return "", err
	}
	out, err := q.dockerCreateID(ctx, q.d.times.command, e.contract.createArgs()...)
	id := strings.TrimSpace(out)
	if !containerIDPattern.MatchString(id) {
		return "", errors.New("resume Create returned no exact identity")
	}
	e.createdID = id
	return id, err
}
func (e *projectMemberResumeEngine) Inspect(ctx context.Context, s projectnetwork.MemberResume) (projectnetwork.MemberResumeObservation, error) {
	fail := func(err error) (projectnetwork.MemberResumeObservation, error) {
		return projectnetwork.MemberResumeObservation{}, err
	}
	if s.Phase != "created" && s.Phase != "started" {
		return fail(errors.New("unsupported resume observation phase"))
	}
	if err := e.selected(ctx, s, s.Phase); err != nil {
		return fail(err)
	}
	running := s.Phase == "started"
	if running && !e.startAcknowledged {
		return fail(errors.New("resume Start acknowledgement is missing"))
	}
	if !running {
		if e.created != nil {
			return fail(errors.New("resume created observation cannot replay"))
		}
		if err := e.source.source.anchor.rejectConflictingProjectMounts(ctx, e.contract, e.createdID); err != nil {
			return fail(err)
		}
	}
	snapshot, err := e.snapshot(ctx, true, running)
	if err != nil {
		return fail(err)
	}
	started, err := time.Parse(time.RFC3339Nano, snapshot.Containers[e.createdID].State.StartedAt)
	if err != nil {
		return fail(err)
	}
	o := projectnetwork.MemberResumeObservation{Runtime: snapshot.Observation, Storage: e.contract.selection.Storage, StartedAt: started, CapturedAt: time.Now().UTC()}
	phase := "start-dispatched"
	if running {
		phase = "running"
	}
	next, err := s.Advance(e.contract.selection.Project.Plan, s.Phase, phase, projectnetwork.MemberResumeResult{Observation: &o})
	if err != nil {
		return fail(err)
	}
	if err = e.fresh(ctx, o.CapturedAt, o.Runtime.ObservedAt); err != nil {
		return fail(err)
	}
	if !running {
		e.created, e.createdSnapshot = next.Created, snapshot
	}
	return o, nil
}
func (e *projectMemberResumeEngine) Start(ctx context.Context, s projectnetwork.MemberResume) error {
	if err := e.selected(ctx, s, "start-dispatched"); err != nil {
		return err
	}
	if e.startAttempted {
		return errors.New("resume Start cannot replay")
	}
	e.startAttempted = true
	if err := e.fresh(ctx, e.created.CapturedAt, e.created.Runtime.ObservedAt); err != nil {
		return err
	}
	if err := e.source.source.anchor.rejectConflictingProjectMounts(ctx, e.contract, e.createdID); err != nil {
		return err
	}
	snapshot, err := e.snapshot(ctx, true, false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e.createdSnapshot.Containers, snapshot.Containers) {
		return errors.New("resume created process changed before Start")
	}
	if err = e.storage(ctx, false); err != nil {
		return err
	}
	if err = e.fresh(ctx, e.created.CapturedAt, e.created.Runtime.ObservedAt); err != nil {
		return err
	}
	if err = e.fresh(ctx, time.Now(), snapshot.Observation.ObservedAt); err != nil {
		return err
	}
	q := e.source.source.anchor.base.q
	if err := q.admitStart(e.contract.mountSource, true, "existing project ship start"); err != nil {
		return err
	}
	out, err := q.docker(ctx, q.d.times.command, "container", "start", e.createdID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.createdID {
		return errors.New("resume Start acknowledgement is unresolved")
	}
	e.startAcknowledged = true
	return nil
}
