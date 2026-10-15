//go:build linux

package pierstore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// TreeReview binds a bounded, same-device tree's metadata without following links.
// File contents are separately verified by the cross-device copier before removal.
func TreeReview(ctx context.Context, path string) (string, int, int64, error) {
	base, err := directory(path)
	if err != nil {
		return "", 0, 0, err
	}
	device := base.Sys().(*syscall.Stat_t).Dev
	var baseMount unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_MNT_ID, &baseMount); err != nil {
		return "", 0, 0, err
	}
	if baseMount.Mask&unix.STATX_MNT_ID == 0 {
		return "", 0, 0, errors.New("filesystem mount identity is unavailable")
	}
	h := sha256.New()
	count, files := 0, 0
	var bytes int64
	err = filepath.WalkDir(path, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		count++
		if count > 8192 || len(strings.Split(rel, "/")) > 32 {
			return errors.New("pier tree exceeds entry/depth limit")
		}
		i, err := e.Info()
		if err != nil {
			return err
		}
		st, ok := i.Sys().(*syscall.Stat_t)
		var mounted unix.Statx_t
		if err := unix.Statx(unix.AT_FDCWD, p, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_MNT_ID, &mounted); err != nil || mounted.Mask&unix.STATX_MNT_ID == 0 || mounted.Mnt_id != baseMount.Mnt_id {
			return errors.New("nested mount in pier tree")
		}
		if !ok || st.Uid != uint32(os.Geteuid()) || st.Dev != device || i.Mode().Perm()&0022 != 0 || !i.IsDir() && (!i.Mode().IsRegular() || st.Nlink != 1) {
			return errors.New("pier contains foreign, linked, mounted, special or group/world-writable files; host administrator must prepare it for urbit-runner")
		}
		if !i.IsDir() {
			files++
			if i.Size() < 0 || i.Size() > 100_000_000_000-bytes {
				return errors.New("pier exceeds 100 GB")
			}
			bytes += i.Size()
		}
		data, _ := json.Marshal([]any{rel, st.Dev, st.Ino, st.Size, st.Mtim, st.Ctim, i.Mode()})
		h.Write(data)
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil)), files, bytes, err
}

func InspectExternal(ctx context.Context, home, path string) (ExternalSource, error) {
	var out ExternalSource
	if err := HostPath(path); err != nil {
		return out, err
	}
	if Overlap(home, path) && !strings.HasPrefix(path, Root(home)+"/") {
		return out, errors.New("source cannot overlap private runner configuration or runtime metadata")
	}
	roots, err := Roots(home)
	if err != nil {
		return out, err
	}
	for _, root := range roots {
		if root.Path == path || strings.HasPrefix(root.Path, path+"/") || Overlap(filepath.Join(root.Path, "archive"), path) {
			return out, errors.New("source cannot be a storage root, its ancestor, or an archived pier")
		}
	}
	identity, err := Identify(path)
	if err != nil {
		return out, fmt.Errorf("source must be an accessible, runner-owned pier containing .urb: %w", err)
	}
	if Busy(path) {
		return out, errors.New("stop the source Vere process and resolve its lock before importing")
	}
	if err := CheckStateFiles(ctx, path); err != nil {
		return out, err
	}
	if err := unix.Faccessat(unix.AT_FDCWD, filepath.Dir(path), unix.R_OK|unix.W_OK|unix.X_OK, unix.AT_EACCESS); err != nil {
		return out, errors.New("urbit-runner needs read/write/search permission on the source parent to move and sync the pier")
	}
	fingerprint, files, bytes, err := TreeReview(ctx, path)
	if err != nil {
		return out, err
	}
	return ExternalSource{Path: path, Identity: identity, Fingerprint: fingerprint, Files: files, Bytes: bytes}, nil
}

// RenameDirectory is no-replacement and syncs both parent directories. Ancestors
// and source identity are checked by the caller immediately before mutation.
func RenameDirectory(source, target string) error {
	a, err := os.OpenFile(filepath.Dir(source), os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer a.Close()
	b, err := os.OpenFile(filepath.Dir(target), os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer b.Close()
	if err = unix.Renameat2(int(a.Fd()), filepath.Base(source), int(b.Fd()), filepath.Base(target), unix.RENAME_NOREPLACE); err != nil {
		return err
	}
	return errors.Join(a.Sync(), b.Sync())
}
