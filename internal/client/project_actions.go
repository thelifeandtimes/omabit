package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// ProjectOperate requires an identity-pinned client; results are operation
// receipts, not live health. Replay-only lookup never schedules new work.
func (c *Client) ProjectOperate(ctx context.Context, expected Identity, id string, req protocol.ProjectActionRequest) (protocol.ProjectActionResult, error) {
	var out protocol.ProjectActionResult
	feature, kind, _ := protocol.ProjectAction(req.Kind)
	if !protocol.ValidID(id) || !protocol.ValidProjectActionRequest(req) {
		return out, errors.New("project action requires an exact reviewed project, revision, request and member selection")
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return out, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return out, err
	}
	if !slices.Contains(hello.Features, feature) {
		return out, errors.New("runner does not advertise " + feature)
	}
	if hello.AccessRole != "operator" || hello.Actor == "" {
		return out, errors.New("project actions require authenticated operator access")
	}
	if IsCreationReplayOnly(ctx) && !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
		return out, errors.New("runner does not support replay-only operation lookup")
	}
	path := protocol.ProjectActionPath(id)
	if err = c.do(ctx, http.MethodPost, path, req, &out, true); err != nil {
		return protocol.ProjectActionResult{}, err
	}
	op := out.Operation
	err = expected.Check(out.RunnerID, out.Mode)
	terminal := op.Status == "succeeded" || op.Status == "failed" || op.Status == "interrupted"
	active := op.Status == "queued" || op.Status == "running"
	if err == nil && (out.SchemaVersion != 1 || out.ReviewDigest != req.ReviewDigest || !protocol.ValidID(op.ID) || op.ProjectID != id || op.InstanceID != req.InstanceID || op.RequestID != req.RequestID || (op.Kind != kind && !(req.Kind == "member-continue" && op.Kind == "start-project-ship")) || op.Actor != hello.Actor || op.ConfigRevision != req.ExpectedRevision || op.CreatedAt.IsZero() || op.ImportRestoration != nil || !terminal && !active || active && op.FinishedAt != nil || terminal && (op.FinishedAt == nil || op.FinishedAt.Before(op.CreatedAt))) {
		err = errors.New("project action response does not match its selected intent")
	}
	if err != nil {
		return protocol.ProjectActionResult{}, &OutcomeUncertainError{Method: http.MethodPost, Path: path, Err: err}
	}
	return out, nil
}
