//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestManagedOfflineMaintenanceAndResume(t *testing.T) {
	for _, kind := range []string{"pack", "meld"} {
		t.Run(kind, func(t *testing.T) { testManagedOfflineMaintenanceAndResume(t, kind) })
	}
}

func testManagedOfflineMaintenanceAndResume(t *testing.T, kind string) {
	f, state, persist := managedFixture(t)
	ctx := context.Background()
	state, _, err := applyManaged(ctx, "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := applyManaged(ctx, kind, state, persist, f.d); err == nil {
		t.Fatal("maintenance admitted on a running pier")
	}
	state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	var recorded bool
	save := func(s ManagedState) error {
		if s.MaintenanceUncertain {
			recorded = true
		}
		return persist(s)
	}
	state, obs, err := applyManaged(ctx, kind, state, save, f.d)
	if err != nil || !recorded || state.MaintenanceUncertain || state.Maintenance == nil || state.Maintenance.Kind != kind || state.Maintenance.FinishedAt == nil || state.Maintenance.MemAvailableBytes < 5<<30 || state.Maintenance.DiskAvailableBytes < 8<<30 || obs.State != "stopped" || len(f.containers) != 0 {
		t.Fatalf("maintenance result: %+v %+v %v", state, obs, err)
	}
	if f.downloads != 1 {
		t.Fatal("maintenance fetched a bootstrap pill")
	}
	state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || !state.Initialized {
		t.Fatalf("post-maintenance resume: %+v %v", obs, err)
	}
}

func TestInterruptedMaintenanceStaysUncertainAfterObservationAndStop(t *testing.T) {
	for _, kind := range []string{"pack", "meld"} {
		t.Run(kind, func(t *testing.T) { testInterruptedMaintenance(t, kind) })
	}
}

func testInterruptedMaintenance(t *testing.T, kind string) {
	f, state, persist := managedFixture(t)
	ctx := context.Background()
	state, _, err := applyManaged(ctx, "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	execute := f.d.execute
	interrupted := false
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if strings.Contains(strings.Join(args, " "), "container wait ") {
			interrupted = true
			return "", errors.New("fixture interruption")
		}
		return execute(ctx, args, env)
	}
	state, _, err = applyManaged(ctx, kind, state, persist, f.d)
	if err == nil || !state.MaintenanceUncertain || !interrupted {
		t.Fatalf("uncertain maintenance forgotten: %v", err)
	}
	state, obs, err := observeManaged(ctx, state, f.d)
	if err != nil || !obs.RecoveryRequired {
		t.Fatal("observation erased uncertainty")
	}
	f.d.execute = execute
	state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil || !obs.RecoveryRequired || !state.MaintenanceUncertain {
		t.Fatalf("stop erased uncertainty: %+v %v", obs, err)
	}
	before := len(f.calls)
	if _, _, err := applyManaged(ctx, "start", state, persist, f.d); err == nil || len(f.calls) != before {
		t.Fatal("uncertain maintenance automatically resumed")
	}
	for _, next := range []string{"pack", "meld"} {
		if _, _, err := applyManaged(ctx, next, state, persist, f.d); err == nil {
			t.Fatal("uncertain maintenance retried or replaced")
		}
	}
}
