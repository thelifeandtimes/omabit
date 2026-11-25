//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// managedSession validates the selection before constructing a qualifier. Other
// qualifier factories retain the historical official profile by omission.
func (q *qualifier) runtimeProfile() candidate.Profile {
	p, _ := candidate.Resolve(q.d.runtimeProfile)
	return p
}

func (q *qualifier) profileArchive() string {
	return filepath.Join(q.d.home, ".local/share/urbit-runner/runtime-profiles", q.runtimeProfile().ID, "image.tar.gz")
}

func (q *qualifier) verifyProfileArchive() error {
	p := q.runtimeProfile()
	if p.Kind == "official" {
		return nil
	}
	if p.Kind != "reviewed" && p.Kind != "local" {
		return errors.New("invalid runtime profile")
	}
	path := q.profileArchive()
	if err := q.owned(path, 0, true); err != nil {
		return fmt.Errorf("runtime %s archive unavailable; stage the exact reviewed artifact with urbit-runner runtime-profile-install before starting or importing: %w", p.ID, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, p.ArchiveBytes+1))
	if err != nil || n != p.ArchiveBytes || hex.EncodeToString(h.Sum(nil)) != p.ArchiveSHA256 {
		return errors.New("reviewed runtime archive checksum mismatch; no fallback runtime is permitted")
	}
	return nil
}

// InstallRuntimeProfile stages a finite catalogue artifact without opening a
// pier, a runner database or changing a ship selection. Repetition verifies the
// retained bytes and loads the same immutable image. No arbitrary image or
// executable is accepted. A partial copy is never published as an active archive.
func InstallRuntimeProfile(ctx context.Context, id, source string) (candidate.Profile, error) {
	return installRuntimeProfile(ctx, id, source, defaults())
}

func installRuntimeProfile(ctx context.Context, id, source string, d dependencies) (candidate.Profile, error) {
	p, err := candidate.Resolve(id)
	if err != nil {
		return p, err
	}
	if p.Kind != "reviewed" && p.Kind != "local" {
		return p, errors.New("archive installation requires a reviewed custom runtime profile")
	}
	q := &qualifier{d: d}
	q.d.runtimeProfile = id
	if err = q.accountBoundary(false); err != nil {
		return p, err
	}
	parent := filepath.Dir(q.profileArchive())
	q.base, q.r.Directory = filepath.Dir(parent), parent
	q.config = filepath.Join(parent, "docker-config")
	for _, dir := range append(q.storagePaths(), parent, q.config) {
		if err = os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return p, err
		}
		if err = q.owned(dir, os.ModeDir, dir == parent || dir == q.config || dir == q.base); err != nil {
			return p, err
		}
	}
	if _, err = os.Lstat(q.profileArchive()); errors.Is(err, os.ErrNotExist) {
		input, e := os.OpenFile(source, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if e != nil {
			return p, e
		}
		defer input.Close()
		info, e := input.Stat()
		if e != nil || !info.Mode().IsRegular() || info.Size() != p.ArchiveBytes {
			return p, errors.New("invalid runtime archive input")
		}
		tmp, e := os.CreateTemp(parent, ".staging-*")
		if e != nil {
			return p, e
		}
		defer os.Remove(tmp.Name())
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(input, p.ArchiveBytes+1))
		if e != nil || n != p.ArchiveBytes || hex.EncodeToString(h.Sum(nil)) != p.ArchiveSHA256 {
			tmp.Close()
			return p, errors.New("runtime archive does not match the reviewed checksum")
		}
		if e = tmp.Sync(); e != nil {
			tmp.Close()
			return p, e
		}
		if e = tmp.Close(); e != nil {
			return p, e
		}
		// Link publishes without overwriting a concurrent installer's archive.
		if e = os.Link(tmp.Name(), q.profileArchive()); e != nil && !errors.Is(e, os.ErrExist) {
			return p, e
		}
		if e = os.Remove(tmp.Name()); e != nil {
			return p, e
		}
		dir, e := os.Open(parent)
		if e != nil {
			return p, e
		}
		e = dir.Sync()
		dir.Close()
		if e != nil {
			return p, e
		}
	} else if err != nil {
		return p, err
	}
	if err = q.verifyProfileArchive(); err != nil {
		return p, err
	}
	q.config = filepath.Join(parent, "docker-config")
	if err = os.Mkdir(q.config, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return p, err
	}
	if err = q.owned(q.config, os.ModeDir, true); err != nil {
		return p, err
	}
	if err = q.verifyEngine(ctx); err != nil {
		return p, err
	}
	if _, err = q.docker(ctx, q.d.times.pull, "image", "load", "--input", q.profileArchive()); err != nil {
		return p, err
	}
	if err = q.inspectImage(ctx); err != nil {
		return p, err
	}
	return p, nil
}

// VerifyRuntimeProfile performs no pier read or runtime execution. Custom image
// availability includes the retained authenticated archive and actual daemon ID.
func VerifyRuntimeProfile(ctx context.Context, id string) error {
	return verifyRuntimeProfile(ctx, id, defaults())
}

func verifyRuntimeProfile(ctx context.Context, id string, d dependencies) error {
	_, err := candidate.Resolve(id)
	if err != nil {
		return err
	}
	q := &qualifier{d: d}
	q.d.runtimeProfile = id
	if err = q.accountBoundary(false); err != nil {
		return err
	}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner")
	for _, dir := range q.storagePaths() {
		if err = q.owned(dir, os.ModeDir, false); err != nil {
			return err
		}
	}
	// A private empty Docker configuration prevents inherited credentials or
	// contexts. This temporary directory contains no pier or persistent state.
	q.r.Directory, err = os.MkdirTemp(q.base, "runtime-preflight-")
	if err != nil {
		return err
	}
	defer os.Remove(q.r.Directory)
	q.config = filepath.Join(q.r.Directory, "docker-config")
	if err = os.Mkdir(q.config, 0700); err != nil {
		return err
	}
	defer os.Remove(q.config)
	if err = q.verifyEngine(ctx); err != nil {
		return err
	}
	return q.inspectImage(ctx)
}
