package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"time"
)

type projectAllocationSample struct {
	observation projectusage.Observation
	finished    time.Time
}

// Called under r.mu only. Samples are bounded memory, not stored authority or a
// background guard. An unknown result clears its old baseline. Fast polling
// retains an earlier comparable sample instead of forever shortening the window.
func (r *Server) projectAllocationGrowth(out projectusage.Observation, now time.Time) projectusage.Growth {
	if r.projectAllocations == nil {
		r.projectAllocations = map[string]projectAllocationSample{}
	}
	key := out.Binding.ProjectID
	previous, found := r.projectAllocations[key]
	var elapsed time.Duration
	if found {
		elapsed = now.Sub(previous.finished)
	}
	growth := projectusage.Compare(previous.observation, out, elapsed)
	if out.Status != "measured" || out.Validate() != nil {
		delete(r.projectAllocations, key)
		return growth
	}
	if found && growth.Reason == "window-too-short" {
		return growth
	}
	if !found && len(r.projectAllocations) >= 32 {
		oldest := ""
		var at time.Time
		for id, sample := range r.projectAllocations {
			if oldest == "" || sample.finished.Before(at) {
				oldest, at = id, sample.finished
			}
		}
		delete(r.projectAllocations, oldest)
	}
	copy := out.Clone()
	copy.Growth = nil
	r.projectAllocations[key] = projectAllocationSample{copy, now}
	return growth
}
