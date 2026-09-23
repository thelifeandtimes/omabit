//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

type identity struct {
	uid, gid, accountUID, accountGID int
	groups                           []int
	account, home, group, platform   string
}
type node struct {
	mode     fs.FileMode
	uid, gid uint32
	links    uint64
}

func currentIdentity() (identity, error) {
	i := identity{uid: os.Geteuid(), gid: os.Getegid(), platform: runtime.GOOS + "/" + runtime.GOARCH}
	u, err := user.Lookup(rootless.AccountName)
	if err != nil {
		return i, err
	}
	i.account, i.home = u.Username, u.HomeDir
	if i.accountUID, err = strconv.Atoi(u.Uid); err != nil {
		return i, err
	}
	if i.accountGID, err = strconv.Atoi(u.Gid); err != nil {
		return i, err
	}
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return i, err
	}
	i.group = g.Name
	i.groups, err = os.Getgroups()
	return i, err
}

func statNode(path string) (node, error) {
	f, err := os.Lstat(path)
	if err != nil {
		return node{}, err
	}
	s, ok := f.Sys().(*syscall.Stat_t)
	if !ok {
		return node{}, errors.New("missing Linux ownership metadata")
	}
	return node{f.Mode(), s.Uid, s.Gid, s.Nlink}, nil
}

func (q *qualifier) owned(path string, kind fs.FileMode, private bool) error {
	n, err := q.d.stat(path)
	if err != nil {
		return errors.New("account path metadata unavailable")
	}
	if n.mode.Type() != kind {
		return errors.New("account path has wrong type or is a symlink")
	}
	if n.uid != uint32(q.uid) || n.gid != uint32(q.gid) {
		return errors.New("account path UID/GID does not match the dedicated account")
	}
	if n.mode.Perm()&0022 != 0 || n.mode&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return errors.New("account file/directory is group/world writable or has special permissions")
	}
	if private && kind == os.ModeDir && n.mode.Perm() != 0700 {
		return errors.New("private directory requires mode 0700")
	}
	if private && kind != os.ModeDir && n.mode.Perm()&0077 != 0 {
		return errors.New("private file forbids group/world permissions")
	}
	return nil
}

// Ancestors outside the private home/runtime must be trusted root-owned paths.
func (q *qualifier) ancestors(path string) error {
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		n, err := q.d.stat(p)
		if err != nil || !n.mode.IsDir() || n.uid != 0 || n.mode.Perm()&0022 != 0 {
			return errors.New("unsafe host path ancestor")
		}
		if p == "/" {
			return nil
		}
	}
}

// Repeated before EVERY Docker invocation, including cancellation cleanup.
func (q *qualifier) boundary() error {
	return q.accountBoundary(true)
}

