package client

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) requireDesks(ctx context.Context, expected Identity) error {
	return c.requireDeskFeature(ctx, expected, protocol.DeskCreateFeature)
}
func (c *Client) requireDeskFeature(ctx context.Context, expected Identity, feature string) error {
	h, err := c.Hello(ctx)
	if err != nil {
		return err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return err
	}
	if !slices.Contains(h.Features, feature) {
		return errors.New("runner lacks " + feature + "; use a compatible runner")
	}
	return nil
}
func (c *Client) DeskObservation(ctx context.Context, expected Identity, selection protocol.DeskSelection) (protocol.DeskObservation, error) {
	var out protocol.DeskObservation
	if err := selection.Validate(); err != nil {
		return out, err
	}
	if err := c.requireDesks(ctx, expected); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.DeskReadTimeout + time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, http.MethodPost, protocol.DeskObservationPath, selection, &out, true); err != nil {
		return out, err
	}
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.DeskObservation{}, err
	}
	if out.Version != 1 || out.Selection != selection || out.ReadAt.IsZero() || time.Since(out.ReadAt) > protocol.DeskReadTimeout || out.ReadAt.After(time.Now().Add(5*time.Second)) || out.Observation.Validate() != nil {
		return protocol.DeskObservation{}, errors.New("invalid desk observation response")
	}
	return out, nil
}
func (c *Client) CreateDesk(ctx context.Context, expected Identity, intent deskwork.Intent) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if err := intent.Validate(); err != nil {
		return out, err
	}
	if intent.Action != "create-desk" {
		return out, errors.New("expected create-desk intent")
	}
	if err := c.requireDesks(ctx, expected); err != nil {
		return out, err
	}
	if err := c.do(ctx, http.MethodPost, protocol.DeskOperationsPath, intent, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent != intent {
		return protocol.DeskOperation{}, errors.New("desk operation response changed the reviewed intent")
	}
	return out, nil
}
func (c *Client) DeskOperation(ctx context.Context, expected Identity, id string) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if !protocol.ValidDeskOperationID(id) {
		return out, errors.New("invalid desk operation ID")
	}
	if err := c.requireDesks(ctx, expected); err != nil {
		return out, err
	}
	if err := c.do(ctx, http.MethodGet, protocol.DeskOperationsPath+"/"+id, nil, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent.ID != id {
		return protocol.DeskOperation{}, errors.New("desk operation response changed identity")
	}
	return out, nil
}
func validateDeskReply(expected Identity, out protocol.DeskOperation) error {
	if err := expected.Check(out.RunnerID, out.Mode); err != nil {
		return err
	}
	if out.Version != 1 || out.Receipt.Validate() != nil {
		return errors.New("invalid desk operation receipt")
	}
	return nil
}

func (c *Client) ReconcileDeskOperation(ctx context.Context, expected Identity, id string) (protocol.DeskOperation, error) {
	var out protocol.DeskOperation
	if !protocol.ValidDeskOperationID(id) {
		return out, errors.New("invalid desk operation ID")
	}
	if err := c.requireDesks(ctx, expected); err != nil {
		return out, err
	}
	if err := c.do(ctx, http.MethodPost, protocol.DeskOperationsPath+"/"+id+"/reconcile", struct{}{}, &out, true); err != nil {
		return out, err
	}
	if err := validateDeskReply(expected, out); err != nil {
		return protocol.DeskOperation{}, err
	}
	if out.Receipt.Intent.ID != id {
		return protocol.DeskOperation{}, errors.New("desk reconciliation changed identity")
	}
	return out, nil
}
