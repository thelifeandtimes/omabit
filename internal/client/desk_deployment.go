package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (c *Client) DeskDeploymentReview(ctx context.Context, expected Identity, q protocol.DeskDeploymentRequest, sourceName, sourceDigest string) (protocol.DeskDeploymentReview, error) {
	var out protocol.DeskDeploymentReview
	if q.Validate() != nil || !deskwork.ValidSourceName(sourceName) || !protocol.ValidDeskOperationID(sourceDigest) {
		return out, errors.New("invalid deployment review selection")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskCommitFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskDeploymentReviewPath, q, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskDeploymentReview{}, err
	}
	if out.Validate() != nil || out.Request != q || out.SourceName != sourceName || out.SourceDigest != sourceDigest || time.Since(out.ReadAt) > protocol.DeskDeploymentTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskDeploymentReview{}, errors.New("invalid or changed deployment review response")
	}
	return out, nil
}
func (c *Client) CommitDesk(ctx context.Context, expected Identity, intent deskwork.CommitIntent) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := intent.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskCommitFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskCommitPath, intent, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent != intent.Intent {
		return protocol.DeskOperation{}, errors.New("commit receipt changed the reviewed intent; query the original operation ID")
	}
	return out, nil
}
