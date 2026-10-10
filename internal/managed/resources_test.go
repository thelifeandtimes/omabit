package managed

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func resourceSample(at time.Time) *runtimequalify.ManagedResources {
	return &runtimequalify.ManagedResources{ContainerID: strings.Repeat("a", 64), ObservedAt: at, MemoryCurrentBytes: 0, MemoryLimitBytes: 4 << 30, PIDsCurrent: 12}
}

func TestResourceFreshnessIsIndependentOfHealthAndRetainsZero(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name   string
		age    time.Duration
		status string
	}{
		{"fresh", 0, "fresh"}, {"boundary", 30 * time.Second, "fresh"},
		{"old", 31 * time.Second, "stale"}, {"clock reversal", -time.Minute, "stale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var v storedInstance
			applyObservation(&v, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(now.Add(-tc.age))}, now)
			ageResources(v.Public.Resources, now)
			r := v.Public.Resources
			if r.Status != tc.status || r.MemoryCurrentBytes == nil || *r.MemoryCurrentBytes != 0 || v.Public.Health != "healthy" {
				t.Fatalf("freshness/zero semantics: %+v", v.Public)
			}
			data, err := json.Marshal(r)
			if err != nil || !strings.Contains(string(data), `"memory_current_bytes":0`) {
				t.Fatalf("zero disappeared from wire output: %s %v", data, err)
			}
		})
	}
	var v storedInstance
	applyObservation(&v, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(now)}, now)
	applyObservation(&v, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, now)
	if v.Public.Resources.Status != "unavailable" || v.Public.Resources.ObservedAt != nil || v.Public.Resources.MemoryCurrentBytes != nil || v.Public.Health != "healthy" || v.Public.RecoveryRequired {
		t.Fatalf("failed resource read retained old numbers or poisoned health: %+v", v.Public)
	}
	applyObservation(&v, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, now)
	if v.Public.Resources != nil {
		t.Fatal("stopped runtime retained resource observation")
	}
}

func TestPartialResourceEvidenceCannotBeFresh(t *testing.T) {
	for _, fault := range []string{"timestamp", "container", "memory", "limit", "tasks"} {
		t.Run(fault, func(t *testing.T) {
			var v storedInstance
			now := time.Now().UTC()
			applyObservation(&v, runtimequalify.ManagedObservation{State: "running", Resources: resourceSample(now)}, now)
			r := v.Public.Resources
			switch fault {
			case "timestamp":
				r.ObservedAt = nil
			case "container":
				r.ContainerID = ""
			case "memory":
				r.MemoryCurrentBytes = nil
			case "limit":
				r.MemoryLimitBytes = nil
			case "tasks":
				r.PIDsCurrent = nil
			}
			ageResources(r, now)
			if r.Status != "unavailable" || r.ObservedAt != nil || r.MemoryCurrentBytes != nil || r.MemoryLimitBytes != nil || r.PIDsCurrent != nil {
				t.Fatalf("partial sample was published: %+v", r)
			}
		})
	}
}

func TestSnapshotsAgeResourcesWithoutRefreshingOrOverwritingThem(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	observed := time.Now().UTC().Add(-time.Minute)
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(observed)}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		snapshot, err := s.snapshot()
		if err != nil {
			t.Fatal(err)
		}
		r := snapshot.Instances[0].Resources
		if r.Status != "stale" || !r.ObservedAt.Equal(observed) || snapshot.Instances[0].Health != "healthy" {
			t.Fatalf("snapshot refreshed measurement or mixed health: %+v", snapshot.Instances[0])
		}
	}
	stored, err := loadInstance(s.db, "alpha")
	if err != nil || stored.Public.Resources.Status != "fresh" || !stored.Public.Resources.ObservedAt.Equal(observed) {
		t.Fatal("snapshot rewrote durable observation")
	}
	// An old in-flight poll cannot resurrect a sample after a lifecycle update.
	op, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	op, current, err := s.beginOperation(op.ID)
	if err != nil || current.Public.Resources != nil {
		t.Fatalf("operation retained running sample: %+v %v", current.Public, err)
	}
	if err := s.finish(op, current.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	if err := s.observe(stored, stored.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(time.Now().UTC())}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.snapshot()
	if err != nil || snapshot.Instances[0].Resources != nil || snapshot.Instances[0].ObservedState != "stopped" {
		t.Fatalf("late observation resurrected stopped sample: %+v %v", snapshot, err)
	}
}
