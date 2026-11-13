package collection

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskPlanningPinsRuntimeAndSubmissionKeepsRequest(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "old-runner", "policy-changed", "runtime-changed", "wrong-reply-runner", "changed-intent", "existing"} {
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
			var mu sync.Mutex
			var submitted []deskwork.Intent
			var saved protocol.DeskOperation
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
					features := []string{protocol.AppPolicyFeature, protocol.DeskCreateFeature}
					if scenario == "old-runner" {
						features = features[:1]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Features: features})
				case protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("agent tried to write policy")
						http.Error(w, "denied", 403)
						return
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target, Ship: "~zod", Label: "publisher", Ready: true}}})
				case "/v1/snapshot":
					encode(protocol.Snapshot{RunnerID: "runner-fixture", Mode: "simulation", Instances: []protocol.Instance{{ID: "publisher", ProjectID: target.ProjectID, ConfigRevision: 3, ObservedState: "running", Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.DeskObservationPath:
					var selection protocol.DeskSelection
					if json.NewDecoder(q.Body).Decode(&selection) != nil {
						t.Error("bad selection")
					}
					if selection.Target.Ship() != target || selection.Target.Desk != "example" || selection.ConfigRevision != 3 || selection.RuntimeID != strings.Repeat("a", 64) {
						t.Error("wrong selection", selection)
					}
					out := protocol.DeskObservation{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Selection: selection, ReadAt: time.Now().UTC()}
					if scenario == "runtime-changed" {
						out.Selection.RuntimeID = strings.Repeat("b", 64)
					}
					if scenario == "policy-changed" {
						out.PolicyRevision = 1
					}
					if scenario == "wrong-reply-runner" {
						out.RunnerID = "runner-other"
					}
					if scenario == "existing" {
						out.Observation = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
					}
					encode(out)
				case protocol.DeskOperationsPath:
					var intent deskwork.Intent
					if json.NewDecoder(q.Body).Decode(&intent) != nil {
						t.Error("bad intent")
					}
					submitted = append(submitted, intent)
					saved = protocol.DeskOperation{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: deskwork.Receipt{Version: 1, Intent: intent, Actor: "fixture", Steps: []deskwork.Step{{Kind: "create-desk", State: "planned", At: time.Now().UTC()}}}}
					out := saved
					if scenario == "changed-intent" {
						out.Receipt.Intent.ConfigRevision++
					}
					encode(out)
				default:
					if q.URL.Path != protocol.DeskOperationsPath+"/"+saved.Receipt.Intent.ID || q.Method != "GET" {
						t.Error("unexpected route", q.Method, q.URL.Path)
						http.NotFound(w, q)
						return
					}
					encode(saved)
				}
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			if _, err = r.Enroll(context.Background(), "fixture", socket); err != nil {
				t.Fatal(err)
			}
			plan, err := r.PlanDeskCreation(context.Background(), "fixture", "runner-fixture", "publisher", "example")
			if scenario != "ok" && scenario != "project" && scenario != "changed-intent" {
				if err == nil {
					t.Fatal("unsafe plan accepted")
				}
				return
			}
			if err != nil || plan.Validate() != nil {
				t.Fatal(plan, err)
			}
			if plan.Intent.Target.Ship() != target || plan.Intent.PolicyRevision != 0 {
				t.Fatal("selection changed", plan)
			}
			for i := 0; i < 2; i++ {
				_, err = r.CreateDesk(context.Background(), "fixture", "runner-fixture", plan)
				if scenario == "changed-intent" {
					if err == nil {
						t.Fatal("changed response accepted")
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := r.DeskOperation(context.Background(), "fixture", "runner-fixture", plan.Intent.ID)
			if err != nil || receipt.Receipt.Intent != plan.Intent {
				t.Fatal(receipt, err)
			}
			mu.Lock()
			if len(submitted) != 2 || submitted[0] != submitted[1] || submitted[0] != plan.Intent {
				t.Error("retry changed intent", submitted)
			}
			mu.Unlock()
			if _, err = r.CreateDesk(context.Background(), "fixture", "runner-other", plan); err == nil {
				t.Fatal("cross-runner submission accepted")
			}
		})
	}
}
