package managed

import (
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestFakeRegistrationKeepsIdentityInstanceAndPierIndependent(t *testing.T) {
	s := testStore(t)
	req := registration("local-bus", "register-bus", 18088)
	req.FakeShip = "bus"
	req.PierName = "bus-data"
	op, replayed, err := s.register("operator", req)
	if err != nil || replayed {
		t.Fatal(op, err)
	}
	v, err := loadInstance(s.db, req.ID)
	if err != nil || v.Public.ID != "local-bus" || v.Public.Ship != "~bus" || v.Public.PierName != "pier" || v.Runtime.Fake == nil || v.Runtime.Fake.Ship != "~bus" || v.Runtime.Storage.Path != "fakeships/bus-data/pier" {
		t.Fatal(v, err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal("missing downgrade guard", version, err)
	}
	again, replayed, err := s.register("operator", req)
	if err != nil || !replayed || again.ID != op.ID {
		t.Fatal("registration replay", again, err)
	}
	req.FakeShip = "~nec"
	if _, _, err := s.register("operator", req); err == nil {
		t.Fatal("changed identity reused request")
	}
}

func TestIndependentFakeNamesAreAllowedButPierPortAndConcurrentStartConflictsAreNot(t *testing.T) {
	s := testStore(t)
	for n, name := range []string{"one", "two"} {
		req := registration(name, "register-"+name, uint16(18080+n))
		req.FakeShip = "~zod"
		if _, _, err := s.register("operator", req); err != nil {
			t.Fatal(err)
		}
	}
	a, err := loadInstance(s.db, "one")
	if err != nil {
		t.Fatal(err)
	}
	b, err := loadInstance(s.db, "two")
	if err != nil {
		t.Fatal(err)
	}
	if a.Public.Ship != "~zod" || b.Public.Ship != "~zod" || a.Runtime.ID == b.Runtime.ID || a.Runtime.Storage.Path == b.Runtime.Storage.Path {
		t.Fatal("independent fake registrations were conflated")
	}
	conflicting := registration("three", "same-pier", 18082)
	conflicting.PierPath = a.Runtime.Storage.Path
	_, _, err = s.register("operator", conflicting)
	requireFault(t, err, "pier_path_conflict")
	conflicting = registration("three", "same-port", 18080)
	_, _, err = s.register("operator", conflicting)
	requireFault(t, err, "port_conflict")
	_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "first-start", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "second-start", ExpectedRevision: 1})
	requireFault(t, err, "instance_busy")
}

func TestFakeRegistrationRejectsUnsafeNamesAndCometOptions(t *testing.T) {
	s := testStore(t)
	for _, value := range []string{"~foo", "~bus; exit", "~sampel--palnet"} {
		req := registration("bad", "bad", 18088)
		req.FakeShip = value
		if _, _, err := s.register("operator", req); err == nil {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"../outside", "/outside", ".", "a/b"} {
		req := registration("bad", "bad", 18088)
		req.FakeShip = "~bus"
		req.PierName = value
		if _, _, err := s.register("operator", req); err == nil {
			t.Fatal(value)
		}
	}
	s.mode = DevelopmentMode
	req := protocol.CreateFakeRequest{ID: "comet", RequestID: "comet", FakeShip: "~bus", Config: protocol.Configuration{HTTPPort: 18088, AmesPort: 34343, Loom: 31}}
	if _, _, err := s.registerKind("operator", req, nil, true); err == nil {
		t.Fatal("comet accepted fake identity")
	}
	all, err := instances(s.db)
	if err != nil || len(all) != 0 {
		t.Fatal("invalid registration persisted", all, err)
	}
}
