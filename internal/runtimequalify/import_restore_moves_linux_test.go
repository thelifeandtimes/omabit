//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func restoredFileHashes(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		value, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[relative] = sha256.Sum256(value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func roundTripRestoreJournal(t *testing.T, journal ImportRestoreJournal) *ImportRestoreJournal {
	t.Helper()
	encoded, err := json.Marshal(journal)
	if err != nil {
		t.Fatal(err)
	}
	var out ImportRestoreJournal
	if err := json.Unmarshal(encoded, &out); err != nil {
		t.Fatal(err)
	}
	return &out
}

func TestImportRestoreMovesPreserveBothTreesAndReplayAcrossEveryBoundary(t *testing.T) {
	cases := []string{"none", "rename-1-before", "rename-1-after", "rename-2-before", "rename-2-after", "cancel-after-intent"}
	for _, stage := range []string{"retaining-destination", "restoring-source", "complete"} {
		cases = append(cases, "persist-"+stage+"-before", "persist-"+stage+"-after")
	}
	for _, fault := range cases {
		t.Run(fault, func(t *testing.T) {
			f, state, target := partialImportRestoreFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			review, err := reviewImportRestore(ctx, state, f.d)
			if err != nil {
				t.Fatal(err)
			}
			sourceBytes := restoredFileHashes(t, review.Source.Path)
			destinationBytes := restoredFileHashes(t, filepath.Dir(target))
			stateBefore, _ := json.Marshal(state)
			var durable *ImportRestoreJournal
			triggered, moves, calls := false, 0, 0
			interrupted := errors.New("simulated interruption")
			persist := func(journal ImportRestoreJournal) error {
				prefix := "persist-" + journal.Stage
				if !triggered && fault == prefix+"-before" {
					triggered = true
					return interrupted
				}
				durable = roundTripRestoreJournal(t, journal)
				if !triggered && fault == prefix+"-after" {
					triggered = true
					return interrupted
				}
				if !triggered && fault == "cancel-after-intent" {
					triggered = true
					cancel()
				}
				return nil
			}
			rename := func(a, b string, object, pa, pb pierstore.DirectoryIdentity, boundary func() error) error {
				calls++
				if durable == nil {
					t.Fatal("rename preceded durable intent")
				}
				prefix := "rename-1"
				if durable.Stage == "restoring-source" {
					prefix = "rename-2"
				}
				if !triggered && fault == prefix+"-before" {
					triggered = true
					return interrupted
				}
				if err := pierstore.RenameBoundDirectory(a, b, object, pa, pb, boundary); err != nil {
					return err
				}
				moves++
				if !triggered && fault == prefix+"-after" {
					triggered = true
					return interrupted
				}
				return nil
			}
			journal, err := restoreImportMoves(ctx, state, review.Digest, nil, persist, f.d, importRestoreIO{rename: rename})
			if fault != "none" {
				if err == nil || !triggered {
					t.Fatal("interruption not exercised", err)
				}
				// Reconstruct only the actually persisted bytes. Never infer that a
				// failed publication made its returned proposal durable.
				journal, err = restoreImportMoves(context.Background(), state, review.Digest, durable, persist, f.d, importRestoreIO{rename: rename})
			}
			if err != nil || journal.Stage != "complete" || durable == nil || durable.Stage != "complete" || moves != 2 {
				t.Fatalf("resume: stage=%s moves=%d error=%v", journal.Stage, moves, err)
			}
			beforeReplay := calls
			if _, err := restoreImportMoves(context.Background(), state, review.Digest, durable, persist, f.d, importRestoreIO{rename: rename}); err != nil {
				t.Fatal(err)
			}
			if calls != beforeReplay {
				t.Fatal("completed replay repeated a move")
			}
			if !reflect.DeepEqual(sourceBytes, restoredFileHashes(t, review.OriginalPath)) || !reflect.DeepEqual(destinationBytes, restoredFileHashes(t, review.RetainedPath)) {
				t.Fatal("restoration lost or changed file bytes")
			}
			identity, err := pierstore.Identify(review.OriginalPath)
			if err != nil || !identity.SameObject(state.ExternalImport.Source.Identity) {
				t.Fatal("restored a different source", err)
			}
			for _, path := range []string{review.Source.Path, review.Destination} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("old launch path remains", path, err)
				}
			}
			entries, err := os.ReadDir(filepath.Dir(review.Source.Path))
			if err != nil || len(entries) != 0 {
				t.Fatal("holding parent not preserved empty", err)
			}
			stateAfter, _ := json.Marshal(state)
			if string(stateBefore) != string(stateAfter) || !state.UncertainProvisioning() || len(f.calls) != 0 || f.downloads != 0 {
				t.Fatal("prototype changed registration/fence or dispatched runtime work")
			}
		})
	}
}

