package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (c *Client) shipRoutes(ctx context.Context, expected Identity, method string, input any, selection protocol.ShipRouteSelection) (protocol.ShipRoutes, error) {
	var out protocol.ShipRoutes
	if err := selection.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.ShipRoutesFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.ShipRoutesTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, method, protocol.ShipRoutesPath, input, &out, true); err != nil {
		return protocol.ShipRoutes{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.ShipRoutes{}, err
	}
	if out.Validate() != nil || out.Selection != selection || time.Since(out.ReadAt) > protocol.ShipRoutesTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.ShipRoutes{}, errors.New("invalid or changed ship route response")
	}
	return out, nil
}
func (c *Client) ShipRoutes(ctx context.Context, expected Identity, selection protocol.ShipRouteSelection) (protocol.ShipRoutes, error) {
	return c.shipRoutes(ctx, expected, http.MethodPost, selection, selection)
}
func (c *Client) PutShipRoutes(ctx context.Context, expected Identity, u protocol.ShipRoutesUpdate) (protocol.ShipRoutes, error) {
	if err := u.Validate(); err != nil {
		return protocol.ShipRoutes{}, err
	}
	if err := expected.Check(u.Update.Binding.RunnerID, expected.Mode); err != nil {
		return protocol.ShipRoutes{}, err
	}
	out, err := c.shipRoutes(ctx, expected, http.MethodPut, u, u.Selection)
	if err != nil {
		return out, err
	}
	if out.Record.Binding != u.Update.Binding || out.Record.Revision != u.Update.ExpectedRevision+1 || out.Record.Settings.Digest() != u.Update.Settings.Digest() {
		return protocol.ShipRoutes{}, errors.New("ship URL save receipt changed; inspect before editing again")
	}
	return out, nil
}
