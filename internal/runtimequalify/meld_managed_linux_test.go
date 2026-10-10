//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func stoppedManagedFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := managedFixture(t)
	var err error
	state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	return f, state, persist
}

func TestManagedMeldDispatchUsesDurableStateAndQualifiedCommand(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	var stored ManagedState
	save := func(s ManagedState) error {
		// Serialize like SQLite: do not accidentally share the measurement pointer.
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		stored = ManagedState{}
		if err := json.Unmarshal(data, &stored); err != nil {
			t.Fatal(err)
		}
		return persist(s)
	}
	execute := f.d.execute
	dispatched := false
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if args[5] == "create" && strings.HasSuffix(args[7], "-meld") {
			dispatched = true
			if !stored.MaintenanceUncertain || stored.Maintenance == nil || stored.Maintenance.Kind != "meld" || stored.Maintenance.FinishedAt != nil {
				t.Fatal("meld dispatched before durable uncertainty")
			}
			if !reflect.DeepEqual(args[len(args)-4:], []string{"meld", "--loom", "31", "/urbit/pier"}) {
				t.Fatalf("unqualified command: %q", args)
			}
			for _, arg := range args {
				if arg == "--publish" || arg == "-F" || arg == "-B" || arg == "-c" || strings.Contains(arg, "boot.pill") {
					t.Fatal("meld mixed networking or bootstrap arguments")
				}
			}
		}
		return execute(ctx, args, env)
	}
	state, obs, err := applyManaged(context.Background(), "meld", state, save, f.d)
	if err != nil || !dispatched || obs.State != "stopped" || stored.MaintenanceUncertain || !reflect.DeepEqual(stored.Maintenance, state.Maintenance) {
		t.Fatalf("managed meld outcome: %+v %+v %v", state, obs, err)
	}
}

func TestManagedMaintenanceRechecksCapacityAfterProbe(t *testing.T) {
	for _, kind := range []string{"pack", "meld"} {
		for _, fault := range []string{"RAM", "disk", "unavailable"} {
			t.Run(kind+"/"+fault, func(t *testing.T) {
				f, state, persist := stoppedManagedFixture(t)
				checks := 0
				f.d.admission = func(string) (uint64, uint64, error) {
					checks++
					if checks == 1 {
						return 6 << 30, 10 << 30, nil
					}
					if fault == "RAM" {
						return 5<<30 - 1, 10 << 30, nil
					}
					if fault == "disk" {
						return 6 << 30, 8<<30 - 1, nil
					}
					return 0, 0, errors.New("capacity unavailable")
				}
				state, _, err := applyManaged(context.Background(), kind, state, persist, f.d)
				if err == nil || checks != 2 || state.MaintenanceUncertain || state.Maintenance != nil || len(f.containers) != 0 {
					t.Fatalf("fresh capacity gate: %+v %v", state, err)
				}
				for _, event := range f.events {
					if event == "create-"+kind {
						t.Fatal("utility dispatched with insufficient or unknown resources")
					}
				}
			})
		}
	}
}

func TestManagedMeldFailuresCannotClearQuarantine(t *testing.T) {
	for _, fault := range []string{"nonzero", "OOM", "initial persistence", "final persistence"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				out, err := execute(ctx, args, env)
				if args[5] == "wait" {
					for _, c := range f.containers {
						if fault == "nonzero" {
							mapAt(c.doc, "State")["ExitCode"] = 1
						} else if fault == "OOM" {
							mapAt(c.doc, "State")["OOMKilled"] = true
						}
					}
				}
				return out, err
			}
			save := func(s ManagedState) error {
				if s.Maintenance != nil && ((fault == "initial persistence" && s.MaintenanceUncertain) || (fault == "final persistence" && !s.MaintenanceUncertain)) {
					return errors.New("journal failure")
				}
				return persist(s)
			}
			state, obs, err := applyManaged(context.Background(), "meld", state, save, f.d)
			if err == nil || !obs.RecoveryRequired || !state.MaintenanceUncertain || state.Maintenance.FinishedAt != nil {
				t.Fatalf("lost uncertainty: %+v %+v %v", state, obs, err)
			}
			if fault == "initial persistence" {
				for _, event := range f.events {
					if event == "create-meld" {
						t.Fatal("dispatched after failed uncertainty persistence")
					}
				}
			}
			state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
			if err != nil || !obs.RecoveryRequired || !state.MaintenanceUncertain || len(f.containers) != 0 {
				t.Fatalf("cleanup cleared uncertain pier: %+v %v", obs, err)
			}
			for _, next := range []string{"start", "pack", "meld"} {
				if _, _, err := applyManaged(context.Background(), next, state, persist, f.d); err == nil {
					t.Fatalf("admitted %s on uncertain pier", next)
				}
			}
		})
	}
}
