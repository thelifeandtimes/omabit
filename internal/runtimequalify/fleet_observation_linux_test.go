//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
)

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
