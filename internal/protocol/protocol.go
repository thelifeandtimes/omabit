// Package protocol defines the versioned demo runner RPC wire format.
package protocol

import "time"

const (
	Version = 1
	Header  = "Urbit-Runner-Protocol"
)

type Hello struct {
	ProtocolVersion int      `json:"protocol_version"`
	RunnerID        string   `json:"runner_id"`
	Mode            string   `json:"mode"`
	Features        []string `json:"features"`
}

type Configuration struct {
	HTTPPort uint16 `json:"http_port"`
	AmesPort uint16 `json:"ames_port"`
	Loom     uint8  `json:"loom"`
}

type Instance struct {
	ID                string        `json:"id"`
	Label             string        `json:"label"`
	Config            Configuration `json:"config"`
	ConfigRevision    uint64        `json:"config_revision"`
	AppliedRevision   uint64        `json:"applied_revision"`
	DesiredState      string        `json:"desired_state"`
	ObservedState     string        `json:"observed_state"`
	ActiveOperationID string        `json:"active_operation_id,omitempty"`
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
	RunnerID   string      `json:"runner_id"`
	Mode       string      `json:"mode"`
	Instances  []Instance  `json:"instances"`
	Operations []Operation `json:"operations"`
	Incidents  []Incident  `json:"incidents"`
}

type ConfigureRequest struct {
	ExpectedRevision uint64        `json:"expected_revision"`
	Config           Configuration `json:"config"`
}

type OperationRequest struct {
	RequestID        string `json:"request_id"`
	Kind             string `json:"kind"`
	ExpectedRevision uint64 `json:"expected_revision"`
}

type APIError struct {
	Code            string  `json:"code"`
	Message         string  `json:"message"`
	CurrentRevision *uint64 `json:"current_revision,omitempty"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}
