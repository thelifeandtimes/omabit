package deskwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func (p Publisher) Digest() string {
	data, _ := json.Marshal(p)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ReceiverReview binds the unused receiver name, selected continuing publisher,
// workflow settings and current pier. Project installations must select a source
// member of that same project; the runner checks its current generation/name.
// A zero PublisherTarget selects an external publisher for a standalone ship.
type ReceiverReview struct {
	Version         int              `json:"version"`
	Publisher       Publisher        `json:"publisher"`
	PublisherTarget apppolicy.Target `json:"publisher_target"`
	Observation     Observation      `json:"observation"`
	Subscription    Subscription     `json:"subscription"`
	ShipSetting     Setting          `json:"ship_setting"`
	DeskSetting     Setting          `json:"desk_setting"`
	Continuity      string           `json:"continuity"`
	Digest          string           `json:"digest"`
}

func (r ReceiverReview) digest() string {
	r.Digest = ""
	data, _ := json.Marshal(r)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func (r *ReceiverReview) Seal() error { r.Digest = r.digest(); return r.Validate() }
func (r ReceiverReview) Validate() error {
	if r.Version != 1 || r.Publisher.Validate() != nil || r.PublisherTarget.Validate() != nil || r.PublisherTarget.Desk != "" || r.Observation.Validate() != nil || r.Observation.Exists || r.Subscription.Validate() != nil || r.Subscription.Present || r.ShipSetting.Validate() != nil || r.ShipSetting.Ship == nil || r.DeskSetting.Validate() != nil || r.DeskSetting.Desk == nil || r.ShipSetting.Target != r.DeskSetting.Target.Ship() || !digestPattern.MatchString(r.Continuity) || r.Digest != r.digest() {
		return errors.New("invalid receiver installation review")
	}
	target := r.ShipSetting.Target
	if target.ProjectID != "" {
		if r.PublisherTarget.ProjectID != target.ProjectID || r.PublisherTarget.InstanceID == target.InstanceID || r.PublisherTarget.Generation == 0 {
			return errors.New("project receiver requires another current member of the same project")
		}
	} else if r.PublisherTarget != (apppolicy.Target{}) {
		return errors.New("standalone receiver cannot select a private project source")
	}
	return nil
}

// CheckConfiguredReceiver refuses replacing a saved Urgit development binding
// with a remote subscription. Ordinary receiver installation does not publish
// the received desk or claim application-specific health checks have passed.
func (r ReceiverReview) CheckConfiguredReceiver() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.DeskSetting.Desk.Workflow != "mounted" {
		return errors.New("configured source workflow owns this desk; review Setup first")
	}
	return nil
}

type ReceiverIntent struct {
	Intent               Intent           `json:"intent"`
	Publisher            Publisher        `json:"publisher"`
	PublisherTarget      apppolicy.Target `json:"publisher_target"`
	SettingsRevision     uint64           `json:"settings_revision"`
	ShipSettingsRevision uint64           `json:"ship_settings_revision"`
	Continuity           string           `json:"continuity"`
	ReviewDigest         string           `json:"review_digest"`
}

func (i ReceiverIntent) Validate() error {
	if i.Intent.Validate() != nil || i.Intent.Action != "install-remote" || i.Publisher.Validate() != nil || i.Intent.SourceDigest != i.Publisher.Digest() || i.PublisherTarget.Validate() != nil || i.SettingsRevision > apppolicy.MaxRevision || i.ShipSettingsRevision > apppolicy.MaxRevision || !digestPattern.MatchString(i.Continuity) || !digestPattern.MatchString(i.ReviewDigest) {
		return errors.New("invalid receiver installation intent")
	}
	return nil
}
func (i ReceiverIntent) CheckReview(r ReceiverReview) error {
	if i.Validate() != nil || r.Validate() != nil || i.Intent.Target != r.DeskSetting.Target || i.Intent.Baseline != r.Observation || i.Publisher != r.Publisher || i.PublisherTarget != r.PublisherTarget || i.SettingsRevision != r.DeskSetting.Revision || i.ShipSettingsRevision != r.ShipSetting.Revision || i.Continuity != r.Continuity || i.ReviewDigest != r.Digest {
		return errors.New("receiver installation review changed")
	}
	return nil
}

type ReceiverRecord struct {
	Version   int            `json:"version"`
	Intent    ReceiverIntent `json:"intent"`
	Review    ReceiverReview `json:"review"`
	Actor     string         `json:"actor"`
	CreatedAt time.Time      `json:"created_at"`
}

func (r ReceiverRecord) Validate() error {
	if r.Version != 1 || r.Intent.CheckReview(r.Review) != nil || r.Review.CheckConfiguredReceiver() != nil || r.Actor == "" || !settingText(r.Actor, 128, false) || r.CreatedAt.IsZero() {
		return errors.New("invalid receiver installation record")
	}
	return nil
}
