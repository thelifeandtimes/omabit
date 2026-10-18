package client

import (
	"context"
	"errors"
	"math"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) PublishAmes(ctx context.Context, id string, req protocol.PublishAmesRequest) (protocol.Operation, error) {
	var op protocol.Operation
	if validateID(id) != nil || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 || req.ExpectedRevision >= math.MaxInt64 || req.AmesPort < 1024 || !req.ConfirmNetwork {
		return op, errors.New("publish-ames requires a valid selection, revision, port, request ID and network confirmation")
	}
	h, err := c.Hello(ctx)
	if err != nil {
		return op, err
	}
	if h.Mode != "development-managed" || h.AccessRole != "operator" || !slices.Contains(h.Features, protocol.AmesPublicationFeature) {
		return op, errors.New("require operator access and publish-ames capability; update the selected runner before migrating Ames publication")
	}
	err = c.storageRequest(ctx, "/v1/instances/"+id+"/publish-ames", req, &op)
	if err == nil && (op.InstanceID != id || op.RequestID != req.RequestID || op.Kind != protocol.AmesPublicationFeature || op.Status != "succeeded" || op.ConfigRevision != req.ExpectedRevision+1 || op.Actor != h.Actor || !protocol.ValidID(op.ID) || op.FinishedAt == nil) {
		err = errors.New("Ames publication receipt does not match the selected instance, request, actor and revision")
	}
	return op, err
}
