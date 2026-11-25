//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func TestFleetInventoryIgnoresUnrelatedPackButRejectsNamespaceChanges(t *testing.T) {
	a := strings.Repeat("a", 64)
	m := strings.Repeat("b", 64)
	u := strings.Repeat("c", 64)
	before := []projectnetwork.NetworkContainer{{ID: a, Name: "anchor", NetworkMode: "bridge"}, {ID: m, Name: "member", NetworkMode: "container:" + a}}
	after := append(append([]projectnetwork.NetworkContainer{}, before...), projectnetwork.NetworkContainer{ID: u, Name: "other-zod-pack", NetworkMode: "none"})
	if !sameProjectNamespaceInventory(a, before, after) || !sameProjectNamespaceInventory(a, after, before) {
		t.Fatal("unrelated maintenance invalidated project display")
	}
	after[2].NetworkMode = "container:" + m
	if sameProjectNamespaceInventory(a, before, after) {
		t.Fatal("foreign indirect namespace join accepted")
	}
	after[2].NetworkMode = "container:missing"
	if sameProjectNamespaceInventory(a, before, after) {
		t.Fatal("ambiguous inventory accepted")
	}
	after = append([]projectnetwork.NetworkContainer{}, before...)
	after[1].Name = "changed"
	if sameProjectNamespaceInventory(a, before, after) {
		t.Fatal("selected identity changed")
	}
}

func TestFleetObservationEvidenceScope(t *testing.T) {
	f, input, _ := resumedStopAdapterFixture(t, false, 2)
	s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
	inspect := f.f.d.InspectHost
	calls := 0
	deny := false
	f.f.d.InspectHost = func() hostcompat.Report {
		calls++
		if deny {
			return hostcompat.Report{}
		}
		return inspect()
	}
	for i := 1; i <= 2; i++ {
		d := fleetObservationDependencies(f.f.d)
		raw, err := openCurrentProjectReadiness(context.Background(), s, d)
		if err != nil {
			t.Fatal(err)
		}
		e := raw.(*projectCurrentReadinessEngine)
		e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
		for j := 0; j < 2; j++ {
			if _, err := projectFleetInspect(context.Background(), e); err != nil {
				t.Fatal(err)
			}
		}
		if calls != i {
			t.Fatalf("package scans=%d for %d independent responses", calls, i)
		}
		probe := e.source.anchor.base.q.d.ProbeEngine
		e.source.anchor.base.q.d.ProbeEngine = func(context.Context, string, string) (hostcompat.Engine, error) {
			return hostcompat.Engine{}, errors.New("engine replaced")
		}
		if _, err := projectFleetInspect(context.Background(), e); err == nil {
			t.Fatal("cached package evidence hid changed live engine")
		}
		e.source.anchor.base.q.d.ProbeEngine = probe
	}
	deny = true
	if _, err := openCurrentProjectReadiness(context.Background(), s, fleetObservationDependencies(f.f.d)); err == nil {
		t.Fatal("new response reused previous host evidence")
	}
	if _, err := openProjectAnchorStart(context.Background(), s.Project, s.Launches[0].Anchor.Prepared, f.f.d); err == nil {
		t.Fatal("mutation admission reused observation evidence")
	}
}

func TestExpandedFleetObservationEvidenceScope(t *testing.T) {
	f, input, _ := resumedStopAdapterFixture(t, false, 2)
	s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
	expanded, err := SeedProjectExpansion(s)
	if err != nil {
		t.Fatal(err)
	}
	inspect := f.f.d.InspectHost
	calls := 0
	f.f.d.InspectHost = func() hostcompat.Report { calls++; return inspect() }
	e, err := expansionEngineWithDependencies(context.Background(), expanded, fleetObservationDependencies(f.f.d))
	if err != nil {
		t.Fatal(err)
	}
	e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	for i := 0; i < 2; i++ {
		if _, err := projectFleetInspect(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("expanded observation scanned packages %d times", calls)
	}
}

func TestFleetObservationEvidenceConcurrentReads(t *testing.T) {
	var calls atomic.Int32
	d := fleetObservationDependencies(dependencies{InspectHost: func() hostcompat.Report { calls.Add(1); return hostcompat.Report{PolicyID: "fixture"} }})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			if d.InspectHost().PolicyID != "fixture" {
				t.Error("lost evidence")
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("duplicate host scan", calls.Load())
	}
}

func TestFleetObservationRemainsHealthyDuringUnrelatedPack(t *testing.T) {
	for _, mode := range []string{"none", "foreign-join", "mutation"} {
		t.Run(mode, func(t *testing.T) {
			f, input, _ := resumedStopAdapterFixture(t, false, 2)
			s := ProjectCurrentReadinessSelection{Project: input.Project, Launches: input.Launches, Resumes: input.Resumes, Original: f.readiness["receiver"]}
			d := f.f.d
			if mode != "mutation" {
				d = fleetObservationDependencies(d)
			}
			raw, err := openCurrentProjectReadiness(t.Context(), s, d)
			if err != nil {
				t.Fatal(err)
			}
			e := raw.(*projectCurrentReadinessEngine)
			e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			execute := e.source.anchor.base.q.d.execute
			inserted := false
			e.source.anchor.base.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				out, err := execute(ctx, args, env)
				if len(args) > 5 && args[4] == "network" && args[5] == "inspect" && !inserted {
					inserted = true
					v := &container{ID: strings.Repeat("7", 64), Name: "/other-zod-pack"}
					v.HostConfig.NetworkMode = "none"
					if mode == "foreign-join" {
						v.HostConfig.NetworkMode = "container:" + e.source.anchor.prepared.Anchor.ID
					}
					f.containers[v.ID] = v
				}
				return out, err
			}
			_, err = projectFleetInspect(t.Context(), e)
			if !inserted {
				t.Fatal("fixture did not exercise inventory churn")
			}
			if (err == nil) != (mode == "none") {
				t.Fatal("inventory boundary", mode, err)
			}
			if f.signals != 0 {
				t.Fatal("observation changed a process")
			}
		})
	}
}
