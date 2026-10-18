//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

func rc11Save(t *testing.T, state ManagedState) []byte {
	t.Helper()
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rc11Reload(t *testing.T, b []byte) (state ManagedState) {
	t.Helper()
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func rc11Must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRC11DeletionContinuationPreservesReplacements(t *testing.T) {
	for _, phase := range []string{"renaming", "removing", "partial"} {
		for _, object := range []string{"pier", "urb", "file", "new-child", "native-lock"} {
			t.Run(phase+"/"+object, func(t *testing.T) {
				f, state, persist := stoppedManagedFixture(t)
				path := managedPier(f, state)
				rc11Must(t, os.WriteFile(filepath.Join(path, "retained-file"), []byte("original"), 0600))
				var saved []byte
				stopPhase := phase
				if phase == "partial" {
					stopPhase = "removing"
				}
				_, _, err := applyManaged(context.Background(), "delete-pier", state, func(v ManagedState) error {
					if v.Deletion != nil && v.Deletion.Stage == stopPhase {
						saved = rc11Save(t, v)
						return errors.New("durable phase interruption")
					}
					return persist(v)
				}, f.d)
				if err == nil || len(saved) == 0 {
					t.Fatal("phase not exercised")
				}
				state = rc11Reload(t, saved)
				if phase != "renaming" {
					path = filepath.Join(f.d.home, "piers/archive/deleted-"+state.ID, filepath.Base(path))
				}
				if phase == "partial" {
					removed := false
					f.d.remove = func(root *os.Root, name string) error {
						if removed {
							return errors.New("unlink interruption")
						}
						removed = true
						return root.Remove(name)
					}
					_, _, err = applyManaged(context.Background(), "delete-pier", state, persist, f.d)
					if err == nil || !removed {
						t.Fatal("partial deletion not exercised")
					}
					state = rc11Reload(t, saved)
				}
				kept := t.TempDir()
				sentinel := filepath.Join(path, "sentinel")
				switch object {
				case "pier":
					rc11Must(t, os.Rename(path, filepath.Join(kept, "pier")))
					rc11Must(t, os.MkdirAll(filepath.Join(path, ".urb"), 0700))
				case "urb":
					rc11Must(t, os.Rename(filepath.Join(path, ".urb"), filepath.Join(kept, "urb")))
					rc11Must(t, os.Mkdir(filepath.Join(path, ".urb"), 0700))
					sentinel = filepath.Join(path, ".urb/sentinel")
				case "file":
					if err := os.Rename(filepath.Join(path, "retained-file"), filepath.Join(kept, "file")); !errors.Is(err, os.ErrNotExist) {
						rc11Must(t, err)
					}
					sentinel = filepath.Join(path, "retained-file")
				case "native-lock":
					sentinel = filepath.Join(path, ".vere.lock")
				}
				rc11Must(t, os.WriteFile(sentinel, []byte("must survive"), 0600))
				unlinks := 0
				f.d.remove = func(*os.Root, string) error { unlinks++; return errors.New("unexpected unlink") }
				_, obs, err := applyManaged(context.Background(), "delete-pier", state, persist, f.d)
				if err == nil || !obs.RecoveryRequired || unlinks != 0 {
					t.Fatal("replacement deletion was not refused", obs, err, unlinks)
				}
				data, err := os.ReadFile(sentinel)
				if err != nil || string(data) != "must survive" {
					t.Fatal("replacement lost", err)
				}
			})
		}
	}
}

func TestRC11DeletionResumesAfterUrbAndAfterRootUnlink(t *testing.T) {
	for _, last := range []string{".urb", "mount"} {
		t.Run(last, func(t *testing.T) {
			f, state, persist := stoppedManagedFixture(t)
			var saved []byte
			interrupted := false
			f.d.remove = func(root *os.Root, name string) error {
				err := root.Remove(name)
				if err == nil && (last == ".urb" && name == ".urb" || last == "mount" && name == "deleted-"+state.ID) {
					interrupted = true
					return errors.New("unlink completed but outcome lost")
				}
				return err
			}
			_, _, err := applyManaged(context.Background(), "delete-pier", state, func(v ManagedState) error {
				saved = rc11Save(t, v)
				return persist(v)
			}, f.d)
			if err == nil || !interrupted {
				t.Fatal("unlink not interrupted", err)
			}
			state = rc11Reload(t, saved)
			f.d.remove = func(root *os.Root, name string) error { return root.Remove(name) }
			state, obs, err := applyManaged(context.Background(), "delete-pier", state, persist, f.d)
			if err != nil || obs.State != "deleted" || state.Deletion.Stage != "complete" {
				t.Fatal("legitimate continuation failed", obs, err)
			}
		})
	}
}

func TestRC11LegacyDeletionRequiresRecordedChild(t *testing.T) {
	for _, phase := range []string{"renaming", "removing"} {
		for _, replaced := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/replaced=%v", phase, replaced), func(t *testing.T) {
				f, state, persist := stoppedManagedFixture(t)
				path := managedPier(f, state)
				state.Deletion = &ManagedDeletion{Stage: phase, Mount: state.Storage.MountIdentity}
				if phase == "removing" {
					trash := filepath.Join(f.d.home, "piers/archive/deleted-"+state.ID)
					rc11Must(t, pierstore.RenameDirectory(filepath.Dir(path), trash))
					path = filepath.Join(trash, filepath.Base(path))
				}
				if replaced {
					rc11Must(t, os.Rename(path, filepath.Join(t.TempDir(), "original")))
					rc11Must(t, os.MkdirAll(filepath.Join(path, ".urb"), 0700))
				}
				_, obs, err := applyManaged(context.Background(), "delete-pier", state, persist, f.d)
				if replaced {
					if err == nil || !obs.RecoveryRequired {
						t.Fatal("legacy replacement accepted", obs, err)
					}
					rc11Must(t, func() error { _, err := os.Stat(path); return err }())
				} else if err != nil || obs.State != "deleted" {
					t.Fatal("intact legacy deletion stranded", obs, err)
				}
			})
		}
	}
}

