//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

func StorageRoots() ([]pierstore.StorageRoot, error) { return pierstore.Roots(defaults().home) }
func RegisterStorageRoot(id, path string) (pierstore.StorageRoot, error) {
	return pierstore.RegisterRoot(defaults().home, id, path)
}
func InspectExternalPier(ctx context.Context, path string) (pierstore.ExternalSource, error) {
	return inspectExternalPier(ctx, path, defaults())
}

func inspectExternalPier(ctx context.Context, path string, d dependencies) (pierstore.ExternalSource, error) {
	source, err := pierstore.InspectExternal(ctx, d.home, path)
	if err != nil {
		return pierstore.ExternalSource{}, err
	}
	q := &qualifier{d: d}
	if err := q.accountBoundary(false); err != nil {
		return pierstore.ExternalSource{}, err
	}
	if _, err := q.inspectEpochsAt(ctx, path); err != nil {
		return pierstore.ExternalSource{}, err
	}
	// Bind compatibility to the same unchanged tree that the caller reviewed.
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
			if state.Initialized {
				if _, e = pierstore.Check(d.home, *state.Storage); e != nil {
					return state, unknown, e
				}
			}
			if _, _, _, e = pierstore.TreeReview(ctx, mount); e != nil {
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
		} else {
			if _, e := pierstore.CheckMount(d.home, *state.Storage); e != nil {
				return state, unknown, e
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
			if _, _, _, e = pierstore.TreeReview(ctx, trash); e != nil {
				return state, unknown, e
			}
			root, e := os.OpenRoot(filepath.Dir(trash))
			if e != nil {
				return state, unknown, e
			}
			err = root.RemoveAll(filepath.Base(trash))
			root.Close()
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

// absentImportPaths conservatively requires both historical names to be absent.
// Even an unrelated replacement needs human review: the journal cannot prove
// that it is unrelated. Never follow symlinks or cross mounts to infer absence.
func absentImportPaths(state ManagedState) error {
	intent := state.ExternalImport
	if intent == nil || !filepath.IsAbs(intent.Source.Path) || filepath.Clean(intent.Source.Path) != intent.Source.Path || intent.Source.Path == "/" ||
		intent.HeldPath != filepath.Join(filepath.Dir(intent.Source.Path), ".omabit-import-"+state.ID, "pier") {
		return errors.New("source/holding evidence paths require review")
	}
	root, err := unix.Open("/", unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	// External sources may live on mounted disks. Traverse without symlinks;
	// once on the recorded source device, disallow every further mount crossing
	// (including same-device bind mounts) through the parent and evidence paths.
	// Legacy journals lack parent inode/mount IDs. Device renumbering therefore
	// needs manual review rather than weakening this absence check.
	parent := root
	defer func() { unix.Close(parent) }()
	var st unix.Stat_t
	if err := unix.Fstat(parent, &st); err != nil {
		return err
	}
	for _, part := range strings.Split(filepath.Dir(intent.Source.Path)[1:], "/") {
		if part == "" {
			continue
		}
		resolve := uint64(unix.RESOLVE_NO_SYMLINKS)
		if st.Dev == intent.Source.Identity.Device {
			resolve |= unix.RESOLVE_NO_XDEV
		}
		next, err := unix.Openat2(parent, part, &unix.OpenHow{
			Flags:   unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC,
			Resolve: resolve,
		})
		if err != nil {
			return errors.New("source parent is unavailable or unsafe")
		}
		unix.Close(parent)
		parent = next
		if err := unix.Fstat(parent, &st); err != nil {
			return err
		}
	}
	if st.Dev != intent.Source.Identity.Device {
		return errors.New("source parent filesystem changed; review absence evidence")
	}
	for _, path := range []string{filepath.Base(intent.Source.Path), filepath.Join(filepath.Base(filepath.Dir(intent.HeldPath)), "pier")} {
		fd, err := unix.Openat2(parent, path, &unix.OpenHow{
			Flags:   unix.O_PATH | unix.O_CLOEXEC | unix.O_NOFOLLOW,
			Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
		})
		if err == nil {
			unix.Close(fd)
			return errors.New("source/holding path exists; review retained or recreated data without removing it")
		}
		if !errors.Is(err, unix.ENOENT) {
			return errors.New("source/holding absence cannot be established without unsafe or inaccessible traversal")
		}
	}
	return nil
}

// applyImportRecovery only reads retained evidence. It never recreates, renames
// or removes source/holding paths, nor runs Docker, Vere or preparation/bootstrap.
func applyImportRecovery(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || !state.ImportRecoveryEligible() {
		return state, unknown, errors.New("recovery requires an interrupted prepared external enrollment with retained identity")
	}
	if err := absentImportPaths(state); err != nil {
		return state, unknown, err
	}
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	if err = q.accountBoundary(false); err != nil {
		return state, unknown, err
	}
	for _, path := range append(q.storagePaths(), q.r.Directory, q.config) {
		if err = q.owned(path, os.ModeDir, path == q.r.Directory || path == q.config); err != nil {
			return state, unknown, err
		}
	}
	path, err := pierstore.Check(d.home, *state.Storage)
	if err != nil {
		return state, unknown, err
	}
	if pierstore.Busy(path) {
		return state, unknown, errors.New("retained destination has a Vere lock")
	}
	before, _, _, err := pierstore.TreeReview(ctx, path)
	if err != nil {
		return state, unknown, err
	}
	if err = pierstore.CheckStateFiles(ctx, path); err != nil {
		return state, unknown, err
	}
	inv, err := q.inspectEpochs(ctx)
	if err != nil {
		return state, unknown, err
	}
	after, _, _, err := pierstore.TreeReview(ctx, path)
	if err != nil {
		return state, unknown, err
	}
	if before != after || pierstore.Busy(path) {
		return state, unknown, errors.New("retained destination changed during recovery")
	}
	if _, err = pierstore.Check(d.home, *state.Storage); err != nil {
		return state, unknown, err
	}
	if err = ctx.Err(); err != nil {
		return state, unknown, err
	}
	if err = absentImportPaths(state); err != nil {
		return state, unknown, err
	}
	// Offer completion to the journal. The managed store stages this callback
	// without clearing durable fences, then commits state, observation and the
	// succeeded receipt together in finish. A rejected callback stays fenced.
	completed := state
	adoption, intent := *state.Adoption, *state.ExternalImport
	completed.Adoption, completed.ExternalImport = &adoption, &intent
	now := time.Now().UTC()
	completed.Initialized, completed.Epochs = true, inv
	completed.Runtime = q.r
	completed.Runtime.Stage = "stopped"
	adoption.Uncertain, adoption.FinishedAt = false, &now
	intent.Stage = "complete"
	if state.Comet != nil {
		comet := *state.Comet
		comet.StartedAt, comet.FinishedAt, comet.Uncertain = &now, &now, false
		completed.Comet = &comet
	}
	if err = persist(completed); err != nil {
		return state, unknown, err
	}
	return completed, ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func applyExternalImport(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	intent := state.ExternalImport
	if persist == nil || intent == nil || intent.Stage != "reviewed" || state.Prepared || state.Storage == nil || state.Adoption == nil {
		return state, unknown, errors.New("import requires its reviewed, unstarted source intent")
	}
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
	if _, err = pierstore.PrepareMount(d.home, state.Storage); err != nil {
		return state, unknown, err
	}
	target, err := pierstore.Absolute(d.home, state.Storage.Path)
	if err != nil {
		return state, unknown, err
	}
	_, disk, err := d.admission(filepath.Dir(target))
	if err != nil || disk < uint64(source.Bytes)+(8<<30) {
		return state, unknown, errors.New("import needs source size plus 8 GiB free on the destination filesystem")
	}
	// Remove the original launch path before any copy. The retained source lives
	// in a private holding directory until the target has been content-verified.
	heldDir := filepath.Join(filepath.Dir(source.Path), ".omabit-import-"+state.ID)
	intent.HeldPath = filepath.Join(heldDir, "pier")
	intent.Stage = "holding-source"
	if err = persist(state); err != nil {
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
	intent.Stage = "moving"
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	rename := d.renamePier
	if rename == nil {
		rename = pierstore.RenameDirectory
	}
	err = rename(intent.HeldPath, target)
	if errors.Is(err, unix.EXDEV) {
		intent.Stage = "copying"
		if err = persist(state); err != nil {
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
		if _, err = piercopy.Copy(ctx, a, b, func(files int, bytes int64) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			intent.CopiedFiles, intent.CopiedBytes = files, bytes
			return persist(state)
		}); err != nil {
			return state, unknown, err
		}
		intent.Stage = "removing-source"
		if err = persist(state); err != nil {
			return state, unknown, err
		}
		held, e := os.OpenRoot(heldDir)
		if e != nil {
			return state, unknown, e
		}
		err = held.RemoveAll("pier")
		held.Close()
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
	intent.Stage = "enrolling"
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	state, obs, err := applyPierEnrollment(ctx, state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	state.ExternalImport.Stage = "complete"
	if err = persist(state); err != nil {
		return state, unknown, err
	}
	return state, obs, nil
}
