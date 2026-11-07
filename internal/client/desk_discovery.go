package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) DiscoverDesks(ctx context.Context, expected Identity, selection protocol.DeskDiscoverySelection) (protocol.DeskDiscovery, error) {
	var out protocol.DeskDiscovery
	if err := selection.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskDiscoveryFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskDiscoveryPath, selection, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskDiscovery{}, err
	}
	names, err := deskwork.NormalizeDeskNames(out.Desks)
	if err != nil || out.Version != 1 || out.Selection != selection || out.ReadAt.IsZero() || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskDiscovery{}, errors.New("invalid desk discovery response")
	}
	out.Desks = names
	return out, nil
}
