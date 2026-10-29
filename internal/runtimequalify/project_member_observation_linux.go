//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type projectObservedMember struct {
	contract projectMemberContract
	id       string
	// Nonzero binds an already observed process. Exited observations additionally
	// require explicit lifecycle permission; initial launch never sets it.
	processStartedAt time.Time
	allowExited      bool
}

type projectMemberSnapshot struct {
	Observation projectnetwork.Observation
	Containers  map[string]container
}

// inspectProjectMembers observes an already journaled subset during initial
// Create/Start. It never discovers/adopts member ownership by name or labels.
// Complete refers to namespace/container inventory, not ship readiness, pier
// identity, or absence of foreign mount users. Those are separate launch gates.
// Initially created and running containers are supported; Stop/recovery requires
// its own recorded lifecycle evidence before exited containers may be accepted.
func (e *projectAnchorEngine) inspectProjectMembers(ctx context.Context, known []projectObservedMember) (projectnetwork.Observation, error) {
	snapshot, err := e.inspectProjectMemberSnapshot(ctx, known)
	return snapshot.Observation, err
}

// inspectProjectMemberSnapshot returns the same bracketed observation together
// with its exact inspected process metadata. No additional read or cached result
// can silently supply Stop authority. Exited members need a recorded start time.
func (e *projectAnchorEngine) inspectProjectMemberSnapshot(ctx context.Context, known []projectObservedMember) (projectMemberSnapshot, error) {
	return e.inspectProjectMemberSnapshotMode(ctx, known, false)
}

