package managed

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestCPUUsesExactCounterDeltasAndPerCoreUnits(t *testing.T) {
	now := time.Now().UTC()
	base := uint64(1) << 63 // beyond JavaScript's exact-integer range
	previous := cpuBaseline{ContainerID: strings.Repeat("a", 64), ObservedAt: now.Add(-10 * time.Second), Counters: runtimequalify.ManagedCPU{TotalNS: base, SystemNS: base, OnlineCPUs: 8, LimitCores: 2}}
	for _, want := range []float64{0, 0.5, 50, 100, 200} {
		current := previous
		current.ObservedAt = now
		current.Counters.TotalNS += uint64(want * 100_000_000)
		current.Counters.SystemNS += 80_000_000_000
		got := cpuUsage(&previous, &current, now)
		if got == nil || *got != want {
			t.Fatalf("CPU percentage: %v; want %v", got, want)
		}
	}
}

func TestCPURefusesMissingResetReplacedAndStaleSamples(t *testing.T) {
	now := time.Now().UTC()
	previous := cpuBaseline{ContainerID: strings.Repeat("a", 64), ObservedAt: now.Add(-10 * time.Second), Counters: runtimequalify.ManagedCPU{TotalNS: 100, SystemNS: 1000, OnlineCPUs: 8, LimitCores: 2}}
	for _, fault := range []string{"first", "container", "total-reset", "system-reset", "system-unchanged", "impossible", "duplicate", "reversal", "gap", "future", "stale", "hotplug", "limit"} {
		t.Run(fault, func(t *testing.T) {
			checkTime := now
			p, c := previous, previous
			c.ObservedAt, c.Counters.TotalNS, c.Counters.SystemNS = now, 200, 2000
			var before = &p
			switch fault {
			case "first":
				before = nil
			case "container":
				c.ContainerID = strings.Repeat("b", 64)
			case "total-reset":
				c.Counters.TotalNS = 0
			case "system-reset":
				c.Counters.SystemNS = 0
			case "system-unchanged":
				c.Counters.SystemNS = p.Counters.SystemNS
			case "impossible":
				c.Counters.TotalNS = 5000
			case "duplicate":
				c.ObservedAt = p.ObservedAt
			case "reversal":
				c.ObservedAt = p.ObservedAt.Add(-time.Second)
			case "gap":
				p.ObservedAt = now.Add(-time.Minute)
			case "future":
				c.ObservedAt = now.Add(6 * time.Second)
			case "stale":
				checkTime = now.Add(time.Minute)
			case "hotplug":
				c.Counters.OnlineCPUs = 4
			case "limit":
				c.Counters.LimitCores = 1
			}
			if value := cpuUsage(before, &c, checkTime); value != nil {
				t.Fatal("invalid evidence produced a rate", *value)
			}
		})
	}
}

func TestCPUObservationsWarmUpIndependentlyAndClearOnStop(t *testing.T) {
	now := time.Now().UTC()
	var v storedInstance
	observe := func(at time.Time, total, system uint64, withCPU bool) {
		r := resourceSample(at)
		if withCPU {
			r.CPU = &runtimequalify.ManagedCPU{TotalNS: total, SystemNS: system, OnlineCPUs: 8, LimitCores: 2}
		}
		applyObservation(&v, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: r}, at)
	}
	observe(now, 0, 1, true)
	if v.Public.Resources.CPUUsagePercent != nil || v.Public.Resources.CPULimitCores == nil || *v.Public.Resources.CPULimitCores != 2 {
		t.Fatal("first sample fabricated a CPU rate")
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	v = storedInstance{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	observe(now.Add(10*time.Second), 0, 80_000_000_001, true)
	if v.Public.Resources.CPUUsagePercent == nil || *v.Public.Resources.CPUUsagePercent != 0 {
		t.Fatal("two idle counter readings lost genuine zero")
	}
	observe(now.Add(20*time.Second), 0, 0, false)
	if v.Public.Resources.CPUUsagePercent != nil || v.CPU != nil || v.Public.Resources.Status != "fresh" || v.Public.Health != "healthy" {
		t.Fatal("missing CPU damaged memory/health or kept a stale rate")
	}
	observe(now.Add(30*time.Second), 100, 1000, true)
	if v.Public.Resources.CPUUsagePercent != nil {
		t.Fatal("missing baseline did not require warmup")
	}
	applyObservation(&v, runtimequalify.ManagedObservation{State: "stopped"}, now.Add(31*time.Second))
	if v.CPU != nil || v.Public.Resources != nil {
		t.Fatal("stopped runtime retained CPU evidence")
	}
}
