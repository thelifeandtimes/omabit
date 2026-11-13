package protocol

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"time"
)

const DeskSettingsFeature = "desk-workflow-settings-v1"
const DeskSettingsPath = "/v1/desk-workflow-settings"

type DeskSetting struct {
	Version  int              `json:"version"`
	RunnerID string           `json:"runner_id"`
	Mode     string           `json:"mode"`
	ReadAt   time.Time        `json:"read_at"`
	Setting  deskwork.Setting `json:"setting"`
}
