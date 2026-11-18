package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"time"

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

func (c *Client) appSourceRPC(ctx context.Context, expected Identity, path string, in, out any) error {
	wire, err := json.Marshal(in)
	if err != nil || len(wire) > protocol.GrubberyAppRequestBytes {
		return errors.New("app request exceeds bounded schema")
	}
	if err := c.requireDeskFeature(ctx, expected, protocol.GrubberyAppSourceFeature); err != nil {
		return err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.GrubberyAppTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	return scoped.do(ctx, http.MethodPost, path, in, out, true)
}

func (c *Client) ReviewGrubberyAppSource(ctx context.Context, expected Identity, q protocol.GrubberyAppReviewRequest) (protocol.GrubberyAppReview, error) {
	var out protocol.GrubberyAppReview
	if err := q.Validate(); err != nil {
		return out, err
	}
	if err := c.appSourceRPC(ctx, expected, protocol.GrubberyAppReviewPath, q, &out); err != nil {
		return protocol.GrubberyAppReview{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyAppReview{}, err
	}
	if out.Validate() != nil || out.Request != q || time.Since(out.ReadAt) > protocol.GrubberyAppTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) {
		return protocol.GrubberyAppReview{}, errors.New("app review response changed selection or source plan")
	}
	return out, nil
}

func (c *Client) BeginGrubberyAppSource(ctx context.Context, expected Identity, q protocol.GrubberyAppBeginRequest) (protocol.GrubberyAppOperation, error) {
	var out protocol.GrubberyAppOperation
	if err := q.Plan.ValidateMetadata(); err != nil {
		return out, err
	}
	if err := expected.Check(q.Plan.Binding.RunnerID, q.Plan.Binding.Mode); err != nil {
		return out, err
	}
	if err := c.appSourceRPC(ctx, expected, protocol.GrubberyAppOperationsPath, q, &out); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if out.Validate() != nil || !reflect.DeepEqual(out.Record.Plan, q.Plan) {
		return protocol.GrubberyAppOperation{}, errors.New("app begin changed original plan; inspect original request ID")
	}
	return out, nil
}

// Action is restricted to three named exact-phase operations. No method, URL,
// native command, source bytes or proof can be supplied through this interface.
func (c *Client) GrubberyAppSourcePhase(ctx context.Context, expected Identity, id, action string, q protocol.GrubberyAppPhaseRequest) (protocol.GrubberyAppOperation, error) {
	var out protocol.GrubberyAppOperation
	path := protocol.GrubberyAppOperationsPath + "/" + id + "/" + action
	if _, _, ok := protocol.GrubberyAppPhasePath(path); !ok || q.Validate() != nil {
		return out, errors.New("invalid original app phase selection")
	}
	if err := c.appSourceRPC(ctx, expected, path, q, &out); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.GrubberyAppOperation{}, err
	}
	if out.Validate() != nil || out.Record.Plan.RequestID != id || out.Record.Plan.Digest != q.PlanDigest {
		return protocol.GrubberyAppOperation{}, errors.New("app phase response changed original plan or history")
	}
	if _, err := out.Record.State(q.StepID); err != nil {
		return protocol.GrubberyAppOperation{}, errors.New("app phase response omitted original phase")
	}
	return out, nil
}
