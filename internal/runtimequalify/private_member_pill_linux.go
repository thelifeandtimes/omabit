//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"golang.org/x/sys/unix"
)

var errPrivateMemberPill = errors.New("selected private member boot pill or workspace changed; retain state for review")

// This handle grants no preparation, sync or dispatch. The workspace identity
// must come from the accepted member's separate staging operation. Full hashing
// happens on open; subsequent checks hold those exact objects and verify their
// immutable metadata, avoiding a repeated large read inside live freshness gates.
// Closing requires a fresh full hash on any later open. Same-UID code is trusted.
type privateMemberPillVerifier struct {
	mu            sync.Mutex
	root, runtime string
	workspace     pierstore.DirectoryIdentity
	expected      protocol.BootPill
	files         []*os.File
	initial       projectPillSnapshot
	verified      bool
}

func openPrivateMemberPill(ctx context.Context, c privateMemberContract, workspace pierstore.DirectoryIdentity) (*privateMemberPillVerifier, error) {
	return openPrivateMemberPillWithMeta(ctx, c, workspace, protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes})
}

// Only package tests substitute a small inert artifact for the pinned pill.
func openPrivateMemberPillWithMeta(ctx context.Context, c privateMemberContract, workspace pierstore.DirectoryIdentity, meta protocol.BootPill) (*privateMemberPillVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := newPrivateMemberContract(c.selection, c.root, c.imageID)
	if err != nil || c.selection.Mode != "boot" || c.digest != canonical.digest || c.pillPath != canonical.pillPath || !validProjectDirectory(workspace) || !privateMemberBirth.MatchString(workspace.Birth) || !protocol.ValidBootPill(&meta) || !privateGatewayID(meta.SHA256) {
		return nil, errPrivateMemberPill
	}
	v := &privateMemberPillVerifier{root: c.root, runtime: c.selection.RuntimeID, workspace: workspace, expected: meta}
	files, snapshot, err := v.openSelected(ctx)
	if err != nil {
		return nil, err
	}
	v.files, v.initial = files, snapshot
	fail := func(err error) (*privateMemberPillVerifier, error) {
		v.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if err = v.current(ctx); err != nil {
		return fail(err)
	}
	h := sha256.New()
	n, err := io.Copy(h, privatePillReader{ctx: ctx, source: io.NewSectionReader(v.files[2], 0, meta.Bytes+1)})
	if err != nil || n != meta.Bytes || hex.EncodeToString(h.Sum(nil)) != meta.SHA256 {
		return fail(errPrivateMemberPill)
	}
	if err = v.current(ctx); err != nil {
		return fail(err)
	}
	v.verified = true
	return v, nil
}

type privatePillReader struct {
	ctx    context.Context
	source io.Reader
}

func (r privatePillReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}

func privatePillDirectory(f *os.File) (pierstore.DirectoryIdentity, error) {
	if !privateClientOwned(f, true) {
		return pierstore.DirectoryIdentity{}, errPrivateMemberPill
	}
	var st unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW, mask, &st) != nil || st.Mask&uint32(mask) != uint32(mask) {
		return pierstore.DirectoryIdentity{}, errPrivateMemberPill
	}
	return pierstore.DirectoryIdentity{Device: unix.Mkdev(st.Dev_major, st.Dev_minor), Inode: st.Ino, Birth: fmt.Sprintf("%d:%d", st.Btime.Sec, st.Btime.Nsec)}, nil
}
func privatePillChild(parent *os.File, name string, directory bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
	if directory {
		flags |= unix.O_DIRECTORY
	}
	fd, err := unix.Openat2(int(parent.Fd()), name, &unix.OpenHow{Flags: uint64(flags), Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV})
	if err != nil {
		return nil, errPrivateMemberPill
	}
	return os.NewFile(uintptr(fd), "private member pill object"), nil
}
func privatePillSnapshot(f *os.File, bytes int64) (projectPillSnapshot, error) {
	var st unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if unix.Statx(int(f.Fd()), "", unix.AT_EMPTY_PATH|unix.AT_SYMLINK_NOFOLLOW, mask, &st) != nil || st.Mask&uint32(mask) != uint32(mask) || st.Mode != unix.S_IFREG|0400 || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 || st.Size != uint64(bytes) {
		return projectPillSnapshot{}, errPrivateMemberPill
	}
	return projectPillSnapshot{Device: unix.Mkdev(st.Dev_major, st.Dev_minor), Inode: st.Ino, Size: st.Size, UID: st.Uid, GID: st.Gid, Links: st.Nlink, Mode: st.Mode, Birth: st.Btime, Mtime: st.Mtime, Ctime: st.Ctime}, nil
}
func privatePillClose(files []*os.File) error {
	var errs []error
	for i := len(files) - 1; i >= 0; i-- {
		errs = append(errs, files[i].Close())
	}
	return errors.Join(errs...)
}

