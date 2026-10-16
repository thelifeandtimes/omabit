//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func TestDeletePierRequiresStoppedResourcesAndPreservesNeighbours(t *testing.T) {
	f, s, persist := managedFixture(t)
	ctx := context.Background()
	s, _, err := applyManaged(ctx, "start", s, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = applyManaged(ctx, "delete-pier", s, persist, f.d); err == nil {
		t.Fatal("deleted running pier")
	}
	s, _, err = applyManaged(ctx, "stop", s, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(f.d.home, "piers/fakeships/neighbour")
	if err = os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	path := managedPier(f, s)
	s, obs, err := applyManaged(ctx, "delete-pier", s, persist, f.d)
	if err != nil || obs.State != "deleted" || s.Deletion.Stage != "complete" {
		t.Fatal(obs, err)
	}
	if _, err = os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("pier mount remains", err)
	}
	if _, err = os.Stat(other); err != nil {
		t.Fatal("neighbour removed", err)
	}
	if len(f.networks) != 0 || len(f.containers) != 0 {
		t.Fatal("runtime resources remain")
	}
	if _, _, err = applyManaged(ctx, "start", s, persist, f.d); err == nil {
		t.Fatal("deleted runtime restarted")
	}
}

func TestDeleteUninitializedOwnedRegistrationRemovesPendingKey(t *testing.T) {
	f, s, persist := managedCometFixture(t)
	s.NativeShip = "~sampel-palnet"
	s.Comet.Ship = s.NativeShip
	s.Boot = &ManagedBoot{}
	s.AmesPort = 34343
	s.Storage = &pierstore.Location{Path: "livenet/planets/sampel-palnet/pier"}
	key := "0w" + strings.Repeat("a", 128)
	s.NativeKeyDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/keys")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, s.ID+".key")
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	s, obs, err := applyManaged(context.Background(), "delete-pier", s, persist, f.d)
	if err != nil || obs.State != "deleted" || s.NativeKeyDigest != "" {
		t.Fatal(obs, err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("pending key retained", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("uninitialized deletion dispatched runtime work")
	}
}

func TestDeletePierInterruptedAfterRenameResumesOnlyRecordedDirectory(t *testing.T) {
	f, s, persist := stoppedManagedFixture(t)
	failed := false
	stop := func(state ManagedState) error {
		if state.Deletion != nil && state.Deletion.Stage == "removing" && !failed {
			failed = true
			return errors.New("simulated journal interruption")
		}
		return persist(state)
	}
	s, _, err := applyManaged(context.Background(), "delete-pier", s, stop, f.d)
	if err == nil || !failed {
		t.Fatal("fault not exercised")
	}
	// An interrupted deletion may be reviewed after a reboot renumbers st_dev.
	s.Deletion.Mount.Device += 7
	if _, obs, err := observeManaged(context.Background(), s, f.d); err != nil || !obs.RecoveryRequired {
		t.Fatal("interrupted deletion lost quarantine", obs, err)
	}
	s, obs, err := applyManaged(context.Background(), "delete-pier", s, persist, f.d)
	if err != nil || obs.State != "deleted" {
		t.Fatal(obs, err)
	}
	if _, err = os.Stat(filepath.Join(f.d.home, "piers/archive/deleted-"+s.ID)); !os.IsNotExist(err) {
		t.Fatal("trash remains", err)
	}
}

func TestDeletePierRefusesReplacedMountAndUnownedLinks(t *testing.T) {
	for _, fault := range []string{"replacement", "link"} {
		t.Run(fault, func(t *testing.T) {
			f, s, persist := stoppedManagedFixture(t)
			p := managedPier(f, s)
			if fault == "replacement" {
				if err := os.Rename(filepath.Dir(p), filepath.Dir(p)+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(p, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Symlink(t.TempDir(), filepath.Join(p, "escape")); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := applyManaged(context.Background(), "delete-pier", s, persist, f.d); err == nil {
				t.Fatal("unsafe deletion accepted")
			}
			if _, err := os.Stat(p); err != nil {
				t.Fatal("unsafe pier removed", err)
			}
		})
	}
}

func externalFixture(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error, string) {
	f, s, persist := managedFixture(t)
	f.d.admission = func(string) (uint64, uint64, error) { return 16 << 30, 32 << 30, nil }
	p := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(filepath.Join(p, ".urb/chk"), 0700); err != nil {
		t.Fatal(err)
	}
	writePierEpochPath(t, p, "0i0", false)
	source, err := pierstore.InspectExternal(context.Background(), f.d.home, p)
	if err != nil {
		t.Fatal(err)
	}
	s.Adoption = &ManagedAdoption{Fingerprint: source.Fingerprint, Uncertain: true}
	s.ExternalImport = &ManagedExternalImport{Source: source, Stage: "reviewed"}
	return f, s, persist, p
}

func TestExternalImportMovesAndEnrollsWithSameAndCrossDevicePaths(t *testing.T) {
	for _, copying := range []bool{false, true} {
		t.Run(map[bool]string{false: "rename", true: "cross-device-copy"}[copying], func(t *testing.T) {
			f, s, persist, source := externalFixture(t)
			if copying {
				f.d.renamePier = func(string, string) error { return unix.EXDEV }
			}
			s, obs, err := applyManaged(context.Background(), "import-pier", s, persist, f.d)
			if err != nil || obs.State != "stopped" || !s.Initialized || s.ExternalImport.Stage != "complete" || s.Adoption.Verified {
				t.Fatal(obs, err)
			}
			if _, err = os.Stat(source); !os.IsNotExist(err) {
				t.Fatal("old launch path remains", err)
			}
			if _, err = pierstore.Check(f.d.home, *s.Storage); err != nil {
				t.Fatal(err)
			}
			if len(f.calls) != 0 {
				t.Fatal("import started Docker/Vere")
			}
			if _, err = os.Stat(filepath.Dir(s.ExternalImport.HeldPath)); !os.IsNotExist(err) {
				t.Fatal("source holding directory remains", err)
			}
		})
	}
}

func TestRecoverImport(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		t.Run(fmt.Sprint(stopped), func(t *testing.T) {
			f, s, persist, source := externalFixture(t)
			// Reproduce the retained RC6 state: moved via EXDEV, prepared,
			// but epoch validation failed before adoption could finish.
			f.d.renamePier = func(string, string) error { return unix.EXDEV }
			interrupt := func(v ManagedState) error {
				if v.Prepared {
					return errors.New("legacy enrollment interrupted")
				}
				return persist(v)
			}
			var err error
			s, _, err = applyManaged(context.Background(), "import-pier", s, interrupt, f.d)
			if err == nil || !s.Prepared || s.ExternalImport.Stage != "enrolling" {
				t.Fatal("fixture did not reach enrolling", err)
			}
			if err = os.Mkdir(s.Runtime.Directory, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.Mkdir(filepath.Join(s.Runtime.Directory, "docker-config"), 0700); err != nil {
				t.Fatal(err)
			}
			if stopped {
				s, _, err = applyManaged(context.Background(), "stop", s, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			before := len(f.calls)
			digest, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, s))
			if err != nil {
				t.Fatal(err)
			}
			s, obs, err := applyManaged(context.Background(), "recover-import", s, persist, f.d)
			if err != nil || obs.State != "stopped" || obs.RecoveryRequired || !s.Initialized || s.UncertainProvisioning() || s.ExternalImport.Stage != "complete" || s.Adoption.Verified {
				t.Fatal("recovery failed", obs, err)
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, s))
			if err != nil || digest != after || len(f.calls) != before {
				t.Fatal("recovery touched data or runtime", err)
			}
			for _, path := range []string{source, s.ExternalImport.HeldPath} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("recovery recreated source", err)
				}
			}
			if _, _, err := applyManaged(context.Background(), "recover-import", s, persist, f.d); err == nil {
				t.Fatal("completed import recovered twice")
			}
		})
	}
}

func TestImportEpochPreflight(t *testing.T) {
	f, s, persist, source := externalFixture(t)
	path := filepath.Join(source, ".urb/log/0i0/vere.txt")
	if err := os.WriteFile(path, []byte("4.6-unknown\n"), 0600); err != nil {
		t.Fatal(err)
	}
	review, err := pierstore.InspectExternal(context.Background(), f.d.home, source)
	if err != nil {
		t.Fatal(err)
	}
	s.ExternalImport.Source = review
	s.Adoption.Fingerprint = review.Fingerprint
	_, _, err = applyManaged(context.Background(), "import-pier", s, persist, f.d)
	if err == nil {
		t.Fatal("unsupported epochs accepted")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("source moved before epoch validation", err)
	}
	if s.ExternalImport.Stage != "reviewed" || s.Prepared || s.Storage.MountIdentity.Inode != 0 {
		t.Fatal("invalid source changed import state")
	}
	if len(f.calls) != 0 {
		t.Fatal("preflight invoked runtime")
	}
}

func TestImportChangedSourceDoesNotMoveAndInterruptedCopyStaysFenced(t *testing.T) {
	f, s, persist, source := externalFixture(t)
	if err := os.WriteFile(filepath.Join(source, "changed"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := applyManaged(context.Background(), "import-pier", s, persist, f.d); err == nil {
		t.Fatal("stale source review accepted")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	f, s, persist, source = externalFixture(t)
	f.d.renamePier = func(string, string) error { return unix.EXDEV }
	interrupt := func(v ManagedState) error {
		if v.ExternalImport.Stage == "copying" && v.ExternalImport.CopiedFiles > 0 {
			return errors.New("interrupted during copy")
		}
		return persist(v)
	}
	s, _, err := applyManaged(context.Background(), "import-pier", s, interrupt, f.d)
	if err == nil || !s.UncertainProvisioning() {
		t.Fatal("interrupted import not quarantined")
	}
	if _, err = os.Stat(source); !os.IsNotExist(err) {
		t.Fatal("old source can still launch", err)
	}
	if _, err = os.Stat(s.ExternalImport.HeldPath); err != nil {
		t.Fatal("held source lost", err)
	}
	if _, _, err = applyManaged(context.Background(), "start", s, persist, f.d); err == nil {
		t.Fatal("partial target bootstrapped")
	}
}

func TestCustomStorageMountAndDiskAdmission(t *testing.T) {
	f, s, persist := managedFixture(t)
	disk := t.TempDir()
	if _, err := pierstore.RegisterRoot(f.d.home, "ssd", disk); err != nil {
		t.Fatal(err)
	}
	s.Storage = &pierstore.Location{Path: "@ssd/fakeships/fixture/pier"}
	original := f.d.admission
	f.d.admission = func(path string) (uint64, uint64, error) {
		if path == disk {
			return 10 << 30, 1 << 30, nil
		}
		return original(path)
	}
	if _, _, err := applyManaged(context.Background(), "start", s, persist, f.d); err == nil {
		t.Fatal("custom disk capacity ignored")
	}
	f.d.admission = original
	s, obs, err := applyManaged(context.Background(), "start", s, persist, f.d)
	if err != nil || obs.Health != "healthy" {
		t.Fatal(obs, err)
	}
	for _, c := range f.containers {
		if fakeMountRoot(c) != filepath.Join(disk, "fakeships/fixture") {
			t.Fatal("wrong mount root", fakeMountRoot(c))
		}
	}
}
