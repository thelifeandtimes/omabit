//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type projectMemberStopExitEngine struct {
	source   *projectMemberStopEngine
	retained projectnetwork.MemberStop
}

// OpenProjectMemberStopExit opens a separate read-only driver for a retained
// dispatched Stop. It cannot re-signal, complete the old operation, or resume a
// pier. The caller journals this observation under its own operation identity.
func OpenProjectMemberStopExit(ctx context.Context, s ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	return openProjectMemberStopExit(ctx, s, defaults())
}
func openProjectMemberStopExit(ctx context.Context, s ProjectMemberStopSelection, d dependencies) (projectnetwork.MemberStopExitDriver, error) {
	retained := s.Stop
	if retained.Validate(s.Project.Plan) != nil || retained.Before == nil {
		return nil, errors.New("Stop exit observation requires retained dispatch and process evidence")
	}
	s.Stop.Phase, s.Stop.Before, s.Stop.After = "planned", nil, nil
	raw, err := openProjectMemberStop(ctx, s, d)
	if err != nil {
		return nil, err
	}
	return memberStopExitFromSource(s.Project.Plan, raw.(*projectMemberStopEngine), retained)
}
func memberStopExitFromSource(plan projectnetwork.Plan, source *projectMemberStopEngine, retained projectnetwork.MemberStop) (projectnetwork.MemberStopExitDriver, error) {
	// Reconstruct with the pure transitions to clone all nested binding/evidence
	// buffers. This is an in-memory copy only, not a replay or persisted transition.
	frozen, err := source.initial.Advance(plan, "planned", "stop-dispatched", retained.Before)
	if err != nil {
		return nil, err
	}
	if retained.Phase != "stop-dispatched" {
		frozen, err = frozen.Advance(plan, "stop-dispatched", "stop-signalled", nil)
		if err != nil {
			return nil, err
		}
	}
	if retained.Phase == "stopped" {
		frozen, err = frozen.Advance(plan, "stop-signalled", "stopped", retained.After)
		if err != nil {
			return nil, err
		}
	}
	if !reflect.DeepEqual(frozen, retained) {
		return nil, errors.New("retained Stop no longer matches recorded creation and readiness")
	}
	source.before = frozen.Before
	return &projectMemberStopExitEngine{source: source, retained: frozen}, nil
}
func (e *projectMemberStopExitEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.source.VerifyEngine(ctx, id)
}
func (e *projectMemberStopExitEngine) InspectExit(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	if ctx.Err() != nil || !reflect.DeepEqual(s, e.retained) {
		return projectnetwork.MemberStopObservation{}, errors.New("Stop exit observation selection changed")
	}
	o, _, err := e.source.observe(ctx, true)
	return o, err
}
