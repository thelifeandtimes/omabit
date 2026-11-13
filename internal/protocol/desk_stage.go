package protocol

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"time"
)

const DeskStageFeature = "desk-source-staging-v1"
const DeskStagesPath = "/v1/desk-stages"
const DeskStageHeader = "Omabit-Desk-Stage"
const DeskStageTimeout = 10 * time.Minute

type DeskStage struct {
	Version  int                   `json:"version"`
	RunnerID string                `json:"runner_id"`
	Mode     string                `json:"mode"`
	Receipt  deskwork.StageReceipt `json:"receipt"`
	ReadAt   time.Time             `json:"read_at"`
}
type DeskStageDiscardRequest struct {
	SourceDigest string `json:"source_digest"`
}
type DeskStageDiscard struct {
	Version      int    `json:"version"`
	RunnerID     string `json:"runner_id"`
	Mode         string `json:"mode"`
	ID           string `json:"id"`
	SourceDigest string `json:"source_digest"`
	Discarded    bool   `json:"discarded"`
}
