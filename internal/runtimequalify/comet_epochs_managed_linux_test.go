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

func stoppedManagedCometEpochFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := managedCometFixture(t)
	for _, action := range []string{"start", "stop"} {
		var err error
		state, _, err = applyManaged(context.Background(), action, state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
	}
	writePierEpochPath(t, managedPier(f, state), "0i0", false)
	execute := f.d.execute
	rolls := 0
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if err == nil && a[4] == "container" && a[5] == "wait" {
			for _, c := range f.containers {
				if c.id == a[6] && c.kind == "comet-roll" {
					rolls++
					writePierEpochPath(t, managedPier(f, state), fmt.Sprintf("0i%d", rolls*100), true)
				}
			}
		}
		return out, err
	}
	return f, state, persist
}

func TestManagedCometEpochInspectionRollCyclesAndIdentity(t *testing.T) {
	f, state, persist := stoppedManagedCometEpochFixture(t)
	ctx := context.Background()
	state, obs, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
	if err != nil || obs.State != "stopped" || len(state.Epochs.Entries) != 1 || state.Epochs.Entries[0].Name != "0i0" {
		t.Fatal("comet baseline", err)
	}
	uncertain := false
	save := func(s ManagedState) error { uncertain = s.MaintenanceUncertain; return persist(s) }
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if a[4] == "container" && a[5] == "create" && strings.HasSuffix(a[7], "-comet-roll") {
			if !uncertain || !reflect.DeepEqual(a[len(a)-4:], []string{"roll", "--loom", "31", "/urbit/pier"}) || !strings.Contains(strings.Join(a, " "), "--network none") {
				t.Fatal("wrong roll command or missing durable intent")
			}
			for _, arg := range a {
				if arg == "--publish" || arg == "-F" || arg == "-B" || arg == "-c" || strings.Contains(arg, "boot.pill") {
					t.Fatal("roll mixed network/bootstrap arguments")
				}
			}
		}
		return execute(ctx, a, e)
	}
	for round := 1; round <= 3; round++ {
		state, obs, err = applyManaged(ctx, "roll", state, save, f.d)
		if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.MaintenanceUncertain || !state.RollNeedsRuntime || state.Maintenance.CreatedEpoch != fmt.Sprintf("0i%d", round*100) || len(state.Epochs.Entries) != round+1 || len(f.containers) != 0 {
			t.Fatalf("roll %d: %+v %v", round, obs, err)
		}
		before := len(f.calls)
		if _, _, err := applyManaged(ctx, "roll", state, save, f.d); err == nil || len(f.calls) != before {
			t.Fatal("consecutive roll dispatched without a runtime cycle")
		}
		state, obs, err = applyManaged(ctx, "start", state, save, f.d)
		if err != nil || obs.Health != "healthy" || state.Comet.Ship != fixtureComet || state.RollNeedsRuntime || state.Epochs != nil || f.downloads != 1 {
			t.Fatal("roll resume/identity", err)
		}
		if _, _, err := applyManaged(ctx, "inspect-epochs", state, save, f.d); err == nil {
			t.Fatal("running comet was inspected")
		}
		state, _, err = applyManaged(ctx, "stop", state, save, f.d)
		if err != nil {
			t.Fatal(err)
		}
		state, _, err = applyManaged(ctx, "inspect-epochs", state, save, f.d)
		if err != nil || len(state.Epochs.Entries) != round+1 {
			t.Fatal("epoch set changed across runtime replacement", err)
		}
	}
	p, err := epochs.Plan(state.Epochs, "0i0", state.Epochs.Digest, state.Epochs.ObservedAt)
	if err != nil || len(p.Selected) != 1 || len(p.Retained) != 3 || p.DeletionSupported {
		t.Fatal("comet preview scope", err)
	}
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i0", InventoryDigest: state.Epochs.Digest, Confirmed: false}
	before := len(f.calls)
	if _, _, err := applyManaged(ctx, "prune-epochs", state, persist, f.d); err == nil || len(f.calls) != before {
		t.Fatal("unconfirmed comet deletion reached runtime")
	}
}

