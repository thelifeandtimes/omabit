package managed

import (
	"math"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// A single private baseline per instance; no usage history or raw uint64 counters
// are sent to JavaScript. This optional observation state is not runtime intent.
type cpuBaseline struct {
	ContainerID string                    `json:"container_id"`
	ObservedAt  time.Time                 `json:"observed_at"`
	Counters    runtimequalify.ManagedCPU `json:"counters"`
}

func cpuUsage(previous, current *cpuBaseline, now time.Time) *float64 {
	if previous == nil || current == nil || previous.ContainerID != current.ContainerID || len(current.ContainerID) != 64 {
		return nil
	}
	elapsed := current.ObservedAt.Sub(previous.ObservedAt)
	if elapsed < time.Second || elapsed > 30*time.Second || now.Sub(current.ObservedAt) > 30*time.Second || current.ObservedAt.After(now.Add(5*time.Second)) {
		return nil
	}
	p, c := previous.Counters, current.Counters
	if c.OnlineCPUs == 0 || c.OnlineCPUs > 4096 || c.OnlineCPUs != p.OnlineCPUs || c.LimitCores != p.LimitCores || c.TotalNS < p.TotalNS || c.SystemNS <= p.SystemNS {
		return nil
	}
	used, system := c.TotalNS-p.TotalNS, c.SystemNS-p.SystemNS
	if used > system {
		return nil
	}
	// Docker's per-core convention: 100% is one fully occupied logical CPU.
	// System time is cumulative across online CPUs, avoiding wall-clock-rate
	// artifacts. Subtract uint64 counters before converting to floating point.
	percent := float64(used) / float64(system) * float64(c.OnlineCPUs) * 100
	if math.IsNaN(percent) || math.IsInf(percent, 0) {
		return nil
	}
	return &percent
}
