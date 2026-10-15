//go:build linux

package pierstore

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"golang.org/x/sys/unix"
)

const rootsFile = ".omabit-storage.json"

// HostPath rejects symlink ancestors. Roots may live on mounted disks, but their
// parent chain must not be writable by other unprivileged users.
func HostPath(p string) error {
	return hostPathMode(p, ownerWorkspace(p))
}

func hostPathMode(p string, ownerMode bool) error {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p || p == "/" || strings.ContainsAny(p, "\x00\r\n,\"") || len(p) > 2048 {
		return errors.New("use a clean absolute host directory path")
	}
	for parent := p; ; parent = filepath.Dir(parent) {
		i, err := os.Lstat(parent)
		if err != nil {
			return err
		}
		ownerWritable := false
		if s, ok := i.Sys().(*syscall.Stat_t); ok && s.Uid == uint32(os.Geteuid()) && ownerMode && i.Mode().Perm()&0002 == 0 {
			ownerWritable = true
		}
		if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 || i.Mode().Perm()&0022 != 0 && i.Mode()&os.ModeSticky == 0 && !runnerOnlyACLWrite(parent) && !ownerWritable {
			return errors.New("host path ancestors must be real directories without group/world write access")
		}
		if parent == "/" {
			return nil
		}
	}
}

