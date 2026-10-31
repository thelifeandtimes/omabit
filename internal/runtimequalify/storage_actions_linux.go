//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

func StorageRoots() ([]pierstore.StorageRoot, error) { return pierstore.Roots(defaults().home) }
func RegisterStorageRoot(id, path string) (pierstore.StorageRoot, error) {
	return pierstore.RegisterRoot(defaults().home, id, path)
}
func InspectExternalPier(ctx context.Context, path string) (pierstore.ExternalSource, error) {
	return inspectExternalPier(ctx, path, defaults())
}

// Validate every import route, including the legacy CLI without --identify,
// before mount preparation or moving/removing source data (adapted from PR #3).
func inspectExternalPier(ctx context.Context, path string, d dependencies) (pierstore.ExternalSource, error) {
	source, err := pierstore.InspectExternal(ctx, d.home, path)
	if err != nil {
		return pierstore.ExternalSource{}, err
	}
	q := &qualifier{d: d}
	if err = q.accountBoundary(false); err != nil {
		return pierstore.ExternalSource{}, err
	}
	if _, err = q.inspectEpochInventoryAt(ctx, path, true); err != nil {
		return pierstore.ExternalSource{}, err
	}
	after, err := pierstore.InspectExternal(ctx, d.home, path)
	if err != nil {
		return pierstore.ExternalSource{}, err
	}
	if source != after {
		return pierstore.ExternalSource{}, errors.New("source changed during epoch validation")
	}
	return source, nil
}

