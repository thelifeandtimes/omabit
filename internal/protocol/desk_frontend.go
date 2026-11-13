package protocol

import (
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"time"
)

const DeskFrontendFeature = "desk-frontend-v1"
const DeskFrontendReviewPath = "/v1/desk-frontend-review"
const DeskFrontendsPath = "/v1/desk-frontends"

// The existing bounded stage transfer also carries built frontend assets.
type DeskFrontendRequest = DeskDeploymentRequest
type DeskFrontendReview struct {
	Version        int                     `json:"version"`
	RunnerID       string                  `json:"runner_id"`
	Mode           string                  `json:"mode"`
	Request        DeskFrontendRequest     `json:"request"`
	PolicyRevision uint64                  `json:"policy_revision"`
	Review         deskwork.FrontendReview `json:"review"`
	ReadAt         time.Time               `json:"read_at"`
}

func (r DeskFrontendReview) Validate() error {
	if r.Version != 1 || !ValidID(r.RunnerID) || !ValidID(r.Mode) || r.Request.Validate() != nil || r.PolicyRevision > apppolicy.MaxRevision || r.Review.Validate() != nil || r.Review.Stage.Target != r.Request.Selection.Target || r.Review.Stage.ID != r.Request.StageID || r.Review.Stage.SettingsRevision != r.Request.SettingsRevision || r.ReadAt.IsZero() {
		return errors.New("invalid frontend review response")
	}
	return nil
}

type DeskFrontendSubmission struct {
	Intent deskwork.FrontendIntent  `json:"intent"`
	Checks []deskwork.FrontendCheck `json:"checks"`
}

func (s DeskFrontendSubmission) Validate() error {
	if s.Intent.Validate() != nil || s.Checks == nil || len(s.Checks) > 16 || s.Intent.ChecksDigest != deskwork.FrontendChecksDigest(s.Checks) {
		return errors.New("invalid frontend submission")
	}
	return nil
}
