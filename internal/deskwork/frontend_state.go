package deskwork

import "errors"

// FrontendReference separates the advertised artifact from Docket's current
// charge. These are metadata observations, not proof of served asset bytes.
// HTTP URLs are intentionally omitted: an upload never selects an external host.
type FrontendReference struct {
	Mode       string `json:"mode"`
	Base       string `json:"base"`
	Ship       string `json:"ship"`
	GlobHash   string `json:"glob_hash"`
	DocketHash string `json:"docket_hash"`
}

func validFrontendHash(hash string) bool {
	return (Observation{Exists: true, Revision: 1, ClayHash: hash, Publication: "private"}).Validate() == nil
}

func (r FrontendReference) Validate() error {
	switch r.Mode {
	case "none":
		if r.Base == "" && r.Ship == "" && r.GlobHash == "" && r.DocketHash == "" {
			return nil
		}
	case "site":
		if r.Base == "" && r.Ship == "" && r.GlobHash == "" && validFrontendHash(r.DocketHash) {
			return nil
		}
	case "glob-http", "glob-ames":
		if deskName.MatchString(r.Base) && validFrontendHash(r.GlobHash) && validFrontendHash(r.DocketHash) && (r.Mode == "glob-http" && r.Ship == "" || r.Mode == "glob-ames" && validSelection(r.Ship, r.Base)) {
			return nil
		}
	}
	return errors.New("invalid frontend reference")
}

type FrontendState struct {
	SourceRevision  uint64            `json:"source_revision"`
	DocketAvailable bool              `json:"docket_available"`
	Stage           string            `json:"stage"`
	Declared        FrontendReference `json:"declared"`
	Current         FrontendReference `json:"current"`
}

func (s FrontendState) Validate() error {
	if s.SourceRevision == 0 || s.SourceRevision > 1<<53-1 || s.Declared.Validate() != nil || s.Current.Validate() != nil {
		return errors.New("invalid frontend observation")
	}
	switch s.Stage {
	case "unavailable":
		if !s.DocketAvailable && s.Current.Mode == "none" {
			return nil
		}
	case "absent":
		if s.DocketAvailable && s.Current.Mode == "none" {
			return nil
		}
	case "site", "glob", "install", "suspend", "hung":
		if s.DocketAvailable && s.Current.Mode != "none" && (s.Stage != "site" || s.Current.Mode == "site") && (s.Stage != "glob" || s.Current.Mode == "glob-ames" || s.Current.Mode == "glob-http") {
			return nil
		}
	}
	return errors.New("invalid frontend stage")
}

// CheckFrontendUpload permits only the ship's own glob-ames artifact on a local
// installation. Uploading a remote/http glob would create a local override whose
// source hash is not updated for receivers. A pending or failed initial glob can
// be replaced, but a missing/stale charge cannot.
func CheckFrontendUpload(ship string, app AppState, s FrontendState) error {
	if app.Validate() != nil || s.Validate() != nil || app.SourceRevision != s.SourceRevision || app.Zest != "live" || app.SourceShip != "" || !app.DocketAvailable || !s.DocketAvailable || s.Stage == "absent" || s.Stage == "unavailable" || s.Stage == "suspend" || s.Declared != s.Current || s.Declared.Mode != "glob-ames" || s.Declared.Ship != ship || app.DocketHash != s.Declared.DocketHash || app.FrontendDocketHash != s.Current.DocketHash || app.FrontendMode != s.Declared.Mode || app.FrontendState != s.Stage {
		return errors.New("frontend upload requires a current local installation with a ship-hosted glob")
	}
	return nil
}

// VerifyFrontendMetadata is one verification stage, not an HTTP health claim.
// The artifact's own hash must be computed from its complete typed MIME map.
func VerifyFrontendMetadata(ship, base, expectedHash string, app AppState, s FrontendState) error {
	if !validFrontendHash(expectedHash) || CheckFrontendUpload(ship, app, s) != nil || s.Stage != "glob" || s.Declared.Base != base || s.Declared.GlobHash != expectedHash {
		return errors.New("frontend metadata does not match the frozen artifact")
	}
	return nil
}
