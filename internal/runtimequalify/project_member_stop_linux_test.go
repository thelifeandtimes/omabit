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

type memberStopFixture struct {
	f          *fakeEngine
	selection  ProjectMemberStopSelection
	containers map[string]*container
	paths      map[string]string
	readiness  map[string]projectnetwork.InitialMemberReadiness
	ns         projectnetwork.NamespaceIdentity
	signals    int
	signal     func(context.Context, string) (string, error)
	singleRead func(string)
}

func memberStopAdapterFixture(t *testing.T) *memberStopFixture {
	t.Helper()
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
	fixture := &memberStopFixture{f: f, containers: map[string]*container{}, paths: paths, readiness: map[string]projectnetwork.InitialMemberReadiness{}, ns: ns}
	for i := range containers {
		fixture.containers[containers[i].ID] = &containers[i]
	}
	original := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) < 6 || args[0] != "--host" || args[2] != "--config" || args[3] != filepath.Join(dir, "docker-config") {
			t.Fatal("unbound Docker command", args)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded Docker command")
		}
		a := args[4:]
		marshal := func(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }
		switch {
		case reflect.DeepEqual(a, []string{"image", "inspect", candidate.Image}):
			return original(ctx, args, env)
		case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
			ids := []string{anchor.ID}
			for id := range fixture.containers {
				ids = append(ids, id)
			}
			slices.Sort(ids)
			return strings.Join(ids, "\n"), nil
		case len(a) > 4 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectInventoryFormat}):
			items := []projectnetwork.NetworkContainer{{ID: anchor.ID, Name: anchor.Name, NetworkMode: anchor.HostConfig.NetworkMode}}
			for _, v := range fixture.containers {
				items = append(items, projectnetwork.NetworkContainer{ID: v.ID, Name: v.Name, NetworkMode: v.HostConfig.NetworkMode})
			}
			slices.SortFunc(items, func(a, b projectnetwork.NetworkContainer) int { return strings.Compare(a.ID, b.ID) })
			var ids []string
			var out strings.Builder
			for _, item := range items {
				ids = append(ids, item.ID)
				b, _ := json.Marshal(item)
				out.Write(b)
				out.WriteByte('\n')
			}
			if !reflect.DeepEqual(ids, a[4:]) {
				t.Fatal("unbound inventory")
			}
			return out.String(), nil
		case reflect.DeepEqual(a, []string{"network", "inspect", network.ID}):
			return marshal([]bridge{network})
		case len(a) >= 3 && a[0] == "container" && a[1] == "inspect":
			var values []container
			for _, id := range a[2:] {
				if id == anchor.ID {
					values = append(values, anchor)
					continue
				}
				v := fixture.containers[id]
				if v == nil {
					t.Fatal("unowned inspect", id)
				}
				if len(a) == 3 && fixture.singleRead != nil {
					fixture.singleRead(id)
				}
				values = append(values, *v)
			}
			return marshal(values)
		case len(a) == 8 && a[0] == "container" && a[1] == "exec" && reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}):
			if a[2] != anchor.ID {
				v := fixture.containers[a[2]]
				if v == nil || !*v.State.Running {
					t.Fatal("namespace probe of unowned/stopped container")
				}
			}
			return "5:42\n", nil
		case len(a) == 5 && reflect.DeepEqual(a[:4], []string{"container", "kill", "--signal", "SIGTERM"}):
			if fixture.containers[a[4]] == nil {
				t.Fatal("unowned signal")
			}
			fixture.signals++
			return fixture.signal(ctx, a[4])
		default:
			t.Fatal("unexpected command, escalation or removal", a)
			return "", errors.New("unexpected")
		}
	}
	f.d.hostIdentity = func(ctx context.Context, endpoint string) (string, error) {
		for _, m := range resource.Selection.Plan.Members {
			if endpoint == fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) {
				return m.Ship, nil
			}
		}
		t.Fatal("unbound HTTP")
		return "", errors.New("unexpected")
	}
	raw, err := openInitialProjectReadiness(context.Background(), ProjectInitialReadinessSelection{Project: resource.Selection, Launches: launches}, f.d)
	must(err)
	readyDriver := raw.(*projectInitialReadinessEngine)
	readyDriver.anchor.readBootID = func() (string, error) { return ns.BootID, nil }
	for _, m := range resource.Selection.Plan.Members {
		proof, err := projectnetwork.CollectInitialMemberReadiness(context.Background(), resource.Selection.Plan, launches, m.InstanceID, readyDriver, time.Now)
		must(err)
		fixture.readiness[m.InstanceID] = proof
	}
	proof := fixture.readiness["sender"]
	stop, err := projectnetwork.NewInitialMemberStop(resource.Selection.Plan, launches, proof, "ready-sender", "stop-sender")
	must(err)
	fixture.selection = ProjectMemberStopSelection{Project: resource.Selection, Launches: launches, Readiness: proof, Stop: stop}
	fixture.signal = func(ctx context.Context, id string) (string, error) {
		fixture.exit(id, 0, false)
		return id + "\n", nil
	}
	return fixture
}
func (f *memberStopFixture) exit(id string, code int, oom bool) {
	v := f.containers[id]
	*v.State.Running = false
	v.State.Status = "exited"
	v.State.Pid = 0
	*v.State.ExitCode = code
	*v.State.OOMKilled = oom
	v.State.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
}
func (f *memberStopFixture) open(t *testing.T) *projectMemberStopEngine {
	t.Helper()
	raw, err := openProjectMemberStop(context.Background(), f.selection, f.f.d)
	if err != nil {
		t.Fatal(err)
	}
	e := raw.(*projectMemberStopEngine)
	e.source.anchor.readBootID = func() (string, error) { return f.ns.BootID, nil }
	return e
}

