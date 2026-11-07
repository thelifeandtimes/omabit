package shiproute

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func fixture() Registry {
	return Registry{Version: 1, Binding: Binding{RunnerID: "runner-a", Target: apppolicy.Target{InstanceID: "ship-a"}, Ship: "~zod", Continuity: strings.Repeat("a", 64)}, Routes: []Route{
		{ID: "public", Origin: "https://publish.example", Network: "public", Source: "manual", Enabled: true},
		{ID: "local", Origin: "http://localhost:8888", Network: "local", Scope: "host-a", Source: "managed", Enabled: true},
		{ID: "tailnet", Origin: "http://box.example.ts.net:8888", Network: "tailnet", Scope: "example.ts.net", Source: "tailscale", Enabled: true},
	}}
}

func TestCandidateScopeAndPreference(t *testing.T) {
	r := fixture()
	for _, tc := range []struct {
		env  Environment
		want []string
	}{
		{Environment{HostID: "host-a", TailnetSuffix: "example.ts.net"}, []string{"local", "tailnet", "public"}},
		{Environment{HostID: "host-b", TailnetSuffix: "example.ts.net"}, []string{"tailnet", "public"}},
		{Environment{HostID: "host-b", TailnetSuffix: "other.ts.net"}, []string{"public"}},
	} {
		got, err := r.Candidates(tc.env)
		if err != nil {
			t.Fatal(err)
		}
		ids := []string{}
		for _, route := range got {
			ids = append(ids, route.ID)
		}
		if !reflect.DeepEqual(ids, tc.want) {
			t.Fatal(ids, tc.want)
		}
	}
	r.Routes[0].Enabled = false
	got, err := r.Candidates(Environment{})
	if err != nil || len(got) != 0 {
		t.Fatal("disabled public route selected", got, err)
	}
}

func TestFallbackDoesNotTreatWrongShipAsSuccess(t *testing.T) {
	r := fixture()
	called := []string{}
	got, attempts, err := r.Select(context.Background(), Environment{HostID: "host-a", TailnetSuffix: "example.ts.net"}, func(ctx context.Context, origin, ship string) error {
		if ship != "~zod" {
			t.Fatal("ship identity changed")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded probe")
		}
		called = append(called, origin)
		switch len(called) {
		case 1:
			return errors.New("private implementation error must not escape")
		case 2:
			return eyre.ErrIdentity
		default:
			return nil
		}
	})
	if err != nil || got.ID != "public" || len(attempts) != 3 || attempts[0].Status != "unavailable" || attempts[1].Status != "wrong-ship" || attempts[2].Status != "verified" {
		t.Fatal(got, attempts, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, attempts, err = r.Select(ctx, Environment{}, func(context.Context, string, string) error { t.Fatal("probe after cancellation"); return nil })
	if err == nil || len(attempts) != 0 {
		t.Fatal("cancel ignored")
	}
}

func TestRouteRegistryRefusesAmbiguousIdentityAndOrigins(t *testing.T) {
	for _, mutate := range []func(*Registry){
		func(r *Registry) { r.Routes = append(r.Routes, r.Routes[0]) },
		func(r *Registry) { r.Routes[0].Origin = "http://publish.example" },
		func(r *Registry) { r.Routes[0].Origin = "https://example/a" },
		func(r *Registry) { r.Routes[0].Origin = "https://user:secret@example" },
		func(r *Registry) { r.Routes[0].Origin = "https://127.0.0.1" },
		func(r *Registry) { r.Routes[1].Scope = "" },
		func(r *Registry) { r.Routes[1].Origin = "http://remote.example" },
		func(r *Registry) { r.Routes[2].Origin = "http://example.ts.net.attacker.example:8888" },
		func(r *Registry) { r.Binding.Target.ProjectID = "fake-project" },
		func(r *Registry) { r.Binding.Target.Desk = "base" },
		func(r *Registry) { r.Binding.Continuity = "" },
		func(r *Registry) { r.Version = 2 },
	} {
		r := fixture()
		mutate(&r)
		if r.Validate() == nil {
			t.Fatal("invalid registry accepted", r)
		}
	}
}

func TestProbeUsesOnlyUnauthenticatedIdentityAndNeverFollowsRedirect(t *testing.T) {
	hits := 0
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.Write([]byte("~zod")) }))
	defer destination.Close()
	for _, response := range []string{"~zod", "~nec", "redirect"} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/~/host" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
					t.Error("unexpected identity request")
				}
				if response == "redirect" {
					http.Redirect(w, r, destination.URL, http.StatusFound)
					return
				}
				w.Write([]byte(response))
			}))
			defer server.Close()
			err := Probe(context.Background(), server.URL, "~zod")
			if (err == nil) != (response == "~zod") {
				t.Fatal(err)
			}
		})
	}
	if hits != 0 {
		t.Fatal("redirect was followed")
	}
}
