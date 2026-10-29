//go:build linux && amd64

package runtimequalify

import (
	"context"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type privateGatewayMembership struct {
	members []privateGatewayMember
	piers   []privatechain.GatewayMemberPier
}

// This private constructor observes one retained ready gateway with explicit
// lifecycle-owned member containers and pier identities. It neither discovers
// ownership nor captures first-boot identities. Uncertain/partial members need
// separately attributed recovery; opening cannot adopt their current objects.
func openPrivateMemberGatewayObserver(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, members []privateGatewayMember, piers []privatechain.GatewayMemberPier) (*privateGatewayObserver, error) {
	return openPrivateGatewayObserverMode(ctx, q, stateRoot, inputRoot, p, w, g, &privateGatewayMembership{members: members, piers: piers})
}

func (r *privateGatewayResources) bindGatewayMemberPiers(selected []privateGatewayMember, piers []privatechain.GatewayMemberPier) ([]privateGatewayMember, error) {
	members, err := r.bindGatewayMembers(selected)
	if err != nil {
		return nil, err
	}
	if len(piers) > len(r.plan.Members) {
		return nil, errPrivateGatewayResource
	}
	known := map[string]bool{}
	for _, m := range r.plan.Members {
		known[m.InstanceID] = true
	}
	identities := map[string]pierstore.Identity{}
	for _, p := range piers {
		if !known[p.InstanceID] || !validProjectPier(p.Identity) {
			return nil, errPrivateGatewayResource
		}
		if _, ok := identities[p.InstanceID]; ok {
			return nil, errPrivateGatewayResource
		}
		identities[p.InstanceID] = p.Identity
	}
	for _, m := range members {
		s := m.Contract.selection
		id, has := identities[s.InstanceID]
		if m.State == "created" && s.Mode == "boot" {
			if has {
				return nil, errPrivateGatewayResource
			}
		} else if !has || (s.Mode == "resume" && id != s.PierIdentity) {
			return nil, errPrivateGatewayResource
		}
	}
	// Full object distinctness and filesystem binding are enforced when the held
	// member-aware state verifier opens. No caller receives unheld authority here.
	return members, nil
}

func (o *privateGatewayObserver) selectedGatewayPhase(g privatechain.Gateway) bool {
	if o.memberGateway == nil {
		return len(o.members) == 0 && (g.Phase == "start-dispatched" || g.Phase == "started")
	}
	return g.Phase == "ready" && reflect.DeepEqual(g, *o.memberGateway)
}

// Process snapshots span both the chain RPC and held filesystem observations.
// Each running member must freshly report the gateway's actual kernel namespace;
// created/exited containers are never executed. This performs no ship request.
func (o *privateGatewayObserver) observeMembers(ctx context.Context, ns projectnetwork.NamespaceIdentity) ([]container, error) {
	r := o.runtime
	if o.memberGateway == nil {
		return nil, r.checkMountsMode(ctx, true)
	}
	if ns != o.memberGateway.Observation.Namespace {
		return nil, errPrivateGatewayResource
	}
	if err := r.checkMemberMounts(ctx, o.members); err != nil {
		return nil, err
	}
	before, err := r.inspectGatewayMembers(ctx, o.members)
	if err != nil {
		return nil, err
	}
	e := &projectAnchorEngine{base: &projectDockerEngine{q: r.q}, readBootID: o.readBootID}
	for _, m := range o.members {
		if m.State != "running" {
			continue
		}
		got, err := e.namespaceFor(ctx, m.ContainerID)
		if err != nil || got != ns {
			return nil, errPrivateGatewayResource
		}
	}
	after, err := r.inspectGatewayMembers(ctx, o.members)
	if err != nil || !reflect.DeepEqual(before, after) {
		return nil, errPrivateGatewayResource
	}
	return after, ctx.Err()
}
