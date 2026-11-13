//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"strings"
	"testing"
)

func TestContainerResourcesStopResizeRestartAndMetrics(t *testing.T) {
	f, state, persist := managedFixture(t)
	ctx := context.Background()
	state, _, err := applyManaged(ctx, "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	// Desired state cannot relabel the original legacy launch, even in recovery.
	state.MemoryGiB, state.CPUCores = 16, 8
	_, obs, err := observeManaged(ctx, state, f.d)
	if err != nil || obs.Resources == nil || obs.Resources.MemoryLimitBytes != 4<<30 {
		t.Fatal("legacy launch changed", obs, err)
	}
	state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	f.cgroups = "17179869184\n0\n256\n800000 100000\n"
	saved := false
	save := func(s ManagedState) error {
		for _, c := range s.Runtime.Containers {
			if !c.Removed && c.Kind != "probe" {
				if c.MemoryGiB != 16 || c.CPUCores != 8 {
					t.Fatal("new intent did not persist selected limits before Docker", c)
				}
				saved = true
			}
		}
		return persist(s)
	}
	state, _, err = applyManaged(ctx, "start", state, save, f.d)
	if err != nil || !saved {
		t.Fatal("resize restart", err)
	}
	for _, c := range f.containers {
		h := mapAt(c.doc, "HostConfig")
		if h["Memory"] != int64(16<<30) || h["MemorySwap"] != int64(16<<30) || h["NanoCpus"] != int64(8e9) {
			t.Fatal("wrong Docker limits", h)
		}
	}
	baseStats := f.d.stats
	f.d.stats = func(ctx context.Context, socket, id string) (string, error) {
		raw, err := baseStats(ctx, socket, id)
		return strings.ReplaceAll(raw, "4294967296", "17179869184"), err
	}
	_, obs, err = observeManaged(ctx, state, f.d)
	if err != nil || obs.Resources == nil || obs.Resources.MemoryLimitBytes != 16<<30 {
		t.Fatal("new limits not observed", obs, err)
	}
	state, _, err = applyManaged(ctx, "stop", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err = applyManaged(ctx, "pack", state, save, f.d)
	if err != nil {
		t.Fatal("maintenance lost limits", err)
	}
}

func TestContainerResourcesProvisionProbeAndContainment(t *testing.T) {
	for _, n := range []uint8{1, 8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f, s, persist := managedFixture(t)
			s.MemoryGiB, s.CPUCores = n, n
			f.cgroups = fmt.Sprintf("%d\n0\n256\n%d 100000\n", int64(n)<<30, int64(n)*100000)
			s, _, err := applyManaged(context.Background(), "start", s, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			q, err := managedSession(&s, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range f.containers {
				raw, _ := json.Marshal(c.doc)
				var inspected container
				if json.Unmarshal(raw, &inspected) != nil {
					t.Fatal("fixture")
				}
				if err = q.containment(inspected, c.kind); err != nil {
					t.Fatal(err)
				}
				inspected.HostConfig.MemorySwap = -1
				if q.containment(inspected, c.kind) == nil {
					t.Fatal("unlimited swap accepted")
				}
			}
		})
	}
}

func TestContainerResourcesInvalidAndStats(t *testing.T) {
	for _, pair := range [][2]uint8{{17, 2}, {4, 17}, {255, 255}} {
		f, s, p := managedFixture(t)
		s.MemoryGiB, s.CPUCores = pair[0], pair[1]
		if _, _, err := applyManaged(context.Background(), "start", s, p, f.d); err == nil || len(f.events) > 0 {
			t.Fatal("invalid resource admission reached Docker", err, f.events)
		}
	}
	id := strings.Repeat("a", 64)
	raw := statsJSON(id, "123", "17179869184", "2")
	raw = raw[:len(raw)-1] + `,"cpu_stats":{"cpu_usage":{"total_usage":1},"system_cpu_usage":100,"online_cpus":16}}`
	got, err := parseResourcesWithLimits(raw, id, 16<<30, 8)
	if err != nil || got.CPU == nil || got.CPU.LimitCores != 8 {
		t.Fatal("CPU limit lost", got, err)
	}
	if _, err = parseResources(raw, id); err == nil {
		t.Fatal("legacy fixed contract accepted mismatched RAM")
	}
	q := qualifier{d: dependencies{memoryGiB: 16, cpuCores: 8}, r: Result{Containers: []ContainerRecord{{Kind: "http-boot"}}}}
	memory, cpu := q.containerLimits("http-boot")
	if memory != 4<<30 || cpu != 2 {
		t.Fatal("historical contract relabeled")
	}
	if protocol.EffectiveMemoryGiB(0) != 4 || protocol.EffectiveCPUCores(0) != 2 {
		t.Fatal("historical defaults")
	}
}
