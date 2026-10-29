//go:build linux && amd64

package managed

import (
	"context"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateWorkspaceParentProvisioner interface {
	ProvisionPrivateWorkspaceParent(context.Context, string) (pierstore.DirectoryIdentity, error)
}

func (realBackend) ProvisionPrivateWorkspaceParent(ctx context.Context, directory string) (pierstore.DirectoryIdentity, error) {
	return privatechain.ProvisionWorkspaceParent(ctx, directory)
}

// Called under r.mu after replay lookup and fresh request review. Initial empty
// grouping metadata may be provisioned only with no retained workspace claims and
// while the host work gate is free. No startup/read-only API calls this helper.
func (r *Server) privateWorkspaceParent(b privateWorkspaceBackend) (pierstore.DirectoryIdentity, error) {
	claims, err := privateWorkspaceClaims(r.s.db)
	if err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	directory := filepath.Join(r.s.directory, "private-projects")
	if len(claims) == 0 {
		if provisioner, ok := b.(privateWorkspaceParentProvisioner); ok {
			select {
			case r.workGate <- struct{}{}:
				defer func() { <-r.workGate }()
			default:
				return pierstore.DirectoryIdentity{}, conflict("runner_busy", "wait for current host work before preparing private project storage")
			}
			if err = projectMaintenanceClear(r.s); err != nil {
				return pierstore.DirectoryIdentity{}, err
			}
			return provisioner.ProvisionPrivateWorkspaceParent(r.ctx, directory)
		}
	}
	parent, err := b.InspectPrivateWorkspaceParent(r.ctx, directory)
	if err != nil {
		if len(claims) != 0 {
			return pierstore.DirectoryIdentity{}, conflict("private_storage_recovery_required", "private storage is unavailable for retained workspaces; preserve and recover the original parent")
		}
		return pierstore.DirectoryIdentity{}, err
	}
	for _, w := range claims {
		if !parent.SameObject(w.Parent) {
			return pierstore.DirectoryIdentity{}, conflict("private_storage_recovery_required", "private storage differs from retained workspaces; preserve and recover the original parent")
		}
	}
	return parent, nil
}
