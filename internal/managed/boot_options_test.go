package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestOwnedDefaultsAllocatePortsAtomicallyAndReplay(t *testing.T) {
	s := testStore(t)
	s.mode = DevelopmentMode
	s.portAvailable = func(kind string, port uint16) bool {
		return !(kind == "tcp" && port == 18080) && !(kind == "udp" && port == 34343)
	}
	req := protocol.CreateShipRequest{Ship: "~sampel-palnet", RequestID: "auto", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true}
	op, _, err := s.registerShip("operator", req)
	if err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "sampel-palnet")
	if err != nil {
		t.Fatal(err)
	}
	if v.Public.Config.HTTPPort != 18081 || v.Public.Config.AmesPort != 34344 || v.Public.PierPath != "livenet/planets/sampel-palnet/pier" || v.Runtime.Boot == nil || v.Runtime.Boot.Pill != nil {
		t.Fatal(v.Public)
	}
	s.portAvailable = func(string, uint16) bool { return false }
	again, replay, err := s.registerShip("operator", req)
	if err != nil || !replay || again.ID != op.ID {
		t.Fatal("automatic ports were reallocated on replay", err)
	}
	s.portAvailable = func(string, uint16) bool { return true }
	req.Ship = "~marzod"
	req.RequestID = "other"
	if _, _, err := s.registerShip("operator", req); err != nil {
		t.Fatal(err)
	}
	other, _ := loadInstance(s.db, "marzod")
	if other.Public.Config.HTTPPort == v.Public.Config.HTTPPort || other.Public.Config.AmesPort == v.Public.Config.AmesPort {
		t.Fatal("allocated a reserved port")
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != currentSchema {
		t.Fatal(version)
	}
}

func TestPillUploadIsContentBoundAndInert(t *testing.T) {
	s := testStore(t)
	r := &Server{s: s, ctx: context.Background()}
	body := []byte(strings.Repeat("test-pill", 8192))
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	for _, role := range []string{"observer", "operator", "operator"} {
		req := httptest.NewRequest("POST", protocol.PillUploadPath, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set(protocol.PillHashHeader, hash)
		w := httptest.NewRecorder()
		r.stagePill(w, req, role)
		if role == "operator" && w.Code != 201 {
			t.Fatal(w.Code, w.Body.String())
		}
		if role == "observer" && w.Code == 201 {
			t.Fatal("observer staged executable boot input")
		}
	}
	all, _ := instances(s.db)
	if len(all) != 0 {
		t.Fatal("pill upload registered or started a ship")
	}
}

func TestPillUploadAdmits4GBFramingButStillRequiresMatchingContent(t *testing.T) {
	s := testStore(t)
	r := &Server{s: s, ctx: context.Background()}
	for _, size := range []int64{4_000_000_000, 4_000_000_001} {
		body := bytes.NewReader([]byte("short fixture"))
		req := httptest.NewRequest("POST", protocol.PillUploadPath, body)
		req.ContentLength = size
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set(protocol.PillHashHeader, strings.Repeat("a", 64))
		w := httptest.NewRecorder()
		r.stagePill(w, req, "operator")
		if size == 4_000_000_000 {
			if body.Len() != 0 || w.Code != 409 {
				t.Fatalf("valid 4 GB framing must reach content verification: %d %s", w.Code, w.Body)
			}
		} else if body.Len() == 0 || w.Code != 400 {
			t.Fatalf("oversize framing must be refused before reading: %d %s", w.Code, w.Body)
		}
	}
	all, err := instances(s.db)
	if err != nil || len(all) != 0 {
		t.Fatal("rejected upload changed registrations", err)
	}
}
