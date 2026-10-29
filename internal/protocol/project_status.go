package protocol

import (
	"strings"
	"time"
)

const ProjectStatusFeature = "project-saved-status-v1"
const ProjectStatusOperationLimit = 128

// ProjectStatus exposes saved history, never live runtime health or action authority.
// Recent operations are in insertion order, newest first; timestamps are not cursors.
type ProjectStatus struct {
	SchemaVersion      int                       `json:"schema_version"`
	RunnerID           string                    `json:"runner_id"`
	Mode               string                    `json:"mode"`
	ReadAt             time.Time                 `json:"read_at"`
	Basis              string                    `json:"basis"`
	RuntimeObservation string                    `json:"runtime_observation"`
	Project            ProjectReservation        `json:"project"`
	Operations         []ProjectOperationSummary `json:"operations"`
	HasOlderOperations bool                      `json:"has_older_operations"`
}

// Deliberately omit runtime IDs, private journal bodies and free-form messages.
type ProjectOperationSummary struct {
	ID             string     `json:"id"`
	RequestID      string     `json:"request_id"`
	InstanceID     string     `json:"instance_id,omitempty"`
	Kind           string     `json:"kind"`
	Actor          string     `json:"actor"`
	Status         string     `json:"status"`
	ConfigRevision uint64     `json:"config_revision"`
	CreatedAt      time.Time  `json:"created_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

func ProjectStatusPath(id string) string { return "/v1/projects/" + id + "/status" }

func ProjectStatusID(path string) (string, bool) {
	const prefix, suffix = "/v1/projects/", "/status"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return id, ValidID(id)
}

// ValidProjectOperationSummary checks public receipt structure without interpreting
// a historical status of "running" as a fresh runtime observation.
func ValidProjectOperationSummary(op ProjectOperationSummary, p ProjectReservation, readAt time.Time) bool {
	if !ValidID(op.ID) || !ValidID(op.RequestID) || op.Actor == "" || len(op.Actor) > 512 ||
		op.ConfigRevision != p.Revision || op.CreatedAt.IsZero() || op.CreatedAt.Before(p.CreatedAt) || op.CreatedAt.After(readAt) {
		return false
	}
	member := false
	switch op.Kind {
	case "register-project", "prepare-project", "start-project-anchor", "recover-project-anchor", "observe-project-recovery", "continue-project-recovery":
	case "add-project-member", "start-project-ship", "stop-project-ship", "prepare-project-member-storage", "launch-project-member", "observe-project-member", "stop-project-member", "observe-project-member-stop-exit", "retire-project-member", "resume-project-member", "observe-resumed-project-member", "observe-current-project-member", "recover-project-member":
		member = true
	default:
		return false
	}
	if member {
		found := false
		for _, m := range append(append([]ProjectMemberReservation{}, p.Members...), p.AdditionalMembers...) {
			if m.InstanceID == op.InstanceID {
				found = true
			}
		}
		if !found {
			return false
		}
	} else if op.InstanceID != "" {
		return false
	}
	switch op.Status {
	case "queued", "running":
		return op.FinishedAt == nil
	case "succeeded", "failed", "interrupted":
		return op.FinishedAt != nil && !op.FinishedAt.Before(op.CreatedAt) && !op.FinishedAt.After(readAt)
	default:
		return false
	}
}
