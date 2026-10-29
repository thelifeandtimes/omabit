//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func absentRestorePath(path string) error {
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.New("restore destination already exists or cannot be inspected; preserve it for review")
	}
	return nil
}

// reviewImportRestore reads only paths from the retained import record and its
// registered storage root. Existing user files and private state are never edited.
// The containing runner must serialize this review with instance operations.
func reviewImportRestore(ctx context.Context, state ManagedState, d dependencies) (ImportRestoreReview, error) {
	var out ImportRestoreReview
	if !importRestoreCandidate(state) {
		return out, errors.New("source restoration requires an unstarted partial external copy with retained source evidence")
	}
	q, err := managedSession(&state, nil, d)
	if err != nil {
		return out, err
	}
	if err := q.accountBoundary(false); err != nil {
		return out, err
	}
	intent := state.ExternalImport
	if !filepath.IsAbs(intent.Source.Path) || filepath.Clean(intent.Source.Path) != intent.Source.Path {
		return out, errors.New("original source path is not canonical")
	}
	parent := filepath.Dir(intent.Source.Path)
	heldParent := filepath.Join(parent, ".omabit-import-"+state.ID)
	if intent.HeldPath != filepath.Join(heldParent, "pier") || intent.Source.Identity.Inode == 0 || intent.Source.Identity.UrbInode == 0 {
		return out, errors.New("held source does not match the recorded import binding")
	}
	for _, path := range []string{parent, heldParent} {
		if err := pierstore.HostPath(path); err != nil {
			return out, err
		}
	}
	if err := absentRestorePath(intent.Source.Path); err != nil {
		return out, err
	}
	source, err := inspectExternalPier(ctx, intent.HeldPath, d)
	if err != nil {
		return out, err
	}
	if !source.Identity.SameObject(intent.Source.Identity) {
		return out, errors.New("held source was replaced; restoration cannot adopt another pier")
	}
	children, err := os.ReadDir(heldParent)
	if err != nil || len(children) != 1 || children[0].Name() != "pier" || !children[0].IsDir() {
		return out, errors.New("source holding directory contains unexplained entries")
	}
	sourceParent, err := pierstore.IdentifyMount(parent)
	if err != nil {
		return out, err
	}
	holdingParent, err := pierstore.IdentifyMount(heldParent)
	if err != nil {
		return out, err
	}
	if sourceParent.Device != source.Identity.Device || holdingParent.Device != source.Identity.Device {
		return out, errors.New("source restoration must stay on its recorded filesystem")
	}
	mount, err := pierstore.CheckMount(d.home, *state.Storage)
	if err != nil {
		return out, err
	}
	target, err := pierstore.Absolute(d.home, state.Storage.Path)
	if err != nil {
		return out, err
	}
	if pierstore.Busy(target) {
		return out, errors.New("partial destination has a Vere lock; it cannot be restored or moved")
	}
	destinationParent, err := pierstore.IdentifyMount(filepath.Dir(mount))
	if err != nil {
		return out, err
	}
	root, _, err := pierstore.Resolve(d.home, state.Storage.Path)
	if err != nil {
		return out, err
	}
	archive := filepath.Join(root, "archive")
	if err := pierstore.HostPath(archive); err != nil {
		return out, err
	}
	archiveParent, err := pierstore.IdentifyMount(archive)
	if err != nil {
		return out, err
	}
	if archiveParent.Device != state.Storage.MountIdentity.Device || destinationParent.Device != state.Storage.MountIdentity.Device {
		return out, errors.New("retaining the partial destination must use a same-filesystem rename")
	}
	retained := filepath.Join(archive, "import-"+state.ID)
	paths := []string{intent.Source.Path, intent.HeldPath, mount, retained}
	for i, path := range paths {
		for _, other := range paths[:i] {
			if pierstore.Overlap(path, other) {
				return out, errors.New("restoration trees must not overlap")
			}
		}
	}
	if err := absentRestorePath(retained); err != nil {
		return out, err
	}
	sourceTree, err := reviewRestoreTree(ctx, intent.HeldPath)
	if err != nil {
		return out, err
	}
	destinationTree, err := reviewRestoreTree(ctx, mount)
	if err != nil {
		return out, err
	}
	// An absent partial child is a different failure phase, not this plan.
	if len(destinationTree.Entries) < 2 || destinationTree.Entries[1].Path != filepath.Base(target) || !destinationTree.Entries[1].Directory {
		return out, errors.New("partial destination child is missing or changed")
	}
	// Recheck both reviewed trees and parent/name evidence after traversal. This
	// is observation only; a later apply must check the digest again before intent.
	after, err := inspectExternalPier(ctx, intent.HeldPath, d)
	if err != nil || after != source {
		return out, errors.New("held source changed during restoration review")
	}
	for path, want := range map[string]*pierstore.RemovalTree{intent.HeldPath: sourceTree, mount: destinationTree} {
		got, err := reviewRestoreTree(ctx, path)
		if err != nil || !reflect.DeepEqual(got, want) {
			return out, errors.New("retained tree changed during restoration review")
		}
	}
	for path, want := range map[string]pierstore.DirectoryIdentity{parent: sourceParent, heldParent: holdingParent, mount: state.Storage.MountIdentity, filepath.Dir(mount): destinationParent, archive: archiveParent} {
		got, err := pierstore.IdentifyMount(path)
		if err != nil || got != want {
			return out, errors.New("restoration parent changed during review")
		}
	}
	for _, path := range []string{intent.Source.Path, retained} {
		if err := absentRestorePath(path); err != nil {
			return out, err
		}
	}
	out = ImportRestoreReview{Version: 1, RuntimeID: state.ID, OriginalPath: intent.Source.Path, Source: source,
		SourceParent: sourceParent, HoldingParent: holdingParent, SourceTree: sourceTree,
		Destination: mount, DestinationParent: destinationParent, DestinationTree: destinationTree,
		ArchiveParent: archiveParent, RetainedPath: retained}
	out.Digest, err = restoreReviewDigest(out)
	if err != nil {
		return ImportRestoreReview{}, err
	}
	return out, nil
}

func checkImportRestoreReview(ctx context.Context, state ManagedState, expected string, d dependencies) (ImportRestoreReview, error) {
	if len(expected) != sha256.Size*2 {
		return ImportRestoreReview{}, errors.New("a complete restoration review digest is required")
	}
	review, err := reviewImportRestore(ctx, state, d)
	if err != nil {
		return ImportRestoreReview{}, err
	}
	if review.Digest != expected {
		return ImportRestoreReview{}, errors.New("restoration review changed; inspect both retained trees again")
	}
	return review, nil
}

// RemovalTree.Check intentionally accepts missing entries during deletion. A
// restoration review needs a complete, unchanged inventory instead. Bracket the
// object manifest with complete metadata reviews, including directory timestamps,
// then compare full manifests at subsequent observation boundaries. These are
// change-detection checks, not exclusion of concurrent writers or file snapshots.
func reviewRestoreTree(ctx context.Context, path string) (*pierstore.RemovalTree, error) {
	before, _, _, err := pierstore.TreeReview(ctx, path)
	if err != nil {
		return nil, err
	}
	tree, err := pierstore.ReviewRemoval(ctx, path)
	if err != nil {
		return nil, err
	}
	after, _, _, err := pierstore.TreeReview(ctx, path)
	if err != nil || after != before {
		return nil, errors.New("retained tree changed while recording its objects")
	}
	return tree, nil
}
