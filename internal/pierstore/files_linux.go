//go:build linux

package pierstore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"golang.org/x/sys/unix"
)

func Root(home string) string { p, _ := ownerhost.DefaultRoot(home); return p }

func directory(p string) (os.FileInfo, error) {
	return directoryMode(p, ownerWorkspace(p))
}

func directoryMode(p string, ownerMode bool) (os.FileInfo, error) {
	i, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	badWrite := i.Mode().Perm()&0022 != 0
	if ownerMode && !statePath(p) {
		badWrite = i.Mode().Perm()&0002 != 0
	}
	if !ok || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 || s.Uid != uint32(os.Geteuid()) || badWrite {
		return nil, errors.New("pier directory must be owned by the runner, not linked or group/world writable")
	}
	return i, nil
}

func statePath(path string) bool {
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		if part == ".urb" {
			return true
		}
	}
	return false
}

func ownerWorkspace(path string) bool {
	p, err := ownerhost.Current()
	if err != nil || p == nil || statePath(path) {
		return false
	}
	if path == p.Home || Overlap(filepath.Join(p.Home, ".local/share/urbit-runner"), path) || Overlap(filepath.Join(p.Home, ".config/omabit/setup"), path) {
		return false
	}
	within := func(root string) bool { return path == root || strings.HasPrefix(path, root+"/") }
	if within(p.Piers) || strings.HasPrefix(path, p.Home+"/") {
		return true
	}
	roots, err := Roots(p.Home)
	if err != nil {
		return false
	}
	for _, root := range roots {
		if within(root.Path) {
			return true
		}
	}
	return false
}

// Init never changes permissions/ownership of existing user-managed entries.
func Init(home string) error {
	if _, err := directory(home); err != nil {
		return err
	}
	root, err := ownerhost.DefaultRoot(home)
	if err != nil {
		return err
	}
	return initRoot(root)
}

func initRoot(root string) error {
	return initRootMode(root, ownerWorkspace(root))
}

func initRootMode(root string, ownerMode bool) error {
	for _, relative := range append([]string{"", "livenet"}, Categories...) {
		p := filepath.Join(root, relative)
		if err := os.Mkdir(p, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if _, err := directoryMode(p, ownerMode); err != nil {
			return err
		}
	}
	return nil
}

// Path validates every ancestor without resolving symlinks or inventing a
// missing pier. A missing leaf remains missing, even for previously enrolled data.
func Path(home, relative string) (string, error) {
	if !ValidPath(relative) {
		return "", errors.New("pier path must be inside the fakeships or classified livenet tree")
	}
	p, relative, err := Resolve(home, relative)
	if err != nil {
		return "", err
	}
	if _, err := directory(p); err != nil {
		return "", err
	}
	for _, part := range strings.Split(relative, "/") {
		p = filepath.Join(p, part)
		if _, err := directory(p); err != nil {
			return "", err
		}
	}
	return p, nil
}

func Parents(home, relative string) (string, error) {
	if !ValidPath(relative) {
		return "", errors.New("invalid pier path")
	}
	if err := Init(home); err != nil {
		return "", err
	}
	p, relative, err := Resolve(home, relative)
	if err != nil {
		return "", err
	}
	if err := initRoot(p); err != nil {
		return "", err
	}
	parts := strings.Split(relative, "/")
	// Create grouping directories, but leave the exclusive mount root absent.
	for _, part := range parts[:len(parts)-2] {
		// Grouping directories may not themselves be piers.
		if err := groupingDirectory(p); err != nil {
			return "", err
		}
		p = filepath.Join(p, part)
		if err := os.Mkdir(p, 0755); err != nil && !errors.Is(err, os.ErrExist) {
			return "", err
		}
		if _, err := directory(p); err != nil {
			return "", err
		}
	}
	if err := groupingDirectory(p); err != nil {
		return "", err
	}
	return filepath.Join(p, parts[len(parts)-2], parts[len(parts)-1]), nil
}

func groupingDirectory(p string) error {
	if _, err := os.Lstat(filepath.Join(p, ".urb")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot nest a pier inside another pier")
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(4097)
	f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 4096 {
		return errors.New("storage grouping directory exceeds entry limit")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Lstat(filepath.Join(p, entry.Name(), ".urb")); !errors.Is(err, os.ErrNotExist) {
			return errors.New("cannot nest a ship inside another mount root")
		}
	}
	return nil
}

func IdentifyMount(p string) (DirectoryIdentity, error) {
	return identifyMountMode(p, ownerWorkspace(p))
}

func identifyMountMode(p string, ownerMode bool) (DirectoryIdentity, error) {
	i, err := directoryMode(p, ownerMode)
	if err != nil {
		return DirectoryIdentity{}, err
	}
	st := i.Sys().(*syscall.Stat_t)
	var x unix.Statx_t
	if err = unix.Statx(unix.AT_FDCWD, p, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &x); err != nil || x.Mask&unix.STATX_BTIME == 0 {
		return DirectoryIdentity{}, errors.New("mount root needs stable Linux birth timestamps")
	}
	return DirectoryIdentity{Device: st.Dev, Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec)}, nil
}

func CheckMount(home string, location Location) (string, error) {
	if !ValidPath(location.Path) {
		return "", errors.New("pier path must identify a named pier inside an exclusive instance directory")
	}
	root, relative, err := Resolve(home, location.Path)
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, MountPath(relative))
	// Validate every ancestor, including grouping directories.
	parent := root
	if _, err := directory(parent); err != nil {
		return "", err
	}
	for _, part := range strings.Split(MountPath(relative), "/") {
		parent = filepath.Join(parent, part)
		if _, err := directory(parent); err != nil {
			return "", err
		}
	}
	i, err := IdentifyMount(p)
	if err != nil || location.MountIdentity.Inode == 0 || !i.SameObject(location.MountIdentity) {
		return "", errors.New("registered mount root is missing or replaced")
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(relative) || !e.IsDir() {
			return "", errors.New("a ship mount root may contain only its pier directory")
		}
	}
	return p, nil
}

