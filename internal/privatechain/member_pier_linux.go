//go:build linux

package privatechain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

var errMemberPier = errors.New("retained private member pier or stopped-state metadata differs; preserve it for review")

// MemberPierVerifier holds the selected mount, pier and .urb objects. It reads
// only metadata beneath that member; no database/key payload, peer pier or engine.
// Absence of lock files is not proof of process exit. The lifecycle driver must
// separately establish acknowledged Stop, exact engine ownership and the runner
// mutation fence. This helper grants no adoption, repair, sync or dispatch.
type MemberPierVerifier struct {
	mu       sync.Mutex
	mount    *MemberMountVerifier
	ship     string
	expected [2]pierstore.DirectoryIdentity
	held     []*os.File
}

func OpenMemberPierVerifier(ctx context.Context, directory string, p Plan, w Workspace, instance string, id pierstore.Identity) (*MemberPierVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	want := [2]pierstore.DirectoryIdentity{{Device: id.Device, Inode: id.Inode, Birth: id.Birth}, {Device: id.UrbDevice, Inode: id.UrbInode, Birth: id.UrbBirth}}
	if !workspaceIdentity(want[0]) || !workspaceIdentity(want[1]) || want[0].SameObject(want[1]) {
		return nil, errMemberPier
	}
	mount, err := OpenMemberMountVerifier(ctx, directory, p, w, instance)
	if err != nil {
		return nil, errMemberPier
	}
	v := &MemberPierVerifier{mount: mount, expected: want}
	for _, m := range w.Mounts {
		if m.InstanceID == instance {
			v.ship = strings.TrimPrefix(m.Ship, "~")
		}
	}
	for _, reserved := range mount.expected {
		if want[0].SameObject(reserved) || want[1].SameObject(reserved) {
			v.Close()
			return nil, errMemberPier
		}
	}
	v.held, err = v.openObjects(ctx)
	if err == nil {
		err = v.Verify(ctx)
	}
	if err != nil {
		v.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errMemberPier
	}
	return v, nil
}

// Ignore atime: a metadata walk may update it without changing accepted state.
type memberPierStat struct {
	Device, Inode, Size uint64
	UID, GID, Links     uint32
	Mode                uint16
	Birth, Mtime, Ctime unix.StatxTimestamp
}

