package projectnetwork

// Clone detaches mutable history while preserving its exact Go representation.
// In particular, an omitted JSON slice must not turn an accepted empty slice
// into nil during a driver's exact selection comparison.
func (h AnchorRecoveryHistory) Clone() AnchorRecoveryHistory {
	return cloneAnchorRecoveryHistory(h)
}

// Clone returns an independent retained intent, including nested observations.
func (s AnchorDrain) Clone() AnchorDrain {
	return cloneAnchorDrain(s)
}
