//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"golang.org/x/sys/unix"
)

// Synthetic opaque metadata, never a copy of a user's pier or Finder attributes.
func macMetadataFixture(t *testing.T, pier string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for _, name := range []string{
		".urb/log/._0i0", ".urb/log/._0i391592", // orphaned historical companion
		".urb/log/._data.mdb", ".urb/log/._lock.mdb", ".urb/log/.DS_Store",
		".urb/log/0i0/._data.mdb", ".urb/log/0i0/._epoc.txt", ".urb/log/0i0/.DS_Store",
		".urb/chk/._image.bin", ".urb/chk/.DS_Store", ".DS_Store",
	} {
		files[name] = append([]byte{0, 5, 22, 7, 0, 2, 0, 0}, []byte("synthetic metadata for "+name)...)
		rc11Must(t, os.WriteFile(filepath.Join(pier, name), files[name], 0600))
	}
	return files
}

func TestMacMetadataImportPreservesFilesAndStoppedIntent(t *testing.T) {
	for _, mode := range []string{"rename", "copy", "interrupted-copy", "interrupted-enrollment"} {
		t.Run(mode, func(t *testing.T) {
			f, state, persist, source := externalFixture(t)
			want := macMetadataFixture(t, source)
			// Exercise the actual external preflight, not just a permissive copy.
			review, err := inspectExternalPier(context.Background(), source, f.d)
			rc11Must(t, err)
			state.ExternalImport.Source = review
			state.Adoption.Fingerprint = review.Fingerprint
			if mode != "rename" {
				f.d.renamePier = func(string, string) error { return unix.EXDEV }
			}
			save := persist
			if strings.HasPrefix(mode, "interrupted-") {
				save = func(s ManagedState) error {
					if mode == "interrupted-copy" && s.ExternalImport.Stage == "copying" && s.ExternalImport.CopiedFiles > 0 || mode == "interrupted-enrollment" && s.Prepared {
						return errors.New("synthetic interrupted copy")
					}
					return persist(s)
				}
			}
			next, obs, err := applyManaged(context.Background(), "import-pier", state, save, f.d)
			if mode == "interrupted-copy" {
				if err == nil || !next.UncertainProvisioning() || len(f.calls) != 0 {
					t.Fatal("partial copy lost its stopped fence", obs, err)
				}
				for name, body := range want {
					got, readErr := os.ReadFile(filepath.Join(next.ExternalImport.HeldPath, name))
					if readErr != nil || !bytes.Equal(got, body) {
						t.Fatalf("partial copy lost original metadata %q: %v", name, readErr)
					}
				}
				return
			}
			if mode == "interrupted-enrollment" {
				if err == nil || !next.ImportRecoveryEligible() {
					t.Fatal("copy did not retain a recoverable operation", err)
				}
				// Simulate persistence/reload of the partial import before recovery.
				next = rc11Reload(t, rc11Save(t, next))
				next, obs, err = applyManaged(context.Background(), "recover-import", next, persist, f.d)
			}
			if err != nil || obs.State != "stopped" || !next.Initialized || next.ExternalImport.Stage != "complete" || obs.RecoveryRequired {
				t.Fatal("metadata import did not complete stopped", obs, err)
			}
			if len(f.calls) != 0 {
				t.Fatal("import contacted Docker or started a runtime", f.calls)
			}
			for name, body := range want {
				got, err := os.ReadFile(filepath.Join(managedPier(f, next), name))
				if err != nil || !bytes.Equal(got, body) {
					t.Fatalf("metadata %q not preserved: %v", name, err)
				}
			}
			if _, err := os.Lstat(source); !os.IsNotExist(err) {
				t.Fatal("completed import retained the old launch path", err)
			}
		})
	}
}

func TestMacMetadataChangesInvalidateImportReview(t *testing.T) {
	f, state, persist, source := externalFixture(t)
	macMetadataFixture(t, source)
	review, err := inspectExternalPier(context.Background(), source, f.d)
	rc11Must(t, err)
	state.ExternalImport.Source = review
	state.Adoption.Fingerprint = review.Fingerprint
	rc11Must(t, os.WriteFile(filepath.Join(source, ".urb/log/._data.mdb"), []byte("changed metadata"), 0600))
	if _, _, err := applyManaged(context.Background(), "import-pier", state, persist, f.d); err == nil {
		t.Fatal("changed companion bypassed reviewed-source binding")
	}
	if _, err := os.Stat(source); err != nil || len(f.calls) != 0 {
		t.Fatal("refused import moved source or contacted runtime", err, f.calls)
	}
}

