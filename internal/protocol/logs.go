package protocol

import "time"

const MaxLogBytes = 64 << 10
const MaxLogLines = 200

type RuntimeStatus struct {
	LaunchLoom              *uint8     `json:"launch_loom,omitempty"`
	LaunchNativeControl     *bool      `json:"launch_native_control,omitempty"`
	NativeControlSocket     string     `json:"native_control_socket,omitempty"`
	ImportRecoveryAvailable bool       `json:"import_recovery_available,omitempty"`
	LaunchError             string     `json:"launch_error,omitempty"`
	Stage                   string     `json:"stage,omitempty"`
	Initialized             bool       `json:"initialized"`
	ContainerID             string     `json:"container_id,omitempty"`
	ContainerKind           string     `json:"container_kind,omitempty"`
	Running                 *bool      `json:"running,omitempty"`
	ExitCode                *int       `json:"exit_code,omitempty"`
	OOMKilled               *bool      `json:"oom_killed,omitempty"`
	ObservedAt              *time.Time `json:"observed_at,omitempty"`
	LastOperation           *Operation `json:"last_operation,omitempty"`
	DefaultSnapshotMinutes  uint32     `json:"default_snapshot_minutes"`
	SnapshotMinutes         *uint32    `json:"snapshot_minutes,omitempty"`
}

type LogsRequest struct {
	ExpectedRevision uint64 `json:"expected_revision"`
}

// Logs are opt-in operator data, never embedded in fleet snapshots.
type RuntimeLogs struct {
	RunnerID       string    `json:"runner_id,omitempty"`
	InstanceID     string    `json:"instance_id,omitempty"`
	ConfigRevision uint64    `json:"config_revision,omitempty"`
	ContainerID    string    `json:"container_id,omitempty"`
	Kind           string    `json:"kind,omitempty"`
	Source         string    `json:"source"`
	ObservedAt     time.Time `json:"observed_at"`
	Lines          []string  `json:"lines"`
	Truncated      bool      `json:"truncated"`
}
