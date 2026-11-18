package client

import (
	"context"
	"errors"
	"net/http"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) GrubberyAppOperation(ctx context.Context, expected Identity, id string) (protocol.GrubberyAppOperation, error) {
	var out protocol.GrubberyAppOperation
	if !protocol.ValidDeskOperationID(id) {
		return out, errors.New("invalid app operation ID")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.GrubberyAppReceiptsFeature); err != nil {
		return out, err
	}
	if err := c.do(ctx, http.MethodGet, protocol.GrubberyAppOperationsPath+"/"+id, nil, &out, true); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if out.Validate() != nil || out.Record.Plan.RequestID != id {
		return protocol.GrubberyAppOperation{}, errors.New("app operation response changed identity or history")
	}
	return out, nil
}
