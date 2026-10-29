//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

type importRestoreRename func(string, string, pierstore.DirectoryIdentity, pierstore.DirectoryIdentity, pierstore.DirectoryIdentity, func() error) error

type importRestoreIO struct {
	rename      importRestoreRename
	syncParents func(map[string]pierstore.DirectoryIdentity) error
}

func restoreReviewDigest(review ImportRestoreReview) (string, error) {
	review.Digest = ""
	encoded, err := json.Marshal(review)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// checkRestoredTree requires the full recorded inventory. Only the root's ctime
// may change after its own rename; descendant metadata, names and objects remain
// exact. Encoding strips ephemeral mount IDs, so a journal round-trip is supported.
// Device renumbering is deliberately refused in this first prototype.
func checkRestoredTree(ctx context.Context, path string, want *pierstore.RemovalTree, moved bool) error {
	if want == nil || len(want.Entries) == 0 {
		return errors.New("missing restoration object inventory")
	}
	got, err := reviewRestoreTree(ctx, path)
	if err != nil {
		return err
	}
	if len(got.Entries) != len(want.Entries) {
		return errors.New("retained tree gained or lost an object")
	}
	if moved {
		got.Entries[0].Changed = want.Entries[0].Changed
	}
	a, err := json.Marshal(got)
	if err != nil {
		return err
	}
	b, err := json.Marshal(want)
	if err != nil {
		return err
	}
	if !bytes.Equal(a, b) {
		return errors.New("retained tree changed or was replaced")
	}
	return nil
}

func oneRestoreLocation(before, after string) (string, bool, error) {
	a, ae := os.Lstat(before)
	b, be := os.Lstat(after)
	if ae == nil && a.IsDir() && errors.Is(be, os.ErrNotExist) {
		return before, false, nil
	}
	if be == nil && b.IsDir() && errors.Is(ae, os.ErrNotExist) {
		return after, true, nil
	}
	return "", false, errors.New("restoration names are occupied, missing or ambiguous; preserve both paths for review")
}

func validateRestoreJournal(state ManagedState, journal ImportRestoreJournal, expected string, d dependencies) error {
	r := journal.Review
	if !importRestoreCandidate(state) || len(expected) != sha256.Size*2 || r.Version != 1 || r.RuntimeID != state.ID || r.Digest != expected {
		return errors.New("restoration journal does not match the approved import")
	}
	digest, err := restoreReviewDigest(r)
	if err != nil || digest != expected {
		return errors.New("restoration journal review is invalid")
	}
	if journal.Stage != "retaining-destination" && journal.Stage != "restoring-source" && journal.Stage != "complete" {
		return errors.New("unknown restoration journal stage")
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return err
	}
	if err := q.accountBoundary(false); err != nil {
		return err
	}
	target, err := pierstore.Absolute(d.home, state.Storage.Path)
	if err != nil {
		return err
	}
	root, _, err := pierstore.Resolve(d.home, state.Storage.Path)
	if err != nil {
		return err
	}
	held := filepath.Join(filepath.Dir(r.OriginalPath), ".omabit-import-"+state.ID, "pier")
	if r.OriginalPath != state.ExternalImport.Source.Path || r.Source.Path != held || held != state.ExternalImport.HeldPath ||
		!r.Source.Identity.SameObject(state.ExternalImport.Source.Identity) || r.Destination != filepath.Dir(target) ||
		r.RetainedPath != filepath.Join(root, "archive", "import-"+state.ID) ||
		r.DestinationTree == nil || len(r.DestinationTree.Entries) < 2 || r.DestinationTree.Entries[0].Identity != state.Storage.MountIdentity ||
		r.DestinationTree.Entries[1].Path != filepath.Base(target) || !r.DestinationTree.Entries[1].Directory {
		return errors.New("restoration paths or objects differ from the import binding")
	}
	return nil
}

type restoreLocations struct {
	source, destination           string
	sourceMoved, destinationMoved bool
}

func inspectRestoreLocations(ctx context.Context, journal ImportRestoreJournal) (restoreLocations, error) {
	var locations restoreLocations
	r := journal.Review
	if err := ctx.Err(); err != nil {
		return locations, err
	}
	parents := map[string]pierstore.DirectoryIdentity{
		filepath.Dir(r.OriginalPath): r.SourceParent,
		filepath.Dir(r.Source.Path):  r.HoldingParent,
		filepath.Dir(r.Destination):  r.DestinationParent,
		filepath.Dir(r.RetainedPath): r.ArchiveParent,
	}
	for path, want := range parents {
		if err := pierstore.HostPath(path); err != nil {
			return locations, err
		}
		got, err := pierstore.IdentifyMount(path)
		if err != nil || got != want {
			return locations, errors.New("restoration parent changed")
		}
	}
	var err error
	locations.source, locations.sourceMoved, err = oneRestoreLocation(r.Source.Path, r.OriginalPath)
	if err != nil {
		return locations, err
	}
	locations.destination, locations.destinationMoved, err = oneRestoreLocation(r.Destination, r.RetainedPath)
	if err != nil {
		return locations, err
	}
	if locations.sourceMoved && !locations.destinationMoved ||
		journal.Stage == "retaining-destination" && locations.sourceMoved ||
		journal.Stage != "retaining-destination" && !locations.destinationMoved ||
		journal.Stage == "complete" && !locations.sourceMoved {
		return locations, errors.New("restoration locations contradict the durable move order")
	}
	children, err := os.ReadDir(filepath.Dir(r.Source.Path))
	if err != nil {
		return locations, err
	}
	if locations.sourceMoved && len(children) != 0 || !locations.sourceMoved && (len(children) != 1 || children[0].Name() != "pier" || !children[0].IsDir()) {
		return locations, errors.New("holding directory changed during restoration")
	}
	if pierstore.Busy(locations.source) || pierstore.Busy(filepath.Join(locations.destination, r.DestinationTree.Entries[1].Path)) {
		return locations, errors.New("a retained pier has a Vere lock; restoration is stopped")
	}
	if err := checkRestoredTree(ctx, locations.source, r.SourceTree, locations.sourceMoved); err != nil {
		return locations, err
	}
	if err := checkRestoredTree(ctx, locations.destination, r.DestinationTree, locations.destinationMoved); err != nil {
		return locations, err
	}
	return locations, nil
}

// restoreImportMoves does only two data-preserving renames. A journal callback
// failure stops work; errors after rename keep the previous durable phase so a
// resumed invocation can recognize exactly that object at its new name. No Docker
// command, source deletion, registration removal or automatic re-import occurs.
func restoreImportMoves(ctx context.Context, state ManagedState, expected string, saved *ImportRestoreJournal, persist func(ImportRestoreJournal) error, d dependencies, io importRestoreIO) (ImportRestoreJournal, error) {
	var journal ImportRestoreJournal
	if persist == nil {
		return journal, errors.New("restoration requires durable move intent")
	}
	if io.rename == nil {
		io.rename = pierstore.RenameBoundDirectory
	}
	if io.syncParents == nil {
		io.syncParents = pierstore.SyncBoundDirectories
	}
	if saved == nil {
		review, err := checkImportRestoreReview(ctx, state, expected, d)
		if err != nil {
			return journal, err
		}
		journal = ImportRestoreJournal{Stage: "retaining-destination", Review: review}
		if err := persist(journal); err != nil {
			return journal, err
		}
	} else {
		journal = *saved
	}
	if err := validateRestoreJournal(state, journal, expected, d); err != nil {
		return journal, err
	}
	boundary := func() error {
		if err := validateRestoreJournal(state, journal, expected, d); err != nil {
			return err
		}
		_, err := inspectRestoreLocations(ctx, journal)
		return err
	}
	locations, err := inspectRestoreLocations(ctx, journal)
	if err != nil {
		return journal, err
	}
	r := journal.Review
	if journal.Stage == "retaining-destination" {
		if !locations.destinationMoved {
			err := io.rename(r.Destination, r.RetainedPath, r.DestinationTree.Entries[0].Identity, r.DestinationParent, r.ArchiveParent, boundary)
			if err != nil {
				return journal, err
			}
		}
		locations, err = inspectRestoreLocations(ctx, journal)
		if err != nil || !locations.destinationMoved {
			if err == nil {
				err = errors.New("partial destination was not retained")
			}
			return journal, err
		}
		// A prior invocation may have renamed but failed during directory sync.
		// Reestablish durability even when the new name was already observed.
		if err := io.syncParents(map[string]pierstore.DirectoryIdentity{filepath.Dir(r.Destination): r.DestinationParent, filepath.Dir(r.RetainedPath): r.ArchiveParent}); err != nil {
			return journal, err
		}
		if err := boundary(); err != nil {
			return journal, err
		}
		next := journal
		next.Stage = "restoring-source"
		if err := persist(next); err != nil {
			return journal, err
		}
		journal = next
	}
	if journal.Stage == "restoring-source" {
		locations, err = inspectRestoreLocations(ctx, journal)
		if err != nil {
			return journal, err
		}
		if !locations.sourceMoved {
			err := io.rename(r.Source.Path, r.OriginalPath, r.SourceTree.Entries[0].Identity, r.HoldingParent, r.SourceParent, boundary)
			if err != nil {
				return journal, err
			}
		}
		locations, err = inspectRestoreLocations(ctx, journal)
		if err != nil || !locations.sourceMoved {
			if err == nil {
				err = errors.New("held source was not restored")
			}
			return journal, err
		}
		if err := io.syncParents(map[string]pierstore.DirectoryIdentity{filepath.Dir(r.Source.Path): r.HoldingParent, filepath.Dir(r.OriginalPath): r.SourceParent}); err != nil {
			return journal, err
		}
		if err := boundary(); err != nil {
			return journal, err
		}
		next := journal
		next.Stage = "complete"
		if err := persist(next); err != nil {
			return journal, err
		}
		journal = next
	}
	return journal, boundary()
}
