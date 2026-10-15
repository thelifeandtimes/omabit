package protocol

import "time"

// PortSuggestion is point-in-time admission evidence, not a reserved listener.
type PortSuggestion struct {
	RunnerID      string    `json:"runner_id"`
	HTTPPort      uint16    `json:"http_port"`
	ContainerPort uint16    `json:"container_port"`
	ObservedAt    time.Time `json:"observed_at"`
}
