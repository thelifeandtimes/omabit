package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) DeskApplication(ctx context.Context, expected Identity, selection protocol.DeskSelection) (protocol.DeskApplication, error) {
	var out protocol.DeskApplication
	if err := selection.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskApplicationFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskApplicationPath, selection, &out, true); err != nil {
		return protocol.DeskApplication{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskApplication{}, err
	}
	if out.Validate() != nil || out.Selection != selection || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskApplication{}, errors.New("invalid or changed application observation")
	}
	return out, nil
}
