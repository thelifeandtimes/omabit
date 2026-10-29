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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func runningAnchorFixture(v *container, n *bridge) {
	v.State.Status = "running"
	v.State.Pid = 42
	v.State.StartedAt = "2026-10-23T12:00:00Z"
	*v.State.Running = true
	n.Containers[v.ID] = json.RawMessage(`{}`)
	v.NetworkSettings.Networks = map[string]json.RawMessage{n.Name: json.RawMessage(`{"NetworkID":"` + n.ID + `"}`)}
	for k, raw := range v.HostConfig.PortBindings {
		v.NetworkSettings.Ports[k] = raw
	}
}
func TestProjectAnchorRunningContractRefusals(t *testing.T) {
	for name, mutate := range map[string]func(*container){
		"valid": func(*container) {}, "missing-pid": func(v *container) { v.State.Pid = 0 }, "missing-start": func(v *container) { v.State.StartedAt = "0001-01-01T00:00:00Z" },
		"missing-tmpfs": func(v *container) { v.Mounts = nil }, "changed-network": func(v *container) {
			v.NetworkSettings.Networks = map[string]json.RawMessage{"other": json.RawMessage(`{}`)}
		},
		"missing-publication": func(v *container) { delete(v.NetworkSettings.Ports, "8080/tcp") }, "extra-publication": func(v *container) {
			v.NetworkSettings.Ports["12321/tcp"] = json.RawMessage(`[{"HostIp":"127.0.0.1","HostPort":"18321"}]`)
		},
		"wildcard-publication": func(v *container) {
			v.NetworkSettings.Ports["8080/tcp"] = json.RawMessage(`[{"HostIp":"0.0.0.0","HostPort":"18282"}]`)
		}, "finished": func(v *container) { v.State.FinishedAt = v.State.StartedAt },
	} {
		t.Run(name, func(t *testing.T) {
			c, n, v := resourceContractFixture(t)
			runningAnchorFixture(&v, &n)
			mutate(&v)
			err := c.validateAnchor(v, v.ID, n.ID, true)
			if (err == nil) != (name == "valid") {
				t.Fatal(err)
			}
		})
	}
}

