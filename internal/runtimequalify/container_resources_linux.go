//go:build linux && amd64

package runtimequalify

import "git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"

// Resolve immutable launch evidence before desired settings. A legacy container
// is always 4 GiB / 2 CPUs even when its ship now requests a different ceiling.
func (q *qualifier) containerLimits(kind string) (int64, int64) {
	memory, cpu := q.d.memoryGiB, q.d.cpuCores
	for _, c := range q.r.Containers {
		if c.Kind == kind {
			memory, cpu = c.MemoryGiB, c.CPUCores
			break
		}
	}
	return int64(protocol.EffectiveMemoryGiB(memory)) << 30, int64(protocol.EffectiveCPUCores(cpu))
}
