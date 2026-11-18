package protocol

import (
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

// Receipt access is independent of live adapter qualification or current app
// policy. It grants no source, native HTTP or phase-dispatch authority.
const GrubberyAppReceiptsFeature = "grubbery-app-receipts-v1"
const GrubberyAppOperationsPath = "/v1/grubbery-app-operations"

type GrubberyAppOperation struct {
	Version  int             `json:"version"`
	RunnerID string          `json:"runner_id"`
	Mode     string          `json:"mode"`
	Record   grubbery.Record `json:"record"`
}

func (r GrubberyAppOperation) Validate() error {
	if r.Version != 1 || !ValidID(r.RunnerID) || r.Record.Validate() != nil || r.Record.Plan.Binding.RunnerID != r.RunnerID || r.Record.Plan.Binding.Mode != r.Mode {
		return errors.New("invalid app operation receipt binding")
	}
	return nil
}
