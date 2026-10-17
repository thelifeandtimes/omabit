//go:build linux && amd64

package runtimequalify

// Retained-evidence and failure cases adapted from wundrellama's PR #3.
import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sort"
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
		s.Storage = &pierstore.Location{Path: "livenet/planets/sampel-palnet/sampel-palnet"}
		s.Boot = &ManagedBoot{}
		s.AmesPort = 34567
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
	makeRecoveryWorkspace(t, s)
	return f, s, persist
}
func makeRecoveryWorkspace(t *testing.T, s ManagedState) {
	t.Helper()
	if err := os.Mkdir(s.Runtime.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(s.Runtime.Directory, "docker-config"), 0700); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverImportStoppedAndNewAmesConfiguration(t *testing.T) {
	for _, live := range []bool{false, true} {
		for _, stopped := range []bool{false, true} {
			f, s, persist := retainedImport(t, live)
			if stopped {
				var err error
				s, _, err = applyManaged(context.Background(), "stop", s, persist, f.d)
				if err != nil {
					t.Fatal(err)
				}
			}
			before, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, s))
			if err != nil {
				t.Fatal(err)
			}
			beforeCalls := len(f.calls)
			next, obs, err := applyManaged(context.Background(), "recover-import", s, persist, f.d)
			if err != nil || obs.State != "stopped" || obs.RecoveryRequired || !next.Initialized || next.UncertainProvisioning() || next.Adoption.Verified || next.ExternalImport.Stage != "complete" {
				t.Fatal(obs, err)
			}
			if live && (next.Boot == nil || next.AmesPort != 34567 || next.Runtime.PillURL != "runtime-canonical") {
				t.Fatal("recovery lost new Ames/provenance configuration")
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, next))
			if err != nil || before != after || len(f.calls) != beforeCalls {
				t.Fatal("recovery changed data or invoked runtime", err)
			}
			if _, _, err = applyManaged(context.Background(), "recover-import", next, persist, f.d); err == nil {
				t.Fatal("completed recovery repeated")
			}
		}
	}
}

func TestRecoverRejectsUnsafeRetainedEvidence(t *testing.T) {
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
			if _, _, err = applyManaged(context.Background(), "start", next, persist, f.d); err == nil {
				t.Fatal("failed recovery unfenced start")
			}
			if len(f.calls) != 0 {
				t.Fatal("recovery invoked Docker/Vere")
			}
			if _, err = os.Lstat(path); err != nil {
				t.Fatal("destination deleted", err)
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
					if err := os.WriteFile(filepath.Join(path, "sentinel"), []byte("requires review"), 0600); err != nil {
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
					t.Fatal("recreated evidence cleared fence", obs, err)
				}
				for i, p := range paths {
					after, _, _, err := pierstore.TreeReview(context.Background(), p)
					if err != nil || after != before[i] {
						t.Fatal("evidence modified", p, err)
					}
				}
				if len(f.calls) != 0 {
					t.Fatal("recovery invoked runtime")
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
					t.Skip("requires unprivileged runner")
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
				s.ExternalImport.HeldPath = source + "-unrecorded"
			case "source-reappeared":
				original := f.d.stat
				changed := false
				f.d.stat = func(path string) (node, error) {
					if !changed && path == filepath.Join(managedPier(f, s), ".urb/log/0i0/vere.txt") {
						changed = true
						must(os.Mkdir(source, 0700))
					}
					return original(path)
				}
			}
			writes := 0
			next, obs, err := applyManaged(context.Background(), "recover-import", s, func(ManagedState) error { writes++; return nil }, f.d)
			if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() || writes != 0 || len(f.calls) != 0 {
				t.Fatal("unsafe evidence cleared fence", obs, err, writes)
			}
		})
	}
}

