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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

// This fixture enters the real import's durable copying phase, then constructs a
// small partial destination. Every file and engine dependency belongs to the test.
func partialImportRestoreFixture(t *testing.T) (*fakeEngine, ManagedState, string) {
	t.Helper()
	f, state, persist, _ := externalFixture(t)
	f.d.renamePier = func(string, string) error { return unix.EXDEV }
	interrupted := errors.New("copy interrupted before destination creation")
	state, _, err := applyManaged(context.Background(), "import-pier", state, func(s ManagedState) error {
		if s.ExternalImport.Stage == "copying" {
			return interrupted
		}
		return persist(s)
	}, f.d)
	if !errors.Is(err, interrupted) || !state.UncertainProvisioning() {
		t.Fatalf("fixture did not enter retained copying state: %v", err)
	}
	target := managedPier(f, state)
	if err := os.MkdirAll(filepath.Join(target, ".urb/log"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, value := range map[string]string{
		filepath.Join(target, ".urb/log/partial"):                     "partial copy",
		filepath.Join(state.ExternalImport.HeldPath, "retained-note"): "source contents",
	} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return f, state, target
}

func TestImportRestoreReviewIsStableReadOnlyAndEngineIndependent(t *testing.T) {
	for _, stage := range []string{"copying", "verifying"} {
		t.Run(stage, func(t *testing.T) {
			f, state, target := partialImportRestoreFixture(t)
			state.ExternalImport.Stage = stage
			ctx := context.Background()
			before, _ := json.Marshal(state)
			sourceTree, err := reviewRestoreTree(ctx, state.ExternalImport.HeldPath)
			if err != nil {
				t.Fatal(err)
			}
			destinationTree, err := reviewRestoreTree(ctx, filepath.Dir(target))
			if err != nil {
				t.Fatal(err)
			}
			review, err := reviewImportRestore(ctx, state, f.d)
			if err != nil {
				t.Fatal(err)
			}
			if review.Version != 1 || review.RuntimeID != state.ID || len(review.Digest) != 64 ||
				review.OriginalPath != state.ExternalImport.Source.Path || review.Destination != filepath.Dir(target) ||
				review.RetainedPath != filepath.Join(pierstore.Root(f.d.home), "archive", "import-"+state.ID) ||
				!reflect.DeepEqual(sourceTree, review.SourceTree) || !reflect.DeepEqual(destinationTree, review.DestinationTree) {
				t.Fatalf("incorrect review: %+v", review)
			}
			checked, err := checkImportRestoreReview(ctx, state, review.Digest, f.d)
			if err != nil || !reflect.DeepEqual(checked, review) {
				t.Fatal("unstable review", err)
			}
			// The portable review omits ephemeral mount IDs, but preserves its digest.
			encoded, _ := json.Marshal(review)
			var decoded ImportRestoreReview
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if _, err := checkImportRestoreReview(ctx, state, decoded.Digest, f.d); err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(state)
			if string(before) != string(after) || len(f.calls) != 0 || f.downloads != 0 {
				t.Fatal("read-only review changed state or dispatched runtime work")
			}
			for _, path := range []string{review.OriginalPath, review.RetainedPath} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("review moved data", path, err)
				}
			}
		})
	}
}

