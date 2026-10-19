package managed

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestSchema21EngineEvidenceAfterOriginal20MigrationAndReopen(t *testing.T) {
	var original g02Fixture
	for _, f := range g02Fixtures(t) {
		if f.Name == "rc12" {
			original = f
		}
	}
	if original.Schema != 20 {
		t.Fatal("require the original published schema20 writer")
	}
	dir := g02Materialize(t, original)
	s := g02Open(t, dir)
	g02AssertRows(t, s.db, original, nil)
	v, err := loadInstance(s.db, "stopped")
	g02Check(t, err)
	legacy, err := json.Marshal(v.Runtime.Runtime.Containers)
	g02Check(t, err)
	if len(v.Runtime.Runtime.Containers) == 0 || bytes.Contains(legacy, []byte(`"engine_version"`)) {
		t.Fatal("original containers must omit per-launch engine evidence")
	}
	provisioningVersion := v.Runtime.Runtime.EngineVersion
	// Exercise current persistence after a real migration, using only synthetic
	// resource observations. Actual daemon evidence capture has runtimequalify tests.
	s.portAvailable = func(string, uint16) bool { return true }
	req := protocol.OperationRequest{Kind: "start", RequestID: "schema21-engine-start", ExpectedRevision: v.Public.ConfigRevision}
	op, hit, err := s.accept("schema21-engine-actor", "stopped", req)
	g02Check(t, err)
	if hit {
		t.Fatal("new start unexpectedly replayed")
	}
	op, v, err = s.beginOperation(op.ID)
	g02Check(t, err)
	const observedVersion = "29.9.1+dfsg1"
	v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{
		ID: strings.Repeat("f", 64), Name: "synthetic-schema21-engine", Kind: "fake-resume",
		CreateDispatched: true, EngineVersion: observedVersion,
	})
	g02Check(t, s.runtimeUpdate("stopped", op.ID, v.Runtime))
	g02Check(t, s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, nil, false))
	readBodies := func(s *store) [][]byte {
		var body, receipt, runtime []byte
		g02Check(t, s.db.QueryRow("SELECT body FROM instances WHERE id='stopped'").Scan(&body))
		g02Check(t, s.db.QueryRow("SELECT body,runtime FROM operations WHERE id=?", op.ID).Scan(&receipt, &runtime))
		return [][]byte{body, receipt, runtime}
	}
	before := readBodies(s)
	g02Check(t, s.close())
	for pass := 0; pass < 3; pass++ {
		s := g02Open(t, dir)
		got, err := loadInstance(s.db, "stopped")
		g02Check(t, err)
		containers := got.Runtime.Runtime.Containers
		if len(containers) != len(v.Runtime.Runtime.Containers) || containers[len(containers)-1].EngineVersion != observedVersion || got.Runtime.Runtime.EngineVersion != provisioningVersion {
			t.Fatal("reopen lost or relabeled provisioning/launch evidence")
		}
		old, err := json.Marshal(containers[:len(containers)-1])
		g02Check(t, err)
		if !bytes.Equal(old, legacy) {
			t.Fatal("current writer changed legacy containers or invented engine evidence")
		}
		replayed, hit, err := s.accept("schema21-engine-actor", "stopped", req)
		g02Check(t, err)
		if !hit || replayed.ID != op.ID || replayed.Status != "succeeded" || !reflect.DeepEqual(readBodies(s), before) {
			t.Fatal("repeated open/replay changed instance or operation runtime/receipt bytes")
		}
		var journal runtimequalify.ManagedState
		g02Check(t, json.Unmarshal(before[2], &journal))
		if !reflect.DeepEqual(journal.Runtime.Containers, containers) {
			t.Fatal("operation journal lost per-container engine evidence")
		}
		g02Check(t, s.close())
	}
	g02Sidecars(t, dir, original)
}