func rc11Incoming(t *testing.T) (*fakeEngine, ManagedState, func(ManagedState) error) {
	t.Helper()
	f, state, persist, source := externalFixture(t)
	root, err := os.OpenRoot(source)
	rc11Must(t, err)
	var archive bytes.Buffer
	summary, err := piercopy.WriteArchive(context.Background(), root, &archive)
	root.Close()
	rc11Must(t, err)
	state.ExternalImport, state.Adoption = nil, nil
	state.Handoff = &protocol.Handoff{ID: strings.Repeat("b", 32), Direction: "in", State: "receiving", SourceRunner: "source", SourceInstance: "original", DestinationRunner: "destination", DestinationInstance: "received", Ship: "~zod", Mode: "fake", Bytes: int64(archive.Len()), SHA256: fmt.Sprintf("%x", sha256.Sum256(archive.Bytes())), TreeDigest: summary.Digest}
	path := HandoffArchive(filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), state.Handoff)
	rc11Must(t, os.Mkdir(filepath.Dir(path), 0700))
	rc11Must(t, os.WriteFile(path, archive.Bytes(), 0400))
	return f, state, persist
}

func TestRC11IncomingPreservesUnrecordedDestination(t *testing.T) {
	f, state, persist := rc11Incoming(t)
	var saved []byte
	_, _, err := applyManaged(context.Background(), "receive-handoff", state, func(v ManagedState) error {
		if v.Storage.MountIdentity.Inode != 0 {
			saved = rc11Save(t, v)
			return errors.New("mount saved")
		}
		return persist(v)
	}, f.d)
	if err == nil || len(saved) == 0 {
		t.Fatal("missing mount interruption")
	}
	state = rc11Reload(t, saved)
	path := managedPier(f, state)
	rc11Must(t, os.MkdirAll(filepath.Join(path, ".urb"), 0700))
	for _, file := range []string{"precious", ".vere.lock"} {
		rc11Must(t, os.WriteFile(filepath.Join(path, file), []byte("preserve"), 0600))
	}
	_, _, err = applyManaged(context.Background(), "receive-handoff", state, persist, f.d)
	if err == nil {
		t.Fatal("unrecorded destination was adopted or replaced")
	}
	for _, file := range []string{"precious", ".vere.lock"} {
		data, err := os.ReadFile(filepath.Join(path, file))
		if err != nil || string(data) != "preserve" {
			t.Fatal("unrecorded child lost", err)
		}
	}
}

