package protocol

// ManagementUpdateFeature is local-root-only. It is deliberately not part of
// the SSH grant allowlist. Tokens are installer-generated 256-bit lowercase hex.
const ManagementUpdateFeature = "management-update-quiesce-v1"
const ManagementUpdatePath = "/v1/management-update"

type ManagementUpdateRequest struct {
	Action         string `json:"action"` // acquire or release; GET reads status
	RunnerID       string `json:"runner_id"`
	Token          string `json:"token"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"` // required on release
}

type ManagementUpdateStatus struct {
	RunnerID         string `json:"runner_id"`
	Mode             string `json:"mode"`
	Quiesced         bool   `json:"quiesced"`
	Token            string `json:"token,omitempty"`
	StateSchema      int    `json:"state_schema"`
	MinStateSchema   int    `json:"min_state_schema"`
	PID              int    `json:"pid"`
	ExecutableSHA256 string `json:"executable_sha256"`
}
