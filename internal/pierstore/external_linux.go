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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"golang.org/x/sys/unix"
)

// TreeReview binds a bounded, same-device tree's metadata without following links.
// File contents are separately verified by the cross-device copier before removal.
func TreeReview(ctx context.Context, path string, ownerMode ...bool) (string, int, int64, error) {
	workspace := ownerWorkspace(path)
	if len(ownerMode) > 0 {
		workspace = ownerMode[0]
	}
	base, err := directoryMode(path, workspace)
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
	type linkKey struct{ device, inode uint64 }
	type links struct {
		expected, seen uint64
		firstPath      string
	}
	docked := map[linkKey]links{}
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
		refuse := func(reason string) error {
			// Quote an untrusted filename, and keep the source's absolute path and
			// file contents out of diagnostics returned to collectors.
			return fmt.Errorf("pier entry %q: %s", rel, reason)
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
			return refuse("nested mount or unavailable filesystem mount identity")
		}
		badWrite := i.Mode().Perm()&0022 != 0
		if workspace && !statePath(p) {
			badWrite = i.Mode().Perm()&0002 != 0
		}
		switch {
		case !ok:
			return refuse("filesystem ownership metadata is unavailable")
		case st.Uid != uint32(os.Geteuid()):
			return refuse(fmt.Sprintf("owned by UID %d; runner UID is %d", st.Uid, os.Geteuid()))
		case st.Dev != device:
			return refuse("entry is on a different filesystem from the pier")
		case i.Mode()&os.ModeSymlink != 0:
			return refuse("symbolic link is not supported")
		case !i.IsDir() && !i.Mode().IsRegular():
			return refuse("special file is not supported; only directories and regular files are accepted")
		case badWrite:
			return refuse(fmt.Sprintf("mode %04o permits unsupported group or world writes; only non-state owner-workspace files may be group writable", i.Mode().Perm()))
		case !i.IsDir() && st.Nlink != 1 && !InertDockedFile(rel):
			return refuse(fmt.Sprintf("regular file has %d hard links; only self-contained .run/.bin executable links are supported", st.Nlink))
		}
		if !i.IsDir() {
			if st.Nlink > 1 {
				key := linkKey{st.Dev, st.Ino}
				old := docked[key]
				if old.expected != 0 && old.expected != st.Nlink {
					return refuse("docked executable links changed during review")
				}
				if old.firstPath == "" {
					old.firstPath = rel
				}
				docked[key] = links{st.Nlink, old.seen + 1, old.firstPath}
			}
			files++
			if i.Size() < 0 || i.Size() > MaxLocalPierBytes-bytes {
				return errors.New("pier exceeds the 8 TiB local import limit")
			}
			bytes += i.Size()
		}
		data, _ := json.Marshal([]any{rel, st.Dev, st.Ino, st.Nlink, st.Size, st.Mtim, st.Ctim, i.Mode()})
		h.Write(data)
		return nil
	})
	if err == nil {
		var outside string
		for _, links := range docked {
			if links.seen != links.expected && (outside == "" || links.firstPath < outside) {
				outside = links.firstPath
			}
		}
		if outside != "" {
			err = fmt.Errorf("pier entry %q: docked executable shares links outside the reviewed pier", outside)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), files, bytes, err
}

func InspectExternal(ctx context.Context, home, path string) (ExternalSource, error) {
	var out ExternalSource
	if err := hostPathMode(path, ownerhost.OwnsHome(home)); err != nil {
		return out, err
	}
	private := Overlap(home, path) && !strings.HasPrefix(path, Root(home)+"/")
	if ownerhost.OwnsHome(home) {
		private = ownerPrivateOverlap(home, path)
	}
	if private {
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
	identity, err := identifyMode(path, ownerhost.OwnsHome(home))
	if err != nil {
		return out, fmt.Errorf("source must be an accessible, runner-owned pier containing .urb: %w", err)
	}
	if Busy(path) {
		return out, errors.New("a Vere lock file is present; stop the source and, if the file is stale, resolve it explicitly before inspecting again")
	}
	if err := CheckStateFiles(ctx, path); err != nil {
		return out, err
	}
	if err := unix.Faccessat(unix.AT_FDCWD, filepath.Dir(path), unix.R_OK|unix.W_OK|unix.X_OK, unix.AT_EACCESS); err != nil {
		return out, errors.New("the runner's Unix owner needs read/write/search permission on the source parent to move and sync the pier")
	}
	fingerprint, files, bytes, err := TreeReview(ctx, path, ownerhost.OwnsHome(home))
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
