//go:build linux

package pierstore

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// RenameBoundDirectory retains both opened parents until the no-replacement
// rename and directory syncs finish. It checks the reviewed source and parent
// objects through those descriptors. The caller must serialize its own jobs and
// use boundary to recheck paths, locks and tree contents immediately before the
// syscall. Linux has no rename-conditional-on-inode primitive: this is change
// detection, not exclusion of an unrelated same-owner writer.
//
// An error may follow a completed rename (for example, a directory sync error).
// Callers must retain their pre-rename durable intent and inspect both names on
// retry; never blindly repeat or roll back an uncertain move.
func RenameBoundDirectory(source, target string, object, sourceParent, targetParent DirectoryIdentity, boundary func() error) error {
	if boundary == nil || !filepath.IsAbs(source) || !filepath.IsAbs(target) ||
		filepath.Clean(source) != source || filepath.Clean(target) != target || source == target {
		return errors.New("bound rename requires canonical paths and a boundary check")
	}
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
	for f, want := range map[*os.File]DirectoryIdentity{a: sourceParent, b: targetParent} {
		got, err := removalFileEntry(f, ".")
		if err != nil || !got.Directory || want.Inode == 0 || want.Birth == "" || got.Identity != want {
			return errors.New("bound rename parent was replaced")
		}
	}
	fd, err := unix.Openat(int(a.Fd()), filepath.Base(source), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	child := os.NewFile(uintptr(fd), filepath.Base(source))
	defer child.Close()
	entry, err := removalFileEntry(child, ".")
	if err != nil || !entry.Directory || object.Inode == 0 || object.Birth == "" || entry.Identity != object {
		return errors.New("bound rename source was replaced")
	}
	if err = boundary(); err != nil {
		return err
	}
	// Reopen the basename after the boundary; an exchanged source must not be
	// mistaken for the descriptor retained above.
	fd, err = unix.Openat(int(a.Fd()), filepath.Base(source), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	current := os.NewFile(uintptr(fd), filepath.Base(source))
	defer current.Close()
	entry, err = removalFileEntry(current, ".")
	if err != nil || entry.Identity != object {
		return errors.New("bound rename source changed at the boundary")
	}
	if err = unix.Renameat2(int(a.Fd()), filepath.Base(source), int(b.Fd()), filepath.Base(target), unix.RENAME_NOREPLACE); err != nil {
		return err
	}
	return errors.Join(a.Sync(), b.Sync())
}

// SyncBoundDirectories repeats directory durability after an uncertain rename.
// Identity is checked through each opened descriptor; a replaced name cannot
// stand in for the parent whose rename needs to reach stable storage.
func SyncBoundDirectories(parents map[string]DirectoryIdentity) error {
	for path, want := range parents {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return errors.New("sync requires a canonical parent")
		}
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		got, err := removalFileEntry(f, ".")
		if err != nil || !got.Directory || want.Inode == 0 || want.Birth == "" || got.Identity != want {
			f.Close()
			return errors.New("restoration sync parent was replaced")
		}
		err = f.Sync()
		closed := f.Close()
		if err := errors.Join(err, closed); err != nil {
			return err
		}
	}
	return nil
}
