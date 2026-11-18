package protocol

import (
	"errors"
	"net/http"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

const GrubberyRuntimeReceiptsFeature = "grubbery-app-runtime-receipts-v1"
const GrubberyRuntimeRecoveryFeature = "grubbery-app-runtime-recovery-v1"
const GrubberyRuntimeReviewPath = "/v1/grubbery-app-runtime-review"
const GrubberyRuntimeRecoveriesPath = "/v1/grubbery-app-runtime-recoveries"

type GrubberyRuntimeReviewRequest struct {
	OriginalRequestID string        `json:"original_request_id"`
	Selection         DeskSelection `json:"selection"`
}

func (q GrubberyRuntimeReviewRequest) Validate() error {
	if !ValidDeskOperationID(q.OriginalRequestID) || q.Selection.Validate() != nil || q.Selection.Target.Desk != grubbery.FrameworkDesk {
		return errors.New("select the original app operation and registered replacement process")
	}
	return nil
}

type GrubberyRuntimeReview struct {
	Version  int                            `json:"version"`
	RunnerID string                         `json:"runner_id"`
	Mode     string                         `json:"mode"`
	Original grubbery.Record                `json:"original"`
	Review   grubbery.RuntimeRecoveryReview `json:"review"`
}

func (q GrubberyRuntimeReview) Validate() error {
	if q.Version != 1 || !ValidID(q.RunnerID) || q.Original.Plan.Binding.RunnerID != q.RunnerID || q.Original.Plan.Binding.Mode != q.Mode || q.Review.ValidateAncestry(q.Original) != nil || q.Review.OriginalRevision != q.Original.Revision {
		return errors.New("invalid app runtime recovery review binding")
	}
	return nil
}

type GrubberyRuntimeApplyRequest struct {
	OriginalRequestID string                         `json:"original_request_id"`
	RequestID         string                         `json:"request_id"`
	Review            grubbery.RuntimeRecoveryReview `json:"review"`
	Confirm           bool                           `json:"confirm"`
}

func (q GrubberyRuntimeApplyRequest) Validate() error {
	if !q.Confirm || !ValidDeskOperationID(q.OriginalRequestID) || !ValidDeskOperationID(q.RequestID) || q.RequestID == q.OriginalRequestID || q.Review.Version != 1 || q.Review.OriginalRequestID != q.OriginalRequestID || q.Review.OriginalBinding.Validate() != nil || q.Review.Process.Validate() != nil || q.Review.Digest != q.Review.ContentDigest() {
		return errors.New("confirm the exact reviewed original app operation and new audit ID")
	}
	return nil
}

// Receipt is nil only for a valid original operation with no saved recovery.
// Original is current immutable history, so clients can validate a historical
// audit after later legitimate phase progress without a native/cache lookup.
type GrubberyRuntimeRecoveryStatus struct {
	Version  int                              `json:"version"`
	RunnerID string                           `json:"runner_id"`
	Mode     string                           `json:"mode"`
	Original grubbery.Record                  `json:"original"`
	Receipt  *grubbery.RuntimeRecoveryReceipt `json:"receipt,omitempty"`
}

func (q GrubberyRuntimeRecoveryStatus) Validate() error {
	if q.Version != 1 || !ValidID(q.RunnerID) || q.Original.Validate() != nil || q.Original.Plan.Binding.RunnerID != q.RunnerID || q.Original.Plan.Binding.Mode != q.Mode {
		return errors.New("invalid original app runtime recovery status")
	}
	if q.Receipt != nil && q.Receipt.ValidateAncestry(q.Original) != nil {
		return errors.New("invalid saved app runtime recovery ancestry")
	}
	return nil
}
func GrubberyRuntimeReceiptPath(path string) (original, audit string, ok bool) {
	if !strings.HasPrefix(path, GrubberyRuntimeRecoveriesPath+"/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, GrubberyRuntimeRecoveriesPath+"/"), "/")
	if len(parts) < 1 || len(parts) > 2 || !ValidDeskOperationID(parts[0]) || len(parts) == 2 && !ValidDeskOperationID(parts[1]) {
		return "", "", false
	}
	if len(parts) == 2 {
		audit = parts[1]
	}
	return parts[0], audit, true
}
func GrubberyRuntimeLiveEndpoint(method, path string) bool {
	return method == http.MethodPost && (path == GrubberyRuntimeReviewPath || path == GrubberyRuntimeRecoveriesPath)
}
