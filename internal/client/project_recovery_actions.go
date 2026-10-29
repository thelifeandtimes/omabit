package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"reflect"
	"slices"
)

func (c *Client) ProjectRecover(ctx context.Context, expected Identity, project, parent, kind string, req protocol.ProjectRecoveryActionRequest) (protocol.ProjectRecoveryActionResult, error) {
	var zero protocol.ProjectRecoveryActionResult
	if !protocol.ValidID(project) || !protocol.ValidProjectRecoveryAction(kind, parent, req) {
		return zero, errors.New("recovery requires an explicit reviewed action and exact history/checkpoint")
	}
	hello, err := c.Hello(ctx)
	if err != nil {
		return zero, err
	}
	if err = expected.Check(hello.RunnerID, hello.Mode); err != nil {
		return zero, err
	}
	feature, operation := protocol.ProjectRecoveryAction(kind)
	if !slices.Contains(hello.Features, feature) {
		return zero, errors.New("runner does not advertise " + feature)
	}
	if hello.AccessRole != "operator" || hello.Actor == "" {
		return zero, errors.New("recovery requires authenticated operator access")
	}
	if IsCreationReplayOnly(ctx) && !slices.Contains(hello.Features, protocol.CreationReplayFeature) {
		return zero, errors.New("runner does not support replay-only lookup")
	}
	path := protocol.ProjectRecoveryActionPath(project, parent, kind)
	var out protocol.ProjectRecoveryActionResult
	if err = c.do(ctx, http.MethodPost, path, req, &out, true); err != nil {
		return zero, err
	}
	op := out.Operation
	err = expected.Check(out.RunnerID, out.Mode)
	terminal := op.Status == "succeeded" || op.Status == "failed" || op.Status == "interrupted"
	active := op.Status == "queued" || op.Status == "running"
	if kind == "start" {
		parent = op.ID
	}
	if err == nil && (out.SchemaVersion != 1 || out.ReviewDigest != req.ReviewDigest || out.HistoryDigest != req.HistoryDigest || out.CheckpointDigest != req.CheckpointDigest || !reflect.DeepEqual(out.ExpectedSequence, req.ExpectedSequence) || out.RecoveryOperationID != parent || !protocol.ValidID(op.ID) || op.ProjectID != project || op.InstanceID != "" || op.RequestID != req.RequestID || op.Kind != operation || op.Actor != hello.Actor || op.ConfigRevision != req.ExpectedRevision || op.CreatedAt.IsZero() || op.ImportRestoration != nil || !terminal && !active || active && op.FinishedAt != nil || terminal && (op.FinishedAt == nil || op.FinishedAt.Before(op.CreatedAt))) {
		err = errors.New("recovery response does not match selected intent")
	}
	if err != nil {
		return zero, &OutcomeUncertainError{Method: http.MethodPost, Path: path, Err: err}
	}
	return out, nil
}
