package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

const ProjectStopRecoveryFeature = "project-member-stop-recovery-v1"
const ProjectStopObservationFeature = "project-member-stop-observation-v1"
const ProjectStopReviewMaxAge = 5 * time.Minute

// Exact private bindings are operator-only review data, not fleet summaries or
// shareable diagnostics. Observation never signals, starts or repairs a runtime.
type ProjectStopObservation struct {
	ContainerID     string             `json:"container_id"`
	ContractDigest  string             `json:"contract_digest"`
	EngineID        string             `json:"engine_id"`
	AnchorID        string             `json:"anchor_id"`
	BridgeID        string             `json:"bridge_id"`
	BootID          string             `json:"boot_id"`
	NamespaceDevice uint64             `json:"namespace_device"`
	NamespaceInode  uint64             `json:"namespace_inode"`
	Storage         pierstore.Location `json:"storage"`
	StartedAt       time.Time          `json:"started_at"`
	FinishedAt      time.Time          `json:"finished_at"`
	ExitCode        int                `json:"exit_code"`
	OOMKilled       bool               `json:"oom_killed"`
	Running         bool               `json:"running"`
	ObservedAt      time.Time          `json:"observed_at"`
}

type ProjectStopReview struct {
	Version               int                    `json:"version"`
	RunnerID              string                 `json:"runner_id"`
	Mode                  string                 `json:"mode"`
	Actor                 string                 `json:"actor"`
	ProjectID             string                 `json:"project_id"`
	ProjectRevision       uint64                 `json:"project_revision"`
	ProjectGeneration     uint64                 `json:"project_generation"`
	InstanceID            string                 `json:"instance_id"`
	ContainerGeneration   uint64                 `json:"container_generation"`
	OriginalOperationID   string                 `json:"original_operation_id"`
	OperationDigest       string                 `json:"operation_digest"`
	OriginalRequestDigest string                 `json:"original_request_digest"`
	LedgerDigest          string                 `json:"ledger_digest"`
	EvidenceBasis         string                 `json:"evidence_basis"`
	Observation           ProjectStopObservation `json:"observation"`
	ReviewDigest          string                 `json:"review_digest"`
}

type ProjectMemberDiagnosis struct {
	SchemaVersion      int                     `json:"schema_version"`
	RunnerID           string                  `json:"runner_id"`
	Mode               string                  `json:"mode"`
	Actor              string                  `json:"actor"`
	ProjectID          string                  `json:"project_id"`
	InstanceID         string                  `json:"instance_id"`
	Operation          Operation               `json:"operation"`
	Phase              string                  `json:"phase"`
	Fence              string                  `json:"fence"`
	ReasonCode         string                  `json:"reason_code"`
	Message            string                  `json:"message"`
	RuntimeObservation string                  `json:"runtime_observation"`
	ReadAt             time.Time               `json:"read_at"`
	Observation        *ProjectStopObservation `json:"observation,omitempty"`
	Review             *ProjectStopReview      `json:"review,omitempty"`
	ResolvedBy         string                  `json:"resolved_by,omitempty"`
}

type ProjectStopReconcileRequest struct {
	RequestID     string            `json:"request_id"`
	Review        ProjectStopReview `json:"review"`
	ConfirmAction bool              `json:"confirm_action"`
}

type ProjectStopReconcileResult struct {
	SchemaVersion       int       `json:"schema_version"`
	RunnerID            string    `json:"runner_id"`
	Mode                string    `json:"mode"`
	OriginalOperationID string    `json:"original_operation_id"`
	ReviewDigest        string    `json:"review_digest"`
	VerifiedState       string    `json:"verified_state"`
	Operation           Operation `json:"operation"`
}

func (r ProjectStopReview) Digest() string {
	r.ReviewDigest = ""
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (r ProjectStopReview) Validate() error {
	o := r.Observation
	if r.Version != 1 || !ValidID(r.RunnerID) || r.Mode == "" || r.Actor == "" || len(r.Actor) > 512 || !ValidID(r.ProjectID) || !ValidID(r.InstanceID) || !ValidID(r.OriginalOperationID) || r.ProjectRevision == 0 || r.ProjectGeneration == 0 || r.ContainerGeneration == 0 || r.ReviewDigest != r.Digest() || r.EvidenceBasis != "retained-stop-intent" && r.EvidenceBasis != "legacy-admission-window" {
		return errors.New("invalid exact project stop review")
	}
	for _, digest := range []string{r.LedgerDigest, r.OperationDigest, r.OriginalRequestDigest, o.ContainerID, o.AnchorID, o.ContractDigest} {
		if b, err := hex.DecodeString(digest); err != nil || len(b) != 32 {
			return errors.New("invalid project stop review binding")
		}
	}
	if o.EngineID == "" || o.BridgeID == "" || o.BootID == "" || o.NamespaceInode == 0 || !pierstore.ValidPath(o.Storage.Path) || o.Storage.Identity.Inode == 0 || o.Storage.Identity.UrbInode == 0 || o.StartedAt.IsZero() || o.FinishedAt.IsZero() || o.FinishedAt.Before(o.StartedAt) || o.ObservedAt.Before(o.FinishedAt) || o.Running || o.OOMKilled || o.ExitCode != 0 {
		return errors.New("review requires complete clean-exit and pier evidence")
	}
	return nil
}

func ProjectMemberDiagnosisPath(project, member, operation string, probe bool) string {
	suffix := "diagnosis"
	if probe {
		suffix = "review"
	}
	return "/v1/projects/" + project + "/members/" + member + "/operations/" + operation + "/" + suffix
}
func ProjectMemberDiagnosisIDs(path string) (project, member, operation string, probe, ok bool) {
	const prefix = "/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return
	}
	p := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(p) != 6 || p[1] != "members" || p[3] != "operations" || !ValidID(p[0]) || !ValidID(p[2]) || !ValidID(p[4]) || p[5] != "diagnosis" && p[5] != "review" {
		return
	}
	return p[0], p[2], p[4], p[5] == "review", true
}
func ProjectStopReconcilePath(project string) string {
	return "/v1/projects/" + project + "/stop-reconciliations"
}
func ProjectStopReconcileID(path string) (string, bool) {
	const prefix, suffix = "/v1/projects/", "/stop-reconciliations"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return id, ValidID(id)
}
