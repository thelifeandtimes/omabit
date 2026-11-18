package client

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) projectStopRecoveryHello(ctx context.Context, expected Identity) (protocol.Hello, error) {
	h, err := c.Hello(ctx)
	if err != nil {
		return h, err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return h, err
	}
	if !slices.Contains(h.Features, protocol.ProjectStopRecoveryFeature) {
		return h, errors.New("runner does not advertise " + protocol.ProjectStopRecoveryFeature)
	}
	if h.AccessRole != "operator" || h.Actor == "" {
		return h, errors.New("exact project stop diagnosis/reconciliation requires authenticated operator access")
	}
	return h, nil
}

func (c *Client) ProjectMemberDiagnosis(ctx context.Context, expected Identity, project, member, operation string, probe bool) (protocol.ProjectMemberDiagnosis, error) {
	var out protocol.ProjectMemberDiagnosis
	if !protocol.ValidID(project) || !protocol.ValidID(member) || !protocol.ValidID(operation) {
		return out, errors.New("require exact project, member and original operation IDs")
	}
	h, err := c.projectStopRecoveryHello(ctx, expected)
	if err != nil {
		return out, err
	}
	if err = c.do(ctx, http.MethodGet, protocol.ProjectMemberDiagnosisPath(project, member, operation, probe), nil, &out, true); err != nil {
		return protocol.ProjectMemberDiagnosis{}, err
	}
	if err = expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.ProjectMemberDiagnosis{}, err
	}
	if out.SchemaVersion != 1 || out.Actor != h.Actor || out.ProjectID != project || out.InstanceID != member || out.Operation.ID != operation || out.Operation.ProjectID != project || out.Operation.InstanceID != member || out.ReadAt.IsZero() || out.ReasonCode == "" || out.RuntimeObservation != "not-performed" && out.RuntimeObservation != "fresh" && out.RuntimeObservation != "unavailable" || !probe && (out.RuntimeObservation != "not-performed" || out.Review != nil || out.Observation != nil) {
		return protocol.ProjectMemberDiagnosis{}, errors.New("project diagnosis does not match the selected read")
	}
	if review := out.Review; review != nil {
		if review.Validate() != nil || review.RunnerID != h.RunnerID || review.Mode != h.Mode || review.Actor != h.Actor || review.ProjectID != project || review.InstanceID != member || review.OriginalOperationID != operation || out.Operation.Status != "interrupted" || out.Observation == nil || !reflect.DeepEqual(*out.Observation, review.Observation) {
			return protocol.ProjectMemberDiagnosis{}, errors.New("project stop review does not match its diagnosis")
		}
	}
	return out, nil
}

func (c *Client) ReconcileProjectStop(ctx context.Context, expected Identity, project string, req protocol.ProjectStopReconcileRequest) (protocol.ProjectStopReconcileResult, error) {
	var out protocol.ProjectStopReconcileResult
	if !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || !req.ConfirmAction || req.Review.Validate() != nil || req.Review.ProjectID != project {
		return out, errors.New("require an exact reviewed stop, request ID and explicit confirmation")
	}
	h, err := c.projectStopRecoveryHello(ctx, expected)
	if err != nil {
		return out, err
	}
	if req.Review.RunnerID != h.RunnerID || req.Review.Mode != h.Mode || req.Review.Actor != h.Actor {
		return out, errors.New("stop review belongs to a different runner or operator")
	}
	if IsCreationReplayOnly(ctx) && !slices.Contains(h.Features, protocol.CreationReplayFeature) {
		return out, errors.New("runner does not support lookup-only reconciliation")
	}
	path := protocol.ProjectStopReconcilePath(project)
	if err = c.do(ctx, http.MethodPost, path, req, &out, true); err != nil {
		return protocol.ProjectStopReconcileResult{}, err
	}
	op := out.Operation
	err = expected.Check(out.RunnerID, out.Mode)
	if err == nil && (out.SchemaVersion != 1 || out.OriginalOperationID != req.Review.OriginalOperationID || out.ReviewDigest != req.Review.ReviewDigest || out.VerifiedState != "stopped" || !protocol.ValidID(op.ID) || op.ProjectID != project || op.InstanceID != req.Review.InstanceID || op.RequestID != req.RequestID || op.Actor != h.Actor || op.Kind != "reconcile-project-ship-stop" || op.ConfigRevision != req.Review.ProjectRevision || op.Status != "succeeded" || op.CreatedAt.IsZero() || op.FinishedAt == nil || !op.FinishedAt.Equal(op.CreatedAt)) {
		err = errors.New("stop reconciliation receipt does not match the exact review")
	}
	if err != nil {
		return protocol.ProjectStopReconcileResult{}, &OutcomeUncertainError{Method: http.MethodPost, Path: path, Err: err}
	}
	return out, nil
}
