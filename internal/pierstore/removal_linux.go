//go:build linux

package pierstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func removalEntry(path, relative string) (RemovalEntry, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return RemovalEntry{}, err
	}
	defer f.Close()
	return removalFileEntry(f, relative)
}

func removalEntryAt(root *os.Root, name, relative string) (RemovalEntry, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return RemovalEntry{}, err
	}
	defer f.Close()
	return removalFileEntry(f, relative)
}

func removalFileEntry(f *os.File, relative string) (RemovalEntry, error) {
	i, err := f.Stat()
	if err != nil {
		return RemovalEntry{}, err
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	var x unix.Statx_t
	if !ok || st.Uid != uint32(os.Geteuid()) || (!i.IsDir() && !i.Mode().IsRegular()) ||
		unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH, unix.STATX_BTIME|unix.STATX_MNT_ID, &x) != nil || x.Mask&(unix.STATX_BTIME|unix.STATX_MNT_ID) != unix.STATX_BTIME|unix.STATX_MNT_ID {
		return RemovalEntry{}, errors.New("deletion object has unsafe or unavailable identity")
	}
	return RemovalEntry{Path: relative, Identity: DirectoryIdentity{Device: st.Dev, Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec)}, Directory: i.IsDir(), Mode: uint32(i.Mode()), Size: i.Size(), Modified: i.ModTime().UnixNano(), Changed: st.Ctim.Nano(), Links: st.Nlink, mountID: x.Mnt_id}, nil
}

func sameRemovalEntry(a, b RemovalEntry) bool {
	return a.Path == b.Path && a.Identity.SameObject(b.Identity) && a.Directory == b.Directory && a.Mode == b.Mode &&
		(a.Directory || a.Size == b.Size && a.Modified == b.Modified &&
			(b.Links == 0 || a.Links <= b.Links) &&
			(b.Changed == 0 || a.Changed == b.Changed || b.Links > 1 && a.Links < b.Links))
}

