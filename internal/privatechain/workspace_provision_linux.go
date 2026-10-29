//go:build linux

package privatechain

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"golang.org/x/sys/unix"
)

// ProvisionWorkspaceParent prepares only the shared empty grouping directory.
// The caller holds its host mutation/update fence and has proved that no retained
// workspace claims exist. It must never use this to replace a missing claimed
// parent. Project roots, keys, chains and piers remain separate journalled effects.
// An empty directory left before the first receipt may be verified and synced;
// any existing contents, aliases, ownership or permission mismatch refuse.
func ProvisionWorkspaceParent(ctx context.Context, directory string) (pierstore.DirectoryIdentity, error) {
	return provisionWorkspaceParent(ctx, directory, (*os.File).Sync)
}
func provisionWorkspaceParent(ctx context.Context, directory string, syncFile func(*os.File) error) (pierstore.DirectoryIdentity, error) {
	bad := func() (pierstore.DirectoryIdentity, error) {
		return pierstore.DirectoryIdentity{}, errors.Join(errWorkspaceTree, ctx.Err())
	}
	if ctx.Err() != nil || syncFile == nil || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return bad()
	}
	base := filepath.Dir(directory)
	root, err := openWorkspaceParentForInspection(base)
	if err != nil {
		return bad()
	}
	defer root.Close()
	rootID, err := workspaceDirectoryID(root)
	if err != nil {
		return bad()
	}
	current := func() error {
		f, e := workspaceParent(base, rootID)
		if e != nil {
			return e
		}
		return f.Close()
	}
	if current() != nil || ctx.Err() != nil {
		return bad()
	}
	// Mkdirat never follows/replaces an existing leaf. EEXIST permits only the
	// subsequently verified empty directory, not adoption of unknown project data.
	if err = unix.Mkdirat(int(root.Fd()), "private-projects", 0700); err != nil && !errors.Is(err, unix.EEXIST) {
		return bad()
	}
	child, err := workspaceOpen(root, "private-projects")
	if err != nil {
		return bad()
	}
	defer child.Close()
	id, err := workspaceDirectoryID(child)
	if err != nil || workspaceContents(child, nil) != nil || id.Device != rootID.Device {
		return bad()
	}
	if ctx.Err() != nil || current() != nil {
		return bad()
	}
	if syncFile(child) != nil || syncFile(root) != nil {
		return bad()
	}
	if ctx.Err() != nil || current() != nil || workspaceContents(child, nil) != nil {
		return bad()
	}
	named, err := workspaceParent(directory, id)
	if err != nil {
		return bad()
	}
	if named.Close() != nil || ctx.Err() != nil {
		return bad()
	}
	return id, nil
}
