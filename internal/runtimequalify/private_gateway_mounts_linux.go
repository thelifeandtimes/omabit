//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"slices"
	"strings"
)

func (e *privateGatewayResources) containerIDs(ctx context.Context) ([]string, error) {
	out, err := e.q.docker(ctx, e.q.d.times.command, "container", "ls", "--all", "--quiet", "--no-trunc")
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(out)
	if len(ids) > projectInventoryLimit {
		return nil, errUntrustedContainer
	}
	slices.Sort(ids)
	for n, id := range ids {
		if !privateGatewayID(id) || n > 0 && id == ids[n-1] {
			return nil, errUntrustedContainer
		}
	}
	return ids, nil
}

// Read only IDs/mounts, including never-started and stopped containers. Empty
// engines are valid for initial gateway creation. No environment/log is read.
func (e *privateGatewayResources) mountInventory(ctx context.Context) ([]projectContainerMounts, error) {
	ids, err := e.containerIDs(ctx)
	if err != nil {
		return nil, err
	}
	var result []projectContainerMounts
	if len(ids) > 0 {
		args := append([]string{"container", "inspect", "--format", projectMountFormat}, ids...)
		out, err := e.q.docker(ctx, e.q.d.times.command, args...)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != len(ids) {
			return nil, errUntrustedContainer
		}
		for _, line := range lines {
			if !privateGatewayUnambiguousJSON([]byte(line)) {
				return nil, errUntrustedContainer
			}
			var row struct {
				ID     string          `json:"id"`
				Mounts json.RawMessage `json:"mounts"`
			}
			decoder := json.NewDecoder(bytes.NewBufferString(line))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&row) != nil || len(row.Mounts) == 0 {
				return nil, errUntrustedContainer
			}
			var extra any
			if decoder.Decode(&extra) != io.EOF {
				return nil, errUntrustedContainer
			}
			v := projectContainerMounts{ID: row.ID}
			if json.Unmarshal(row.Mounts, &v.Mounts) != nil || len(v.Mounts) > 128 {
				return nil, errUntrustedContainer
			}
			result = append(result, v)
		}
		slices.SortFunc(result, func(a, b projectContainerMounts) int { return strings.Compare(a.ID, b.ID) })
		for n, id := range ids {
			if result[n].ID != id {
				return nil, errUntrustedContainer
			}
		}
	}
	after, err := e.containerIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(ids, after) {
		return nil, errUntrustedContainer
	}
	return result, nil
}

// Negative preflight only. Current configured mount paths cannot prove the
// historical kernel identity of a running container's renamed mount source.
// The exact never-started gateway is the sole writable workspace exception.
// Other containers may share immutable inputs read-only; writable overlap and
// unknown volume/plugin sources are unresolved and refuse.
func (e *privateGatewayResources) checkMounts(ctx context.Context) error {
	return e.checkMountsMode(ctx, false)
}

// Running inspection is observation-only. It still excludes every other
// workspace writer; configured paths do not prove historical mount identities.
func (e *privateGatewayResources) checkMountsMode(ctx context.Context, running bool) error {
	return e.checkMountsWithMembers(ctx, running, nil)
}

// Nonempty exceptions are private, validated durable selections supplied by
// checkMemberMounts. Initial gateway creation/waiting never supplies them.
func (e *privateGatewayResources) checkMountsWithMembers(ctx context.Context, running bool, members map[string]privateMemberContract) error {
	return e.checkMountsWithInventory(ctx, running, members, e.mountInventory)
}

// The inventory provider must return a complete, ID-bracketed snapshot. Member
// observations use a combined mount/sharing projection with the same checks.
func (e *privateGatewayResources) checkMountsWithInventory(ctx context.Context, running bool, members map[string]privateMemberContract, inventory func(context.Context) ([]projectContainerMounts, error)) error {
	inspect := func() error {
		if running {
			_, err := e.inspectRunning(ctx)
			return err
		}
		return e.inspectCreated(ctx)
	}
	if running && e.containerID == "" {
		return errPrivateGatewayResource
	}
	if e.containerID != "" {
		if err := inspect(); err != nil {
			return err
		}
	}
	before, err := inventory(ctx)
	if err != nil {
		return err
	}
	root, err := projectPathObjects(ctx, e.contract.root)
	if err != nil {
		return err
	}
	inputs, err := projectPathObjects(ctx, e.contract.inputs)
	if err != nil {
		return err
	}
	sources := map[string][]projectPathObject{}
	var memberObjects [][]projectPathObject
	for _, c := range members {
		for _, m := range c.mounts() {
			objects, err := projectPathObjects(ctx, m.Source)
			if err != nil {
				return err
			}
			sources[m.Source] = objects
			memberObjects = append(memberObjects, objects)
		}
	}
	foundOwn := e.containerID == ""
	foundMembers := map[string]bool{}
	for _, row := range before {
		if row.ID == e.containerID {
			foundOwn = true
			want := e.contract.binds()
			seen := map[string]bool{}
			binds, tmpfs := 0, 0
			for _, m := range row.Mounts {
				if seen[m.Destination] {
					return errPrivateGatewayResource
				}
				seen[m.Destination] = true
				if slices.Contains(want, m) {
					binds++
					continue
				}
				if m.Type != "tmpfs" || m.Source != "" || m.Propagation != "" || (m.Destination != "/tmp" && m.Destination != "/urbit") || m.RW != (m.Destination == "/tmp") {
					return errPrivateGatewayResource
				}
				tmpfs++
			}
			if binds != len(want) || tmpfs != 0 && tmpfs != 2 {
				return errPrivateGatewayResource
			}
			continue
		}
		if c, ok := members[row.ID]; ok {
			if !running || !c.validObservedMounts(row.Mounts) {
				return errPrivateGatewayResource
			}
			foundMembers[row.ID] = true
			continue
		}
		for _, m := range row.Mounts {
			if m.Type == "tmpfs" && m.Source == "" {
				continue
			}
			if m.Type != "bind" {
				return errPrivateGatewayResource
			}
			objects, ok := sources[m.Source]
			if !ok {
				objects, err = projectPathObjects(ctx, m.Source)
				if err != nil {
					return err
				}
				sources[m.Source] = objects
			}
			if projectPathsOverlap(root, objects) || m.RW && projectPathsOverlap(inputs, objects) {
				return errPrivateGatewayResource
			}
			for _, owned := range memberObjects {
				if projectPathsOverlap(owned, objects) {
					return errPrivateGatewayResource
				}
			}
		}
	}
	if !foundOwn || len(foundMembers) != len(members) {
		return errPrivateGatewayResource
	}
	after, err := inventory(ctx)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return errPrivateGatewayResource
	}
	for path, objects := range sources {
		now, err := projectPathObjects(ctx, path)
		if err != nil {
			return err
		}
		if !sameProjectPathObjects(objects, now) {
			return errPrivateGatewayResource
		}
	}
	for _, objects := range [][]projectPathObject{root, inputs} {
		now, err := projectPathObjects(ctx, objects[0].path)
		if err != nil {
			return err
		}
		if !sameProjectPathObjects(objects, now) {
			return errPrivateGatewayResource
		}
	}
	if e.containerID != "" {
		if err := inspect(); err != nil {
			return err
		}
	}
	return ctx.Err()
}
