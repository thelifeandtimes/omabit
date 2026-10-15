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

// A retained terminal container needs fresh, complete evidence. Only a durable
// reboot-resume intent can authorize a nonzero/OOM exit; missing or active
// containers are never inferred to be stopped.
func StartupResumeReady(s ManagedState, observation ManagedObservation) bool {
	if !StartupResumeCandidate(s) || (observation.State != "stopped" && observation.State != "failed") {
		return false
	}
	now := time.Now()
	for _, c := range s.Runtime.Containers {
		if c.Removed {
			continue
		}
		last := c.LastState
		intent := s.StartupResume
		if intent != nil && intent.ContainerID != c.ID {
			return false
		}
		unclean := intent != nil && intent.AllowUnclean
		if last == nil || last.ObservedAt.IsZero() || now.Sub(last.ObservedAt) > 30*time.Second || last.ObservedAt.After(now.Add(5*time.Second)) || last.Running || last.Paused || last.Restarting || last.Dead || (!unclean && (last.OOMKilled || last.ExitCode != 0)) {
			return false
		}
	}
	return true
}
