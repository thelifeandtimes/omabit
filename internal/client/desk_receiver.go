package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) DeskReceiverReview(ctx context.Context, expected Identity, q protocol.DeskReceiverRequest) (protocol.DeskReceiverReview, error) {
	var out protocol.DeskReceiverReview
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskReceiverFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskReceiverReviewPath, q, &out, true); err != nil {
		return protocol.DeskReceiverReview{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskReceiverReview{}, err
	}
	if out.Validate() != nil || out.Request != q || time.Since(out.ReadAt) > protocol.DeskDeploymentTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskReceiverReview{}, errors.New("invalid or changed receiver review")
	}
	return out, nil
}
func (c *Client) InstallDeskReceiver(ctx context.Context, expected Identity, input deskwork.ReceiverIntent) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := input.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskReceiverFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskDeploymentTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskReceiversPath, input, &out, true); err != nil {
		return protocol.DeskOperation{}, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent != input.Intent {
		return protocol.DeskOperation{}, errors.New("receiver receipt changed reviewed intent; query original operation ID")
	}
	return out, nil
}

func (c *Client) DeskReceiverStatus(ctx context.Context, expected Identity, selection protocol.DeskSelection) (protocol.DeskReceiverStatus, error) {
	var out protocol.DeskReceiverStatus
	if err := selection.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.DeskReceiverFeature); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskReceiverStatusPath, selection, &out, true); err != nil {
		return protocol.DeskReceiverStatus{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskReceiverStatus{}, err
	}
	if out.Validate() != nil || out.Selection != selection || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.DeskReceiverStatus{}, errors.New("invalid or changed receiver status")
	}
	return out, nil
}
