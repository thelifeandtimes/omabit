//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestProjectResourcesMemoryCPUAndStaleSamples(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	q, err := managedSession(&state, nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	c, err := q.inspect(context.Background(), len(q.r.Containers)-1)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	at := now.Add(-2 * time.Second)
	total := 100
	system := 1000
	q.d.stats = func(_ context.Context, socket, id string) (string, error) {
		if id != c.ID {
			t.Fatal("wrong container")
		}
		return fmt.Sprintf(`{"id":%q,"read":%q,"memory_stats":{"usage":1234,"limit":4294967296},"pids_stats":{"current":8},"cpu_stats":{"cpu_usage":{"total_usage":%d},"system_cpu_usage":%d,"online_cpus":2}}`, id, at.Format(time.RFC3339Nano), total, system), nil
	}
	a := projectResourceMetrics(context.Background(), q, c)
	if a.Status != "fresh" || *a.MemoryCurrentBytes != 1234 || a.CPUUsagePercent != nil {
		t.Fatal(a)
	}
	at = now
	total = 200
	system = 2000
	b := projectResourceMetrics(context.Background(), q, c)
	if b.CPUUsagePercent == nil || *b.CPUUsagePercent != 20 {
		t.Fatal("CPU delta missing", b)
	}
	at = now.Add(-time.Minute)
	if v := projectResourceMetrics(context.Background(), q, c); v.Status != "unavailable" {
		t.Fatal("stale sample")
	}
	at = now
	c.ID = strings.Repeat("c", 64)
	if v := projectResourceMetrics(context.Background(), q, c); v.CPUUsagePercent != nil {
		t.Fatal("replacement inherited CPU")
	}
}
