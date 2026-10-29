package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrProjectInventoryUnsupported = errors.New("runner does not advertise project-reservations-v1; saved project inventory is unavailable")
var ErrProjectInventoryInvalid = errors.New("invalid stored project inventory")

// Projects reads only the selected runner's saved reservation claims. The caller
// uses a pinned transport for reconnects, just as for ordinary fleet observations.
func (c *Client) Projects(ctx context.Context, expected Identity) (protocol.ProjectInventory, error) {
	var result protocol.ProjectInventory
	h, err := c.Hello(ctx)
	if err != nil {
		return result, err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return result, err
	}
	if !slices.Contains(h.Features, protocol.ProjectInventoryFeature) {
		return result, ErrProjectInventoryUnsupported
	}
	if err = c.do(ctx, http.MethodGet, protocol.ProjectInventoryPath, nil, &result, true); err != nil {
		return protocol.ProjectInventory{}, err
	}
	if err = expected.Check(result.RunnerID, result.Mode); err != nil {
		return protocol.ProjectInventory{}, err
	}
	if !validProjectInventory(result) {
		return protocol.ProjectInventory{}, ErrProjectInventoryInvalid
	}
	return result, nil
}

func validProjectInventory(r protocol.ProjectInventory) bool {
	if r.SchemaVersion != 1 || r.ReadAt.IsZero() || r.Basis != "stored-reservations" || r.RuntimeObservation != "not-performed" || r.Projects == nil {
		return false
	}
	lastID := ""
	reserved := projectnetwork.Reservations{Instances: map[string]bool{}, HTTPPorts: map[uint16]bool{}}
	for _, p := range r.Projects {
		if !protocol.ValidID(p.ID) || p.ID <= lastID || p.Revision != 1 || p.CreatedAt.IsZero() || p.CreatedAt.After(r.ReadAt) {
			return false
		}
		request := projectnetwork.Request{Label: p.Label, ConfirmSharedAdministration: true}
		for _, m := range p.Members {
			request.Members = append(request.Members, projectnetwork.MemberRequest{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
		}
		plan, err := projectnetwork.Build(request, reserved)
		if err != nil {
			return false
		}
		digest, err := plan.Digest()
		if err != nil || digest != p.ReviewDigest {
			return false
		}
		for i, m := range p.Members {
			want := plan.Members[i]
			if m.InstanceID != want.InstanceID || m.Ship != want.Ship || m.HTTPPort != want.HTTPPort || m.InternalHTTPPort != want.InternalHTTPPort {
				return false
			}
			reserved.Instances[m.InstanceID], reserved.HTTPPorts[m.HTTPPort] = true, true
		}
		internal, ships := map[uint16]bool{}, map[string]bool{}
		for _, m := range p.Members {
			internal[m.InternalHTTPPort] = true
			ships[m.Ship] = true
		}
		for _, m := range p.AdditionalMembers {
			if !protocol.ValidID(m.InstanceID) || len(m.Ship) != 4 || !protocol.ValidFakeShip(m.Ship) || ships[m.Ship] || m.HTTPPort < 1024 || m.InternalHTTPPort < 8080 || m.InternalHTTPPort > 8335 || internal[m.InternalHTTPPort] || reserved.Instances[m.InstanceID] || reserved.HTTPPorts[m.HTTPPort] {
				return false
			}
			internal[m.InternalHTTPPort] = true
			ships[m.Ship] = true
			reserved.Instances[m.InstanceID] = true
			reserved.HTTPPorts[m.HTTPPort] = true
		}
		lastID = p.ID
	}
	return true
}
