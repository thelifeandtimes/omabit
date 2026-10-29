package protocol

import (
	"encoding/hex"
	"strings"
)

// Explicit project/member lifecycle actions. Generic Docker and coordinated
// recovery effects are not accepted by this request.
type ProjectActionRequest struct {
	ContainerID                  string `json:"container_id,omitempty"`
	Ship                         string `json:"ship,omitempty"`
	HTTPPort                     uint16 `json:"http_port,omitempty"`
	AfterOperationID             string `json:"after_operation_id,omitempty"`
	OriginalReadinessOperationID string `json:"original_readiness_operation_id,omitempty"`
	Kind                         string `json:"kind"`
	InstanceID                   string `json:"instance_id,omitempty"`
	RequestID                    string `json:"request_id"`
	ExpectedRevision             uint64 `json:"expected_revision"`
	ReviewDigest                 string `json:"review_digest"`
	ConfirmAction                bool   `json:"confirm_action"`
}
type ProjectActionResult struct {
	SchemaVersion int       `json:"schema_version"`
	RunnerID      string    `json:"runner_id"`
	Mode          string    `json:"mode"`
	ReviewDigest  string    `json:"review_digest"`
	Operation     Operation `json:"operation"`
}

func ProjectActionPath(id string) string { return "/v1/projects/" + id + "/operations" }
func ProjectActionID(path string) (string, bool) {
	const prefix, suffix = "/v1/projects/", "/operations"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return id, ValidID(id)
}

// ProjectAction describes the capability, durable kind and member requirement.
func ProjectAction(kind string) (feature, operation string, member bool) {
	switch kind {
	case "member-continue":
		return "project-add-member-v1", "add-project-member", true
	case "add-member":
		return "project-add-member-v1", "add-project-member", true
	case "member-start":
		return "project-add-member-v1", "start-project-ship", true
	case "member-stop":
		return "project-add-member-v1", "stop-project-ship", true
	case "prepare":
		return "project-prepare-v1", "prepare-project", false
	case "start-anchor":
		return "project-anchor-start-v1", "start-project-anchor", false
	case "prepare-storage":
		return "project-member-storage-v1", "prepare-project-member-storage", true
	case "launch":
		return "project-member-launch-v1", "launch-project-member", true
	case "observe":
		return "project-member-observe-v1", "observe-project-member", true

	case "stop":
		return "project-member-stop-v1", "stop-project-member", true
	case "observe-exit":
		return "project-member-exit-v1", "observe-project-member-stop-exit", true
	case "retire":
		return "project-member-retire-v1", "retire-project-member", true
	case "resume":
		return "project-member-resume-v1", "resume-project-member", true
	case "observe-resumed":
		return "project-member-resumed-observe-v1", "observe-resumed-project-member", true
	case "observe-current":
		return "project-member-current-observe-v1", "observe-current-project-member", true
	default:
		return "", "", false
	}
}

// The prerequisite is a specific prior receipt, never a caller-supplied container
// or path. Original readiness is required only when observing a current creator.
func ValidProjectActionRequest(req ProjectActionRequest) bool {
	feature, _, member := ProjectAction(req.Kind)
	_, err := hex.DecodeString(req.ReviewDigest)
	if feature == "" || !req.ConfirmAction || !ValidID(req.RequestID) || req.ExpectedRevision == 0 || len(req.ReviewDigest) != 64 || err != nil || member && !ValidID(req.InstanceID) || !member && req.InstanceID != "" {
		return false
	}
	if req.Kind == "add-member" {
		if len(req.Ship) != 4 || !ValidFakeShip(req.Ship) || req.HTTPPort < 1024 {
			return false
		}
	} else if req.Ship != "" || req.HTTPPort != 0 {
		return false
	}
	if req.Kind == "member-start" || req.Kind == "member-stop" {
		if len(req.ContainerID) != 64 {
			return false
		}
		if _, err := hex.DecodeString(req.ContainerID); err != nil {
			return false
		}
	} else if req.ContainerID != "" {
		return false
	}
	after := false
	switch req.Kind {
	case "member-continue", "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current":
		after = true
	}
	if after && !ValidID(req.AfterOperationID) || !after && req.AfterOperationID != "" {
		return false
	}
	if req.Kind == "observe-current" {
		return ValidID(req.OriginalReadinessOperationID)
	}
	return req.OriginalReadinessOperationID == ""
}