// allowStoppedAnchor is for recovery review and its separately bound drain.
// Ordinary lifecycle callers keep the strict running-anchor inspection above.
func (e *projectAnchorEngine) inspectProjectMemberSnapshotMode(ctx context.Context, known []projectObservedMember, allowStoppedAnchor bool) (projectMemberSnapshot, error) {
	e.readyAt = time.Time{}
	p := e.prepared
	refuse := func(reason string) (projectMemberSnapshot, error) {
		return projectMemberSnapshot{}, fmt.Errorf("project member observation cannot establish complete unchanged ownership: %s", reason)
	}
	ids := map[string]bool{p.Anchor.ID: true, p.Bridge.ID: true}
	instances := map[string]bool{}
	runtimes := map[string]bool{}
	for _, member := range known {
		c := member.contract
		s := c.selection
		if member.allowExited && member.processStartedAt.IsZero() || !containerIDPattern.MatchString(member.id) || ids[member.id] || instances[s.InstanceID] || runtimes[s.RuntimeID] || s.Anchor.Validate(e.base.contract.Selection.Plan) != nil || !reflect.DeepEqual(s.Anchor.Prepared, p) || !reflect.DeepEqual(s.Project, e.base.contract.Selection) || c.imageID != e.base.contract.ImageID || c.home != e.base.q.d.home {
			return refuse("recorded member selection is invalid")
		}
		ids[member.id] = true
		instances[s.InstanceID] = true
		runtimes[s.RuntimeID] = true
	}
	if err := e.VerifyEngine(ctx, p.EngineID); err != nil {
		return projectMemberSnapshot{}, err
	}
	at := time.Now() // Retain the monotonic clock for the freshness bound.
	inventory, err := e.inventory(ctx)
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	joined, err := projectnetwork.NamespaceMembers(p.Anchor.ID, inventory)
	if err != nil || len(joined) != len(known) {
		return refuse("namespace membership does not match the recorded members")
	}
	for _, id := range joined {
		if !ids[id] {
			return refuse("namespace contains an unrecorded member")
		}
	}
	byID := map[string]projectnetwork.NetworkContainer{}
	for _, item := range inventory {
		byID[item.ID] = item
	}
	n, err := e.bridge(ctx)
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	selectedIDs := []string{p.Anchor.ID}
	for _, member := range known {
		selectedIDs = append(selectedIDs, member.id)
	}
	first, err := e.inspectProjectContainers(ctx, selectedIDs)
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	v := first[p.Anchor.ID]
	runningAnchor := *v.State.Running
	validateAnchor := func(v container) error {
		if allowStoppedAnchor {
			return e.base.contract.validateRecoveryAnchor(v, p.Anchor.ID, p.Bridge.ID)
		}
		return e.base.contract.validateAnchor(v, p.Anchor.ID, p.Bridge.ID, *v.State.Running)
	}
	if err = validateAnchor(v); err != nil {
		return projectMemberSnapshot{}, err
	}
	if !runningAnchor && !allowStoppedAnchor || runningAnchor && len(n.Containers) != 1 || !runningAnchor && len(n.Containers) != 0 || byID[p.Anchor.ID].Name != v.Name || byID[p.Anchor.ID].NetworkMode != v.HostConfig.NetworkMode {
		return refuse("anchor state, bridge attachments or inventory disagree")
	}
	var namespace projectnetwork.NamespaceIdentity
	if runningAnchor {
		namespace, err = e.namespace(ctx)
		if err != nil {
			return projectMemberSnapshot{}, err
		}
	}
	readMember := func(member projectObservedMember, actual container) (container, projectnetwork.MemberObservation, error) {
		running := *actual.State.Running
		if !running && member.allowExited {
			err = member.contract.validateExitedContainer(actual, member.id, member.processStartedAt)
		} else {
			err = member.contract.validateContainer(actual, member.id, running)
			if err == nil && !member.processStartedAt.IsZero() {
				started, parseErr := time.Parse(time.RFC3339Nano, actual.State.StartedAt)
				if parseErr != nil || !running || !started.Equal(member.processStartedAt) {
					err = errUntrustedContainer
				}
			}
		}
		if err != nil {
			return actual, projectnetwork.MemberObservation{}, err
		}
		item, ok := byID[member.id]
		if !ok || item.Name != actual.Name || item.NetworkMode != actual.HostConfig.NetworkMode {
			return actual, projectnetwork.MemberObservation{}, errUntrustedContainer
		}
		observed := projectnetwork.MemberObservation{InstanceID: member.contract.selection.InstanceID, ResourceObservation: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: member.id, ContractDigest: member.contract.digest}, Present: true}, Running: running, AnchorID: p.Anchor.ID}
		if running {
			observed.Namespace, err = e.namespaceFor(ctx, member.id)
		}
		return actual, observed, err
	}
	containers := make([]container, len(known))
	members := make([]projectnetwork.MemberObservation, len(known))
	for i, m := range known {
		containers[i], members[i], err = readMember(m, first[m.id])
		if err != nil {
			return projectMemberSnapshot{}, err
		}
	}
	// Batch the second complete known-container read as well. Full individual
	// contracts and both namespace probes remain required; unknown inventory
	// entries still come from the independently bracketed all-container scan.
	second := first
	if len(known) > 0 {
		second, err = e.inspectProjectContainers(ctx, selectedIDs)
		if err != nil {
			return projectMemberSnapshot{}, err
		}
		if !reflect.DeepEqual(v, second[p.Anchor.ID]) {
			return refuse("anchor changed during member inspection")
		}
	}
	for i, m := range known {
		again, observed, err := readMember(m, second[m.id])
		if err != nil {
			return projectMemberSnapshot{}, err
		}
		if !reflect.DeepEqual(containers[i], again) || !reflect.DeepEqual(members[i], observed) {
			return refuse("member process or namespace changed during inspection")
		}
	}
	afterInventory, err := e.inventory(ctx)
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	afterBridge, err := e.bridge(ctx)
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	afterAnchors, err := e.inspectProjectContainers(ctx, []string{p.Anchor.ID})
	if err != nil {
		return projectMemberSnapshot{}, err
	}
	afterAnchor := afterAnchors[p.Anchor.ID]
	if err = validateAnchor(afterAnchor); err != nil {
		return projectMemberSnapshot{}, err
	}
	var afterNamespace projectnetwork.NamespaceIdentity
	if runningAnchor {
		afterNamespace, err = e.namespace(ctx)
		if err != nil {
			return projectMemberSnapshot{}, err
		}
	}
	if !reflect.DeepEqual(inventory, afterInventory) || !reflect.DeepEqual(n, afterBridge) || !reflect.DeepEqual(v, afterAnchor) || namespace != afterNamespace {
		return refuse("inventory, bridge, anchor or namespace changed during inspection")
	}
	if err = e.VerifyEngine(ctx, p.EngineID); err != nil {
		return projectMemberSnapshot{}, err
	}
	if err = ctx.Err(); err != nil {
		return projectMemberSnapshot{}, err
	}
	if err := projectMemberObservationFreshness(time.Since(at)); err != nil {
		return projectMemberSnapshot{}, err
	}
	slices.SortFunc(members, func(a, b projectnetwork.MemberObservation) int {
		if a.InstanceID < b.InstanceID {
			return -1
		}
		if a.InstanceID > b.InstanceID {
			return 1
		}
		return 0
	})
	o := projectnetwork.Observation{ProjectID: p.ProjectID, Revision: p.ProjectRevision, Generation: p.Generation, EngineID: p.EngineID, ObservedAt: at.UTC(), Complete: true, Bridge: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Bridge.ID, ContractDigest: e.base.contract.digest("bridge")}, Present: true}, BridgeAttachments: []string{}, Anchor: projectnetwork.AnchorObservation{ResourceObservation: projectnetwork.ResourceObservation{Resource: projectnetwork.ResourceIdentity{ID: p.Anchor.ID, ContractDigest: e.base.contract.digest("anchor")}, Present: true}, Running: runningAnchor, Namespace: namespace, BridgeID: p.Bridge.ID}, Members: members}
	for _, m := range e.base.contract.Selection.Plan.Members {
		o.Anchor.Requested = append(o.Anchor.Requested, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: e.base.contract.Selection.Plan.BindAddress, HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	if runningAnchor {
		o.Anchor.Active = slices.Clone(o.Anchor.Requested)
		o.BridgeAttachments = []string{p.Anchor.ID}
	}
	return projectMemberSnapshot{Observation: o, Containers: second}, nil
}

// Inspect exact full IDs in one bounded request. Never discover/adopt by name,
// accept duplicate/missing/extra rows, or let response ordering choose ownership.
func (e *projectAnchorEngine) inspectProjectContainers(ctx context.Context, selected []string) (map[string]container, error) {
	ids := slices.Clone(selected)
	slices.Sort(ids)
	if len(ids) == 0 || len(ids) > projectInventoryLimit {
		return nil, errUntrustedContainer
	}
	for i, id := range ids {
		if !containerIDPattern.MatchString(id) || i > 0 && ids[i-1] == id {
			return nil, errUntrustedContainer
		}
	}
	q := e.base.q
	out, err := q.docker(ctx, q.d.times.command, append([]string{"container", "inspect"}, ids...)...)
	if err != nil {
		return nil, err
	}
	var values []container
	if json.Unmarshal([]byte(out), &values) != nil || len(values) != len(ids) {
		return nil, errUntrustedContainer
	}
	result := map[string]container{}
	for _, v := range values {
		if !slices.Contains(ids, v.ID) || result[v.ID].ID != "" || v.State == nil || v.State.Running == nil {
			return nil, errUntrustedContainer
		}
		result[v.ID] = v
	}
	return result, nil
}

// Keep slow but otherwise matching observations distinct from changed ownership.
// This reports the existing bound; it does not extend it or authorize a retry.
func projectMemberObservationFreshness(elapsed time.Duration) error {
	if elapsed > 30*time.Second {
		return fmt.Errorf("project member observation exceeded 30s freshness limit (elapsed %s); ownership remains unconfirmed", elapsed.Round(time.Millisecond))
	}
	return nil
}
