//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func managedCometPrefixFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist := stoppedManagedCometEpochFixture(t)
	for _, name := range []string{"0i100", "0i200", "0i300", "0i400"} {
		writePierEpochPath(t, managedPier(f, state), name, true)
	}
	return f, state, persist
}

func TestManagedCometRepeatedPrefixesAndPinnedResumes(t *testing.T) {
	f, state, persist := managedCometPrefixFixture(t)
	var durable []byte
	save := func(s ManagedState) error {
		var err error
		durable, err = json.Marshal(s)
		if err != nil {
			return err
		}
		return persist(s)
	}
	var removed []string
	f.d.remove = func(root *os.Root, name string) error {
		var stored ManagedState
		if json.Unmarshal(durable, &stored) != nil || stored.PrefixPrune == nil || !stored.PrefixPrune.Uncertain || !stored.MaintenanceUncertain || stored.PruneIntent == nil || !stored.PruneIntent.Confirmed {
			t.Fatal("unlink preceded durable confirmed intent")
		}
		if !strings.HasPrefix(root.Name(), filepath.Join(managedPier(f, state), ".urb/log")) {
			t.Fatal("unlink escaped comet log", root.Name())
		}
		removed = append(removed, filepath.Join(filepath.Base(root.Name()), name))
		return root.Remove(name)
	}
	ctx := context.Background()
	var first []byte
	for round, cutoff := range []string{"0i100", "0i200"} {
		var err error
		state, _, err = applyManaged(ctx, "inspect-epochs", state, save, f.d)
		if err != nil {
			t.Fatal(err)
		}
		before := state.Epochs
		state.PruneIntent = &ManagedPruneIntent{Cutoff: cutoff, InventoryDigest: before.Digest, Confirmed: true}
		var obs ManagedObservation
		state, obs, err = applyManaged(ctx, "prune-epochs", state, save, f.d)
		selected, remaining := 2, 3
		if round == 1 {
			selected, remaining = 1, 2
		}
		if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.UncertainMaintenance() || state.PruneIntent != nil || len(state.Epochs.Entries) != remaining || len(state.PrefixPrune.Targets) != selected || !sameRetainedPrefix(before, state.Epochs, selected) || state.Comet.Ship != fixtureComet {
			t.Fatalf("prefix %d: %+v %v", round, obs, err)
		}
		if round == 0 {
			first = []byte(encode(t, state.PrefixPrune))
		} else {
			var previous PrefixPruneRecord
			if json.Unmarshal(first, &previous) != nil || previous.Cutoff != "0i100" || previous.Uncertain || len(previous.After.Entries) != 3 {
				t.Fatal("historical prefix evidence changed")
			}
		}
		state, obs, err = applyManaged(ctx, "start", state, save, f.d)
		if err != nil || obs.Health != "healthy" || state.Comet.Ship != fixtureComet || state.Epochs != nil {
			t.Fatal("post-prune identity resume", err)
		}
		state, _, err = applyManaged(ctx, "stop", state, save, f.d)
		if err != nil {
			t.Fatal(err)
		}
		data := encode(t, state)
		state = ManagedState{}
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"0i0/data.mdb", "0i0/epoc.txt", "0i0/lock.mdb", "0i0/vere.txt", "log/0i0", "0i100/data.mdb", "0i100/epoc.txt", "0i100/image.bin", "0i100/lock.mdb", "0i100/vere.txt", "log/0i100", "0i200/data.mdb", "0i200/epoc.txt", "0i200/image.bin", "0i200/lock.mdb", "0i200/vere.txt", "log/0i200"}
	if !reflect.DeepEqual(removed, want) {
		t.Fatal("wrong managed deletion set", removed)
	}
	state, _, err := applyManaged(ctx, "inspect-epochs", state, save, f.d)
	if err != nil || !reflect.DeepEqual(epochNames(state.Epochs), []string{"0i300", "0i400"}) {
		t.Fatal("wrong retained suffix", err)
	}
	q, err := managedSession(&state, save, f.d)
	if err != nil {
		t.Fatal(err)
	}
	var recovery *EpochRecoveryRecord
	if err := q.recoverLatest(ctx, state.Epochs, "comet-prefix-recovery-1", &recovery); err == nil || recovery != nil {
		t.Fatal("destructive recovery escaped disposable qualification")
	}
}

