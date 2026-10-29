// Package ownerhost records the explicitly installed, pier-owning Unix identity.
// It does not infer runtime authority from HOME or caller-supplied environment.
package ownerhost

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

const Path = "/etc/urbit-runner/owner.json"
const DefaultPiers = "/srv/urbit/piers"

type Profile struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	UID     uint32 `json:"uid"`
	GID     uint32 `json:"gid"`
	Home    string `json:"home"`
	Piers   string `json:"piers"`
}

var accountName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]{0,63}$`)

func (p Profile) Validate() error {
	if p.Version != 1 || !accountName.MatchString(p.Name) || p.Name == "root" || p.Name == "urbit-runner" || p.UID == 0 || p.GID == 0 {
		return errors.New("owner profile requires an existing non-root Unix user")
	}
	for _, path := range []string{p.Home, p.Piers} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" || strings.ContainsAny(path, "\x00\r\n\"'\\%") {
			return errors.New("owner profile paths must be clean absolute paths")
		}
	}
	return nil
}

func ForUser(name string) (Profile, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return Profile{}, err
	}
	uid, e1 := strconv.ParseUint(u.Uid, 10, 32)
	gid, e2 := strconv.ParseUint(u.Gid, 10, 32)
	p := Profile{Version: 1, Name: u.Username, UID: uint32(uid), GID: uint32(gid), Home: u.HomeDir, Piers: DefaultPiers}
	if e1 != nil || e2 != nil {
		return p, errors.New("invalid owner IDs")
	}
	return p, p.Validate()
}

func Read(path string) (*Profile, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || !i.Mode().IsRegular() || s.Uid != 0 || s.Nlink != 1 || i.Mode().Perm()&0022 != 0 || i.Size() > 8192 {
		return nil, errors.New("unsafe owner profile")
	}
	var p Profile
	d := json.NewDecoder(io.LimitReader(f, 8193))
	d.DisallowUnknownFields()
	if err = d.Decode(&p); err != nil {
		return nil, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, errors.New("invalid owner profile trailing data")
	}
	if err = p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func loadProfile(path string) (*Profile, error) {
	p, err := Read(path)
	if err != nil || p == nil {
		return p, err
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		i, err := os.Lstat(dir)
		if err != nil {
			return nil, err
		}
		s, ok := i.Sys().(*syscall.Stat_t)
		if !ok || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 || s.Uid != 0 || i.Mode().Perm()&0022 != 0 {
			return nil, errors.New("unsafe owner profile directory")
		}
		if dir == "/" {
			break
		}
	}
	return p, nil
}

func Load() (*Profile, error) {
	p, err := loadProfile(selectedProfilePath())
	if err != nil || p == nil {
		return p, err
	}
	return runtimeProfile(p), nil
}

func Current() (*Profile, error) {
	p, err := loadProfile(selectedProfilePath())
	if err != nil || p == nil {
		return p, err
	}
	u, err := ForUser(p.Name)
	if err != nil || u.UID != p.UID || u.GID != p.GID || u.Home != p.Home || uint32(os.Geteuid()) != p.UID {
		return nil, errors.New("runtime must run as the installed pier owner")
	}
	return runtimeProfile(p), nil
}

func OwnsHome(home string) bool {
	p, err := Current()
	return err == nil && p != nil && p.Home == home
}

func DefaultRoot(home string) (string, error) {
	p, err := Load()
	if err != nil {
		return "", err
	}
	if p != nil && p.Home == home {
		return p.Piers, nil
	}
	return filepath.Join(home, "piers"), nil
}
