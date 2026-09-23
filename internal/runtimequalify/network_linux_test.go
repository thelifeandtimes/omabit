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
	"strings"
	"testing"
	"time"
)

type fakeNetwork struct {
	id, name string
	doc      map[string]any
}

func (f *fakeEngine) networkByID(id string) *fakeNetwork {
	for _, n := range f.networks {
		if n.id == id {
			return n
		}
	}
	return nil
}

func (f *fakeEngine) networkCommand(args []string, config string) (string, error) {
	t := f.t
	if args[1] == "ls" {
		if !reflect.DeepEqual(args, []string{"network", "ls", "--filter", "label=" + label + "=true", "--format", "{{.Name}}"}) {
			t.Fatalf("unscoped network inventory: %q", args)
		}
		var names []string
		for name := range f.networks {
			names = append(names, name)
		}
		return strings.Join(names, "\n"), nil
	}
	if f.d.profile != ProfileHTTP {
		t.Fatal("network mutation in lifecycle profile")
	}
	switch args[1] {
	case "create":
		var report Result
		data, err := os.ReadFile(filepath.Join(filepath.Dir(config), "report.json"))
		if err != nil || json.Unmarshal(data, &report) != nil || report.Network == nil || !report.Network.CreateDispatched || !report.Network.CreateUncertain {
			t.Fatal("network intent was not persisted before dispatch")
		}
		name := report.Network.Name
		want := []string{"network", "create", "--driver", "bridge", "--internal=false", "--ipv4=true", "--ipv6=false", "--label", label + "=true", "--label", runLabel + "=" + report.RunID,
			"--opt", "com.docker.network.bridge.host_binding_ipv4=127.0.0.1", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=nat", name}
		if !reflect.DeepEqual(args, want) || f.networks[name] != nil {
			t.Fatalf("unsafe/repeated network create: %q", args)
		}
		id := strings.Repeat("e", 64)
		n := &fakeNetwork{id, name, map[string]any{
			"Id": id, "Name": name, "Driver": "bridge", "Scope": "local", "Internal": false, "EnableIPv4": true,
			"EnableIPv6": false, "Attachable": false, "Ingress": false, "ConfigOnly": false,
			"Options": bridgeOptions(), "Labels": map[string]any{label: "true", runLabel: report.RunID}, "Containers": map[string]any{},
		}}
		f.networks[name] = n
		f.events = append(f.events, "network-create")
		return id + "\n", nil
	case "inspect":
		n := f.networks[args[2]]
		if n == nil {
			return "", errors.New("missing network")
		}
		doc := copyDoc(t, n.doc)
		if f.editNetwork != nil {
			f.editNetwork(doc)
		}
		return encode(t, []any{doc}), nil
	case "rm":
		if len(args) != 3 || !containerIDPattern.MatchString(args[2]) {
			t.Fatal("network removal requires a full verified ID")
		}
		n := f.networkByID(args[2])
		if n == nil || len(mapAt(n.doc, "Containers")) != 0 {
			t.Fatal("removing an unknown or attached network")
		}
		f.events = append(f.events, "network-rm")
		delete(f.networks, n.name)
		return args[2] + "\n", nil
	default:
		t.Fatalf("unexpected network command: %q", args)
	}
	return "", nil
}

func (f *fakeEngine) publishHTTP(c *fakeContainer) {
	n := f.networkByID(mapAt(c.doc, "HostConfig")["NetworkMode"].(string))
	if n == nil {
		f.t.Fatal("missing owned HTTP network")
	}
	port := "38101"
	if c.kind == "http-restart" {
		port = "38102"
	}
	mapAt(n.doc, "Containers")[c.id] = map[string]any{"Name": c.name}
	c.doc["NetworkSettings"] = map[string]any{
		"Networks": map[string]any{n.name: map[string]any{"NetworkID": n.id}},
		"Ports":    map[string]any{"80/tcp": nil, "34343/udp": nil, "8080/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": port}}},
	}
}