func (q *qualifier) accountBoundary(requireEngine bool) error {
	i, err := q.d.identity()
	if err != nil || i.platform != "linux/amd64" || i.uid <= 0 || i.gid <= 0 || i.uid != i.accountUID || i.gid != i.accountGID || i.account != rootless.AccountName || i.group != rootless.AccountName || i.home != q.d.home {
		return errors.New("qualification requires the dedicated private linux/amd64 account")
	}
	for _, gid := range i.groups {
		if gid != i.gid {
			return errors.New("supplementary groups are forbidden")
		}
	}
	if q.uid != 0 && (q.uid != i.uid || q.gid != i.gid) {
		return errors.New("account identity changed")
	}
	q.uid, q.gid = i.uid, i.gid
	run := q.d.runtimeDir
	if run == "" {
		run = filepath.Join("/run/user", strconv.Itoa(i.uid))
	}
	q.socket = filepath.Join(run, rootless.SocketName)
	for _, p := range []string{q.d.home, run} {
		if err := q.ancestors(p); err != nil {
			return fmt.Errorf("private directory %s: %w", p, err)
		}
		if err := q.owned(p, os.ModeDir, true); err != nil {
			return fmt.Errorf("private directory %s: %w", p, err)
		}
	}
	if !requireEngine {
		return nil
	}
	// Docker marks runtime sockets sticky to protect them from periodic cleanup.
	// That flag grants no access. Keep 0600/0660 access permissions, private
	// ownership and the 0700 runtime parent, and reject all other extra flags.
	socket, err := q.d.stat(q.socket)
	if err != nil || socket.mode.Type() != os.ModeSocket {
		return errors.New("private Docker socket missing, wrong type or symlink")
	}
	if socket.uid != uint32(q.uid) || socket.gid != uint32(q.gid) {
		return errors.New("private Docker socket UID/GID does not match the dedicated account")
	}
	accessMode := socket.mode &^ os.ModeSticky
	if accessMode != os.ModeSocket|0600 && accessMode != os.ModeSocket|0660 {
		return fmt.Errorf("private Docker socket requires mode 0600 or 0660 with an optional sticky bit; got %s (permissions %04o)", socket.mode, socket.mode.Perm())
	}
	if err := q.ancestors("/usr/bin/docker"); err != nil {
		return err
	}
	n, err := q.d.stat("/usr/bin/docker")
	if err != nil || !n.mode.IsRegular() || n.uid != 0 || n.mode.Perm()&0022 != 0 || n.mode.Perm()&0111 == 0 || n.mode&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return errors.New("unsafe /usr/bin/docker executable")
	}
	return nil
}

func (q *qualifier) storagePaths() []string {
	return []string{filepath.Join(q.d.home, ".local"), filepath.Join(q.d.home, ".local/share"), filepath.Join(q.d.home, ".local/share/urbit-runner"), q.base}
}

func (q *qualifier) prevalidate() (uint64, uint64, error) {
	if err := q.boundary(); err != nil {
		return 0, 0, err
	}
	q.base = filepath.Join(q.d.home, ".local/share/urbit-runner/qualification")
	nearest := q.d.home
	for _, p := range q.storagePaths() {
		_, err := q.d.stat(p)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err := q.owned(p, os.ModeDir, false); err != nil {
			return 0, 0, err
		}
		nearest = p
	}
	mem, disk, err := q.d.admission(nearest)
	if err != nil || mem < 5<<30 || disk < 8<<30 {
		return 0, 0, errors.New("admission requires MemAvailable >= 5 GiB and free disk >= 8 GiB")
	}
	return mem, disk, nil
}

func (q *qualifier) lockStorage() (func(), error) {
	for _, p := range q.storagePaths() {
		err := os.Mkdir(p, 0700)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return nil, errors.New("cannot prepare qualification storage")
		}
		if err == nil {
			if err = os.Chmod(p, 0700); err != nil {
				return nil, errors.New("cannot secure qualification storage")
			}
		}
		if err := q.owned(p, os.ModeDir, false); err != nil {
			return nil, err
		}
	}
	path := filepath.Join(q.base, ".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, errors.New("cannot open qualification lock")
	}
	n, statErr := q.d.stat(path)
	if err = q.owned(path, 0, true); err != nil || statErr != nil || n.links != 1 {
		f.Close()
		return nil, errors.New("unsafe qualification lock")
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("qualification lock is busy or unavailable")
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}

func available(path string) (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	var mem uint64
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 3 && f[0] == "MemAvailable:" && f[2] == "kB" {
			mem, err = strconv.ParseUint(f[1], 10, 64)
			if err != nil || mem > (^uint64(0))/1024 {
				return 0, 0, errors.New("invalid MemAvailable")
			}
			mem *= 1024
		}
	}
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return 0, 0, err
	}
	if s.Bsize <= 0 || s.Bavail > (^uint64(0))/uint64(s.Bsize) {
		return 0, 0, errors.New("invalid filesystem capacity")
	}
	return mem, s.Bavail * uint64(s.Bsize), nil
}
