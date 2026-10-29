package protocol

import (
	"encoding/hex"
	"strings"
)

type ProjectRecoveryActionRequest struct {
	RequestID        string  `json:"request_id"`
	ExpectedRevision uint64  `json:"expected_revision"`
	ReviewDigest     string  `json:"review_digest"`
	HistoryDigest    string  `json:"history_digest"`
	ExpectedSequence *uint64 `json:"expected_sequence,omitempty"`
	CheckpointDigest string  `json:"checkpoint_digest,omitempty"`
	ConfirmAction    bool    `json:"confirm_action"`
}
type ProjectRecoveryActionResult struct {
	ProjectActionResult
	RecoveryOperationID string  `json:"recovery_operation_id"`
	HistoryDigest       string  `json:"history_digest"`
	ExpectedSequence    *uint64 `json:"expected_sequence,omitempty"`
	CheckpointDigest    string  `json:"checkpoint_digest,omitempty"`
}

func ProjectRecoveryAction(kind string) (feature, operation string) {
	switch kind {
	case "start":
		return "project-anchor-recovery-v1", "recover-project-anchor"
	case "observe":
		return "project-recovery-observe-v1", "observe-project-recovery"
	case "continue":
		return "project-recovery-continue-v1", "continue-project-recovery"
	}
	return "", ""
}
func ProjectRecoveryActionPath(project, parent, kind string) string {
	base := "/v1/projects/" + project + "/recovery"
	switch kind {
	case "start":
		return base
	case "observe":
		return base + "/" + parent + "/observations"
	case "continue":
		return base + "/" + parent + "/continuations"
	}
	return ""
}
func ProjectRecoveryActionIDs(path string) (project, parent, kind string, ok bool) {
	const prefix = "/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if (len(parts) != 2 && len(parts) != 4) || !ValidID(parts[0]) || parts[1] != "recovery" {
		return
	}
	project = parts[0]
	if len(parts) == 2 {
		return project, "", "start", true
	}
	if !ValidID(parts[2]) {
		return "", "", "", false
	}
	parent = parts[2]
	switch parts[3] {
	case "observations":
		return project, parent, "observe", true
	case "continuations":
		return project, parent, "continue", true
	}
	return "", "", "", false
}
func ValidProjectRecoveryAction(kind, parent string, r ProjectRecoveryActionRequest) bool {
	feature, _ := ProjectRecoveryAction(kind)
	if feature == "" || !r.ConfirmAction || !ValidID(r.RequestID) || r.ExpectedRevision == 0 {
		return false
	}
	for _, digest := range []string{r.ReviewDigest, r.HistoryDigest} {
		if len(digest) != 64 {
			return false
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	if kind == "start" {
		return parent == "" && r.ExpectedSequence == nil && r.CheckpointDigest == ""
	}
	if !ValidID(parent) || r.ExpectedSequence == nil || len(r.CheckpointDigest) != 64 {
		return false
	}
	_, err := hex.DecodeString(r.CheckpointDigest)
	return err == nil
}
