package protocol

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"time"
)

const AppPolicyFeature = "app-development-policy-v2"
const AppPolicyPath = "/v1/app-development-policy"

type AppPolicyTarget struct {
	Target apppolicy.Target `json:"target"`
	Label  string           `json:"label"`
	Ship   string           `json:"ship"`
	Ready  bool             `json:"ready"`
}
type AppPolicyStatus struct {
	Version  int               `json:"version"`
	RunnerID string            `json:"runner_id"`
	Mode     string            `json:"mode"`
	ReadAt   time.Time         `json:"read_at"`
	Policy   apppolicy.Policy  `json:"policy"`
	Targets  []AppPolicyTarget `json:"targets"`
	// Reserved false for v1 status readers. Typed operation capabilities are
	// advertised separately by /hello (for example desk-create-v1).
	DeskOperations bool `json:"desk_operations"`
}