func TestManagedCometEpochFailuresStayQuarantined(t *testing.T) {
	for _, fault := range []string{"timeout", "no-growth", "snapshot", "renamed-old", "intent-save", "completion-save", "resume-identity"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedCometEpochFixture(t)
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
				if a[4] == "container" && a[5] == "wait" && fault == "timeout" {
					return "", context.DeadlineExceeded
				}
				out, err := execute(ctx, a, e)
				if err == nil && a[4] == "container" && a[5] == "wait" {
					base := filepath.Join(managedPier(f, state), ".urb/log")
					switch fault {
					case "no-growth":
						if err := os.RemoveAll(filepath.Join(base, "0i100")); err != nil {
							t.Fatal(err)
						}
					case "snapshot":
						if err := os.Remove(filepath.Join(base, "0i100/image.bin")); err != nil {
							t.Fatal(err)
						}
					case "renamed-old":
						if err := os.Rename(filepath.Join(base, "0i0"), filepath.Join(base, "0i50")); err != nil {
							t.Fatal(err)
						}
					}
				}
				return out, err
			}
			save := func(s ManagedState) error {
				if s.Maintenance != nil && (fault == "intent-save" && s.MaintenanceUncertain || fault == "completion-save" && !s.MaintenanceUncertain) {
					return errors.New("journal failed")
				}
				return persist(s)
			}
			ctx := context.Background()
			state, obs, err := applyManaged(ctx, "roll", state, save, f.d)
			if fault == "resume-identity" {
				if err != nil {
					t.Fatal(err)
				}
				f.d.hostIdentity = func(context.Context, string) (string, error) {
					return "~marzod-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", nil
				}
				state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
				if !state.Comet.Uncertain || !state.RollNeedsRuntime {
					t.Fatal("identity change cleared the roll/resume guard")
				}
			} else if !state.MaintenanceUncertain || state.Maintenance == nil || state.Maintenance.FinishedAt != nil {
				t.Fatal("roll uncertainty lost", err)
			}
			if err == nil || !obs.RecoveryRequired {
				t.Fatal("unverified outcome accepted")
			}
			if fault == "intent-save" {
				for _, event := range f.events {
					if event == "create-comet-roll" {
						t.Fatal("dispatch after persistence failure")
					}
				}
			}
			state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
			if err != nil || !obs.RecoveryRequired || len(f.containers) != 0 {
				t.Fatal("cleanup lost quarantine", err)
			}
			_, obs, err = observeManaged(ctx, state, f.d)
			if err != nil || !obs.RecoveryRequired {
				t.Fatal("observation erased quarantine")
			}
			for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs", "prune-epochs"} {
				if _, _, err := applyManaged(ctx, kind, state, persist, f.d); err == nil {
					t.Fatal("unsafe continuation", kind)
				}
			}
		})
	}
}

func TestManagedCometEpochAdmissionAndProvenance(t *testing.T) {
	for _, fault := range []string{"memory", "epoch-limit", "provenance"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedCometEpochFixture(t)
			switch fault {
			case "memory":
				f.d.admission = func(string) (uint64, uint64, error) { return 0, 8 << 30, nil }
			case "epoch-limit":
				for i := 1; i < epochs.MaxEpochs; i++ {
					writePierEpochPath(t, managedPier(f, state), fmt.Sprintf("0i%d", i), true)
				}
			case "provenance":
				state.Comet.FinishedAt = nil
			}
			state, _, err := applyManaged(context.Background(), "roll", state, persist, f.d)
			if err == nil || state.Maintenance != nil || state.MaintenanceUncertain {
				t.Fatal("admission allowed a roll", err)
			}
			for _, event := range f.events {
				if event == "create-comet-roll" {
					t.Fatal("roll dispatched after refusal")
				}
			}
			if fault == "provenance" {
				if _, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d); err == nil {
					t.Fatal("inspection trusted incomplete provenance")
				}
			}
		})
	}
}
