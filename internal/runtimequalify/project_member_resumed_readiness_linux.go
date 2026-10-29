//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type ProjectResumedReadinessSelection struct {
	Resume    ProjectMemberResumeSelection
	Completed projectnetwork.MemberResume
}
type projectResumedReadinessEngine struct {
	source         *projectMemberResumeEngine
	completed      projectnetwork.MemberResume
	binding        projectnetwork.RuntimeBinding
	ship, endpoint string
	inspectedAt    time.Time
}

// OpenResumedProjectReadiness reconstructs the exact replacement contract from
// immutable history. It only observes; it cannot create, start or complete a
// replacement. The caller holds the common project resource/update fence.
func OpenResumedProjectReadiness(ctx context.Context, s ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	return openResumedProjectReadiness(ctx, s, defaults())
}
func openResumedProjectReadiness(ctx context.Context, s ProjectResumedReadinessSelection, d dependencies) (projectnetwork.ResumedReadinessDriver, error) {
	project, _, err := memberResumeStopSelection(s.Resume)
	if err != nil {
		return nil, err
	}
	plan := project.Plan
	binding, err := projectnetwork.ResumedRuntimeBinding(plan, s.Completed)
	if err != nil {
		return nil, err
	}
	source, err := newProjectMemberResumeEngine(ctx, s.Resume, d)
	if err != nil {
		return nil, err
	}
	saved := s.Completed
	frozen := source.initial
	results := []projectnetwork.MemberResumeResult{{Before: saved.Before}, {ContainerID: saved.Container.ID}, {Observation: saved.Created}, {}, {Observation: saved.After}}
	for i, next := range []string{"create-dispatched", "created", "start-dispatched", "started", "running"} {
		frozen, err = frozen.Advance(plan, frozen.Phase, next, results[i])
		if err != nil {
			return nil, err
		}
	}
	if !reflect.DeepEqual(frozen, saved) {
		return nil, errors.New("resumed readiness lost completed contract binding")
	}
	source.createdID = frozen.Container.ID
	e := &projectResumedReadinessEngine{source: source, completed: frozen, binding: binding}
	for _, m := range plan.Members {
		if m.InstanceID == frozen.Retirement.Exit.Stop.InstanceID {
			e.ship = m.Ship
			e.endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	return e, nil
}
func (e *projectResumedReadinessEngine) VerifyEngine(ctx context.Context, id string) error {
	err := e.source.VerifyEngine(ctx, id)
	if err != nil {
		e.inspectedAt = time.Time{}
	}
	return err
}
func (e *projectResumedReadinessEngine) Inspect(ctx context.Context, s projectnetwork.MemberResume) (projectnetwork.MemberResumeObservation, error) {
	e.inspectedAt = time.Time{}
	fail := func(err error) (projectnetwork.MemberResumeObservation, error) {
		return projectnetwork.MemberResumeObservation{}, err
	}
	if ctx.Err() != nil || !reflect.DeepEqual(s, e.completed) {
		return fail(errors.New("resumed readiness selection changed"))
	}
	snapshot, err := e.source.snapshot(ctx, true, true)
	if err != nil {
		return fail(err)
	}
	started, err := time.Parse(time.RFC3339Nano, snapshot.Containers[s.Container.ID].State.StartedAt)
	if err != nil {
		return fail(err)
	}
	o := projectnetwork.MemberResumeObservation{Runtime: snapshot.Observation, Storage: e.source.contract.selection.Storage, StartedAt: started, CapturedAt: time.Now().UTC()}
	r := projectnetwork.ResumedMemberReadiness{InstanceID: s.Retirement.Exit.Stop.InstanceID, ResumeOperationID: s.OperationID, ContainerGeneration: s.PreviousGeneration + 1, Binding: e.binding, Storage: o.Storage, Ship: e.ship, Endpoint: e.endpoint, Before: o, After: o, CapturedAt: o.CapturedAt}
	if err = r.Validate(e.source.contract.selection.Project.Plan, e.completed); err != nil {
		return fail(err)
	}
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	// The second observation must preserve all process metadata, including peers
	// which were already stopped. The target start time also matches saved history.
	e.source.beforeSnapshot = snapshot
	e.inspectedAt = o.Runtime.ObservedAt
	return o, nil
}
func (e *projectResumedReadinessEngine) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if ctx.Err() != nil || endpoint != e.endpoint || e.inspectedAt.IsZero() || time.Since(e.inspectedAt) > 30*time.Second || time.Until(e.inspectedAt) > 5*time.Second {
		return "", errors.New("resumed readiness HTTP requires fresh exact publication evidence")
	}
	return e.source.source.source.anchor.base.q.d.hostIdentity(ctx, endpoint)
}
