package runtimequalify

// ManagedState is private driver state, persisted transactionally by the runner.
// It is not an API for importing a pier or arbitrary Docker resources.
type ManagedState struct {
	ID          string `json:"id"`
	Port        uint16 `json:"port"`
	Prepared    bool   `json:"prepared"`
	Initialized bool   `json:"initialized"`
	Runtime     Result `json:"runtime"`
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
