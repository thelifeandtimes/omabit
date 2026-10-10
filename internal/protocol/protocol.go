// Package protocol defines the versioned runner RPC wire format.
package protocol

import (
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

const (
	Version = 1
	Header  = "Urbit-Runner-Protocol"
)

type Hello struct {
	ProtocolVersion int      `json:"protocol_version"`
	RunnerID        string   `json:"runner_id"`
	Mode            string   `json:"mode"`
	Features        []string `json:"features"`
	Actor           string   `json:"actor,omitempty"`
	AccessRole      string   `json:"access_role,omitempty"`
}

type Configuration struct {
	HTTPPort uint16 `json:"http_port"`
	AmesPort uint16 `json:"ames_port"`
	Loom     uint8  `json:"loom"`
}

type Instance struct {
	Bootstrap         *Bootstrap         `json:"bootstrap,omitempty"`
	Handoff           *Handoff           `json:"handoff,omitempty"`
	ID                string             `json:"id"`
	Label             string             `json:"label"`
	Config            Configuration      `json:"config"`
	ConfigRevision    uint64             `json:"config_revision"`
	AppliedRevision   uint64             `json:"applied_revision"`
	DesiredState      string             `json:"desired_state"`
	ObservedState     string             `json:"observed_state"`
	ActiveOperationID string             `json:"active_operation_id,omitempty"`
	Mode              string             `json:"mode,omitempty"`
	Ship              string             `json:"ship,omitempty"`
	PierName          string             `json:"pier_name,omitempty"`
	PierPath          string             `json:"pier_path,omitempty"`
	AmesPublished     bool               `json:"ames_published,omitempty"`
	IdentityPending   bool               `json:"identity_pending,omitempty"`
	Endpoint          string             `json:"endpoint,omitempty"`
	Health            string             `json:"health,omitempty"`
	RecoveryRequired  bool               `json:"recovery_required,omitempty"`
	ObservedAt        *time.Time         `json:"observed_at,omitempty"`
	Maintenance       *MaintenanceReport `json:"maintenance,omitempty"`
	Resources         *ResourceMetrics   `json:"resources,omitempty"`
	Epochs            *epochs.Inventory  `json:"epochs,omitempty"`
	RollNeedsRuntime  bool               `json:"roll_requires_run,omitempty"`
	Pruning           *PruningReport     `json:"pruning,omitempty"`
	Import            *ImportReport      `json:"import,omitempty"`
	Schedule          *routines.Policy   `json:"schedule,omitempty"`
	Routine           *routines.Run      `json:"routine,omitempty"`
	EpochHistory      []EpochCreation    `json:"epoch_history,omitempty"`
	LastRoll          *MaintenanceReport `json:"last_roll,omitempty"`
	RuntimeStatus     *RuntimeStatus     `json:"runtime_status,omitempty"`
}

// EpochCreation is journal evidence of a completed roll, not filesystem birth
// time or a fresh inventory authorizing history deletion.
type EpochCreation struct {
	Name     string    `json:"name"`
	RolledAt time.Time `json:"rolled_at"`
}

type PruningReport struct {
	Cutoff          string     `json:"cutoff"`
	InventoryDigest string     `json:"inventory_digest"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	RemovedFiles    int        `json:"removed_files"`
	ExpectedFiles   int        `json:"expected_files"`
	EpochRemoved    bool       `json:"epoch_removed"`
	SelectedEpochs  []string   `json:"selected_epochs,omitempty"`
	RemovedEpochs   []string   `json:"removed_epochs,omitempty"`
	RetainedEpochs  []string   `json:"retained_epochs,omitempty"`
	Uncertain       bool       `json:"uncertain"`
}

// Numeric pointers distinguish unavailable readings from genuine zero values.
// Fresh/stale describe this measurement, independently of runtime health.
type ResourceMetrics struct {
	Status             string     `json:"status"`
	Source             string     `json:"source"`
	ContainerID        string     `json:"container_id,omitempty"`
	ObservedAt         *time.Time `json:"observed_at,omitempty"`
	MemoryCurrentBytes *uint64    `json:"memory_current_bytes,omitempty"`
	MemoryLimitBytes   *uint64    `json:"memory_limit_bytes,omitempty"`
	PIDsCurrent        *uint64    `json:"pids_current,omitempty"`
	CPUUsagePercent    *float64   `json:"cpu_usage_percent,omitempty"`
	CPULimitCores      *float64   `json:"cpu_limit_cores,omitempty"`
}

type MaintenanceReport struct {
	Kind               string     `json:"kind"`
	BeforePierBytes    int64      `json:"before_pier_bytes"`
	AfterPierBytes     int64      `json:"after_pier_bytes"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	MemAvailableBytes  uint64     `json:"mem_available_bytes,omitempty"`
	DiskAvailableBytes uint64     `json:"disk_available_bytes,omitempty"`
	EpochsBefore       []string   `json:"epochs_before,omitempty"`
	EpochsAfter        []string   `json:"epochs_after,omitempty"`
	CreatedEpoch       string     `json:"created_epoch,omitempty"`
}

type Operation struct {
	ID             string     `json:"id"`
	RequestID      string     `json:"request_id"`
	InstanceID     string     `json:"instance_id"`
	Kind           string     `json:"kind"`
	Actor          string     `json:"actor"`
	Status         string     `json:"status"`
	ConfigRevision uint64     `json:"config_revision"`
	CreatedAt      time.Time  `json:"created_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	Message        string     `json:"message,omitempty"`
}

type CreateFakeRequest struct {
	Bootstrap *Bootstrap    `json:"bootstrap,omitempty"`
	ID        string        `json:"id"`
	Label     string        `json:"label"`
	Config    Configuration `json:"config"`
	RequestID string        `json:"request_id"`
	FakeShip  string        `json:"fake_ship,omitempty"`
	PierName  string        `json:"pier_name,omitempty"`
	PierPath  string        `json:"pier_path,omitempty"`
}

type CreateCometRequest struct {
	CreateFakeRequest
	ConfirmNetwork bool `json:"confirm_network"`
}

type ImportSource struct {
	RunID         string    `json:"run_id"`
	ContentDigest string    `json:"content_digest"`
	Files         int       `json:"files"`
	Bytes         int64     `json:"bytes"`
	Ship          string    `json:"ship"`
	QualifiedAt   time.Time `json:"qualified_at"`
	ClaimedBy     string    `json:"claimed_by,omitempty"`
}

type ImportCatalogue struct {
	RunnerID string         `json:"runner_id"`
	Sources  []ImportSource `json:"sources"`
}

type RoutineResponse struct {
	RunnerID   string           `json:"runner_id"`
	InstanceID string           `json:"instance_id"`
	Policy     *routines.Policy `json:"policy"`
}

type ImportFakeRequest struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	Config         Configuration `json:"config"`
	RequestID      string        `json:"request_id"`
	SourceRunID    string        `json:"source_run_id"`
	ExpectedDigest string        `json:"expected_digest"`
	ConfirmStopped bool          `json:"confirm_stopped"`
}

type ImportReport struct {
	SourceRunID   string     `json:"source_run_id"`
	ContentDigest string     `json:"content_digest"`
	CopiedFiles   int        `json:"copied_files"`
	CopiedBytes   int64      `json:"copied_bytes"`
	ExpectedFiles int        `json:"expected_files"`
	ExpectedBytes int64      `json:"expected_bytes"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	Uncertain     bool       `json:"uncertain"`
}

func ValidID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range []byte(id) {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

type Incident struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instance_id"`
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Active     bool      `json:"active"`
	ObservedAt time.Time `json:"observed_at"`
}

type Snapshot struct {
	RunnerID   string               `json:"runner_id"`
	Mode       string               `json:"mode"`
	Instances  []Instance           `json:"instances"`
	Operations []Operation          `json:"operations"`
	Incidents  []Incident           `json:"incidents"`
	Piers      *pierstore.Catalogue `json:"piers,omitempty"`
}

type ConfigureRequest struct {
	ExpectedRevision uint64        `json:"expected_revision"`
	Config           Configuration `json:"config"`
}

type OperationRequest struct {
	RequestID          string `json:"request_id"`
	Kind               string `json:"kind"`
	ExpectedRevision   uint64 `json:"expected_revision"`
	EpochCutoff        string `json:"epoch_cutoff,omitempty"`
	InventoryDigest    string `json:"inventory_digest,omitempty"`
	ConfirmHistoryLoss bool   `json:"confirm_history_loss,omitempty"`
	PierPath           string `json:"pier_path,omitempty"`
}

type EnrollPierRequest struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Ship             string `json:"ship"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint"`
	HTTPPort         uint16 `json:"http_port"`
	RequestID        string `json:"request_id"`
	ConfirmStopped   bool   `json:"confirm_stopped"`
	Relink           bool   `json:"relink,omitempty"`
	ExpectedRevision uint64 `json:"expected_revision,omitempty"`
}

type LoginCodeRequest struct {
	ExpectedRevision uint64 `json:"expected_revision"`
	ContainerID      string `json:"container_id"`
}

// LoginCode is an ephemeral response. Never include it in snapshots or journals.
type LoginCode struct {
	RunnerID       string `json:"runner_id"`
	InstanceID     string `json:"instance_id"`
	ConfigRevision uint64 `json:"config_revision"`
	ContainerID    string `json:"container_id"`
	Code           string `json:"code"`
}

func ValidLoginCode(code string) bool {
	if len(code) != 27 {
		return false
	}
	for i, c := range []byte(code) {
		if i == 6 || i == 13 || i == 20 {
			if c != '-' {
				return false
			}
		} else if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

type APIError struct {
	Code            string  `json:"code"`
	Message         string  `json:"message"`
	CurrentRevision *uint64 `json:"current_revision,omitempty"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}