func TestProjectMemberStopAdapter(t *testing.T) {
	for _, mode := range []string{"pass", "delayed-exit", "signal-error", "signal-bad-ack", "cancel-signal", "no-exit", "nonzero", "oom", "peer-changed", "target-restarted", "pre-signal-restart", "pre-signal-pier", "pre-signal-engine", "commit-dispatch", "commit-signalled", "commit-stopped", "peer-exited-unrecorded"} {
		t.Run(mode, func(t *testing.T) {
			f := memberStopAdapterFixture(t)
			if mode == "no-exit" {
				f.f.d.times.stop = 30 * time.Millisecond
			}
			e := f.open(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			initial := f.selection.Stop
			persisted := initial
			phases := []string{}
			originalSignal := f.signal
			polls := 0
			f.singleRead = func(id string) {
				if mode == "delayed-exit" && f.signals > 0 && id == e.target.id {
					polls++
					if polls == 3 {
						f.exit(id, 0, false)
					}
				}
			}
			f.signal = func(ctx context.Context, id string) (string, error) {
				if persisted.Phase != "stop-dispatched" || id != e.target.id {
					t.Fatal("signal before exact durable intent")
				}
				switch mode {
				case "signal-error":
					return "secret engine output", errors.New("secret engine error")
				case "signal-bad-ack":
					return "wrong-id", nil
				case "cancel-signal":
					cancel()
					return id, nil
				case "no-exit", "delayed-exit":
					return id, nil
				case "nonzero":
					f.exit(id, 143, false)
					return id, nil
				case "oom":
					f.exit(id, 137, true)
					return id, nil
				case "peer-changed":
					for key, v := range f.containers {
						if key != id {
							v.State.Pid++
						}
					}
				case "target-restarted":
					f.containers[id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					return id, nil
				}
				return originalSignal(ctx, id)
			}
			if mode == "peer-exited-unrecorded" {
				for id := range f.containers {
					if id != e.target.id {
						f.exit(id, 0, false)
					}
				}
			}
			commit := func(expected string, next projectnetwork.MemberStop) error {
				if expected != persisted.Phase {
					t.Fatal("stale commit")
				}
				fail := mode == "commit-dispatch" && next.Phase == "stop-dispatched" || mode == "commit-signalled" && next.Phase == "stop-signalled" || mode == "commit-stopped" && next.Phase == "stopped"
				if fail {
					return errors.New("retained database failure")
				}
				persisted = next
				phases = append(phases, next.Phase)
				if next.Phase == "stop-dispatched" {
					switch mode {
					case "pre-signal-restart":
						f.containers[e.target.id].State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
					case "pre-signal-engine":
						f.f.engine.ID = "replaced-engine"
					case "pre-signal-pier":
						path := f.paths[initial.InstanceID]
						if err := os.Rename(filepath.Join(path, ".urb"), filepath.Join(path, ".urb-retained")); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(filepath.Join(path, ".urb"), 0700); err != nil {
							t.Fatal(err)
						}
					}
				}
				return nil
			}
			result, err := projectnetwork.ExecuteMemberStop(ctx, f.selection.Project.Plan, initial, e, commit, time.Now)
			pass := mode == "pass" || mode == "delayed-exit"
			if (err == nil) != pass || strings.Contains(fmt.Sprint(err), "secret") {
				t.Fatalf("%s: result=%s error=%v", mode, result.Phase, err)
			}
			expectedSignals := 1
			if strings.HasPrefix(mode, "pre-signal-") || mode == "commit-dispatch" || mode == "peer-exited-unrecorded" {
				expectedSignals = 0
			}
			if f.signals != expectedSignals {
				t.Fatal("wrong signal count", f.signals, expectedSignals)
			}
			if pass || mode == "nonzero" || mode == "oom" {
				if result.Phase != "stopped" || result.Clean() != pass || !reflect.DeepEqual(phases, []string{"stop-dispatched", "stop-signalled", "stopped"}) {
					t.Fatal("exit evidence missing", result.Phase, phases)
				}
			} else if result.Phase == "stopped" {
				t.Fatal("uncertain operation gained completed exit")
			}
			if !reflect.DeepEqual(result, persisted) {
				t.Fatal("returned state differs from last durable state")
			}
			before := f.signals
			if e.SignalStop(context.Background(), persisted) == nil || f.signals != before {
				t.Fatal("signal replay allowed")
			}
			if len(f.containers) != 2 || f.f.downloads != 0 {
				t.Fatal("resource removed or downloaded")
			}
			for _, path := range f.paths {
				if _, err := os.Stat(filepath.Join(path, ".urb")); err != nil {
					t.Fatal("pier removed", err)
				}
			}
		})
	}
}

func TestProjectMemberStopAdapterBindsInitialIntentAndCannotInventAcknowledgement(t *testing.T) {
	for _, mode := range []string{"changed-intent", "wrong-ready", "partial-launches", "unresolved-peer", "foreign-before", "stale-before", "await-without-signal", "caller-alias"} {
		t.Run(mode, func(t *testing.T) {
			f := memberStopAdapterFixture(t)
			switch mode {
			case "changed-intent":
				f.selection.Stop.OperationID = "changed"
			case "wrong-ready":
				f.selection.Readiness.Ship = "~nec"
			case "partial-launches":
				f.selection.Launches = f.selection.Launches[:1]
			case "unresolved-peer":
				f.selection.StoppedPeers = []projectnetwork.MemberStop{f.selection.Stop}
			}
			if mode == "changed-intent" {
				// Changing only the operation ID is another valid fresh intent; an altered
				// binding, rather than a new request identifier, must be refused here.
				f.selection.Stop.Storage.Identity.Inode++
			}
			if mode == "changed-intent" || mode == "wrong-ready" || mode == "partial-launches" || mode == "unresolved-peer" {
				if _, err := openProjectMemberStop(context.Background(), f.selection, f.f.d); err == nil {
					t.Fatal("invalid selection accepted")
				}
				if f.signals != 0 {
					t.Fatal("setup signalled")
				}
				return
			}
			e := f.open(t)
			initial := f.selection.Stop
			o, err := e.Inspect(context.Background(), initial)
			if err != nil {
				t.Fatal(err)
			}
			next, err := initial.Advance(f.selection.Project.Plan, "planned", "stop-dispatched", &o)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "foreign-before":
				next.Before.StartedAt = next.Before.StartedAt.Add(time.Nanosecond)
			case "stale-before":
				next.Before.CapturedAt = next.Before.CapturedAt.Add(-31 * time.Second)
				next.Before.Runtime.ObservedAt = next.Before.Runtime.ObservedAt.Add(-31 * time.Second)
				// Model a stalled dispatch of the exact observation retained by the adapter.
				frozen := *next.Before
				e.before = &frozen
			case "await-without-signal":
				next, err = next.Advance(f.selection.Project.Plan, "stop-dispatched", "stop-signalled", nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = e.AwaitStopped(context.Background(), next); err == nil {
					t.Fatal("invented signal acknowledgement")
				}
				return
			case "caller-alias":
				o.Runtime.Members[0].Resource.ID = strings.Repeat("0", 64)
				f.selection.Stop.Binding.Members[0].Container.ID = strings.Repeat("1", 64)
				if e.before.Runtime.Members[0].Resource.ID == o.Runtime.Members[0].Resource.ID || e.initial.Binding.Members[0].Container.ID == f.selection.Stop.Binding.Members[0].Container.ID {
					t.Fatal("caller mutated retained authority")
				}
				if err = e.SignalStop(context.Background(), next); err != nil {
					t.Fatal("valid frozen intent changed", err)
				}
				return
			}
			if e.SignalStop(context.Background(), next) == nil || f.signals != 0 {
				t.Fatal("unbound/stale signal allowed")
			}
		})
	}
}

func TestProjectMemberStopAdapterSequentialPeerExit(t *testing.T) {
	for _, mode := range []string{"pass", "failed-peer", "peer-restarted", "peer-exit-changed", "duplicate-peer"} {
		t.Run(mode, func(t *testing.T) {
			f := memberStopAdapterFixture(t)
			if mode == "failed-peer" {
				f.signal = func(ctx context.Context, id string) (string, error) { f.exit(id, 137, true); return id, nil }
			}
			e := f.open(t)
			commit := func(expected string, next projectnetwork.MemberStop) error { return nil }
			stopped, err := projectnetwork.ExecuteMemberStop(context.Background(), f.selection.Project.Plan, f.selection.Stop, e, commit, time.Now)
			if stopped.Phase != "stopped" || (err == nil) != (mode != "failed-peer") {
				t.Fatal("first exit", err)
			}
			proof := f.readiness["receiver"]
			next, err := projectnetwork.NewInitialMemberStop(f.selection.Project.Plan, f.selection.Launches, proof, "ready-receiver", "stop-receiver")
			if err != nil {
				t.Fatal(err)
			}
			f.selection.Readiness, f.selection.Stop, f.selection.StoppedPeers = proof, next, []projectnetwork.MemberStop{stopped}
			switch mode {
			case "peer-restarted":
				v := f.containers[e.target.id]
				*v.State.Running = true
				v.State.Status = "running"
				v.State.Pid = 61
				*v.State.ExitCode = 0
				*v.State.OOMKilled = false
				v.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				v.State.FinishedAt = "0001-01-01T00:00:00Z"
			case "peer-exit-changed":
				f.containers[e.target.id].State.FinishedAt = time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
			case "duplicate-peer":
				f.selection.StoppedPeers = append(f.selection.StoppedPeers, stopped)
				if _, err = openProjectMemberStop(context.Background(), f.selection, f.f.d); err == nil {
					t.Fatal("duplicate peer accepted")
				}
				return
			}
			f.signal = func(ctx context.Context, id string) (string, error) { f.exit(id, 0, false); return id, nil }
			e = f.open(t)
			result, err := projectnetwork.ExecuteMemberStop(context.Background(), f.selection.Project.Plan, next, e, commit, time.Now)
			pass := mode == "pass" || mode == "failed-peer"
			if (err == nil) != pass || pass && !result.Clean() || f.signals != map[bool]int{true: 2, false: 1}[pass] {
				t.Fatal("second Stop", mode, result.Phase, f.signals, err)
			}
		})
	}
}
