//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectMemberRetirementSelection struct {
	Stop        ProjectMemberStopSelection
	Exit        projectnetwork.MemberStopExitEvidence
	OperationID string
}
type projectMemberRetirementEngine struct {
	source                  *projectMemberStopEngine
	initial                 projectnetwork.MemberRetirement
	before                  *projectnetwork.MemberStopObservation
	snapshot                projectMemberSnapshot
	attempted, acknowledged bool
}

func OpenProjectMemberRetirement(ctx context.Context, s ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	return openProjectMemberRetirement(ctx, s, defaults())
}
func openProjectMemberRetirement(ctx context.Context, s ProjectMemberRetirementSelection, d dependencies) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	initial, err := projectnetwork.NewMemberRetirement(s.Stop.Project.Plan, s.Exit, s.OperationID)
	if err != nil || !reflect.DeepEqual(s.Exit.Stop, s.Stop.Stop) {
		return projectnetwork.MemberRetirement{}, nil, errors.New("retirement requires its exact retained clean exit")
	}
	raw, err := openProjectMemberStopExit(ctx, s.Stop, d)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	// Build a second independent clone; caller edits cannot rewrite adapter intent.
	frozen, err := projectnetwork.NewMemberRetirement(s.Stop.Project.Plan, s.Exit, s.OperationID)
	if err != nil {
		return projectnetwork.MemberRetirement{}, nil, err
	}
	return initial, &projectMemberRetirementEngine{source: raw.(*projectMemberStopExitEngine).source, initial: frozen}, nil
}
func (e *projectMemberRetirementEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.source.VerifyEngine(ctx, id)
}
func (e *projectMemberRetirementEngine) selected(ctx context.Context, s projectnetwork.MemberRetirement, phase string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.Phase != phase || s.Validate(e.source.source.anchor.base.contract.Selection.Plan) != nil {
		return errors.New("retirement phase changed")
	}
	base := s
	base.Phase, base.Before, base.After = "planned", nil, nil
	if !reflect.DeepEqual(base, e.initial) || phase != "planned" && (e.before == nil || !reflect.DeepEqual(s.Before, e.before)) {
		return errors.New("retirement binding changed")
	}
	return nil
}
func (e *projectMemberRetirementEngine) storage(ctx context.Context) error {
	if err := e.source.storage(ctx); err != nil {
		return err
	}
	path, err := pierstore.Check(e.source.source.anchor.base.q.d.home, e.initial.Exit.Stop.Storage)
	if err != nil {
		return err
	}
	if pierstore.Busy(path) {
		return errors.New("retirement pier still has a Vere lock")
	}
	return pierstore.CheckStateFiles(ctx, path)
}
func (e *projectMemberRetirementEngine) InspectStopped(ctx context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberStopObservation, error) {
	if err := e.selected(ctx, s, "planned"); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	if e.before != nil || e.attempted {
		return projectnetwork.MemberStopObservation{}, errors.New("retirement observation cannot replace retained evidence")
	}
	if err := e.storage(ctx); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	o, snapshot, err := e.source.observe(ctx, true)
	if err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	frozen, err := s.Advance(e.source.source.anchor.base.contract.Selection.Plan, "planned", "remove-dispatched", &o, nil)
	if err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	if err = e.storage(ctx); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	if err = e.source.fresh(ctx, o); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	e.before, e.snapshot = frozen.Before, snapshot
	return o, nil
}
func (e *projectMemberRetirementEngine) RemoveStopped(ctx context.Context, s projectnetwork.MemberRetirement) error {
	if err := e.selected(ctx, s, "remove-dispatched"); err != nil {
		return err
	}
	if e.attempted {
		return errors.New("retirement removal cannot replay")
	}
	e.attempted = true
	if err := e.source.fresh(ctx, *e.before); err != nil {
		return err
	}
	if err := e.storage(ctx); err != nil {
		return err
	}
	o, snapshot, err := e.source.observe(ctx, true)
	if err != nil {
		return err
	}
	expected := *e.before
	expected.CapturedAt = o.CapturedAt
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	if !reflect.DeepEqual(expected, o) || !reflect.DeepEqual(e.snapshot.Containers, snapshot.Containers) {
		return errors.New("retirement process or peer metadata changed before removal")
	}
	if err = e.storage(ctx); err != nil {
		return err
	}
	if err = e.source.fresh(ctx, *e.before); err != nil {
		return err
	}
	if err = e.source.fresh(ctx, o); err != nil {
		return err
	}
	q := e.source.source.anchor.base.q
	// No --force, --volumes, link removal, signal, resource cleanup or pier write.
	out, err := q.docker(ctx, q.d.times.command, "container", "rm", e.source.target.id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != e.source.target.id {
		return errors.New("retirement acknowledgement is unresolved")
	}
	e.acknowledged = true
	return nil
}
func (e *projectMemberRetirementEngine) InspectAbsent(ctx context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberAbsenceObservation, error) {
	fail := func(err error) (projectnetwork.MemberAbsenceObservation, error) {
		return projectnetwork.MemberAbsenceObservation{}, err
	}
	if err := e.selected(ctx, s, "removed"); err != nil {
		return fail(err)
	}
	if !e.acknowledged {
		return fail(errors.New("retirement has no acknowledged removal from this dispatch"))
	}
	if err := e.storage(ctx); err != nil {
		return fail(err)
	}
	known := make([]projectObservedMember, 0, len(e.source.source.members)-1)
	expected := map[string]container{}
	for id, c := range e.snapshot.Containers {
		if id != e.source.target.id {
			expected[id] = c
		}
	}
	for _, m := range e.source.source.members {
		if m.id == e.source.target.id {
			continue
		}
		started, err := time.Parse(time.RFC3339Nano, expected[m.id].State.StartedAt)
		if err != nil {
			return fail(err)
		}
		m.processStartedAt = started
		m.allowExited = !*expected[m.id].State.Running
		known = append(known, m)
	}
	snapshot, err := e.source.inspectMembers(ctx, known, e.source.target.id)
	if err != nil {
		return fail(err)
	}
	if !reflect.DeepEqual(expected, snapshot.Containers) {
		return fail(errors.New("retirement changed project or peer metadata"))
	}
	if err = e.storage(ctx); err != nil {
		return fail(err)
	}
	after := projectnetwork.MemberAbsenceObservation{Runtime: snapshot.Observation, Storage: e.initial.Exit.Stop.Storage, CapturedAt: time.Now().UTC()}
	if _, err = s.Advance(e.source.source.anchor.base.contract.Selection.Plan, "removed", "absent", nil, &after); err != nil {
		return fail(err)
	}
	return after, ctx.Err()
}
