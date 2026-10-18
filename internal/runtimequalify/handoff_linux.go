//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func applyHandoff(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error, d dependencies, extract ...func(context.Context, io.Reader, *os.Root) (piercopy.Summary, error)) (ManagedState, ManagedObservation, error) {
	obs := ManagedObservation{State: "moving", Health: "unknown"}
	h := state.Handoff
	if h == nil || !runIDPattern.MatchString(h.ID) || persist == nil {
		return state, obs, errors.New("missing handoff intent")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	q, err := managedSession(&state, persist, d)
	if err != nil {
		return state, obs, err
	}
	finish := func(err error) (ManagedState, ManagedObservation, error) { state.Runtime = q.r; return state, obs, err }
	if err = q.accountBoundary(false); err != nil {
		return finish(err)
	}
	for _, c := range q.r.Containers {
		if !c.Removed || c.CreateUncertain {
			return finish(errors.New("handoff requires resolved stopped containers"))
		}
	}
	if action == "prepare-handoff" {
		if h.Direction != "out" || !state.Initialized {
			return finish(errors.New("source handoff requires an initialized pier"))
		}
		if err = q.pier(); err != nil {
			return finish(err)
		}
		if pierstore.Busy(q.pierPath()) {
			return finish(errors.New("source still has a Vere lock"))
		}
		if err = pierstore.CheckStateFiles(ctx, q.pierPath()); err != nil {
			return finish(err)
		}
		root, err := os.OpenRoot(q.pierPath())
		if err != nil {
			return finish(err)
		}
		defer root.Close()
		summary, err := piercopy.Measure(ctx, root)
		if err != nil {
			return finish(err)
		}
		_, disk, err := q.d.admission(q.base)
		if err != nil || disk < uint64(summary.Bytes)+(8<<30) {
			return finish(errors.New("handoff needs transfer bytes plus 8 GiB free disk"))
		}
		dir := filepath.Join(q.base, "handoffs")
		if err = os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return finish(err)
		}
		if err = bootpill.PrivateDirectory(dir); err != nil {
			return finish(err)
		}
		f, err := os.CreateTemp(dir, ".outgoing-")
		if err != nil {
			return finish(err)
		}
		defer os.Remove(f.Name())
		hash := sha256.New()
		written, err := piercopy.WriteArchive(ctx, root, io.MultiWriter(f, hash))
		if err != nil {
			f.Close()
			return finish(err)
		}
		i, err := f.Stat()
		if err != nil {
			f.Close()
			return finish(err)
		}
		if written != summary || i.Size() > protocol.MaxHandoffBytes {
			f.Close()
			return finish(errors.New("handoff source changed or archive exceeds limit"))
		}
		err = errors.Join(f.Chmod(0400), f.Sync(), f.Close())
		if err != nil {
			return finish(err)
		}
		path := HandoffArchive(q.base, h)
		h.Bytes = i.Size()
		h.TreeDigest = summary.Digest
		h.SHA256 = fmt.Sprintf("%x", hash.Sum(nil))
		if err = os.Link(f.Name(), path); err != nil {
			if !errors.Is(err, os.ErrExist) || !VerifyHandoffArchive(path, h) {
				return finish(errors.New("existing handoff artifact does not match the frozen source"))
			}
		}
		if err = os.Remove(f.Name()); err != nil {
			return finish(err)
		}
		h.Error = ""
		h.State = "ready"
		if err = SyncHandoffDirectory(dir); err != nil {
			return finish(err)
		}
		h.Message = "Source stopped and fenced; ready to move to the selected runner."
		return finish(q.save())
	}
	if action == "receive-handoff" {
		if h.Direction != "in" || h.State != "receiving" {
			return finish(errors.New("incoming handoff requires fresh destination storage"))
		}
		file, err := OpenHandoffArchive(HandoffArchive(q.base, h))
		if err != nil {
			return finish(err)
		}
		defer file.Close()
		hash := sha256.New()
		n, err := io.Copy(hash, io.LimitReader(file, protocol.MaxHandoffBytes+1))
		if err != nil || n != h.Bytes || fmt.Sprintf("%x", hash.Sum(nil)) != h.SHA256 {
			return finish(errors.New("incoming handoff artifact changed"))
		}
		if _, err = file.Seek(0, 0); err != nil {
			return finish(err)
		}
		dir := filepath.Join(q.base, state.ID)
		_, disk, admissionErr := q.d.admission(q.base)
		if admissionErr != nil || disk < uint64(h.Bytes)+(8<<30) {
			return finish(errors.New("handoff extraction needs transfer bytes plus 8 GiB free disk"))
		}
		if !state.Prepared {
			if _, err = os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
				return finish(errors.New("destination working storage already exists"))
			}
			state.Prepared = true
			q.r = Result{Profile: q.d.profile, RunID: state.ID, Directory: dir, Image: candidate.Image, Platform: candidate.Platform, EngineVersion: rootless.EngineVersion, VereVersion: candidate.VereVersion, PillURL: q.pillSource(), PillSHA256: q.d.pillSHA, PillBytes: q.d.pillBytes, PillPath: filepath.Join(dir, candidate.PillName), Stage: "receiving-handoff", Containers: []ContainerRecord{}}
			if err = q.save(); err != nil {
				return finish(err)
			}
		}
		for _, p := range []string{dir, q.config} {
			if err = os.Mkdir(p, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return finish(err)
			}
			if err = q.owned(p, os.ModeDir, true); err != nil {
				return finish(err)
			}
		}
		if _, err = pierstore.PrepareMount(d.home, state.Storage); err != nil {
			return finish(err)
		}
		if err = q.save(); err != nil {
			return finish(err)
		}
		if err = q.receiveHandoffPier(ctx, &state, file, extract...); err != nil {
			return finish(err)
		}
		if err = q.pier(); err != nil {
			return finish(err)
		}
		root, err := os.OpenRoot(q.pierPath())
		if err != nil {
			return finish(err)
		}
		measured, err := piercopy.Measure(ctx, root)
		root.Close()
		if err != nil || measured.Digest != h.TreeDigest {
			return finish(errors.New("received pier content does not match the frozen source"))
		}
		if err = pierstore.CheckStateFiles(ctx, q.pierPath()); err != nil {
			return finish(err)
		}
		if pierstore.Busy(q.pierPath()) {
			return finish(errors.New("received pier has a Vere lock"))
		}
		inv, err := q.inspectEpochs(ctx)
		if err != nil {
			return finish(err)
		}
		if err = SyncHandoffDirectory(filepath.Dir(q.pierPath())); err != nil {
			return finish(err)
		}
		now := time.Now().UTC()
		state.Initialized = true
		state.Epochs = inv
		state.Adoption = &ManagedAdoption{Fingerprint: pierstore.Fingerprint(state.Storage.Identity), FinishedAt: &now, Verified: state.Comet == nil}
		if state.Comet != nil {
			state.Comet.StartedAt = &now
			state.Comet.FinishedAt = &now
			state.Comet.Uncertain = false
		}
		h.State = "received"
		h.Error = ""
		h.Message = "Pier verified on destination; waiting for source handoff commitment."
		return finish(q.save())
	}
	if action == "commit-handoff" {
		if h.Direction != "out" || h.State != "ready" || !state.Initialized || len(state.HandoffSigningKey) != ed25519.PrivateKeySize {
			return finish(errors.New("source handoff is not ready to commit"))
		}
		storageRoot, _, rootErr := pierstore.Resolve(d.home, state.Storage.Path)
		if rootErr != nil {
			return finish(rootErr)
		}
		archive := filepath.Join(storageRoot, "archive", "moved-"+h.ID, filepath.Base(q.pierPath()))
		if err = os.Mkdir(filepath.Dir(filepath.Dir(archive)), 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return finish(err)
		}
		if _, err = pierstore.IdentifyMount(filepath.Dir(filepath.Dir(archive))); err != nil {
			return finish(err)
		}
		if _, statErr := os.Lstat(q.pierPath()); errors.Is(statErr, os.ErrNotExist) {
			identity, e := pierstore.Identify(archive)
			mount, e2 := pierstore.IdentifyMount(filepath.Dir(archive))
			if e != nil || e2 != nil || !identity.SameObject(state.Storage.Identity) || !mount.SameObject(state.Storage.MountIdentity) {
				return finish(errors.New("source archive outcome needs review"))
			}
		} else {
			if err = q.pier(); err != nil {
				return finish(err)
			}
			root, e := os.OpenRoot(q.pierPath())
			if e != nil {
				return finish(e)
			}
			measured, e := piercopy.Measure(ctx, root)
			root.Close()
			if e != nil || measured.Digest != h.TreeDigest {
				return finish(errors.New("frozen source changed before commitment; keep both ends stopped"))
			}
			if err = pierstore.Move(q.pierPath(), archive); err != nil {
				return finish(err)
			}
		}
		h.State = "moved"
		h.Commitment = fmt.Sprintf("%x", ed25519.Sign(state.HandoffSigningKey, h.CommitmentBytes()))
		if !h.ValidCommitment(h.Commitment) {
			return finish(errors.New("handoff signing identity changed"))
		}
		h.Error = ""
		h.Message = "Moved to " + h.DestinationRunner + " / " + h.DestinationInstance + ". Source cannot be started."
		obs.State = "moved"
		return finish(q.save())
	}
	return finish(errors.New("unknown handoff operation"))
}

