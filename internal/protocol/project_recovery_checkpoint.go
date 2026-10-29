package protocol

import (
	"strings"
	"time"
)

const ProjectRecoveryCheckpointFeature = "project-recovery-checkpoint-v1"

// Saved checkpoint hints are neither current runtime observations nor permission
// to repeat effects. A separate explicit request must match these exact digests.
type ProjectRecoveryCheckpoint struct {
	SchemaVersion      int                     `json:"schema_version"`
	RunnerID           string                  `json:"runner_id"`
	Mode               string                  `json:"mode"`
	ReadAt             time.Time               `json:"read_at"`
	Basis              string                  `json:"basis"`
	RuntimeObservation string                  `json:"runtime_observation"`
	ActionAuthority    string                  `json:"action_authority"`
	Project            ProjectReservation      `json:"project"`
	Operation          ProjectOperationSummary `json:"operation"`
	HistoryDigest      string                  `json:"history_digest"`
	CheckpointDigest   string                  `json:"checkpoint_digest"`
	Sequence           uint64                  `json:"sequence"`
	Phase              string                  `json:"phase"`
	Next               ProjectRecoveryHint     `json:"next"`
}
type ProjectRecoveryHint struct {
	Kind       string `json:"kind"`
	TargetKind string `json:"target_kind,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
}

func ProjectRecoveryCheckpointPath(project, operation string) string {
	return "/v1/projects/" + project + "/recovery/" + operation
}
func ProjectRecoveryCheckpointIDs(path string) (project, operation string, ok bool) {
	const prefix = "/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) != 3 || parts[1] != "recovery" || !ValidID(parts[0]) || !ValidID(parts[2]) {
		return "", "", false
	}
	return parts[0], parts[2], true
}
