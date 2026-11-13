//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func (d deskRuntimeFixture) RouteCode(ctx context.Context, expected shiproute.Runtime) (string, error) {
	d.b.calls.Add(1)
	if expected != (shiproute.Runtime{Ship: "~zod", Continuity: strings.Repeat("c", 64), Origin: "http://127.0.0.1:18080"}) {
		return "", errors.New("PRIVATE bad binding")
	}
	if d.b.scenario == "code-error" {
		return "", errors.New("PRIVATE lidlut-tabwed-pillex-ridrup")
	}
	if d.b.scenario == "invalid-code" {
		return "PRIVATE-NOT-A-CODE", nil
	}
	return "lidlut-tabwed-pillex-ridrup", nil
}

func TestShipRouteCodeAPIExactEnabledRouteAndRedactedAudit(t *testing.T) {
	for _, kind := range []string{"ok", "manual", "observer", "method", "wrong-runner", "replacement", "revision", "runtime", "route-origin", "route-id", "disabled", "manual-shadow", "code-error", "invalid-code"} {
		t.Run(kind, func(t *testing.T) {
			b := &deskBackendFixture{}
			r, intent := deskServerFixture(t, b)
			selection := protocol.ShipRouteSelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
			out := deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
			if out.Code != 200 {
				t.Fatal(out.Code, out.Body.String())
			}
			var view protocol.ShipRoutes
			g02Check(t, json.Unmarshal(out.Body.Bytes(), &view))
			u := protocol.ShipRouteCodeRequest{Selection: selection, Binding: view.Record.Binding, RegistryRevision: view.Record.Revision, Route: view.Registry.Routes[0]}
			if kind == "manual" || kind == "disabled" || kind == "manual-shadow" {
				settings := shiproute.DefaultSettings()
				if kind == "manual" {
					settings.Routes = []shiproute.Route{{ID: "website", Origin: "https://example.com", Network: "public", Source: "manual", Enabled: true}}
				}
				if kind == "disabled" {
					settings.DisabledAutomatic = []string{"managed-http"}
				}
				if kind == "manual-shadow" {
					shadow := u.Route
					shadow.ID = "override"
					shadow.Source = "manual"
					shadow.Enabled = false
					settings.Routes = []shiproute.Route{shadow}
				}
				update := protocol.ShipRoutesUpdate{Selection: selection, Update: shiproute.Update{Binding: u.Binding, Settings: settings}}
				out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, update)
				if out.Code != 200 {
					t.Fatal(out.Code, out.Body.String())
				}
				u.RegistryRevision = 1
				if kind == "manual" {
					u.Route = settings.Routes[0]
				}
			}
			switch kind {
			case "wrong-runner":
				u.Binding.RunnerID = "other"
			case "replacement":
				b.scenario = "replacement"
			case "revision":
				u.RegistryRevision++
			case "runtime":
				u.Selection.RuntimeID = strings.Repeat("f", 64)
			case "route-origin":
				u.Route.Origin = "http://127.0.0.1:19999"
			case "route-id":
				u.Route.ID = "different"
			case "code-error", "invalid-code":
				b.scenario = kind
			}
			if kind == "observer" {
				body, _ := json.Marshal(u)
				out = httptest.NewRecorder()
				r.serveShipRouteCode(out, httptest.NewRequest("POST", protocol.ShipRouteCodePath, bytes.NewReader(body)), "observer", "observer")
			} else {
				method := "POST"
				if kind == "method" {
					method = "GET"
				}
				out = deskHTTPRequest(t, r, method, protocol.ShipRouteCodePath, u)
			}
			if kind == "ok" || kind == "manual" {
				var code protocol.ShipRouteCode
				if out.Code != 200 || json.Unmarshal(out.Body.Bytes(), &code) != nil || code.Request != u || code.Code != "lidlut-tabwed-pillex-ridrup" {
					t.Fatal("valid bound code rejected", out.Code)
				}
			} else if out.Code < 400 || strings.Contains(out.Body.String(), "PRIVATE") || strings.Contains(out.Body.String(), "lidlut") {
				t.Fatal("invalid request/diagnostic", kind, out.Code, out.Body.String())
			}
			want := int32(0)
			if kind == "ok" || kind == "manual" || kind == "code-error" || kind == "invalid-code" {
				want = 1
			}
			if b.calls.Load() != want {
				t.Fatal("credential reader reached before admission", kind, b.calls.Load())
			}
			// Audit only contains redacted status. Operations store kind inside JSON.
			rows, err := r.s.db.Query("SELECT body FROM operations WHERE json_extract(body,'$.kind')='login-code'")
			g02Check(t, err)
			defer rows.Close()
			audits := 0
			for rows.Next() {
				var body string
				g02Check(t, rows.Scan(&body))
				audits++
				if strings.Contains(body, "lidlut") || strings.Contains(body, "PRIVATE") {
					t.Fatal("secret retained in audit")
				}
			}
			g02Check(t, rows.Err())
			if audits != int(want) {
				t.Fatal("audit count", audits, want)
			}
		})
	}
}

func TestShipRouteCodeAPIRechecksServeDiscovery(t *testing.T) {
	for _, kind := range []string{"tailnet", "public", "changed", "unavailable", "invalid", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			base := &deskBackendFixture{}
			r, intent := deskServerFixture(t, base)
			b := &shipRouteNetworkFixture{deskBackendFixture: base, t: t, result: shiproute.Discovery{Network: shiproute.NetworkDiscovery{Tailscale: "ready", DNSName: "box.example.ts.net", TailnetSuffix: "example.ts.net", Serve: "tailnet"}, Route: &shiproute.Route{ID: "tailscale-serve", Origin: "https://box.example.ts.net", Network: "tailnet", Scope: "example.ts.net", Source: "tailscale", Enabled: true}}}
			r.b = b
			if kind == "public" {
				b.result.Network.Serve = "public"
				b.result.Route.Network = "public"
				b.result.Route.Scope = ""
			}
			selection := protocol.ShipRouteSelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
			out := deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
			if out.Code != 200 {
				t.Fatal(out.Code)
			}
			var view protocol.ShipRoutes
			g02Check(t, json.Unmarshal(out.Body.Bytes(), &view))
			u := protocol.ShipRouteCodeRequest{Selection: selection, Binding: view.Record.Binding, RegistryRevision: 0, Route: *b.result.Route}
			switch kind {
			case "changed":
				b.result.Route.Origin = "https://box.example.ts.net:8443"
			case "invalid":
				b.result.Route.Origin = "https://other.example.ts.net"
			case "unavailable":
				b.result = shiproute.Discovery{Network: shiproute.NetworkDiscovery{Tailscale: "unavailable"}}
			case "disabled":
				settings := shiproute.DefaultSettings()
				settings.DisabledAutomatic = []string{"tailscale-serve"}
				out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, protocol.ShipRoutesUpdate{Selection: selection, Update: shiproute.Update{Binding: u.Binding, Settings: settings}})
				if out.Code != 200 {
					t.Fatal(out.Code)
				}
				u.RegistryRevision = 1
			}
			out = deskHTTPRequest(t, r, "POST", protocol.ShipRouteCodePath, u)
			if kind == "tailnet" || kind == "public" {
				if out.Code != 200 || base.calls.Load() != 1 {
					t.Fatal("current proxy rejected", out.Code)
				}
			} else if out.Code < 400 || base.calls.Load() != 0 {
				t.Fatal("changed proxy reached credentials", out.Code, base.calls.Load())
			}
		})
	}
}
