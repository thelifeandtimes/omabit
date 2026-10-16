//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func retainedImport(t *testing.T, livenet ...bool) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, s, persist, source := externalFixture(t)
	if len(livenet) > 0 && livenet[0] {
		s.NativeShip, s.Fake = "~sampel-palnet", nil
		s.Comet = &ManagedComet{Ship: s.NativeShip}
		s.Storage = &pierstore.Location{Path: "livenet/planets/sampel-palnet/pier"}
	}
	if err := os.WriteFile(filepath.Join(source, ".urb/log/0i0/vere.txt"), []byte("4.6\n"), 0600); err != nil {
		t.Fatal(err)
	}
	review, err := pierstore.InspectExternal(context.Background(), f.d.home, source)
	if err != nil {
		t.Fatal(err)
	}
	s.ExternalImport.Source = review
	s.Adoption.Fingerprint = review.Fingerprint
	f.d.renamePier = func(string, string) error { return unix.EXDEV }
	s, _, err = applyManaged(context.Background(), "import-pier", s, func(v ManagedState) error {
		if v.Prepared {
			return errors.New("legacy failure after preparation intent")
		}
		return persist(v)
	}, f.d)
	if err == nil || !s.Prepared || s.ExternalImport.Stage != "enrolling" {
		t.Fatal("bad retained fixture", err)
	}
	if err = os.Mkdir(s.Runtime.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(s.Runtime.Directory, "docker-config"), 0700); err != nil {
		t.Fatal(err)
	}
	return f, s, persist
}

func TestRecoverRejects(t *testing.T) {
	for _, fault := range []string{"phase", "identity", "fingerprint", "mount", "replace", "urb", "version", "format", "symlink", "hardlink", "lock", "urb-lock", "changed", "runtime-pin", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			f, s, persist := retainedImport(t)
			path := managedPier(f, s)
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch fault {
			case "phase":
				s.ExternalImport.Stage = "copying"
			case "identity":
				s.Storage.Identity.Inode++
			case "fingerprint":
				s.Adoption.Fingerprint = "bad"
			case "mount":
				s.Storage.MountIdentity.Inode++
			case "replace":
				must(os.Rename(path, path+"-retained"))
				must(os.MkdirAll(filepath.Join(path, ".urb"), 0700))
			case "urb":
				must(os.Rename(filepath.Join(path, ".urb"), filepath.Join(path, ".urb-retained")))
				must(os.Mkdir(filepath.Join(path, ".urb"), 0700))
			case "version":
				must(os.WriteFile(filepath.Join(path, ".urb/log/0i0/vere.txt"), []byte("4.6-other"), 0600))
			case "format":
				must(os.WriteFile(filepath.Join(path, ".urb/log/0i0/epoc.txt"), []byte("3"), 0600))
			case "symlink":
				must(os.Symlink(".urb", filepath.Join(path, "linked")))
			case "hardlink":
				must(os.Link(filepath.Join(path, ".urb/log/0i0/data.mdb"), filepath.Join(path, "linked")))
			case "lock":
				must(os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("123"), 0600))
			case "urb-lock":
				must(os.WriteFile(filepath.Join(path, ".urb/lock"), []byte("123"), 0600))
			case "runtime-pin":
				s.Runtime.VereVersion = "4.6"
			case "changed":
				original := f.d.stat
				changed := false
				f.d.stat = func(p string) (node, error) {
					if !changed && p == filepath.Join(path, ".urb/log/0i0/vere.txt") {
						changed = true
						must(os.WriteFile(filepath.Join(path, "changed-during-inspection"), []byte("change"), 0600))
					}
					return original(p)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancel" {
				cancel()
			}
			writes := 0
			next, obs, err := applyManaged(ctx, "recover-import", s, func(v ManagedState) error { writes++; return persist(v) }, f.d)
			if err == nil || !obs.RecoveryRequired || next.Initialized || writes != 0 {
				t.Fatal("unsafe recovery completed", obs, err, writes)
			}
			if _, _, err := applyManaged(context.Background(), "start", next, persist, f.d); err == nil {
				t.Fatal("failed recovery unfenced start")
			}
			if len(f.calls) != 0 {
				t.Fatal("recovery invoked Docker/Vere")
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("recovery deleted destination", err)
			}
		})
	}
}

