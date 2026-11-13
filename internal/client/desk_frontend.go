package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) DeskFrontendReview(ctx context.Context, expected Identity, q protocol.DeskFrontendRequest) (protocol.DeskFrontendReview, error) {
	var out protocol.DeskFrontendReview
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskFrontendFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskFrontendReviewPath, q, &out, true); err != nil {
		return protocol.DeskFrontendReview{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskFrontendReview{}, err
	}
	if out.Validate() != nil || out.Request != q || time.Since(out.ReadAt) > protocol.DeskDeploymentTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskFrontendReview{}, errors.New("invalid or changed frontend review")
	}
	return out, nil
}
func (c *Client) UploadDeskFrontend(ctx context.Context, expected Identity, input protocol.DeskFrontendSubmission) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := input.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskFrontendFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskFrontendsPath, input, &out, true); err != nil {
		return protocol.DeskOperation{}, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent != input.Intent.Intent {
		return protocol.DeskOperation{}, errors.New("frontend receipt changed reviewed intent; query original operation ID")
	}
	return out, nil
}
