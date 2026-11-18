//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ObserveExpandedProjectStop performs only full, bracketed ownership/process
// reads. It has no signal, Start, container removal or database mutation path.
func ObserveExpandedProjectStop(ctx context.Context, s ProjectExpansion, id string) (protocol.ProjectStopObservation, error) {
	return observeExpandedProjectStop(ctx, s, id, defaults())
}

func observeExpandedProjectStop(ctx context.Context, s ProjectExpansion, id string, d dependencies) (protocol.ProjectStopObservation, error) {
	var selected *ProjectExpandedMember
	for i := range s.Members {
		if s.Members[i].Member.InstanceID == id {
			selected = &s.Members[i]
		}
	}
	if selected == nil || selected.Phase != "stop-dispatched" || selected.Initialized.Identity.Inode == 0 {
		return protocol.ProjectStopObservation{}, errors.New("interrupted stop lacks retained member or pier evidence")
	}
	e, snap, err := expansionSnapshotWithDependencies(ctx, s, d)
	if err != nil {
		return protocol.ProjectStopObservation{}, err
	}
	v, ok := snap.Containers[selected.Creator.Container.ID]
	if !ok || v.State == nil {
		return protocol.ProjectStopObservation{}, errUntrustedContainer
	}
	if _, err = pierstore.Check(e.source.anchor.base.q.d.home, selected.Initialized); err != nil {
		return protocol.ProjectStopObservation{}, err
	}
	started, err := time.Parse(time.RFC3339Nano, v.State.StartedAt)
	if err != nil || started.IsZero() {
		return protocol.ProjectStopObservation{}, errUntrustedContainer
	}
	finished, err := time.Parse(time.RFC3339Nano, v.State.FinishedAt)
	if err != nil {
		return protocol.ProjectStopObservation{}, errUntrustedContainer
	}
	ns := snap.Observation.Anchor.Namespace
	return protocol.ProjectStopObservation{ContainerID: v.ID, ContractDigest: selected.Creator.Container.ContractDigest, EngineID: snap.Observation.EngineID, AnchorID: s.Anchor.Prepared.Anchor.ID, BridgeID: s.Anchor.Prepared.Bridge.ID, BootID: ns.BootID, NamespaceDevice: ns.Device, NamespaceInode: ns.Inode, Storage: selected.Initialized, StartedAt: started, FinishedAt: finished, ExitCode: *v.State.ExitCode, OOMKilled: *v.State.OOMKilled, Running: *v.State.Running, ObservedAt: snap.Observation.ObservedAt}, nil
}
