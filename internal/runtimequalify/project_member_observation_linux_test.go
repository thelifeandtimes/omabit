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

func TestProjectMemberObservation(t *testing.T) {
	for _, mode := range []string{"empty", "created", "running", "old-namespace", "hidden", "indirect", "missing", "inventory-name", "inventory-mode", "anchor-inventory", "image", "mount", "restarted", "namespace-changed", "boot-changed", "anchor-restarted", "inventory-changed", "bridge-changed", "engine-changed", "cancelled", "wrong-contract", "duplicate", "batch-missing", "batch-extra", "batch-duplicate", "batch-reordered", "batch-foreign", "batch-missing-state", "bound-running", "bound-running-changed", "bound-created", "exited", "exited-failed", "exited-unrecorded", "exited-no-start", "exited-wrong-start", "exited-invalid-finish", "exited-changed"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _ := managedFixture(t)
			resource, network, anchor := resourceContractFixture(t)
			f.engine.ID = resource.Preparation.EngineID
			runningAnchorFixture(&anchor, &network)
			originalMember := projectMemberContractFixture(t)
			c, err := newProjectMemberContract(originalMember.selection, f.d.home, originalMember.mountSource, originalMember.imageID)
			if err != nil {
				t.Fatal(err)
			}
			prepared := c.selection.Anchor.Prepared
			dir := filepath.Join(f.d.home, ".local/share/urbit-runner/managed/project-resources-"+prepared.ProjectID)
			if err = os.MkdirAll(filepath.Join(dir, "docker-config"), 0700); err != nil {
				t.Fatal(err)
			}
			member := projectMemberContainerFixture(t, c)
			if mode != "created" && mode != "bound-created" {
				*member.State.Running = true
				member.State.Status = "running"
				member.State.Pid = 51
				member.State.StartedAt = "2026-10-23T12:00:01Z"
			}
			known := []projectObservedMember{{contract: c, id: member.ID}}
			if strings.HasPrefix(mode, "bound-") || strings.HasPrefix(mode, "exited") {
				known[0].processStartedAt = time.Date(2026, 10, 23, 12, 0, 1, 0, time.UTC)
			}
			if strings.HasPrefix(mode, "exited") {
				known[0].allowExited = true
				*member.State.Running = false
				member.State.Status, member.State.Pid = "exited", 0
				member.State.FinishedAt = "2026-10-23T12:00:02Z"
			}
			switch mode {
			case "exited-failed":
				*member.State.ExitCode, *member.State.OOMKilled = 137, true
			case "exited-unrecorded":
				known[0].allowExited = false
			case "exited-no-start":
				known[0].processStartedAt = time.Time{}
			case "exited-wrong-start", "bound-running-changed":
				known[0].processStartedAt = known[0].processStartedAt.Add(time.Nanosecond)
			case "exited-invalid-finish":
				member.State.FinishedAt = "0001-01-01T00:00:00Z"
			}
			items := []projectnetwork.NetworkContainer{{ID: anchor.ID, Name: anchor.Name, NetworkMode: anchor.HostConfig.NetworkMode}}
			if mode != "empty" && mode != "missing" {
				items = append(items, projectnetwork.NetworkContainer{ID: member.ID, Name: member.Name, NetworkMode: member.HostConfig.NetworkMode})
			}
			switch mode {
			case "empty":
				known = nil
			case "hidden", "indirect":
				target := anchor.ID
				if mode == "indirect" {
					target = strings.TrimPrefix(member.Name, "/")
				}
				items = append(items, projectnetwork.NetworkContainer{ID: strings.Repeat("d", 64), Name: "/unknown", NetworkMode: "container:" + target})
			case "inventory-name":
				items[1].Name = "/foreign"
			case "inventory-mode":
				items[1].NetworkMode = "none"
			case "anchor-inventory":
				items[0].Name = "/foreign"
			case "image":
				member.Image = "sha256:" + strings.Repeat("f", 64)
			case "mount":
				member.Mounts[0].Source += "-peer"
			case "wrong-contract":
				known[0].contract.selection.Project.ProjectID = "different"
			case "duplicate":
				known = append(known, known[0])
			}
			calls, memberReads, anchorReads, inventories, bridges := 0, 0, 0, 0, 0
			probes := map[string]int{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			original := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				calls++
				if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
					t.Fatal("unbound call", args)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded call")
				}
				a := args[4:]
				switch {
				case reflect.DeepEqual(a, []string{"image", "inspect", candidate.Image}):
					return original(ctx, args, env)
				case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
					var ids []string
					for _, item := range items {
						ids = append(ids, item.ID)
					}
					return strings.Join(ids, "\n") + "\n", nil
				case len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectInventoryFormat}):
					inventories++
					var ids []string
					var out strings.Builder
					for _, item := range items {
						ids = append(ids, item.ID)
						if mode == "inventory-changed" && inventories > 1 {
							item.Name += "-changed"
						}
						b, _ := json.Marshal(item)
						out.Write(b)
						out.WriteByte('\n')
					}
					slices.Sort(ids)
					if !reflect.DeepEqual(ids, a[4:]) {
						t.Fatal("inventory not bound to exact listed IDs")
					}
					return out.String(), nil
				case reflect.DeepEqual(a, []string{"network", "inspect", network.ID}):
					bridges++
					if mode == "bridge-changed" && bridges > 1 {
						network.Name += "-changed"
					}
					b, _ := json.Marshal([]bridge{network})
					return string(b), nil
				case len(a) > 3 && a[0] == "container" && a[1] == "inspect":
					var values []container
					for _, id := range a[2:] {
						switch id {
						case anchor.ID:
							anchorReads++
							if mode == "anchor-restarted" && anchorReads > 1 {
								anchor.State.Pid++
							}
							values = append(values, anchor)
						case member.ID:
							memberReads++
							if mode == "exited-changed" && memberReads > 1 {
								member.State.FinishedAt = "2026-10-23T12:00:03Z"
							}
							if mode == "restarted" && memberReads > 1 {
								member.State.Pid++
							}
							values = append(values, member)
						default:
							t.Fatal("unowned bulk inspection", id)
						}
					}
					switch mode {
					case "batch-missing":
						values = values[:1]
					case "batch-extra":
						values = append(values, member)
					case "batch-duplicate":
						values[1] = values[0]
					case "batch-foreign":
						values[1].ID = strings.Repeat("f", 64)
					case "batch-missing-state":
						values[1].State = nil
					case "batch-reordered":
						slices.Reverse(values)
					}
					raw, _ := json.Marshal(values)
					return string(raw), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", anchor.ID}):
					anchorReads++
					if mode == "anchor-restarted" && anchorReads > 1 {
						anchor.State.Pid++
					}
					b, _ := json.Marshal([]container{anchor})
					return string(b), nil
				case reflect.DeepEqual(a, []string{"container", "inspect", member.ID}):
					memberReads++
					if mode == "restarted" && memberReads > 1 {
						member.State.Pid++
					}
					b, _ := json.Marshal([]container{member})
					return string(b), nil
				case len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}):
					id := a[2]
					if id != anchor.ID && id != member.ID {
						t.Fatal("unowned namespace probe", id)
					}
					probes[id]++
					if id == member.ID {
						if mode == "namespace-changed" && probes[id] > 1 {
							return "5:44\n", nil
						}
						if mode == "old-namespace" {
							return "5:41\n", nil
						}
						if mode == "engine-changed" {
							f.engine.ID = "replacement-engine"
						}
						if mode == "cancelled" {
							cancel()
						}
					}
					return "5:42\n", nil
				default:
					t.Fatal("unexpected mutation or command", a)
					return "", errors.New("unexpected")
				}
			}
			raw, err := openProjectAnchorStart(ctx, resource.Selection, prepared, f.d)
			if err != nil {
				t.Fatal(err)
			}
			engine := raw.(*projectAnchorEngine)
			boots := 0
			engine.readBootID = func() (string, error) {
				boots++
				if mode == "boot-changed" && boots > 3 {
					return "12345678-1234-1234-1234-123456789abd", nil
				}
				return "12345678-1234-1234-1234-123456789abc", nil
			}
			engine.readyAt = time.Now() // observation must revoke stale initial-Start authority.
			before := calls
			snapshot, err := engine.inspectProjectMemberSnapshot(ctx, known)
			observation := snapshot.Observation
			success := mode == "exited" || mode == "exited-failed" || mode == "bound-running" || mode == "batch-reordered" || mode == "empty" || mode == "created" || mode == "running" || mode == "old-namespace"
			if (err == nil) != success {
				t.Fatalf("%s: %v", mode, err)
			}
			if !engine.readyAt.IsZero() {
				t.Fatal("member observation retained Start authority")
			}
			if mode == "wrong-contract" || mode == "duplicate" || mode == "exited-no-start" {
				if calls != before {
					t.Fatal("invalid selection reached engine")
				}
			}
			expectedReason := map[string]string{
				"wrong-contract":    "recorded member selection is invalid",
				"duplicate":         "recorded member selection is invalid",
				"exited-no-start":   "recorded member selection is invalid",
				"hidden":            "namespace membership does not match the recorded members",
				"missing":           "namespace membership does not match the recorded members",
				"restarted":         "member process or namespace changed during inspection",
				"namespace-changed": "member process or namespace changed during inspection",
				"exited-changed":    "member process or namespace changed during inspection",
			}[mode]
			if expectedReason != "" && (err == nil || !strings.HasSuffix(err.Error(), ": "+expectedReason)) {
				t.Fatalf("missing actionable refusal %q: %v", expectedReason, err)
			}
			if !success {
				if observation.Complete || len(snapshot.Containers) != 0 {
					t.Fatal("refused observation marked complete")
				}
				return
			}
			if !observation.Complete || len(observation.Members) != len(known) || len(observation.Anchor.Active) != 2 || !reflect.DeepEqual(observation.Anchor.Requested, observation.Anchor.Active) || observation.Anchor.Namespace != *c.selection.Anchor.Namespace {
				t.Fatal(observation)
			}
			if mode == "empty" {
				if probes[member.ID] != 0 {
					t.Fatal("unrecorded member probed")
				}
				return
			}
			if !reflect.DeepEqual(snapshot.Containers[member.ID], member) || !reflect.DeepEqual(snapshot.Containers[anchor.ID], anchor) || len(snapshot.Containers) != 2 {
				t.Fatal("snapshot lost bracketed full process evidence")
			}
			m := observation.Members[0]
			if m.InstanceID != c.selection.InstanceID || m.Resource.ID != member.ID || m.Resource.ContractDigest != c.digest || m.AnchorID != anchor.ID || !m.Present {
				t.Fatal(m)
			}
			switch mode {
			case "created", "exited", "exited-failed":
				if m.Running || m.Namespace != (projectnetwork.NamespaceIdentity{}) || probes[member.ID] != 0 {
					t.Fatal("created container acquired running namespace", m)
				}
			case "old-namespace":
				if !m.Running || m.Namespace.Inode != 41 || m.Namespace == observation.Anchor.Namespace {
					t.Fatal("old member namespace silently rebound", m)
				}
			case "running":
				if !m.Running || m.Namespace != observation.Anchor.Namespace || probes[member.ID] != 2 {
					t.Fatal(m)
				}
			}
		})
	}
}