func applyPierDeletion(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || state.Storage == nil || state.Handoff.Fenced() || state.ExternalImportInFlight() {
		return state, unknown, errors.New("pier deletion requires exclusive, settled storage")
	}
	for _, c := range state.Runtime.Containers {
		if !c.Removed || c.CreateUncertain {
			return state, unknown, errors.New("explicitly stop and remove all runtime containers before deleting")
		}
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err = q.accountBoundary(false); err != nil {
		return state, unknown, err
	}
	if state.Prepared && q.r.Network != nil && !q.r.Network.Removed {
		if err = q.verifyEngine(ctx); err != nil {
			return state, unknown, err
		}
		if err = q.cleanupNetwork(); err != nil {
			return state, unknown, err
		}
		state.Runtime = q.r
	}
	if state.Deletion == nil {
		state.Deletion = &ManagedDeletion{Stage: "review", Mount: state.Storage.MountIdentity}
		if err = persist(state); err != nil {
			return state, unknown, err
		}
	}
	intent := state.Deletion
	source, err := pierstore.Absolute(d.home, state.Storage.Path)
	if err != nil {
		return state, unknown, err
	}
	storageRoot, _, err := pierstore.Resolve(d.home, state.Storage.Path)
	if err != nil {
		return state, unknown, err
	}
	trash := filepath.Join(storageRoot, "archive", "deleted-"+state.ID)
	if intent.Stage == "review" {
		if intent.Mount.Inode == 0 {
			if _, err = os.Lstat(filepath.Dir(source)); !errors.Is(err, os.ErrNotExist) {
				return state, unknown, errors.New("uninitialized registration has unrecorded storage; it will not be deleted")
			}
			intent.Stage = "complete"
		} else {
			mount, e := pierstore.CheckMount(d.home, *state.Storage)
			if e != nil {
				return state, unknown, e
			}
			if pierstore.Busy(source) {
				return state, unknown, errors.New("pier still has a Vere lock")
			}
			if state.Initialized || state.Storage.Identity.Inode != 0 {
				if _, e = pierstore.Check(d.home, *state.Storage); e != nil {
					return state, unknown, e
				}
			} else if children, err := os.ReadDir(mount); err != nil || len(children) != 0 {
				return state, unknown, errors.New("uninitialized mount contains an unrecorded child; retained for review")
			}
			if intent.Tree, e = pierstore.ReviewRemoval(ctx, mount); e != nil {
				return state, unknown, e
			}
			if _, e = pierstore.IdentifyMount(filepath.Dir(trash)); e != nil {
				return state, unknown, e
			}
			intent.Stage = "renaming"
			if err = persist(state); err != nil {
				return state, unknown, err
			}
		}
	}
	if intent.Stage == "renaming" {
		if _, e := os.Lstat(filepath.Dir(source)); errors.Is(e, os.ErrNotExist) {
			actual, e := pierstore.IdentifyMount(trash)
			if e != nil || !actual.SameObject(intent.Mount) {
				return state, unknown, errors.New("deletion rename outcome needs review")
			}
			if err = deletionTreeBoundary(ctx, &state, trash, filepath.Base(source), persist); err != nil {
				return state, unknown, err
			}
		} else {
			if _, e := pierstore.CheckMount(d.home, *state.Storage); e != nil {
				return state, unknown, e
			}
			if err = deletionTreeBoundary(ctx, &state, filepath.Dir(source), filepath.Base(source), persist); err != nil {
				return state, unknown, err
			}
			if err = pierstore.RenameDirectory(filepath.Dir(source), trash); err != nil {
				return state, unknown, err
			}
		}
		intent.Stage = "removing"
		if err = persist(state); err != nil {
			return state, unknown, err
		}
	}
	if intent.Stage == "removing" {
		actual, e := pierstore.IdentifyMount(trash)
		if e == nil {
			if !actual.SameObject(intent.Mount) {
				return state, unknown, errors.New("deletion directory was replaced")
			}
			if err = deletionTreeBoundary(ctx, &state, trash, filepath.Base(source), persist); err != nil {
				return state, unknown, err
			}
			err = intent.Tree.Remove(ctx, trash, func() error {
				if pierstore.Busy(filepath.Join(trash, filepath.Base(source))) {
					return errors.New("pier acquired a Vere lock during deletion")
				}
				return nil
			}, d.remove)
			if err != nil {
				return state, unknown, err
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return state, unknown, e
		}
		if err = SyncHandoffDirectory(filepath.Dir(trash)); err != nil {
			return state, unknown, err
		}
		intent.Stage = "complete"
	}
	if intent.Stage != "complete" {
		return state, unknown, errors.New("invalid deletion phase")
	}
	if state.NativeKeyDigest != "" {
		q.r.RunID = state.ID
		key := q.keyPath()
		if _, err = os.Lstat(key); err == nil {
			if err = q.verifyBootKey(); err != nil {
				return state, unknown, err
			}
			if err = os.Remove(key); err != nil {
				return state, unknown, err
			}
			if err = SyncHandoffDirectory(filepath.Dir(key)); err != nil {
				return state, unknown, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return state, unknown, err
		}
		state.NativeKeyDigest = ""
	}
	if state.Prepared {
		if _, err = os.Lstat(state.Runtime.Directory); err == nil {
			if _, _, _, err = pierstore.TreeReview(ctx, state.Runtime.Directory); err != nil {
				return state, unknown, err
			}
			root, e := os.OpenRoot(q.base)
			if e != nil {
				return state, unknown, e
			}
			err = root.RemoveAll(state.ID)
			root.Close()
			if err != nil {
				return state, unknown, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return state, unknown, err
		}
		if err = SyncHandoffDirectory(q.base); err != nil {
			return state, unknown, err
		}
	}
	state.Runtime.Stage = "deleted"
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	return state, ManagedObservation{State: "deleted", Health: "unknown"}, nil
}

func deletionTreeBoundary(ctx context.Context, state *ManagedState, mount, leaf string, persist func(ManagedState) error) error {
	intent := state.Deletion
	actual, err := pierstore.IdentifyMount(mount)
	if err != nil || !actual.SameObject(intent.Mount) {
		return errors.New("deletion mount was replaced")
	}
	path := filepath.Join(mount, leaf)
	if pierstore.Busy(path) {
		return errors.New("pier still has a Vere lock")
	}
	if state.Storage.Identity.Inode == 0 {
		children, err := os.ReadDir(mount)
		if err != nil || len(children) != 0 || intent.Tree != nil && len(intent.Tree.Entries) != 1 {
			return errors.New("deletion has no recorded child identity; only an empty mount may be removed")
		}
		if intent.Tree == nil {
			intent.Tree, err = pierstore.ReviewRemoval(ctx, mount)
			if err != nil {
				return err
			}
			if err = persist(*state); err != nil {
				return err
			}
		}
	}
	if intent.Tree == nil {
		// Pre-schema20 journals have no per-object manifest. Upgrade only while
		// the complete recorded child identity still exists, including .urb.
		// Never invent ownership from a remaining/replacement subtree.
		id, err := pierstore.Identify(path)
		if err != nil || state.Storage.Identity.Inode == 0 || !id.SameObject(state.Storage.Identity) {
			return errors.New("legacy deletion needs its recorded pier identity before continuation")
		}
		children, err := os.ReadDir(mount)
		if err != nil || len(children) != 1 || children[0].Name() != leaf || !children[0].IsDir() {
			return errors.New("legacy deletion mount has unexplained children")
		}
		intent.Tree, err = pierstore.ReviewRemoval(ctx, mount)
		if err != nil {
			return err
		}
		if err = persist(*state); err != nil {
			return err
		}
	}
	if recorded := state.Storage.Identity; recorded.Inode != 0 {
		// Bind the manifest itself to the enrolled child, not just whatever
		// happened to be present when the tree traversal ran. Keep this evidence
		// after those directories have been unlinked in a partial deletion.
		want := map[string]pierstore.DirectoryIdentity{
			leaf:                        {Device: recorded.Device, Inode: recorded.Inode, Birth: recorded.Birth},
			filepath.Join(leaf, ".urb"): {Device: recorded.UrbDevice, Inode: recorded.UrbInode, Birth: recorded.UrbBirth},
		}
		for _, entry := range intent.Tree.Entries {
			if id, exists := want[entry.Path]; exists && entry.Directory && entry.Identity.SameObject(id) {
				delete(want, entry.Path)
			}
		}
		if len(want) != 0 {
			return errors.New("deletion manifest does not bind the registered pier")
		}
	}
	return intent.Tree.Check(ctx, mount)
}

func applyExternalImport(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies, destinationSync ...func(string) error) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	intent := state.ExternalImport
	if persist == nil || intent == nil || intent.Stage != "reviewed" || state.Prepared || state.Storage == nil || state.Adoption == nil {
		return state, unknown, errors.New("import requires its reviewed, unstarted source intent")
	}
	ctx, cancel := context.WithTimeout(ctx, localImportTimeout(intent.Source.Bytes))
	defer cancel()
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err = q.accountBoundary(false); err != nil {
		return state, unknown, err
	}
	source, err := inspectExternalPier(ctx, intent.Source.Path, d)
	if err != nil || source != intent.Source {
		return state, unknown, errors.New("source changed or became inaccessible since import review; nothing was moved")
	}
	if intent.Identify {
		identity, e := InspectPierIdentity(ctx, source.Path)
		ship, kind := "~zod", "fake"
		if state.Fake != nil {
			ship = state.Fake.Ship
		}
		if state.Comet != nil {
			ship = state.Comet.Ship
			kind = protocol.ShipClass(ship)
		}
		if e != nil || identity.Ship != ship || identity.Kind != kind {
			return state, unknown, errors.New("source identity or network changed before moving")
		}
	}
	target, err := pierstore.Absolute(d.home, state.Storage.Path)
	if err != nil {
		return state, unknown, err
	}
	stage := func(name string) error {
		intent.Stage = name
		if len(intent.Events) == 0 || intent.Events[len(intent.Events)-1].Stage != name {
			intent.Events = append(intent.Events, protocol.ImportEvent{Stage: name, At: time.Now().UTC()})
			if len(intent.Events) > 16 {
				intent.Events = intent.Events[len(intent.Events)-16:]
			}
		}
		return persist(state)
	}
	if filepath.Dir(source.Path) == filepath.Dir(target) {
		// A reviewed pier already under its chosen exclusive parent can be
		// enrolled in place, or atomically renamed from "pier" to its @p.
		mount, e := pierstore.IdentifyMount(filepath.Dir(source.Path))
		if e != nil {
			return state, unknown, e
		}
		id, relative := pierstore.Split(state.Storage.Path)
		old := pierstore.Location{Path: pierstore.Qualify(id, filepath.Join(filepath.Dir(relative), filepath.Base(source.Path))), Identity: source.Identity, MountIdentity: mount}
		if _, e = pierstore.Check(d.home, old); e != nil {
			return state, unknown, e
		}
		state.Storage.MountIdentity = mount
	} else if _, err = pierstore.PrepareMount(d.home, state.Storage); err != nil {
		return state, unknown, err
	}
	if source.Path == target {
		state.Storage.Identity = source.Identity
		state.Adoption.Fingerprint = pierstore.Fingerprint(source.Identity)
		if err = stage("enrolling"); err != nil {
			return state, unknown, err
		}
		return applyPierEnrollment(ctx, state, persist, d)
	}
	_, disk, err := d.admission(filepath.Dir(target))
	if err != nil || disk < 8<<30 {
		return state, unknown, errors.New("import needs 8 GiB free on the destination filesystem")
	}
	parent, err := os.Stat(filepath.Dir(target))
	if err != nil {
		return state, unknown, err
	}
	if device, ok := parent.Sys().(*syscall.Stat_t); !ok || device.Dev != source.Identity.Device && disk < uint64(source.Bytes)+(8<<30) {
		return state, unknown, errors.New("cross-filesystem import needs source size plus 8 GiB free on the destination")
	}
	// Remove the original launch path before any copy. The retained source lives
	// in a private holding directory until the target has been content-verified.
	heldDir := filepath.Join(filepath.Dir(source.Path), ".omabit-import-"+state.ID)
	intent.HeldPath = filepath.Join(heldDir, "pier")
	if err = stage("holding-source"); err != nil {
		return state, unknown, err
	}
	if err = os.Mkdir(heldDir, 0700); err != nil {
		return state, unknown, err
	}
	if err = pierstore.RenameDirectory(source.Path, intent.HeldPath); err != nil {
		return state, unknown, err
	}
	identity, err := pierstore.Identify(intent.HeldPath)
	if err != nil || identity != source.Identity || pierstore.Busy(intent.HeldPath) {
		return state, unknown, errors.New("held source identity changed")
	}
	if err = stage("moving"); err != nil {
		return state, unknown, err
	}
	if err = pierstore.SyncDestinationParents(d.home, *state.Storage, nil); err != nil {
		return state, unknown, err
	}
	rename := d.renamePier
	if rename == nil {
		rename = pierstore.RenameDirectory
	}
	err = rename(intent.HeldPath, target)
	if errors.Is(err, unix.EXDEV) {
		if disk < uint64(source.Bytes)+(8<<30) {
			return state, unknown, errors.New("cross-filesystem import needs source size plus 8 GiB free; source remains in its holding directory")
		}
		if err = stage("copying"); err != nil {
			return state, unknown, err
		}
		if err = os.Mkdir(target, 0700); err != nil {
			return state, unknown, err
		}
		a, e := os.OpenRoot(intent.HeldPath)
		if e != nil {
			return state, unknown, e
		}
		defer a.Close()
		b, e := os.OpenRoot(target)
		if e != nil {
			return state, unknown, e
		}
		defer b.Close()
		// Freeze deletion authority before copying. A later review must not
		// silently authorize newer source bytes that the destination never got.
		removal, e := pierstore.ReviewRemoval(ctx, intent.HeldPath)
		if e != nil {
			return state, unknown, e
		}
		lastReport := time.Time{}
		copied, err := piercopy.Copy(ctx, a, b, func(files int, bytes int64) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			intent.CopiedFiles, intent.CopiedBytes = files, bytes
			if files != source.Files && time.Since(lastReport) < 250*time.Millisecond {
				return nil
			}
			lastReport = time.Now()
			return persist(state)
		}, func() error { return stage("verifying") })
		if err != nil {
			return state, unknown, err
		}
		var sync func(string) error
		if len(destinationSync) != 0 {
			sync = destinationSync[0]
		}
		if err = pierstore.SyncDestination(d.home, *state.Storage, sync); err != nil {
			return state, unknown, err
		}
		if err = stage("removing-source"); err != nil {
			return state, unknown, err
		}
		identity, e := pierstore.Identify(intent.HeldPath)
		if e != nil || !identity.SameObject(source.Identity) || pierstore.Busy(intent.HeldPath) {
			return state, unknown, errors.New("held source changed before removal")
		}
		if err = removal.Check(ctx, intent.HeldPath); err != nil {
			return state, unknown, err
		}
		still, e := piercopy.Measure(ctx, a)
		if e != nil || still != copied {
			return state, unknown, errors.New("held source content changed after copy verification; retained for review")
		}
		published, e := piercopy.Measure(ctx, b)
		if e != nil || published != copied {
			return state, unknown, errors.New("destination content changed after copy verification; source retained")
		}
		for root, path := range map[*os.Root]string{a: intent.HeldPath, b: target} {
			opened, e := root.Stat(".")
			current, statErr := os.Lstat(path)
			if e != nil || statErr != nil || !os.SameFile(opened, current) {
				return state, unknown, errors.New("verified import directory moved or was replaced before source removal")
			}
		}
		if _, err = pierstore.CheckMount(d.home, *state.Storage); err != nil {
			return state, unknown, err
		}
		err = removal.Remove(ctx, intent.HeldPath, func() error {
			if pierstore.Busy(intent.HeldPath) {
				return errors.New("held source acquired a Vere lock")
			}
			return nil
		}, d.remove)
		if err != nil {
			return state, unknown, err
		}
	} else if err != nil {
		return state, unknown, err
	}
	if err = os.Remove(heldDir); err != nil {
		return state, unknown, err
	}
	if err = SyncHandoffDirectory(filepath.Dir(source.Path)); err != nil {
		return state, unknown, err
	}
	state.Storage.Identity, err = pierstore.Identify(target)
	if err != nil {
		return state, unknown, err
	}
	state.Adoption.Fingerprint = pierstore.Fingerprint(state.Storage.Identity)
	if err = stage("enrolling"); err != nil {
		return state, unknown, err
	}
	return applyPierEnrollment(ctx, state, persist, d)
}