func httpFixture(t *testing.T) *fakeEngine {
	f := fixture(t)
	f.d.profile = ProfileHTTP
	f.d.hostProbe = func(ctx context.Context, endpoint string) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded host probe")
		}
		for _, c := range f.containers {
			if isHTTPKind(c.kind) && mapAt(c.doc, "State")["Running"] == true {
				port := "38101"
				if c.kind == "http-restart" {
					port = "38102"
				}
				if endpoint != "http://127.0.0.1:"+port+"/~/host" {
					t.Fatalf("unsafe host probe endpoint: %s", endpoint)
				}
				f.events = append(f.events, "host-http-"+c.kind)
				return nil
			}
		}
		t.Fatal("host probe without a live fake ship")
		return errors.New("missing")
	}
	return f
}

func TestHTTPCheckpointAndReplacement(t *testing.T) {
	f := httpFixture(t)
	var phases []string
	r, err := run(context.Background(), func(p string) { phases = append(phases, p) }, f.d)
	if err != nil || !r.CheckpointPassed || !r.CleanupComplete || r.RuntimeQualified || r.Profile != ProfileHTTP {
		t.Fatalf("HTTP workflow failed: %+v, %v", r, err)
	}
	if !reflect.DeepEqual(phases, []string{"engine", "image", "pill", "probe", "network", "host-http-boot", "host-http-restart", "cleanup"}) {
		t.Fatalf("unexpected phases: %v", phases)
	}
	if len(r.Containers) != 3 || r.Network == nil || !r.Network.Removed || r.Network.CreateUncertain || len(f.networks) != 0 || len(f.containers) != 0 {
		t.Fatalf("resources not cleaned: %+v", r)
	}
	for _, c := range r.Containers[1:] {
		if !c.HostHTTPVerified || c.HostEndpoint == "" || c.PrivateHTTPReachable || !c.Removed {
			t.Fatalf("host evidence missing or confused with in-container probe: %+v", c)
		}
	}
	if r.Containers[1].HostEndpoint == r.Containers[2].HostEndpoint {
		t.Fatal("replacement did not rediscover the assigned host endpoint")
	}
	if f.events[len(f.events)-1] != "network-rm" {
		t.Fatalf("network cleanup order: %v", f.events)
	}
	assertReport(t, r)
}

