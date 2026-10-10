//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func managedPruneFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := stoppedManagedFixture(t)
	for _, name := range []string{"0i0", "0i100", "0i200", "0i300"} {
		writePierEpochPath(t, managedPier(f, state), name, name != "0i0")
	}
	var err error
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i0", InventoryDigest: state.Epochs.Digest, Confirmed: true}
	return f, state, persist
}

func TestManagedGenesisPrunePersistsIntentAndResumes(t *testing.T) {
	f, state, persist := managedPruneFixture(t)
	var saved ManagedState
	save := func(s ManagedState) error { saved = s; return persist(s) }
	var removed []string
	f.d.remove = func(root *os.Root, name string) error {
		if !saved.MaintenanceUncertain || saved.PrefixPrune == nil || !saved.PrefixPrune.Uncertain || saved.PruneIntent == nil || !saved.PruneIntent.Confirmed {
			t.Fatal("managed deletion without durable confirmed intent")
		}
		removed = append(removed, filepath.Join(filepath.Base(root.Name()), name))
		return root.Remove(name)
	}
	state, obs, err := applyManaged(context.Background(), "prune-epochs", state, save, f.d)
	if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.MaintenanceUncertain || state.PrefixPrune.Uncertain || state.PrefixPrune.FinishedAt == nil || state.PruneIntent != nil || state.Epochs == nil || len(state.Epochs.Entries) != 3 {
		t.Fatalf("managed deletion failed: %+v %+v %v", state, obs, err)
	}
	want := []string{"0i0/data.mdb", "0i0/epoc.txt", "0i0/lock.mdb", "0i0/vere.txt", "log/0i0"}
	if !reflect.DeepEqual(removed, want) || !sameRetainedEntries(state.PrefixPrune.Before, state.PrefixPrune.After) {
		t.Fatal("wrong deletion/retention set")
	}
	state, obs, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || obs.Health != "healthy" || state.Epochs != nil || f.downloads != 1 {
		t.Fatalf("same-pier resume: %+v %v", obs, err)
	}
}

func TestManagedPruneRefusesUnconfirmedStaleOrRunningInput(t *testing.T) {
	for _, fault := range []string{"unconfirmed", "cutoff", "digest", "changed disk", "running"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedPruneFixture(t)
			spec := *state.PruneIntent
			switch fault {
			case "unconfirmed":
				spec.Confirmed = false
			case "cutoff":
				spec.Cutoff = "0i200"
			case "digest":
				spec.InventoryDigest = strings.Repeat("0", 64)
			case "changed disk":
				if err := os.WriteFile(filepath.Join(managedPier(f, state), ".urb/log/0i0/data.mdb"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "running":
				var err error
				state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			state.PruneIntent = &spec
			removed := 0
			f.d.remove = func(*os.Root, string) error { removed++; return errors.New("unexpected unlink") }
			state, _, err := applyManaged(context.Background(), "prune-epochs", state, persist, f.d)
			if err == nil || removed != 0 || state.MaintenanceUncertain || state.PrefixPrune != nil {
				t.Fatalf("invalid intent mutated pier: %v", err)
			}
		})
	}
}

func TestManagedPruneFailuresStayQuarantinedAfterStop(t *testing.T) {
	for _, fault := range []string{"partial unlink", "intent persistence", "completion persistence"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedPruneFixture(t)
			removed := 0
			f.d.remove = func(root *os.Root, name string) error {
				removed++
				if fault == "partial unlink" && removed == 2 {
					return errors.New("unlink failed")
				}
				return root.Remove(name)
			}
			save := func(s ManagedState) error {
				if p := s.PrefixPrune; p != nil && ((fault == "intent persistence" && p.Uncertain) || (fault == "completion persistence" && !p.Uncertain)) {
					return errors.New("journal failed")
				}
				return persist(s)
			}
			state, _, err := applyManaged(context.Background(), "prune-epochs", state, save, f.d)
			if err == nil || !state.MaintenanceUncertain || !state.PrefixPrune.Uncertain || state.Epochs != nil {
				t.Fatalf("prune quarantine lost: %+v %v", state, err)
			}
			if fault == "intent persistence" && removed != 0 {
				t.Fatal("mutation preceded intent persistence")
			}
			state, obs, err := applyManaged(context.Background(), "stop", state, persist, f.d)
			if err != nil || !obs.RecoveryRequired || !state.MaintenanceUncertain {
				t.Fatal("stop cleared partial deletion")
			}
			for _, kind := range []string{"start", "pack", "meld", "roll", "prune-epochs", "inspect-epochs"} {
				if _, _, err := applyManaged(context.Background(), kind, state, persist, f.d); err == nil {
					t.Fatalf("admitted %s after partial deletion", kind)
				}
			}
		})
	}
}
