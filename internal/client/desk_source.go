package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) DeskSource(ctx context.Context, expected Identity, q protocol.DeskSourceRequest, sourceName string) (protocol.DeskSourceStatus, error) {
	var out protocol.DeskSourceStatus
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskSourceFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskSourcePath, q, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskSourceStatus{}, err
	}
	if out.Validate() != nil || out.Request != q || out.SourceName != sourceName || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskSourceStatus{}, errors.New("source response did not match the selected workflow and runtime")
	}
	return out, nil
}