func TestRecoverWorkspaceAndPublicationFailure(t *testing.T) {
	for _, fault := range []string{"missing", "linked", "writable", "foreign-owner", "safe-replacement", "journal"} {
		t.Run(fault, func(t *testing.T) {
			f, s, persist := retainedImport(t, true)
			config := filepath.Join(s.Runtime.Directory, "docker-config")
			before, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, s))
			if err != nil {
				t.Fatal(err)
			}
			if fault != "journal" {
				if err = os.Rename(config, config+"-retained"); err != nil {
					t.Fatal(err)
				}
				if fault == "linked" {
					err = os.Symlink(config+"-retained", config)
				} else if fault != "missing" {
					err = os.Mkdir(config, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
				if fault == "writable" {
					if err = os.Chmod(config, 0777); err != nil {
						t.Fatal(err)
					}
				}
				if fault == "foreign-owner" {
					original := f.d.stat
					f.d.stat = func(path string) (node, error) {
						n, err := original(path)
						if path == config {
							n.uid++
						}
						return n, err
					}
				}
			}
			callback := persist
			if fault == "journal" {
				callback = func(ManagedState) error { return errors.New("journal unavailable") }
			}
			next, obs, err := applyManaged(context.Background(), "recover-import", s, callback, f.d)
			if fault == "safe-replacement" {
				if err != nil || obs.RecoveryRequired || !next.Initialized {
					t.Fatal(obs, err)
				}
			} else if err == nil || !obs.RecoveryRequired || !next.ImportRecoveryEligible() {
				t.Fatal("failed recovery unfenced state", obs, err)
			}
			if fault == "journal" {
				if next.Adoption.FinishedAt != nil || next.Comet.FinishedAt != nil || len(next.ExternalImport.Events) != len(s.ExternalImport.Events) {
					t.Fatal("proposed completion mutated original")
				}
				next, obs, err = applyManaged(context.Background(), "recover-import", next, persist, f.d)
				if err != nil || obs.State != "stopped" || next.UncertainProvisioning() || next.Adoption.Verified {
					t.Fatal("explicit retry failed", obs, err)
				}
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, next))
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("data changed or runtime invoked", err)
			}
		})
	}
}

func TestPreviewAndWorkerRejectEpochsBeforeMoving(t *testing.T) {
	for _, version := range []string{"4.6", "4.6-5a7614f", "4.6-other"} {
		t.Run(version, func(t *testing.T) {
			f, s, persist, source := externalFixture(t)
			if err := os.WriteFile(filepath.Join(source, ".urb/log/0i0/vere.txt"), []byte(version), 0600); err != nil {
				t.Fatal(err)
			}
			review, err := pierstore.InspectExternal(context.Background(), f.d.home, source)
			if err != nil {
				t.Fatal(err)
			}
			s.ExternalImport.Source = review
			s.Adoption.Fingerprint = review.Fingerprint
			_, err = inspectExternalPier(context.Background(), source, f.d)
			if (err == nil) != (version != "4.6-other") {
				t.Fatal("preview compatibility", err)
			}
			if version == "4.6-other" {
				next, _, err := applyManaged(context.Background(), "import-pier", s, persist, f.d)
				if err == nil || next.Prepared || next.ExternalImport.Stage != "reviewed" || next.Storage.MountIdentity.Inode != 0 {
					t.Fatal("bad preflight moved source", err)
				}
			}
			after, err := pierstore.InspectExternal(context.Background(), f.d.home, source)
			if err != nil || after != review || len(f.calls) != 0 {
				t.Fatal("preflight modified data", err)
			}
		})
	}
}

