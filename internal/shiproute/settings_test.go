package shiproute

import (
	"testing"
	"time"
)

func TestRouteSettingsKeepDisabledDiscoveryAndDistinctIncarnations(t *testing.T) {
	b := fixture().Binding
	manual := Route{ID: "website", Origin: "https://example.com", Network: "public", Source: "manual", Enabled: true}
	automatic := Route{ID: "managed-http", Origin: "http://127.0.0.1:8080", Network: "local", Scope: "runner-host", Source: "managed", Enabled: true}
	s := DefaultSettings()
	s.Routes = []Route{manual}
	s.DisabledAutomatic = []string{"managed-http"}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	first, err := s.Effective(b, 1, []Route{automatic})
	if err != nil {
		t.Fatal(err)
	}
	automatic.Origin = "http://127.0.0.1:8888"
	next, err := s.Effective(b, 1, []Route{automatic})
	if err != nil {
		t.Fatal(err)
	}
	if first.Routes[0].Enabled || next.Routes[0].Enabled || next.Routes[0].Origin == first.Routes[0].Origin || next.Revision != first.Revision {
		t.Fatal("disabled automatic route changed user intent", first, next)
	}
	replacement := b
	replacement.Continuity = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if replacement.Key() == b.Key() {
		t.Fatal("replacement inherited route registry")
	}
	r := Record{Version: 1, Binding: b, Revision: 1, Settings: s, Actor: "fixture", UpdatedAt: time.Now().UTC()}
	if err = r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Revision = 0
	if r.Validate() == nil {
		t.Fatal("authored settings masqueraded as defaults")
	}
	if err = DefaultRecord(b).Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestConfiguredRoutesCannotForgeAutomaticProvenance(t *testing.T) {
	b := fixture().Binding
	for _, r := range []Route{
		{ID: "managed-http", Origin: "https://example.com", Network: "public", Source: "manual", Enabled: true},
		{ID: "manual", Origin: "http://localhost:8080", Network: "local", Scope: "host", Source: "managed", Enabled: true},
	} {
		s := DefaultSettings()
		s.Routes = []Route{r}
		if s.Validate() == nil {
			t.Fatal("forged automatic route accepted")
		}
	}
	s := DefaultSettings()
	s.Routes = []Route{{ID: "explicit", Origin: "http://localhost:8080", Network: "local", Scope: "host", Source: "manual", Enabled: false}}
	auto := Route{ID: "managed-http", Origin: "http://localhost:8080", Network: "local", Scope: "host", Source: "managed", Enabled: true}
	out, err := s.Effective(b, 1, []Route{auto})
	if err != nil || len(out.Routes) != 1 || out.Routes[0].Enabled {
		t.Fatal("automatic route bypassed explicit disabled URL", out, err)
	}
	if _, err = s.Effective(b, 1, []Route{auto, auto}); err == nil {
		t.Fatal("duplicate discovery accepted")
	}
	auto.ID = "tailscale-http"
	if _, err = s.Effective(b, 1, []Route{auto}); err == nil {
		t.Fatal("mismatched discovery ID/source accepted")
	}
}
