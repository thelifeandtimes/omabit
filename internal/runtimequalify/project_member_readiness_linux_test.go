//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func TestProjectInitialReadinessAdapter(t *testing.T) {
	for _, mode := range []string{"pass", "partial-launches", "contract-changed", "wrong-ship", "http-error", "missing-urb", "symlink-urb", "urb-replaced", "parent-replaced", "peer-stopped", "peer-namespace", "namespace-after-http", "engine-after-http", "unobserved-http", "arbitrary-endpoint", "stale-http", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			resource, network, anchor := resourceContractFixture(t)
			f.engine.ID = resource.Preparation.EngineID
			runningAnchorFixture(&anchor, &network)
			prepared := resource.Preparation
			prepared.Sequence = 6
			prepared.Bridge.Phase, prepared.Bridge.ID = "observed", network.ID
			prepared.Anchor.Phase, prepared.Anchor.ID = "observed", anchor.ID
			ns := projectnetwork.NamespaceIdentity{BootID: "12345678-1234-1234-1234-123456789abc", Device: 5, Inode: 42}
			started := projectnetwork.AnchorStart{Prepared: prepared, OperationID: "start-anchor", Phase: "observed", Namespace: &ns}
			base := filepath.Join(f.d.home, ".local/share/urbit-runner/managed")
			dir := filepath.Join(base, "project-resources-"+prepared.ProjectID)
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			must(os.MkdirAll(filepath.Join(dir, "docker-config"), 0700))
			must(pierstore.Init(f.d.home))
			var launches []projectnetwork.MemberLaunch
			var peers []projectnetwork.RuntimeMember
			var containers []container
			paths := map[string]string{}
			for index, m := range resource.Selection.Plan.Members {
				runtime := strings.Repeat(string(rune('e'+index)), 32)
				storage, err := projectnetwork.NewMemberStorage(resource.Selection.Plan, prepared.ProjectID, 1, m.InstanceID, runtime, "storage-"+m.InstanceID)
				must(err)
				parent, err := pierstore.PrepareMount(f.d.home, &storage.Location)
				must(err)
				storage.Phase = "observed"
				c, err := newProjectMemberContract(projectMemberSelection{Project: resource.Selection, Anchor: started, InstanceID: m.InstanceID, RuntimeID: runtime, OperationID: "launch-" + m.InstanceID, ContainerGeneration: 1, Mode: "boot", Storage: storage.Location}, f.d.home, parent, candidate.ManifestDigest)
				must(err)
				container := projectMemberContainerFixture(t, c)
				container.ID = strings.Repeat(string(rune('c'+index)), 64)
				*container.State.Running = true
				container.State.Status = "running"
				container.State.Pid = 51 + index
				container.State.StartedAt = "2026-10-23T12:00:01Z"
				containers = append(containers, container)
				path := filepath.Join(parent, strings.TrimPrefix(m.Ship, "~"))
				paths[m.InstanceID] = path
				must(os.Mkdir(path, 0700))
				must(os.Mkdir(filepath.Join(path, ".urb"), 0700))
				workspace := filepath.Dir(c.pillPath)
				must(os.Mkdir(workspace, 0700))
				must(os.WriteFile(c.pillPath, []byte("tiny disposable test pill\n"), 0400))
				identity, err := pierstore.IdentifyMount(workspace)
				must(err)
				launch, err := projectnetwork.NewMemberLaunch(resource.Selection.Plan, started, storage, c.selection.OperationID, c.digest, peers)
				must(err)
				launch.Phase = "running"
				launch.Workspace = identity
				launch.Container.ID = container.ID
				launch.Namespace = &ns
				must(launch.Validate(resource.Selection.Plan))
				launches = append(launches, launch)
				peers = append(peers, projectnetwork.RuntimeMember{InstanceID: m.InstanceID, RuntimeID: runtime, Container: launch.Container})
			}
			selected := ProjectInitialReadinessSelection{Project: resource.Selection, Launches: launches}
			switch mode {
			case "partial-launches":
				selected.Launches = selected.Launches[:1]
			case "contract-changed":
				selected.Launches[1].Container.ContractDigest = strings.Repeat("f", 64)
			case "missing-urb":
				must(os.Rename(filepath.Join(paths["sender"], ".urb"), filepath.Join(base, "retained-urb")))
			case "symlink-urb":
				must(os.Rename(filepath.Join(paths["sender"], ".urb"), filepath.Join(base, "retained-urb")))
				must(os.Symlink(filepath.Join(base, "retained-urb"), filepath.Join(paths["sender"], ".urb")))
			case "peer-stopped":
				*containers[0].State.Running = false
				containers[0].State.Status = "exited"
				containers[0].State.Pid = 0
				containers[0].State.FinishedAt = "2026-10-23T12:01:00Z"
			}
			readCount := 0
			namespaceChanged := false
			original := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
					t.Fatal("unbound Docker read", args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded Docker read")
				}
				a := args[4:]
				items := []projectnetwork.NetworkContainer{{ID: anchor.ID, Name: anchor.Name, NetworkMode: anchor.HostConfig.NetworkMode}}
				for _, v := range containers {
					items = append(items, projectnetwork.NetworkContainer{ID: v.ID, Name: v.Name, NetworkMode: v.HostConfig.NetworkMode})
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
					if a[3] != projectInventoryFormat {
						t.Fatal("unexpected formatted read")
					}
					var ids []string
					for _, v := range items {
						ids = append(ids, v.ID)
					}
					slices.Sort(ids)
					if !reflect.DeepEqual(ids, a[4:]) {
						t.Fatal("unbound inventory")
					}
					var out strings.Builder
					for _, v := range items {
						raw, _ := json.Marshal(v)
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
							continue
						}
						found := false
						for _, member := range containers {
							if id == member.ID {
								values = append(values, member)
								found = true
								break
							}
						}
						if !found {
							t.Fatal("unowned bulk inspection", id)
						}
					}
					raw, _ := json.Marshal(values)
					return string(raw), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", anchor.ID}):
					raw, _ := json.Marshal([]container{anchor})
					return string(raw), nil
				case len(a) == 3 && a[0] == "container" && a[1] == "inspect":
					for _, v := range containers {
						if a[2] == v.ID {
							raw, _ := json.Marshal([]container{v})
							return string(raw), nil
						}
					}
				case len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}):
					if a[2] != anchor.ID && a[2] != containers[0].ID && a[2] != containers[1].ID {
						t.Fatal("unowned namespace probe")
					}
					if namespaceChanged || mode == "peer-namespace" && a[2] == containers[0].ID {
						return "5:43\n", nil
					}
					return "5:42\n", nil
				}
				t.Fatal("unexpected effect or command", a)
				return "", errors.New("unexpected")
			}
			f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
				readCount++
				if endpoint != "http://127.0.0.1:18281/~/host" {
					t.Fatal("unbound HTTP endpoint", endpoint)
				}
				switch mode {
				case "wrong-ship":
					return "~nec", nil
				case "http-error":
					return "", errors.New("secret provider response")
				case "urb-replaced":
					must(os.Rename(filepath.Join(paths["sender"], ".urb"), filepath.Join(base, "retained-urb")))
					must(os.Mkdir(filepath.Join(paths["sender"], ".urb"), 0700))
				case "parent-replaced":
					parent := filepath.Dir(paths["sender"])
					must(os.Rename(parent, filepath.Join(base, "retained-parent")))
					must(os.Mkdir(parent, 0700))
				case "namespace-after-http":
					namespaceChanged = true
				case "engine-after-http":
					f.engine.ID = "different-engine"
				}
				return "~zod", nil
			}
			raw, err := openInitialProjectReadiness(context.Background(), selected, f.d)
			if mode == "partial-launches" || mode == "contract-changed" {
				if err == nil || raw != nil || readCount != 0 {
					t.Fatal("invalid selection admitted", err)
				}
				return
			}
			must(err)
			driver := raw.(*projectInitialReadinessEngine)
			driver.anchor.readBootID = func() (string, error) { return ns.BootID, nil }
			if mode == "unobserved-http" || mode == "arbitrary-endpoint" || mode == "stale-http" {
				endpoint := "http://127.0.0.1:18281/~/host"
				if mode != "unobserved-http" {
					_, err = driver.InspectProject(context.Background())
					must(err)
				}
				if mode == "arbitrary-endpoint" {
					endpoint = "http://127.0.0.1:19999/~/host"
				}
				if mode == "stale-http" {
					driver.inspectedAt = time.Now().Add(-31 * time.Second)
				}
				if _, err = driver.ReadHostIdentity(context.Background(), endpoint); err == nil || readCount != 0 {
					t.Fatal("uninspected HTTP request allowed")
				}
				return
			}
			before, _ := json.Marshal(launches)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			proof, err := projectnetwork.CollectInitialMemberReadiness(ctx, resource.Selection.Plan, launches, "sender", driver, time.Now)
			after, _ := json.Marshal(launches)
			if string(before) != string(after) {
				t.Fatal("original launch receipts changed")
			}
			if mode == "pass" {
				if err != nil || proof.Validate(resource.Selection.Plan, launches) != nil || readCount != 1 || proof.Storage.Identity.Inode == 0 || proof.Storage.Identity.UrbInode == 0 {
					t.Fatal(proof, err)
				}
			} else if err == nil || !reflect.DeepEqual(proof, projectnetwork.InitialMemberReadiness{}) || strings.Contains(fmt.Sprint(err), "secret") {
				t.Fatal("invalid readiness accepted", proof, err)
			}
			if f.downloads != 0 {
				t.Fatal("readiness downloaded an artifact")
			}
		})
	}
}
