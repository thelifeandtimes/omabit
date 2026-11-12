package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"strings"
	"testing"
)

func TestContainerResourcesStoppedConfigurePreserveAndLaunch(t *testing.T) {
	s := testStore(t)
	req := registration("alpha", "register", 18080)
	_, _, err := s.register("actor", req)
	g02Check(t, err)
	config := req.Config
	config.MemoryGiB, config.CPUCores = 16, 8
	got, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: config})
	g02Check(t, err)
	if got.Config.MemoryGiB != 16 || got.Config.CPUCores != 8 || got.ConfigRevision != 2 {
		t.Fatal(got)
	}
	// An older collector's unrelated port edit must preserve both limits.
	old := req.Config
	old.HTTPPort++
	got, err = s.configure("old-client", "alpha", protocol.ConfigureRequest{ExpectedRevision: 2, Config: old})
	g02Check(t, err)
	if got.Config.MemoryGiB != 16 || got.Config.CPUCores != 8 {
		t.Fatal("legacy client reset resources", got)
	}
	start, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", ExpectedRevision: 3, RequestID: "start"})
	g02Check(t, err)
	op, v, err := s.beginOperation(start.ID)
	g02Check(t, err)
	if v.Runtime.MemoryGiB != 16 || v.Runtime.CPUCores != 8 {
		t.Fatal("launch lost resources")
	}
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "http-boot", ID: strings.Repeat("a", 64), MemoryGiB: 16, CPUCores: 8}}
	g02Check(t, s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, nil, false))
	smaller := got.Config
	smaller.MemoryGiB = 4
	smaller.CPUCores = 2
	if _, err = s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 3, Config: smaller}); err == nil {
		t.Fatal("running resize accepted")
	}
	snap, err := s.snapshot()
	g02Check(t, err)
	status := snap.Instances[0].RuntimeStatus
	if status.LaunchMemoryGiB == nil || *status.LaunchMemoryGiB != 16 || status.LaunchCPUCores == nil || *status.LaunchCPUCores != 8 {
		t.Fatal(status)
	}
	stop, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "stop", ExpectedRevision: 3, RequestID: "stop"})
	g02Check(t, err)
	op, v, err = s.beginOperation(stop.ID)
	g02Check(t, err)
	if v.Runtime.MemoryGiB != 16 || v.Runtime.CPUCores != 8 {
		t.Fatal("stop contract changed")
	}
	v.Runtime.Runtime.Containers = nil
	g02Check(t, s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false))
	got, err = s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 3, Config: smaller})
	g02Check(t, err)
	if got.Config.MemoryGiB != 4 || got.Config.CPUCores != 2 || got.ConfigRevision != 4 {
		t.Fatal("explicit default reset failed", got)
	}
	g02Check(t, s.initialize())
}

func TestContainerResourcesRefusesUncertainOrInvalidConfiguration(t *testing.T) {
	for _, state := range []string{"starting", "unknown", "failed", "stopping", "running", "recovery"} {
		t.Run(state, func(t *testing.T) {
			s := testStore(t)
			req := registration("alpha", "register", 18080)
			_, _, err := s.register("actor", req)
			g02Check(t, err)
			v, err := loadInstance(s.db, "alpha")
			g02Check(t, err)
			v.Public.ObservedState = state
			if state == "recovery" {
				v.Public.ObservedState = "stopped"
				v.Public.RecoveryRequired = true
			}
			saveRecoveryFixture(t, s, v)
			req.Config.MemoryGiB = 8
			if _, err = s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: req.Config}); err == nil {
				t.Fatal("unsafe resize accepted")
			}
		})
	}
	for _, pair := range [][2]uint8{{17, 2}, {4, 17}, {255, 255}} {
		s := testStore(t)
		req := registration("alpha", "register", 18080)
		req.Config.MemoryGiB, req.Config.CPUCores = pair[0], pair[1]
		if _, _, err := s.register("actor", req); err == nil {
			t.Fatal("invalid registration accepted")
		}
	}
}

func TestContainerResourcesCorruptStoreRefusesWithoutMigration(t *testing.T) {
	s := testStore(t)
	_, _, err := s.register("actor", registration("alpha", "register", 18080))
	g02Check(t, err)
	v, err := loadInstance(s.db, "alpha")
	g02Check(t, err)
	v.Runtime.MemoryGiB = 17
	saveRecoveryFixture(t, s, v)
	_, err = s.db.Exec("PRAGMA user_version=58")
	g02Check(t, err)
	if err = s.initialize(); err == nil {
		t.Fatal("corrupt limits accepted")
	}
	var schema int
	g02Check(t, s.db.QueryRow("PRAGMA user_version").Scan(&schema))
	if schema != 58 {
		t.Fatal("migration did not roll back")
	}
}
