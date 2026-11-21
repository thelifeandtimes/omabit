package protocol

const RunnerBuildIdentityFeature = "runner-build-identity-v1"

// ExecutableBuild describes the running executable, independently of desktop
// manifests. Omitted values are unknown, never inferred from a client version.
// This authenticated self-report is not a substitute for artifact verification.
type ExecutableBuild struct {
	SourceRevision string `json:"source_revision,omitempty"`
	Modified       *bool  `json:"modified,omitempty"`
	GoVersion      string `json:"go_version,omitempty"`
	ModuleVersion  string `json:"module_version,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
}
