package protocol

import (
	"strings"
	"time"
)

const ProjectRecoverySourceFeature = "project-recovery-source-v1"

// The selected saved history is a review input; it grants no runtime authority.
type ProjectRecoverySource struct {
	SchemaVersion      int                           `json:"schema_version"`
	RunnerID           string                        `json:"runner_id"`
	Mode               string                        `json:"mode"`
	ReadAt             time.Time                     `json:"read_at"`
	Basis              string                        `json:"basis"`
	RuntimeObservation string                        `json:"runtime_observation"`
	ActionAuthority    string                        `json:"action_authority"`
	Project            ProjectReservation            `json:"project"`
	HistoryDigest      string                        `json:"history_digest"`
	Members            []ProjectRecoverySourceMember `json:"members"`
}
type ProjectRecoverySourceMember struct {
	InstanceID                   string `json:"instance_id"`
	CreatorOperationID           string `json:"creator_operation_id"`
	ContainerGeneration          uint64 `json:"container_generation"`
	OriginalReadinessOperationID string `json:"original_readiness_operation_id"`
	CurrentReadinessOperationID  string `json:"current_readiness_operation_id,omitempty"`
}

func ProjectRecoverySourcePath(id string) string { return "/v1/projects/" + id + "/recovery-source" }
func ProjectRecoverySourceID(path string) (string, bool) {
	const prefix, suffix = "/v1/projects/", "/recovery-source"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return id, ValidID(id)
}
