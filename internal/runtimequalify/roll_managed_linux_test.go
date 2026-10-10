//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
)

func TestManagedRollCreatesEpochAndResumesSamePier(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	writePierEpochPath(t, managedPier(f, state), "0i0", false)
	execute := f.d.execute
	rolls := 0
	uncertain := false
	save := func(s ManagedState) error { uncertain = s.MaintenanceUncertain; return persist(s) }
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if a[4] == "container" && a[5] == "create" && strings.HasSuffix(a[7], "-roll") {
			if !uncertain || !reflect.DeepEqual(a[len(a)-4:], []string{"roll", "--loom", "31", "/urbit/pier"}) {
				t.Fatal("roll command or durable intent mismatch")
			}
			for _, arg := range a {
				if arg == "--publish" || arg == "-F" || arg == "-B" || arg == "-c" || strings.Contains(arg, "boot.pill") {
					t.Fatal("roll mixed network/bootstrap arguments")
				}
			}
		}
		out, err := execute(ctx, a, e)
		if a[4] == "container" && a[5] == "wait" {
			rolls++
			writePierEpochPath(t, managedPier(f, state), fmt.Sprintf("0i%d", rolls*100), true)
		}
		return out, err
	}
	for round := 1; round <= 3; round++ {
		var obs ManagedObservation
		var err error
		state, obs, err = applyManaged(context.Background(), "roll", state, save, f.d)
		if err != nil || state.MaintenanceUncertain || obs.State != "stopped" || obs.RecoveryRequired || state.Maintenance.FinishedAt == nil || state.Maintenance.CreatedEpoch != fmt.Sprintf("0i%d", round*100) || len(state.Maintenance.EpochsBefore) != round || len(state.Maintenance.EpochsAfter) != round+1 || state.Epochs == nil || len(f.containers) != 0 {
			t.Fatalf("roll: %+v %+v %v", state, obs, err)
		}
		if round == 3 {
			p, err := epochs.Plan(state.Epochs, "0i0", state.Epochs.Digest, state.Epochs.ObservedAt)
			if err != nil || len(p.Selected) != 1 || len(p.Retained) != 3 {
				t.Fatalf("roll inventory not usable for prefix preview: %+v %v", p, err)
			}
		}
		calls := len(f.calls)
		if _, _, err := applyManaged(context.Background(), "roll", state, save, f.d); err == nil || len(f.calls) != calls || state.MaintenanceUncertain || !state.RollNeedsRuntime {
			t.Fatal("empty consecutive roll was dispatched or quarantined before runtime cycle")
		}
		state, obs, err = applyManaged(context.Background(), "start", state, save, f.d)
		if err != nil || obs.Health != "healthy" || state.Epochs != nil || state.RollNeedsRuntime || f.downloads != 1 {
			t.Fatalf("post-roll resume: %+v %v", obs, err)
		}
		if round < 3 {
			state, _, err = applyManaged(context.Background(), "stop", state, save, f.d)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestManagedRollUnverifiedOutcomesRemainQuarantined(t *testing.T) {
	for _, fault := range []string{"no new epoch", "missing snapshot", "changed old epoch", "timeout", "nonzero", "OOM", "intent persistence", "completion persistence"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			writePierEpochPath(t, managedPier(f, state), "0i0", false)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" && fault == "timeout" {
					return "", context.DeadlineExceeded
				}
				out, err := execute(ctx, a, e)
				if a[4] == "container" && a[5] == "wait" {
					if fault != "no new epoch" {
						writePierEpochPath(t, managedPier(f, state), "0i100", fault != "missing snapshot")
					}
					if fault == "changed old epoch" {
						base := filepath.Join(managedPier(f, state), ".urb/log")
						if err := os.Rename(filepath.Join(base, "0i0"), filepath.Join(base, "0i50")); err != nil {
							t.Fatal(err)
						}
					}
					for _, c := range f.containers {
						if fault == "nonzero" {
							mapAt(c.doc, "State")["ExitCode"] = 1
						}
						if fault == "OOM" {
							mapAt(c.doc, "State")["OOMKilled"] = true
						}
					}
				}
				return out, err
			}
			save := func(s ManagedState) error {
				if s.Maintenance != nil && ((fault == "intent persistence" && s.MaintenanceUncertain) || (fault == "completion persistence" && !s.MaintenanceUncertain)) {
					return errors.New("journal failure")
				}
				return persist(s)
			}
			state, obs, err := applyManaged(context.Background(), "roll", state, save, f.d)
			if err == nil || !state.MaintenanceUncertain || !obs.RecoveryRequired || state.Maintenance.FinishedAt != nil {
				t.Fatalf("unverified roll accepted: %+v %v", obs, err)
			}
			if fault == "intent persistence" {
				for _, event := range f.events {
					if event == "create-roll" {
						t.Fatal("roll dispatched after journal refusal")
					}
				}
			}
			state, obs, err = applyManaged(context.Background(), "stop", state, persist, f.d)
			if err != nil || !state.MaintenanceUncertain || !obs.RecoveryRequired || len(f.containers) != 0 {
				t.Fatalf("cleanup erased roll uncertainty: %+v %v", obs, err)
			}
			_, obs, err = observeManaged(context.Background(), state, f.d)
			if err != nil || !obs.RecoveryRequired {
				t.Fatal("observation erased roll quarantine")
			}
			for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs"} {
				if _, _, err := applyManaged(context.Background(), kind, state, persist, f.d); err == nil {
					t.Fatalf("admitted %s on uncertain roll", kind)
				}
			}
		})
	}
}

func TestManagedRollCapacityAndInventoryBoundsPrecedeDispatch(t *testing.T) {
	for _, fault := range []string{"capacity", "epoch bound"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			writePierEpochPath(t, managedPier(f, state), "0i0", false)
			if fault == "epoch bound" {
				for i := 1; i < epochs.MaxEpochs; i++ {
					writePierEpochPath(t, managedPier(f, state), fmt.Sprintf("0i%d", i), true)
				}
			}
			checks := 0
			f.d.admission = func(string) (uint64, uint64, error) {
				checks++
				if fault == "capacity" && checks == 2 {
					return 5<<30 - 1, 8 << 30, nil
				}
				return 5 << 30, 8 << 30, nil
			}
			state, _, err := applyManaged(context.Background(), "roll", state, persist, f.d)
			if err == nil || state.MaintenanceUncertain || state.Maintenance != nil {
				t.Fatalf("pre-dispatch refusal mishandled: %v", err)
			}
			for _, event := range f.events {
				if event == "create-roll" {
					t.Fatal("roll dispatched past admission limit")
				}
			}
		})
	}
}