func TestRecoverRecreatedPaths(t *testing.T) {
	for _, location := range []string{"source", "held"} {
		for _, evidence := range []string{"copied-pier", "locked-pier", "unrelated-sentinel"} {
			t.Run(location+"/"+evidence, func(t *testing.T) {
				f, s, _ := retainedImport(t)
				path := s.ExternalImport.Source.Path
				if location == "held" {
					path = s.ExternalImport.HeldPath
				}
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
				if evidence == "unrelated-sentinel" {
					if err := os.WriteFile(filepath.Join(path, "sentinel"), []byte("not a pier; still needs review"), 0600); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.CopyFS(path, os.DirFS(managedPier(f, s))); err != nil {
						t.Fatal(err)
					}
					if evidence == "locked-pier" {
						if err := os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("123"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				paths := []string{path, managedPier(f, s)}
				before := make([]string, len(paths))
				for i, p := range paths {
					var err error
					before[i], _, _, err = pierstore.TreeReview(context.Background(), p)
					if err != nil {
						t.Fatal(err)
					}
				}
				writes := 0
				next, obs, err := applyManaged(context.Background(), "recover-import", s, func(ManagedState) error { writes++; return nil }, f.d)
				if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() || writes != 0 {
					t.Error("recreated evidence cleared recovery fence", obs, err, writes)
				}
				for i, p := range paths {
					after, _, _, err := pierstore.TreeReview(context.Background(), p)
					if err != nil || after != before[i] {
						t.Fatal("evidence modified", p, err)
					}
				}
				if len(f.calls) != 0 {
					t.Fatal("recovery invoked Docker/Vere")
				}
			})
		}
	}
}

func TestRecoverPathTraversal(t *testing.T) {
	for _, fault := range []string{"source-link", "held-link", "parent-link", "held-parent-link", "inaccessible-parent", "inaccessible-held-parent", "device", "bad-held-path", "source-reappeared"} {
		t.Run(fault, func(t *testing.T) {
			f, s, _ := retainedImport(t)
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			source, held := s.ExternalImport.Source.Path, s.ExternalImport.HeldPath
			parent := filepath.Dir(source)
			switch fault {
			case "source-link":
				must(os.Symlink("nonexistent", source))
			case "held-link":
				must(os.Mkdir(filepath.Dir(held), 0700))
				must(os.Symlink("nonexistent", held))
			case "parent-link":
				must(os.Rename(parent, parent+"-saved"))
				t.Cleanup(func() { os.RemoveAll(parent + "-saved") })
				must(os.Symlink(parent+"-saved", parent))
			case "held-parent-link":
				must(os.Symlink(parent, filepath.Dir(held)))
			case "inaccessible-parent", "inaccessible-held-parent":
				if os.Geteuid() == 0 {
					t.Skip("permission denial requires unprivileged runner")
				}
				if fault == "inaccessible-held-parent" {
					parent = filepath.Dir(held)
					must(os.Mkdir(parent, 0700))
				}
				must(os.Chmod(parent, 0000))
				t.Cleanup(func() { os.Chmod(parent, 0700) })
			case "device":
				s.ExternalImport.Source.Identity.Device++
			case "bad-held-path":
				s.ExternalImport.HeldPath = source + "-not-recorded-holding"
			case "source-reappeared":
				stat := f.d.stat
				recreated := false
				f.d.stat = func(path string) (node, error) {
					if !recreated && path == filepath.Join(managedPier(f, s), ".urb/log/0i0/vere.txt") {
						recreated = true
						must(os.Mkdir(source, 0700))
					}
					return stat(path)
				}
			}
			writes := 0
			next, obs, err := applyManaged(context.Background(), "recover-import", s, func(ManagedState) error { writes++; return nil }, f.d)
			if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() || writes != 0 {
				t.Fatal("unsafe path evidence cleared fence", obs, err, writes)
			}
			if len(f.calls) != 0 {
				t.Fatal("recovery invoked Docker/Vere")
			}
		})
	}
}

func TestRecoverWorkspaceChecks(t *testing.T) {
	for _, fault := range []string{"missing", "linked", "writable", "foreign-owner", "safe-replacement"} {
		t.Run(fault, func(t *testing.T) {
			f, s, persist := retainedImport(t)
			config := filepath.Join(s.Runtime.Directory, "docker-config")
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			must(os.Rename(config, config+"-retained"))
			must(os.WriteFile(filepath.Join(config+"-retained", "sentinel"), []byte("retained workspace"), 0600))
			if fault == "linked" {
				must(os.Symlink(config+"-retained", config))
			}
			if fault == "writable" || fault == "safe-replacement" || fault == "foreign-owner" {
				must(os.Mkdir(config, 0700))
			}
			if fault == "writable" {
				must(os.Chmod(config, 0777))
			}
			if fault == "foreign-owner" {
				stat := f.d.stat
				f.d.stat = func(path string) (node, error) {
					n, err := stat(path)
					if path == config {
						n.uid++
					}
					return n, err
				}
			}
			next, obs, err := applyManaged(context.Background(), "recover-import", s, persist, f.d)
			if fault == "safe-replacement" {
				// Runtime workspaces have path/ownership checks, not retained inode
				// identities. Do not advertise them as historical integrity proof.
				if err != nil || obs.RecoveryRequired || !next.Initialized {
					t.Fatal("safe workspace rejected", obs, err)
				}
			} else if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() {
				t.Fatal("unsafe workspace accepted", obs, err)
			}
			body, err := os.ReadFile(filepath.Join(config+"-retained", "sentinel"))
			if err != nil || string(body) != "retained workspace" || len(f.calls) != 0 {
				t.Fatal("workspace changed or runtime invoked", err)
			}
		})
	}
}

func TestRecoverJournalFailure(t *testing.T) {
	f, s, persist := retainedImport(t, true)
	before, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, s))
	if err != nil {
		t.Fatal(err)
	}
	next, obs, err := applyManaged(context.Background(), "recover-import", s, func(ManagedState) error { return errors.New("journal unavailable") }, f.d)
	if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() || next.Adoption.FinishedAt != nil || next.Comet.FinishedAt != nil {
		t.Fatal("failed publication cleared fence", obs, err)
	}
	if _, _, err := applyManaged(context.Background(), "start", next, persist, f.d); err == nil {
		t.Fatal("start escaped failed journal")
	}
	next, obs, err = applyManaged(context.Background(), "recover-import", next, persist, f.d)
	if err != nil || obs.State != "stopped" || next.UncertainProvisioning() || next.Adoption.Verified || next.Comet.FinishedAt == nil {
		t.Fatal("explicit retry failed", obs, err)
	}
	after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, next))
	if err != nil || before != after {
		t.Fatal("destination changed", err)
	}
	for _, path := range []string{s.ExternalImport.Source.Path, s.ExternalImport.HeldPath} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("source/holding recreated", err)
		}
	}
	if len(f.calls) != 0 {
		t.Fatal("recovery invoked runtime")
	}
}

func TestPreviewEpochs(t *testing.T) {
	for _, version := range []string{"4.6", "4.6-5a7614f", "4.6-other"} {
		t.Run(version, func(t *testing.T) {
			f, _, _, source := externalFixture(t)
			path := filepath.Join(source, ".urb/log/0i0/vere.txt")
			if err := os.WriteFile(path, []byte(version), 0600); err != nil {
				t.Fatal(err)
			}
			before, _, _, err := pierstore.TreeReview(context.Background(), source)
			if err != nil {
				t.Fatal(err)
			}
			_, err = inspectExternalPier(context.Background(), source, f.d)
			if (err == nil) != (version != "4.6-other") {
				t.Fatal("preview compatibility", err)
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), source)
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("preview not read-only", err)
			}
		})
	}
}