// PrepareMount never adopts an existing unrecorded directory or creates a pier.
func PrepareMount(home string, location *Location) (string, error) {
	if location == nil || !ValidPath(location.Path) {
		return "", errors.New("missing permanent pier location")
	}
	if location.MountIdentity.Inode != 0 {
		return CheckMount(home, *location)
	}
	p, err := Parents(home, location.Path)
	if err != nil {
		return "", err
	}
	mount := filepath.Dir(p)
	if err = os.Mkdir(mount, 0700); err != nil {
		return "", errors.New("mount root already exists or cannot be created; review it instead of adopting it")
	}
	location.MountIdentity, err = IdentifyMount(mount)
	if err == nil {
		err = SyncDestinationParents(home, *location, nil)
	}
	return mount, err
}

func Identify(p string) (Identity, error) {
	return identifyMode(p, ownerWorkspace(p))
}

func identifyMode(p string, ownerMode bool) (Identity, error) {
	i, err := directoryMode(p, ownerMode)
	if err != nil {
		return Identity{}, err
	}
	u, err := directory(filepath.Join(p, ".urb"))
	if err != nil {
		return Identity{}, err
	}
	a, b := i.Sys().(*syscall.Stat_t), u.Sys().(*syscall.Stat_t)
	birth := func(name string) (string, error) {
		var x unix.Statx_t
		if err := unix.Statx(unix.AT_FDCWD, name, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &x); err != nil {
			return "", err
		}
		if x.Mask&unix.STATX_BTIME == 0 {
			return "", errors.New("pier storage requires stable Linux directory birth timestamps")
		}
		return fmt.Sprintf("%d:%d", x.Btime.Sec, x.Btime.Nsec), nil
	}
	created, err := birth(p)
	if err != nil {
		return Identity{}, err
	}
	urbCreated, err := birth(filepath.Join(p, ".urb"))
	if err != nil {
		return Identity{}, err
	}
	return Identity{Device: a.Dev, Inode: a.Ino, UrbDevice: b.Dev, UrbInode: b.Ino, Birth: created, UrbBirth: urbCreated}, nil
}

func Fingerprint(i Identity) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%d:%d:%s:%s", i.Device, i.Inode, i.UrbDevice, i.UrbInode, i.Birth, i.UrbBirth))))
}

func Check(home string, location Location) (string, error) {
	if _, err := CheckMount(home, location); err != nil {
		return "", err
	}
	p, err := Path(home, location.Path)
	if err != nil {
		return "", err
	}
	i, err := Identify(p)
	if err != nil {
		return "", err
	}
	if location.Identity.Inode == 0 || location.Uncertain || !i.SameObject(location.Identity) {
		return "", errors.New("registered pier directory was replaced or needs location review")
	}
	return p, nil
}

func Busy(p string) bool {
	for _, name := range []string{".vere.lock", ".urb/vere.lock", ".urb/lock"} {
		if _, err := os.Lstat(filepath.Join(p, name)); !errors.Is(err, os.ErrNotExist) {
			return true
		}
	}
	return false
}

