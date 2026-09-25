package runtimequalify

import "time"

// ManagedState is private driver state, persisted transactionally by the runner.
// It is not an API for importing a pier or arbitrary Docker resources.
type ManagedState struct {
	ID                   string              `json:"id"`
	Port                 uint16              `json:"port"`
	Prepared             bool                `json:"prepared"`
	Initialized          bool                `json:"initialized"`
	Runtime              Result              `json:"runtime"`
	MaintenanceUncertain bool                `json:"maintenance_uncertain,omitempty"`
	Maintenance          *ManagedMaintenance `json:"maintenance,omitempty"`
}

type ManagedMaintenance struct {
	Kind            string     `json:"kind"`
	BeforePierBytes int64      `json:"before_pier_bytes"`
	AfterPierBytes  int64      `json:"after_pier_bytes"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
}

type ManagedObservation struct {
	State            string
	Health           string
	Endpoint         string
	RecoveryRequired bool
}

type ManagedHost struct {
	Directory string
	Socket    string
	close     func()
}

func (h *ManagedHost) Close() { h.close() }
