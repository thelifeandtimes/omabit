//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
)

// Startup resume never starts an old boot container: its command may still have
// creation flags. Remove only the authenticated, terminal runtime and use
// the normal initialized-pier replacement path under the caller's durable job.
func applyStartupResume(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || !StartupResumeCandidate(state) {
		return state, unknown, errors.New("startup resume requires a resolved initialized runtime")
	}
	state, observed, err := observeManaged(ctx, state, d)
	if err != nil {
		return state, unknown, err
	}
	if !StartupResumeReady(state, observed) {
		return state, unknown, errors.New("startup resume requires an authorized, verified terminal container")
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err := q.verifyEngine(ctx); err != nil {
		return state, compatibilityObservation(state), err
	}
	if err := q.admitStart(q.r.Directory, true, "startup resume"); err != nil {
		return state, unknown, err
	}
	for i, c := range q.r.Containers {
		if c.Removed {
			continue
		}
		actual, err := q.inspect(ctx, i)
		if err != nil {
			return state, unknown, err
		}
		if err := q.containment(actual, c.Kind); err != nil {
			return state, unknown, err
		}
		unclean := state.StartupResume != nil && state.StartupResume.AllowUnclean && state.StartupResume.ContainerID == c.ID
		if active(actual) || actual.State.Dead || (!unclean && (*actual.State.OOMKilled || *actual.State.ExitCode != 0)) {
			return state, unknown, errors.New("startup container state changed before removal")
		}
		if err := q.remove(ctx, i); err != nil {
			state.Runtime = q.r
			return state, unknown, err
		}
	}
	state.Runtime = q.r
	state.StartupResume = nil // the authorization applies only to the old container
	return applyManaged(ctx, "start", state, persist, d)
}
