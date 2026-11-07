package deskwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// FrontendReview binds an immutable staged artifact to a local installation.
// UnchangedSourceDigest excludes desk.docket-0, which Docket itself commits.
// No credential or artifact bytes are retained in the operation journal.
type FrontendReview struct {
	Version               int           `json:"version"`
	Ship                  string        `json:"ship"`
	Observation           Observation   `json:"observation"`
	Application           AppState      `json:"application"`
	Frontend              FrontendState `json:"frontend"`
	ShipSetting           Setting       `json:"ship_setting"`
	DeskSetting           Setting       `json:"desk_setting"`
	Continuity            string        `json:"continuity"`
	Stage                 StageIntent   `json:"stage"`
	ArtifactDigest        string        `json:"artifact_digest"`
	ExpectedGlobHash      string        `json:"expected_glob_hash"`
	UnchangedSourceDigest string        `json:"unchanged_source_digest"`
	Digest                string        `json:"digest"`
}

func (r FrontendReview) digest() string {
	r.Digest = ""
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (r *FrontendReview) Seal() error { r.Digest = r.digest(); return r.Validate() }
func (r FrontendReview) Validate() error {
	if r.Version != 1 || !validSelection(r.Ship, r.Stage.Target.Desk) || r.Observation.Validate() != nil || !r.Observation.Exists || r.Application.Validate() != nil || r.Frontend.Validate() != nil || r.Application.SourceRevision != r.Observation.Revision || r.Frontend.SourceRevision != r.Observation.Revision || r.ShipSetting.Validate() != nil || r.ShipSetting.Ship == nil || r.DeskSetting.Validate() != nil || r.DeskSetting.Desk == nil || r.ShipSetting.Target != r.DeskSetting.Target.Ship() || r.Stage.Validate() != nil || r.Stage.Target != r.DeskSetting.Target || r.Stage.SettingsRevision != r.DeskSetting.Revision || r.Stage.Files < 1 || r.Stage.Files > MaxFrontendFiles || r.Stage.Bytes > MaxFrontendBytes || !digestPattern.MatchString(r.Continuity) || !digestPattern.MatchString(r.ArtifactDigest) || !validFrontendHash(r.ExpectedGlobHash) || !digestPattern.MatchString(r.UnchangedSourceDigest) || r.Digest != r.digest() {
		return errors.New("invalid frontend review")
	}
	return nil
}
func (r FrontendReview) CheckConfiguredUpload() error {
	if r.Validate() != nil || r.DeskSetting.Desk.Workflow != "mounted" {
		return errors.New("frontend upload requires the reviewed Unix workflow; preserve Urgit's native pipeline")
	}
	if err := CheckFrontendUpload(r.Ship, r.Application, r.Frontend); err != nil {
		return err
	}
	if r.Application.TreatyListed && r.Application.TreatyHash != r.Observation.ClayHash {
		return errors.New("Treaty has not observed the reviewed source")
	}
	if r.Observation.Publication == "published" || r.Application.TreatyListed {
		if r.DeskSetting.Desk.Publication == "none" || r.Application.TreatyListed && r.DeskSetting.Desk.Publication != "treaty" {
			return errors.New("Setup publication method does not match this distributed desk")
		}
	}
	return nil
}

// FrontendChecks attest the developer's configured release checks against both
// reviewed source and built artifact. They are data, never executable commands.
// Private preparation may precede application checks; a distributed update may not.
type FrontendCheck struct {
	Name           string `json:"name"`
	ClayHash       string `json:"clay_hash"`
	ArtifactDigest string `json:"artifact_digest"`
	Passed         bool   `json:"passed"`
	Evidence       string `json:"evidence"`
}

func FrontendChecksDigest(checks []FrontendCheck) string {
	b, _ := json.Marshal(checks)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func CheckFrontendEvidence(r FrontendReview, checks []FrontendCheck) error {
	if r.Validate() != nil || checks == nil || len(checks) > 16 {
		return errors.New("invalid frontend check evidence")
	}
	required := r.DeskSetting.Desk.Checks
	if r.Observation.Publication == "private" && !r.Application.TreatyListed {
		required = nil
	}
	if len(checks) != len(required) {
		return errors.New("supply the configured checks for this frontend release")
	}
	for i, c := range checks {
		if c.Name != required[i] || c.ClayHash != r.Observation.ClayHash || c.ArtifactDigest != r.ArtifactDigest || !c.Passed || strings.TrimSpace(c.Evidence) == "" || !settingText(c.Evidence, 2048, true) {
			return errors.New("frontend checks are missing, failed or bound to another artifact/source")
		}
	}
	return nil
}

type FrontendIntent struct {
	Intent               Intent `json:"intent"`
	StageID              string `json:"stage_id"`
	SettingsRevision     uint64 `json:"settings_revision"`
	ShipSettingsRevision uint64 `json:"ship_settings_revision"`
	Continuity           string `json:"continuity"`
	ReviewDigest         string `json:"review_digest"`
	ChecksDigest         string `json:"checks_digest"`
}

func (i FrontendIntent) Validate() error {
	if i.Intent.Validate() != nil || i.Intent.Action != "upload-frontend" || !digestPattern.MatchString(i.StageID) || i.SettingsRevision > apppolicy.MaxRevision || i.ShipSettingsRevision > apppolicy.MaxRevision || !digestPattern.MatchString(i.Continuity) || !digestPattern.MatchString(i.ReviewDigest) || !digestPattern.MatchString(i.ChecksDigest) {
		return errors.New("invalid frontend intent")
	}
	return nil
}
func (i FrontendIntent) CheckReview(r FrontendReview, checks []FrontendCheck) error {
	if i.Validate() != nil || r.Validate() != nil || i.Intent.Target != r.DeskSetting.Target || i.Intent.Baseline != r.Observation || i.Intent.SourceDigest != r.ArtifactDigest || i.StageID != r.Stage.ID || i.SettingsRevision != r.DeskSetting.Revision || i.ShipSettingsRevision != r.ShipSetting.Revision || i.Continuity != r.Continuity || i.ReviewDigest != r.Digest || i.ChecksDigest != FrontendChecksDigest(checks) {
		return errors.New("frontend review changed")
	}
	return CheckFrontendEvidence(r, checks)
}

type FrontendRecord struct {
	Version   int             `json:"version"`
	Intent    FrontendIntent  `json:"intent"`
	Review    FrontendReview  `json:"review"`
	Checks    []FrontendCheck `json:"checks"`
	Actor     string          `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
}

func (r FrontendRecord) Validate() error {
	if r.Version != 1 || r.Intent.CheckReview(r.Review, r.Checks) != nil || r.Review.CheckConfiguredUpload() != nil || r.Actor == "" || !settingText(r.Actor, 128, false) || r.CreatedAt.IsZero() {
		return errors.New("invalid durable frontend record")
	}
	return nil
}

// Transfer evidence keeps cleanup independent of upload acknowledgement. A lost
// login/logout reply may leave an owner session; no credential is persisted here.
type FrontendTransfer struct {
	Attempted      bool   `json:"attempted"`
	Acknowledged   bool   `json:"acknowledged"`
	SessionCleanup string `json:"session_cleanup"`
}

func (r FrontendTransfer) Validate() error {
	if r.Acknowledged && !r.Attempted {
		return errors.New("invalid frontend acknowledgement")
	}
	switch r.SessionCleanup {
	case "not-created":
		if r.Attempted {
			return errors.New("upload without session")
		}
	case "confirmed", "unconfirmed":
	default:
		return errors.New("invalid frontend session cleanup")
	}
	return nil
}

// FrontendSourceDigest detects unrelated source changes. Docket is intentionally
// omitted; its expected glob, base and host are verified separately. This neither
// adopts source ownership nor provides an atomic native revision precondition.
func FrontendSourceDigest(clay Manifest) (string, error) {
	if !canonicalManifest(clay) {
		return "", errors.New("invalid Clay inventory")
	}
	files := []File{}
	found := false
	for _, f := range clay.Files {
		if f.Path == "desk.docket-0" {
			found = true
		} else {
			files = append(files, f)
		}
	}
	if !found {
		return "", errors.New("frontend requires a Clay docket")
	}
	m, err := NewManifest(files)
	return m.Digest, err
}
func VerifyFrontendResult(r FrontendReview, after Observation, app AppState, frontend FrontendState, sourceDigest string) error {
	if r.CheckConfiguredUpload() != nil || after.Validate() != nil || !after.Exists || after.Revision <= r.Observation.Revision || after.Publication != r.Observation.Publication || app.SourceRevision != after.Revision || app.TreatyListed != r.Application.TreatyListed || sourceDigest != r.UnchangedSourceDigest || VerifyFrontendMetadata(r.Ship, r.Frontend.Declared.Base, r.ExpectedGlobHash, app, frontend) != nil {
		return errors.New("frontend result remains unverified")
	}
	if app.TreatyListed && app.TreatyHash != after.ClayHash {
		return errors.New("Treaty has not observed the frontend source update")
	}
	return nil
}

func (r FrontendRecord) CheckReceipt(receipt Receipt) error {
	if r.Validate() != nil || receipt.Validate() != nil || receipt.Intent != r.Intent.Intent || receipt.Actor != r.Actor {
		return errors.New("frontend journal binding mismatch")
	}
	for _, step := range receipt.Steps {
		if step.State == "observed" {
			if VerifyFrontendResult(r.Review, *step.Observation, *step.Application, *step.Frontend, step.FrontendSourceDigest) != nil {
				return errors.New("frontend result differs from reviewed artifact")
			}
		}
	}
	return nil
}
