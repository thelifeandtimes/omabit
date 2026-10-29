package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrProjectRegistrationUnsupported = errors.New("runner does not advertise project-registration-v1; project reservations are unavailable")

// RegisterProject uses an identity-pinned client. A lost or malformed response
// is uncertain; callers retain the same request and use replay-only lookup.
func (c *Client) RegisterProject(ctx context.Context, expected Identity, req protocol.ProjectRegisterRequest) (protocol.ProjectRegistrationResult, error) {
	var result protocol.ProjectRegistrationResult
	request := projectnetwork.Request{Label: req.Label, ConfirmSharedAdministration: req.ConfirmSharedAdministration}
	for _, m := range req.Members {
		request.Members = append(request.Members, projectnetwork.MemberRequest{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	plan, err := projectnetwork.Build(request, projectnetwork.Reservations{})
	if err != nil {
		return result, err
	}
	digest, err := plan.Digest()
	if err != nil || digest != req.ReviewDigest || !req.ConfirmReservation || !protocol.ValidID(req.RequestID) {
		return result, errors.New("project registration must match the reviewed plan and explicit reservation acknowledgement")
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return result, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return result, err
	}
	if !slices.Contains(hello.Features, protocol.ProjectRegistrationFeature) {
		return result, ErrProjectRegistrationUnsupported
	}
	if hello.AccessRole != "operator" || hello.Actor == "" {
		return result, errors.New("project reservations require authenticated operator access")
	}
	if IsCreationReplayOnly(ctx) && !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
		return result, errors.New("runner does not support replay-only reservation lookup")
	}
	if err = c.do(ctx, http.MethodPost, protocol.ProjectInventoryPath, req, &result, true); err != nil {
		return protocol.ProjectRegistrationResult{}, err
	}
	op := result.Operation
	if err = expected.Check(result.RunnerID, result.Mode); err == nil {
		if result.SchemaVersion != 1 || !result.ReservationOnly || result.ReviewDigest != digest || !protocol.ValidID(op.ID) || !protocol.ValidID(op.ProjectID) || op.InstanceID != "" || op.RequestID != req.RequestID || op.Actor != hello.Actor || op.Kind != "register-project" || op.Status != "succeeded" || op.ConfigRevision != 1 || op.CreatedAt.IsZero() || op.FinishedAt == nil || !op.FinishedAt.Equal(op.CreatedAt) || op.ImportRestoration != nil {
			err = errors.New("project registration response does not match the selected reservation")
		}
	}
	if err != nil {
		return protocol.ProjectRegistrationResult{}, &OutcomeUncertainError{Method: http.MethodPost, Path: protocol.ProjectInventoryPath, Err: err}
	}
	return result, nil
}
