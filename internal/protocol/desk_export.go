package protocol

import (
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

const DeskSourceExportFeature = "desk-source-export-v1"
const DeskSourceExportPath = "/v1/desk-source-export"

type DeskSourceExportRequest struct {
	Selection DeskSelection        `json:"selection"`
	Baseline  deskwork.Observation `json:"baseline"`
	Paths     []string             `json:"paths"`
}

func (q DeskSourceExportRequest) Validate() error {
	if q.Selection.Validate() != nil || q.Baseline.Validate() != nil || !q.Baseline.Exists || deskwork.ValidateExportPaths(q.Paths) != nil {
		return errors.New("invalid exact source export selection")
	}
	return nil
}

type DeskSourceExport struct {
	Version  int                     `json:"version"`
	RunnerID string                  `json:"runner_id"`
	Mode     string                  `json:"mode"`
	Request  DeskSourceExportRequest `json:"request"`
	Files    []deskwork.ExportedFile `json:"files"`
	ReadAt   time.Time               `json:"read_at"`
}

func (v DeskSourceExport) Validate() error {
	if v.Version != 1 || !ValidID(v.RunnerID) || !ValidID(v.Mode) || v.Request.Validate() != nil || v.ReadAt.IsZero() || deskwork.ValidateExportedFiles(v.Request.Paths, v.Files) != nil {
		return errors.New("invalid source export response")
	}
	return nil
}