func TestRC11IncomingRetryAtPublicationBoundaries(t *testing.T) {
	for _, phase := range []string{"before-stage-record", "partial-stage", "ready", "published"} {
		t.Run(phase, func(t *testing.T) {
			f, state, persist := rc11Incoming(t)
			var saved []byte
			var partial string
			_, _, err := applyManaged(context.Background(), "receive-handoff", state, func(v ManagedState) error {
				if phase == "before-stage-record" && v.HandoffStaging != nil {
					partial = v.HandoffStaging.Path
					return errors.New("before staging identity is recorded")
				}
				if phase == "published" && v.Initialized {
					return errors.New("before received state publication")
				}
				saved = rc11Save(t, v)
				if v.HandoffStaging != nil && (phase == "partial-stage" && !v.HandoffStaging.Ready || phase == "ready" && v.HandoffStaging.Ready) {
					partial = v.HandoffStaging.Path
					return errors.New("after staging journal callback")
				}
				return persist(v)
			}, f.d)
			if err == nil || len(saved) == 0 {
				t.Fatal("missing interruption", err)
			}
			state = rc11Reload(t, saved)
			if phase == "partial-stage" {
				rc11Must(t, os.Mkdir(filepath.Join(partial, ".urb"), 0700))
				rc11Must(t, os.WriteFile(filepath.Join(partial, ".urb/partial"), []byte("retained incomplete data"), 0600))
			}
			state, _, err = applyManaged(context.Background(), "receive-handoff", state, persist, f.d)
			if err != nil || state.Handoff.State != "received" || !state.Initialized {
				t.Fatal("legitimate incoming recovery stranded", err)
			}
			if phase == "before-stage-record" || phase == "partial-stage" {
				_, err := os.Stat(partial)
				rc11Must(t, err)
			}
			if phase == "partial-stage" {
				_, err := os.Stat(filepath.Join(partial, ".urb/partial"))
				rc11Must(t, err)
			}
			if len(f.calls) != 0 {
				t.Fatal("receipt dispatched runtime work")
			}
		})
	}
}

func TestRC11IncomingReadyStagingReplacementIsPreserved(t *testing.T) {
	for _, replacement := range []string{"root", "child", "lock"} {
		t.Run(replacement, func(t *testing.T) {
			f, state, persist := rc11Incoming(t)
			var saved []byte
			_, _, err := applyManaged(context.Background(), "receive-handoff", state, func(v ManagedState) error {
				if v.HandoffStaging != nil && v.HandoffStaging.Ready {
					saved = rc11Save(t, v)
					return errors.New("before publish")
				}
				return persist(v)
			}, f.d)
			if err == nil || len(saved) == 0 {
				t.Fatal("missing ready interruption")
			}
			state = rc11Reload(t, saved)
			stage := state.HandoffStaging.Path
			if replacement == "root" {
				rc11Must(t, os.Rename(stage, stage+"-original"))
				rc11Must(t, os.Mkdir(stage, 0700))
			}
			file := filepath.Join(stage, "precious")
			if replacement == "lock" {
				file = filepath.Join(stage, ".vere.lock")
			}
			rc11Must(t, os.WriteFile(file, []byte("preserve"), 0600))
			_, _, err = applyManaged(context.Background(), "receive-handoff", state, persist, f.d)
			if err == nil {
				t.Fatal("changed staging published")
			}
			data, err := os.ReadFile(file)
			if err != nil || string(data) != "preserve" {
				t.Fatal("changed staging deleted", err)
			}
		})
	}
}

func TestRC11IncomingInterruptedExtractionRetainsPartialAndRetries(t *testing.T) {
	f, state, persist := rc11Incoming(t)
	var saved []byte
	_, _, err := applyHandoff(context.Background(), "receive-handoff", state, func(v ManagedState) error {
		saved = rc11Save(t, v)
		return persist(v)
	}, f.d, func(ctx context.Context, archive io.Reader, root *os.Root) (piercopy.Summary, error) {
		return piercopy.ReadArchive(ctx, io.LimitReader(archive, state.Handoff.Bytes/2+19), root)
	})
	if err == nil {
		t.Fatal("truncated extraction succeeded")
	}
	state = rc11Reload(t, saved)
	if state.HandoffStaging == nil || state.HandoffStaging.Ready {
		t.Fatal("extraction did not retain its incomplete ownership record")
	}
	partial := state.HandoffStaging.Path
	before, _, _, err := pierstore.TreeReview(context.Background(), partial)
	rc11Must(t, err)
	entries, err := os.ReadDir(partial)
	if err != nil || len(entries) == 0 {
		t.Fatal("no partial extraction was exercised", err)
	}
	state, _, err = applyManaged(context.Background(), "receive-handoff", state, persist, f.d)
	if err != nil || !state.Initialized || state.Handoff.State != "received" {
		t.Fatal("partial extraction cannot retry", err)
	}
	after, _, _, err := pierstore.TreeReview(context.Background(), partial)
	if err != nil || before != after {
		t.Fatal("partial attempt was modified during retry", err)
	}
}

