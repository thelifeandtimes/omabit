package deskwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

func IsReleaseAction(action string) bool {
	return action == "activate-local" || action == "publish-source" || action == "publish-treaty"
}

// VerifyRelease proves only the requested stage on the exact reviewed source.
// It does not attribute the effect to a particular caller, prove the revision
// of a running Gall agent, or replace the developer's application health tests.
func VerifyRelease(action string, before, after Observation, app AppState) error {
	if !IsReleaseAction(action) || before.Validate() != nil || after.Validate() != nil || !before.Exists || !after.Exists || app.Validate() != nil || before.Revision != after.Revision || before.ClayHash != after.ClayHash || app.SourceRevision != after.Revision {
		return errors.New("release source changed or is unavailable")
	}
	switch action {
	case "activate-local":
		if app.Zest != "live" || app.SourceShip != "" || before.Publication != after.Publication {
			return errors.New("local installation not observed")
		}
		for _, name := range app.ExpectedAgents {
			if !slices.ContainsFunc(app.Agents, func(a AppAgent) bool { return a.Name == name && a.Live }) {
				return errors.New("declared agent is not live")
			}
		}
	case "publish-source":
		if after.Publication != "published" {
			return errors.New("source publication not observed")
		}
	case "publish-treaty":
		if after.Publication != "published" || !app.TreatyListed || app.TreatyHash != after.ClayHash {
			return errors.New("selected Treaty publication not observed")
		}
	}
	return nil
}

// ReleaseReview binds native observations and both levels of saved workflow
// instructions. It is a read-only review, not permission or readiness evidence.
type ReleaseReview struct {
	Version     int         `json:"version"`
	Action      string      `json:"action"`
	Observation Observation `json:"observation"`
	Application AppState    `json:"application"`
	ShipSetting Setting     `json:"ship_setting"`
	DeskSetting Setting     `json:"desk_setting"`
	Continuity  string      `json:"continuity"`
	Digest      string      `json:"digest"`
}

func (r ReleaseReview) digest() string {
	r.Digest = ""
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (r *ReleaseReview) Seal() error { r.Digest = r.digest(); return r.Validate() }
func (r ReleaseReview) Validate() error {
	if r.Version != 1 || !IsReleaseAction(r.Action) || r.Observation.Validate() != nil || !r.Observation.Exists || r.Application.Validate() != nil || r.Application.SourceRevision != r.Observation.Revision || r.ShipSetting.Validate() != nil || r.ShipSetting.Ship == nil || r.DeskSetting.Validate() != nil || r.DeskSetting.Desk == nil || r.ShipSetting.Target != r.DeskSetting.Target.Ship() || !digestPattern.MatchString(r.Continuity) || r.Digest != r.digest() {
		return errors.New("invalid release review")
	}
	return nil
}

// CheckConfiguredRelease is separate from validation so blocked/no-op reviews
// remain inspectable. Only saved Setup settings select the publication method.
func (r ReleaseReview) CheckConfiguredRelease() error {
	if err := r.Validate(); err != nil {
		return err
	}
	switch r.Action {
	case "activate-local":
		if r.Application.SourceShip != "" {
			return errors.New("local activation cannot replace a source subscription")
		}
	case "publish-source":
		if r.DeskSetting.Desk.Publication != "source" {
			return errors.New("Setup does not select source publication for this desk")
		}
	case "publish-treaty":
		if r.DeskSetting.Desk.Publication != "treaty" {
			return errors.New("Setup does not select Treaty publication for this desk")
		}
		if !r.Application.TreatyAvailable || r.Application.DocketHash == "" {
			return errors.New("Treaty and a valid desk docket are required")
		}
	}
	return nil
}

// ReleaseCheck is caller-supplied evidence, not a runner-executed test. Exact
// check names/order and source hashes prevent accidentally reusing another
// revision's checklist. Nothing here is evaluated as a command or URL.
type ReleaseCheck struct {
	Name     string `json:"name"`
	ClayHash string `json:"clay_hash"`
	Passed   bool   `json:"passed"`
	Evidence string `json:"evidence"`
}

func CheckReleaseEvidence(review ReleaseReview, checks []ReleaseCheck) error {
	if review.Validate() != nil || checks == nil || len(checks) > 16 {
		return errors.New("invalid release check evidence")
	}
	// Installation can be necessary to perform the configured health checks.
	// Their evidence is required at publication, never fabricated at activation.
	if review.Action == "activate-local" {
		if len(checks) != 0 {
			return errors.New("activation does not attest publication checks")
		}
		return nil
	}
	if len(checks) != len(review.DeskSetting.Desk.Checks) {
		return errors.New("supply evidence for every configured release check")
	}
	for i, c := range checks {
		if c.Name != review.DeskSetting.Desk.Checks[i] || !c.Passed || c.ClayHash != review.Observation.ClayHash || strings.TrimSpace(c.Evidence) == "" || !settingText(c.Evidence, 2048, true) {
			return errors.New("release checks are missing, failed or bound to another source")
		}
	}
	return nil
}
func ReleaseChecksDigest(checks []ReleaseCheck) string {
	b, _ := json.Marshal(checks)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type ReleaseIntent struct {
	Intent               Intent `json:"intent"`
	SettingsRevision     uint64 `json:"settings_revision"`
	ShipSettingsRevision uint64 `json:"ship_settings_revision"`
	Continuity           string `json:"continuity"`
	ReviewDigest         string `json:"review_digest"`
	ChecksDigest         string `json:"checks_digest"`
}

func (i ReleaseIntent) Validate() error {
	if i.Intent.Validate() != nil || !IsReleaseAction(i.Intent.Action) || i.SettingsRevision > apppolicy.MaxRevision || i.ShipSettingsRevision > apppolicy.MaxRevision || !digestPattern.MatchString(i.Continuity) || !digestPattern.MatchString(i.ReviewDigest) || !digestPattern.MatchString(i.ChecksDigest) {
		return errors.New("invalid release intent")
	}
	return nil
}
func (i ReleaseIntent) CheckReview(r ReleaseReview, checks []ReleaseCheck) error {
	if i.Validate() != nil || r.Validate() != nil || i.Intent.Action != r.Action || i.Intent.Target != r.DeskSetting.Target || i.Intent.Baseline != r.Observation || i.SettingsRevision != r.DeskSetting.Revision || i.ShipSettingsRevision != r.ShipSetting.Revision || i.Continuity != r.Continuity || i.ReviewDigest != r.Digest || i.ChecksDigest != ReleaseChecksDigest(checks) {
		return errors.New("release review changed")
	}
	return CheckReleaseEvidence(r, checks)
}

type ReleaseRecord struct {
	Version   int            `json:"version"`
	Intent    ReleaseIntent  `json:"intent"`
	Review    ReleaseReview  `json:"review"`
	Checks    []ReleaseCheck `json:"checks"`
	Actor     string         `json:"actor"`
	CreatedAt time.Time      `json:"created_at"`
}

func (r ReleaseRecord) Validate() error {
	if r.Version != 1 || r.Intent.CheckReview(r.Review, r.Checks) != nil || r.Review.CheckConfiguredRelease() != nil || r.Actor == "" || !settingText(r.Actor, 128, false) || r.CreatedAt.IsZero() {
		return errors.New("invalid durable release record")
	}
	return nil
}
