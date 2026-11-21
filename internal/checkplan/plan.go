package checkplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const Lifecycle = "runtime-lifecycle-v1"
const HTTP = "runtime-http-v1"

var ErrSelection = errors.New("unknown named checkpoint or report selection")

// Runtime is the existing runtime-plan/http-plan payload. The legacy map
// constructors remain the only source for both legacy output and this view.
type Runtime struct {
	Profile                   string             `json:"profile"`
	Mode                      string             `json:"mode"`
	Account                   string             `json:"account"`
	Home                      string             `json:"home"`
	EngineVersion             string             `json:"engine_version"`
	Image                     string             `json:"image"`
	ManifestDigest            string             `json:"manifest_digest"`
	ConfigDigest              string             `json:"config_digest"`
	Platform                  string             `json:"platform"`
	VereVersion               string             `json:"vere_version"`
	PillDefaultURL            string             `json:"pill_default_url"`
	PillGeneration            string             `json:"pill_generation"`
	Pill                      candidate.Artifact `json:"pill"`
	PillPinBasis              string             `json:"pill_pin_basis"`
	MinimumAvailableRAMBytes  int64              `json:"minimum_available_ram_bytes"`
	MinimumAvailableDiskBytes int64              `json:"minimum_available_disk_bytes"`
	ContainerMemoryBytes      int64              `json:"container_memory_bytes"`
	ContainerSwapBytes        int64              `json:"container_swap_bytes"`
	CPUs                      int                `json:"cpus"`
	PIDs                      int                `json:"pids"`
	LoomExponent              int                `json:"loom_exponent"`
	Network                   string             `json:"network"`
	ControlSocket             string             `json:"control_socket"`
	RuntimeQualified          bool               `json:"runtime_qualified"`
	HostProbe                 string             `json:"host_probe,omitempty"`
	Steps                     []string           `json:"steps"`
	RemainingGates            []string           `json:"remaining_gates"`
}

type Plan struct {
	Version         int      `json:"version"`
	ID              string   `json:"id"`
	SHA256          string   `json:"sha256"`
	Kind            string   `json:"kind"`
	Authority       string   `json:"authority"`
	ExecutableHere  bool     `json:"executable_here"`
	ExecutionStatus string   `json:"execution_status"`
	Effects         []string `json:"execution_effects"`
	Runtime         Runtime  `json:"runtime"`
}

func Read(id string) (Plan, error) {
	var legacy map[string]any
	switch id {
	case Lifecycle:
		legacy = LegacyRuntime()
	case HTTP:
		legacy = LegacyHTTP()
	default:
		return Plan{}, ErrSelection
	}
	out := Plan{Version: 1, ID: id, Kind: "disposable-runtime-checkpoint", Authority: "local-owner only; existing explicit acknowledged urbit-runner qualifier; collector enrollment conveys no execution authority", ExecutionStatus: "metadata-only; no shared test executor", Effects: []string{"local-metadata-read", "live-probe", "disposable-test-execution", "runtime-mutation", "runner-cache-write", "network-download"}}
	raw, err := json.Marshal(legacy)
	if err != nil {
		return Plan{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&out.Runtime); err != nil {
		return Plan{}, err
	}
	raw, err = json.Marshal(out)
	if err != nil {
		return Plan{}, err
	}
	digest := sha256.Sum256(raw)
	out.SHA256 = hex.EncodeToString(digest[:])
	return out, nil
}

func List() ([]Plan, error) {
	out := make([]Plan, 0, 2)
	for _, id := range []string{Lifecycle, HTTP} {
		p, err := Read(id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