func TestImportRestoreMovesStopOnChangesBetweenDurablePhases(t *testing.T) {
	for _, stage := range []string{"retaining-destination", "restoring-source", "complete"} {
		for _, fault := range []string{"source-write", "destination-write", "source-delete", "destination-delete", "source-replace", "destination-replace", "source-lock", "destination-lock", "occupied-old-source", "occupied-old-destination", "archive-parent-replace"} {
			t.Run(stage+"/"+fault, func(t *testing.T) {
				f, state, _ := partialImportRestoreFixture(t)
				review, err := reviewImportRestore(context.Background(), state, f.d)
				if err != nil {
					t.Fatal(err)
				}
				var durable *ImportRestoreJournal
				stop := errors.New("stop after durable phase")
				persist := func(j ImportRestoreJournal) error {
					durable = roundTripRestoreJournal(t, j)
					if j.Stage == stage {
						return stop
					}
					return nil
				}
				_, err = restoreImportMoves(context.Background(), state, review.Digest, nil, persist, f.d, importRestoreIO{})
				if !errors.Is(err, stop) || durable == nil || durable.Stage != stage {
					t.Fatal("phase not reached", err)
				}
				locations, err := inspectRestoreLocations(context.Background(), *durable)
				if err != nil {
					t.Fatal(err)
				}
				sourceFile := filepath.Join(locations.source, "retained-note")
				destinationFile := filepath.Join(locations.destination, "pier/.urb/log/partial")
				switch fault {
				case "source-write":
					err = os.WriteFile(sourceFile, []byte("new source bytes"), 0600)
				case "destination-write":
					err = os.WriteFile(destinationFile, []byte("new partial bytes"), 0600)
				case "source-delete":
					err = os.Remove(sourceFile)
				case "destination-delete":
					err = os.Remove(destinationFile)
				case "source-replace":
					if e := os.Rename(locations.source, locations.source+"-original"); e != nil {
						t.Fatal(e)
					}
					err = os.Mkdir(locations.source, 0700)
				case "destination-replace":
					if e := os.Rename(locations.destination, locations.destination+"-original"); e != nil {
						t.Fatal(e)
					}
					err = os.MkdirAll(filepath.Join(locations.destination, "pier"), 0700)
				case "source-lock":
					err = os.WriteFile(filepath.Join(locations.source, ".vere.lock"), []byte("lock"), 0600)
				case "destination-lock":
					err = os.WriteFile(filepath.Join(locations.destination, "pier/.vere.lock"), []byte("lock"), 0600)
				case "occupied-old-source":
					other := review.OriginalPath
					if locations.sourceMoved {
						other = review.Source.Path
					}
					err = os.Mkdir(other, 0700)
				case "occupied-old-destination":
					other := review.RetainedPath
					if locations.destinationMoved {
						other = review.Destination
					}
					err = os.Mkdir(other, 0700)
				case "archive-parent-replace":
					p := filepath.Dir(review.RetainedPath)
					if e := os.Rename(p, p+"-original"); e != nil {
						t.Fatal(e)
					}
					err = os.Mkdir(p, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
				dispatched := false
				noMove := func(string, string, pierstore.DirectoryIdentity, pierstore.DirectoryIdentity, pierstore.DirectoryIdentity, func() error) error {
					dispatched = true
					return errors.New("unexpected move")
				}
				if _, err := restoreImportMoves(context.Background(), state, review.Digest, durable, func(ImportRestoreJournal) error { return nil }, f.d, importRestoreIO{rename: noMove}); err == nil {
					t.Fatal("changed retained evidence accepted")
				}
				if dispatched || len(f.calls) != 0 {
					t.Fatal("changed tree dispatched work")
				}
			})
		}
	}
}

func TestImportRestoreMovesRequireBoundDurableAuthority(t *testing.T) {
	f, state, _ := partialImportRestoreFixture(t)
	ctx := context.Background()
	review, err := reviewImportRestore(ctx, state, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restoreImportMoves(ctx, state, review.Digest, nil, nil, f.d, importRestoreIO{}); err == nil {
		t.Fatal("missing journal accepted")
	}
	for _, fault := range []string{"digest", "unknown-stage", "different-import", "path", "manifest"} {
		t.Run(fault, func(t *testing.T) {
			j := roundTripRestoreJournal(t, ImportRestoreJournal{Stage: "retaining-destination", Review: review})
			s := state
			switch fault {
			case "digest":
				j.Review.Digest = strings.Repeat("f", 64)
			case "unknown-stage":
				j.Stage = "invented"
			case "different-import":
				s.ID = strings.Repeat("b", 32)
			case "path":
				j.Review.OriginalPath += "-elsewhere"
			case "manifest":
				j.Review.SourceTree.Entries = j.Review.SourceTree.Entries[:1]
			}
			if _, err := restoreImportMoves(ctx, s, review.Digest, j, func(ImportRestoreJournal) error { return nil }, f.d, importRestoreIO{}); err == nil {
				t.Fatal("invalid journal accepted")
			}
		})
	}
}

func TestImportRestoreMovesRepeatFailedDirectorySyncBeforeRecordingCompletion(t *testing.T) {
	for _, stage := range []string{"retaining-destination", "restoring-source"} {
		t.Run(stage, func(t *testing.T) {
			f, state, _ := partialImportRestoreFixture(t)
			review, err := reviewImportRestore(context.Background(), state, f.d)
			if err != nil {
				t.Fatal(err)
			}
			var durable *ImportRestoreJournal
			lastSynced := ""
			interrupted, resumed := false, false
			persist := func(j ImportRestoreJournal) error {
				if j.Stage == "restoring-source" && lastSynced != "retaining-destination" || j.Stage == "complete" && lastSynced != "restoring-source" {
					t.Fatal("recorded move before parent sync")
				}
				durable = roundTripRestoreJournal(t, j)
				return nil
			}
			io := importRestoreIO{syncParents: func(parents map[string]pierstore.DirectoryIdentity) error {
				if durable == nil {
					t.Fatal("sync without intent")
				}
				if durable.Stage == stage && !interrupted {
					interrupted = true
					return errors.New("directory sync failed after rename")
				}
				if durable.Stage == stage && interrupted {
					resumed = true
				}
				if err := pierstore.SyncBoundDirectories(parents); err != nil {
					return err
				}
				lastSynced = durable.Stage
				return nil
			}}
			_, err = restoreImportMoves(context.Background(), state, review.Digest, nil, persist, f.d, io)
			if err == nil || !interrupted || durable == nil || durable.Stage != stage {
				t.Fatal("sync fault not exercised", err)
			}
			j, err := restoreImportMoves(context.Background(), state, review.Digest, durable, persist, f.d, io)
			if err != nil || !resumed || j.Stage != "complete" {
				t.Fatal("did not resync renamed parents", err)
			}
		})
	}
}
