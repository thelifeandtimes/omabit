//go:build linux

package privatechain

import (
	"context"
	"path/filepath"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// GatewayMemberPier identifies a previously accepted member pier. These inputs
// must come from retained lifecycle evidence, never an unverified path discovery.
// Omitted members must still have empty mounts. This is metadata selection, not
// authority to adopt a pier, run a member or accept a container as its owner.
type GatewayMemberPier struct {
	InstanceID string
	Identity   pierstore.Identity
}

// OpenMemberGatewayStateVerifier pins the ready gateway's original filesystem
// objects while treating explicitly retained member piers as opaque live data.
// Chain/key metadata still receives the complete bounded scan. Each selected
// mount must contain exactly its ship-named pier with the retained .urb object.
// No state payload is read, no lock is removed, and no process state is inferred.
// The caller separately proves complete container membership and namespace
// ownership under the runner lifecycle fence. This helper grants no dispatch.
func OpenMemberGatewayStateVerifier(ctx context.Context, directory string, p Plan, w Workspace, g Gateway, members []GatewayMemberPier) (*GatewayStateVerifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g.Validate(p, w) != nil || g.Phase != "ready" || len(members) > len(p.Members) || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "private-projects" {
		return nil, errWorkspaceTree
	}
	known := map[string]bool{}
	for _, m := range p.Members {
		known[m.InstanceID] = true
	}
	reserved := []pierstore.DirectoryIdentity{w.Parent, w.Root, w.Observation.PiersRoot, g.Observation.Chain, g.Observation.Secrets}
	reserved = append(reserved, w.Observation.Mounts...)
	seen := map[string]bool{}
	for _, m := range members {
		if !known[m.InstanceID] || seen[m.InstanceID] {
			return nil, errWorkspaceTree
		}
		seen[m.InstanceID] = true
		for _, id := range []pierstore.DirectoryIdentity{{Device: m.Identity.Device, Inode: m.Identity.Inode, Birth: m.Identity.Birth}, {Device: m.Identity.UrbDevice, Inode: m.Identity.UrbInode, Birth: m.Identity.UrbBirth}} {
			if !workspaceIdentity(id) || id.Device != w.Root.Device {
				return nil, errWorkspaceTree
			}
			for _, prior := range reserved {
				if id.SameObject(prior) {
					return nil, errWorkspaceTree
				}
			}
			reserved = append(reserved, id)
		}
	}
	expected := GatewayStateObservation{Root: g.Observation.Root, Chain: g.Observation.Chain, Secrets: g.Observation.Secrets, Mounts: append([]pierstore.DirectoryIdentity{}, g.Observation.Mounts...), Private: true}
	return openGatewayStateVerifier(ctx, directory, p, w, &expected, members)
}

func openGatewayStateVerifier(ctx context.Context, directory string, p Plan, w Workspace, expected *GatewayStateObservation, members []GatewayMemberPier) (*GatewayStateVerifier, error) {
	parent, err := workspaceParent(directory, w.Parent)
	if err != nil {
		return nil, err
	}
	p, w = gatewayInputs(p, w)
	d := &WorkspaceFilesystem{directory: directory, parent: parent, parentIdentity: w.Parent, plan: p, initial: w.Clone()}
	v := &GatewayStateVerifier{filesystem: d, workspace: w, initial: expected, memberPiers: map[string]*MemberPierVerifier{}}
	for _, m := range members {
		held, e := OpenMemberPierVerifier(ctx, directory, p, w, m.InstanceID, m.Identity)
		if e != nil {
			v.Close()
			return nil, e
		}
		v.memberPiers["piers/"+m.InstanceID] = held
	}
	o, err := v.Verify(ctx)
	if err != nil {
		v.Close()
		return nil, err
	}
	v.initial = &o
	return v, nil
}
