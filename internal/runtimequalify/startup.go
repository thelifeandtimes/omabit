package runtimequalify

import "time"

// StartupResumeCandidate excludes bootstrap, maintenance and unresolved resource
// intents. Actual container ownership and terminal state are checked separately.
func StartupResumeCandidate(s ManagedState) bool {
	if !s.Prepared || !s.MaintenanceQualified() || s.UncertainProvisioning() || s.UncertainMaintenance() || s.RollNeedsRuntime || s.PruneIntent != nil || s.Routine != nil && s.Routine.FinishedAt == nil {
		return false
	}
	n := s.Runtime.Network
	if n == nil || n.Removed || n.CreateUncertain || !n.CreateDispatched || !containerIDPattern.MatchString(n.ID) {
		return false
	}
	live := 0
	for _, c := range s.Runtime.Containers {
		if c.CreateUncertain {
			return false
		}
		if c.Removed {
			continue
		}
		kindOK := c.Kind == "http-boot" || c.Kind == "http-restart"
		if s.Comet != nil {
			kindOK = c.Kind == "comet-boot" || c.Kind == "comet-resume"
		}
		if !kindOK || !c.CreateDispatched || !containerIDPattern.MatchString(c.ID) || !c.HostHTTPVerified {
			return false
		}
		live++
	}
	return live == 1
}

// A retained stopped container normally needs operator cleanup. Startup may do
// that cleanup only with fresh evidence of a clean exit, never absence or OOM.
func StartupResumeReady(s ManagedState, observation ManagedObservation) bool {
	if !StartupResumeCandidate(s) || observation.State != "stopped" {
		return false
	}
	now := time.Now()
	for _, c := range s.Runtime.Containers {
		if c.Removed {
			continue
		}
		last := c.LastState
		if last == nil || last.ObservedAt.IsZero() || now.Sub(last.ObservedAt) > 30*time.Second || last.ObservedAt.After(now.Add(5*time.Second)) || last.Running || last.Paused || last.Restarting || last.Dead || last.OOMKilled || last.ExitCode != 0 {
			return false
		}
	}
	return true
}
