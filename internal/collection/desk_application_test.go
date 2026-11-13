package collection

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskApplicationChecksIdentityFreshnessAndSeparateStages(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "old-runner", "wrong-runner", "wrong-runtime", "wrong-generation", "wrong-config", "stale", "future", "nil-agents", "wrong-revision", "invalid-stage", "policy-changed", "stopped", "unresolved"} {
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
			if scenario == "project" || scenario == "wrong-generation" {
				target.ProjectID = "network"
				target.Generation = 2
				target.Creator = "resume-2"
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.DeskApplicationFeature}
					if scenario == "old-runner" {
						features = features[:1]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Features: features})
				case protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("policy write attempted")
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target, Ship: "~zod", Label: "publisher", Ready: scenario != "unresolved"}}})
				case "/v1/snapshot":
					state := "running"
					if scenario == "stopped" {
						state = "stopped"
					}
					encode(protocol.Snapshot{RunnerID: "runner-fixture", Mode: "simulation", Instances: []protocol.Instance{{ID: "publisher", ProjectID: target.ProjectID, ConfigRevision: 3, ObservedState: state, Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.DeskApplicationPath:
					var selected protocol.DeskSelection
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&selected) != nil || selected.Target.Ship() != target || selected.Target.Desk != "example" || selected.ConfigRevision != 3 || selected.RuntimeID != strings.Repeat("a", 64) {
						t.Error("wrong selection", selected)
					}
					out := protocol.DeskApplication{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Selection: selected, Observation: deskwork.Observation{Exists: true, Revision: 3, ClayHash: "0v123", Publication: "private"}, Application: deskwork.AppState{Zest: "live", SourceRevision: 3, ExpectedAgents: []string{"example"}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}, ReadAt: time.Now().UTC()}
					switch scenario {
					case "wrong-runner":
						out.RunnerID = "another"
					case "wrong-runtime":
						out.Selection.RuntimeID = strings.Repeat("b", 64)
					case "wrong-generation":
						out.Selection.Target.Generation++
					case "wrong-config":
						out.Selection.ConfigRevision++
					case "stale":
						out.ReadAt = time.Now().Add(-2 * time.Minute)
					case "future":
						out.ReadAt = time.Now().Add(time.Minute)
					case "nil-agents":
						out.Application.Agents = nil
					case "wrong-revision":
						out.Application.SourceRevision++
					case "invalid-stage":
						out.Application.TreatyListed = true
					case "policy-changed":
						out.PolicyRevision++
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
			out, err := r.InspectDeskApplication(context.Background(), "fixture", "runner-fixture", "publisher", "example")
			if scenario == "ok" || scenario == "project" {
				if err != nil || out.Observation.Selection.Target.Ship() != target || out.Observation.Application.Agents == nil {
					t.Fatal(out, err)
				}
				if out.Permissions.Commit.Value != apppolicy.Block || out.Permissions.Publish.Value != apppolicy.Block || len(out.Observation.Application.Agents) != 0 {
					t.Fatal("invented permission or readiness", out)
				}
			} else if err == nil {
				t.Fatal("unsafe application observation accepted", out)
			}
		})
	}
}
