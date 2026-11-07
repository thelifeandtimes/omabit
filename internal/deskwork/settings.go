package deskwork

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
)

// These settings describe the developer's workflow. They contain no permission
// grants, secrets, executable hooks or automatic release scheduler.
type ShipConfig struct {
	Purpose      string `json:"purpose"`
	Instructions string `json:"instructions"`
}
type DeskConfig struct {
	Workflow     string   `json:"workflow"`
	SourceName   string   `json:"source_name"`
	Publication  string   `json:"publication"`
	Repository   string   `json:"repository"`
	Branch       string   `json:"branch"`
	Checks       []string `json:"checks"`
	Instructions string   `json:"instructions"`
}
type Setting struct {
	Version   int              `json:"version"`
	Target    apppolicy.Target `json:"target"`
	Revision  uint64           `json:"revision"`
	Ship      *ShipConfig      `json:"ship,omitempty"`
	Desk      *DeskConfig      `json:"desk,omitempty"`
	Actor     string           `json:"actor,omitempty"`
	UpdatedAt time.Time        `json:"updated_at"`
}
type SettingUpdate struct {
	Target           apppolicy.Target `json:"target"`
	ExpectedRevision uint64           `json:"expected_revision"`
	Ship             *ShipConfig      `json:"ship,omitempty"`
	Desk             *DeskConfig      `json:"desk,omitempty"`
}

func DefaultSetting(target apppolicy.Target) Setting {
	out := Setting{Version: 1, Target: target}
	if target.Desk == "" {
		out.Ship = &ShipConfig{Purpose: "standard"}
	} else {
		out.Desk = &DeskConfig{Workflow: "mounted", SourceName: target.Desk, Publication: "none", Checks: []string{}}
	}
	return out
}
func settingText(s string, limit int, multiline bool) bool {
	if !utf8.ValidString(s) || len(s) > limit {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\t')) {
			return false
		}
	}
	return true
}
func (s Setting) Validate() error {
	if s.Version != 1 || s.Target.Validate() != nil || s.Target.InstanceID == "" || s.Target.ProjectID != "" && s.Target.Generation == 0 || s.Revision > apppolicy.MaxRevision {
		return errors.New("invalid development setting")
	}
	if s.Revision == 0 {
		if s.Actor != "" || !s.UpdatedAt.IsZero() {
			return errors.New("invalid default development setting")
		}
	} else if s.Actor == "" || !settingText(s.Actor, 128, false) || s.UpdatedAt.IsZero() {
		return errors.New("development setting lacks author evidence")
	}
	if s.Target.Desk == "" {
		if s.Ship == nil || s.Desk != nil || !settingText(s.Ship.Instructions, 2048, true) {
			return errors.New("invalid ship development settings")
		}
		switch s.Ship.Purpose {
		case "standard", "development", "distribution":
		default:
			return errors.New("invalid ship purpose")
		}
	} else {
		if s.Desk == nil || s.Ship != nil {
			return errors.New("invalid desk development settings")
		}
		d := s.Desk
		if !settingText(d.Instructions, 2048, true) || !settingText(d.Repository, 256, false) || !settingText(d.Branch, 128, false) || d.Checks == nil || len(d.Checks) > 16 {
			return errors.New("invalid release configuration")
		}
		switch d.Workflow {
		case "mounted":
			if !ValidSourceName(d.SourceName) {
				return errors.New("select a top-level source directory")
			}
		case "urgit":
			if d.SourceName != "" {
				return errors.New("Urgit owns its source binding")
			}
		default:
			return errors.New("unsupported source workflow")
		}
		switch d.Publication {
		case "none", "source", "treaty":
		default:
			return errors.New("unsupported publication method")
		}
		for _, check := range d.Checks {
			if strings.TrimSpace(check) == "" || !settingText(check, 256, false) {
				return errors.New("invalid release check description")
			}
		}
	}
	return nil
}
func (u SettingUpdate) Validate() error {
	if u.ExpectedRevision >= apppolicy.MaxRevision {
		return errors.New("invalid development settings revision")
	}
	return (Setting{Version: 1, Target: u.Target, Ship: u.Ship, Desk: u.Desk}).Validate()
}
