//go:build linux && amd64

package runtimequalify

// Adapted from wundrellama's PR #3 (3496300/d512fdc), retaining its read-only
// filesystem evidence checks and atomic-publication boundary. RC.8 extensions:
// in-place enrollment, named piers, public identity checks and Ames publication.
import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"golang.org/x/sys/unix"
)

// A moved import must retain absence evidence for both historical names. Even
// an unrelated replacement requires review, never removal. Device renumbering
// needs review for legacy journals lacking source-parent/mount identities.
func absentImportPaths(state ManagedState) error {
	intent := state.ExternalImport
	if intent == nil || !filepath.IsAbs(intent.Source.Path) || filepath.Clean(intent.Source.Path) != intent.Source.Path || intent.Source.Path == "/" ||
		intent.HeldPath != filepath.Join(filepath.Dir(intent.Source.Path), ".omabit-import-"+state.ID, "pier") {
		return errors.New("source/holding evidence paths require review")
	}
	parent, err := unix.Open("/", unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { unix.Close(parent) }()
	var st unix.Stat_t
	if err = unix.Fstat(parent, &st); err != nil {
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
		next, err := unix.Openat2(parent, part, &unix.OpenHow{Flags: unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: resolve})
		if err != nil {
			return errors.New("source parent is unavailable or unsafe")
		}
		unix.Close(parent)
		parent = next
		if err = unix.Fstat(parent, &st); err != nil {
			return err
		}
	}
	if st.Dev != intent.Source.Identity.Device {
		return errors.New("source parent filesystem changed; review absence evidence")
	}
	for _, path := range []string{filepath.Base(intent.Source.Path), filepath.Join(filepath.Base(filepath.Dir(intent.HeldPath)), "pier")} {
		fd, err := unix.Openat2(parent, path, &unix.OpenHow{Flags: unix.O_PATH | unix.O_CLOEXEC | unix.O_NOFOLLOW, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
		if err == nil {
			unix.Close(fd)
			return errors.New("source/holding path exists; review retained or recreated data without removing it")
		}
		if !errors.Is(err, unix.ENOENT) {
			return errors.New("source/holding absence cannot be established through unsafe or inaccessible paths")
		}
	}
	return nil
}

func importRecoveryEvidence(state ManagedState, home string) error {
	if state.ExternalImport == nil || state.Storage == nil {
		return errors.New("missing import evidence")
	}
	target, err := pierstore.Absolute(home, state.Storage.Path)
	if err != nil {
		return err
	}
	source := state.ExternalImport.Source
	if source.Path == target && state.ExternalImport.HeldPath == "" {
		// In-place import never had an absent source. Prove it is the same
		// reviewed filesystem object, not another copy or a reused path.
		if source.Identity.Inode == 0 || !source.Identity.SameObject(state.Storage.Identity) {
			return errors.New("in-place source identity does not match the retained pier")
		}
		_, err = pierstore.Check(home, *state.Storage)
		return err
	}
	return absentImportPaths(state)
}

func applyImportRecovery(ctx context.Context, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
	unknown := ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
	if persist == nil || !state.ImportRecoveryEligible() {
		return state, unknown, errors.New("recovery requires an interrupted prepared external enrollment with retained identity")
	}
	if err := ctx.Err(); err != nil {
		return state, unknown, err
	}
	if err := importRecoveryEvidence(state, d.home); err != nil {
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
	inv, err := q.inspectImportEpochs(ctx)
	if err != nil {
		return state, unknown, err
	}
	if state.ExternalImport.Identify {
		identity, err := InspectPierIdentity(ctx, path)
		ship, kind := "~zod", "fake"
		if state.Fake != nil {
			ship = state.Fake.Ship
		}
		if state.Comet != nil {
			ship = state.Comet.Ship
			kind = protocol.ShipClass(ship)
		}
		if err != nil || identity.Ship != ship || identity.Kind != kind {
			return state, unknown, errors.New("retained identity or network differs from the guided import")
		}
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
	if err = importRecoveryEvidence(state, d.home); err != nil {
		return state, unknown, err
	}
	// This is a proposed completion only. The store suppresses incremental
	// writes for recovery and publishes this state with its success receipt.
	completed := state
	adoption, intent := *state.Adoption, *state.ExternalImport
	completed.Adoption, completed.ExternalImport = &adoption, &intent
	now := time.Now().UTC()
	completed.Initialized, completed.Epochs = true, inv
	completed.Runtime = q.r
	completed.Runtime.Stage = "stopped"
	adoption.Uncertain, adoption.FinishedAt = false, &now
	intent.Stage = "complete"
	intent.Events = append(append([]protocol.ImportEvent(nil), intent.Events...), protocol.ImportEvent{Stage: "complete", At: now})
	if len(intent.Events) > 16 {
		intent.Events = intent.Events[len(intent.Events)-16:]
	}
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
