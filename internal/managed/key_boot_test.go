package managed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestOwnedKeyRegistrationKeepsSecretOutOfJournalAndReplay(t *testing.T) {
	s := testStore(t)
	s.mode = DevelopmentMode
	key := "0w" + strings.Repeat("a", 128)
	s.portAvailable = func(string, uint16) bool { return true }
	req := protocol.CreateShipRequest{ID: "sampel-palnet", Ship: "~sampel-palnet", PierPath: "livenet/planets/sampel-palnet/pier", Key: key, RequestID: "key-once", ConfirmNetwork: true, Config: protocol.Configuration{HTTPPort: 18099, AmesPort: 34343, Loom: 31}}
	op, hit, err := s.registerShip("actor", req)
	if err != nil || hit {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Runtime.NativeShip != req.Ship || v.Public.Mode != "planet" || v.Runtime.NativeKeyDigest == "" {
		t.Fatal(v.Public)
	}
	file := filepath.Join(s.directory, "keys", v.Runtime.ID+".key")
	data, err := os.ReadFile(file)
	if err != nil || string(data) != key {
		t.Fatal("private key staging failed", err)
	}
	i, _ := os.Stat(file)
	if i.Mode().Perm() != 0600 {
		t.Fatal("key file permissions")
	}
	var journal string
	if err := s.db.QueryRow("SELECT body||fingerprint||COALESCE(runtime,'') FROM operations WHERE id=?", op.ID).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	snap, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(snap)
	if strings.Contains(journal, key) || strings.Contains(string(encoded), key) {
		t.Fatal("key leaked into management output/journal")
	}
	again, hit, err := s.registerShip("actor", req)
	if err != nil || !hit || again.ID != op.ID {
		t.Fatal("key replay", err)
	}
	req.Key = "0w" + strings.Repeat("b", 128)
	if _, _, err := s.registerShip("actor", req); err == nil {
		t.Fatal("changed key replay accepted")
	}
}

func TestGeneratedMoonRegistersWithDerivedPathAndAutomaticPorts(t *testing.T) {
	s := testStore(t)
	s.mode = DevelopmentMode
	s.portAvailable = func(string, uint16) bool { return true }
	req := protocol.CreateShipRequest{Ship: "~dibpen-fogweg-dozzod-sarlev", RequestID: "moon", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true}
	op, _, err := s.registerShip("actor", req)
	if err != nil {
		t.Fatal(err)
	}
	if op.InstanceID != "dibpen-fogweg-dozzod-sarlev" {
		t.Fatal(op.InstanceID)
	}
	v, err := loadInstance(s.db, op.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Public.Mode != "moon" || v.Public.PierPath != "livenet/moons/dibpen-fogweg-dozzod-sarlev/dibpen-fogweg-dozzod-sarlev" || v.Public.Config.HTTPPort < 1024 || v.Public.Config.AmesPort < 1024 || v.Public.ObservedState != "uninitialized" {
		t.Fatal(v.Public)
	}
}
