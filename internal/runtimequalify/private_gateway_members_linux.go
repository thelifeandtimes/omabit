//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
)

// Selections must come from retained lifecycle records, never discovery by
// label/name. This read-only preflight does not adopt containers, inspect pier
// payload, establish historical kernel mount identity or authorize a launch.
// Running/exited selections require a previously recorded process start.
type privateGatewayMember struct {
	Contract           privateMemberContract
	ContainerID, State string
	StartedAt          time.Time
}

func (c privateMemberContract) validObservedMounts(actual []mount) bool {
	// Reuse the contract's exact configured/live mount validator. The configured
	// half is checked against real HostConfig by inspectGatewayMembers as well.
	var configured []json.RawMessage
	for _, m := range c.mounts() {
		raw, err := json.Marshal(map[string]any{"Type": m.Type, "Source": m.Source, "Target": m.Destination, "ReadOnly": !m.RW, "BindOptions": map[string]string{"Propagation": m.Propagation}})
		if err != nil {
			return false
		}
		configured = append(configured, raw)
	}
	return c.validMounts(configured, actual)
}

func (r *privateGatewayResources) bindGatewayMembers(selected []privateGatewayMember) ([]privateGatewayMember, error) {
	if r == nil || r.closed || r.q == nil || !privateGatewayID(r.containerID) || !privateGatewayID(r.networkID) || len(selected) > len(r.plan.Members) {
		return nil, errPrivateGatewayResource
	}
	ids, instances, runtimes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var result []privateGatewayMember
	for _, m := range selected {
		c := m.Contract
		s := c.selection
		if !privateGatewayID(m.ContainerID) || m.ContainerID == r.containerID || m.ContainerID == r.networkID || ids[m.ContainerID] || instances[s.InstanceID] || runtimes[s.RuntimeID] ||
			!reflect.DeepEqual(s.Plan, r.plan) || !reflect.DeepEqual(s.Workspace, r.workspace) || s.Gateway.Selection != r.contract.network.selection || s.Gateway.NetworkID != r.networkID || s.Gateway.ContainerID != r.containerID || c.imageID != r.contract.imageID || filepath.Join(c.root, s.Workspace.Path) != r.contract.root {
			return nil, errPrivateGatewayResource
		}
		copy, err := newPrivateMemberContract(s, c.root, c.imageID)
		if err != nil || !reflect.DeepEqual(copy, c) {
			return nil, errPrivateGatewayResource
		}
		switch m.State {
		case "created":
			if !m.StartedAt.IsZero() {
				return nil, errPrivateGatewayResource
			}
		case "running", "exited":
			if m.StartedAt.IsZero() {
				return nil, errPrivateGatewayResource
			}
		default:
			return nil, errPrivateGatewayResource
		}
		ids[m.ContainerID], instances[s.InstanceID], runtimes[s.RuntimeID] = true, true, true
		m.Contract = copy
		result = append(result, m)
	}
	slices.SortFunc(result, func(a, b privateGatewayMember) int { return strings.Compare(a.ContainerID, b.ContainerID) })
	return result, nil
}

func (r *privateGatewayResources) inspectGatewayMembers(ctx context.Context, members []privateGatewayMember) ([]container, error) {
	var result []container
	for _, m := range members {
		raw, err := r.q.docker(ctx, r.q.d.times.command, "container", "inspect", m.ContainerID)
		if err != nil || len(raw) > 1<<20 || !privateGatewayUnambiguousJSON([]byte(raw)) {
			return nil, errPrivateGatewayResource
		}
		var rows []container
		if json.Unmarshal([]byte(raw), &rows) != nil || len(rows) != 1 {
			return nil, errPrivateGatewayResource
		}
		v := rows[0]
		if m.State == "exited" {
			err = m.Contract.validateExitedContainer(v, m.ContainerID, m.StartedAt)
		} else {
			err = m.Contract.validateContainer(v, m.ContainerID, m.State == "running")
			if err == nil && m.State == "running" {
				at, e := time.Parse(time.RFC3339Nano, v.State.StartedAt)
				if e != nil || !at.Equal(m.StartedAt) {
					err = errPrivateGatewayResource
				}
			}
		}
		if err != nil {
			return nil, errPrivateGatewayResource
		}
		result = append(result, v)
	}
	return result, nil
}

const privateMemberInventoryFormat = `{"id":{{json .Id}},"name":{{json .Name}},"network_mode":{{json .HostConfig.NetworkMode}},"mounts":{{json .Mounts}}}`

type privateSharingRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mode string `json:"network_mode"`
}

