package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"reflect"
	"strings"
	"testing"
)

func shipRouteFixture(t *testing.T, s *store) shiproute.Update {
	t.Helper()
	_, _, err := s.register("fixture", registration("alpha", "ship-route-fixture", 18080))
	g02Check(t, err)
	settings := shiproute.DefaultSettings()
	settings.Routes = []shiproute.Route{{ID: "website", Origin: "https://example.com", Network: "public", Source: "manual", Enabled: true}}
	return shiproute.Update{Binding: shiproute.Binding{RunnerID: s.runnerID, Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}, Settings: settings}
}
func TestShipRoutesStoreCASIncarnationAndReopen(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	u := shipRouteFixture(t, s)
	first, err := s.shipRoutes(u.Binding)
	g02Check(t, err)
	if first.Revision != 0 || len(first.Settings.Routes) != 0 {
		t.Fatal("default invented routes")
	}
	saved, err := s.putShipRoutes("desktop-a", u)
	g02Check(t, err)
	if _, err = s.putShipRoutes("desktop-b", u); err == nil {
		t.Fatal("stale edit accepted")
	}
	u.ExpectedRevision = 1
	u.Settings.DisabledAutomatic = []string{"managed-http"}
	current, err := s.putShipRoutes("desktop-b", u)
	g02Check(t, err)
	u.Binding.Continuity = strings.Repeat("d", 64)
	u.ExpectedRevision = 0
	replaced, err := s.shipRoutes(u.Binding)
	g02Check(t, err)
	if replaced.Revision != 0 || len(replaced.Settings.Routes) != 0 {
		t.Fatal("replacement inherited choices")
	}
	if _, err = s.putShipRoutes("desktop-c", u); err != nil {
		t.Fatal(err)
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, err := s.shipRoutes(saved.Binding)
	g02Check(t, err)
	if !reflect.DeepEqual(got, current) {
		t.Fatal("history changed on reopen")
	}
	var n int
	g02Check(t, s.db.QueryRow("SELECT count(*) FROM ship_routes_history").Scan(&n))
	if n != 3 {
		t.Fatal("history lost", n)
	}
}
func TestShipRoutesStoreAtomicAndCorruptionRefusal(t *testing.T) {
	for _, kind := range []string{"history-write", "missing-current", "missing-history", "changed-history", "wrong-binding", "unknown-field", "wrong-runner"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			u := shipRouteFixture(t, s)
			if kind == "history-write" {
				_, err = s.db.Exec("CREATE TRIGGER fail_routes BEFORE INSERT ON ship_routes_history BEGIN SELECT RAISE(ABORT,'fixture'); END")
				g02Check(t, err)
				if _, err = s.putShipRoutes("desktop", u); err == nil {
					t.Fatal("failed history write accepted")
				}
				got, err := s.shipRoutes(u.Binding)
				g02Check(t, err)
				if got.Revision != 0 {
					t.Fatal("partial write")
				}
				g02Check(t, s.close())
				return
			}
			saved, err := s.putShipRoutes("desktop", u)
			g02Check(t, err)
			switch kind {
			case "missing-current":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM ship_routes")
			case "missing-history":
				_, err = s.db.Exec("DELETE FROM ship_routes_history")
			case "changed-history":
				saved.Actor = "changed"
				b, _ := json.Marshal(saved)
				_, err = s.db.Exec("UPDATE ship_routes_history SET body=?", b)
			case "wrong-binding":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("UPDATE ship_routes SET binding='wrong'")
			case "unknown-field":
				_, err = s.db.Exec("UPDATE ship_routes SET body=json_set(body,'$.unknown',true)")
			case "wrong-runner":
				_, err = s.db.Exec("UPDATE meta SET value='runner-another' WHERE key='runner_id'")
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if s, err = testOpenStore(dir); err == nil {
				s.close()
				t.Fatal("corrupt route history accepted")
			}
		})
	}
}
