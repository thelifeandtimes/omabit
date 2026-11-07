package collection

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestDeskSettingsReadAndSaveBindings(t *testing.T) {
	for _, scenario := range []string{"ok", "ship", "project", "old-runner", "wrong-runner", "wrong-target", "stale", "future", "invalid-config", "unresolved", "stale-revision", "save-revision", "save-config"} {
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
			desk := "my-app"
			if scenario == "ship" {
				desk = ""
			}
			var writes atomic.Int32
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				encode := func(v any) {
					if err := json.NewEncoder(w).Encode(v); err != nil {
						t.Error(err)
					}
				}
				switch q.URL.Path {
				case "/hello":
					features := []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature}
					if scenario == "old-runner" {
						features = features[:1]
					}
					encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-fixture", Mode: "simulation", AccessRole: "operator", Features: features})
				case protocol.AppPolicyPath:
					if q.Method != "GET" {
						t.Error("policy write attempted")
					}
					encode(protocol.AppPolicyStatus{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target, Ship: "~zod", Label: "publisher", Ready: scenario != "unresolved"}}})
				case protocol.DeskSettingsPath:
					selected := target
					selected.Desk = desk
					out := protocol.DeskSetting{Version: 1, RunnerID: "runner-fixture", Mode: "simulation", ReadAt: time.Now().UTC(), Setting: deskwork.DefaultSetting(selected)}
					if q.Method == "POST" {
						var got apppolicy.Target
						if json.NewDecoder(q.Body).Decode(&got) != nil || got != selected {
							t.Error("wrong target", got)
						}
						switch scenario {
						case "wrong-runner":
							out.RunnerID = "another"
						case "wrong-target":
							out.Setting.Target.Generation++
						case "stale":
							out.ReadAt = time.Now().Add(-3 * time.Minute)
						case "future":
							out.ReadAt = time.Now().Add(time.Minute)
						case "invalid-config":
							out.Setting.Desk.SourceName = "../outside"
						}
					} else if q.Method == "PUT" {
						writes.Add(1)
						var u deskwork.SettingUpdate
						if json.NewDecoder(q.Body).Decode(&u) != nil || u.Target != selected || u.ExpectedRevision != 0 || u.Validate() != nil {
							t.Error("wrong update", u)
						}
						out.Setting.Ship = u.Ship
						out.Setting.Desk = u.Desk
						out.Setting.Revision = 1
						out.Setting.Actor = "fixture"
						out.Setting.UpdatedAt = time.Now().UTC()
						if scenario == "save-revision" {
							out.Setting.Revision++
						}
						if scenario == "save-config" {
							out.Setting.Desk.Publication = "treaty"
						}
					} else {
						t.Error("unexpected settings method", q.Method)
					}
					encode(out)
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
			out, err := r.InspectDeskSetting(context.Background(), "fixture", "runner-fixture", "publisher", desk)
			valid := scenario == "ok" || scenario == "ship" || scenario == "project" || scenario == "stale-revision" || scenario == "save-revision" || scenario == "save-config"
			if !valid {
				if err == nil {
					t.Fatal("unsafe read accepted", out)
				}
				return
			}
			selected := target
			selected.Desk = desk
			if err != nil || out.Setting.Target != selected {
				t.Fatal(out, err)
			}
			u := deskwork.SettingUpdate{Target: selected, Ship: out.Setting.Ship, Desk: out.Setting.Desk}
			if u.Desk != nil {
				u.Desk.Branch = "master"
			} else {
				u.Ship.Purpose = "distribution"
			}
			if scenario == "stale-revision" {
				u.ExpectedRevision = 1
			}
			saved, err := r.SaveDeskSetting(context.Background(), "fixture", "runner-fixture", u)
			if scenario == "stale-revision" || scenario == "save-revision" || scenario == "save-config" {
				if err == nil {
					t.Fatal("unsafe save confirmed", saved)
				}
			} else if err != nil || saved.Setting.Revision != 1 {
				t.Fatal(saved, err)
			}
			want := int32(1)
			if scenario == "stale-revision" {
				want = 0
			}
			if writes.Load() != want {
				t.Fatal("save replay or missing precondition", writes.Load())
			}
		})
	}
}
