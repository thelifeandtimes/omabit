//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func rc11MaintenanceReport(t *testing.T, kind string) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	fixture := stoppedManagedFixture
	if strings.HasPrefix(kind, "comet-") {
		fixture = stoppedManagedCometEpochFixture
	}
	f, state, persist := fixture(t) // real driver Start and Stop, injected engine
	if kind == "roll" {
		writePierEpochPath(t, managedPier(f, state), "0i0", false)
	}
	var durable []byte
	_, _, err := applyManaged(context.Background(), strings.TrimPrefix(kind, "comet-"), state, func(s ManagedState) error {
		if err := persist(s); err != nil {
			return err
		}
		if s.MaintenanceUncertain && s.Maintenance != nil {
			for _, c := range s.Runtime.Containers {
				if c.Kind != "probe" || !c.Removed || c.CreateUncertain {
					t.Fatal("report boundary already had utility work", c)
				}
			}
			durable = rc11Save(t, s)
			return errors.New("crash after report publication before utility intent")
		}
		return nil
	}, f.d)
	if err == nil || len(durable) == 0 || len(f.containers) != 0 {
		t.Fatal("report-only interruption was not reached", err)
	}
	return f, rc11Reload(t, durable), persist
}

func TestRC11ReportOnlyMaintenanceResolvesAcrossStopCallbackCrashes(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll", "comet-pack", "comet-meld", "comet-roll"} {
		for _, boundary := range []string{"none", "stop-intent", "before-resolution", "after-resolution"} {
			t.Run(kind+"/"+boundary, func(t *testing.T) {
				f, state, persist := rc11MaintenanceReport(t, kind)
				before := len(f.calls)
				if !managedMaintenanceReportPending(state, state.Runtime) {
					t.Fatal("report did not retain its explicit non-dispatch phase")
				}
				if boundary != "none" {
					durable := rc11Save(t, state)
					cut := false
					_, _, err := applyManaged(context.Background(), "stop", state, func(s ManagedState) error {
						cut = boundary == "stop-intent" || !s.MaintenanceUncertain
						if cut && boundary == "before-resolution" {
							return errors.New("crash before resolution commit")
						}
						if err := persist(s); err != nil {
							return err
						}
						durable = rc11Save(t, s)
						if cut {
							return errors.New("crash after stop callback")
						}
						return nil
					}, f.d)
					if err == nil || !cut {
						t.Fatal("Stop interruption missed", err)
					}
					state = rc11Reload(t, durable)
				}
				for range 2 {
					var durable []byte
					next, obs, err := applyManaged(context.Background(), "stop", state, func(s ManagedState) error {
						durable = rc11Save(t, s)
						return persist(s)
					}, f.d)
					if err != nil || obs.RecoveryRequired || obs.State != "stopped" || next.MaintenanceUncertain || next.Maintenance.FinishedAt != nil || next.RollNeedsRuntime {
						t.Fatal("report-only maintenance not resolved, or falsely marked successful", obs, err)
					}
					state = rc11Reload(t, durable)
				}
				for _, call := range f.calls[before:] {
					if call[0] == "container" {
						t.Fatal("report-only Stop attempted container work", call)
					}
				}
				if _, obs, err := applyManaged(context.Background(), "start", state, persist, f.d); err != nil || obs.Health != "healthy" {
					t.Fatal("resolved report still prevented Start", obs, err)
				}
			})
		}
	}
}

func TestRC11ReportOnlyMaintenanceRequiresMatchingDurablePhase(t *testing.T) {
	for _, fault := range []string{"legacy", "different-phase", "dispatched-phase", "contrary-dispatch", "prune"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := rc11MaintenanceReport(t, "pack")
			switch fault {
			case "legacy":
				state.Runtime.Stage = "stopped"
			case "different-phase":
				state.Runtime.Stage = managedMaintenancePendingStage("meld")
			case "dispatched-phase":
				state.Runtime.Stage = "maintenance-dispatched-pack"
			case "contrary-dispatch":
				state.Runtime.Containers = append(state.Runtime.Containers, ContainerRecord{Kind: "pack", Name: "urbit-managed-fake-" + state.ID + "-pack", Removed: true, CreateDispatched: true})
			case "prune":
				state.PrefixPrune = &PrefixPruneRecord{Uncertain: true}
			}
			for range 2 {
				next, obs, err := applyManaged(context.Background(), "stop", state, persist, f.d)
				if err != nil || !obs.RecoveryRequired || !next.MaintenanceUncertain {
					t.Fatal("ambiguous/dispatched or unrelated uncertainty cleared", obs, err)
				}
				state = rc11Reload(t, rc11Save(t, next))
			}
		})
	}
}

func TestRC11MaintenancePendingPhaseRevokedBeforeDaemonDispatch(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	var durable ManagedState
	create := false
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if args[4] == "container" && args[5] == "create" && strings.HasSuffix(args[7], "-pack") {
			create = true
			last := durable.Runtime.Containers[len(durable.Runtime.Containers)-1]
			if managedMaintenanceReportPending(durable, durable.Runtime) || !last.CreateDispatched || !last.CreateUncertain || !durable.MaintenanceUncertain {
				t.Fatal("utility dispatch still had non-dispatch evidence")
			}
		}
		if args[4] == "container" && args[5] == "wait" {
			return "", errors.New("utility outcome lost")
		}
		return execute(ctx, args, env)
	}
	_, _, err := applyManaged(context.Background(), "pack", state, func(s ManagedState) error {
		durable = rc11Reload(t, rc11Save(t, s))
		return persist(s)
	}, f.d)
	if err == nil || !create {
		t.Fatal("dispatch witness missed", err)
	}
	state, obs, err := applyManaged(context.Background(), "stop", durable, persist, f.d)
	if err != nil || !obs.RecoveryRequired || !state.MaintenanceUncertain || len(f.containers) != 0 {
		t.Fatal("cleanup erased dispatched maintenance quarantine", obs, err)
	}
}
