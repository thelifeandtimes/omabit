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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/piercopy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func applyHandoff(ctx context.Context, action string, state ManagedState, persist func(ManagedState) error, d dependencies) (ManagedState, ManagedObservation, error) {
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
		if !state.Initialized {
			// Only unactivated data owned by this incoming handoff may be restaged.
			if _, err = pierstore.CheckMount(d.home, *state.Storage); err != nil {
				return finish(err)
			}
			if err = os.RemoveAll(q.pierPath()); err != nil {
				return finish(err)
			}
			state.Storage.Identity = pierstore.Identity{}
			if err = os.Mkdir(q.pierPath(), 0700); err != nil {
				return finish(err)
			}
			root, err := os.OpenRoot(q.pierPath())
			if err != nil {
				return finish(err)
			}
			_, err = piercopy.ReadArchive(ctx, file, root)
			root.Close()
			if err != nil {
				return finish(err)
			}
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
		archive := filepath.Join(storageRoot, "archive", "moved-"+h.ID, "pier")
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
