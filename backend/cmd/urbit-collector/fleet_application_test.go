package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/applicationauth"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestFleetApplicationCLISelectionIssuanceAndRecovery(t *testing.T) {
	if os.Getenv("OMABIT_FLEET_CLI_FIXTURE") != "1" {
		t.Skip("requires scripts/test-fleet-connect.py offline credential transport")
	}
	marker, err := os.ReadFile("/omabit-fleet-cli-fixture")
	if err != nil || string(marker) != "offline synthetic credential transport\n" {
		t.Fatal("missing private fixture")
	}
	for _, kind := range []string{"ok", "lost-login", "lost-delivery", "secret-store-failure"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			home := t.TempDir()
			t.Setenv("HOME", home)
			for _, key := range []string{"XDG_CONFIG_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"} {
				t.Setenv(key, "")
			}
			root := filepath.Join(home, ".config/omarchy/plugins/fixture.auth-lab")
			if err := os.MkdirAll(root, 0700); err != nil {
				t.Fatal(err)
			}
			script := `import json,sys,os
from pathlib import Path
v=json.load(sys.stdin)
reply={k:v[k] for k in ('version','request_id','connection_id','plugin_id','profile')}
receipt=Path(os.environ['HOME'])/'fixture-delivered'
if sys.argv[1:]==['--reconcile']:
    receipt.unlink(missing_ok=True)
    reply.update(status='reconciled',outcome='removed')
else:
    assert v['session']['cookie_value']=='fixture-fleet-cookie'
    receipt.write_text(v['request_id'])
    if (Path(os.environ['HOME'])/'fail-delivery').exists():sys.exit(2)
    reply.update(status='accepted')
print(json.dumps(reply))
`
			declaration, _ := json.Marshal(applicationauth.Declaration{Version: 1, Runtime: "python3", Authority: applicationauth.OwnerAuthority, Entry: "receiver.py", Files: []string{"manifest.json", "application-connection.json", "receiver.py"}})
			for name, data := range map[string]string{"manifest.json": `{"id":"fixture.auth-lab","name":"Fleet fixture"}`, "application-profile.json": `{"version":1,"profile":"auth-lab"}`, "application-connection.json": string(declaration), "receiver.py": script} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "lost-delivery" {
				if err := os.WriteFile(filepath.Join(home, "fail-delivery"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "secret-store-failure" {
				if err := os.WriteFile(filepath.Join(home, "fail-secret-store"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var login, logout, codes, third, runnerCalls atomic.Int32
			var active atomic.Bool
			wrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.URL.Path != "/~/host" || q.Header.Get("Cookie") != "" {
					t.Error("credential sent to wrong ship")
				}
				fmt.Fprint(w, "~nec")
			}))
			defer wrong.Close()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if q.URL.Path == "/~/host" {
					fmt.Fprint(w, "~zod")
					return
				}
				cookie, _ := q.Cookie("urbauth-~zod")
				if q.URL.Path == "/~/login" {
					if cookie != nil || q.Method != "POST" {
						t.Error("login inherited session")
					}
					if q.ParseForm() != nil || q.Form.Get("password") != "lidlut-tabwed-pillex-ridrup" {
						t.Error("wrong credential")
					}
					login.Add(1)
					active.Store(true)
					if kind == "lost-login" {
						conn, _, e := w.(http.Hijacker).Hijack()
						if e != nil {
							t.Error(e)
						} else {
							conn.Close()
						}
						return
					}
					http.SetCookie(w, &http.Cookie{Name: "urbauth-~zod", Value: "fixture-fleet-cookie", Path: "/", MaxAge: 3600})
					return
				}
				if cookie == nil || cookie.Value != "fixture-fleet-cookie" || !active.Load() {
					w.WriteHeader(403)
					return
				}
				switch q.URL.Path {
				case "/~/name":
					fmt.Fprint(w, "~zod")
				case "/~/logout":
					logout.Add(1)
					active.Store(false)
					w.Header().Set("Location", "/~/login")
					w.WriteHeader(303)
				default:
					t.Error("unexpected Eyre call", q.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) { third.Add(1); w.WriteHeader(503) }))
			defer other.Close()
			binding := shiproute.Binding{RunnerID: "runner-fixture", Target: apppolicy.Target{InstanceID: "alpha"}, Ship: "~zod", Continuity: strings.Repeat("c", 64)}
			selection := protocol.ShipRouteSelection{Target: binding.Target, ConfigRevision: 3, RuntimeID: strings.Repeat("a", 64)}
			record := shiproute.DefaultRecord(binding)
			record.Revision = 1
			record.Actor = "fixture"
			record.UpdatedAt = time.Now().UTC()
			for i, origin := range []string{wrong.URL, server.URL, other.URL} {
				record.Settings.Routes = append(record.Settings.Routes, shiproute.Route{ID: []string{"a-wrong", "b-good", "z-other"}[i], Origin: origin, Network: "local", Scope: binding.RunnerID, Source: "manual", Enabled: true})
			}
			routes, _ := record.Settings.Effective(binding, 1, nil)
			socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
				runnerCalls.Add(1)
				switch q.URL.Path {
				case "/hello":
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: binding.RunnerID, Mode: "simulation", AccessRole: "operator", Features: []string{protocol.AppPolicyFeature, protocol.ShipRoutesFeature, protocol.ShipRouteCodeFeature}})
				case protocol.AppPolicyPath:
					reply(t, w, protocol.AppPolicyStatus{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: binding.Target, Ship: binding.Ship, Ready: true}}})
				case "/v1/snapshot":
					reply(t, w, protocol.Snapshot{RunnerID: binding.RunnerID, Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha", ConfigRevision: 3, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: selection.RuntimeID}}}})
				case protocol.ShipRoutesPath:
					reply(t, w, protocol.ShipRoutes{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", Selection: selection, Record: record, Registry: routes, ReadAt: time.Now().UTC()})
				case protocol.ShipRouteCodePath:
					var request protocol.ShipRouteCodeRequest
					if json.NewDecoder(q.Body).Decode(&request) != nil || request.Binding != binding || request.Selection != selection || request.RegistryRevision != 1 || request.Route != record.Settings.Routes[1] {
						t.Error("wrong selected route")
						w.WriteHeader(400)
						return
					}
					codes.Add(1)
					reply(t, w, protocol.ShipRouteCode{Version: 1, RunnerID: binding.RunnerID, Mode: "simulation", Request: request, Code: "lidlut-tabwed-pillex-ridrup"})
				default:
					t.Error("unexpected runner request", q.URL.Path)
					w.WriteHeader(404)
				}
			})
			dir := filepath.Join(t.TempDir(), "collector")
			r, e := collection.Open(dir)
			if e != nil {
				t.Fatal(e)
			}
			if _, err := r.Enroll(ctx, "fixture", socket); err != nil {
				t.Fatal(err)
			}
			invoke := func(want int, args ...string) string {
				t.Helper()
				var out, stderr bytes.Buffer
				code := run(ctx, append([]string{"--config-dir", dir, "--format", "json"}, args...), &out, &stderr)
				if code != want {
					t.Fatalf("%v exit=%d want=%d out=%s err=%s", args, code, want, &out, &stderr)
				}
				for _, s := range []string{"fixture-fleet-cookie", "lidlut-tabwed-pillex-ridrup", "session_ref"} {
					if strings.Contains(out.String()+stderr.String(), s) {
						t.Fatal("credential in CLI output")
					}
				}
				return out.String()
			}
			var plugins []applicationauth.InstalledPlugin
			if err := json.Unmarshal([]byte(invoke(0, "app-plugins")), &plugins); err != nil || len(plugins) != 1 {
				t.Fatal(err, plugins)
			}
			u := collection.FleetApplicationRequest{Version: 1, Binding: binding, Target: plugins[0].Target, GrantRevision: 1}
			raw, _ := json.Marshal(u)
			id := strings.Repeat("a", 32)
			pin := []string{"--runner", "fixture", "--expected-runner-id", binding.RunnerID}
			args := append(append([]string{}, pin...), "app-fleet-connect", "--request-json", string(raw), "--request-id", id)
			before := runnerCalls.Load()
			if !strings.Contains(invoke(1, args...), "permission_required") || runnerCalls.Load() != before {
				t.Fatal("default deny contacted runner")
			}
			allow := true
			grant, _ := json.Marshal(collection.ApplicationShipApprovalRequest{Version: 1, Binding: binding, Target: u.Target, Allowed: &allow, Selection: &selection})
			invoke(0, append(append([]string{}, pin...), "app-ship-approve", "--request-json", string(grant), "--allow-owner-session")...)
			want := 1
			if kind == "ok" {
				want = 0
			}
			var result applicationResult
			if err := json.Unmarshal([]byte(invoke(want, args...)), &result); err != nil || result.Connection == nil {
				t.Fatal(err, result)
			}
			current := result.Connection
			if len(result.RouteAttempts) != 2 || result.RouteAttempts[0].Status != "wrong-ship" || result.RouteAttempts[1].Status != "verified" || current.Fleet.Request.Binding != binding || current.Origin != server.URL || codes.Load() != 1 || login.Load() != 1 || third.Load() != 0 {
				t.Fatal("selection/issuance mismatch", result, codes.Load(), login.Load(), third.Load())
			}
			before = runnerCalls.Load()
			invoke(1, args...)
			if runnerCalls.Load() != before || login.Load() != 1 {
				t.Fatal("duplicate issued/reprobed")
			}
			second := append([]string{}, args...)
			second[len(second)-1] = strings.Repeat("b", 32)
			if kind != "secret-store-failure" {
				invoke(1, second...)
				if login.Load() != 1 {
					t.Fatal("pending origin switched")
				}
			}
			if kind == "ok" {
				if current.Phase != "connected" || !active.Load() {
					t.Fatal(current)
				}
				// Revoke standing permission before using ordinary cleanup commands.
				allow = false
				grant, _ = json.Marshal(collection.ApplicationShipApprovalRequest{Version: 1, ExpectedRevision: 1, Binding: binding, Target: u.Target, Allowed: &allow})
				invoke(0, "app-ship-approve", "--request-json", string(grant))
				if err := json.Unmarshal([]byte(invoke(0, "app-disconnect", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))), &result); err != nil {
					t.Fatal(err)
				}
				current = result.Connection
			}
			if kind == "ok" || kind == "lost-delivery" {
				if current.Phase != "closed" || !current.ReceiverReconciliation || active.Load() || logout.Load() != 1 {
					t.Fatal(current)
				}
				if err := json.Unmarshal([]byte(invoke(0, "app-reconcile", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))), &result); err != nil {
					t.Fatal(err)
				}
				current = result.Connection
				if current.ReceiverReconciliation {
					t.Fatal("receiver cleanup not recorded")
				}
				invoke(0, "app-archive", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))
				if !strings.Contains(invoke(1, args...), `"code": "archived"`) {
					t.Fatal("archived request replay")
				}
			} else if kind == "lost-login" {
				if current.Phase != "unresolved" || !active.Load() || logout.Load() != 0 {
					t.Fatal(current)
				}
				invoke(1, "app-recover", id, "--expected-revision", strconv.FormatUint(current.Revision, 10))
			} else if current.Phase != "closed" || current.ReceiverReconciliation || active.Load() || logout.Load() != 1 {
				t.Fatal("storage failure did not clean session", current)
			}
			for _, name := range []string{"ships.json", "credentials"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
					t.Fatal("created parent or plaintext fallback", name)
				}
			}
			if login.Load() != 1 || codes.Load() != 1 || third.Load() != 0 {
				t.Fatal("authentication fallback/replay")
			}
		})
	}
}

func TestFleetApplicationCLIInvalidInputAndHelpHaveNoEffects(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	for _, args := range [][]string{{"app-fleet-connect"}, {"app-fleet-connect", "--url", "https://arbitrary.example"}, {"--runner", "fixture", "--expected-runner-id", "runner-fixture", "app-fleet-connect", "--request-id", strings.Repeat("a", 32), "--request-json", "{}"}} {
		var out, err bytes.Buffer
		if code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &err); code != 2 {
			t.Fatal(code, &out, &err)
		}
	}
	var out, err bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", dir, "app-fleet-connect", "--help"}, &out, &err); code != 0 || !strings.Contains(out.String(), "standing permission") {
		t.Fatal(code, &out, &err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("invalid input created state")
	}
}
