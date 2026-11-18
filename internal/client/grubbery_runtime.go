package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) GrubberyRuntimeRecovery(ctx context.Context, expected Identity, original, audit string) (protocol.GrubberyRuntimeRecoveryStatus, error) {
	var out protocol.GrubberyRuntimeRecoveryStatus
	if !protocol.ValidDeskOperationID(original) || audit != "" && !protocol.ValidDeskOperationID(audit) {
		return out, errors.New("invalid original app operation or audit ID")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.GrubberyRuntimeReceiptsFeature); err != nil {
		return out, err
	}
	path := protocol.GrubberyRuntimeRecoveriesPath + "/" + original
	if audit != "" {
		path += "/" + audit
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out, true); err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	if out.Validate() != nil || out.Original.Plan.RequestID != original || audit != "" && (out.Receipt == nil || out.Receipt.RequestID != audit) {
		return protocol.GrubberyRuntimeRecoveryStatus{}, errors.New("app runtime recovery lookup changed original identity or ancestry")
	}
	return out, nil
}

func (c *Client) appRuntimeRPC(ctx context.Context, expected Identity, path string, in, out any) error {
	wire, err := json.Marshal(in)
	if err != nil || len(wire) > protocol.GrubberyAppRequestBytes {
		return errors.New("app runtime recovery exceeds bounded schema")
	}
	if err = c.requireDeskFeature(ctx, expected, protocol.GrubberyRuntimeRecoveryFeature); err != nil {
		return err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.GrubberyAppTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	return scoped.do(ctx, http.MethodPost, path, in, out, true)
}
func (c *Client) ReviewGrubberyRuntime(ctx context.Context, expected Identity, q protocol.GrubberyRuntimeReviewRequest) (protocol.GrubberyRuntimeReview, error) {
	var out protocol.GrubberyRuntimeReview
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := c.appRuntimeRPC(ctx, expected, protocol.GrubberyRuntimeReviewPath, q, &out); err != nil {
		return protocol.GrubberyRuntimeReview{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyRuntimeReview{}, err
	}
	b := out.Review.OriginalBinding
	if out.Validate() != nil || out.Original.Plan.RequestID != q.OriginalRequestID || b.Target != q.Selection.Target || b.ConfigRevision != q.Selection.ConfigRevision || out.Review.Process.ContainerID != q.Selection.RuntimeID || time.Since(out.Review.ObservedAt) > 5*time.Minute || out.Review.ObservedAt.After(time.Now().Add(5*time.Second)) {
		return protocol.GrubberyRuntimeReview{}, errors.New("app runtime review changed exact original selection or freshness")
	}
	return out, nil
}
func (c *Client) ApplyGrubberyRuntime(ctx context.Context, expected Identity, q protocol.GrubberyRuntimeApplyRequest) (protocol.GrubberyRuntimeRecoveryStatus, error) {
	var out protocol.GrubberyRuntimeRecoveryStatus
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := expected.Check(q.Review.OriginalBinding.RunnerID, q.Review.OriginalBinding.Mode); err != nil {
		return out, err
	}
	if err := c.appRuntimeRPC(ctx, expected, protocol.GrubberyRuntimeRecoveriesPath, q, &out); err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	if out.Validate() != nil || out.Original.Plan.RequestID != q.OriginalRequestID || out.Receipt == nil || out.Receipt.RequestID != q.RequestID || out.Receipt.Review.Digest != q.Review.Digest {
		return protocol.GrubberyRuntimeRecoveryStatus{}, errors.New("app runtime apply changed original review/audit identity; inspect saved receipt")
	}
	return out, nil
}
