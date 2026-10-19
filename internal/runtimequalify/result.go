// Package runtimequalify implements disposable development checkpoints and the
// bounded fake-runtime adapter shared by the persistent development runner.
// It is not a production runtime or a real-identity provisioner.
package runtimequalify

import (
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const (
	ProfileLifecycle        = "isolated-lifecycle"
	ProfileHTTP             = "host-http"
	ProfileMeld             = "offline-meld"
	ProfileEpochs           = "epoch-lifecycle"
	ProfilePrune            = "epoch-prune-recovery"
	ProfilePrefix           = "epoch-prefix-recovery"
	ProfileImport           = "stopped-pier-copy"
	ProfileComet            = "fresh-comet-bootstrap"
	ProfileCometMaintenance = "comet-offline-maintenance"
	ProfileCometEpochs      = "comet-epoch-lifecycle"
	ProfileCometPrefix      = "comet-prefix-recovery"
)

// Result is also the private report.json schema. RuntimeQualified is always
// false: these bounded checkpoints do not qualify a production runtime.
// EngineVersion records original provisioning/qualification evidence and is
// never rewritten when a newer daemon observes or resumes this runtime.
type Result struct {
	Stage              string                `json:"stage,omitempty"`
	RetainedLogs       *protocol.RuntimeLogs `json:"retained_logs,omitempty"`
	Profile            string                `json:"profile,omitempty"`
	RunID              string                `json:"run_id"`
	Directory          string                `json:"directory"`
	ReportPath         string                `json:"report_path"`
	CheckpointPassed   bool                  `json:"checkpoint_passed"`
	RuntimeQualified   bool                  `json:"runtime_qualified"`
	CleanupComplete    bool                  `json:"cleanup_complete"`
	Phases             []Phase               `json:"phases"`
	Containers         []ContainerRecord     `json:"containers"`
	Network            *NetworkRecord        `json:"network,omitempty"`
	Image              string                `json:"image"`
	ImageID            string                `json:"image_id"`
	RepoDigest         string                `json:"repo_digest"`
	Platform           string                `json:"platform"`
	EngineVersion      string                `json:"engine_version"`
	VereVersion        string                `json:"vere_version"`
	PillURL            string                `json:"pill_url"`
	PillSHA256         string                `json:"pill_sha256"`
	PillBytes          int64                 `json:"pill_bytes"`
	PillPath           string                `json:"pill_path"`
	MemAvailableBytes  uint64                `json:"mem_available_bytes"`
	DiskAvailableBytes uint64                `json:"disk_available_bytes"`
	LimitsObserved     bool                  `json:"limits_observed"`
	PierMetadataSeen   bool                  `json:"pier_metadata_seen"`
	Meld               *MeldRecord           `json:"meld,omitempty"`
	EpochLifecycle     *EpochLifecycleRecord `json:"epoch_lifecycle,omitempty"`
	Prune              *PruneRecord          `json:"prune,omitempty"`
	Recovery           *EpochRecoveryRecord  `json:"epoch_recovery,omitempty"`
	PrefixPrunes       []*PrefixPruneRecord  `json:"prefix_prunes,omitempty"`
	PierImport         *PierImportRecord     `json:"pier_import,omitempty"`
	Comet              *CometRecord          `json:"comet,omitempty"`
	CometUtilities     []CometUtilityRecord  `json:"comet_utilities,omitempty"`
	CometFinalEpochs   *epochs.Inventory     `json:"comet_final_epochs,omitempty"`
	Failure            string                `json:"failure,omitempty"`
}

type CometUtilityRecord struct {
	Kind string `json:"kind"`
	MeldRecord
	RestartVerified bool `json:"restart_verified"`
}

type CometRecord struct {
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	Ship            string     `json:"ship,omitempty"`
	BootCompleted   bool       `json:"boot_completed"`
	ResumeCompleted bool       `json:"resume_completed"`
	Uncertain       bool       `json:"uncertain"`
}

type PierImportRecord struct {
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	SourceMoved       bool       `json:"source_moved"`
	CopiedFiles       int        `json:"copied_files"`
	CopiedBytes       int64      `json:"copied_bytes"`
	ContentDigest     string     `json:"content_digest"`
	CopyVerified      bool       `json:"copy_verified"`
	FirstResume       bool       `json:"first_resume"`
	ReplacementResume bool       `json:"replacement_resume"`
	SourceUnchanged   bool       `json:"source_unchanged"`
	Uncertain         bool       `json:"uncertain"`
}

type EpochDeletion struct {
	Name         string      `json:"name"`
	Directory    PruneFile   `json:"directory"`
	Files        []PruneFile `json:"files"`
	RemovedFiles []string    `json:"removed_files"`
	Removed      bool        `json:"removed"`
}

type PrefixPruneRecord struct {
	StartedAt   time.Time            `json:"started_at"`
	FinishedAt  *time.Time           `json:"finished_at,omitempty"`
	Before      *epochs.Inventory    `json:"before"`
	Cutoff      string               `json:"cutoff"`
	Targets     []EpochDeletion      `json:"targets"`
	After       *epochs.Inventory    `json:"after,omitempty"`
	PostRestart *epochs.Inventory    `json:"post_restart,omitempty"`
	Recovery    *EpochRecoveryRecord `json:"recovery,omitempty"`
	Uncertain   bool                 `json:"uncertain"`
}

type PruneFile struct {
	Name     string `json:"name"`
	Device   uint64 `json:"device"`
	Inode    uint64 `json:"inode"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Changed  int64  `json:"changed"`
}

type PruneRecord struct {
	StartedAt    time.Time         `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at,omitempty"`
	Before       *epochs.Inventory `json:"before"`
	Cutoff       string            `json:"cutoff"`
	Directory    PruneFile         `json:"directory"`
	Files        []PruneFile       `json:"files"`
	RemovedFiles []string          `json:"removed_files"`
	EpochRemoved bool              `json:"epoch_removed"`
	After        *epochs.Inventory `json:"after,omitempty"`
	PostRestart  *epochs.Inventory `json:"post_restart,omitempty"`
	Uncertain    bool              `json:"uncertain"`
}

type EpochRecoveryRecord struct {
	StartedAt                time.Time         `json:"started_at"`
	FinishedAt               *time.Time        `json:"finished_at,omitempty"`
	Before                   *epochs.Inventory `json:"before"`
	Checkpoint               PruneFile         `json:"checkpoint"`
	CheckpointRemoved        bool              `json:"checkpoint_removed"`
	RecoveredCheckpointBytes int64             `json:"recovered_checkpoint_bytes"`
	MemAvailableBytes        uint64            `json:"mem_available_bytes"`
	DiskAvailableBytes       uint64            `json:"disk_available_bytes"`
	After                    *epochs.Inventory `json:"after,omitempty"`
	PostRestart              *epochs.Inventory `json:"post_restart,omitempty"`
	Uncertain                bool              `json:"uncertain"`
}

type EpochLifecycleRecord struct {
	Initial *epochs.Inventory `json:"initial"`
	Rolls   []EpochRollRecord `json:"rolls"`
	Final   *epochs.Inventory `json:"final,omitempty"`
	Preview *epochs.Preview   `json:"preview,omitempty"`
}

type EpochRollRecord struct {
	Kind               string            `json:"kind"`
	StartedAt          time.Time         `json:"started_at"`
	FinishedAt         *time.Time        `json:"finished_at,omitempty"`
	MemAvailableBytes  uint64            `json:"mem_available_bytes"`
	DiskAvailableBytes uint64            `json:"disk_available_bytes"`
	Before             *epochs.Inventory `json:"before"`
	After              *epochs.Inventory `json:"after,omitempty"`
	PostRestart        *epochs.Inventory `json:"post_restart,omitempty"`
	Uncertain          bool              `json:"uncertain"`
}

// MeldRecord separates resource cleanup from the outcome of a pier mutation.
// Uncertain is persisted before dispatch and is not cleared by container cleanup.
// Byte counts are logical regular-file lengths, not allocated space or loom usage.
type MeldRecord struct {
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	BeforePierBytes    int64      `json:"before_pier_bytes"`
	AfterPierBytes     *int64     `json:"after_pier_bytes,omitempty"`
	MemAvailableBytes  uint64     `json:"mem_available_bytes"`
	DiskAvailableBytes uint64     `json:"disk_available_bytes"`
	Uncertain          bool       `json:"uncertain"`
}

type Phase struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// A record is persisted BEFORE create. An empty ID may mean an uncertain API
// response; the exact intended name must then be inspected and authenticated.
// Dispatched intent remains uncertain until authenticated inspection, regardless
// of empty inventories. A canceled client does not cancel a daemon-side create.
type ContainerRecord struct {
	// Empty on historical records; never infer it from the current daemon.
	EngineVersion        string          `json:"engine_version,omitempty"`
	Loom                 uint8           `json:"loom,omitempty"`
	NativeControl        bool            `json:"native_control,omitempty"`
	Name                 string          `json:"name"`
	ID                   string          `json:"id"`
	Kind                 string          `json:"kind"`
	SnapshotMinutes      uint32          `json:"snapshot_minutes,omitempty"`
	Removed              bool            `json:"removed"`
	CreateDispatched     bool            `json:"create_dispatched"`
	CreateUncertain      bool            `json:"create_uncertain"`
	AliveObserved        bool            `json:"alive_observed"`
	PrivateHTTPReachable bool            `json:"private_http_reachable"`
	HostEndpoint         string          `json:"host_endpoint,omitempty"`
	HostHTTPVerified     bool            `json:"host_http_verified,omitempty"`
	LastState            *ContainerState `json:"last_state,omitempty"`
}

// Only authenticated Docker state, never container logs or response bodies.
// ExitCode alone is not proof of a particular signal; host evidence may be needed.
type ContainerState struct {
	LaunchError string    `json:"launch_error,omitempty"`
	ObservedAt  time.Time `json:"observed_at"`
	Running     bool      `json:"running"`
	Paused      bool      `json:"paused"`
	Restarting  bool      `json:"restarting"`
	Dead        bool      `json:"dead"`
	OOMKilled   bool      `json:"oom_killed"`
	ExitCode    int       `json:"exit_code"`
}

// NetworkRecord is a persisted intent for one dedicated, run-owned bridge. An
// empty inventory cannot resolve an uncertain create, just as for containers.
type NetworkRecord struct {
	Name             string `json:"name"`
	ID               string `json:"id"`
	Removed          bool   `json:"removed"`
	CreateDispatched bool   `json:"create_dispatched"`
	CreateUncertain  bool   `json:"create_uncertain"`
}
