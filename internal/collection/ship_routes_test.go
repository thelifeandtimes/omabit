package collection

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestShipRoutesCollectorsShareRunnerSettingsAndVerifyReplies(t *testing.T) {
	for _, kind := range []string{"ok", "project", "old-runner", "changed-runtime", "changed-binding", "changed-setting", "changed-host", "stale-read", "changed-save"} {
		t.Run(kind, func(t *testing.T) {
			r := registry(t)
			other := registry(t)
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			target := apppolicy.Target{InstanceID: "alpha"}
			if kind == "project" {
				target.ProjectID = "network"
				target.Generation = 2
				target.Creator = "resume-2"
			}
			binding := shiproute.Binding{RunnerID: "runner-fixture", Target: target, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			saved := shiproute.DefaultRecord(binding)
			var mu sync.Mutex
			puts := 0
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature}
					if kind == "old-runner" {
						features = features[:1]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Features: features})
				case protocol.AppPolicyPath:
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target, Ship: "~zod", Label: "alpha", Ready: true}}})
				case "/v1/snapshot":
					encode(protocol.Snapshot{RunnerID: "runner-fixture", Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha", ProjectID: target.ProjectID, ConfigRevision: 3, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.ShipRoutesPath:
					var selection protocol.ShipRouteSelection
					if q.Method == "PUT" {
						puts++
						var u protocol.ShipRoutesUpdate
						if json.NewDecoder(q.Body).Decode(&u) != nil || u.Validate() != nil {
							t.Error("bad update")
							w.WriteHeader(400)
							return
						}
						selection = u.Selection
						if u.Update.ExpectedRevision != saved.Revision {
							w.WriteHeader(409)
							encode(map[string]any{"error": protocol.APIError{Code: "stale", Message: "refresh"}})
							return
						}
						saved = shiproute.Record{Version: 1, Binding: binding, Revision: saved.Revision + 1, Settings: u.Update.Settings, Actor: "fixture", UpdatedAt: time.Now().UTC()}
					} else if q.Method == "POST" {
						if json.NewDecoder(q.Body).Decode(&selection) != nil {
							t.Error("bad selection")
						}
					} else {
						t.Error("unexpected method")
						w.WriteHeader(400)
						return
					}
					if selection.Target != target || selection.ConfigRevision != 3 || selection.RuntimeID != strings.Repeat("a", 64) {
						t.Error("selection changed", selection)
					}
					v := protocol.ShipRoutes{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Selection: selection, Record: saved, ReadAt: time.Now().UTC()}
					var routeErr error
					v.Registry, routeErr = v.Record.Settings.Effective(binding, saved.Revision, []shiproute.Route{{ID: "managed-http", Origin: "http://127.0.0.1:18080", Network: "local", Scope: binding.RunnerID, Source: "managed", Enabled: true}})
					if routeErr != nil {
						t.Error(routeErr)
					}
					switch kind {
					case "changed-runtime":
						v.Selection.RuntimeID = strings.Repeat("b", 64)
					case "changed-binding":
						v.Registry.Binding.Continuity = strings.Repeat("d", 64)
					case "changed-setting":
						v.Registry.Routes[0].Enabled = false
					case "changed-host":
						v.Registry.Routes[0].Scope = "runner-other"
					case "stale-read":
						v.ReadAt = v.ReadAt.Add(-time.Hour)
					case "changed-save":
						if q.Method == "PUT" {
							v.Record.Settings = shiproute.DefaultSettings()
							v.Registry, _ = v.Record.Settings.Effective(binding, saved.Revision, nil)
						}
					}
					encode(v)
				default:
					t.Error("unexpected route", q.URL.Path)
					w.WriteHeader(404)
				}
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			for _, reg := range []*Registry{r, other} {
				if _, err = reg.Enroll(context.Background(), "fixture", socket); err != nil {
					t.Fatal(err)
				}
			}
			v, err := r.ShipRoutes(context.Background(), "fixture", binding.RunnerID, "alpha")
			if kind != "ok" && kind != "project" && kind != "changed-save" {
				if err == nil {
					t.Fatal("unsafe response accepted", kind)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			u := protocol.ShipRoutesUpdate{Selection: v.Selection, Update: shiproute.Update{Binding: v.Record.Binding, Settings: shiproute.DefaultSettings()}}
			u.Update.Settings.DisabledAutomatic = []string{"managed-http"}
			_, err = r.PutShipRoutes(context.Background(), "fixture", binding.RunnerID, u)
			if kind == "changed-save" {
				if err == nil {
					t.Fatal("changed save receipt accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			v, err = other.ShipRoutes(context.Background(), "fixture", binding.RunnerID, "alpha")
			if err != nil || v.Record.Revision != 1 || v.Registry.Routes[0].Enabled {
				t.Fatal("other desktop missed saved choice", v, err)
			}
			if _, err = other.PutShipRoutes(context.Background(), "fixture", binding.RunnerID, u); err == nil {
				t.Fatal("stale collector edit succeeded")
			}
			mu.Lock()
			if puts != 2 {
				t.Error("write replayed", puts)
			}
			mu.Unlock()
		})
	}
}
