package client

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) restoreAuthority(ctx context.Context, runner string) (protocol.Hello, error) {
	h, err := c.Hello(ctx)
	if err != nil {
		return h, err
	}
	if h.RunnerID != runner {
		return h, ErrIdentityMismatch
	}
	if h.AccessRole != "operator" || !slices.Contains(h.Features, protocol.ImportRestoreFeature) {
		return h, errors.New("source restoration requires operator access and restore-import capability")
	}
	return h, nil
}
func validRestoreResult(r protocol.ImportRestoreResult) bool {
	if !epochs.ValidDigest(r.ReviewDigest) {
		return false
	}
	paths := []string{r.Source, r.RestoreTo, r.Destination, r.RetainAt}
	for i, p := range paths {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return false
		}
		for _, q := range paths[:i] {
			if pierstore.Overlap(p, q) {
				return false
			}
		}
	}
	return true
}
func (c *Client) PreviewImportRestore(ctx context.Context, runner, id string, revision uint64) (protocol.ImportRestorePreview, error) {
	var out protocol.ImportRestorePreview
	if !protocol.ValidID(runner) || validateID(id) != nil || revision == 0 || revision > math.MaxInt64 {
		return out, errors.New("restoration preview requires a pinned runner, instance and revision")
	}
	if _, err := c.restoreAuthority(ctx, runner); err != nil {
		return out, err
	}
	err := c.storageRequest(ctx, "/v1/instances/"+id+"/restore-import-preview", protocol.ImportRestorePreviewRequest{ExpectedRevision: revision}, &out)
	if err == nil && (out.RunnerID != runner || out.InstanceID != id || out.ConfigRevision != revision || !validRestoreResult(out.ImportRestoreResult) || out.SourceBytes < 0 || out.SourceBytes > protocol.MaxPierBytes || out.DestinationBytes < 0 || out.DestinationBytes > protocol.MaxPierBytes || out.SourceFiles < 0 || out.SourceFiles > 8192 || out.DestinationFiles < 0 || out.DestinationFiles > 8192 || !slices.Contains([]string{"review", "retaining-destination", "restoring-source", "complete"}, out.Stage) || out.Continuation != (out.Stage != "review")) {
		err = errors.New("restoration preview does not match the selected runner, instance, revision or bounded move plan")
	}
	return out, err
}
func (c *Client) RestoreImport(ctx context.Context, runner, id string, req protocol.OperationRequest) (protocol.Operation, error) {
	var out protocol.Operation
	if !protocol.ValidID(runner) || validateID(id) != nil || !protocol.ValidID(req.RequestID) || req.Kind != protocol.ImportRestoreFeature || req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 || !req.ConfirmRestore || !epochs.ValidDigest(req.RestoreDigest) || req.ConfirmDelete || req.PierPath != "" || req.EpochCutoff != "" || req.InventoryDigest != "" || req.ConfirmHistoryLoss {
		return out, errors.New("restoration requires the exact reviewed digest, revision, request ID and explicit confirmation")
	}
	h, err := c.restoreAuthority(ctx, runner)
	if err != nil {
		return out, err
	}
	out, err = c.Operate(ctx, id, req)
	if err == nil && (out.InstanceID != id || out.RequestID != req.RequestID || out.ConfigRevision != req.ExpectedRevision || out.Kind != req.Kind || out.Actor != h.Actor || !protocol.ValidID(out.ID) || !slices.Contains([]string{"queued", "running", "succeeded", "failed", "interrupted", "canceled"}, out.Status)) {
		err = errors.New("restoration receipt does not match the reviewed operation")
	}
	if err == nil && out.Status == "succeeded" && (out.FinishedAt == nil || out.ImportRestoration == nil || out.ImportRestoration.ReviewDigest != req.RestoreDigest || !validRestoreResult(*out.ImportRestoration)) {
		err = errors.New("successful restoration is missing its retained-data receipt")
	}
	return out, err
}
