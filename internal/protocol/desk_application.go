package protocol

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const DeskApplicationFeature = "desk-application-status-v1"
const DeskApplicationPath = "/v1/desk-application"

type DeskApplication struct {
	Version          int                  `json:"version"`
	RunnerID         string               `json:"runner_id"`
	Mode             string               `json:"mode"`
	Selection        DeskSelection        `json:"selection"`
	PolicyRevision   uint64               `json:"policy_revision"`
	SettingsRevision uint64               `json:"settings_revision"`
	Observation      deskwork.Observation `json:"observation"`
	Application      deskwork.AppState    `json:"application"`
	ReadAt           time.Time            `json:"read_at"`
}

func (s DeskApplication) Validate() error {
	if s.Version != 1 || !ValidID(s.RunnerID) || !ValidID(s.Mode) || s.Selection.Validate() != nil || s.PolicyRevision > apppolicy.MaxRevision || s.SettingsRevision > apppolicy.MaxRevision || s.ReadAt.IsZero() || s.Observation.Validate() != nil || !s.Observation.Exists || s.Application.Validate() != nil || s.Application.SourceRevision != s.Observation.Revision {
		return errors.New("invalid application stage observation")
	}
	return nil
}
