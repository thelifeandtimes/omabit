//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// PrepareProjectDependencies is an explicit download/cache step, separate from
// project resource creation. The caller holds the runner resource/update fence.
// Only compiled runtime pins are used; no caller URL, image or path is accepted.
func PrepareProjectDependencies(ctx context.Context) error {
	return prepareProjectDependencies(ctx, defaults())
}
func prepareProjectDependencies(ctx context.Context, d dependencies) error {
	q := &qualifier{d: d, phase: "project-dependencies"}
	q.base = filepath.Join(d.home, ".local/share/urbit-runner/managed")
	if err := q.verifyEngine(ctx); err != nil {
		return err
	}
	for _, path := range q.storagePaths() {
		if err := q.owned(path, os.ModeDir, path == q.base); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(q.base, ".project-dependencies-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir) // only this invocation's inert temporary downloads/config
	q.r.Directory = dir
	q.config = filepath.Join(dir, "docker-config")
	if err = os.Mkdir(q.config, 0700); err != nil {
		return err
	}
	if err = q.managedImage(ctx); err != nil {
		return err
	}
	meta := protocol.BootPill{SHA256: d.pillSHA, Bytes: d.pillBytes}
	cache := bootpill.CachePath(q.base, meta)
	if err = os.Mkdir(filepath.Dir(cache), 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	if err = bootpill.PrivateDirectory(filepath.Dir(cache)); err != nil {
		return err
	}
	if _, err = os.Lstat(cache); err == nil {
		if err = q.owned(cache, 0, true); err != nil {
			return err
		}
		return bootpill.Verify(ctx, cache, meta)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	q.r.PillPath = filepath.Join(dir, "bootstrap.pill")
	if err = q.pill(ctx); err != nil {
		return err
	}
	if err = os.Chmod(q.r.PillPath, 0400); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Link(q.r.PillPath, cache); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if err = q.owned(cache, 0, true); err != nil {
			return err
		}
		if err = bootpill.Verify(ctx, cache, meta); err != nil {
			return err
		}
	}
	directory, err := os.Open(filepath.Dir(cache))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
