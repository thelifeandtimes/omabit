package protocol

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// Receipt access is independent of live adapter qualification or current app
// policy. It grants no source, native HTTP or phase-dispatch authority.
const GrubberyAppReceiptsFeature = "grubbery-app-receipts-v1"
const GrubberyAppOperationsPath = "/v1/grubbery-app-operations"

// This denotes typed source API support, not a qualified complete application
// workflow. Preflight requires the complete bounded source/restart capability
// set and reports that scope; every native effect retains its own admission.
const GrubberyAppSourceFeature = "grubbery-app-source-v1"
const GrubberyAppReviewPath = "/v1/grubbery-app-review"
const GrubberyAppRequestBytes = 96 << 10
const GrubberyAppTimeout = 4 * time.Minute

type GrubberyAppReviewRequest struct {
	Selection          DeskSelection `json:"selection"`
	FrameworkReceiptID string        `json:"framework_receipt_id"`
	AppID              string        `json:"app_id"`
	StageID            string        `json:"stage_id"`
	RequestID          string        `json:"request_id"`
}

func (q GrubberyAppReviewRequest) Validate() error {
	if q.Selection.Validate() != nil || q.Selection.Target.Desk != grubbery.FrameworkDesk || !ValidDeskOperationID(q.FrameworkReceiptID) || !grubbery.ValidAppID(q.AppID) || !ValidDeskOperationID(q.StageID) || !ValidDeskOperationID(q.RequestID) {
		return errors.New("invalid app source review selection")
	}
	return nil
}

type GrubberyAppReview struct {
	Version  int                      `json:"version"`
	RunnerID string                   `json:"runner_id"`
	Mode     string                   `json:"mode"`
	Request  GrubberyAppReviewRequest `json:"request"`
	Plan     grubbery.Plan            `json:"plan"`
	ReadAt   time.Time                `json:"read_at"`
}

func (r GrubberyAppReview) Validate() error {
	b, q := r.Plan.Binding, r.Request
	if r.Version != 1 || r.Request.Validate() != nil || r.Plan.ValidateMetadata() != nil || r.Plan.Version != 2 || r.ReadAt.IsZero() || b.RunnerID != r.RunnerID || b.Mode != r.Mode || b.Target != q.Selection.Target || b.RuntimeID != q.Selection.RuntimeID || b.ConfigRevision != q.Selection.ConfigRevision || b.FrameworkReceiptID != q.FrameworkReceiptID || b.AppID != q.AppID || r.Plan.StageID != q.StageID || r.Plan.RequestID != q.RequestID {
		return errors.New("invalid app source review binding")
	}
	return nil
}

type GrubberyAppBeginRequest struct {
	Plan grubbery.Plan `json:"plan"`
}

type GrubberyAppPhaseRequest struct {
	PlanDigest string `json:"plan_digest"`
	StepID     string `json:"step_id"`
}

func (q GrubberyAppPhaseRequest) Validate() error {
	if !ValidDeskOperationID(q.PlanDigest) || !ValidDeskOperationID(q.StepID) {
		return errors.New("select the exact original app plan and phase")
	}
	return nil
}

func GrubberyAppPhasePath(path string) (id, action string, ok bool) {
	parts := strings.Split(strings.TrimPrefix(path, GrubberyAppOperationsPath+"/"), "/")
	if !strings.HasPrefix(path, GrubberyAppOperationsPath+"/") || len(parts) != 2 || !ValidDeskOperationID(parts[0]) || (parts[1] != "advance" && parts[1] != "reconcile" && parts[1] != "abandon") {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func GrubberyAppSourceEndpoint(method, path string) bool {
	_, _, phase := GrubberyAppPhasePath(path)
	return method == http.MethodPost && (path == GrubberyAppOperationsPath || path == GrubberyAppReviewPath || phase)
}

type GrubberyAppOperation struct {
	Version  int             `json:"version"`
	RunnerID string          `json:"runner_id"`
	Mode     string          `json:"mode"`
	Record   grubbery.Record `json:"record"`
}

func (r GrubberyAppOperation) Validate() error {
	if r.Version != 1 || !ValidID(r.RunnerID) || r.Record.Validate() != nil || r.Record.Plan.Binding.RunnerID != r.RunnerID || r.Record.Plan.Binding.Mode != r.Mode {
		return errors.New("invalid app operation receipt binding")
	}
	return nil
}
