// Package runtimequalify implements a one-shot DEVELOPMENT checkpoint using a
// disposable fake zod. It is not a production runner or an identity provisioner.
package runtimequalify

// Result is also the private report.json schema. RuntimeQualified is always
// false: reachability, local restart and offline pack do not qualify a runtime.
type Result struct {
	RunID              string            `json:"run_id"`
	Directory          string            `json:"directory"`
	ReportPath         string            `json:"report_path"`
	CheckpointPassed   bool              `json:"checkpoint_passed"`
	RuntimeQualified   bool              `json:"runtime_qualified"`
	CleanupComplete    bool              `json:"cleanup_complete"`
	Phases             []Phase           `json:"phases"`
	Containers         []ContainerRecord `json:"containers"`
	Image              string            `json:"image"`
	ImageID            string            `json:"image_id"`
	RepoDigest         string            `json:"repo_digest"`
	Platform           string            `json:"platform"`
	EngineVersion      string            `json:"engine_version"`
	VereVersion        string            `json:"vere_version"`
	PillURL            string            `json:"pill_url"`
	PillSHA256         string            `json:"pill_sha256"`
	PillBytes          int64             `json:"pill_bytes"`
	PillPath           string            `json:"pill_path"`
	MemAvailableBytes  uint64            `json:"mem_available_bytes"`
	DiskAvailableBytes uint64            `json:"disk_available_bytes"`
	LimitsObserved     bool              `json:"limits_observed"`
	PierMetadataSeen   bool              `json:"pier_metadata_seen"`
	Failure            string            `json:"failure,omitempty"`
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
	Name                 string `json:"name"`
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	Removed              bool   `json:"removed"`
	CreateDispatched     bool   `json:"create_dispatched"`
	CreateUncertain      bool   `json:"create_uncertain"`
	AliveObserved        bool   `json:"alive_observed"`
	PrivateHTTPReachable bool   `json:"private_http_reachable"`
}
