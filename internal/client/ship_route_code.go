package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) ShipRouteCode(ctx context.Context, expected Identity, request protocol.ShipRouteCodeRequest) (protocol.ShipRouteCode, error) {
	var out protocol.ShipRouteCode
	unavailable := errors.New("ship route credential unavailable; inspect the selected ship and route")
	if request.Validate() != nil || request.Binding.RunnerID != expected.RunnerID {
		return out, unavailable
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.ShipRouteCodeFeature); err != nil {
		return out, unavailable
	}
	hc := *c.httpClient
	hc.Timeout = protocol.ShipRoutesTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	// Suppress raw transport/JSON/runner errors: hostile responses may contain
	// the secret in any field, not just the documented code field.
	if scoped.do(ctx, http.MethodPost, protocol.ShipRouteCodePath, request, &out, true) != nil || out.Version != 1 || expected.Check(out.RunnerID, out.Mode) != nil || out.Request != request || !protocol.ValidLoginCode(out.Code) {
		return protocol.ShipRouteCode{}, unavailable
	}
	return out, nil
}
