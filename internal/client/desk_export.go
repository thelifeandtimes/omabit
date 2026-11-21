package client

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) ExportDeskSource(ctx context.Context, expected Identity, request protocol.DeskSourceExportRequest) (protocol.DeskSourceExport, error) {
	var out protocol.DeskSourceExport
	if err := request.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskSourceExportFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, "POST", protocol.DeskSourceExportPath, request, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskSourceExport{}, err
	}
	if out.Validate() != nil || !reflect.DeepEqual(out.Request, request) || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskSourceExport{}, errors.New("source export response binding or bytes changed")
	}
	return out, nil
}
