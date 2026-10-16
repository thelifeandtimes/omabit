package protocol

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"time"
)

type StorageRootRequest struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	RequestID string `json:"request_id"`
}
type ImportPierRequest struct {
	EnrollPierRequest
	Source   string `json:"source"`
	Identify bool   `json:"identify,omitempty"`
}

type ImportPierPreviewRequest struct {
	Source      string `json:"source"`
	StorageRoot string `json:"storage_root,omitempty"`
}

type PierIdentity struct {
	Ship    string `json:"ship"`
	Network string `json:"network"`
	Kind    string `json:"kind"`
}

type PierImportPreview struct {
	RunnerID string                   `json:"runner_id"`
	Source   pierstore.ExternalSource `json:"source"`
	PierIdentity
	StorageRoot string `json:"storage_root"`
	InstanceID  string `json:"instance_id"`
	Destination string `json:"destination"`
	HTTPPort    uint16 `json:"http_port"`
	InPlace     bool   `json:"in_place"`
}

type ImportEvent struct {
	Stage string    `json:"stage"`
	At    time.Time `json:"at"`
}

type ExternalImportStatus struct {
	CopiedFiles int           `json:"copied_files"`
	CopiedBytes int64         `json:"copied_bytes"`
	Source      string        `json:"source"`
	HoldingPath string        `json:"holding_path,omitempty"`
	Stage       string        `json:"stage"`
	TotalFiles  int           `json:"total_files"`
	TotalBytes  int64         `json:"total_bytes"`
	Events      []ImportEvent `json:"events,omitempty"`
}