func TestProjectAnchorAdapterInitialStart(t *testing.T) {
	for _, mode := range []string{"success", "lost-reply", "wrong-ack", "hidden-member", "indirect-member", "inventory-changed", "metadata-changed", "engine-replaced", "namespace-changed", "boot-changed", "active-port-changed", "anchor-restarted"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			c, n, v := resourceContractFixture(t)
			f.engine.ID = c.Preparation.EngineID
			prepared := c.Preparation
			prepared.Sequence = 6
			prepared.Bridge.Phase, prepared.Bridge.ID = "observed", n.ID
			prepared.Anchor.Phase, prepared.Anchor.ID = "observed", v.ID
			dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/project-resources-"+prepared.ProjectID)
			if err := os.MkdirAll(filepath.Join(dir, "docker-config"), 0700); err != nil {
				t.Fatal(err)
			}
			state, err := projectnetwork.NewAnchorStart(c.Selection.Plan, prepared, "start-operation")
			if err != nil {
				t.Fatal(err)
			}
			original := f.d.execute
			starts, lists, metadata, probes, anchorReads := 0, 0, 0, 0, 0
			items := []projectnetwork.NetworkContainer{{ID: v.ID, Name: v.Name, NetworkMode: n.ID}}
			if mode == "hidden-member" || mode == "indirect-member" {
				items = append(items, projectnetwork.NetworkContainer{ID: strings.Repeat("c", 64), Name: "/member", NetworkMode: "container:" + v.ID})
				if mode == "indirect-member" {
					items = append(items, projectnetwork.NetworkContainer{ID: strings.Repeat("d", 64), Name: "/indirect", NetworkMode: "container:member"})
				}
			}
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
					t.Fatal("unbound Docker call", args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded Docker call")
				}
				a := args[4:]
				switch {
				case reflect.DeepEqual(a, []string{"image", "inspect", candidate.Image}):
					return original(ctx, args, env)
				case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
					lists++
					var ids []string
					for _, item := range items {
						ids = append(ids, item.ID)
					}
					if mode == "inventory-changed" && lists >= 2 {
						ids = append(ids, strings.Repeat("f", 64))
					}
					return strings.Join(ids, "\n") + "\n", nil
				case len(a) >= 5 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectInventoryFormat}):
					metadata++
					var ids []string
					for _, item := range items {
						ids = append(ids, item.ID)
					}
					slices.Sort(ids)
					if !reflect.DeepEqual(a[4:], ids) {
						t.Fatal("inspect not exact listed IDs", a)
					}
					var out strings.Builder
					for _, item := range items {
						if mode == "metadata-changed" && metadata >= 2 {
							item.Name = "/changed"
						}
						data, _ := json.Marshal(item)
						out.Write(data)
						out.WriteByte('\n')
					}
					return out.String(), nil
				case reflect.DeepEqual(a, []string{"network", "inspect", n.ID}):
					data, _ := json.Marshal([]bridge{n})
					return string(data), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", v.ID}):
					anchorReads++
					if mode == "anchor-restarted" && starts > 0 && anchorReads >= 4 {
						v.State.Pid++
					}
					data, _ := json.Marshal([]container{v})
					return string(data), nil
				case reflect.DeepEqual(a, []string{"container", "exec", v.ID, "/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}):
					probes++
					if mode == "namespace-changed" && probes > 1 {
						return "4:1002\n", nil
					}
					return "4:1001\n", nil
				case reflect.DeepEqual(a, []string{"container", "start", v.ID}):
					starts++
					if state.Phase != "dispatched" || starts != 1 {
						t.Fatal("Start before intent or replay", state)
					}
					runningAnchorFixture(&v, &n)
					if mode == "engine-replaced" {
						f.engine.ID = "other-engine"
					}
					if mode == "active-port-changed" {
						v.NetworkSettings.Ports["8080/tcp"] = json.RawMessage(`[{"HostIp":"0.0.0.0","HostPort":"18282"}]`)
					}
					if mode == "lost-reply" {
						return "", errors.New("synthetic client loss")
					}
					if mode == "wrong-ack" {
						return "wrong\n", nil
					}
					return v.ID + "\n", nil
				default:
					t.Fatal("unexpected mutation or command", a)
					return "", errors.New("unexpected")
				}
			}
			raw, err := openProjectAnchorStart(context.Background(), c.Selection, prepared, f.d)
			if err != nil {
				t.Fatal(err)
			}
			engine := raw.(*projectAnchorEngine)
			boots := 0
			engine.readBootID = func() (string, error) {
				boots++
				if mode == "boot-changed" && boots > 1 {
					return "00000000-0000-0000-0000-000000000002", nil
				}
				return "00000000-0000-0000-0000-000000000001", nil
			}
			got, err := projectnetwork.ExecuteAnchorStart(context.Background(), c.Selection.Plan, state, engine, func(expected, phase string, namespace *projectnetwork.NamespaceIdentity) error {
				next, err := state.Advance(c.Selection.Plan, expected, phase, namespace)
				if err == nil {
					state = next
				}
				return err
			}, time.Now)
			if (err == nil) != (mode == "success") || !reflect.DeepEqual(got, state) {
				t.Fatal(mode, state, err)
			}
			wantPhase, wantStarts := "started", 1
			switch mode {
			case "success":
				wantPhase = "observed"
			case "lost-reply", "wrong-ack":
				wantPhase = "dispatched"
			case "hidden-member", "indirect-member", "inventory-changed", "metadata-changed":
				wantPhase, wantStarts = "planned", 0
			}
			if state.Phase != wantPhase || starts != wantStarts {
				t.Fatal(state, starts)
			}
			if mode == "success" && (*state.Namespace != (projectnetwork.NamespaceIdentity{BootID: "00000000-0000-0000-0000-000000000001", Device: 4, Inode: 1001})) {
				t.Fatal(state)
			}
			if err = engine.StartAnchor(context.Background(), prepared); err == nil || starts != wantStarts {
				t.Fatal("adapter replayed or bypassed incomplete inspection", err)
			}
		})
	}
}
func TestProjectAnchorSessionRefusesChangedPreparationAndMissingWorkspace(t *testing.T) {
	for _, mode := range []string{"changed-contract", "changed-engine", "missing-workspace", "incomplete"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			c, n, v := resourceContractFixture(t)
			f.engine.ID = c.Preparation.EngineID
			p := c.Preparation
			p.Sequence = 6
			p.Bridge.Phase, p.Bridge.ID = "observed", n.ID
			p.Anchor.Phase, p.Anchor.ID = "observed", v.ID
			dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/project-resources-"+p.ProjectID)
			if mode != "missing-workspace" {
				if err := os.MkdirAll(filepath.Join(dir, "docker-config"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "changed-contract":
				p.Anchor.ContractDigest = strings.Repeat("e", 64)
			case "changed-engine":
				f.engine.ID = "different-engine"
			case "incomplete":
				p.Sequence = 5
				p.Anchor.Phase = "created"
			}
			calls := 0
			original := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				calls++
				if !reflect.DeepEqual(args[4:], []string{"image", "inspect", candidate.Image}) {
					t.Fatal("session setup mutated engine", args)
				}
				return original(ctx, args, env)
			}
			engine, err := openProjectAnchorStart(context.Background(), c.Selection, p, f.d)
			if err == nil || engine != nil {
				t.Fatal("unowned session admitted", err)
			}
			if mode != "changed-contract" && calls != 0 {
				t.Fatal("failed preflight reached Docker", calls)
			}
			if mode == "missing-workspace" {
				if _, err = os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("missing metadata recreated", err)
				}
			}
		})
	}
}
