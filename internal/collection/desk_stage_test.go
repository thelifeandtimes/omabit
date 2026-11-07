//go:build linux

package collection

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskStagingCollectorStreamsFrozenSourceAndReconciles(t *testing.T) {
	for _, scenario := range []string{"ok", "replay", "lost-reply", "old-runner", "changed-source", "wrong-runner", "wrong-receipt", "urgit", "discard-ok", "discard-lost", "discard-wrong"} {
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
			source := t.TempDir()
			if err = os.WriteFile(filepath.Join(source, "hello.txt"), []byte("hello\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cache := t.TempDir()
			if err = os.Chmod(cache, 0700); err != nil {
				t.Fatal(err)
			}
			target := apppolicy.Target{InstanceID: "publisher", Desk: "example"}
			var uploads atomic.Int32
			var discards atomic.Int32
			var cacheMu sync.Mutex
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch {
				case q.URL.Path == "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature, protocol.DeskStageFeature}
					if scenario == "old-runner" {
						features = features[:2]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Actor: "fixture", Features: features})
				case q.URL.Path == protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("policy write attempted")
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target.Ship(), Ship: "~zod", Label: "publisher", Ready: true}}})
				case q.URL.Path == protocol.DeskSettingsPath:
					var got apppolicy.Target
					if q.Method != "POST" || json.NewDecoder(q.Body).Decode(&got) != nil || got != target {
						t.Error("wrong workflow selection", got)
					}
					s := deskwork.DefaultSetting(target)
					if scenario == "urgit" {
						s.Desk.Workflow = "urgit"
						s.Desk.SourceName = ""
					}
					encode(protocol.DeskSetting{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Setting: s, ReadAt: time.Now().UTC()})
				case q.URL.Path == protocol.DeskStagesPath:
					uploads.Add(1)
					data, e := base64.RawURLEncoding.DecodeString(q.Header.Get(protocol.DeskStageHeader))
					var intent deskwork.StageIntent
					if e != nil || json.Unmarshal(data, &intent) != nil || q.Method != "POST" || q.Header.Get("Content-Type") != "application/octet-stream" || intent.Target != target {
						t.Error("wrong staged request", intent, e)
						http.Error(w, "bad", 400)
						return
					}
					cacheMu.Lock()
					s, e := deskwork.CacheStage(q.Context(), cache, intent, "fixture", q.Body)
					cacheMu.Unlock()
					if e != nil {
						t.Error(e)
						http.Error(w, "bad", 400)
						return
					}
					out := protocol.DeskStage{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: s.Receipt, ReadAt: time.Now().UTC()}
					if scenario == "lost-reply" {
						w.Write([]byte("{"))
						return
					}
					if scenario == "wrong-runner" {
						out.RunnerID = "other"
					}
					if scenario == "wrong-receipt" {
						out.Receipt.Intent.SourceDigest = strings.Repeat("f", 64)
					}
					encode(out)
				case strings.HasPrefix(q.URL.Path, protocol.DeskStagesPath+"/"):
					if q.Method == "DELETE" {
						discards.Add(1)
						id := strings.TrimPrefix(q.URL.Path, protocol.DeskStagesPath+"/")
						var input protocol.DeskStageDiscardRequest
						if json.NewDecoder(q.Body).Decode(&input) != nil {
							t.Error("bad discard payload")
							http.Error(w, "bad", 400)
							return
						}
						cacheMu.Lock()
						err := deskwork.DiscardStage(cache, id, "fixture", input.SourceDigest, nil)
						cacheMu.Unlock()
						if err != nil {
							t.Error(err)
							http.Error(w, "bad", 409)
							return
						}
						if scenario == "discard-lost" {
							w.Write([]byte("{"))
							return
						}
						out := protocol.DeskStageDiscard{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ID: id, SourceDigest: input.SourceDigest, Discarded: true}
						if scenario == "discard-wrong" {
							out.ID = strings.Repeat("f", 64)
						}
						encode(out)
						return
					}
					if q.Method != "GET" {
						t.Error("receipt mutation attempted")
					}
					cacheMu.Lock()
					s, _, e := deskwork.ReadStage(q.Context(), cache, strings.TrimPrefix(q.URL.Path, protocol.DeskStagesPath+"/"), "fixture")
					cacheMu.Unlock()
					if e != nil {
						t.Error(e)
						http.Error(w, "bad", 400)
						return
					}
					encode(protocol.DeskStage{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", Receipt: s.Receipt, ReadAt: time.Now().UTC()})
				default:
					t.Error("unexpected runtime or route", q.URL.Path)
					http.NotFound(w, q)
				}
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			if _, err = r.Enroll(context.Background(), "fixture", socket); err != nil {
				t.Fatal(err)
			}
			plan, err := r.PlanDeskStage(context.Background(), "fixture", "runner-fixture", "publisher", "example", source)
			if scenario == "urgit" {
				if err == nil {
					t.Fatal("Urgit stage plan accepted")
				}
				return
			}
			if err != nil || plan.Intent.Target != target || plan.Intent.Bytes != 6 || plan.Intent.Files != 1 {
				t.Fatal(plan, err)
			}
			if scenario == "changed-source" {
				if err = os.WriteFile(filepath.Join(source, "hello.txt"), []byte("edited"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			staged, err := r.StageDesk(context.Background(), "fixture", "runner-fixture", plan, source)
			if scenario == "ok" || scenario == "replay" || strings.HasPrefix(scenario, "discard-") {
				if err != nil || staged.Receipt.Intent != plan.Intent {
					t.Fatal(staged, err)
				}
			} else if err == nil {
				t.Fatal("unverified stage accepted", staged)
			}
			if scenario == "old-runner" || scenario == "changed-source" {
				if uploads.Load() != 0 {
					t.Fatal("upload sent without matching source/capability")
				}
				return
			}
			if uploads.Load() != 1 {
				t.Fatal("upload automatically replayed", uploads.Load())
			}
			read, err := r.DeskStage(context.Background(), "fixture", "runner-fixture", plan.Intent.ID)
			if err != nil || read.Receipt.Intent != plan.Intent {
				t.Fatal("lost response cannot be reconciled", read, err)
			}
			if strings.HasPrefix(scenario, "discard-") {
				out, err := r.DiscardDeskStage(context.Background(), "fixture", "runner-fixture", plan.Intent.ID, plan.Intent.SourceDigest)
				if scenario == "discard-ok" {
					if err != nil || !out.Discarded {
						t.Fatal(out, err)
					}
				} else if err == nil {
					t.Fatal("bad discard response trusted")
				}
				if discards.Load() != 1 {
					t.Fatal("discard automatically replayed")
				}
				if _, err = os.Stat(filepath.Join(cache, plan.Intent.ID)); !os.IsNotExist(err) {
					t.Fatal("cache not discarded", err)
				}
				if got, err := os.ReadFile(filepath.Join(source, "hello.txt")); err != nil || string(got) != "hello\n" {
					t.Fatal("source modified", err)
				}
			}
			if scenario == "replay" {
				again, err := r.StageDesk(context.Background(), "fixture", "runner-fixture", plan, source)
				if err != nil || again.Receipt.CreatedAt != staged.Receipt.CreatedAt {
					t.Fatal("exact replay failed", again, err)
				}
			}
		})
	}
}
