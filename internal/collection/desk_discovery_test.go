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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskDiscoveryChecksCapabilitiesAndGeneration(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "empty", "old-runner", "wrong-runner", "wrong-runtime", "wrong-generation", "wrong-config", "stale", "future", "nil-desks", "duplicate", "invalid-name", "stopped", "unresolved"} {
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
					features := []string{protocol.AppPolicyFeature, protocol.DeskDiscoveryFeature}
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
				case protocol.DeskDiscoveryPath:
					var selected protocol.DeskDiscoverySelection
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&selected) != nil || selected.Target != target || selected.ConfigRevision != 3 || selected.RuntimeID != strings.Repeat("a", 64) {
						t.Error("wrong selection", selected)
					}
					out := protocol.DeskDiscovery{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Selection: selected, Desks: []string{"zeta", "base"}, ReadAt: time.Now().UTC()}
					switch scenario {
					case "empty":
						out.Desks = []string{}
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
					case "nil-desks":
						out.Desks = nil
					case "duplicate":
						out.Desks = []string{"base", "base"}
					case "invalid-name":
						out.Desks = []string{"base/other"}
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
			out, err := r.DiscoverDesks(context.Background(), "fixture", "runner-fixture", "publisher")
			if scenario == "ok" || scenario == "project" || scenario == "empty" {
				if err != nil || out.Selection.Target != target || out.Desks == nil {
					t.Fatal(out, err)
				}
				if scenario != "empty" && (len(out.Desks) != 2 || out.Desks[0] != "base") {
					t.Fatal(out)
				}
			} else if err == nil {
				t.Fatal("unsafe discovery accepted", out)
			}
		})
	}
}
