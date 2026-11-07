//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"net/http/httptest"
	"strings"
	"testing"
)

func (d deskRuntimeFixture) RouteIdentity(context.Context) (shiproute.Runtime, error) {
	d.b.reads.Add(1)
	if d.b.scenario == "read-fails" {
		return shiproute.Runtime{}, errors.New("PRIVATE runtime diagnostic")
	}
	continuity := strings.Repeat("c", 64)
	if d.b.scenario == "replacement" {
		continuity = strings.Repeat("d", 64)
	}
	return shiproute.Runtime{Ship: "~zod", Continuity: continuity, Origin: "http://127.0.0.1:18080"}, nil
}
func TestShipRoutesAPIReadsAndCASWritesExactIncarnation(t *testing.T) {
	b := &deskBackendFixture{}
	r, intent := deskServerFixture(t, b)
	selection := protocol.ShipRouteSelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
	out := deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	var v protocol.ShipRoutes
	g02Check(t, json.Unmarshal(out.Body.Bytes(), &v))
	g02Check(t, v.Validate())
	if v.Record.Revision != 0 || len(v.Registry.Routes) != 1 || v.Registry.Routes[0].Scope != r.s.runnerID {
		t.Fatal("bad default discovery", v)
	}
	u := protocol.ShipRoutesUpdate{Selection: selection, Update: shiproute.Update{Binding: v.Record.Binding, Settings: shiproute.DefaultSettings()}}
	u.Update.Settings.Routes = []shiproute.Route{{ID: "website", Origin: "https://example.com", Network: "public", Source: "manual", Enabled: true}}
	out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, u)
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, u)
	if out.Code != 409 {
		t.Fatal("stale edit", out.Code, out.Body.String())
	}
	b.scenario = "replacement"
	out = deskHTTPRequest(t, r, "PUT", protocol.ShipRoutesPath, u)
	if out.Code != 409 {
		t.Fatal("replacement rebound", out.Code)
	}
	out = deskHTTPRequest(t, r, "POST", protocol.ShipRoutesPath, selection)
	if out.Code != 200 {
		t.Fatal(out.Code)
	}
	g02Check(t, json.Unmarshal(out.Body.Bytes(), &v))
	if v.Record.Revision != 0 || len(v.Record.Settings.Routes) != 0 {
		t.Fatal("replacement inherited URLs")
	}
	if b.calls.Load() != 0 {
		t.Fatal("URL configuration changed ship")
	}
}
func TestShipRoutesAPIRefusesUnreadyUnauthorizedAndChangedSelections(t *testing.T) {
	for _, kind := range []string{"observer", "method", "revision", "container", "read-fails", "runtime-changed", "active-operation"} {
		t.Run(kind, func(t *testing.T) {
			b := &deskBackendFixture{scenario: kind}
			r, intent := deskServerFixture(t, b)
			selection := protocol.ShipRouteSelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
			switch kind {
			case "revision":
				selection.ConfigRevision++
			case "container":
				selection.RuntimeID = strings.Repeat("f", 64)
			case "active-operation":
				v, err := loadInstance(r.s.db, "alpha")
				g02Check(t, err)
				v.Public.ActiveOperationID = "busy"
				saveRecoveryFixture(t, r.s, v)
			}
			var out *httptest.ResponseRecorder
			if kind == "observer" {
				body, _ := json.Marshal(selection)
				out = httptest.NewRecorder()
				r.serveShipRoutes(out, httptest.NewRequest("POST", protocol.ShipRoutesPath, bytes.NewReader(body)), "observer", "observer")
			} else {
				method := "POST"
				if kind == "method" {
					method = "DELETE"
				}
				out = deskHTTPRequest(t, r, method, protocol.ShipRoutesPath, selection)
			}
			if out.Code < 400 || strings.Contains(out.Body.String(), "PRIVATE") {
				t.Fatal("unexpected result", out.Code, out.Body.String())
			}
			if b.calls.Load() != 0 {
				t.Fatal("ship effect during URL read")
			}
		})
	}
}