// ReviewRemoval is called before publishing destructive intent, never to adopt
// the remaining contents of a partially removed tree on retry.
func ReviewRemoval(ctx context.Context, path string) (*RemovalTree, error) {
	if _, _, _, err := TreeReview(ctx, path); err != nil {
		return nil, err
	}
	tree := &RemovalTree{}
	err := filepath.WalkDir(path, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		e, err := removalEntry(p, rel)
		if err != nil {
			return err
		}
		tree.Entries = append(tree.Entries, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tree, tree.Check(ctx, path)
}

func (t *RemovalTree) index() (map[string]RemovalEntry, error) {
	if t == nil || len(t.Entries) == 0 || len(t.Entries) > 8192 || t.Entries[0].Path != "." || !t.Entries[0].Directory {
		return nil, errors.New("missing deletion object manifest")
	}
	entries := make(map[string]RemovalEntry, len(t.Entries))
	for _, e := range t.Entries {
		if e.Path != "." && !filepath.IsLocal(e.Path) || filepath.Clean(e.Path) != e.Path || e.Identity.Inode == 0 || e.Identity.Birth == "" {
			return nil, errors.New("invalid deletion object manifest")
		}
		if _, exists := entries[e.Path]; exists {
			return nil, errors.New("duplicate deletion object")
		}
		if e.Path != "." {
			if parent, exists := entries[filepath.Dir(e.Path)]; !exists || !parent.Directory {
				return nil, errors.New("unordered deletion object manifest")
			}
		}
		entries[e.Path] = e
	}
	return entries, nil
}

// Check accepts only a subset of the original objects. It does not infer that a
// new child is disposable merely because its containing directory is owned.
func (t *RemovalTree) Check(ctx context.Context, path string) error {
	entries, err := t.index()
	if err != nil {
		return err
	}
	base, err := removalEntry(path, ".")
	if err != nil {
		return err
	}
	return filepath.WalkDir(path, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		want, exists := entries[rel]
		got, err := removalEntry(p, rel)
		if err != nil || !exists || !sameRemovalEntry(got, want) || got.mountID != base.mountID {
			return errors.New("deletion tree contains an unrecorded, replaced or changed object")
		}
		return nil
	})
}

// Remove unlinks only recorded objects, deepest first. Each unlink receives an
// opened, identity-checked immediate parent and a single basename. Renaming an
// ancestor after the last check therefore cannot redirect it into a replacement
// pier. Boundaries also run after unlink: a late change stops further removal.
// Linux has no conditional-unlink-by-inode primitive; these checks do not exclude
// a same-owner writer atomically between the last check and the unlink syscall.
func (t *RemovalTree) Remove(ctx context.Context, path string, boundary func() error, remove func(*os.Root, string) error) error {
	entries, err := t.index()
	if err != nil {
		return err
	}
	if err = t.Check(ctx, path); err != nil {
		return err
	}
	parent, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	root, err := parent.OpenRoot(filepath.Base(path))
	if err != nil {
		return err
	}
	defer root.Close()
	if remove == nil {
		remove = func(r *os.Root, p string) error { return r.Remove(p) }
	}
	for i := len(t.Entries) - 1; i >= 0; i-- {
		if err = ctx.Err(); err != nil {
			return err
		}
		e := t.Entries[i]
		if err = removeRecordedEntry(path, parent, root, entries, e, boundary, remove); err != nil {
			return err
		}
	}
	return nil
}

func removeRecordedEntry(path string, parent, root *os.Root, entries map[string]RemovalEntry, e RemovalEntry, boundary func() error, remove func(*os.Root, string) error) error {
	base, err := removalEntryAt(root, ".", ".")
	if err != nil || !sameRemovalEntry(base, entries["."]) {
		return errors.New("opened deletion root changed")
	}
	current := root
	opened := []*os.Root{}
	defer func() {
		for _, r := range opened {
			r.Close()
		}
	}()
	// Keep every opened ancestor until this step finishes. Check native markers
	// through these descriptors too, including when their old path was renamed.
	ancestors := map[string]*os.Root{".": root}
	relative := "."
	if dir := filepath.Dir(e.Path); dir != "." {
		for _, part := range strings.Split(dir, string(filepath.Separator)) {
			relative = filepath.Join(relative, part)
			got, err := removalEntryAt(current, part, relative)
			if errors.Is(err, os.ErrNotExist) {
				return nil // an earlier deletion already removed this subtree
			}
			if err != nil || !sameRemovalEntry(got, entries[relative]) || got.mountID != base.mountID {
				return errors.New("deletion ancestor changed before open")
			}
			next, err := current.OpenRoot(part)
			if err != nil {
				return err
			}
			opened = append(opened, next)
			actual, err := removalEntryAt(next, ".", relative)
			if err != nil || !sameRemovalEntry(actual, entries[relative]) || actual.mountID != base.mountID {
				return errors.New("opened deletion ancestor does not match intent")
			}
			current, ancestors[relative] = next, next
		}
	}
	name := filepath.Base(e.Path)
	if e.Path == "." {
		current, name = parent, filepath.Base(path)
	}
	check := func() error {
		if boundary != nil {
			if err := boundary(); err != nil {
				return err
			}
		}
		for rel, descriptor := range ancestors {
			actual, err := removalEntry(filepath.Join(path, rel), rel)
			if err != nil || !sameRemovalEntry(actual, entries[rel]) || actual.mountID != base.mountID {
				return errors.New("deletion ancestor moved or changed at unlink boundary")
			}
			if _, pier := entries[filepath.Join(rel, ".urb")]; pier {
				for _, lock := range []string{".vere.lock", ".urb/vere.lock", ".urb/lock"} {
					if _, err := descriptor.Lstat(lock); !errors.Is(err, os.ErrNotExist) {
						return errors.New("opened pier has a Vere lock")
					}
				}
			}
		}
		return nil
	}
	if err := check(); err != nil {
		return err
	}
	actual, err := removalEntryAt(current, name, e.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !sameRemovalEntry(actual, e) || actual.mountID != base.mountID {
		return errors.New("deletion target changed at unlink boundary")
	}
	if err := remove(current, name); err != nil {
		return err
	}
	f, err := current.Open(".")
	if err != nil {
		return err
	}
	err = errors.Join(f.Sync(), f.Close())
	if e.Path == "." {
		// The root itself is now absent. No further tree unlink is possible.
		if boundary != nil {
			err = errors.Join(err, boundary())
		}
		return err
	}
	return errors.Join(err, check())
}

func SyncDirectory(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

// SyncDestination publishes the leaf, mount, grouping directories and storage
// root in bottom-up order. The root's parent is included because Parents may
// have created the default root during this operation. Existing registered root
// ancestors are established by root registration rather than this import.
func SyncDestination(home string, location Location, sync func(string) error) error {
	return syncDestination(home, location, true, sync)
}

func SyncDestinationParents(home string, location Location, sync func(string) error) error {
	return syncDestination(home, location, false, sync)
}

func syncDestination(home string, location Location, leaf bool, sync func(string) error) error {
	root, relative, err := Resolve(home, location.Path)
	if err != nil {
		return err
	}
	if sync == nil {
		sync = SyncDirectory
	}
	start := filepath.Join(root, relative)
	if !leaf {
		start = filepath.Dir(start)
	}
	for p := start; ; p = filepath.Dir(p) {
		if p != root && !strings.HasPrefix(p, root+string(filepath.Separator)) {
			return errors.New("destination escaped storage root")
		}
		if err = sync(p); err != nil {
			return err
		}
		if p == root {
			return sync(filepath.Dir(root))
		}
	}
}
