//go:build linux

package privatechain

import (
	"context"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// InspectWorkspaceParent reads an existing backend-selected storage parent. It
// creates nothing and grants no mutation authority. Fresh worker setup must open
// it again against this exact identity before any journalled directory changes.
func InspectWorkspaceParent(ctx context.Context, directory string) (pierstore.DirectoryIdentity, error) {
	if err := ctx.Err(); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return pierstore.DirectoryIdentity{}, errWorkspaceTree
	}
	// Opening a directory with an all-zero expected identity is not permitted by
	// workspaceParent, so independently inspect then use its identity-bound reopen.
	f, err := openWorkspaceParentForInspection(directory)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	id, err := workspaceDirectoryID(f)
	closeErr := f.Close()
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if closeErr != nil {
		return pierstore.DirectoryIdentity{}, closeErr
	}
	checked, err := workspaceParent(directory, id)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if err = checked.Close(); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	if err = ctx.Err(); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	return id, nil
}

func openWorkspaceParentForInspection(directory string) (*os.File, error) {
	real, err := filepath.EvalSymlinks(directory)
	if err != nil || real != directory {
		return nil, errWorkspaceTree
	}
	fd, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, errWorkspaceTree
	}
	return os.NewFile(uintptr(fd), "private workspace inspection"), nil
}
