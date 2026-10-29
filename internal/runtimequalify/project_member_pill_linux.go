//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// An in-memory identity for one already hash-verified, private, immutable file.
// Never persisted or reused after reopening an operation. Atime is intentionally
// absent: reading the pill can update atime without changing its contents.
type projectPillSnapshot struct {
	Device, Inode, Size uint64
	UID, GID, Links     uint32
	Mode                uint16
	Birth, Mtime, Ctime unix.StatxTimestamp
}

func (e *projectMemberLaunchEngine) workspaceSnapshot(ctx context.Context) (projectPillSnapshot, error) {
	refuse := func() (projectPillSnapshot, error) {
		return projectPillSnapshot{}, errors.New("member workspace or immutable boot artifact changed")
	}
	if ctx.Err() != nil {
		return refuse()
	}
	q := e.workspaceQualifier()
	if q.owned(q.r.Directory, os.ModeDir, true) != nil || q.owned(q.r.PillPath, 0, true) != nil {
		return refuse()
	}
	id, err := pierstore.IdentifyMount(q.r.Directory)
	if err != nil || id != e.workspace {
		return refuse()
	}
	entries, err := os.ReadDir(q.r.Directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(q.r.PillPath) {
		return refuse()
	}
	var v unix.Statx_t
	mask := unix.STATX_BASIC_STATS | unix.STATX_BTIME
	if unix.Statx(unix.AT_FDCWD, q.r.PillPath, unix.AT_SYMLINK_NOFOLLOW, mask, &v) != nil || v.Mask&uint32(mask) != uint32(mask) || v.Mode != unix.S_IFREG|0400 || v.Nlink != 1 || v.Size != uint64(q.d.pillBytes) || ctx.Err() != nil {
		return refuse()
	}
	return projectPillSnapshot{Device: unix.Mkdev(v.Dev_major, v.Dev_minor), Inode: v.Ino, Size: v.Size, UID: v.Uid, GID: v.Gid, Links: v.Nlink, Mode: v.Mode, Birth: v.Btime, Mtime: v.Mtime, Ctime: v.Ctime}, nil
}
func (e *projectMemberLaunchEngine) checkWorkspace(ctx context.Context) error {
	e.verifiedPill = projectPillSnapshot{}
	before, err := e.workspaceSnapshot(ctx)
	if err != nil {
		return err
	}
	if err = bootpill.Verify(ctx, e.contract.pillPath, e.pillMeta()); err != nil {
		return err
	}
	after, err := e.workspaceSnapshot(ctx)
	if err != nil || before != after {
		return errors.New("member boot artifact changed during hash verification")
	}
	e.verifiedPill = after
	return nil
}
func (e *projectMemberLaunchEngine) checkVerifiedWorkspace(ctx context.Context) error {
	actual, err := e.workspaceSnapshot(ctx)
	if err != nil || e.verifiedPill == (projectPillSnapshot{}) || actual != e.verifiedPill {
		return errors.New("member dispatch requires the unchanged hash-verified boot artifact")
	}
	return nil
}