func TestImportRestoreReviewRejectsChangedReviewedObjects(t *testing.T) {
	for _, fault := range []string{"source-write", "source-remove", "source-add", "destination-write", "destination-remove", "destination-add", "destination-mode", "destination-directory-time", "same-bytes-restored-mtime"} {
		t.Run(fault, func(t *testing.T) {
			f, state, target := partialImportRestoreFixture(t)
			ctx := context.Background()
			review, err := reviewImportRestore(ctx, state, f.d)
			if err != nil {
				t.Fatal(err)
			}
			sourceFile := filepath.Join(state.ExternalImport.HeldPath, "retained-note")
			partial := filepath.Join(target, ".urb/log/partial")
			switch fault {
			case "source-write":
				err = os.WriteFile(sourceFile, []byte("changed source"), 0600)
			case "source-remove":
				err = os.Remove(sourceFile)
			case "source-add":
				err = os.WriteFile(filepath.Join(state.ExternalImport.HeldPath, "new"), []byte("new"), 0600)
			case "destination-write":
				err = os.WriteFile(partial, []byte("changed target"), 0600)
			case "destination-remove":
				err = os.Remove(partial)
			case "destination-add":
				err = os.WriteFile(filepath.Join(target, "new"), []byte("new"), 0600)
			case "destination-mode":
				err = os.Chmod(partial, 0400)
			case "destination-directory-time":
				err = os.Chtimes(filepath.Dir(partial), time.Unix(1, 0), time.Unix(1, 0))
			case "same-bytes-restored-mtime":
				info, e := os.Stat(partial)
				if e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(partial, []byte("partial copy"), 0600); e != nil {
					t.Fatal(e)
				}
				err = os.Chtimes(partial, info.ModTime(), info.ModTime())
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := checkImportRestoreReview(ctx, state, review.Digest, f.d); err == nil {
				t.Fatal("stale review remained usable")
			}
			if len(f.calls) != 0 {
				t.Fatal("stale review dispatched runtime")
			}
			if _, err := os.Stat(state.ExternalImport.HeldPath); err != nil {
				t.Fatal("source lost", err)
			}
			if _, err := os.Stat(target); err != nil {
				t.Fatal("destination lost", err)
			}
		})
	}
}

func TestImportRestoreReviewRejectsUnsafeBindings(t *testing.T) {
	for _, fault := range []string{"original-occupied", "archive-occupied", "source-replaced", "urb-replaced", "mount-replaced", "target-missing", "target-symlink", "source-lock", "destination-lock", "source-symlink", "destination-symlink", "extra-held-child", "extra-mount-child", "archive-symlink", "wrong-held-name", "source-noncanonical"} {
		t.Run(fault, func(t *testing.T) {
			f, state, target := partialImportRestoreFixture(t)
			held := state.ExternalImport.HeldPath
			archive := filepath.Join(pierstore.Root(f.d.home), "archive")
			var err error
			switch fault {
			case "original-occupied":
				err = os.Mkdir(state.ExternalImport.Source.Path, 0700)
			case "archive-occupied":
				err = os.Mkdir(filepath.Join(archive, "import-"+state.ID), 0700)
			case "source-replaced":
				if e := os.Rename(held, held+"-original"); e != nil {
					t.Fatal(e)
				}
				err = os.MkdirAll(filepath.Join(held, ".urb/chk"), 0700)
				writePierEpochPath(t, held, "0i0", false)
			case "urb-replaced":
				if e := os.Rename(filepath.Join(held, ".urb"), filepath.Join(held, "old-urb")); e != nil {
					t.Fatal(e)
				}
				err = os.MkdirAll(filepath.Join(held, ".urb/chk"), 0700)
				writePierEpochPath(t, held, "0i0", false)
			case "mount-replaced":
				if e := os.Rename(filepath.Dir(target), filepath.Dir(target)+"-original"); e != nil {
					t.Fatal(e)
				}
				err = os.MkdirAll(target, 0700)
			case "target-missing":
				err = os.Rename(target, filepath.Join(t.TempDir(), "partial"))
			case "target-symlink":
				if e := os.Rename(target, filepath.Join(t.TempDir(), "partial")); e != nil {
					t.Fatal(e)
				}
				err = os.Symlink(t.TempDir(), target)
			case "source-lock":
				err = os.WriteFile(filepath.Join(held, ".vere.lock"), []byte("busy"), 0600)
			case "destination-lock":
				err = os.WriteFile(filepath.Join(target, ".vere.lock"), []byte("busy"), 0600)
			case "source-symlink":
				err = os.Symlink(t.TempDir(), filepath.Join(held, "link"))
			case "destination-symlink":
				err = os.Symlink(t.TempDir(), filepath.Join(target, "link"))
			case "extra-held-child":
				err = os.WriteFile(filepath.Join(filepath.Dir(held), "other"), []byte("keep"), 0600)
			case "extra-mount-child":
				err = os.WriteFile(filepath.Join(filepath.Dir(target), "other"), []byte("keep"), 0600)
			case "archive-symlink":
				if e := os.Rename(archive, archive+"-original"); e != nil {
					t.Fatal(e)
				}
				err = os.Symlink(archive+"-original", archive)
			case "wrong-held-name":
				state.ExternalImport.HeldPath = held + "-other"
			case "source-noncanonical":
				state.ExternalImport.Source.Path = filepath.Dir(state.ExternalImport.Source.Path) + "/../" + filepath.Base(filepath.Dir(state.ExternalImport.Source.Path)) + "/existing"
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := reviewImportRestore(context.Background(), state, f.d); err == nil {
				t.Fatal("unsafe binding accepted")
			}
			if len(f.calls) != 0 {
				t.Fatal("unsafe review dispatched runtime")
			}
		})
	}
}

func TestImportRestoreReviewRefusesOtherPhasesAndInvalidAuthority(t *testing.T) {
	for _, fault := range []string{"missing-import", "enrolling", "removing-source", "prepared", "initialized", "runtime", "no-adoption", "settled-adoption", "storage-uncertain", "missing-mount-identity", "settled-storage", "adoption-binding", "handoff-staging", "pending-key", "move", "maintenance"} {
		t.Run(fault, func(t *testing.T) {
			f, state, _ := partialImportRestoreFixture(t)
			switch fault {
			case "missing-import":
				state.ExternalImport = nil
			case "enrolling", "removing-source":
				state.ExternalImport.Stage = fault
			case "prepared":
				state.Prepared = true
			case "initialized":
				state.Initialized = true
			case "runtime":
				state.Runtime.RunID = state.ID
			case "no-adoption":
				state.Adoption = nil
			case "settled-adoption":
				state.Adoption.Uncertain = false
			case "storage-uncertain":
				state.Storage.Uncertain = true
			case "missing-mount-identity":
				state.Storage.MountIdentity = pierstore.DirectoryIdentity{}
			case "settled-storage":
				state.Storage.Identity = state.ExternalImport.Source.Identity
			case "adoption-binding":
				state.Adoption.Fingerprint = strings.Repeat("b", 64)
			case "handoff-staging":
				state.HandoffStaging = &ManagedHandoffStaging{}
			case "pending-key":
				state.NativeKeyDigest = strings.Repeat("a", 64)
			case "move":
				state.MoveTarget = "fakeships/elsewhere/zod"
			case "maintenance":
				state.MaintenanceUncertain = true
			}
			if _, err := reviewImportRestore(context.Background(), state, f.d); err == nil {
				t.Fatal("unrelated or uncertain phase accepted")
			}
		})
	}
	f, state, _ := partialImportRestoreFixture(t)
	for _, digest := range []string{"", "short", strings.Repeat("z", 64)} {
		if _, err := checkImportRestoreReview(context.Background(), state, digest, f.d); err == nil {
			t.Fatal("invalid digest accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := reviewImportRestore(ctx, state, f.d); err == nil {
		t.Fatal("cancelled review succeeded")
	}
}

func TestImportRestoreReviewRefusesSourceNestedInPartialDestination(t *testing.T) {
	f, state, target := partialImportRestoreFixture(t)
	inside := filepath.Join(target, ".omabit-import-"+state.ID)
	if err := os.Mkdir(inside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(state.ExternalImport.HeldPath, filepath.Join(inside, "pier")); err != nil {
		t.Fatal(err)
	}
	state.ExternalImport.HeldPath = filepath.Join(inside, "pier")
	state.ExternalImport.Source.Path = filepath.Join(target, "original")
	if _, err := reviewImportRestore(context.Background(), state, f.d); err == nil || !strings.Contains(err.Error(), "must not overlap") {
		t.Fatal("overlapping restoration accepted", err)
	}
}
