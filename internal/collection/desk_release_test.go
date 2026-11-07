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

func TestDeskReleasePlansBindSelectionAndKeepUncertainSubmission(t *testing.T) {
	for _, scenario := range []string{"ok", "project", "old-runner", "wrong-runner", "wrong-runtime", "wrong-generation", "wrong-config", "stale", "future", "bad-digest", "policy-changed", "wrong-checks", "changed-receipt", "lost-reply", "stopped", "unresolved"} {
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
			var submissions atomic.Int32
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
					features := []string{protocol.AppPolicyFeature, protocol.DeskReleaseFeature}
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
				case protocol.DeskReleaseReviewPath:
					var request protocol.DeskReleaseRequest
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&request) != nil || request.Action != "publish-source" {
						t.Error("wrong release review request", request)
					}
					selected := request.Selection
					if selected.Target.Ship() != target || selected.Target.Desk != "example" || selected.ConfigRevision != 3 || selected.RuntimeID != strings.Repeat("a", 64) {
						t.Error("wrong release selection", selected)
					}
					desk := deskwork.DefaultSetting(selected.Target)
					desk.Desk.Publication = "source"
					desk.Desk.Checks = []string{"smoke"}
					review := deskwork.ReleaseReview{Version: 1, Action: request.Action, Observation: deskwork.Observation{Exists: true, Revision: 3, ClayHash: "0v123", Publication: "private"}, Application: deskwork.AppState{Zest: "dead", SourceRevision: 3, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}, ShipSetting: deskwork.DefaultSetting(target), DeskSetting: desk, Continuity: strings.Repeat("c", 64)}
					if err := review.Seal(); err != nil {
						t.Error(err)
					}
					out := protocol.DeskReleaseReview{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Request: request, Review: review, ReadAt: time.Now().UTC()}

					switch scenario {
					case "wrong-runner":
						out.RunnerID = "another"
					case "wrong-runtime":
						out.Request.Selection.RuntimeID = strings.Repeat("b", 64)
					case "wrong-generation":
						out.Request.Selection.Target.Generation++
					case "wrong-config":
						out.Request.Selection.ConfigRevision++
					case "stale":
						out.ReadAt = time.Now().Add(-protocol.DeskDeploymentTimeout - time.Minute)
					case "future":
						out.ReadAt = time.Now().Add(time.Minute)
					case "bad-digest":
						out.Review.Digest = strings.Repeat("f", 64)
					case "policy-changed":
						out.PolicyRevision++
					}
					encode(out)
				case protocol.DeskReleasePath:
					submissions.Add(1)
					var submission protocol.DeskReleaseSubmission
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&submission) != nil || submission.Validate() != nil {
						t.Error("invalid submission")
					}
					if scenario == "lost-reply" {
						connection, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
						} else {
							connection.Close()
						}
						return
					}
					intent := submission.Intent.Intent
					if scenario == "changed-receipt" {
						intent.ConfigRevision++
					}
					encode(protocol.DeskOperation{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: deskwork.Receipt{Version: 1, Intent: intent, Actor: "fixture", Steps: []deskwork.Step{{Kind: intent.Action, State: "planned", At: time.Now().UTC()}}}})
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
			checks := []deskwork.ReleaseCheck{{Name: "smoke", ClayHash: "0v123", Passed: true, Evidence: "fixture checks passed"}}
			if scenario == "wrong-checks" {
				checks[0].ClayHash = "0v999"
			}
			plan, err := r.PlanDeskRelease(context.Background(), "fixture", "runner-fixture", "publisher", "example", "publish-source", checks)
			if scenario != "ok" && scenario != "project" && scenario != "changed-receipt" && scenario != "lost-reply" {
				if err == nil {
					t.Fatal("unsafe plan accepted", plan)
				}
				if submissions.Load() != 0 {
					t.Fatal("review submitted an effect")
				}
				return
			}
			if err != nil || plan.Validate() != nil {
				t.Fatal(plan, err)
			}
			_, err = r.ReleaseDesk(context.Background(), "fixture", "runner-fixture", plan)
			if scenario == "changed-receipt" || scenario == "lost-reply" {
				if err == nil {
					t.Fatal("uncertain or changed reply accepted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if submissions.Load() != 1 {
				t.Fatal("submission was implicitly retried", submissions.Load())
			}
			if _, err = r.ReleaseDesk(context.Background(), "fixture", "wrong-runner", plan); err == nil {
				t.Fatal("release crossed runner")
			}
			if submissions.Load() != 1 {
				t.Fatal("invalid runner dispatched")
			}
		})
	}
}
