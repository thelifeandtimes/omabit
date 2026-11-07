package collection

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Registry) ShipRoutes(ctx context.Context, name, runner, instance string) (protocol.ShipRoutes, error) {
	var out protocol.ShipRoutes
	if !protocol.ValidID(instance) {
		return out, errors.New("select a registered ship instance")
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	policy, err := c.AppPolicy(ctx, entry.Identity())
	if err != nil {
		return out, err
	}
	selected, err := selectDeskRuntime(ctx, c, entry.Identity(), policy, instance, "base")
	if err != nil {
		return out, err
	}
	return c.ShipRoutes(ctx, entry.Identity(), protocol.ShipRouteSelection{Target: selected.Target.Ship(), RuntimeID: selected.RuntimeID, ConfigRevision: selected.ConfigRevision})
}

// Edits bind the reviewed runtime and revision. A lost response requires reading
// the shared registry again, never silently rebasing or retrying an old edit.
func (r *Registry) PutShipRoutes(ctx context.Context, name, runner string, u protocol.ShipRoutesUpdate) (protocol.ShipRoutes, error) {
	if err := u.Validate(); err != nil {
		return protocol.ShipRoutes{}, err
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return protocol.ShipRoutes{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	return c.PutShipRoutes(ctx, entry.Identity(), u)
}