func TestHTTPUnsafeNetworkNeverStartsShip(t *testing.T) {
	for _, fault := range []string{"label", "id", "driver", "scope", "internal", "ipv4", "ipv6", "gateway", "foreign attachment"} {
		t.Run(fault, func(t *testing.T) {
			f := httpFixture(t)
			f.editNetwork = func(n map[string]any) {
				switch fault {
				case "label":
					mapAt(n, "Labels")[runLabel] = "other"
				case "id":
					n["Id"] = strings.Repeat("f", 64)
				case "driver":
					n["Driver"] = "host"
				case "scope":
					n["Scope"] = "swarm"
				case "internal":
					n["Internal"] = true
				case "ipv4":
					n["EnableIPv4"] = false
				case "ipv6":
					n["EnableIPv6"] = true
				case "gateway":
					mapAt(n, "Options")["com.docker.network.bridge.gateway_mode_ipv4"] = "routed"
				case "foreign attachment":
					mapAt(n, "Containers")[strings.Repeat("c", 64)] = map[string]any{"Name": "other-ship"}
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || r.CleanupComplete || len(r.Containers) != 1 || len(f.networks) != 1 {
				t.Fatalf("unsafe network accepted or removed: %+v, %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestHTTPUnsafePublicationNeverProbed(t *testing.T) {
	for _, fault := range []string{"requested wildcard", "actual wildcard", "ipv6", "other port", "extra network", "wrong network ID", "privileged port"} {
		t.Run(fault, func(t *testing.T) {
			f := httpFixture(t)
			f.d.hostProbe = func(context.Context, string) error { t.Fatal("unsafe binding reached the host client"); return nil }
			f.editInspect = func(kind string, doc map[string]any) {
				if !isHTTPKind(kind) {
					return
				}
				h, n := mapAt(doc, "HostConfig"), mapAt(doc, "NetworkSettings")
				switch fault {
				case "requested wildcard":
					h["PortBindings"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": ""}}}
				case "actual wildcard", "ipv6", "privileged port":
					ip, port := "127.0.0.1", "38101"
					if fault == "actual wildcard" {
						ip = "0.0.0.0"
					}
					if fault == "ipv6" {
						ip = "::1"
					}
					if fault == "privileged port" {
						port = "80"
					}
					n["Ports"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": ip, "HostPort": port}}}
				case "other port":
					n["Ports"] = map[string]any{"34343/udp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "38101"}}}
				case "extra network":
					mapAt(n, "Networks")["bridge"] = map[string]any{}
				case "wrong network ID":
					for name := range mapAt(n, "Networks") {
						mapAt(n, "Networks")[name] = map[string]any{"NetworkID": strings.Repeat("f", 64)}
					}
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || len(f.networks) != 0 || len(f.containers) != 0 {
				t.Fatalf("publication failure cleanup: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestHTTPProbeFailuresAreBoundedAndCleaned(t *testing.T) {
	for _, failure := range []error{errHostNotReady, errors.New("SECRET RESPONSE BODY")} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			f := httpFixture(t)
			f.d.hostProbe = func(context.Context, string) error { return failure }
			r, err := run(context.Background(), nil, f.d)
			if err == nil || strings.Contains(err.Error(), "SECRET") || r.CheckpointPassed || !r.CleanupComplete || len(f.networks) != 0 || len(f.containers) != 0 {
				t.Fatalf("HTTP failure mishandled: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestNetworkCreateUncertaintyAndPriorGate(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprint(visible), func(t *testing.T) {
			f := httpFixture(t)
			f.d.times.cleanup = 30 * time.Millisecond
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				out, err := execute(ctx, args, env)
				if len(args) > 5 && args[4] == "network" && args[5] == "create" {
					if !visible {
						f.networks = map[string]*fakeNetwork{}
					}
					return "", context.DeadlineExceeded
				}
				return out, err
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || r.CleanupComplete != visible || r.Network == nil || r.Network.CreateUncertain == visible {
				t.Fatalf("uncertain create mishandled: %+v %v", r, err)
			}
			assertReport(t, r)
			if !visible {
				f.d.execute = execute
				f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
				calls := len(f.calls)
				next, err := run(context.Background(), nil, f.d)
				if err == nil || !reflect.DeepEqual(next, Result{}) || len(f.calls) != calls || !strings.Contains(err.Error(), "network intent") {
					t.Fatalf("unresolved network bypassed prior gate: %+v %v", next, err)
				}
			}
		})
	}
}

func TestHTTPReadsAccepted2BHistoryWithoutReusingPier(t *testing.T) {
	f := httpFixture(t)
	f.d.profile = ProfileLifecycle
	old, err := run(context.Background(), nil, f.d)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(old.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal(data, &legacy); err != nil {
		t.Fatal(err)
	}
	delete(legacy, "profile") // the user's accepted 2B report has no profile field
	delete(legacy, "network")
	data, err = json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old.ReportPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	// Retained pier contents must not be traversed or adopted by the new run.
	if err := os.Symlink("/not-a-real-target", filepath.Join(old.Directory, "data/fake-zod/untouched")); err != nil {
		t.Fatal(err)
	}
	f.d.profile = ProfileHTTP
	f.d.randomID = func() (string, error) { return strings.Repeat("b", 32), nil }
	next, err := run(context.Background(), nil, f.d)
	if err != nil || !next.CheckpointPassed || next.Directory == old.Directory {
		t.Fatalf("new HTTP run failed over accepted 2B history: %+v %v", next, err)
	}
	after, err := os.ReadFile(old.ReportPath)
	if err != nil || string(after) != string(data) {
		t.Fatal("prior report changed")
	}
	if _, err := os.Lstat(filepath.Join(old.Directory, "data/fake-zod/untouched")); err != nil {
		t.Fatal("prior pier changed")
	}
	// The baseline workflow also accepts a fully resolved HTTP report.
	f.d.profile = ProfileLifecycle
	f.d.randomID = func() (string, error) { return strings.Repeat("c", 32), nil }
	last, err := run(context.Background(), nil, f.d)
	if err != nil || !last.CheckpointPassed || last.Network != nil {
		t.Fatalf("baseline after HTTP: %+v %v", last, err)
	}
}

func TestHTTPCancellationStopsContainersBeforeNetwork(t *testing.T) {
	f := httpFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.d.hostProbe = func(context.Context, string) error { cancel(); return context.Canceled }
	r, err := run(ctx, nil, f.d)
	if err == nil || r.CheckpointPassed || !r.CleanupComplete || len(f.networks) != 0 || len(f.containers) != 0 || f.events[len(f.events)-1] != "network-rm" {
		t.Fatalf("canceled HTTP cleanup: %+v %v, events %v", r, err, f.events)
	}
	assertReport(t, r)
}

func TestHTTPBindingChangeDuringRequestFails(t *testing.T) {
	f := httpFixture(t)
	f.d.hostProbe = func(context.Context, string) error {
		for _, c := range f.containers {
			if isHTTPKind(c.kind) {
				mapAt(c.doc, "NetworkSettings")["Ports"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "38103"}}}
			}
		}
		return nil
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || !strings.Contains(err.Error(), "binding changed") || r.CheckpointPassed || !r.CleanupComplete || r.Containers[1].HostHTTPVerified {
		t.Fatalf("binding race was accepted: %+v %v", r, err)
	}
}

func TestHTTPContainerCleanupFailureRetainsNetwork(t *testing.T) {
	f := httpFixture(t)
	f.d.times.cleanup = 20 * time.Millisecond
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) > 5 && args[4] == "container" && args[5] == "stop" {
			for _, c := range f.containers {
				if isHTTPKind(c.kind) {
					return "", errors.New("stop unavailable")
				}
			}
		}
		return execute(ctx, args, env)
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CleanupComplete || r.Network == nil || r.Network.Removed || len(f.networks) != 1 || len(f.containers) != 1 {
		t.Fatalf("network removed around unresolved container: %+v %v", r, err)
	}
	for _, event := range f.events {
		if event == "network-rm" {
			t.Fatal("network removal attempted")
		}
	}
	assertReport(t, r)
}

func TestNetworkRemoveUncertaintyIsNotInferredFromAbsence(t *testing.T) {
	f := httpFixture(t)
	f.d.times.cleanup = 20 * time.Millisecond
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		out, err := execute(ctx, args, env)
		if len(args) > 5 && args[4] == "network" && args[5] == "rm" {
			return "", context.DeadlineExceeded
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CleanupComplete || r.CheckpointPassed || r.Network.Removed || len(f.networks) != 0 {
		t.Fatalf("lost removal acknowledgement became success: %+v %v", r, err)
	}
	assertReport(t, r)
}

func TestNetworkCleanupWaitsForLateCreate(t *testing.T) {
	f := httpFixture(t)
	execute := f.d.execute
	var pending *fakeNetwork
	inspections := 0
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		out, err := execute(ctx, args, env)
		if len(args) > 5 && args[4] == "network" {
			if args[5] == "create" {
				for name, n := range f.networks {
					pending = n
					delete(f.networks, name)
				}
				return "", context.DeadlineExceeded
			}
			if args[5] == "inspect" && pending != nil {
				inspections++
				f.networks[pending.name] = pending // appears after the first failed inspection
				pending = nil
			}
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CheckpointPassed || !r.CleanupComplete || inspections != 1 || r.Network == nil || !r.Network.Removed || r.Network.CreateUncertain || len(f.networks) != 0 {
		t.Fatalf("late network was abandoned or falsely passed: %+v %v", r, err)
	}
	assertReport(t, r)
}

func TestLoopbackBindingShapes(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{}`, `[{"HostIp":"127.0.0.1","HostPort":"38101"},{"HostIp":"::","HostPort":"38101"}]`,
		`[{"HostIp":"0.0.0.0","HostPort":"38101"}]`, `[{"HostIp":"127.0.0.1","HostPort":"+38101"}]`,
		`[{"HostIp":"127.0.0.1","HostPort":"038101"}]`, `[{"HostIp":"127.0.0.1","HostPort":"65536"}]`,
		`[{"HostIp":"127.0.0.1","HostPort":"80"}]`,
	} {
		if _, err := decodeLoopbackBinding(json.RawMessage(raw), false); err == nil {
			t.Fatalf("unsafe mapping: %s", raw)
		}
	}
	for _, port := range []string{"", "0"} {
		raw := json.RawMessage(fmt.Sprintf(`[{"HostIp":"127.0.0.1","HostPort":%q}]`, port))
		if _, err := decodeLoopbackBinding(raw, true); err != nil {
			t.Fatal(err)
		}
		if _, err := decodeLoopbackBinding(raw, false); err == nil {
			t.Fatal("unassigned port was treated as a usable endpoint")
		}
	}
}
