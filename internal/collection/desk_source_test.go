package collection

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskSourceCollectionPinsWorkflowAndRuntime(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "unresolved", "stopped", "urgit", "old-runner", "wrong-runner", "wrong-runtime", "wrong-target", "wrong-settings", "wrong-source", "stale", "future", "false-clean", "wrong-count", "unsafe-path"} {
		t.Run(scenario, func(t *testing.T) {
			r := registry(t)
			socket := filepath.Join(t.TempDir(), "runner.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			target := apppolicy.Target{InstanceID: "publisher"}
			if scenario == "project" {
				target.ProjectID = "network"
				target.Generation = 2
				target.Creator = "resume-2"
			}
			var reads atomic.Int32
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature, protocol.DeskSourceFeature}
					if scenario == "old-runner" {
						features = features[:2]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Features: features})
				case protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("policy changed")
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target, Ship: "~zod", Label: "publisher", Ready: scenario != "unresolved"}}})
				case "/v1/snapshot":
					state := "running"
					if scenario == "stopped" {
						state = "stopped"
					}
					encode(protocol.Snapshot{RunnerID: "runner-fixture", Mode: "simulation", Instances: []protocol.Instance{{ID: "publisher", ProjectID: target.ProjectID, ConfigRevision: 3, ObservedState: state, Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.DeskSettingsPath:
					selected := target
					selected.Desk = "my-app"
					var got apppolicy.Target
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&got) != nil || got != selected {
						t.Error("wrong settings read", got)
					}
					setting := deskwork.DefaultSetting(selected)
					setting.Revision = 2
					setting.Actor = "fixture"
					setting.UpdatedAt = time.Now().UTC()
					setting.Desk.SourceName = "source-dir"
					if scenario == "urgit" {
						setting.Desk.Workflow = "urgit"
						setting.Desk.SourceName = ""
					}
					encode(protocol.DeskSetting{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Setting: setting})
				case protocol.DeskSourcePath:
					reads.Add(1)
					var request protocol.DeskSourceRequest
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&request) != nil || request.SettingsRevision != 2 || request.Selection.Target.Ship() != target || request.Selection.Target.Desk != "my-app" || request.Selection.ConfigRevision != 3 || request.Selection.RuntimeID != strings.Repeat("a", 64) {
						t.Error("wrong source request", request)
					}
					out := protocol.DeskSourceStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Request: request, SourceName: "source-dir", ReadAt: time.Now().UTC(), Observation: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, State: "different", Autocommit: "unknown", UnixDigest: strings.Repeat("a", 64), CanonicalUnixDigest: strings.Repeat("b", 64), ClayDigest: strings.Repeat("c", 64), DifferenceCount: 1, UnixChangeCount: 1, Differences: []protocol.SourceDifference{{Path: "app/my-app.hoon", Kind: "only-unix"}}}
					switch scenario {
					case "wrong-runner":
						out.RunnerID = "another"
					case "wrong-runtime":
						out.Request.Selection.RuntimeID = strings.Repeat("b", 64)
					case "wrong-target":
						out.Request.Selection.Target.InstanceID = "another"
					case "wrong-settings":
						out.Request.SettingsRevision++
					case "wrong-source":
						out.SourceName = "another"
					case "stale":
						out.ReadAt = time.Now().Add(-3 * time.Minute)
					case "future":
						out.ReadAt = time.Now().Add(time.Minute)
					case "false-clean":
						out.State = "synchronized"
					case "wrong-count":
						out.DifferenceCount = 2
					case "unsafe-path":
						out.Differences[0].Path = "../etc/passwd"
					}
					encode(out)
				default:
					t.Error("unexpected route", q.URL.Path)
					http.NotFound(w, q)
				}
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			if _, err = r.Enroll(context.Background(), "fixture", socket); err != nil {
				t.Fatal(err)
			}
			out, err := r.InspectDeskSource(context.Background(), "fixture", "runner-fixture", "publisher", "my-app")
			if scenario == "ok" || scenario == "project" {
				if err != nil || out.State != "different" {
					t.Fatal(out, err)
				}
			} else if err == nil {
				t.Fatal("unsafe source status accepted", out)
			}
			if (scenario == "urgit" || scenario == "old-runner" || scenario == "stopped" || scenario == "unresolved") && reads.Load() != 0 {
				t.Fatal("read proceeded without resolved supported workflow")
			}
		})
	}
}
