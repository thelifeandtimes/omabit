//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"testing"
)

type shipRouteNetworkFixture struct {
	*deskBackendFixture
	result shiproute.Discovery
	reads  int
	t      *testing.T
}

func (b *shipRouteNetworkFixture) DiscoverShipRoutes(ctx context.Context, origin string) shiproute.Discovery {
	if _, ok := ctx.Deadline(); !ok || origin != "http://127.0.0.1:18080" {
		b.t.Error("discovery did not bind current listener")
	}
	b.reads++
	return b.result
}
func TestShipRoutesAPIDiscoversServeAndKeepsDisableAcrossPortChanges(t *testing.T) {
	for _, kind := range []string{"tailnet", "public", "unavailable", "disabled", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			base := &deskBackendFixture{}
			r, intent := deskServerFixture(t, base)
			b := &shipRouteNetworkFixture{deskBackendFixture: base, t: t, result: shiproute.Discovery{Network: shiproute.NetworkDiscovery{Tailscale: "ready", DNSName: "box.example.ts.net", TailnetSuffix: "example.ts.net", Serve: "tailnet"}, Route: &shiproute.Route{ID: "tailscale-serve", Origin: "https://box.example.ts.net", Network: "tailnet", Scope: "example.ts.net", Source: "tailscale", Enabled: true}}}
			r.b = b
			switch kind {
			case "public":
				b.result.Network.Serve = "public"
				b.result.Route.Network = "public"
				b.result.Route.Scope = ""
			case "unavailable":
				b.result = shiproute.Discovery{Network: shiproute.NetworkDiscovery{Tailscale: "unavailable"}}
			case "invalid":
				b.result.Route.Origin = "https://other.example.ts.net"
			}
			selection := protocol.ShipRouteSelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
			out := deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
			if kind == "invalid" {
				if out.Code < 400 {
					t.Fatal("invalid discovery accepted")
				}
				return
			}
			if out.Code != 200 {
				t.Fatal(out.Code, out.Body.String())
			}
			var v protocol.ShipRoutes
			g02Check(t, json.Unmarshal(out.Body.Bytes(), &v))
			g02Check(t, v.Validate())
			want := 2
			if kind == "unavailable" {
				want = 1
			}
			if len(v.Registry.Routes) != want || v.Network == nil {
				t.Fatal("discovery omitted", v)
			}
			if kind == "disabled" {
				u := protocol.ShipRoutesUpdate{Selection: selection, Update: shiproute.Update{Binding: v.Record.Binding, Settings: shiproute.DefaultSettings()}}
				u.Update.Settings.DisabledAutomatic = []string{"tailscale-serve"}
				out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, u)
				if out.Code != 200 {
					t.Fatal(out.Code, out.Body.String())
				}
				b.result.Route.Origin = "https://box.example.ts.net:8443"
				out = deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
				if out.Code != 200 {
					t.Fatal(out.Code, out.Body.String())
				}
				g02Check(t, json.Unmarshal(out.Body.Bytes(), &v))
				g02Check(t, v.Validate())
				if v.Registry.Routes[1].Enabled || v.Registry.Routes[1].Origin != b.result.Route.Origin || v.Record.Revision != 1 {
					t.Fatal("port change lost disable choice", v)
				}
			}
			if b.reads < 1 || base.calls.Load() != 0 {
				t.Fatal("network discovery caused a ship mutation")
			}
		})
	}
}