func TestMacMetadataEpochInventoryTracksCompanions(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	base := epochFixture(t, f, state, "0i0")
	rc11Must(t, os.WriteFile(filepath.Join(base, "._0i0"), []byte("metadata"), 0600))
	state, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	rc11Must(t, err)
	before := state.Epochs.Digest
	if len(state.Epochs.Entries) != 1 || state.Epochs.Entries[0].Name != "0i0" {
		t.Fatal("metadata became an epoch", state.Epochs)
	}
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || state.Epochs.Digest != before {
		t.Fatal("unchanged metadata was unstable", err)
	}
	rc11Must(t, os.WriteFile(filepath.Join(base, "._0i0"), []byte("changed"), 0600))
	state, _, err = applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || state.Epochs.Digest == before {
		t.Fatal("metadata mutation was not fingerprinted", err)
	}
}

func TestMacMetadataKeepsEpochCountBound(t *testing.T) {
	f, state, persist := stoppedManagedFixture(t)
	names := make([]string, epochs.MaxEpochs)
	for i := range names {
		names[i] = fmt.Sprintf("0i%d", i)
	}
	base := epochFixture(t, f, state, names...)
	for _, name := range append(names, "data.mdb", "lock.mdb") {
		rc11Must(t, os.WriteFile(filepath.Join(base, "._"+name), []byte("metadata"), 0600))
	}
	for _, name := range []string{"data.mdb", "lock.mdb", ".DS_Store"} {
		rc11Must(t, os.WriteFile(filepath.Join(base, name), []byte("fixture"), 0600))
	}
	state, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d)
	if err != nil || len(state.Epochs.Entries) != epochs.MaxEpochs {
		t.Fatal("companions consumed the epoch allowance", err)
	}
	rc11Must(t, os.Mkdir(filepath.Join(base, "0i999"), 0700))
	if _, _, err := applyManaged(context.Background(), "inspect-epochs", state, persist, f.d); err == nil {
		t.Fatal("more than maximum epochs allowed")
	}
}

func TestMacMetadataDoesNotBypassImportSafety(t *testing.T) {
	for _, fault := range []string{"symlink", "directory", "fifo", "hardlink", "writable", "foreign", "missing-real-database", "missing-real-version", "unknown", "noncanonical", "escaped-name"} {
		t.Run(fault, func(t *testing.T) {
			f, _, _, source := externalFixture(t)
			path := filepath.Join(source, ".urb/log/._data.mdb")
			rc11Must(t, os.WriteFile(path, []byte("metadata"), 0600))
			switch fault {
			case "symlink", "directory", "fifo", "hardlink":
				rc11Must(t, os.Remove(path))
				switch fault {
				case "symlink":
					rc11Must(t, os.Symlink("0i0/data.mdb", path))
				case "directory":
					rc11Must(t, os.Mkdir(path, 0700))
				case "fifo":
					rc11Must(t, syscall.Mkfifo(path, 0600))
				case "hardlink":
					rc11Must(t, os.Link(filepath.Join(source, ".urb/log/0i0/data.mdb"), path))
				}
			case "writable":
				rc11Must(t, os.Chmod(path, 0666))
			case "foreign":
				original := f.d.stat
				f.d.stat = func(p string) (node, error) {
					n, err := original(p)
					if p == path {
						n.uid++
					}
					return n, err
				}
			case "missing-real-database", "missing-real-version":
				name := "data.mdb"
				if fault == "missing-real-version" {
					name = "epoc.txt"
				}
				rc11Must(t, os.Rename(filepath.Join(source, ".urb/log/0i0", name), filepath.Join(source, ".urb/log/0i0", "._"+name)))
			case "unknown", "noncanonical", "escaped-name":
				name := "._unexpected"
				if fault == "noncanonical" {
					name = "._0i00"
				}
				if fault == "escaped-name" {
					name = "odd\nentry"
				}
				rc11Must(t, os.WriteFile(filepath.Join(source, ".urb/log", name), []byte("unknown"), 0600))
			}
			_, err := inspectExternalPier(context.Background(), source, f.d)
			if err == nil {
				t.Fatal("unsafe import accepted", fault)
			}
			if fault == "escaped-name" && (!strings.Contains(err.Error(), `"odd\nentry"`) || strings.Contains(err.Error(), "\n")) {
				t.Fatal("unexpected filename is missing or not escaped", err)
			}
			if len(f.calls) != 0 {
				t.Fatal("rejected inspection contacted runtime")
			}
		})
	}
}
