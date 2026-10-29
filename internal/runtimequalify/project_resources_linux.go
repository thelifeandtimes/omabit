//go:build linux && amd64

package runtimequalify

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"sync"
	"time"
)

// Ephemeral, bounded samples only. No journal or disk writes during polling.
var projectResourceSamples = struct {
	sync.Mutex
	values map[string]*ManagedResources
}{values: map[string]*ManagedResources{}}

func projectResourceMetrics(ctx context.Context, q *qualifier, c container) *protocol.ResourceMetrics {
	fallback := &protocol.ResourceMetrics{Status: "unavailable", Source: "docker-stats-v1.47", ContainerID: c.ID}
	if q.d.stats == nil || !*c.State.Running {
		return fallback
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	data, err := q.d.stats(ctx, q.socket, c.ID)
	if err != nil || ctx.Err() != nil {
		return fallback
	}
	sample, err := parseResources(data, c.ID)
	if err != nil {
		return fallback
	}
	now := time.Now().UTC()
	if now.Sub(sample.ObservedAt) > 30*time.Second || sample.ObservedAt.After(now.Add(5*time.Second)) {
		return fallback
	}
	out := &protocol.ResourceMetrics{Status: "fresh", Source: "docker-stats-v1.47", ContainerID: c.ID, ObservedAt: &sample.ObservedAt, MemoryCurrentBytes: &sample.MemoryCurrentBytes, MemoryLimitBytes: &sample.MemoryLimitBytes, PIDsCurrent: &sample.PIDsCurrent}
	key := q.socket + "/" + c.ID + "/" + c.State.StartedAt
	projectResourceSamples.Lock()
	defer projectResourceSamples.Unlock()
	old := projectResourceSamples.values[key]
	for k, v := range projectResourceSamples.values {
		if now.Sub(v.ObservedAt) > 30*time.Second {
			delete(projectResourceSamples.values, k)
		}
	}
	if len(projectResourceSamples.values) < 1024 {
		projectResourceSamples.values[key] = sample
	}
	if sample.CPU != nil {
		out.CPULimitCores = &sample.CPU.LimitCores
		if old != nil && old.CPU != nil {
			p, n := old.CPU, sample.CPU
			age := sample.ObservedAt.Sub(old.ObservedAt)
			if age >= time.Second && age <= 30*time.Second && n.OnlineCPUs == p.OnlineCPUs && n.TotalNS >= p.TotalNS && n.SystemNS > p.SystemNS && n.TotalNS-p.TotalNS <= n.SystemNS-p.SystemNS {
				value := float64(n.TotalNS-p.TotalNS) / float64(n.SystemNS-p.SystemNS) * float64(n.OnlineCPUs) * 100
				out.CPUUsagePercent = &value
			}
		}
	}
	return out
}
