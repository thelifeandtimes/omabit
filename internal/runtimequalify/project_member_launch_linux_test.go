//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/resourcepolicy"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProjectMemberLaunchAdapter(t *testing.T) {
	for _, mode := range []string{"success", "lost-create-reply", "id-with-error", "lost-start-reply", "wrong-start-ack", "partial-pier", "anchor-changed", "member-namespace-changed", "existing-workspace", "missing-pill", "changed-pill", "changed-pill-after-observation", "replaced-pill-after-observation", "changed-pill-before-start", "new-minimum", "low-create", "low-start"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			resource, network, anchor := resourceContractFixture(t)
			f.engine.ID = resource.Preparation.EngineID
			runningAnchorFixture(&anchor, &network)
			prepared := resource.Preparation
			prepared.Sequence = 6
			prepared.Bridge.Phase, prepared.Bridge.ID = "observed", network.ID
			prepared.Anchor.Phase, prepared.Anchor.ID = "observed", anchor.ID
			ns := &projectnetwork.NamespaceIdentity{BootID: "12345678-1234-1234-1234-123456789abc", Device: 5, Inode: 42}
			started := projectnetwork.AnchorStart{Prepared: prepared, OperationID: "start-anchor", Phase: "observed", Namespace: ns}
			storage, err := projectnetwork.NewMemberStorage(resource.Selection.Plan, prepared.ProjectID, 1, "sender", strings.Repeat("e", 32), "storage-sender")
			if err != nil {
				t.Fatal(err)
			}
			if err = pierstore.Init(f.d.home); err != nil {
				t.Fatal(err)
			}
			mountSource, err := pierstore.PrepareMount(f.d.home, &storage.Location)
			if err != nil {
				t.Fatal(err)
			}
			storage.Phase = "observed"
			base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
			dir := filepath.Join(base, "project-resources-"+prepared.ProjectID)
			if err = os.MkdirAll(filepath.Join(dir, "docker-config"), 0700); err != nil {
				t.Fatal(err)
			}
			pill := bootpill.CachePath(base, protocol.BootPill{SHA256: f.d.pillSHA, Bytes: f.d.pillBytes})
			if err = os.Mkdir(filepath.Dir(pill), 0700); err != nil {
				t.Fatal(err)
			}
			if mode != "missing-pill" {
				body := []byte("tiny disposable test pill\n")
				if mode == "changed-pill" {
					body = []byte("changed")
				}
				if err = os.WriteFile(pill, body, 0400); err != nil {
					t.Fatal(err)
				}
			}
			workspace := filepath.Join(base, storage.RuntimeID)
			if mode == "existing-workspace" {
				if err = os.Mkdir(workspace, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(workspace, "keep"), []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var c projectMemberContract
			var member *container
			var state projectnetwork.MemberLaunch
			creates, starts := 0, 0
			original := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
					t.Fatal("unbound Docker call", args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded Docker call")
				}
				a := args[4:]
				items := []projectnetwork.NetworkContainer{{ID: anchor.ID, Name: anchor.Name, NetworkMode: anchor.HostConfig.NetworkMode}}
				if member != nil {
					items = append(items, projectnetwork.NetworkContainer{ID: member.ID, Name: member.Name, NetworkMode: member.HostConfig.NetworkMode})
				}
				switch {
				case reflect.DeepEqual(a, []string{"image", "inspect", candidate.Image}):
					return original(ctx, args, env)
				case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
					var ids []string
					for _, v := range items {
						ids = append(ids, v.ID)
					}
					return strings.Join(ids, "\n"), nil
				case len(a) > 4 && a[0] == "container" && a[1] == "inspect" && a[2] == "--format":
					var ids []string
					for _, v := range items {
						ids = append(ids, v.ID)
					}
					slices.Sort(ids)
					if !reflect.DeepEqual(ids, a[4:]) {
						t.Fatal("unbound inventory IDs")
					}
					var out strings.Builder
					for _, v := range items {
						var raw []byte
						switch a[3] {
						case projectInventoryFormat:
							raw, _ = json.Marshal(v)
						case projectMountFormat:
							mounts := anchor.Mounts
							if member != nil && v.ID == member.ID {
								mounts = member.Mounts
							}
							raw, _ = json.Marshal(projectContainerMounts{ID: v.ID, Mounts: mounts})
						default:
							t.Fatal("unexpected inventory", a[3])
						}
						out.Write(raw)
						out.WriteByte('\n')
					}
					return out.String(), nil
				case reflect.DeepEqual(a, []string{"network", "inspect", network.ID}):
					raw, _ := json.Marshal([]bridge{network})
					return string(raw), nil
				case len(a) > 3 && a[0] == "container" && a[1] == "inspect":
					var values []container
					for _, id := range a[2:] {
						if id == anchor.ID {
							values = append(values, anchor)
						} else if member != nil && id == member.ID {
							values = append(values, *member)
						} else {
							t.Fatal("unowned bulk inspection", id)
						}
					}
					raw, _ := json.Marshal(values)
					return string(raw), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", anchor.ID}):
					raw, _ := json.Marshal([]container{anchor})
					return string(raw), nil
				case member != nil && reflect.DeepEqual(a, []string{"container", "inspect", member.ID}):
					raw, _ := json.Marshal([]container{*member})
					return string(raw), nil
				case len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}):
					if a[2] != anchor.ID && (member == nil || a[2] != member.ID) {
						t.Fatal("unowned exec")
					}
					if mode == "anchor-changed" && a[2] == anchor.ID || mode == "member-namespace-changed" && member != nil && a[2] == member.ID {
						return "5:43\n", nil
					}
					return "5:42\n", nil
				case len(a) > 2 && a[0] == "container" && a[1] == "create":
					creates++
					if state.Phase != "create-dispatched" || creates != 1 || !reflect.DeepEqual(a, c.createArgs()) {
						t.Fatal("unbound or unjournaled Create", a)
					}
					v := projectMemberContainerFixture(t, c)
					member = &v
					if mode == "partial-pier" {
						if err = os.Mkdir(filepath.Join(mountSource, "zod"), 0700); err != nil {
							t.Fatal(err)
						}
					}
					if mode == "lost-create-reply" {
						return "", errors.New("synthetic loss")
					}
					if mode == "id-with-error" {
						return member.ID + "\n", errors.New("synthetic late error")
					}
					return member.ID + "\n", nil
				case member != nil && reflect.DeepEqual(a, []string{"container", "start", member.ID}):
					starts++
					if state.Phase != "start-dispatched" || starts != 1 {
						t.Fatal("Start without durable intent")
					}
					*member.State.Running = true
					member.State.Status = "running"
					member.State.Pid = 88
					member.State.StartedAt = "2026-10-23T12:00:01Z"
					if err = os.Mkdir(filepath.Join(mountSource, "zod"), 0700); err != nil {
						t.Fatal(err)
					}
					if mode == "lost-start-reply" {
						return "", errors.New("synthetic loss")
					}
					if mode == "wrong-start-ack" {
						return "wrong\n", nil
					}
					return member.ID + "\n", nil
				default:
					t.Fatal("unexpected effect or command", a)
					return "", errors.New("unexpected")
				}
			}
			capacityChecks := 0
			if mode == "new-minimum" {
				f.d.resourcePolicy = func() (resourcepolicy.Policy, error) {
					v := resourcepolicy.Default()
					v.New.Minimum.RAMMiB = 16384
					v.New.Recommended.RAMMiB = 16384
					return v, nil
				}
			}
			if mode == "low-create" || mode == "low-start" {
				f.d.admission = func(string) (uint64, uint64, error) {
					capacityChecks++
					if mode == "low-create" && capacityChecks >= 2 || mode == "low-start" && capacityChecks >= 3 {
						return 0, 8 << 30, nil
					}
					return 5 << 30, 8 << 30, nil
				}
			}
			initial, raw, err := openProjectMemberLaunch(context.Background(), ProjectMemberLaunchSelection{Project: resource.Selection, Anchor: started, Storage: storage, OperationID: "launch-member"}, f.d)
			if mode == "existing-workspace" || mode == "missing-pill" || mode == "changed-pill" || mode == "new-minimum" {
				if err == nil || raw != nil || creates+starts != 0 {
					t.Fatal("unprepared session admitted", err)
				}
				if mode == "existing-workspace" {
					data, e := os.ReadFile(filepath.Join(workspace, "keep"))
					if e != nil || string(data) != "original" {
						t.Fatal("existing workspace changed")
					}
				} else if _, e := os.Lstat(workspace); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("setup created workspace", e)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			engine := raw.(*projectMemberLaunchEngine)
			engine.anchor.readBootID = func() (string, error) { return ns.BootID, nil }
			c = engine.contract
			state = initial
			got, err := projectnetwork.ExecuteMemberLaunch(context.Background(), resource.Selection.Plan, state, engine, func(expected, phase string, result projectnetwork.MemberLaunchResult) error {
				next, e := state.Advance(resource.Selection.Plan, expected, phase, result)
				if e == nil {
					state = next
					if phase == "create-dispatched" && (mode == "changed-pill-after-observation" || mode == "replaced-pill-after-observation") || phase == "start-dispatched" && mode == "changed-pill-before-start" {
						path := filepath.Join(workspace, candidate.PillName)
						if mode == "replaced-pill-after-observation" {
							if err := os.Rename(path, filepath.Join(base, "retained-original-pill")); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(path, []byte("tiny disposable test pill\n"), 0400); err != nil {
								t.Fatal(err)
							}
						} else {
							if err := os.Chmod(path, 0600); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(path, []byte("changed disposable pill!!\n"), 0400); err != nil {
								t.Fatal(err)
							}
							if err := os.Chmod(path, 0400); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				return e
			}, time.Now)
			if (err == nil) != (mode == "success") || !reflect.DeepEqual(got, state) {
				t.Fatal(mode, state.Phase, err)
			}
			want := map[string]string{"low-create": "create-dispatched", "low-start": "start-dispatched", "success": "running", "lost-create-reply": "create-dispatched", "id-with-error": "created", "partial-pier": "created", "lost-start-reply": "start-dispatched", "wrong-start-ack": "start-dispatched", "anchor-changed": "planned", "member-namespace-changed": "started", "changed-pill-after-observation": "create-dispatched", "replaced-pill-after-observation": "create-dispatched", "changed-pill-before-start": "start-dispatched"}[mode]
			if state.Phase != want {
				t.Fatal("wrong retained phase", state.Phase, want)
			}
			if (mode == "low-create" || mode == "changed-pill-after-observation" || mode == "replaced-pill-after-observation") && creates != 0 {
				t.Fatal("mutated artifact reached Create")
			}
			if (mode == "low-start" || mode == "partial-pier" || mode == "changed-pill-before-start") && starts != 0 {
				t.Fatal("bootstrapped over unrecorded pier")
			}
			if mode != "anchor-changed" {
				if _, e := os.Stat(workspace); e != nil {
					t.Fatal("workspace evidence removed", e)
				}
			}
			if f.downloads != 0 {
				t.Fatal("driver downloaded a dependency")
			}
			if err = engine.Start(context.Background(), state); err == nil {
				t.Fatal("direct Start replay accepted")
			}
		})
	}
}

func TestProjectMemberFreshnessDiagnosticRetainsBoundedAgesAndPhase(t *testing.T) {
	e := &projectMemberLaunchEngine{}
	if err := e.fresh("created"); err == nil || !strings.Contains(err.Error(), "preflight=missing") || !strings.Contains(err.Error(), "inspection=missing") {
		t.Fatal(err)
	}
	e.preflightAt = time.Now().Add(-31 * time.Second)
	e.inspectedAt = time.Now()
	e.inspectedPhase = "created"
	if err := e.fresh("created"); err == nil || !strings.Contains(err.Error(), "preflight=31") || !strings.Contains(err.Error(), "phase=created, expected=created") {
		t.Fatal(err)
	}
	e.preflightAt = time.Now()
	if err := e.fresh("created"); err != nil {
		t.Fatal(err)
	}
	if err := e.fresh("workspace-created"); err == nil || !strings.Contains(err.Error(), "expected=workspace-created") {
		t.Fatal(err)
	}
}
