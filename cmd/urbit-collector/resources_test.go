package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestResourceTextReportsUnitsFreshnessAndUnavailable(t *testing.T) {
	current, limit, pids := uint64(64<<20), uint64(4<<30), uint64(12)
	at := time.Date(2026, 9, 25, 18, 0, 0, 0, time.UTC)
	for _, status := range []string{"fresh", "stale", "unavailable"} {
		t.Run(status, func(t *testing.T) {
			var text bytes.Buffer
			r := &protocol.ResourceMetrics{Status: status, Source: "docker-stats-v1.47", ContainerID: strings.Repeat("a", 64), ObservedAt: &at, MemoryCurrentBytes: &current, MemoryLimitBytes: &limit, PIDsCurrent: &pids}
			if err := newOutput(&text, "text", false).Encode(protocol.Instance{ID: "alpha", Health: "healthy", Resources: r}); err != nil {
				t.Fatal(err)
			}
			out := text.String()
			if !strings.Contains(out, "Container resources: "+status) || !strings.Contains(out, "healthy") {
				t.Fatalf("missing independent resource status: %s", out)
			}
			if status == "unavailable" {
				if strings.Contains(out, "Cgroup RAM") {
					t.Fatalf("unavailable sample displayed as a number: %s", out)
				}
			} else if !strings.Contains(out, "64.00 / 4096.00 MiB") || !strings.Contains(out, "tasks: 12") || !strings.Contains(out, "includes cache") || !strings.Contains(out, "2026-09-25T18:00:00Z") {
				t.Fatalf("resource units/evidence missing: %s", out)
			}
		})
	}
}
