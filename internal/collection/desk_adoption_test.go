package collection

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDeskAdoptionCollectionPinsReviewAndReadsLostReply(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "lost-reply", "wrong-receipt", "partial-receipt", "old-runner", "stopped", "changed-generation", "changed-workflow", "wrong-runner", "wrong-source", "wrong-stage", "wrong-runtime", "stale", "bad-count", "unsafe-path", "changed-policy", "no-op", "conflict"} {
		t.Run(scenario, func(t *testing.T) {
			registry := registry(t)
			socket := filepath.Join(t.TempDir(), "runner.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.Chmod(socket, 0600); err != nil {
				t.Fatal(err)
			}
			target := apppolicy.Target{InstanceID: "publisher", Desk: "example"}
			if scenario == "project" || scenario == "changed-generation" {
				target.ProjectID = "network"
				target.Generation = 2
				target.Creator = "resume-2"
			}
			stage := deskwork.StageReceipt{Version: 1, Intent: deskwork.StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: target, SourceDigest: strings.Repeat("e", 64), Files: 1, Bytes: 6}, Actor: "fixture", CreatedAt: time.Now().UTC()}
			var mu sync.Mutex
			commits := 0
			var saved protocol.DeskOperation
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set(protocol.Header, "1")
				encode := func(value any) {
					if err := json.NewEncoder(w).Encode(value); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.DeskCreateFeature, protocol.DeskStageFeature, protocol.DeskSettingsFeature, protocol.DeskAdoptionFeature}
					if scenario == "old-runner" {
						features = features[:4]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Actor: "fixture", Features: features})
				case protocol.DeskStagesPath + "/" + stage.Intent.ID:
					encode(protocol.DeskStage{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: stage, ReadAt: time.Now().UTC()})
				case protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("policy write attempted")
					}
					current := target.Ship()
					if scenario == "changed-generation" {
						current.Generation++
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: current, Ship: "~zod", Label: "publisher", Ready: true}}})
				case "/v1/snapshot":
					state := "running"
					if scenario == "stopped" {
						state = "stopped"
					}
					encode(protocol.Snapshot{RunnerID: "runner-fixture", Mode: "simulation", Instances: []protocol.Instance{{ID: "publisher", ProjectID: target.ProjectID, ConfigRevision: 3, ObservedState: state, Resources: &protocol.ResourceMetrics{ContainerID: strings.Repeat("a", 64)}}}})
				case protocol.DeskSettingsPath:
					setting := deskwork.DefaultSetting(target)
					if scenario == "changed-workflow" {
						setting.Desk.Workflow = "urgit"
						setting.Desk.SourceName = ""
					}
					encode(protocol.DeskSetting{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Setting: setting, ReadAt: time.Now().UTC()})
				case protocol.DeskAdoptionReviewPath:
					var request protocol.DeskDeploymentRequest
					if json.NewDecoder(q.Body).Decode(&request) != nil || request.Selection.Target != target || request.StageID != stage.Intent.ID {
						t.Error("changed review selection")
					}
					out := protocol.DeskAdoptionReview{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Request: request, Observation: deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, SourceName: "example", SourceDigest: stage.Intent.SourceDigest, PlanDigest: strings.Repeat("f", 64), Continuity: strings.Repeat("b", 64), ReadAt: time.Now().UTC(), PathCount: 1, Paths: []string{"hello.txt"}, Conflicts: []string{}, NewCount: 1}
					switch scenario {
					case "wrong-runner":
						out.RunnerID = "another"
					case "wrong-source":
						out.SourceDigest = strings.Repeat("a", 64)
					case "wrong-stage":
						out.Request.StageID = strings.Repeat("a", 64)
					case "wrong-runtime":
						out.Request.Selection.RuntimeID = strings.Repeat("b", 64)
					case "stale":
						out.ReadAt = time.Now().Add(-10 * time.Minute)
					case "bad-count":
						out.PathCount = 2
					case "unsafe-path":
						out.Paths[0] = "../outside.txt"
					case "changed-policy":
						out.PolicyRevision++
					case "no-op":
						out.NewCount = 0
					case "conflict":
						out.ConflictCount = 1
						out.NewCount = 0
						out.Conflicts = []string{"other.txt"}
					}
					encode(out)
				case protocol.DeskAdoptionPath:
					var input deskwork.AdoptionIntent
					if json.NewDecoder(q.Body).Decode(&input) != nil || input.Validate() != nil || input.StageID != stage.Intent.ID {
						t.Error("wrong adoption selection")
					}
					commits++
					saved = protocol.DeskOperation{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: deskwork.Receipt{Version: 1, Intent: input.Intent, Actor: "fixture", Steps: []deskwork.Step{{Kind: "adopt-source", State: "planned", At: time.Now().UTC()}, {Kind: "adopt-source", State: "observed", At: time.Now().UTC(), Observation: &input.Intent.Baseline, VerifiedSourceDigest: input.Intent.SourceDigest}}}}
					if scenario == "lost-reply" {
						w.Write([]byte("{"))
						return
					}
					out := saved
					if scenario == "partial-receipt" {
						out.Receipt.Steps = out.Receipt.Steps[:1]
					}
					if scenario == "wrong-receipt" {
						out.Receipt.Intent.SourceDigest = strings.Repeat("b", 64)
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
			go server.Serve(listener)
			t.Cleanup(func() { server.Close() })
			if _, err = registry.Enroll(context.Background(), "fixture", socket); err != nil {
				t.Fatal(err)
			}
			plan, err := registry.PlanDeskAdoption(context.Background(), "fixture", "runner-fixture", stage.Intent.ID)
			proceed := scenario == "ok" || scenario == "project" || scenario == "lost-reply" || scenario == "wrong-receipt" || scenario == "partial-receipt" || scenario == "no-op" || scenario == "conflict"
			if !proceed {
				if err == nil {
					t.Fatal("unsafe review accepted", plan)
				}
				return
			}
			if err != nil || plan.Validate() != nil {
				t.Fatal(plan, err)
			}
			result, err := registry.AdoptDeskSource(context.Background(), "fixture", "runner-fixture", plan)
			if scenario == "ok" || scenario == "project" {
				if err != nil || result.Receipt.Intent != plan.Intent.Intent {
					t.Fatal(result, err)
				}
			} else if err == nil {
				t.Fatal("invalid adoption or reply accepted")
			}
			mu.Lock()
			count := commits
			mu.Unlock()
			if scenario == "no-op" || scenario == "conflict" {
				if count != 0 {
					t.Fatal("non-runnable plan submitted")
				}
				return
			}
			if count != 1 {
				t.Fatal("implicit adoption replay", count)
			}
			got, err := registry.DeskOperation(context.Background(), "fixture", "runner-fixture", plan.Intent.Intent.ID)
			if err != nil || got.Receipt.Intent != plan.Intent.Intent {
				t.Fatal("lost reply cannot be reconciled", got, err)
			}
			if _, err = registry.AdoptDeskSource(context.Background(), "fixture", "another-runner", plan); err == nil {
				t.Fatal("cross-runner adoption accepted")
			}
		})
	}
}