func TestRC11CrossDeviceDestinationDurabilityBeforeRemoval(t *testing.T) {
	// Every namespace that can have been created by Parents must be durable,
	// including nested grouping directories and the root's parent.
	for _, failure := range []int{-1, 0, 1, 2, 3, 4, 5, 6} {
		t.Run(fmt.Sprintf("barrier-%d", failure), func(t *testing.T) {
			f, state, persist, _ := externalFixture(t)
			state.Storage.Path = "fakeships/group/sub/received/pier"
			f.d.renamePier = func(string, string) error { return unix.EXDEV }
			target := managedPier(f, state)
			var expected []string
			for p := target; ; p = filepath.Dir(p) {
				expected = append(expected, p)
				if p == f.d.home {
					break
				}
			}
			var synced []string
			unlinks := 0
			f.d.remove = func(root *os.Root, path string) error {
				unlinks++
				if !reflect.DeepEqual(synced, expected) {
					t.Fatal("source removed before full destination publication", synced, expected)
				}
				return root.Remove(path)
			}
			state, _, err := applyExternalImport(context.Background(), state, persist, f.d, func(path string) error {
				synced = append(synced, path)
				if len(synced)-1 == failure {
					return errors.New("injected destination sync failure")
				}
				return pierstore.SyncDirectory(path)
			})
			if failure >= 0 {
				if err == nil || unlinks != 0 || state.ExternalImport.Stage == "complete" || state.ExternalImport.Stage == "removing-source" {
					t.Fatal("failed barrier allowed source removal", err, unlinks)
				}
				_, err = pierstore.Identify(state.ExternalImport.HeldPath)
				rc11Must(t, err)
			} else if err != nil || unlinks == 0 || state.ExternalImport.Stage != "complete" {
				t.Fatal("successful durable copy did not complete", err, unlinks)
			}
		})
	}
}

func TestRC11ImportCompletionPublishesTogether(t *testing.T) {
	for _, mode := range []string{"rename", "copy", "in-place"} {
		t.Run(mode, func(t *testing.T) {
			f, state, persist, source := externalFixture(t)
			if mode == "copy" {
				f.d.renamePier = func(string, string) error { return unix.EXDEV }
			}
			if mode == "in-place" {
				target := managedPier(f, state)
				rc11Must(t, os.MkdirAll(filepath.Dir(target), 0700))
				rc11Must(t, os.Rename(source, target))
				var err error
				state.ExternalImport.Source, err = pierstore.InspectExternal(context.Background(), f.d.home, target)
				rc11Must(t, err)
				state.Adoption.Fingerprint = state.ExternalImport.Source.Fingerprint
			}
			var saved []byte
			returned, _, err := applyManaged(context.Background(), "import-pier", state, func(v ManagedState) error {
				if v.Initialized != (v.ExternalImport.Stage == "complete") {
					t.Fatal("split import completion state published")
				}
				if v.Initialized {
					return errors.New("completion transaction failed")
				}
				saved = rc11Save(t, v)
				return persist(v)
			}, f.d)
			if err == nil || !returned.ImportRecoveryEligible() {
				t.Fatal("failed publication lost recovery", err)
			}
			state = rc11Reload(t, saved)
			before, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, state))
			rc11Must(t, err)
			state, obs, err := applyManaged(context.Background(), "recover-import", state, persist, f.d)
			if err != nil || obs.State != "stopped" || !state.Initialized || state.ExternalImport.Stage != "complete" {
				t.Fatal("completion recovery failed", obs, err)
			}
			after, _, _, err := pierstore.TreeReview(context.Background(), managedPier(f, state))
			if err != nil || before != after || len(f.calls) != 0 {
				t.Fatal("completion recovery modified data or dispatched runtime", err)
			}
		})
	}
}

