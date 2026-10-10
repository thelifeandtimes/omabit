//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func epochFixture(t *testing.T, f *fakeEngine, state ManagedState, names ...string) string {
	t.Helper()
	base := filepath.Join(managedPier(f, state), ".urb/log")
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		dir := filepath.Join(base, name)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for file, body := range map[string]string{"epoc.txt": "2\n", "vere.txt": candidate.VereVersion + "\n", "data.mdb": "fixture-data-not-an-LMDB\n"} {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return base
}

func TestEpochInspectionIsMetadataOnlyStableAndNumericallySorted(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	base := epochFixture(t, f, state, "0i0", "0i2", "0i10", "0i9007199254740993")
	beforeCalls := len(f.calls)
	state, obs, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || obs.State != "stopped" || obs.RecoveryRequired || state.Epochs == nil {
		t.Fatalf("inspection: %+v %v", obs, err)
	}
	first := state.Epochs
	if first.Source != epochs.Source || first.PierID != state.ID || !epochs.ValidDigest(first.Digest) || first.ObservedAt.IsZero() || len(first.Entries) != 4 {
		t.Fatalf("missing inventory binding: %+v", first)
	}
	for i, want := range []string{"0i0", "0i2", "0i10", "0i9007199254740993"} {
		e := first.Entries[i]
		if e.Name != want || e.Protected != (i >= 2) || e.FileCount != 3 || e.LogicalBytes <= 0 || e.AllocatedBytes <= 0 {
			t.Fatalf("epoch entry: %+v", e)
		}
	}
	for _, call := range f.calls[beforeCalls:] {
		if call[0] != "info" {
			t.Fatalf("inspection mutated/contacted runtime: %v", call)
		}
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || state.Epochs.Digest != first.Digest {
		t.Fatalf("stable metadata changed digest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "0i2/extra"), []byte("new metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || state.Epochs.Digest == first.Digest {
		t.Fatalf("changed inventory reused digest: %v", err)
	}
	state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil || state.Epochs != nil {
		t.Fatalf("start retained inventory: %v", err)
	}
	if _, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d); err == nil {
		t.Fatal("inspected a running pier")
	}
}

func TestEpochInspectionRejectsUnsafeAndUnsupportedLayouts(t *testing.T) {
	for _, fault := range []string{"symlink file", "symlink epoch", "symlink log", "nested directory", "fifo", "unknown version", "unknown vere", "missing database", "oversized metadata", "noncanonical epoch", "unexpected root entry", "too many files", "canceled"} {
		t.Run(fault, func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			base := epochFixture(t, f, state, "0i0")
			path := filepath.Join(base, "0i0")
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch fault {
			case "symlink file":
				must(os.Symlink("epoc.txt", filepath.Join(path, "linked")))
			case "symlink epoch":
				must(os.Rename(path, path+"-original"))
				must(os.Symlink("0i0-original", path))
			case "symlink log":
				must(os.Rename(base, base+"-original"))
				must(os.Symlink("log-original", base))
			case "nested directory":
				must(os.Mkdir(filepath.Join(path, "nested"), 0700))
			case "fifo":
				must(syscall.Mkfifo(filepath.Join(path, "pipe"), 0600))
			case "unknown version":
				must(os.WriteFile(filepath.Join(path, "epoc.txt"), []byte("1"), 0600))
			case "unknown vere":
				must(os.WriteFile(filepath.Join(path, "vere.txt"), []byte("other"), 0600))
			case "missing database":
				must(os.Remove(filepath.Join(path, "data.mdb")))
			case "oversized metadata":
				must(os.WriteFile(filepath.Join(path, "epoc.txt"), []byte(strings.Repeat("2", 129)), 0600))
			case "noncanonical epoch":
				must(os.Rename(path, filepath.Join(base, "0i00")))
			case "unexpected root entry":
				must(os.Mkdir(filepath.Join(base, "tmp"), 0700))
			case "too many files":
				for i := 0; i < epochs.MaxFilesPerEpoch; i++ {
					must(os.WriteFile(filepath.Join(path, fmt.Sprint("extra-", i)), nil, 0600))
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "canceled" {
				cancel()
			}
			state, _, err := applyManaged(ctx, "inspect-epochs", state, persist, f.d)
			if err == nil || state.Epochs != nil || state.MaintenanceUncertain {
				t.Fatalf("invalid inspection published or quarantined pier: %+v %v", state, err)
			}
		})
	}
}

func TestEpochInspectionDetectsConcurrentChangeAndSharedFiles(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	base := epochFixture(t, f, state, "0i0", "0i10", "0i20")
	if err := os.Link(filepath.Join(base, "0i0/data.mdb"), filepath.Join(base, "0i10/shared.mdb")); err != nil {
		t.Fatal(err)
	}
	state, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || !state.Epochs.Entries[0].SharedFiles || !state.Epochs.Entries[1].SharedFiles {
		t.Fatalf("shared allocation hidden: %v", err)
	}
	stat := f.d.stat
	seen := 0
	f.d.stat = func(path string) (node, error) {
		if path == base {
			seen++
			if seen == 3 {
				if err := os.WriteFile(filepath.Join(base, "0i0/new"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		return stat(path)
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err == nil || !strings.Contains(err.Error(), "changed during inspection") || state.Epochs != nil {
		t.Fatalf("mixed inventory accepted: %v", err)
	}
}

func TestEpochInspectionCountAndPersistenceGates(t *testing.T) {
	t.Run("epoch bound", func(t *testing.T) {
		f, state, persist := stoppedManagedFixture(t)
		var names []string
		for i := 0; i <= epochs.MaxEpochs; i++ {
			names = append(names, fmt.Sprintf("0i%d", i))
		}
		epochFixture(t, f, state, names...)
		state, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
		if err == nil || state.Epochs != nil {
			t.Fatal("oversized inventory was published")
		}
	})
	t.Run("journal failure", func(t *testing.T) {
		f, state, _ := stoppedManagedFixture(t)
		epochFixture(t, f, state, "0i0")
		state, _, err := applyManaged(context.Background(), "inspect-epochs", state, func(ManagedState) error { return errors.New("journal unavailable") }, f.d)
		if err == nil || state.Epochs != nil || state.MaintenanceUncertain {
			t.Fatal("failed persistence published inventory or quarantined an unmodified pier")
		}
	})
}
