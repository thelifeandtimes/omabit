//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Retain serialized callbacks, not the returned in-memory state: these are the
// exact interruption boundaries that must survive a management restart.
func rc11MaintenanceIntent(t *testing.T, kind string, dispatched bool) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	fixture := stoppedManagedFixture
	if strings.HasPrefix(kind, "comet-") {
		fixture = stoppedManagedCometEpochFixture
	}
	f, state, persist := fixture(t)
	action := strings.TrimPrefix(kind, "comet-")
	if kind == "roll" {
		writePierEpochPath(t, managedPier(f, state), "0i0", false)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var durable []byte
	cut := false
	save := func(s ManagedState) error {
		if err := persist(s); err != nil {
			return err
		}
		var err error
		durable, err = json.Marshal(s)
		for _, c := range s.Runtime.Containers {
			if c.Kind == kind && c.CreateDispatched == dispatched {
				cut = true
				cancel()
			}
		}
		return err
	}
	if _, _, err := applyManaged(ctx, action, state, save, f.d); err == nil || !cut || len(f.containers) != 0 {
		t.Fatalf("did not interrupt utility intent: %v cut=%t containers=%d", err, cut, len(f.containers))
	}
	state = ManagedState{}
	if err := json.Unmarshal(durable, &state); err != nil {
		t.Fatal(err)
	}
	return f, state, persist
}

func TestRC11UndispatchedMaintenanceStopAfterReload(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll", "comet-pack", "comet-meld", "comet-roll"} {
		t.Run(kind, func(t *testing.T) {
			f, state, persist := rc11MaintenanceIntent(t, kind, false)
			before := len(f.calls)
			for range 2 {
				var durable []byte
				var obs ManagedObservation
				var err error
				state, obs, err = applyManaged(context.Background(), "stop", state, func(s ManagedState) error {
					durable, err = json.Marshal(s)
					if err != nil {
						return err
					}
					return persist(s)
				}, f.d)
				if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.UncertainMaintenance() || state.Maintenance.FinishedAt != nil || state.RollNeedsRuntime {
					t.Fatalf("undispatched work not resolved (or reported successful): %+v %v", obs, err)
				}
				for _, c := range state.Runtime.Containers {
					if !c.Removed || c.CreateUncertain {
						t.Fatalf("retained resource: %+v", c)
					}
				}
				state = ManagedState{}
				if err := json.Unmarshal(durable, &state); err != nil {
					t.Fatal(err)
				}
			}
			for _, call := range f.calls[before:] {
				if call[0] == "container" {
					t.Fatalf("nonexistent utility reached Docker: %q", call)
				}
			}
			_, obs, err := applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil || obs.Health != "healthy" {
				t.Fatalf("resolved maintenance prevented resume: %+v %v", obs, err)
			}
		})
	}
}

func TestRC11DispatchedMaintenanceAbsenceDoesNotResolveIntent(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll", "comet-pack", "comet-meld", "comet-roll"} {
		t.Run(kind, func(t *testing.T) {
			f, state, persist := rc11MaintenanceIntent(t, kind, true)
			for range 2 {
				var err error
				state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
				last := state.Runtime.Containers[len(state.Runtime.Containers)-1]
				if err == nil || !state.MaintenanceUncertain || !last.CreateDispatched || !last.CreateUncertain || last.Removed {
					t.Fatalf("absence resolved dispatched intent: %+v %v", last, err)
				}
			}
		})
	}
}

func TestRC11MaintenanceStopPreservesUnrelatedOrUnprovenFences(t *testing.T) {
	for _, fault := range []string{"missing-report", "different-report", "missing-intent", "prefix-prune", "prune", "provisioning", "contrary-daemon-evidence"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := rc11MaintenanceIntent(t, "pack", false)
			switch fault {
			case "missing-report":
				state.Maintenance = nil
			case "different-report":
				state.Maintenance.Kind = "meld"
			case "missing-intent":
				state.Runtime.Containers = state.Runtime.Containers[:1]
				state.Runtime.Stage = "stopped" // legacy report-only state has no phase evidence
			case "prefix-prune":
				state.PrefixPrune = &PrefixPruneRecord{Uncertain: true}
			case "prune":
				state.Runtime.Prune = &PruneRecord{Uncertain: true}
			case "provisioning":
				state.Storage.Uncertain = true
			case "contrary-daemon-evidence":
				state.Runtime.Containers[len(state.Runtime.Containers)-1].CreateUncertain = true
			}
			state, obs, err := applyManaged(context.Background(), "stop", state, persist, f.d)
			if fault == "contrary-daemon-evidence" {
				if err == nil || state.Runtime.Containers[len(state.Runtime.Containers)-1].Removed {
					t.Fatal("contrary evidence ignored", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !obs.RecoveryRequired || fault != "provisioning" && !state.MaintenanceUncertain || fault == "provisioning" && !state.Storage.Uncertain {
				t.Fatalf("unrelated or unproven quarantine cleared: %+v", obs)
			}
		})
	}
}

func TestRC11MaintenanceResolutionSurvivesInterruptedStopPersistence(t *testing.T) {
	for _, boundary := range []string{"removal", "quarantine"} {
		t.Run(boundary, func(t *testing.T) {
			f, state, persist := rc11MaintenanceIntent(t, "pack", false)
			durable, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			cut := false
			save := func(s ManagedState) error {
				last := s.Runtime.Containers[len(s.Runtime.Containers)-1]
				if boundary == "removal" && last.Removed || boundary == "quarantine" && !s.MaintenanceUncertain {
					cut = true
					return errors.New("interrupted persistence")
				}
				var err error
				durable, err = json.Marshal(s)
				return err
			}
			if _, _, err := applyManaged(context.Background(), "stop", state, save, f.d); err == nil || !cut {
				t.Fatal("did not reach persistence cut", err)
			}
			state = ManagedState{}
			if err := json.Unmarshal(durable, &state); err != nil {
				t.Fatal(err)
			}
			state, obs, err := applyManaged(context.Background(), "stop", state, persist, f.d)
			if err != nil || obs.RecoveryRequired || state.MaintenanceUncertain {
				t.Fatalf("reloaded cleanup was not retryable: %+v %v", obs, err)
			}
		})
	}
}
