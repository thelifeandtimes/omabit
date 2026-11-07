package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestShipRouteCodeClientPinsAndRedactsSecretReplies(t *testing.T) {
	for _, kind := range []string{"ok", "missing-feature", "changed-binding", "changed-route", "changed-runtime", "wrong-runner", "wrong-mode", "invalid-code", "bad-json", "remote-error"} {
		t.Run(kind, func(t *testing.T) {
			binding := shiproute.Binding{RunnerID: "runner", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			u := protocol.ShipRouteCodeRequest{Selection: protocol.ShipRouteSelection{Target: binding.Target, RuntimeID: strings.Repeat("a", 64), ConfigRevision: 1}, Binding: binding, Route: shiproute.Route{ID: "managed-http", Origin: "http://127.0.0.1:18080", Network: "local", Scope: binding.RunnerID, Source: "managed", Enabled: true}}
			var codes atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					features := []string{protocol.ShipRouteCodeFeature}
					if kind == "missing-feature" {
						features = nil
					}
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: features})
					return
				}
				if q.URL.Path != protocol.ShipRouteCodePath {
					t.Error("unexpected path", q.URL.Path)
				}
				codes.Add(1)
				var received protocol.ShipRouteCodeRequest
				if json.NewDecoder(q.Body).Decode(&received) != nil || received != u {
					t.Error("request binding changed")
				}
				v := protocol.ShipRouteCode{Version: 1, RunnerID: "runner", Mode: "simulation", Request: u, Code: "lidlut-tabwed-pillex-ridrup"}
				switch kind {
				case "changed-binding":
					v.Request.Binding.Continuity = strings.Repeat("b", 64)
				case "changed-route":
					v.Request.Route.Origin = "http://127.0.0.1:19999"
				case "changed-runtime":
					v.Request.Selection.RuntimeID = strings.Repeat("b", 64)
				case "wrong-runner":
					v.RunnerID = "other"
				case "wrong-mode":
					v.Mode = "other"
				case "invalid-code":
					v.Code = "PRIVATE-NOT-A-CODE"
				case "bad-json":
					fmt.Fprint(w, "PRIVATE lidlut-tabwed-pillex-ridrup")
					return
				case "remote-error":
					w.WriteHeader(500)
					fmt.Fprint(w, `{"error":{"code":"PRIVATE","message":"lidlut-tabwed-pillex-ridrup"}}`)
					return
				}
				json.NewEncoder(w).Encode(v)
			})
			got, err := c.ShipRouteCode(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, u)
			if kind == "ok" {
				if err != nil || got.Code == "" {
					t.Fatal(err)
				}
			} else if err == nil || got != (protocol.ShipRouteCode{}) || strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "lidlut") {
				t.Fatal("unsafe credential reply", kind)
			}
			want := 1
			if kind == "missing-feature" {
				want = 0
			}
			if codes.Load() != int32(want) {
				t.Fatal("unexpected credential read count", codes.Load(), want)
			}
			if c.httpClient.Timeout != 5*time.Second {
				t.Fatal("changed shared client timeout")
			}
		})
	}
}