func TestManagedCometPruneRefusalsHaveNoUnlinks(t *testing.T) {
	for _, fault := range []string{"confirmation", "stale", "protected", "provenance", "changed-source", "hardlink", "running"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedCometPrefixFixture(t)
			ctx := context.Background()
			state, _, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: state.Epochs.Digest, Confirmed: true}
			switch fault {
			case "confirmation":
				state.PruneIntent.Confirmed = false
			case "stale":
				state.PruneIntent.InventoryDigest = strings.Repeat("f", 64)
			case "protected":
				state.PruneIntent.Cutoff = "0i300"
			case "provenance":
				state.Comet.FinishedAt = nil
			case "changed-source":
				if err := os.WriteFile(filepath.Join(managedPier(f, state), ".urb/log/0i100/data.mdb"), []byte("changed after inspection"), 0600); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(managedPier(f, state), ".urb/log/0i100/image.bin"), filepath.Join(state.Runtime.Directory, "outside-link")); err != nil {
					t.Fatal(err)
				}
			case "running":
				state, _, err = applyManaged(ctx, "start", state, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			f.d.remove = func(*os.Root, string) error { t.Fatal("refused request unlinked data"); return nil }
			state, _, err = applyManaged(ctx, "prune-epochs", state, persist, f.d)
			if err == nil || state.PrefixPrune != nil || state.MaintenanceUncertain {
				t.Fatal("unsafe prune accepted", fault, err)
			}
		})
	}
}

func TestManagedCometPartialPruneCannotBeRetriedOrResumed(t *testing.T) {
	for _, fault := range []string{"second-epoch", "snapshot", "intent-save", "finish-save"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := managedCometPrefixFixture(t)
			ctx := context.Background()
			state, _, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			state.PruneIntent = &ManagedPruneIntent{Cutoff: "0i100", InventoryDigest: state.Epochs.Digest, Confirmed: true}
			unlinks := 0
			f.d.remove = func(root *os.Root, name string) error {
				unlinks++
				if fault == "second-epoch" && filepath.Base(root.Name()) == "0i100" || fault == "snapshot" && name == "image.bin" {
					return errors.New("unlink interrupted")
				}
				return root.Remove(name)
			}
			save := func(s ManagedState) error {
				if p := s.PrefixPrune; p != nil && (fault == "intent-save" || fault == "finish-save" && !p.Uncertain) {
					return errors.New("journal failed")
				}
				return persist(s)
			}
			state, obs, err := applyManaged(ctx, "prune-epochs", state, save, f.d)
			if err == nil || !obs.RecoveryRequired || state.PrefixPrune == nil || !state.PrefixPrune.Uncertain || state.PrefixPrune.FinishedAt != nil {
				t.Fatal("partial prune outcome erased", err)
			}
			if fault == "intent-save" && unlinks != 0 {
				t.Fatal("unlink preceded intent persistence")
			}
			state.MaintenanceUncertain = false
			data := encode(t, state)
			state = ManagedState{}
			if err := json.Unmarshal([]byte(data), &state); err != nil {
				t.Fatal(err)
			}
			state, obs, err = applyManaged(ctx, "stop", state, persist, f.d)
			if err != nil || !obs.RecoveryRequired || !state.UncertainMaintenance() {
				t.Fatal("stop cleared independent prefix quarantine", err)
			}
			_, obs, err = observeManaged(ctx, state, f.d)
			if err != nil || !obs.RecoveryRequired {
				t.Fatal("observation adopted partial prune")
			}
			for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs", "prune-epochs"} {
				if _, _, err := applyManaged(ctx, kind, state, persist, f.d); err == nil {
					t.Fatal("unsafe continuation", kind)
				}
			}
		})
	}
}