// Synthetic public META database; no event/key material. The reader itself has
// independent official @p vectors and fuzz/format tests in piermeta/protocol.
func writeRecoveryIdentity(t *testing.T, path string, who, fake byte) {
	t.Helper()
	data := make([]byte, 16384)
	for n := 0; n < 2; n++ {
		p := data[n*4096:]
		binary.LittleEndian.PutUint64(p, uint64(n))
		binary.LittleEndian.PutUint16(p[10:], 8)
		binary.LittleEndian.PutUint32(p[16:], 0xbeefc0de)
		binary.LittleEndian.PutUint32(p[20:], 1)
		binary.LittleEndian.PutUint32(p[40:], 4096)
		binary.LittleEndian.PutUint64(p[128:], 2)
		binary.LittleEndian.PutUint64(p[136:], 3)
		binary.LittleEndian.PutUint64(p[144:], uint64(n+1))
	}
	db := make([]byte, 48)
	binary.LittleEndian.PutUint64(db[40:], 3)
	for n, values := range []map[string][]byte{{"META": db}, {"fake": {fake}, "version": {3}, "who": {who}}} {
		p := data[(n+2)*4096 : (n+3)*4096]
		binary.LittleEndian.PutUint64(p, uint64(n+2))
		binary.LittleEndian.PutUint16(p[10:], 2)
		keys := []string{}
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		upper := 4096
		for i, k := range keys {
			v := values[k]
			upper -= (8 + len(k) + len(v) + 1) &^ 1
			node := p[upper:]
			binary.LittleEndian.PutUint32(node, uint32(len(v)))
			if n == 0 {
				binary.LittleEndian.PutUint16(node[4:], 2)
			}
			binary.LittleEndian.PutUint16(node[6:], uint16(len(k)))
			copy(node[8:], k)
			copy(node[8+len(k):], v)
			binary.LittleEndian.PutUint16(p[16+2*i:], uint16(upper))
		}
		binary.LittleEndian.PutUint16(p[12:], uint16(16+2*len(keys)))
		binary.LittleEndian.PutUint16(p[14:], uint16(upper))
	}
	if err := os.WriteFile(filepath.Join(path, ".urb/log/data.mdb"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverGuidedInPlaceNamedPier(t *testing.T) {
	for _, fault := range []string{"none", "ames-guided", "moved-guided", "ship", "network", "source-identity", "unexpected-holding", "journal"} {
		t.Run(fault, func(t *testing.T) {
			f, s, persist, source := externalFixture(t)
			s.Storage = &pierstore.Location{Path: "fakeships/imported/zod"}
			if fault == "ames-guided" {
				s.Storage.Path = "livenet/galaxies/imported/zod"
				s.Fake = nil
				s.Comet = &ManagedComet{Ship: "~zod"}
				s.NativeShip = "~zod"
				s.Boot = &ManagedBoot{}
				s.AmesPort = 34567
			}
			target, err := pierstore.Parents(f.d.home, s.Storage.Path)
			if err != nil {
				t.Fatal(err)
			}
			input := source
			if fault != "moved-guided" {
				if err = os.Mkdir(filepath.Dir(target), 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.Rename(source, target); err != nil {
					t.Fatal(err)
				}
				input = target
			} else {
				f.d.renamePier = func(string, string) error { return unix.EXDEV }
			}
			fake := byte(0)
			if fault == "ames-guided" {
				fake = 1
			}
			writeRecoveryIdentity(t, input, 0, fake)
			review, err := pierstore.InspectExternal(context.Background(), f.d.home, input)
			if err != nil {
				t.Fatal(err)
			}
			s.ExternalImport.Source = review
			s.ExternalImport.Identify = true
			s.Adoption.Fingerprint = review.Fingerprint
			s, _, err = applyManaged(context.Background(), "import-pier", s, func(v ManagedState) error {
				if v.Prepared {
					return errors.New("interrupted enrollment")
				}
				return persist(v)
			}, f.d)
			if err == nil || !s.Prepared || s.ExternalImport.Stage != "enrolling" || (s.ExternalImport.HeldPath != "") != (fault == "moved-guided") {
				t.Fatal("bad in-place fixture", err)
			}
			makeRecoveryWorkspace(t, s)
			switch fault {
			case "ship":
				writeRecoveryIdentity(t, target, 1, 0)
			case "network":
				writeRecoveryIdentity(t, target, 0, 1)
			case "source-identity":
				s.ExternalImport.Source.Identity.Inode++
			case "unexpected-holding":
				s.ExternalImport.HeldPath = filepath.Join(filepath.Dir(target), ".omabit-import-"+s.ID, "pier")
			}
			before, _, _, err := pierstore.TreeReview(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			callback := persist
			if fault == "journal" {
				callback = func(ManagedState) error { return errors.New("unavailable journal") }
			}
			next, obs, err := applyManaged(context.Background(), "recover-import", s, callback, f.d)
			if fault == "none" || fault == "ames-guided" || fault == "moved-guided" {
				if err != nil || obs.State != "stopped" || !next.Initialized || next.Adoption.Verified {
					t.Fatal(obs, err)
				}
				if fault == "ames-guided" && (next.Boot == nil || next.AmesPort != 34567) {
					t.Fatal("guided recovery lost Ames publication")
				}
			} else if err == nil || !obs.RecoveryRequired || next.Initialized {
				t.Fatal("unsafe in-place recovery", obs, err)
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), target)
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("in-place data/runtime changed", err)
			}
		})
	}
}
