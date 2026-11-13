package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (c *Client) DeskAdoptionReview(ctx context.Context, expected Identity, q protocol.DeskDeploymentRequest, sourceName, sourceDigest string) (protocol.DeskAdoptionReview, error) {
	var out protocol.DeskAdoptionReview
	if q.Validate() != nil || !deskwork.ValidSourceName(sourceName) || !protocol.ValidDeskOperationID(sourceDigest) {
		return out, errors.New("invalid adoption review selection")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskAdoptionFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskAdoptionReviewPath, q, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskAdoptionReview{}, err
	}
	if out.Validate() != nil || out.Request != q || out.SourceName != sourceName || out.SourceDigest != sourceDigest || time.Since(out.ReadAt) > protocol.DeskDeploymentTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskAdoptionReview{}, errors.New("invalid or changed adoption review response")
	}
	return out, nil
}
func (c *Client) AdoptDeskSource(ctx context.Context, expected Identity, intent deskwork.AdoptionIntent) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := intent.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskAdoptionFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskAdoptionPath, intent, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent != intent.Intent {
		return protocol.DeskOperation{}, errors.New("adoption receipt changed the reviewed intent; query the original operation ID")
	}
	state := out.Receipt.Steps[len(out.Receipt.Steps)-1].State
	if state != "observed" && state != "blocked" {
		return protocol.DeskOperation{}, errors.New("adoption response is incomplete; query the original operation ID")
	}
	return out, nil
}