// Each attempt is extracted outside the active mount into a uniquely named,
// transfer-bound staging directory. Incomplete attempts are retained in archive:
// a recorded parent alone is never authority to delete its unrecorded children.
// This also permits retry after mkdir but before the staging identity was saved.
func (q *qualifier) receiveHandoffPier(ctx context.Context, state *ManagedState, file io.Reader, extract ...func(context.Context, io.Reader, *os.Root) (piercopy.Summary, error)) error {
	h := state.Handoff
	target := q.pierPath()
	if _, err := pierstore.CheckMount(q.d.home, *state.Storage); err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		// Legacy initialized/partially published receipts may already have a
		// recorded child. Verify it; never restage over it or adopt by name.
		if _, err = pierstore.Check(q.d.home, *state.Storage); err != nil {
			return errors.New("incoming mount contains an unrecorded or replaced pier; retained for review")
		}
		return nil // caller verifies the frozen content and epoch inventory
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if state.Initialized {
		return errors.New("initialized incoming pier is missing")
	}
	storageRoot, _, err := pierstore.Resolve(q.d.home, state.Storage.Path)
	if err != nil {
		return err
	}
	archive := filepath.Join(storageRoot, "archive")
	if _, err = pierstore.IdentifyMount(archive); err != nil {
		return err
	}
	stage := state.HandoffStaging
	if stage != nil {
		if stage.TransferID != h.ID || stage.SHA256 != h.SHA256 || filepath.Dir(stage.Path) != archive || !strings.HasPrefix(filepath.Base(stage.Path), "incoming-"+h.ID+"-") {
			return errors.New("incoming staging does not belong to this transfer")
		}
		actual, err := pierstore.IdentifyMount(stage.Path)
		if err != nil || !actual.SameObject(stage.Directory) {
			return errors.New("incoming staging was replaced or moved; retained for review")
		}
		if pierstore.Busy(stage.Path) {
			return errors.New("incoming staging has a Vere lock")
		}
	}
	if stage == nil || !stage.Ready {
		path, err := os.MkdirTemp(archive, "incoming-"+h.ID+"-")
		if err != nil {
			return err
		}
		identity, err := pierstore.IdentifyMount(path)
		if err != nil {
			return err
		}
		if err = pierstore.SyncDirectory(archive); err != nil {
			return err
		}
		stage = &ManagedHandoffStaging{TransferID: h.ID, SHA256: h.SHA256, Path: path, Directory: identity}
		state.HandoffStaging = stage
		if err = q.save(); err != nil {
			return err
		}
		root, err := os.OpenRoot(path)
		if err != nil {
			return err
		}
		read := piercopy.ReadArchive
		if len(extract) > 0 && extract[0] != nil {
			read = extract[0]
		}
		summary, err := read(ctx, file, root)
		root.Close()
		if err != nil {
			return err
		}
		if summary.Digest != h.TreeDigest || pierstore.Busy(path) {
			return errors.New("incoming staging does not match the frozen source")
		}
		stage.Pier, err = pierstore.Identify(path)
		if err != nil {
			return err
		}
		stage.Ready = true
		// Save the final child identity before its no-replacement publication.
		state.Storage.Identity = stage.Pier
		if err = q.save(); err != nil {
			return err
		}
	}
	actual, err := pierstore.Identify(stage.Path)
	if err != nil || !actual.SameObject(stage.Pier) || !stage.Pier.SameObject(state.Storage.Identity) || pierstore.Busy(stage.Path) {
		return errors.New("incoming staging identity changed before publication")
	}
	root, err := os.OpenRoot(stage.Path)
	if err != nil {
		return err
	}
	summary, err := piercopy.Measure(ctx, root)
	root.Close()
	if err != nil || summary.Digest != h.TreeDigest {
		return errors.New("incoming staging content changed before publication")
	}
	if _, err = pierstore.CheckMount(q.d.home, *state.Storage); err != nil {
		return err
	}
	return pierstore.RenameDirectory(stage.Path, target)
}