func TestProjectMemberNamespaceRejectsNameBeforeEngineAccess(t *testing.T) {
	engine := &projectAnchorEngine{}
	for _, id := range []string{"anchor", strings.Repeat("a", 12), "", strings.Repeat("A", 64)} {
		if _, err := engine.namespaceFor(context.Background(), id); err == nil {
			t.Fatal("non-exact ID admitted", id)
		}
	}
}

func TestProjectContainerBatchRejectsUnboundIDsBeforeEngineAccess(t *testing.T) {
	e := &projectAnchorEngine{}
	id := strings.Repeat("a", 64)
	for _, ids := range [][]string{nil, {}, {"anchor"}, {id[:12]}, {strings.ToUpper(id)}, {id, id}, make([]string, projectInventoryLimit+1)} {
		if _, err := e.inspectProjectContainers(context.Background(), ids); err == nil {
			t.Fatal("invalid bulk IDs reached engine", ids)
		}
	}
}

func TestProjectMemberObservationFreshness(t *testing.T) {
	for _, elapsed := range []time.Duration{0, 30*time.Second - time.Nanosecond, 30 * time.Second} {
		if err := projectMemberObservationFreshness(elapsed); err != nil {
			t.Fatalf("valid freshness boundary %s refused: %v", elapsed, err)
		}
	}
	for _, elapsed := range []time.Duration{30*time.Second + time.Nanosecond, 61 * time.Second} {
		err := projectMemberObservationFreshness(elapsed)
		if err == nil || !strings.Contains(err.Error(), "exceeded 30s freshness limit") || !strings.HasSuffix(err.Error(), "ownership remains unconfirmed") {
			t.Fatalf("stale observation %s lacked a distinct refusal: %v", elapsed, err)
		}
	}
}
