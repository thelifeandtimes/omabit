package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func settingsConfig(t *testing.T, loom int, native bool) protocol.Configuration {
	t.Helper()
	var c protocol.Configuration
	if err := json.Unmarshal([]byte(fmt.Sprintf(`{"http_port":18080,"ames_port":34343,"loom":%d,"native_control":%t}`, loom, native)), &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRuntimeSettingsDurableDesiredAndLaunch(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.close() }()
	req := registration("alpha", "one", 18080)
	req.Config = settingsConfig(t, 34, true)
	if _, _, err = s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
		t.Fatal("registration did not propagate runtime settings", v.Runtime)
	}
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	s, err = testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err = loadInstance(s.db, "alpha")
	if err != nil || v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
		t.Fatal("save/load lost settings", err)
	}
	start, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", ExpectedRevision: 1, RequestID: "start"})
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(start.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
		t.Fatal("start lost settings")
	}
	// Model the runtime receipt, including the immutable per-container launch.
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "http-boot", ID: strings.Repeat("a", 64), Loom: 34, NativeControl: true}}
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, nil, false); err != nil {
		t.Fatal(err)
	}
	conf := settingsConfig(t, 32, false)
	got, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: conf})
	if err != nil || got.ConfigRevision != 2 || got.AppliedRevision != 1 {
		t.Fatal("desired/applied revision", got, err)
	}
	snap, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(snap.Instances[0].RuntimeStatus)
	var status map[string]any
	json.Unmarshal(b, &status)
	if status["launch_loom"] != float64(34) || status["launch_native_control"] != true {
		t.Fatal("status relabeled/omitted launch", string(b))
	}
	stop, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "stop", ExpectedRevision: 2, RequestID: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	op, v, err = s.beginOperation(stop.ID)
	if err != nil || v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
		t.Fatal("stop used desired rather than previous runtime", err)
	}
	v.Runtime.Runtime.Containers = nil
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	start, _, err = s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", ExpectedRevision: 2, RequestID: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	_, v, err = s.beginOperation(start.ID)
	if err != nil || v.Runtime.Loom != 32 || v.Runtime.NativeControl {
		t.Fatal("replacement did not use desired settings", v.Runtime, err)
	}
}

func TestRuntimeSettingsSchemaAndLegacyDefaults(t *testing.T) {
	s := testStore(t)
	req := registration("alpha", "old", 18080)
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	// Schema16 has no runtime settings fields. Leave original bytes intact.
	_, err := s.db.Exec(`UPDATE instances SET body=json_remove(body,'$.runtime.loom','$.runtime.native_control'); PRAGMA user_version=16`)
	if err != nil {
		t.Fatal(err)
	}
	var before []byte
	s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&before)
	if err = s.initialize(); err != nil {
		t.Fatal(err)
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != currentSchema {
		t.Fatal("new settings lack downgrade fence", version)
	}
	var after []byte
	s.db.QueryRow("SELECT body FROM instances WHERE id='alpha'").Scan(&after)
	if string(before) != string(after) {
		t.Fatal("migration rewrote legacy launch or desired settings")
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil || protocol.EffectiveLoom(v.Runtime.Loom) != 31 || v.Runtime.NativeControl || v.Public.Config.NativeControl {
		t.Fatal("unsafe legacy defaults", err)
	}
	if err = s.initialize(); err != nil {
		t.Fatal("migration not repeatable", err)
	}
	if _, err = s.db.Exec(fmt.Sprintf("PRAGMA user_version=%d", currentSchema+1)); err != nil {
		t.Fatal(err)
	}
	if err = s.initialize(); err == nil {
		t.Fatal("future schema/downgrade accepted")
	}
}

func TestRuntimeSettingsSocketPathDiscovery(t *testing.T) {
	b := &storageActionBackend{home: privateDir(t)}
	if err := pierstore.Init(b.home); err != nil {
		t.Fatal(err)
	}
	disk := privateDir(t)
	if _, err := pierstore.RegisterRoot(b.home, "disk", disk); err != nil {
		t.Fatal(err)
	}
	s := testStore(t)
	req := registration("alpha", "one", 18080)
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	for _, path := range []string{"fakeships/alpha/pier", "@disk/fakeships/alpha/pier"} {
		for _, enabled := range []bool{false, true} {
			v, err := loadInstance(s.db, "alpha")
			if err != nil {
				t.Fatal(err)
			}
			v.Public.PierPath = path
			v.Runtime.Storage.Path = path
			v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "http-boot", ID: strings.Repeat("a", 64), Loom: 34, NativeControl: enabled}}
			saveRecoveryFixture(t, s, v)
			call := httptest.NewRequest("GET", "/v1/snapshot", nil)
			call.Header.Set(protocol.Header, "1")
			call = call.WithContext(context.WithValue(call.Context(), peerKey{}, r.uid))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, call)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			var snap protocol.Snapshot
			if err = json.Unmarshal(w.Body.Bytes(), &snap); err != nil {
				t.Fatal(err)
			}
			expected := ""
			if enabled {
				expected = filepath.Join(pierstore.Root(b.home), "fakeships/alpha/pier/.urb/conn.sock")
				if strings.HasPrefix(path, "@disk/") {
					expected = filepath.Join(disk, "fakeships/alpha/pier/.urb/conn.sock")
				}
			}
			if got := snap.Instances[0].RuntimeStatus.NativeControlSocket; got != expected {
				t.Fatal("host socket path discovery", got, expected)
			}
		}
	}
}