func memberPierMetadata(f *os.File) (memberPierStat, error) {
	var x unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if f == nil || unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW, mask, &x) != nil || x.Mask&uint32(mask) != uint32(mask) || x.Uid != uint32(os.Geteuid()) || x.Mode&07022 != 0 {
		return memberPierStat{}, errMemberPier
	}
	kind := x.Mode & unix.S_IFMT
	if kind != unix.S_IFDIR && (kind != unix.S_IFREG || x.Nlink != 1) {
		return memberPierStat{}, errMemberPier
	}
	return memberPierStat{Device: unix.Mkdev(x.Dev_major, x.Dev_minor), Inode: x.Ino, Size: x.Size, UID: x.Uid, GID: x.Gid, Links: x.Nlink, Mode: x.Mode, Birth: x.Btime, Mtime: x.Mtime, Ctime: x.Ctime}, nil
}
func (s memberPierStat) identity() pierstore.DirectoryIdentity {
	return pierstore.DirectoryIdentity{Device: s.Device, Inode: s.Inode, Birth: fmt.Sprintf("%d:%d", s.Birth.Sec, s.Birth.Nsec)}
}
func memberPierChild(parent *os.File, name string, directory bool) (*os.File, error) {
	flags := unix.O_PATH | unix.O_NOFOLLOW | unix.O_CLOEXEC
	if directory {
		flags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	}
	fd, err := unix.Openat2(int(parent.Fd()), name, &unix.OpenHow{Flags: uint64(flags), Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
	if err != nil {
		return nil, errMemberPier
	}
	f := os.NewFile(uintptr(fd), "private member metadata")
	if _, err = memberPierMetadata(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func (v *MemberPierVerifier) openObjects(ctx context.Context) (files []*os.File, err error) {
	defer func() {
		if err != nil {
			_ = closeMemberMountFiles(files)
			files = nil
		}
	}()
	if v.mount == nil || v.mount.Verify(ctx) != nil {
		return nil, errMemberPier
	}
	if workspaceContents(v.mount.held[3], []string{v.ship}) != nil {
		return nil, errMemberPier
	}
	parent := v.mount.held[3]
	for i, name := range []string{v.ship, ".urb"} {
		var f *os.File
		f, err = memberPierChild(parent, name, true)
		if err != nil {
			return
		}
		files = append(files, f)
		var st memberPierStat
		st, err = memberPierMetadata(f)
		if err != nil {
			return
		}
		if !v.expected[i].SameObject(st.identity()) {
			err = errMemberPier
			return
		}
		parent = f
	}
	err = ctx.Err()
	return
}
func (v *MemberPierVerifier) verifyLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v.mount == nil || len(v.held) != 2 {
		return errMemberPier
	}
	named, err := v.openObjects(ctx)
	if err != nil {
		return err
	}
	defer closeMemberMountFiles(named)
	for i := range v.held {
		held, e := memberPierMetadata(v.held[i])
		current, e2 := memberPierMetadata(named[i])
		if e != nil || e2 != nil || held.identity() != current.identity() || !v.expected[i].SameObject(held.identity()) {
			return errMemberPier
		}
	}
	if v.mount.Verify(ctx) != nil {
		return errMemberPier
	}
	return ctx.Err()
}

// Verify checks identities/layout only and can also be used while the selected
// ship is running. It does not establish stopped-state or content durability.
func (v *MemberPierVerifier) Verify(ctx context.Context) error {
	if v == nil {
		return errMemberPier
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.verifyLocked(ctx)
}
func (v *MemberPierVerifier) locksAbsent() error {
	for _, item := range []struct {
		file *os.File
		name string
	}{{v.held[0], ".vere.lock"}, {v.held[1], "vere.lock"}, {v.held[1], "lock"}} {
		var st unix.Stat_t
		if err := unix.Fstatat(int(item.file.Fd()), item.name, &st, unix.AT_SYMLINK_NOFOLLOW); !errors.Is(err, unix.ENOENT) {
			return errMemberPier
		}
	}
	return nil
}

// VerifyStoppedState checks the established log/checkpoint metadata boundary,
// with 8192 entries and depth128 limits. It never reads database contents, removes
// stale locks or claims process quiescence. The caller supplies that authority.
func (v *MemberPierVerifier) VerifyStoppedState(ctx context.Context) error {
	if v == nil {
		return errMemberPier
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.verifyLocked(ctx); err != nil {
		return err
	}
	if err := v.locksAbsent(); err != nil {
		return err
	}
	count := 0
	for _, name := range []string{"log", "chk"} {
		f, err := memberPierChild(v.held[1], name, true)
		if err != nil {
			return err
		}
		before, err := memberPierMetadata(f)
		if err == nil {
			err = memberPierWalk(ctx, f, &count, 0)
		}
		named, e := memberPierChild(v.held[1], name, true)
		if e == nil {
			after, e2 := memberPierMetadata(named)
			named.Close()
			if e2 != nil || before != after {
				e = errMemberPier
			}
		}
		f.Close()
		if err != nil {
			return err
		}
		if e != nil {
			return e
		}
	}
	if err := v.verifyLocked(ctx); err != nil {
		return err
	}
	return v.locksAbsent()
}
func memberPierWalk(ctx context.Context, dir *os.File, count *int, depth int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	*count++
	if *count > 8192 || depth > 128 {
		return errMemberPier
	}
	before, err := memberPierMetadata(dir)
	if err != nil {
		return err
	}
	listing, err := memberPierChild(dir, ".", true)
	if err != nil {
		return err
	}
	defer listing.Close()
	for {
		names, e := listing.Readdirnames(128)
		if e != nil && e != io.EOF {
			return errMemberPier
		}
		for _, name := range names {
			if err = ctx.Err(); err != nil {
				return err
			}
			child, e := memberPierChild(dir, name, false)
			if e != nil {
				return e
			}
			st, e := memberPierMetadata(child)
			child.Close()
			if e != nil {
				return e
			}
			if st.Mode&unix.S_IFMT == unix.S_IFDIR {
				nested, e := memberPierChild(dir, name, true)
				if e != nil {
					return e
				}
				seen, e := memberPierMetadata(nested)
				if e == nil && seen != st {
					e = errMemberPier
				}
				if e == nil {
					e = memberPierWalk(ctx, nested, count, depth+1)
				}
				nested.Close()
				if e != nil {
					return e
				}
			} else {
				*count++
				if *count > 8192 {
					return errMemberPier
				}
			}
			named, e := memberPierChild(dir, name, false)
			if e != nil {
				return e
			}
			after, e := memberPierMetadata(named)
			named.Close()
			if e != nil || after != st {
				return errMemberPier
			}
		}
		if e == io.EOF {
			break
		}
	}
	after, err := memberPierMetadata(dir)
	if err != nil || after != before {
		return errMemberPier
	}
	return ctx.Err()
}
func (v *MemberPierVerifier) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	files := v.held
	v.held = nil
	err := closeMemberMountFiles(files)
	if v.mount != nil {
		err = errors.Join(err, v.mount.Close())
		v.mount = nil
	}
	return err
}