func (v *privateMemberPillVerifier) openSelected(ctx context.Context) (files []*os.File, snapshot projectPillSnapshot, err error) {
	defer func() {
		if err != nil {
			_ = privatePillClose(files)
			files = nil
			snapshot = projectPillSnapshot{}
			if ctx.Err() != nil {
				err = ctx.Err()
			} else {
				err = errPrivateMemberPill
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return
	}
	var real string
	real, err = filepath.EvalSymlinks(v.root)
	if err != nil {
		return
	}
	if real != v.root {
		err = errPrivateMemberPill
		return
	}
	var root, workspace, pill *os.File
	root, err = privateClientFile(v.root, true)
	if err != nil {
		return
	}
	files = append(files, root)
	var rootID, pillID pierstore.DirectoryIdentity
	rootID, err = privatePillDirectory(root)
	if err != nil {
		return
	}
	workspace, err = privatePillChild(root, v.runtime, true)
	if err != nil {
		return
	}
	files = append(files, workspace)
	pillID, err = privatePillDirectory(workspace)
	if err != nil {
		return
	}
	if !v.workspace.SameObject(pillID) || rootID.SameObject(pillID) {
		err = errPrivateMemberPill
		return
	}
	var listing *os.File
	listing, err = privatePillChild(workspace, ".", true)
	if err != nil {
		return
	}
	names, e := listing.Readdirnames(2)
	listing.Close()
	if e != nil && e != io.EOF || len(names) != 1 || names[0] != candidate.PillName {
		err = errPrivateMemberPill
		return
	}
	pill, err = privatePillChild(workspace, candidate.PillName, false)
	if err != nil {
		return
	}
	files = append(files, pill)
	snapshot, err = privatePillSnapshot(pill, v.expected.Bytes)
	if err != nil {
		return
	}
	err = ctx.Err()
	return
}

func (v *privateMemberPillVerifier) current(ctx context.Context) error {
	if len(v.files) != 3 {
		return errPrivateMemberPill
	}
	current, snapshot, err := v.openSelected(ctx)
	if err != nil {
		return err
	}
	defer privatePillClose(current)
	if snapshot != v.initial {
		return errPrivateMemberPill
	}
	for i := 0; i < 2; i++ {
		held, e := privatePillDirectory(v.files[i])
		named, e2 := privatePillDirectory(current[i])
		if e != nil || e2 != nil || held != named {
			return errPrivateMemberPill
		}
	}
	held, err := privatePillSnapshot(v.files[2], v.expected.Bytes)
	if err != nil || held != v.initial {
		return errPrivateMemberPill
	}
	return ctx.Err()
}

func (v *privateMemberPillVerifier) Verify(ctx context.Context) error {
	if v == nil {
		return errPrivateMemberPill
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !v.verified {
		return errPrivateMemberPill
	}
	return v.current(ctx)
}
func (v *privateMemberPillVerifier) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	files := v.files
	v.files = nil
	v.initial = projectPillSnapshot{}
	v.verified = false
	return privatePillClose(files)
}
