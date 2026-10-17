package managed

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestNativeAllShipsSchema17PreservesBodiesAndReceipts(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	req := registration("alpha", "one", 18080)
	req.Config = settingsConfig(t, 34, true)
	op, _, err := s.register("actor", req)
	if err != nil {
		t.Fatal(err)
	}
	var before, receipt string
	if err = s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&receipt); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("PRAGMA user_version=17"); err != nil {
		t.Fatal(err)
	}
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		s, err = testOpenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		var after, got string
		var version int
		s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&after)
		s.db.QueryRow("SELECT body FROM operations WHERE id=?", op.ID).Scan(&got)
		s.db.QueryRow("PRAGMA user_version").Scan(&version)
		if version != 18 || before != after || receipt != got {
			s.close()
			t.Fatal("version-only migration lost contract", version)
		}
		replay, hit, err := s.register("actor", req)
		if err != nil || !hit || replay.ID != op.ID {
			t.Fatal("receipt replay lost", err)
		}
		s.close()
	}
}

func TestNativeAllShipsCapability(t *testing.T) {
	s := testStore(t)
	r := &Server{s: s, b: &storageActionBackend{home: privateDir(t)}, uid: uint32(os.Geteuid())}
	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set(protocol.Header, "1")
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var hello protocol.Hello
	if err := json.Unmarshal(w.Body.Bytes(), &hello); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !slices.Contains(hello.Features, "runtime-settings") || !slices.Contains(hello.Features, "native-control-all-ships") {
		t.Fatal("missing capability", w.Code, hello)
	}
}

func TestNativeAllShipsImportedConfiguration(t *testing.T) {
	for _, ship := range []string{"~zod", "~marzod", "~sampel-palnet", "~dibpen-fogweg-dozzod-sarlev", "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"} {
		t.Run(ship, func(t *testing.T) {
			s := testStore(t)
			s.mode = DevelopmentMode
			kind := protocol.ShipClass(ship)
			tier := map[string]string{"galaxy": "galaxies", "star": "stars", "planet": "planets", "moon": "moons", "comet": "comets"}[kind]
			source := pierstore.Candidate{Path: "livenet/" + tier + "/imported/pier", Kind: kind}
			req := protocol.EnrollPierRequest{ID: "imported", Path: source.Path, Ship: ship, HTTPPort: 18080, RequestID: "enroll", ConfirmStopped: true}
			op, err := s.enrollPierSource("actor", req, source, "enroll", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, v, err := s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			if v.Public.Config.NativeControl || v.Runtime.NativeControl {
				t.Fatal("import default enabled native control")
			}
			v.Runtime.Initialized = true
			v.Runtime.Adoption.Uncertain = false
			v.Runtime.Adoption.FinishedAt = &op.CreatedAt
			v.Runtime.Comet.StartedAt, v.Runtime.Comet.FinishedAt = &op.CreatedAt, &op.CreatedAt
			if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
				t.Fatal(err)
			}
			config := v.Public.Config
			config.NativeControl = true
			config.Loom = 34
			updated, err := s.configure("actor", "imported", protocol.ConfigureRequest{ExpectedRevision: 1, Config: config})
			if err != nil {
				t.Fatal(err)
			}
			start, _, err := s.accept("actor", "imported", protocol.OperationRequest{Kind: "start", ExpectedRevision: updated.ConfigRevision, RequestID: "start"})
			if err != nil {
				t.Fatal(err)
			}
			_, v, err = s.beginOperation(start.ID)
			if err != nil || !v.Runtime.NativeControl || v.Runtime.Loom != 34 || v.Runtime.NativeKeyDigest != "" {
				t.Fatal("imported opt-in lost or key introduced", err)
			}
		})
	}
}

func TestNativeAllShipsSocketPaths(t *testing.T) {
	enabled := true
	for _, tc := range []struct{ mode, tier string }{{"fake", "fakeships"}, {"comet", "livenet/comets"}, {"moon", "livenet/moons"}, {"planet", "livenet/planets"}, {"star", "livenet/stars"}, {"galaxy", "livenet/galaxies"}} {
		// Reuse the authenticated catalogue shape, without any pier filesystem reads.
		var snap protocol.Snapshot
		wire := `{"instances":[{"id":"fixture","mode":"` + tc.mode + `","pier_path":"` + tc.tier + `/fixture/pier"}],"piers":{"roots":[{"id":"default","path":"/fixture/piers"}]}}`
		if err := json.Unmarshal([]byte(wire), &snap); err != nil {
			t.Fatal(err)
		}
		snap.Instances[0].RuntimeStatus = &protocol.RuntimeStatus{LaunchNativeControl: &enabled}
		attachNativeControlPaths(&snap)
		if got := snap.Instances[0].RuntimeStatus.NativeControlSocket; got != "/fixture/piers/"+tc.tier+"/fixture/pier/.urb/conn.sock" {
			t.Fatal(tc.mode, "socket discovery lost", got)
		}
	}
}
