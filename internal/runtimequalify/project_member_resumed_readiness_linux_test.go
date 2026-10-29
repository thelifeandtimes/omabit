//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func resumedReadinessAdapterFixture(t *testing.T, stoppedPeer bool) (*memberStopFixture, ProjectResumedReadinessSelection, string) {
	t.Helper()
	f, input, sentinel := resumeAdapterFixture(t, stoppedPeer)
	ctx := context.Background()
	source, err := newProjectMemberResumeEngine(ctx, input, f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	source.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	s := source.initial
	before := *input.Retirement.After
	before.CapturedAt = time.Now().UTC()
	before.Runtime.ObservedAt = before.CapturedAt
	s, err = s.Advance(input.Stop.Project.Plan, "planned", "create-dispatched", projectnetwork.MemberResumeResult{Before: &before})
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("8", 64)
	s, err = s.Advance(input.Stop.Project.Plan, "create-dispatched", "created", projectnetwork.MemberResumeResult{ContainerID: id})
	if err != nil {
		t.Fatal(err)
	}
	source.createdID = id
	v := projectMemberContainerFixture(t, source.contract)
	v.ID = id
	f.containers[id] = &v
	snapshot, err := source.snapshot(ctx, true, false)
	if err != nil {
		t.Fatal(err)
	}
	o := projectnetwork.MemberResumeObservation{Runtime: snapshot.Observation, Storage: input.Retirement.Exit.Stop.Storage, CapturedAt: time.Now().UTC()}
	s, err = s.Advance(input.Stop.Project.Plan, "created", "start-dispatched", projectnetwork.MemberResumeResult{Observation: &o})
	if err != nil {
		t.Fatal(err)
	}
	s, err = s.Advance(input.Stop.Project.Plan, "start-dispatched", "started", projectnetwork.MemberResumeResult{})
	if err != nil {
		t.Fatal(err)
	}
	*v.State.Running = true
	v.State.Status = "running"
	v.State.Pid = 99
	started := time.Now().UTC()
	v.State.StartedAt = started.Format(time.RFC3339Nano)
	snapshot, err = source.snapshot(ctx, true, true)
	if err != nil {
		t.Fatal(err)
	}
	o = projectnetwork.MemberResumeObservation{Runtime: snapshot.Observation, Storage: input.Retirement.Exit.Stop.Storage, StartedAt: started, CapturedAt: time.Now().UTC()}
	s, err = s.Advance(input.Stop.Project.Plan, "started", "running", projectnetwork.MemberResumeResult{Observation: &o})
	if err != nil {
		t.Fatal(err)
	}
	return f, ProjectResumedReadinessSelection{Resume: input, Completed: s}, sentinel
}
func TestProjectResumedReadinessAdapter(t *testing.T) {
	modes := []string{"pass", "stopped-peer", "live-lock", "unfinished", "changed-contract", "changed-generation", "wrong-ship", "http-error", "target-restarted", "peer-restarted", "stopped-peer-changed", "namespace-after-http", "engine-after-http", "urb-replaced", "parent-replaced", "unknown-container", "target-stopped", "unobserved-http", "arbitrary-endpoint", "stale-http", "cancelled", "caller-alias"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			f, input, sentinel := resumedReadinessAdapterFixture(t, strings.HasPrefix(mode, "stopped-peer"))
			plan := input.Resume.Stop.Project.Plan
			id := input.Completed.Container.ID
			var peer string
			for key := range f.containers {
				if key != id {
					peer = key
				}
			}
			path := f.paths[input.Completed.Retirement.Exit.Stop.InstanceID]
			bytes, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			oldStat, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			original := f.f.d.execute
			f.f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				// The namespace proof runs only a fixed read-only stat in a known container.
				namespaceRead := len(a) == 8 && a[0] == "container" && a[1] == "exec" && (a[2] == input.Completed.Retirement.Exit.Stop.Binding.Anchor.ID || f.containers[a[2]] != nil) && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"})
				if len(a) < 2 || !(namespaceRead || a[0] == "image" && a[1] == "inspect" || a[0] == "network" && a[1] == "inspect" || a[0] == "container" && (a[1] == "inspect" || a[1] == "ls")) {
					t.Fatal("readiness attempted mutation", a)
				}
				if namespaceRead {
					return fmt.Sprintf("%d:%d", f.ns.Device, f.ns.Inode), nil
				}
				return original(ctx, args, env)
			}
			switch mode {
			case "unfinished":
				input.Completed.Phase = "started"
				input.Completed.After = nil
			case "changed-contract":
				input.Completed.Container.ContractDigest = strings.Repeat("1", 64)
			case "changed-generation":
				input.Completed.PreviousGeneration = 2
			case "live-lock":
				if err = os.WriteFile(filepath.Join(path, ".vere.lock"), []byte("99"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			raw, err := openResumedProjectReadiness(ctx, input, f.f.d)
			if slices.Contains([]string{"unfinished", "changed-contract", "changed-generation"}, mode) {
				if err == nil {
					t.Fatal("invalid history opened")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			d := raw.(*projectResumedReadinessEngine)
			d.source.source.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
			reads := 0
			f.f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
				t.Fatal("unfrozen dependency used")
				return "", nil
			}
			d.source.source.source.anchor.base.q.d.hostIdentity = func(_ context.Context, endpoint string) (string, error) {
				reads++
				if endpoint != d.endpoint {
					t.Fatal("wrong publication")
				}
				must := func(err error) {
					t.Helper()
					if err != nil {
						t.Fatal(err)
					}
				}
				switch mode {
				case "wrong-ship":
					return "~bus", nil
				case "http-error":
					return "", errors.New("private response")
				case "target-restarted":
					f.containers[id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				case "peer-restarted":
					f.containers[peer].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				case "stopped-peer-changed":
					*f.containers[peer].State.ExitCode = 1
				case "namespace-after-http":
					f.ns.Inode++
				case "engine-after-http":
					f.f.engine.ID = "changed-engine"
				case "urb-replaced":
					must(os.Rename(filepath.Join(path, ".urb"), filepath.Join(path, ".urb-retained")))
					must(os.Mkdir(filepath.Join(path, ".urb"), 0700))
				case "parent-replaced":
					parent := filepath.Dir(path)
					must(os.Rename(parent, parent+"-retained"))
					must(os.Mkdir(parent, 0700))
				case "unknown-container":
					v := *f.containers[peer]
					v.ID = strings.Repeat("9", 64)
					v.Name = "/foreign"
					f.containers[v.ID] = &v
				case "target-stopped":
					f.exit(id, 0, false)
				}
				return d.ship, nil
			}
			if slices.Contains([]string{"unobserved-http", "arbitrary-endpoint", "stale-http"}, mode) {
				endpoint := d.endpoint
				if mode != "unobserved-http" {
					if _, err = d.Inspect(ctx, input.Completed); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "arbitrary-endpoint" {
					endpoint = "http://example.invalid/~/host"
				}
				if mode == "stale-http" {
					d.inspectedAt = time.Now().Add(-time.Minute)
				}
				if _, err = d.ReadHostIdentity(ctx, endpoint); err == nil || reads != 0 {
					t.Fatal("unbound HTTP allowed")
				}
				return
			}
			if mode == "cancelled" {
				cancel()
			}
			if mode == "caller-alias" {
				input.Completed.After.Runtime.Members[0].InstanceID = "caller-mutated"
				if reflect.DeepEqual(d.completed, input.Completed) {
					t.Fatal("selection aliases driver")
				}
				input.Completed = d.completed
			}
			r, err := projectnetwork.CollectResumedMemberReadiness(ctx, plan, input.Completed, d, time.Now)
			pass := slices.Contains([]string{"pass", "stopped-peer", "live-lock", "caller-alias"}, mode)
			if pass {
				if err != nil || r.Validate(plan, input.Completed) != nil || reads != 1 {
					t.Fatal("readiness failed", err)
				}
			} else if err == nil || !reflect.DeepEqual(r, projectnetwork.ResumedMemberReadiness{}) {
				t.Fatal("changed project accepted", r, err)
			}
			if !slices.Contains([]string{"urb-replaced", "parent-replaced"}, mode) {
				newStat, err := os.Stat(sentinel)
				if err != nil {
					t.Fatal(err)
				}
				actual, err := os.ReadFile(sentinel)
				if err != nil || string(actual) != string(bytes) || !os.SameFile(oldStat, newStat) {
					t.Fatal("saved state changed", err)
				}
			}
		})
	}
}
