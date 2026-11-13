package protocol

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const DeskReleaseFeature = "desk-release-v1"
const DeskReleaseReviewPath = "/v1/desk-release-review"
const DeskReleasePath = "/v1/desk-releases"

type DeskReleaseRequest struct {
	Selection DeskSelection `json:"selection"`
	Action    string        `json:"action"`
}

func (q DeskReleaseRequest) Validate() error {
	if q.Selection.Validate() != nil || !deskwork.IsReleaseAction(q.Action) {
		return errors.New("invalid release selection")
	}
	return nil
}

type DeskReleaseReview struct {
	Version        int                    `json:"version"`
	RunnerID       string                 `json:"runner_id"`
	Mode           string                 `json:"mode"`
	Request        DeskReleaseRequest     `json:"request"`
	PolicyRevision uint64                 `json:"policy_revision"`
	Review         deskwork.ReleaseReview `json:"review"`
	ReadAt         time.Time              `json:"read_at"`
}

func (r DeskReleaseReview) Validate() error {
	if r.Version != 1 || !ValidID(r.RunnerID) || !ValidID(r.Mode) || r.Request.Validate() != nil || r.PolicyRevision > apppolicy.MaxRevision || r.Review.Validate() != nil || r.Review.Action != r.Request.Action || r.Review.DeskSetting.Target != r.Request.Selection.Target || r.ReadAt.IsZero() {
		return errors.New("invalid release review response")
	}
	return nil
}

type DeskReleaseSubmission struct {
	Intent deskwork.ReleaseIntent  `json:"intent"`
	Checks []deskwork.ReleaseCheck `json:"checks"`
}

func (s DeskReleaseSubmission) Validate() error {
	if s.Intent.Validate() != nil || s.Checks == nil || len(s.Checks) > 16 || s.Intent.ChecksDigest != deskwork.ReleaseChecksDigest(s.Checks) {
		return errors.New("invalid release submission")
	}
	return nil
}
