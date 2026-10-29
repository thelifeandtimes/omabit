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

// ProjectMemberStopSelection comes from validated journals under the common
// resource/update fence. Earlier peer Stops supply exact exited-process identity.
// No path, image, command or arbitrary container selection is supplied by clients.
type ProjectMemberStopSelection struct {
	Project      ProjectPreparationSelection
	Launches     []projectnetwork.MemberLaunch
	Readiness    projectnetwork.InitialMemberReadiness
	Stop         projectnetwork.MemberStop
	StoppedPeers []projectnetwork.MemberStop
}
type projectMemberExit struct {
	started, finished time.Time
	code              int
	oom               bool
	storage           pierstore.Location
}
type projectMemberStopEngine struct {
	recovery                            *projectAnchorRecoveryReviewEngine
	source                              *projectInitialReadinessEngine
	initial                             projectnetwork.MemberStop
	target                              projectObservedMember
	exits                               map[string]projectMemberExit
	before                              *projectnetwork.MemberStopObservation
	snapshot                            projectMemberSnapshot
	signalAttempted, signalAcknowledged bool
}

// OpenProjectMemberStop reopens the exact original contracts without changing
// any resource. The only mutation later exposed by this driver is a single TERM
// signal to the recorded full container ID. It never forces exit or removes data.
func OpenProjectMemberStop(ctx context.Context, s ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	return openProjectMemberStop(ctx, s, defaults())
}
func openProjectMemberStop(ctx context.Context, s ProjectMemberStopSelection, d dependencies) (projectnetwork.MemberStopDriver, error) {
	expected, err := projectnetwork.NewInitialMemberStop(s.Project.Plan, s.Launches, s.Readiness, s.Stop.ReadinessOperationID, s.Stop.OperationID)
	if err != nil || !reflect.DeepEqual(s.Stop, expected) {
		return nil, errors.New("member Stop requires a fresh original readiness-bound intent")
	}
	e := &projectMemberStopEngine{initial: expected, exits: map[string]projectMemberExit{}}
	for _, peer := range s.StoppedPeers {
		if peer.Validate(s.Project.Plan) != nil || peer.Phase != "stopped" || peer.InstanceID == expected.InstanceID || peer.OperationID == expected.OperationID || !reflect.DeepEqual(peer.Binding, expected.Binding) {
			return nil, errors.New("member Stop peer exit binding changed")
		}
		if _, exists := e.exits[peer.InstanceID]; exists {
			return nil, errors.New("duplicate member Stop peer exit")
		}
		matched := false
		for _, launch := range s.Launches {
			if launch.Storage.InstanceID == peer.InstanceID && launch.OperationID == peer.LaunchOperationID && launch.Storage.Location.Path == peer.Storage.Path && launch.Storage.Location.MountIdentity == peer.Storage.MountIdentity {
				matched = true
			}
		}
		if !matched {
			return nil, errors.New("member Stop peer exit lost original launch")
		}
		o := peer.After
		e.exits[peer.InstanceID] = projectMemberExit{o.StartedAt, o.FinishedAt, o.ExitCode, o.OOMKilled, peer.Storage}
	}
	raw, err := openInitialProjectReadiness(ctx, ProjectInitialReadinessSelection{Project: s.Project, Launches: s.Launches}, d)
	if err != nil {
		return nil, err
	}
	e.source = raw.(*projectInitialReadinessEngine)
	for _, m := range e.source.members {
		if m.contract.selection.InstanceID == expected.InstanceID {
			e.target = m
		}
	}
	if e.target.id == "" {
		return nil, errors.New("member Stop target is missing")
	}
	if err = e.storage(ctx); err != nil {
		return nil, err
	}
	return e, nil
}
func (e *projectMemberStopEngine) VerifyEngine(ctx context.Context, id string) error {
	if id != e.initial.Binding.EngineID {
		return errors.New("member Stop engine binding changed")
	}
	return e.source.VerifyEngine(ctx, id)
}
func (e *projectMemberStopEngine) selected(ctx context.Context, s projectnetwork.MemberStop, phase string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Validate(e.source.anchor.base.contract.Selection.Plan) != nil || s.Phase != phase {
		return errors.New("member Stop phase changed")
	}
	base := s
	base.Phase, base.Before, base.After = "planned", nil, nil
	if !reflect.DeepEqual(base, e.initial) || phase != "planned" && (e.before == nil || !reflect.DeepEqual(s.Before, e.before)) {
		return errors.New("member Stop binding or observed process changed")
	}
	return nil
}
func (e *projectMemberStopEngine) storage(ctx context.Context) error {
	if err := e.source.parents(ctx); err != nil {
		return err
	}
	check := func(loc pierstore.Location) error {
		path, err := pierstore.Absolute(e.source.anchor.base.q.d.home, loc.Path)
		if err != nil {
			return err
		}
		actual, err := pierstore.Check(e.source.anchor.base.q.d.home, loc)
		if err != nil || actual != path {
			return errors.New("member Stop initialized storage identity changed")
		}
		parent, err := pierstore.CheckMount(e.source.anchor.base.q.d.home, loc)
		if err != nil || parent != filepath.Dir(path) {
			return errors.New("member Stop parent identity changed")
		}
		return ctx.Err()
	}
	if err := check(e.initial.Storage); err != nil {
		return err
	}
	for _, peer := range e.exits {
		if err := check(peer.storage); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (e *projectMemberStopEngine) observe(ctx context.Context, exited bool) (projectnetwork.MemberStopObservation, projectMemberSnapshot, error) {
	fail := func(err error) (projectnetwork.MemberStopObservation, projectMemberSnapshot, error) {
		return projectnetwork.MemberStopObservation{}, projectMemberSnapshot{}, err
	}
	if err := e.storage(ctx); err != nil {
		return fail(err)
	}
	known := slices.Clone(e.source.members)
	for i := range known {
		m := &known[i]
		if peer, ok := e.exits[m.contract.selection.InstanceID]; ok {
			m.processStartedAt, m.allowExited = peer.started, true
		}
		if m.id == e.target.id && e.before != nil {
			m.processStartedAt, m.allowExited = e.before.StartedAt, exited
		}
	}
	snapshot, err := e.inspectMembers(ctx, known)
	if err != nil {
		return fail(err)
	}
	if err = e.storage(ctx); err != nil {
		return fail(err)
	}
	for _, m := range known {
		if peer, ok := e.exits[m.contract.selection.InstanceID]; ok {
			v := snapshot.Containers[m.id]
			start, se := time.Parse(time.RFC3339Nano, v.State.StartedAt)
			finish, fe := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
			if se != nil || fe != nil || *v.State.Running || !start.Equal(peer.started) || !finish.Equal(peer.finished) || *v.State.ExitCode != peer.code || *v.State.OOMKilled != peer.oom {
				return fail(errors.New("previously stopped peer process changed"))
			}
		}
	}
	v := snapshot.Containers[e.target.id]
	start, se := time.Parse(time.RFC3339Nano, v.State.StartedAt)
	finish, fe := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
	if se != nil || fe != nil || *v.State.Running == exited {
		return fail(errors.New("member Stop target process state is unresolved"))
	}
	o := projectnetwork.MemberStopObservation{Runtime: snapshot.Observation, Storage: e.initial.Storage, StartedAt: start, FinishedAt: finish, ExitCode: *v.State.ExitCode, OOMKilled: *v.State.OOMKilled, CapturedAt: time.Now().UTC()}
	if err = e.fresh(ctx, o); err != nil {
		return fail(err)
	}
	return o, snapshot, nil
}
func (e *projectMemberStopEngine) fresh(ctx context.Context, o projectnetwork.MemberStopObservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now()
	if o.CapturedAt.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second {
		return errors.New("member Stop observation expired")
	}
	a, err := projectnetwork.AssessRuntime(e.source.anchor.base.contract.Selection.Plan, e.initial.Binding, o.Runtime, now)
	if err != nil || a.RecoveryRequired || a.State != "ready" && a.State != "member-stopped" {
		return projectnetwork.ErrRuntimeObservation
	}
	return nil
}
func (e *projectMemberStopEngine) Inspect(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	if err := e.selected(ctx, s, "planned"); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	if e.before != nil || e.signalAttempted {
		return projectnetwork.MemberStopObservation{}, errors.New("member Stop observation cannot replace an accepted process binding")
	}
	o, snapshot, err := e.observe(ctx, false)
	if err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	frozen, err := s.Advance(e.source.anchor.base.contract.Selection.Plan, "planned", "stop-dispatched", &o)
	if err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	// Advance clones the externally returned observation before retaining it.
	e.before, e.snapshot = frozen.Before, snapshot
	return o, nil
}
func (e *projectMemberStopEngine) SignalStop(ctx context.Context, s projectnetwork.MemberStop) error {
	if err := e.selected(ctx, s, "stop-dispatched"); err != nil {
		return err
	}
	if e.signalAttempted {
		return errors.New("member Stop signal cannot replay")
	}
	e.signalAttempted = true
	if err := e.fresh(ctx, *e.before); err != nil {
		return err
	}
	o, snapshot, err := e.observe(ctx, false)
	if err != nil {
		return err
	}
	expected := e.before.Runtime
	expected.ObservedAt = o.Runtime.ObservedAt
	if !reflect.DeepEqual(expected, o.Runtime) || !reflect.DeepEqual(e.snapshot.Containers, snapshot.Containers) || o.StartedAt != e.before.StartedAt {
		return errors.New("member Stop ownership changed before signal")
	}
	if err = e.fresh(ctx, *e.before); err != nil {
		return err
	}
	if err = e.fresh(ctx, o); err != nil {
		return err
	}
	q := e.source.anchor.base.q
	// The explicit signal is essential: Docker's default kill signal is SIGKILL.
	// A TERM request has no timeout escalation; the separate bounded wait is read-only.
	out, err := q.docker(ctx, q.d.times.command, "container", "kill", "--signal", "SIGTERM", e.target.id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.target.id {
		return errors.New("member Stop signal acknowledgement is unresolved")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	e.signalAcknowledged = true
	return nil
}
func (e *projectMemberStopEngine) AwaitStopped(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	refuse := func(err error) (projectnetwork.MemberStopObservation, error) {
		return projectnetwork.MemberStopObservation{}, err
	}
	if err := e.selected(ctx, s, "stop-signalled"); err != nil {
		return refuse(err)
	}
	if !e.signalAcknowledged {
		return refuse(errors.New("member Stop has no acknowledged signal from this dispatch"))
	}
	q := e.source.anchor.base.q
	ctx, cancel := context.WithTimeout(ctx, q.d.times.stop)
	defer cancel()
	for {
		rows, err := e.source.anchor.inspectProjectContainers(ctx, []string{e.target.id})
		if err != nil {
			return refuse(err)
		}
		v := rows[e.target.id]
		if !*v.State.Running {
			if err = e.target.contract.validateExitedContainer(v, e.target.id, e.before.StartedAt); err != nil {
				return refuse(err)
			}
			o, snapshot, err := e.observe(ctx, true)
			if err != nil {
				return refuse(err)
			}
			// Every peer/anchor inspection remains exact; only target process exit fields
			// may differ. The model separately validates all exit times and namespace state.
			expected := make(map[string]container, len(e.snapshot.Containers))
			for id, before := range e.snapshot.Containers {
				expected[id] = before
			}
			target := expected[e.target.id]
			target.State = snapshot.Containers[e.target.id].State
			expected[e.target.id] = target
			if !reflect.DeepEqual(expected, snapshot.Containers) {
				return refuse(errors.New("project process metadata changed during member Stop"))
			}
			if _, err = s.Advance(e.source.anchor.base.contract.Selection.Plan, "stop-signalled", "stopped", &o); err != nil {
				return refuse(err)
			}
			return o, nil
		}
		if err = e.target.contract.validateContainer(v, e.target.id, true); err != nil {
			return refuse(err)
		}
		started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
		if err != nil || !started.Equal(e.before.StartedAt) {
			return refuse(errors.New("member process restarted while awaiting exit"))
		}
		timer := time.NewTimer(q.d.times.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return refuse(ctx.Err())
		case <-timer.C:
		}
	}
}