func TestRuntimeSettingsLivenetRegistrationAndConfigure(t *testing.T) {
	for _, ship := range []string{"~zod", "~marzod", "~sampel-palnet", "~dibpen-fogweg-dozzod-sarlev"} {
		s := testStore(t)
		s.mode = DevelopmentMode
		req := protocol.CreateShipRequest{Ship: ship, RequestID: "native-opt-in", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true, Config: settingsConfig(t, 34, true)}
		op, _, err := s.registerShip("actor", req)
		if err != nil {
			t.Fatal("loom34 livenet registration refused", ship, err)
		}
		v, err := loadInstance(s.db, op.InstanceID)
		if err != nil || v.Runtime.Loom != 34 || !v.Runtime.NativeControl {
			t.Fatal("livenet settings lost", err)
		}
		config := v.Public.Config
		config.NativeControl = true
		if _, err := s.configure("actor", op.InstanceID, protocol.ConfigureRequest{ExpectedRevision: 1, Config: config}); err != nil {
			t.Fatal("native livenet configure refused", ship, err)
		}
		start, _, err := s.accept("actor", op.InstanceID, protocol.OperationRequest{Kind: "start", ExpectedRevision: 1, RequestID: "start"})
		if err != nil {
			t.Fatal(err)
		}
		_, running, err := s.beginOperation(start.ID)
		if err != nil || !running.Runtime.NativeControl {
			t.Fatal("work admission lost native control", err)
		}
	}
	s := testStore(t)
	s.mode = DevelopmentMode
	req := registration("comet", "comet", 18080)
	req.Config.Loom, req.Config.NativeControl = 34, true
	if _, _, err := s.registerKind("actor", req, nil, true); err != nil {
		t.Fatal(err)
	}
	req.Config.NativeControl = true
	if _, err := s.configure("actor", req.ID, protocol.ConfigureRequest{ExpectedRevision: 1, Config: req.Config}); err != nil {
		t.Fatal("comet configure refused native control", err)
	}
}

func TestRuntimeSettingsPinnedLoomRange(t *testing.T) {
	for loom := 0; loom <= 255; loom++ {
		c := settingsConfig(t, loom, false)
		want := loom >= 30 && loom <= 34
		if err := validateConfig(c); (err == nil) != want {
			t.Errorf("fake loom %d accepted=%t want=%t", loom, err == nil, want)
		}
		if err := validateOwnedConfig(c); (err == nil) != want {
			t.Errorf("owned loom %d accepted=%t want=%t", loom, err == nil, want)
		}
	}
}

func TestRuntimeSettingsNativeOptInRoundTripAllShips(t *testing.T) {
	c := settingsConfig(t, 31, true)
	data, _ := json.Marshal(c)
	var wire map[string]any
	json.Unmarshal(data, &wire)
	if wire["native_control"] != true {
		t.Fatal("explicit native control lost in configuration JSON", string(data))
	}
	s := testStore(t)
	req := registration("alpha", "register", 18080)
	req.Config = c
	if _, _, err := s.register("actor", req); err != nil {
		t.Fatal(err)
	}
	snap, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(snap.Instances[0].Config)
	json.Unmarshal(data, &wire)
	if wire["native_control"] != true {
		t.Fatal("native setting not persisted")
	}
	s.mode = DevelopmentMode
	req.ID, req.RequestID, req.Config.HTTPPort = "comet", "comet", 18081
	if _, _, err := s.registerKind("actor", req, nil, true); err != nil {
		t.Fatal("comet refused native control", err)
	}
	if err := validateOwnedConfig(c); err != nil {
		t.Fatal("owned livenet configuration refused native control", err)
	}
}