// An ACL granting this service UID access may set the mode's group-write mask
// even when no group can write. Accept that narrow grant, never broad writers.
func runnerOnlyACLWrite(path string) bool {
	data := make([]byte, 4096)
	n, err := unix.Getxattr(path, "system.posix_acl_access", data)
	if err != nil || n < 4 || (n-4)%8 != 0 || binary.LittleEndian.Uint32(data[:4]) != 2 {
		return false
	}
	data = data[:n]
	mask := uint16(7)
	for i := 4; i < n; i += 8 {
		if binary.LittleEndian.Uint16(data[i:]) == 0x10 {
			mask = binary.LittleEndian.Uint16(data[i+2:])
		}
	}
	for i := 4; i < n; i += 8 {
		tag, perm, id := binary.LittleEndian.Uint16(data[i:]), binary.LittleEndian.Uint16(data[i+2:]), binary.LittleEndian.Uint32(data[i+4:])
		if perm&^uint16(7) != 0 {
			return false
		}
		switch tag {
		case 1, 0x10: // owner and mask do not grant access to additional users
		case 2:
			if perm&mask&2 != 0 && id != uint32(os.Geteuid()) && id != 0 {
				return false
			}
		case 4, 8:
			if perm&mask&2 != 0 {
				return false
			}
		case 0x20:
			if perm&2 != 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func Roots(home string) ([]StorageRoot, error) {
	defaultRoot, err := ownerhost.DefaultRoot(home)
	if err != nil {
		return nil, err
	}
	out := []StorageRoot{{ID: "default", Path: defaultRoot}}
	if _, err := directory(home); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(home, rootsFile), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || i.Size() > 64<<10 {
		return nil, errors.New("unsafe storage registry")
	}
	var roots []StorageRoot
	d := json.NewDecoder(io.LimitReader(f, (64<<10)+1))
	d.DisallowUnknownFields()
	if err := d.Decode(&roots); err != nil {
		return nil, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || len(roots) > 32 {
		return nil, errors.New("invalid storage registry")
	}
	seen := map[string]bool{"default": true}
	for _, r := range roots {
		if !component.MatchString(r.ID) || seen[r.ID] || !filepath.IsAbs(r.Path) || filepath.Clean(r.Path) != r.Path || r.Identity.Inode == 0 {
			return nil, errors.New("invalid storage root")
		}
		seen[r.ID] = true
		out = append(out, r)
	}
	return out, nil
}

// RegisterRoot adds a location, never retargets an existing location or chmods
// unrelated paths. The operator must first have OS permission at that location.
func RegisterRoot(home, id, path string) (StorageRoot, error) {
	return registerRoot(home, id, path, ownerhost.OwnsHome(home))
}

func registerRoot(home, id, path string, ownerMode bool) (StorageRoot, error) {
	if !component.MatchString(id) || id == "default" {
		return StorageRoot{}, errors.New("choose a storage name other than default")
	}
	if ownerMode {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
			return StorageRoot{}, errors.New("use an absolute storage directory")
		}
		var err error
		path, err = canonicalNewPath(path)
		if err != nil {
			return StorageRoot{}, err
		}
		if ownerPrivateOverlap(home, path) {
			return StorageRoot{}, errors.New("storage cannot overlap private management state or the home root")
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return StorageRoot{}, err
		}
	}
	if err := hostPathMode(path, ownerMode); err != nil {
		return StorageRoot{}, fmt.Errorf("storage access: %w; select a directory writable by the runner's Unix owner", err)
	}
	identity, err := identifyMountMode(path, ownerMode)
	if err != nil {
		return StorageRoot{}, err
	}
	if _, err := os.Lstat(filepath.Join(path, ".urb")); !errors.Is(err, os.ErrNotExist) {
		return StorageRoot{}, errors.New("select a storage directory, not an existing pier")
	}
	roots, err := Roots(home)
	if err != nil {
		return StorageRoot{}, err
	}
	for _, root := range roots {
		if root.ID == id && root.Path == path && root.Identity.SameObject(identity) {
			return root, nil
		}
		if root.ID == id || Overlap(root.Path, path) {
			return StorageRoot{}, errors.New("storage name or overlapping path is already configured")
		}
	}
	privateOverlap := Overlap(home, path)
	if ownerMode {
		privateOverlap = ownerPrivateOverlap(home, path)
	}
	if privateOverlap || len(roots) > 32 {
		return StorageRoot{}, errors.New("storage overlaps private management state or exceeds the 32-location limit")
	}
	root := StorageRoot{ID: id, Path: path, Identity: identity}
	if err := initRootMode(path, ownerMode); err != nil {
		return StorageRoot{}, err
	}
	data, err := json.Marshal(append(roots[1:], root))
	if err != nil {
		return StorageRoot{}, err
	}
	if len(data) > 64<<10 {
		return StorageRoot{}, errors.New("storage registry exceeds its byte limit")
	}
	f, err := os.CreateTemp(home, ".storage-*")
	if err != nil {
		return StorageRoot{}, err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	err = errors.Join(err, f.Sync(), f.Close())
	if err != nil {
		return StorageRoot{}, err
	}
	if err = os.Rename(f.Name(), filepath.Join(home, rootsFile)); err != nil {
		return StorageRoot{}, err
	}
	d, err := os.Open(home)
	if err != nil {
		return StorageRoot{}, err
	}
	return root, errors.Join(d.Sync(), d.Close())
}

func ownerPrivateOverlap(home, path string) bool {
	return path == home || strings.HasPrefix(home, path+"/") || Overlap(filepath.Join(home, ".local/share/urbit-runner"), path) || Overlap(filepath.Join(home, ".config/omabit/setup"), path)
}

// Resolve the existing prefix before creating a user's new storage directory.
// The registry pins the real directory, not a mutable shortcut/symlink.
func canonicalNewPath(path string) (string, error) {
	parent := path
	var suffix []string
	for {
		_, err := os.Lstat(parent)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		suffix = append(suffix, filepath.Base(parent))
		parent = filepath.Dir(parent)
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	for n := len(suffix) - 1; n >= 0; n-- {
		resolved = filepath.Join(resolved, suffix[n])
	}
	return resolved, nil
}

func Resolve(home, relative string) (string, string, error) {
	id, rel := Split(relative)
	if id == "" {
		root, err := ownerhost.DefaultRoot(home)
		return root, rel, err
	}
	roots, err := Roots(home)
	if err != nil {
		return "", "", err
	}
	for _, r := range roots {
		if r.ID != id {
			continue
		}
		if err = HostPath(r.Path); err != nil {
			return "", "", err
		}
		identity, err := IdentifyMount(r.Path)
		if err != nil || !identity.SameObject(r.Identity) {
			return "", "", errors.New("storage root is missing or replaced; restore the original mounted location")
		}
		return r.Path, rel, nil
	}
	return "", "", errors.New("unknown storage location")
}

func Absolute(home, relative string) (string, error) {
	if !ValidPath(relative) {
		return "", errors.New("invalid pier path")
	}
	root, rel, err := Resolve(home, relative)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}