// Collect mounts and namespace-sharing references from the same narrow Docker
// projection, with the complete container ID list checked before and after it.
// No foreign environment, command, log or application state is read. Callers
// still compare complete snapshots around filesystem and process verification.
func (r *privateGatewayResources) memberInventory(ctx context.Context) ([]projectContainerMounts, []privateSharingRow, error) {
	ids, err := r.containerIDs(ctx)
	if err != nil || len(ids) == 0 {
		return nil, nil, errPrivateGatewayResource
	}
	args := append([]string{"container", "inspect", "--format", privateMemberInventoryFormat}, ids...)
	raw, err := r.q.docker(ctx, r.q.d.times.command, args...)
	if err != nil {
		return nil, nil, errPrivateGatewayResource
	}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) != len(ids) {
		return nil, nil, errPrivateGatewayResource
	}
	names := map[string]bool{}
	var rows []privateSharingRow
	var mounts []projectContainerMounts
	for _, line := range lines {
		if !privateGatewayUnambiguousJSON([]byte(line)) {
			return nil, nil, errPrivateGatewayResource
		}
		var v struct {
			ID     string          `json:"id"`
			Name   string          `json:"name"`
			Mode   string          `json:"network_mode"`
			Mounts json.RawMessage `json:"mounts"`
		}
		d := json.NewDecoder(strings.NewReader(line))
		d.DisallowUnknownFields()
		var extra any
		if d.Decode(&v) != nil || d.Decode(&extra) != io.EOF || !strings.HasPrefix(v.Name, "/") || len(v.Name) < 2 || strings.ContainsAny(v.Name[1:], "/\x00\r\n") || names[v.Name] || v.Mode == "" || len(v.Mounts) == 0 {
			return nil, nil, errPrivateGatewayResource
		}
		m := projectContainerMounts{ID: v.ID}
		if json.Unmarshal(v.Mounts, &m.Mounts) != nil || len(m.Mounts) > 128 {
			return nil, nil, errPrivateGatewayResource
		}
		names[v.Name] = true
		rows = append(rows, privateSharingRow{ID: v.ID, Name: v.Name, Mode: v.Mode})
		mounts = append(mounts, m)
	}
	slices.SortFunc(rows, func(a, b privateSharingRow) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(mounts, func(a, b projectContainerMounts) int { return strings.Compare(a.ID, b.ID) })
	for n, id := range ids {
		if rows[n].ID != id || mounts[n].ID != id {
			return nil, nil, errPrivateGatewayResource
		}
	}
	after, err := r.containerIDs(ctx)
	if err != nil || !slices.Equal(ids, after) || ctx.Err() != nil {
		return nil, nil, errPrivateGatewayResource
	}
	return mounts, rows, nil
}

// Resolve namespace-sharing references by exact name/ID or unambiguous ID
// prefix. Indirect sharing via a member also refuses. Unresolved/cyclic names
// remain uncertain; unrelated, completely resolved projects may coexist.
func privateSharingOwned(rows []privateSharingRow, gateway string, members map[string]privateMemberContract) bool {
	byID := map[string]privateSharingRow{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	if _, ok := byID[gateway]; !ok {
		return false
	}
	for id := range members {
		if _, ok := byID[id]; !ok {
			return false
		}
	}
	resolve := func(ref string) string {
		if ref == "" {
			return ""
		}
		found := ""
		for _, row := range rows {
			if row.ID == ref || row.Name == ref || strings.TrimPrefix(row.Name, "/") == ref || strings.HasPrefix(row.ID, ref) {
				if found != "" && found != row.ID {
					return ""
				}
				found = row.ID
			}
		}
		return found
	}
	for _, row := range rows {
		if row.ID == gateway {
			continue
		}
		if _, ok := members[row.ID]; ok {
			if row.Mode != "container:"+gateway {
				return false
			}
			continue
		}
		seen := map[string]bool{}
		current := row
		for strings.HasPrefix(current.Mode, "container:") {
			if seen[current.ID] {
				return false
			}
			seen[current.ID] = true
			id := resolve(strings.TrimPrefix(current.Mode, "container:"))
			if id == "" || id == gateway {
				return false
			}
			if _, ok := members[id]; ok {
				return false
			}
			current = byID[id]
		}
	}
	return true
}

// Under the caller's lifecycle fence, verify exact member contracts and recorded
// process state around complete mount/sharing inventories. No effect/retry or
// readiness authority is provided. Fresh kernel namespace observation, held pier
// identities and durable ownership remain separate observer/driver requirements.
func (r *privateGatewayResources) checkMemberMounts(ctx context.Context, selected []privateGatewayMember) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	members, err := r.bindGatewayMembers(selected)
	if err != nil {
		return err
	}
	if r.engine(ctx) != nil {
		return errPrivateGatewayResource
	}
	before, err := r.inspectGatewayMembers(ctx, members)
	if err != nil {
		return err
	}
	contracts := map[string]privateMemberContract{}
	for _, m := range members {
		contracts[m.ContainerID] = m.Contract
	}
	// Preserve the two complete mount/sharing snapshots, using one projection
	// for each. The mount verifier brackets path-object checks; these same reads
	// bracket sharing policy too, instead of four redundant inventory walks.
	var sharing []privateSharingRow
	calls := 0
	inventory := func(ctx context.Context) ([]projectContainerMounts, error) {
		mounts, current, err := r.memberInventory(ctx)
		if err != nil || !privateSharingOwned(current, r.containerID, contracts) {
			return nil, errPrivateGatewayResource
		}
		for _, row := range current {
			if row.ID == r.containerID && (row.Name != "/"+r.contract.network.gatewayName() || row.Mode != r.networkID) {
				return nil, errPrivateGatewayResource
			}
			if c, ok := contracts[row.ID]; ok && row.Name != "/"+c.name {
				return nil, errPrivateGatewayResource
			}
		}
		if calls == 0 {
			sharing = current
		} else if calls != 1 || !reflect.DeepEqual(sharing, current) {
			return nil, errPrivateGatewayResource
		}
		calls++
		return mounts, nil
	}
	if err = r.checkMountsWithInventory(ctx, true, contracts, inventory); err != nil || calls != 2 {
		return errPrivateGatewayResource
	}
	after, err := r.inspectGatewayMembers(ctx, members)
	if err != nil || !reflect.DeepEqual(before, after) {
		return errPrivateGatewayResource
	}
	if r.engine(ctx) != nil {
		return errPrivateGatewayResource
	}
	return ctx.Err()
}
