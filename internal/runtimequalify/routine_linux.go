//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"time"
)

// applyRoutine is one durable operation, holding the normal per-instance
// exclusion and host work gate for the entire sequence. It never resumes after
// a failed step and never bootstraps an uninitialized instance.
func applyRoutine(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	p := state.Routine
	if persist == nil || p == nil || !state.MaintenanceQualified() || state.UncertainProvisioning() || state.UncertainMaintenance() || p.StartedAt != nil || p.FinishedAt != nil || (p.Action != "pack" && p.Action != "meld") || action != "routine-"+p.Action || p.Resume && !p.StopRunning {
		return state, unknown, errors.New("routine requires a fresh authorized intent and initialized qualified pier")
	}
	if time.Now().After(p.Deadline) {
		return state, unknown, errors.New("routine dispatch window expired; no work dispatched")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	state, observed, err := observeManaged(ctx, state, d)
	if err != nil || observed.RecoveryRequired || (observed.State != "running" && observed.State != "stopped") || p.WasRunning != (observed.State == "running") || p.WasRunning && !p.StopRunning {
		return state, unknown, errors.New("routine runtime state changed before dispatch")
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	mem, disk, err := q.d.admission(q.base)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return state, observed, errors.New("routine requires 5 GiB available RAM and 8 GiB disk before any stop")
	}
	if time.Now().After(p.Deadline) {
		return state, observed, errors.New("routine dispatch window expired during preflight; no work dispatched")
	}
	now := time.Now().UTC()
	p.StartedAt = &now
	steps := []string{}
	if p.WasRunning {
		steps = append(steps, "stop")
	}
	steps = append(steps, p.Action)
	if p.WasRunning && p.Resume {
		steps = append(steps, "start")
	}
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return state, unknown, err
		}
		p.Stage = step
		if err := persist(state); err != nil {
			return state, unknown, err
		}
		state, observed, err = applyManaged(ctx, step, state, persist, d)
		if err != nil {
			return state, observed, err
		}
	}
	finished := time.Now().UTC()
	p.FinishedAt = &finished
	p.Succeeded = true
	p.Stage = "complete"
	if err := persist(state); err != nil {
		return state, unknown, err
	}
	return state, observed, nil
}
