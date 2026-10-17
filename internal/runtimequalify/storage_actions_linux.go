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
		state, obs, err := applyPierEnrollment(ctx, state, persist, d)
		if err != nil {
			return state, unknown, err
		}
		state.ExternalImport.Stage = "complete"
		state.ExternalImport.Events = append(state.ExternalImport.Events, protocol.ImportEvent{Stage: "complete", At: time.Now().UTC()})
		return state, obs, persist(state)
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
		lastReport := time.Time{}
		if _, err = piercopy.Copy(ctx, a, b, func(files int, bytes int64) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			intent.CopiedFiles, intent.CopiedBytes = files, bytes
			if files != source.Files && time.Since(lastReport) < 250*time.Millisecond {
				return nil
			}
			lastReport = time.Now()
			return persist(state)
		}, func() error { return stage("verifying") }); err != nil {
			return state, unknown, err
		}
		if err = stage("removing-source"); err != nil {
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
	if err = stage("enrolling"); err != nil {
		return state, unknown, err
	}
	state, obs, err := applyPierEnrollment(ctx, state, persist, d)
	if err != nil {
		return state, unknown, err
	}
	intent = state.ExternalImport
	if err = stage("complete"); err != nil {
		return state, unknown, err
	}
	return state, obs, nil
}
