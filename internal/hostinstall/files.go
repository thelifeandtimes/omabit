package hostinstall

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func (i *installer) path(path string) string { return filepath.Join(i.root, path) }

func ownership(info os.FileInfo) (uint32, uint32, bool) {
	if info == nil {
		return 0, 0, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return stat.Uid, stat.Gid, true
}

func (i *installer) absent(path string) error {
	_, err := i.lstat(i.path(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	return fmt.Errorf("refusing existing %s; first install never overwrites or takes over resources", path)
}

// Every existing component, including /, is checked before creation. Once the
// chain is root-owned and non-writable, only another privileged actor can race
// these operations. Concurrent privileged administration is outside this helper.
func (i *installer) parents(ctx context.Context, path string, create bool) error {
	dir := filepath.Dir(path)
	current := "/"
	for _, part := range append([]string{""}, strings.Split(strings.TrimPrefix(dir, "/"), "/")...) {
		if err := ctx.Err(); err != nil {
			return err
		}
		current = filepath.Join(current, part)
		info, err := i.lstat(i.path(current))
		if errors.Is(err, os.ErrNotExist) {
			if !create {
				return nil
			}
			if err := os.Mkdir(i.path(current), 0755); err != nil {
				return fmt.Errorf("exclusively create directory %s: %w", current, err)
			}
			i.mutated = true
			f, openErr := os.OpenFile(i.path(current), os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
			if openErr != nil {
				return fmt.Errorf("open created directory %s: %w", current, openErr)
			}
			err = errors.Join(i.chown(f), f.Chmod(0755), f.Sync(), f.Close())
			if err != nil {
				return fmt.Errorf("secure created directory %s: %w", current, err)
			}
			info, err = i.lstat(i.path(current))
		}
		uid, _, ok := ownership(info)
		if err != nil || !ok || !info.IsDir() || uid != 0 || info.Mode().Perm()&0022 != 0 {
			return fmt.Errorf("unsafe parent %s: must be a root-owned directory without symlinks or group/world write", current)
		}
	}
	return nil
}

func (i *installer) write(ctx context.Context, path string, data []byte, mode os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.OpenFile(i.path(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return fmt.Errorf("exclusively create %s: %w", path, err)
	}
	i.mutated = true
	_, writeErr := f.Write(data)
	err = errors.Join(writeErr, i.chown(f), f.Chmod(mode), f.Sync(), f.Close())
	if err != nil {
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

func (i *installer) readMetadata(ctx context.Context, path string) ([]byte, error) {
	if err := i.parents(ctx, path, false); err != nil {
		return nil, err
	}
	info, err := i.lstat(i.path(path))
	uid, _, ok := ownership(info)
	if err != nil || !ok || !info.Mode().IsRegular() || uid != 0 || info.Mode().Perm()&0022 != 0 {
		return nil, fmt.Errorf("unsafe account metadata %s", path)
	}
	f, err := os.OpenFile(i.path(path), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open account metadata %s: %w", path, err)
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxOutput {
		return nil, fmt.Errorf("account metadata %s must be a bounded regular file", path)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxOutput+1))
	if err != nil || len(data) > maxOutput {
		return nil, fmt.Errorf("cannot read bounded account metadata %s", path)
	}
	return data, ctx.Err()
}
