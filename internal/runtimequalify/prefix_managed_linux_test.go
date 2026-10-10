//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestManagedMultiEpochAndRepeatedPrefixes(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	for _, name := range []string{"0i0", "0i100", "0i200", "0i300", "0i400"} {
		writePierEpochPath(t, managedPier(f, state), name, name != "0i0")
	}
	ctx := context.Background()
	var first *PrefixPruneRecord
	for i, cutoff := range []string{"0i100", "0i200"} {
		var err error
		var obs ManagedObservation
		state, _, err = applyManaged(ctx, "inspect-epochs", state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
		before := state.Epochs
		state.PruneIntent = &ManagedPruneIntent{Cutoff: cutoff, InventoryDigest: before.Digest, Confirmed: true}
		state, obs, err = applyManaged(ctx, "prune-epochs", state, persist, f.d)
		selected, remaining := 2, 3
		if i == 1 {
			selected, remaining = 1, 2
		}
		if err != nil || state.UncertainMaintenance() || state.PruneIntent != nil || obs.State != "stopped" || len(state.Epochs.Entries) != remaining || state.PrefixPrune.Cutoff != cutoff || len(state.PrefixPrune.Targets) != selected || !sameRetainedPrefix(before, state.Epochs, selected) {
			t.Fatalf("prefix round: %+v %v", obs, err)
		}
		if i == 0 {
			first = state.PrefixPrune
		} else if first == state.PrefixPrune || first.Cutoff != "0i100" || first.Uncertain || len(first.After.Entries) != 3 {
			t.Fatal("later prune rewrote earlier completed record")
		}
		state, obs, err = applyManaged(ctx, "start", state, persist, f.d)
		if err != nil || obs.Health != "healthy" {
			t.Fatalf("post-prefix resume: %+v %v", obs, err)
		}
		state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
		// Persisted state, not a shared mutable pointer, is what the next driver
		// session receives. Reload it via serialization as the store does.
		data := encode(t, state)
		state = ManagedState{}
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(managedPier(f, state), ".urb/log"))
	if err != nil || len(entries) != 2 || entries[0].Name() != "0i300" || entries[1].Name() != "0i400" {
		t.Fatal("retained newest suffix incorrect")
	}
}

func TestManagedPrefixAfterLegacyGenesisAndIndependentMarker(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	for _, name := range []string{"0i100", "0i200", "0i300"} {
		writePierEpochPath(t, managedPier(f, state), name, true)
	}
	now := time.Now().UTC()
	legacy := &PruneRecord{Cutoff: "0i0", EpochRemoved: true, FinishedAt: &now}
	state.Runtime.Prune = legacy
	ctx := context.Background()
	state, _, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: state.Epochs.Digest, Confirmed: true}
	f.d.remove = func(root *os.Root, name string) error {
		if name == "image.bin" {
			return errors.New("injected snapshot unlink failure")
		}
		return root.Remove(name)
	}
	state, _, err = applyManaged(ctx, "prune-epochs", state, persist, f.d)
	if err == nil || !state.UncertainMaintenance() || !reflect.DeepEqual(state.Runtime.Prune, legacy) || state.PrefixPrune.Cutoff != "0i100" {
		t.Fatalf("legacy/repeated state lost: %+v %v", state, err)
	}
	// The deletion's own marker is independently authoritative.
	state.MaintenanceUncertain = false
	if _, _, err := applyManaged(ctx, "start", state, persist, f.d); err == nil {
		t.Fatal("generic flag erased prefix quarantine")
	}
	_, obs, err := observeManaged(ctx, state, f.d)
	if err != nil || !obs.RecoveryRequired {
		t.Fatal("observation erased prefix marker")
	}
	state.PrefixPrune.Uncertain = false
	if _, _, err := applyManaged(ctx, "start", state, persist, f.d); err == nil {
		t.Fatal("missing completion evidence became safe")
	}
}

func TestManagedPrefixStopsBetweenSelectedEpochs(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	for _, name := range []string{"0i0", "0i100", "0i200", "0i300", "0i400"} {
		writePierEpochPath(t, managedPier(f, state), name, name != "0i0")
	}
	ctx := context.Background()
	state, _, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: state.Epochs.Digest, Confirmed: true}
	f.d.remove = func(root *os.Root, name string) error {
		if filepath.Base(root.Name()) == "0i100" {
			return errors.New("second epoch unavailable")
		}
		return root.Remove(name)
	}
	state, _, err = applyManaged(ctx, "prune-epochs", state, persist, f.d)
	if err == nil || !state.UncertainMaintenance() || len(state.PrefixPrune.Targets) != 2 || !state.PrefixPrune.Targets[0].Removed || state.PrefixPrune.Targets[1].Removed {
		t.Fatalf("partial prefix not recorded: %+v %v", state.PrefixPrune, err)
	}
	state, obs, err := applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil || !obs.RecoveryRequired || !state.UncertainMaintenance() {
		t.Fatal("stop cleared partial prefix")
	}
}