func TestRC11PruneRechecksNativeLocksBetweenUnlinks(t *testing.T) {
	for _, lock := range []string{".vere.lock", ".urb/vere.lock", ".urb/lock"} {
		for _, after := range []int{0, 1, 4} {
			t.Run(fmt.Sprintf("%s/after-%d", lock, after), func(t *testing.T) {
				f, state, persist := managedPruneFixture(t)
				path := filepath.Join(managedPier(f, state), lock)
				if after == 0 {
					rc11Must(t, os.WriteFile(path, []byte("native lock"), 0600))
				}
				unlinks := 0
				f.d.remove = func(root *os.Root, name string) error {
					unlinks++
					err := root.Remove(name)
					if unlinks == after {
						rc11Must(t, os.WriteFile(path, []byte("native lock"), 0600))
					}
					return err
				}
				state, obs, err := applyManaged(context.Background(), "prune-epochs", state, persist, f.d)
				if err == nil || unlinks != after {
					t.Fatal("native lock ignored", obs, err, unlinks)
				}
				if after > 0 && (!state.UncertainMaintenance() || state.PrefixPrune == nil || !state.PrefixPrune.Uncertain) {
					t.Fatal("partial deletion lost uncertainty")
				}
				_, err = os.Stat(filepath.Join(managedPier(f, state), ".urb/log/0i0"))
				rc11Must(t, err)
			})
		}
	}
}

func TestRC11LegacyImportCompletionEvidenceRecovery(t *testing.T) {
	for _, live := range []bool{false, true} {
		for _, fault := range []string{"", "lock", "replace", "source-reappeared", "publication"} {
			t.Run(fmt.Sprintf("live=%v/%s", live, fault), func(t *testing.T) {
				f, state, persist := retainedImport(t, live)
				var err error
				state, _, err = applyImportRecovery(context.Background(), state, persist, f.d)
				rc11Must(t, err)
				// Historical final enrollment write, with the enclosing completion
				// write missing. JSON round-trip models durable prior-schema state.
				state.ExternalImport.Stage = "enrolling"
				state = rc11Reload(t, rc11Save(t, state))
				if !state.ImportRecoveryEligible() {
					t.Fatal("historical completion not eligible through ordinary recovery admission")
				}
				path := managedPier(f, state)
				finished := *state.Adoption.FinishedAt
				switch fault {
				case "lock":
					rc11Must(t, os.WriteFile(filepath.Join(path, ".urb/lock"), []byte("native"), 0600))
				case "replace":
					rc11Must(t, os.Rename(path, filepath.Join(t.TempDir(), "retained")))
					rc11Must(t, os.MkdirAll(filepath.Join(path, ".urb"), 0700))
				case "source-reappeared":
					rc11Must(t, os.Mkdir(state.ExternalImport.Source.Path, 0700))
				}
				before, _, _, err := pierstore.TreeReview(context.Background(), path)
				rc11Must(t, err)
				writes := 0
				next, obs, err := applyManaged(context.Background(), "recover-import", state, func(v ManagedState) error {
					writes++
					if !v.Initialized || v.ExternalImport.Stage != "complete" {
						t.Fatal("legacy recovery proposed an intermediate state")
					}
					if fault == "publication" {
						return errors.New("publication failed")
					}
					return persist(v)
				}, f.d)
				if fault == "" {
					if err != nil || obs.State != "stopped" || next.ExternalImport.Stage != "complete" || writes != 1 || !next.Adoption.FinishedAt.Equal(finished) {
						t.Fatal("legacy completion recovery failed", obs, err)
					}
				} else if err == nil || !obs.RecoveryRequired || next.ExternalImport.Stage != "enrolling" || !next.Initialized || !next.Adoption.FinishedAt.Equal(finished) {
					t.Fatal("unsafe legacy recovery lost its original state", obs, err)
				}
				after, _, _, err := pierstore.TreeReview(context.Background(), path)
				if err != nil || before != after || len(f.calls) != 0 {
					t.Fatal("legacy recovery mutated data or dispatched runtime", err)
				}
			})
		}
	}
}
