package runtimequalify

import (
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

// ErrHostCompatibility is an admission/observation failure of the host runtime,
// not evidence of pier damage or a failed persistent store. Callers must keep
// inventory available and preserve existing maintenance/provisioning uncertainty.
var ErrHostCompatibility = errors.New("host runtime compatibility denied")

func compatibilityObservation(state ManagedState) ManagedObservation {
	return ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: RecoveryRequiredOnCompatibility(state, false)}
}

// RecoveryRequiredOnCompatibility preserves previously established recovery and
// durable unresolved outcomes when current host evidence is unavailable. An
// admission failure or canceled probe alone does not introduce recovery work.
func RecoveryRequiredOnCompatibility(state ManagedState, prior bool) bool {
	if prior || state.UncertainMaintenance() || state.UncertainProvisioning() {
		return true
	}
	if network := state.Runtime.Network; network != nil && network.CreateUncertain {
		return true
	}
	for _, record := range state.Runtime.Containers {
		if record.CreateUncertain {
			return true
		}
		if last := record.LastState; !record.Removed && last != nil &&
			(!last.Running || last.Paused || last.Restarting || last.Dead || last.OOMKilled || last.ExitCode != 0 || last.LaunchError != "") {
			return true
		}
	}
	return false
}

// Filesystem-only enrollments have no daemon observation. Historical reports
// retain their original version, independently of today's engine support range.
func recordedEngineEvidenceValid(r Result) bool {
	if r.EngineVersion == "" {
		return len(r.Containers) == 0 && r.ImageID == "" && r.RepoDigest == "" && !r.LimitsObserved && !r.CheckpointPassed
	}
	return hostcompat.ValidateRecordedEngineVersion(r.EngineVersion)
}
