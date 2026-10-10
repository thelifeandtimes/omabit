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
