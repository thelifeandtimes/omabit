package protocol

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const DeskReceiverFeature = "desk-receiver-v1"
const DeskReceiverReviewPath = "/v1/desk-receiver-review"
const DeskReceiversPath = "/v1/desk-receivers"
const DeskReceiverStatusPath = "/v1/desk-receiver-status"

// SourceReady distinguishes a pending subscription (including a revision-zero
// reserved Clay name) from received source. App stages are observations, not a
// declaration of application-specific health or publication readiness.
type DeskReceiverStatus struct {
	Version      int                   `json:"version"`
	RunnerID     string                `json:"runner_id"`
	Mode         string                `json:"mode"`
	Selection    DeskSelection         `json:"selection"`
	Subscription deskwork.Subscription `json:"subscription"`
	SourceReady  bool                  `json:"source_ready"`
	Observation  *deskwork.Observation `json:"observation,omitempty"`
	Application  *deskwork.AppState    `json:"application,omitempty"`
	ReadAt       time.Time             `json:"read_at"`
}

func (s DeskReceiverStatus) Validate() error {
	if s.Version != 1 || !ValidID(s.RunnerID) || !ValidID(s.Mode) || s.Selection.Validate() != nil || s.Subscription.Validate() != nil || s.ReadAt.IsZero() {
		return errors.New("invalid receiver status")
	}
	if s.SourceReady {
		if s.Observation == nil || s.Application == nil || s.Observation.Validate() != nil || !s.Observation.Exists || s.Application.Validate() != nil || s.Application.SourceRevision != s.Observation.Revision {
			return errors.New("invalid received source observation")
		}
	} else if s.Observation != nil || s.Application != nil {
		return errors.New("pending source cannot claim received application stages")
	}
	return nil
}

type DeskReceiverRequest struct {
	Selection       DeskSelection      `json:"selection"`
	Publisher       deskwork.Publisher `json:"publisher"`
	PublisherTarget apppolicy.Target   `json:"publisher_target"`
}

func (q DeskReceiverRequest) Validate() error {
	if q.Selection.Validate() != nil || q.Publisher.Validate() != nil || q.PublisherTarget.Validate() != nil || q.PublisherTarget.Desk != "" {
		return errors.New("invalid receiver selection")
	}
	target := q.Selection.Target
	if target.ProjectID != "" {
		if q.PublisherTarget.ProjectID != target.ProjectID || q.PublisherTarget.InstanceID == target.InstanceID || q.PublisherTarget.Generation == 0 {
			return errors.New("select a publisher in the same project")
		}
	} else if q.PublisherTarget != (apppolicy.Target{}) {
		return errors.New("standalone receivers cannot select private project sources")
	}
	return nil
}

type DeskReceiverReview struct {
	Version        int                     `json:"version"`
	RunnerID       string                  `json:"runner_id"`
	Mode           string                  `json:"mode"`
	Request        DeskReceiverRequest     `json:"request"`
	PolicyRevision uint64                  `json:"policy_revision"`
	Review         deskwork.ReceiverReview `json:"review"`
	ReadAt         time.Time               `json:"read_at"`
}

func (r DeskReceiverReview) Validate() error {
	if r.Version != 1 || !ValidID(r.RunnerID) || !ValidID(r.Mode) || r.Request.Validate() != nil || r.PolicyRevision > apppolicy.MaxRevision || r.Review.Validate() != nil || r.Review.DeskSetting.Target != r.Request.Selection.Target || r.Review.Publisher != r.Request.Publisher || r.Review.PublisherTarget != r.Request.PublisherTarget || r.ReadAt.IsZero() {
		return errors.New("invalid receiver review response")
	}
	return nil
}