// Inspect active state metadata, not arbitrary pier files or key contents.
// Docked .run/.bin conveniences outside .urb remain inert and need not be copied.
func CheckStateFiles(ctx context.Context, p string) error {
	count := 0
	for _, sub := range []string{".urb/log", ".urb/chk"} {
		root := filepath.Join(p, sub)
		if _, err := directory(root); err != nil {
			return err
		}
		err := filepath.WalkDir(root, func(name string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			count++
			if count > 8192 {
				return errors.New("pier state entry limit exceeded")
			}
			i, err := e.Info()
			if err != nil {
				return err
			}
			s, ok := i.Sys().(*syscall.Stat_t)
			if !ok || s.Uid != uint32(os.Geteuid()) || i.Mode().Perm()&0022 != 0 || (!i.IsDir() && (!i.Mode().IsRegular() || s.Nlink != 1)) {
				return errors.New("pier state contains linked, shared, foreign or special files")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func Scan(ctx context.Context, home string) ([]Candidate, error) {
	roots, err := Roots(home)
	if err != nil {
		return nil, err
	}
	out := []Candidate{}
	for _, root := range roots {
		if root.ID != "default" {
			if _, _, err := Resolve(home, Qualify(root.ID, "fakeships/check/pier")); err != nil {
				return nil, err
			}
		}
		found, err := scanRoot(ctx, home, root)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	return out, nil
}

func scanRoot(ctx context.Context, home string, storage StorageRoot) ([]Candidate, error) {
	out := []Candidate{}
	root := storage.Path
	if _, err := directory(root); err != nil {
		return out, err
	}
	visited := 0
	var walk func(string, int) error
	walk = func(relative string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > 4096 || len(out) > 128 {
			return errors.New("pier discovery limit exceeded")
		}
		p := filepath.Join(root, relative)
		if _, err := directory(p); err != nil {
			if info, statErr := os.Lstat(p); statErr == nil && info.IsDir() && ValidPath(relative) {
				out = append(out, Candidate{Path: Qualify(storage.ID, relative), Kind: Kind(relative), Problem: "Directory ownership or permissions need review; files must belong to the runner's Unix owner."})
			}
			return nil
		}
		if _, err := os.Lstat(filepath.Join(p, ".urb")); !errors.Is(err, os.ErrNotExist) {
			if !ValidPath(relative) {
				return nil
			}
			c := Candidate{Path: Qualify(storage.ID, relative), Kind: Kind(relative), Busy: Busy(p)}
			i, err := Identify(p)
			mount, rootErr := IdentifyMount(filepath.Dir(p))
			if rootErr == nil {
				_, rootErr = CheckMount(home, Location{Path: c.Path, MountIdentity: mount})
			}
			if err != nil || rootErr != nil {
				c.Problem = "Pier metadata is missing, linked, unsupported by this filesystem, or has incompatible ownership/permissions."
			} else {
				c.Identity = i
				c.MountIdentity = mount
				c.Fingerprint = Fingerprint(i)
			}
			out = append(out, c)
			return nil // Never traverse the pier's contents, keys or event databases.
		}
		if depth >= 8 {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		entries, err := f.ReadDir(4097 - visited)
		f.Close()
		if err != nil && err != io.EOF {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() || !component.MatchString(e.Name()) {
				continue
			}
			if err := walk(filepath.Join(relative, e.Name()), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, prefix := range Categories[:len(Categories)-1] {
		if err := walk(prefix, len(strings.Split(prefix, "/"))); err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Move is an atomic same-filesystem, no-replacement rename. It never deletes a
// destination, follows a symlink or falls back to an unverified cross-device copy.
func Move(source, destination string) error {
	if !component.MatchString(filepath.Base(source)) || !component.MatchString(filepath.Base(destination)) {
		return errors.New("move requires named pier directories")
	}
	if _, err := Identify(source); err != nil {
		return err
	}
	if Busy(source) {
		return errors.New("stop the pier and resolve its Vere lock before moving it")
	}
	if filepath.Dir(source) == filepath.Dir(destination) {
		return RenameDirectory(source, destination)
	}
	if filepath.Base(source) != filepath.Base(destination) {
		return errors.New("keep the pier directory name when moving its exclusive parent")
	}
	// Move the whole exclusive mount root, retaining its identity and contents.
	source, destination = filepath.Dir(source), filepath.Dir(destination)
	open := func(p string) (*os.File, error) {
		if _, err := directory(p); err != nil {
			return nil, err
		}
		return os.OpenFile(p, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	}
	a, err := open(filepath.Dir(source))
	if err != nil {
		return err
	}
	defer a.Close()
	b, err := open(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer b.Close()
	if err := unix.Renameat2(int(a.Fd()), filepath.Base(source), int(b.Fd()), filepath.Base(destination), unix.RENAME_NOREPLACE); err != nil {
		return err
	}
	return errors.Join(a.Sync(), b.Sync())
}
