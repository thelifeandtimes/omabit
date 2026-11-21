package service

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type AppRuntimeStatusInput struct {
	Target
	OriginalRequestID string `json:"original_request_id"`
	AuditRequestID    string `json:"audit_request_id,omitempty"`
}

func (in AppRuntimeStatusInput) Validate() error {
	if in.Target.Validate() != nil || !protocol.ValidDeskOperationID(in.OriginalRequestID) || in.AuditRequestID != "" && !protocol.ValidDeskOperationID(in.AuditRequestID) {
		return invalid()
	}
	return nil
}

type AppRuntimeReviewInput struct {
	Target
	Request protocol.GrubberyRuntimeReviewRequest `json:"request"`
}

func (in AppRuntimeReviewInput) Validate() error {
	if in.Target.Validate() != nil || in.Request.Validate() != nil {
		return invalid()
	}
	return nil
}

type AppRuntimeApplyInput struct {
	Target
	Request protocol.GrubberyRuntimeApplyRequest `json:"request"`
}

func (in AppRuntimeApplyInput) Validate() error {
	if in.Target.Validate() != nil || in.Request.Validate() != nil || in.RunnerID != in.Request.Review.OriginalBinding.RunnerID {
		return invalid()
	}
	return nil
}

func (s *Service) registerGrubberyRuntime() error {
	bindings := "selected runner/mode, original authenticated actor, immutable original app prefix and saved audit head, exact registered replacement process/pier/framework"
	status := description("app.runtime-recovery.v1", "Read an exact saved app runtime recovery audit or latest head without native/source access.", nil, []string{"local-metadata-read", "rpc-read"}, []string{protocol.GrubberyRuntimeReceiptsFeature}, "operator", bindings, "saved-only; retain original operation and audit IDs after lost apply reply")
	review := description("app.review-runtime-recovery.v1", "Review retained native app uncertainty/prefix on a separately verified replacement process; no app write or journal change.", nil, []string{"local-metadata-read", "live-probe"}, []string{protocol.GrubberyRuntimeRecoveryFeature}, "operator", bindings, "fresh bounded review only; no redispatch, lifecycle action or permission grant")
	apply := description("app.apply-runtime-recovery.v1", "Confirm and reobserve an exact process recovery review; append only a new linked audit, preserving original app uncertainty.", nil, []string{"local-metadata-read", "live-probe", "journal-write"}, []string{protocol.GrubberyRuntimeRecoveryFeature}, "operator", bindings, "exact confirmed review and original/audit IDs; lost reply uses saved lookup before another apply")
	return errors.Join(
		selected(s, status, func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppRuntimeStatusInput) (protocol.GrubberyRuntimeRecoveryStatus, error) {
			return r.GrubberyRuntimeRecovery(ctx, in.Name, in.RunnerID, in.OriginalRequestID, in.AuditRequestID)
		}),
		selected(s, review, func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppRuntimeReviewInput) (protocol.GrubberyRuntimeReview, error) {
			return r.ReviewGrubberyRuntime(ctx, in.Name, in.RunnerID, in.Request)
		}),
		selected(s, apply, func(ctx context.Context, r *collection.Registry, _ collection.Connection, in AppRuntimeApplyInput) (protocol.GrubberyRuntimeRecoveryStatus, error) {
			return r.ApplyGrubberyRuntime(ctx, in.Name, in.RunnerID, in.Request)
		}),
	)
}
