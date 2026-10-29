//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// All OS identity, socket, executable, capacity, engine and download observations
// are injected. Real filesystem operations are confined to t.TempDir(). No test
// calls Run/defaults' executor, NSS, Docker, an HTTP client or a privilege tool.
type fakeEngine struct {
	t                      *testing.T
	d                      dependencies
	i                      identity
	containers             map[string]*fakeContainer
	calls                  [][]string
	events                 []string
	leftover               bool
	editInfo               func(map[string]any)
	engine                 hostcompat.Engine
	hostReport             hostcompat.Report
	editImage              func(map[string]any)
	editInspect            func(string, map[string]any)
	cgroups, version, http string
	resources              string
	stopExit               int
	oom, earlyExit         bool
	downloads              int
	imageID                string
	networks               map[string]*fakeNetwork
	editNetwork            func(map[string]any)
}
type fakeContainer struct {
	id, name, kind string
	doc            map[string]any
}

func (f *fakeEngine) labelName() string {
	if f.d.managed {
		if f.d.profile == ProfileComet {
			return "org.urbit-runner.managed-comet"
		}
		return "org.urbit-runner.managed-fake"
	}
	return label
}
func (f *fakeEngine) runLabelName() string { return f.labelName() + ".run" }

func fixture(t *testing.T) *fakeEngine {
	t.Helper()
	root := t.TempDir()
	f := &fakeEngine{t: t, containers: map[string]*fakeContainer{}, networks: map[string]*fakeNetwork{}, cgroups: "4294967296\n0\n256\n200000 100000\n", version: "urbit " + candidate.VereVersion + "\n", http: "302"}
	f.imageID = candidate.ManifestDigest
	f.d = defaults()
	f.d.portAvailable = func(string, uint16) bool { return true }
	f.d.home, f.d.runtimeDir = filepath.Join(root, "home"), filepath.Join(root, "runtime")
	for _, p := range []string{f.d.home, f.d.runtimeDir} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	f.i = identity{uid: 12345, gid: 12345, accountUID: 12345, accountGID: 12345, groups: []int{12345}, account: rootless.AccountName, group: rootless.AccountName, home: f.d.home, platform: "linux/amd64"}
	f.d.identity = func() (identity, error) { return f.i, nil }
	f.d.stat = func(path string) (node, error) {
		if path == "/usr/bin/docker" {
			return node{mode: 0755, links: 1}, nil
		}
		if path == filepath.Join(f.d.runtimeDir, rootless.SocketName) {
			return node{mode: os.ModeSocket | os.ModeSticky | 0660, uid: 12345, gid: 12345, links: 1}, nil
		}
		for _, base := range []string{f.d.home, f.d.runtimeDir, "/usr/bin/docker"} {
			for p := filepath.Dir(base); ; p = filepath.Dir(p) {
				if path == p {
					return node{mode: os.ModeDir | 0755, links: 1}, nil
				}
				if p == "/" {
					break
				}
			}
		}
		n, err := statNode(path)
		n.uid, n.gid = 12345, 12345
		return n, err
	}
	f.d.admission = func(string) (uint64, uint64, error) { return 5 << 30, 8 << 30, nil }
	f.d.randomID = func() (string, error) { return strings.Repeat("a", 32), nil }
	// Successful paths include authenticated inspections and durable report writes.
	// Give these room under the race detector; deadline-specific tests override
	// their own budgets instead of making unrelated setup depend on 40–50 ms.
	f.d.times = timings{
		command: time.Second, pull: time.Second, download: time.Second,
		ready: time.Second, poll: time.Millisecond, stop: time.Second,
		cleanup: 3 * time.Second, pack: time.Second, cleanupCommand: time.Second,
	}
	pill := []byte("tiny disposable test pill\n")
	hash := sha256.Sum256(pill)
	f.d.pillSHA, f.d.pillBytes = hex.EncodeToString(hash[:]), int64(len(pill))
	f.d.download = func(ctx context.Context, path string) (candidate.Artifact, error) {
		f.downloads++
		f.events = append(f.events, "download")
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded download")
		}
		err := os.WriteFile(path, pill, 0600)
		return candidate.Artifact{Path: path, SHA256: f.d.pillSHA, Bytes: f.d.pillBytes, URL: candidate.PillURL}, err
	}
	f.d.execute = f.execute
	f.engine = hostcompat.Engine{Version: "29.7.2", APIVersion: "1.54", MinAPIVersion: "1.44", Rootless: true, CgroupDriver: "systemd", CgroupVersion: "2", DataRoot: filepath.Join(f.d.home, ".local/share/urbit-runner/docker")}
	// Authentication is explicit fixture evidence, not a version inference.
	// This distro/stack remains unqualified even though its bytes are verified.
	f.hostReport = hostcompat.Report{Platform: "linux/amd64", InstalledEngineVersion: "29.7.2", Distribution: hostcompat.Distribution{ID: "fixture", Profile: "unqualified"}}
	for _, role := range []string{"engine", "cli", "containerd", "oci-runtime", "rootlesskit", "network", "uidmap", "gidmap"} {
		f.hostReport.Packages = append(f.hostReport.Packages, hostcompat.Package{Role: role, Name: "fixture-" + role, Version: "fixture", Provenance: "verified", Executables: []string{"/usr/bin/fixture-" + role}})
	}
	f.d.InspectHost = func() hostcompat.Report { return f.hostReport }
	f.d.ProbeEngine = func(ctx context.Context, socket, dataRoot string) (hostcompat.Engine, error) {
		if _, ok := ctx.Deadline(); !ok || socket != filepath.Join(f.d.runtimeDir, rootless.SocketName) || dataRoot != filepath.Join(f.d.home, ".local/share/urbit-runner/docker") {
			t.Fatal("unbounded or misrouted engine probe")
		}
		f.events = append(f.events, "engine-api")
		e := f.engine
		// Retain the existing negative /info fixtures, now explicitly converted
		// at the native probe seam alongside separate /version API evidence.
		info := map[string]any{"ServerVersion": e.Version, "SecurityOptions": []string{"name=rootless"}, "CgroupDriver": e.CgroupDriver, "CgroupVersion": e.CgroupVersion, "DockerRootDir": e.DataRoot}
		if f.editInfo != nil {
			f.editInfo(info)
		}
		e.Version = info["ServerVersion"].(string)
		e.CgroupDriver, e.CgroupVersion, e.DataRoot = info["CgroupDriver"].(string), info["CgroupVersion"].(string), info["DockerRootDir"].(string)
		rootless := false
		for _, option := range info["SecurityOptions"].([]string) {
			rootless = rootless || option == "name=rootless"
		}
		e.Rootless = e.Rootless && rootless
		return e, nil
	}
	f.d.stats = func(ctx context.Context, socket, id string) (string, error) {
		if _, ok := ctx.Deadline(); !ok || socket != filepath.Join(f.d.runtimeDir, rootless.SocketName) {
			t.Fatal("unbounded or misrouted stats read")
		}
		owned := false
		for _, c := range f.containers {
			if c.id == id && f.d.managed && isNetworkedKind(c.kind) && mapAt(c.doc, "State")["Running"] == true {
				owned = true
			}
		}
		if !owned {
			t.Fatal("stats read targeted an unknown or stopped container")
		}
		f.events = append(f.events, "resources")
		if f.resources != "" {
			return f.resources, nil
		}
		return encode(t, map[string]any{"id": id, "read": time.Now().UTC(), "memory_stats": map[string]any{"usage": 64 << 20, "limit": 4 << 30}, "pids_stats": map[string]any{"current": 12}}), nil
	}
	return f
}

func encode(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func copyDoc(t *testing.T, v map[string]any) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(encode(t, v)), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func mapAt(v map[string]any, name string) map[string]any { return v[name].(map[string]any) }

func (f *fakeEngine) execute(ctx context.Context, full, env []string) (string, error) {
	t := f.t
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("unbounded Docker invocation")
	}
	if len(full) < 5 || !reflect.DeepEqual(full[:3], []string{"--host", "unix://" + filepath.Join(f.d.runtimeDir, rootless.SocketName), "--config"}) {
		t.Fatalf("unsafe routing: %q", full)
	}
	if filepath.Base(full[3]) != "docker-config" {
		t.Fatalf("unsafe config: %q", full)
	}
	if !reflect.DeepEqual(env, []string{"HOME=" + f.d.home, "PATH=/usr/bin:/bin", "LANG=C", "DOCKER_API_VERSION=" + hostcompat.APIVersion}) {
		t.Fatalf("inherited environment: %q", env)
	}
	args := append([]string(nil), full[4:]...)
	f.calls = append(f.calls, args)
	for _, arg := range args {
		if arg == "logs" || arg == "--force" || arg == "-f" || arg == "-v" || (arg == "--publish" && !networkProfile(f.d.profile)) || arg == "--privileged" || arg == "--net=host" || arg == "--keys" || arg == "-u" || strings.Contains(arg, "conn.sock") || strings.HasPrefix(arg, "tcp://") {
			t.Fatalf("forbidden operation: %q", args)
		}
	}
	switch args[0] {
	case "image":
		if args[1] == "pull" {
			f.events = append(f.events, "pull")
			if !reflect.DeepEqual(args, []string{"image", "pull", "--platform", "linux/amd64", candidate.Image}) {
				t.Fatalf("unpinned pull %q", args)
			}
			return "", nil
		}
		if !reflect.DeepEqual(args, []string{"image", "inspect", candidate.Image}) {
			t.Fatalf("unexpected image command: %q", args)
		}
		f.events = append(f.events, "image-inspect")
		image := map[string]any{"Id": f.imageID, "Os": "linux", "Architecture": "amd64", "RepoDigests": []string{strings.TrimPrefix(candidate.Image, "docker.io/")}}
		if f.editImage != nil {
			f.editImage(image)
		}
		return encode(t, []any{image}), nil
	case "network":
		return f.networkCommand(args, full[3])
	case "container":
	default:
		t.Fatalf("unexpected command %q", args)
	}
	verb := args[1]
	if verb == "ls" {
		if !strings.Contains(strings.Join(args, " "), "--filter label="+f.labelName()+"=true") || args[len(args)-1] != "{{.Names}}" {
			t.Fatalf("unscoped inventory: %q", args)
		}
		if len(args) == 7 && f.leftover {
			return "urbit-qualification-existing-boot\n", nil
		}
		var names []string
		for name, c := range f.containers {
			match := true
			for _, arg := range args {
				if strings.HasPrefix(arg, "name=^/") && arg != "name=^/"+name+"$" {
					match = false
				}
				if strings.HasPrefix(arg, "label="+f.runLabelName()+"=") && strings.TrimPrefix(arg, "label="+f.runLabelName()+"=") != mapAt(mapAt(c.doc, "Config"), "Labels")[f.runLabelName()] {
					match = false
				}
			}
			if match {
				names = append(names, name)
			}
		}
		return strings.Join(names, "\n"), nil
	}
	if verb == "create" {
		return f.create(args, full[3])
	}
	if verb == "inspect" {
		c := f.containers[args[2]]
		if c == nil {
			return "", errors.New("missing")
		}
		doc := copyDoc(t, c.doc)
		if f.editInspect != nil {
			f.editInspect(c.kind, doc)
		}
		return encode(t, []any{doc}), nil
	}
	id := args[2]
	if verb == "stop" {
		if !reflect.DeepEqual(args[2:4], []string{"--timeout", "300"}) {
			t.Fatal("stop timeout mismatch")
		}
		id = args[4]
	}
	var c *fakeContainer
	for _, item := range f.containers {
		if item.id == id {
			c = item
		}
	}
	if c == nil || !containerIDPattern.MatchString(id) {
		t.Fatalf("control without known full ID: %q", args)
	}
	state := mapAt(c.doc, "State")
	switch verb {
	case "start":
		f.events = append(f.events, "start-"+c.kind)
		state["Running"] = true
		if c.kind == "boot" || c.kind == "http-boot" {
			cmd := mapAt(c.doc, "Config")["Cmd"].([]string)
			pier := strings.TrimPrefix(cmd[len(cmd)-1], "/urbit/")
			if filepath.Base(pier) != pier {
				t.Fatal("fixture pier escaped its mount")
			}
			if err := os.MkdirAll(filepath.Join(fakeMountRoot(c), pier, ".urb"), 0700); err != nil {
				t.Fatal(err)
			}
			if f.earlyExit {
				state["Running"], state["ExitCode"] = false, 1
			}
		}
		if c.kind == "comet-boot" {
			cmd := mapAt(c.doc, "Config")["Cmd"].([]string)
			if err := os.MkdirAll(filepath.Join(fakeMountRoot(c), strings.TrimPrefix(cmd[len(cmd)-1], "/urbit/"), ".urb"), 0700); err != nil {
				t.Fatal(err)
			}
		}
		if c.kind == "comet-boot" {
			cmd := mapAt(c.doc, "Config")["Cmd"].([]string)
			for _, arg := range cmd {
				if arg == "-k" {
					path := filepath.Join(fakeMountRoot(c), strings.TrimPrefix(cmd[len(cmd)-1], "/urbit/"), ".http.ports")
					if err := os.WriteFile(path, []byte("8080 insecure public\n12321 insecure loopback\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if isNetworkedKind(c.kind) {
			f.publishHTTP(c)
		}
	case "exec":
		if state["Running"] != true {
			t.Fatal("exec in stopped container")
		}
		switch args[3] {
		case "/bin/cat":
			if !reflect.DeepEqual(args[3:], []string{"/bin/cat", "/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory.swap.max", "/sys/fs/cgroup/pids.max", "/sys/fs/cgroup/cpu.max"}) {
				t.Fatal("cgroup evidence command changed")
			}
			f.events = append(f.events, "cgroups")
			return f.cgroups, nil
		case "/bin/urbit":
			if !reflect.DeepEqual(args[3:], []string{"/bin/urbit", "-R"}) {
				t.Fatal("unexpected Vere execution")
			}
			f.events = append(f.events, "version")
			return f.version, nil
		case "/usr/bin/curl":
			joined := strings.Join(args, " ")
			if strings.Contains(joined, "--data-binary") {
				cmd := mapAt(c.doc, "Config")["Cmd"].([]string)
				ship := ""
				for i, arg := range cmd {
					if arg == "-w" && i+1 < len(cmd) {
						ship = cmd[i+1]
					}
				}
				if ship == "" {
					t.Fatal("private shutdown read on non-keyed fixture")
				}
				if strings.Contains(joined, `"dojo":"our"`) {
					b, _ := json.Marshal(ship)
					return string(b), nil
				}
				if strings.Contains(joined, `"dojo":"+hood/exit"`) && strings.Contains(joined, `"app":"hood"`) {
					f.events = append(f.events, "stop-"+c.kind)
					state["Running"], state["ExitCode"], state["OOMKilled"] = false, f.stopExit, f.oom
					return "null", nil
				}
				t.Fatal("unrecognized private shutdown payload")
			}
			if !reflect.DeepEqual(args[3:], []string{"/usr/bin/curl", "--noproxy", "*", "--silent", "--show-error", "--max-time", "5", "--output", "/dev/null", "--write-out", "%{http_code}", "http://127.0.0.1:8080/~/host"}) {
				t.Fatal("unsafe HTTP probe")
			}
			f.events = append(f.events, "http-"+c.kind)
			return f.http, nil
		default:
			t.Fatalf("unexpected exec: %q", args)
		}
	case "kill", "stop":
		if verb == "kill" && !reflect.DeepEqual(args[3:], []string{"--signal", "SIGTERM"}) {
			t.Fatal("unsafe signal", args)
		}
		f.events = append(f.events, "stop-"+c.kind)
		if c.kind == "probe" && state["Running"] == true && mapAt(c.doc, "HostConfig")["Init"] != true {
			// A PID-1 sleep ignores default-action SIGTERM. Do not model the
			// legacy probe as terminating promptly when it actually waits 120s.
			return "", context.DeadlineExceeded
		}
		state["Running"], state["ExitCode"] = false, 0
		if c.kind == "probe" {
			state["ExitCode"] = 143
		} else {
			state["ExitCode"], state["OOMKilled"] = f.stopExit, f.oom
		}
	case "wait":
		if c.kind != "pack" && c.kind != "meld" && c.kind != "roll" && !isRollKind(c.kind) && !isCometRoll(c.kind) && !isManagedCometUtility(c.kind) && c.kind != "epoch-recovery" && !isPrefixRecovery(c.kind) && !isCometPrefixRecovery(c.kind) {
			t.Fatal("unexpected wait")
		}
		f.events = append(f.events, "wait-"+c.kind)
		state["Running"], state["ExitCode"] = false, 0
		return "0\n", nil
	case "rm":
		if state["Running"] == true {
			t.Fatal("removing live container")
		}
		f.events = append(f.events, "rm-"+c.kind)
		for _, n := range f.networks {
			delete(mapAt(n.doc, "Containers"), c.id)
		}
		delete(f.containers, c.name)
	default:
		t.Fatalf("unexpected control: %q", args)
	}
	return id + "\n", nil
}

func fakeMountRoot(c *fakeContainer) string {
	for _, raw := range c.doc["Mounts"].([]any) {
		m := raw.(map[string]any)
		if m["Destination"] == "/urbit" {
			return m["Source"].(string)
		}
	}
	panic("fixture has no /urbit mount")
}

func (f *fakeEngine) create(args []string, config string) (string, error) {
	t := f.t
	opts := map[string][]string{}
	i := 2
	for i < len(args) && strings.HasPrefix(args[i], "--") {
		if args[i] == "--init" {
			opts["--init"] = append(opts["--init"], "true")
			i++
			continue
		}
		if i+1 >= len(args) {
			t.Fatal("missing create option value")
		}
		opts[args[i]] = append(opts[args[i]], args[i+1])
		i += 2
	}
	value := func(key string) string {
		if len(opts[key]) != 1 {
			t.Fatalf("create missing or duplicate %s", key)
		}
		return opts[key][0]
	}
	name := value("--name")
	if f.containers[name] != nil {
		return "", errors.New("duplicate container")
	}
	var report Result
	data, err := os.ReadFile(filepath.Join(filepath.Dir(config), "report.json"))
	if err != nil || json.Unmarshal(data, &report) != nil {
		t.Fatal("intent report missing")
	}
	if len(report.Containers) == 0 || report.Containers[len(report.Containers)-1].Name != name {
		t.Fatal("create was not preceded by persisted intent")
	}
	intent := report.Containers[len(report.Containers)-1]
	if !intent.CreateDispatched || !intent.CreateUncertain || intent.Removed {
		t.Fatal("dispatch uncertainty was not persisted before create")
	}
	if report.ImageID != f.imageID || args[i] != f.imageID || report.RepoDigest == "" {
		t.Fatal("create did not use the authenticated backend image ID")
	}
	kind := report.Containers[len(report.Containers)-1].Kind
	f.events = append(f.events, "create-"+kind)
	labels, logOpts := map[string]any{}, map[string]any{}
	for _, p := range opts["--label"] {
		k, v, _ := strings.Cut(p, "=")
		labels[k] = v
	}
	for _, p := range opts["--log-opt"] {
		k, v, _ := strings.Cut(p, "=")
		logOpts[k] = v
	}
	var mounts []any
	for _, spec := range opts["--mount"] {
		m := map[string]any{"RW": true}
		for _, part := range strings.Split(spec, ",") {
			key, val, _ := strings.Cut(part, "=")
			switch key {
			case "type":
				m["Type"] = val
			case "src":
				m["Source"] = val
			case "dst":
				m["Destination"] = val
			case "bind-propagation":
				m["Propagation"] = val
			case "readonly":
				m["RW"] = false
			default:
				t.Fatalf("unknown mount %s", part)
			}
		}
		mounts = append(mounts, m)
	}
	// Match the official image's VOLUME /urbit behavior. A child bind alone
	// leaves an extra anonymous volume, which strict containment must reject.
	rootBound := false
	for _, m := range mounts {
		if m.(map[string]any)["Destination"] == "/urbit" {
			rootBound = true
		}
	}
	if !rootBound {
		mounts = append(mounts, map[string]any{"Type": "volume", "Source": "/fixture/anonymous-volume", "Destination": "/urbit", "RW": true, "Propagation": ""})
	}
	number := func(key string) int64 {
		n, err := strconv.ParseInt(value(key), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if value("--platform") != "linux/amd64" || value("--pull") != "never" {
		t.Fatal("unsafe create image policy")
	}
	id := fmt.Sprintf("%064x", len(report.Containers))
	doc := map[string]any{
		"Id": id, "Name": "/" + name, "Image": args[i],
		"Config":     map[string]any{"Image": args[i], "User": value("--user"), "StopSignal": value("--stop-signal"), "StopTimeout": number("--stop-timeout"), "Entrypoint": []string{value("--entrypoint")}, "Cmd": args[i+1:], "Labels": labels},
		"HostConfig": map[string]any{"NetworkMode": value("--network"), "IpcMode": value("--ipc"), "CgroupnsMode": value("--cgroupns"), "CapDrop": opts["--cap-drop"], "SecurityOpt": opts["--security-opt"], "Memory": number("--memory"), "MemorySwap": number("--memory-swap"), "PidsLimit": number("--pids-limit"), "NanoCpus": number("--cpus") * 1_000_000_000, "RestartPolicy": map[string]any{"Name": value("--restart")}, "LogConfig": map[string]any{"Type": value("--log-driver"), "Config": logOpts}},
		"Mounts":     mounts, "NetworkSettings": map[string]any{"Networks": map[string]any{"none": map[string]any{}}},
		"State": map[string]any{"Running": false, "ExitCode": 0, "OOMKilled": false},
	}
	if len(opts["--init"]) != 0 {
		if len(opts["--init"]) != 1 || opts["--init"][0] != "true" || kind != "probe" {
			t.Fatal("init is only allowed on the probe")
		}
		mapAt(doc, "HostConfig")["Init"] = true
	}
	if isNetworkedKind(kind) {
		port := ""
		if f.d.hostPort != 0 {
			port = fmt.Sprint(f.d.hostPort)
		}
		published := opts["--publish"]
		if !networkProfile(f.d.profile) || len(published) < 1 || len(published) > 2 || published[0] != "127.0.0.1:"+port+":8080/tcp" {
			t.Fatal("unsafe HTTP publication")
		}
		n := f.networkByID(value("--network"))
		if n == nil {
			t.Fatal("HTTP container did not use the owned network ID")
		}
		mapAt(doc, "HostConfig")["PortBindings"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": port}}}
		if len(published) == 2 {
			parts := strings.Split(published[1], ":")
			if len(parts) != 3 || parts[0] != "0.0.0.0" || parts[2] != parts[1]+"/udp" {
				t.Fatal("invalid Ames publication", published)
			}
			mapAt(mapAt(doc, "HostConfig"), "PortBindings")[parts[2]] = []any{map[string]any{"HostIp": "0.0.0.0", "HostPort": parts[1]}}
		}
		doc["NetworkSettings"] = map[string]any{
			"Networks": map[string]any{n.name: map[string]any{"NetworkID": ""}},
			"Ports":    map[string]any{"80/tcp": nil, "34343/udp": nil, "8080/tcp": nil},
		}
	}
	f.containers[name] = &fakeContainer{id, name, kind, doc}
	return id + "\n", nil
}

func assertReport(t *testing.T, r Result) {
	t.Helper()
	if r.RuntimeQualified {
		t.Fatal("development result qualified a runtime")
	}
	if r.Directory == "" {
		t.Fatal("missing result directory")
	}
	info, err := os.Stat(r.ReportPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("unsafe/missing report: %v", err)
	}
	info, err = os.Stat(r.Directory)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("unsafe/missing run directory: %v", err)
	}
	data, err := os.ReadFile(r.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored Result
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, stored) {
		t.Fatalf("report not updated\nreturned: %+v\nstored: %+v", r, stored)
	}
}

func TestSuccessCheckpointOrderAndIsolation(t *testing.T) {
	f := fixture(t)
	var phases []string
	r, err := run(context.Background(), func(s string) { phases = append(phases, s) }, f.d)
	if err != nil {
		t.Fatal(err)
	}
	if !r.CheckpointPassed || !r.CleanupComplete || !r.LimitsObserved || !r.PierMetadataSeen || len(r.Containers) != 5 {
		t.Fatalf("incomplete result: %+v", r)
	}
	assertReport(t, r)
	if !reflect.DeepEqual(phases, []string{"engine", "image", "pill", "probe", "boot", "restart", "pack", "post-pack", "cleanup"}) {
		t.Fatalf("progress leaked or missing: %q", phases)
	}
	want := []string{"engine-api", "pull", "image-inspect", "download", "engine-api", "create-probe", "start-probe", "cgroups", "version", "stop-probe", "rm-probe", "engine-api", "create-boot", "start-boot", "http-boot", "stop-boot", "rm-boot", "engine-api", "create-restart", "start-restart", "http-restart", "stop-restart", "rm-restart", "engine-api", "create-pack", "start-pack", "wait-pack", "rm-pack", "engine-api", "create-post-pack", "start-post-pack", "http-post-pack", "stop-post-pack", "rm-post-pack"}
	if !reflect.DeepEqual(f.events, want) {
		t.Fatalf("wrong safety ordering\n%q", f.events)
	}
	for _, c := range r.Containers {
		if !c.Removed || !containerIDPattern.MatchString(c.ID) {
			t.Fatalf("unresolved container: %+v", c)
		}
		if c.Kind != "probe" && c.Kind != "pack" && (!c.AliveObserved || !c.PrivateHTTPReachable) {
			t.Fatal("missing private liveness observations")
		}
	}
	for _, call := range f.calls {
		if len(call) < 2 || call[1] != "create" {
			continue
		}
		image := -1
		for i, a := range call {
			if a == r.ImageID {
				image = i
				break
			}
		}
		if image < 0 {
			t.Fatal("create did not use verified image ID")
		}
		args := call[image+1:]
		switch {
		case reflect.DeepEqual(args, []string{"120"}):
		case len(args) > 0 && args[0] == "pack":
			if !reflect.DeepEqual(args, []string{"pack", "--loom", "31", "/urbit/fake-zod"}) {
				t.Fatalf("pack mixed boot flags: %q", args)
			}
		default:
			want := []string{"-t", "--snap-time", "60", "-L", "--no-conn", "--no-dock", "--loom", "31", "-b", "127.0.0.1", "--http-port", "8080"}
			if strings.HasSuffix(call[3], "-boot") {
				want = append(want, "-F", "zod", "-B", "/artifacts/boot.pill", "-c")
			}
			want = append(want, "/urbit/fake-zod")
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("unsafe Vere arguments: %q", args)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(r.Directory, "data/fake-zod/.urb")); err != nil {
		t.Fatal("pier not retained")
	}
}

func TestPrevalidationIsReadOnly(t *testing.T) {
	cases := map[string]func(*fakeEngine){
		"root":                func(f *fakeEngine) { f.i.uid = 0 },
		"wrong uid":           func(f *fakeEngine) { f.i.uid++ },
		"wrong gid":           func(f *fakeEngine) { f.i.gid++ },
		"wrong home":          func(f *fakeEngine) { f.i.home = "/root" },
		"wrong account":       func(f *fakeEngine) { f.i.account = "other" },
		"shared group":        func(f *fakeEngine) { f.i.group = "users" },
		"supplementary group": func(f *fakeEngine) { f.i.groups = append(f.i.groups, 42) },
		"wrong platform":      func(f *fakeEngine) { f.i.platform = "linux/arm64" },
		"low memory ignores swap": func(f *fakeEngine) {
			f.d.admission = func(string) (uint64, uint64, error) { return 5<<30 - 1, 100 << 30, nil }
		},
		"low disk": func(f *fakeEngine) {
			f.d.admission = func(string) (uint64, uint64, error) { return 100 << 30, 8<<30 - 1, nil }
		},
	}
	for _, target := range []string{"home", "runtime", "socket", "docker"} {
		for _, fault := range []string{"symlink", "owner", "writable"} {
			cases[target+" "+fault] = func(f *fakeEngine) {
				path := map[string]string{"home": f.d.home, "runtime": f.d.runtimeDir, "socket": filepath.Join(f.d.runtimeDir, rootless.SocketName), "docker": "/usr/bin/docker"}[target]
				stat := f.d.stat
				f.d.stat = func(p string) (node, error) {
					n, e := stat(p)
					if p == path {
						switch fault {
						case "symlink":
							n.mode = os.ModeSymlink | 0777
						case "owner":
							n.uid++
						case "writable":
							n.mode |= 0002
						}
					}
					return n, e
				}
			}
		}
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := fixture(t)
			setup(f)
			r, err := run(context.Background(), nil, f.d)
			if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 || f.downloads != 0 {
				t.Fatalf("unsafe prevalidation: %+v %v", r, err)
			}
			entries, e := os.ReadDir(f.d.home)
			if e != nil || len(entries) != 0 {
				t.Fatal("prevalidation mutated home")
			}
		})
	}
}

func TestEngineRejectedBeforePullOrDownload(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"rootful":   func(d map[string]any) { d["SecurityOptions"] = []string{"name=notrootless"} },
		"substring": func(d map[string]any) { d["SecurityOptions"] = []string{"name=rootless,untrusted=true"} },
		"version":   func(d map[string]any) { d["ServerVersion"] = "30.0.0" },
		"driver":    func(d map[string]any) { d["CgroupDriver"] = "cgroupfs" },
		"v1":        func(d map[string]any) { d["CgroupVersion"] = "1" },
		"rootdir":   func(d map[string]any) { d["DockerRootDir"] = "/var/lib/docker" },
	} {
		t.Run(name, func(t *testing.T) {
			f := fixture(t)
			f.editInfo = edit
			r, err := run(context.Background(), nil, f.d)
			if err == nil || !reflect.DeepEqual(f.events, []string{"engine-api"}) || f.downloads != 0 {
				t.Fatalf("engine accepted: %v %q", err, f.events)
			}
			assertReport(t, r)
		})
	}
}

func TestImageAndLiveLimitsGateVere(t *testing.T) {
	cases := map[string]func(*fakeEngine){
		"image ID":         func(f *fakeEngine) { f.editImage = func(d map[string]any) { d["Id"] = "sha256:wrong" } },
		"image platform":   func(f *fakeEngine) { f.editImage = func(d map[string]any) { d["Architecture"] = "arm64" } },
		"image digest":     func(f *fakeEngine) { f.editImage = func(d map[string]any) { d["RepoDigests"] = []string{"untrusted"} } },
		"unlimited memory": func(f *fakeEngine) { f.cgroups = "max\n0\n256\n200000 100000\n" },
		"swap":             func(f *fakeEngine) { f.cgroups = "4294967296\n4294967296\n256\n200000 100000\n" },
		"pids":             func(f *fakeEngine) { f.cgroups = "4294967296\n0\nmax\n200000 100000\n" },
		"CPU unlimited":    func(f *fakeEngine) { f.cgroups = "4294967296\n0\n256\nmax 100000\n" },
		"CPU wrong ratio":  func(f *fakeEngine) { f.cgroups = "4294967296\n0\n256\n199999 100000\n" },
		"CPU zero period":  func(f *fakeEngine) { f.cgroups = "4294967296\n0\n256\n0 0\n" },
		"version":          func(f *fakeEngine) { f.version = "urbit unpinned\n" },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := fixture(t)
			setup(f)
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete {
				t.Fatalf("gate failed: %+v %v", r, err)
			}
			for _, event := range f.events {
				if event == "create-boot" {
					t.Fatal("Vere booted before gate passed")
				}
			}
			assertReport(t, r)
		})
	}
}

func TestPillFailuresDoNotCreateContainers(t *testing.T) {
	for _, fault := range []string{"download", "SHA", "size", "provenance", "symlink"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			download := f.d.download
			f.d.download = func(ctx context.Context, path string) (candidate.Artifact, error) {
				if fault == "download" {
					return candidate.Artifact{}, errors.New("SECRET-CODE")
				}
				a, e := download(ctx, path)
				if e != nil {
					return a, e
				}
				switch fault {
				case "SHA":
					e = os.WriteFile(path, []byte(strings.Repeat("x", int(f.d.pillBytes))), 0600)
				case "size":
					e = os.WriteFile(path, []byte("short"), 0600)
				case "provenance":
					a.URL = "https://untrusted.invalid/"
				case "symlink":
					target := path + ".target"
					if e = os.Rename(path, target); e == nil {
						e = os.Symlink(target, path)
					}
				}
				return a, e
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || len(r.Containers) != 0 || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("pill failure mishandled: %+v %v", r, err)
			}
			assertReport(t, r)
		})
	}
}

func TestInspectContainmentBeforeStart(t *testing.T) {
	for _, fault := range []string{"network", "published", "caps", "privileged", "memory", "swap", "pids", "cpus", "restart", "logs", "mount", "pill writable", "entrypoint", "labels", "image", "name", "ID"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			f.editInspect = func(kind string, d map[string]any) {
				if fault == "pill writable" && kind != "boot" {
					return
				}
				h := mapAt(d, "HostConfig")
				cfg := mapAt(d, "Config")
				switch fault {
				case "network":
					h["NetworkMode"] = "host"
				case "published":
					h["PortBindings"] = map[string]any{"8080/tcp": []any{map[string]any{"HostPort": "8080"}}}
				case "caps":
					h["CapAdd"] = []string{"SYS_ADMIN"}
				case "privileged":
					h["Privileged"] = true
				case "memory":
					h["Memory"] = 0
				case "swap":
					h["MemorySwap"] = -1
				case "pids":
					h["PidsLimit"] = nil
				case "cpus":
					h["NanoCpus"] = 0
				case "restart":
					mapAt(h, "RestartPolicy")["Name"] = "always"
				case "logs":
					mapAt(h, "LogConfig")["Type"] = "json-file"
				case "mount":
					d["Mounts"].([]any)[0].(map[string]any)["Source"] = "/existing/pier"
				case "pill writable":
					d["Mounts"].([]any)[1].(map[string]any)["RW"] = true
				case "entrypoint":
					cfg["Entrypoint"] = []string{"/bin/sh"}
				case "labels":
					mapAt(cfg, "Labels")[runLabel] = "other"
				case "image":
					d["Image"] = "sha256:other"
				case "name":
					d["Name"] = "/other"
				case "ID":
					d["Id"] = strings.Repeat("e", 64)
				}
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil {
				t.Fatal("unsafe inspection accepted")
			}
			for _, event := range f.events {
				if event == "start-boot" || (fault != "pill writable" && event == "start-probe") {
					t.Fatal("unsafe container started")
				}
			}
			unowned := fault == "labels" || fault == "image" || fault == "name" || fault == "ID"
			if r.CleanupComplete == unowned {
				t.Fatalf("ownership cleanup mismatch %+v", r)
			}
			assertReport(t, r)
		})
	}
}

func TestReadinessAndShutdownFailures(t *testing.T) {
	for _, fault := range []string{"early exit", "HTTP timeout", "137", "OOM"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			switch fault {
			case "early exit":
				f.earlyExit = true
			case "HTTP timeout":
				f.http = "503"
			case "137":
				f.stopExit = 137
			case "OOM":
				f.oom = true
			}
			r, err := run(context.Background(), nil, f.d)
			if err == nil || r.CheckpointPassed || !r.CleanupComplete || len(f.containers) != 0 {
				t.Fatalf("failed lifecycle accepted: %+v %v", r, err)
			}
			if _, err := os.Stat(filepath.Join(r.Directory, "data/fake-zod/.urb")); err != nil {
				t.Fatal("failed pier was not retained")
			}
			assertReport(t, r)
		})
	}
}

func TestBusyLockLeftoversAndReusedPaths(t *testing.T) {
	t.Run("busy", func(t *testing.T) {
		f := fixture(t)
		q := &qualifier{d: f.d}
		if _, _, err := q.prevalidate(); err != nil {
			t.Fatal(err)
		}
		unlock, err := q.lockStorage()
		if err != nil {
			t.Fatal(err)
		}
		defer unlock()
		r, err := run(context.Background(), nil, f.d)
		if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 {
			t.Fatal("busy account was used")
		}
	})
	t.Run("leftovers", func(t *testing.T) {
		f := fixture(t)
		f.leftover = true
		r, err := run(context.Background(), nil, f.d)
		if err == nil || f.downloads != 0 || len(r.Containers) != 0 {
			t.Fatal("leftover was adopted")
		}
		assertReport(t, r)
	})
	t.Run("reused", func(t *testing.T) {
		f := fixture(t)
		r, err := run(context.Background(), nil, f.d)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(r.ReportPath)
		calls := len(f.calls)
		r2, err := run(context.Background(), nil, f.d)
		if err == nil || !reflect.DeepEqual(r2, Result{}) || len(f.calls) != calls {
			t.Fatal("reused run directory")
		}
		after, _ := os.ReadFile(r.ReportPath)
		if string(before) != string(after) {
			t.Fatal("prior report overwritten")
		}
	})
	t.Run("unsafe descendant", func(t *testing.T) {
		f := fixture(t)
		base := filepath.Join(f.d.home, ".local/share/urbit-runner/qualification")
		if err := os.MkdirAll(base, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(f.d.home, filepath.Join(base, "existing")); err != nil {
			t.Fatal(err)
		}
		r, err := run(context.Background(), nil, f.d)
		if err == nil || !reflect.DeepEqual(r, Result{}) || len(f.calls) != 0 {
			t.Fatal("unsafe prior storage accepted")
		}
	})
}

func TestCreateUncertaintyAndCancellationCleanup(t *testing.T) {
	for _, fault := range []string{"accepted cancel", "accepted error", "invalid response ID", "not accepted cancel", "foreign label", "stop unavailable", "remove unavailable"} {
		t.Run(fault, func(t *testing.T) {
			f := fixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			execute := f.d.execute
			f.d.execute = func(c context.Context, a, e []string) (string, error) {
				args := a[4:]
				if len(args) > 1 && args[1] == "create" {
					if fault == "not accepted cancel" {
						cancel()
						return "", context.Canceled
					}
					out, err := execute(c, a, e)
					switch fault {
					case "accepted cancel":
						cancel()
						return "", context.Canceled
					case "accepted error":
						return "", errors.New("SECRET-CODE")
					case "invalid response ID":
						return "invalid response\n", err
					case "foreign label":
						for _, item := range f.containers {
							mapAt(mapAt(item.doc, "Config"), "Labels")[runLabel] = "another-run"
						}
						return "", errors.New("uncertain create")
					case "stop unavailable", "remove unavailable":
						for _, item := range f.containers {
							mapAt(item.doc, "State")["Running"] = true
						}
						return "", errors.New("uncertain create")
					}
					return out, err
				}
				if len(args) > 1 && ((fault == "stop unavailable" && args[1] == "stop") || (fault == "remove unavailable" && args[1] == "rm")) {
					return "", errors.New("SECRET-CODE")
				}
				return execute(c, a, e)
			}
			r, err := run(ctx, nil, f.d)
			if fault == "invalid response ID" {
				if err != nil || !r.CheckpointPassed {
					t.Fatalf("owned fallback failed: %v", err)
				}
			} else if err == nil || r.CheckpointPassed {
				t.Fatal("uncertainty reported success")
			}
			incomplete := fault == "foreign label" || fault == "stop unavailable" || fault == "remove unavailable" || fault == "not accepted cancel"
			if r.CleanupComplete == incomplete {
				t.Fatalf("cleanup evidence wrong: %+v %v", r, err)
			}
			if !incomplete && len(f.containers) != 0 {
				t.Fatal("accepted container orphaned")
			}
			if err != nil && strings.Contains(err.Error(), "SECRET") {
				t.Fatal("raw command error leaked")
			}
			assertReport(t, r)
		})
	}
}

func TestInterruptedPackIsNotRetried(t *testing.T) {
	f := fixture(t)
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		if len(a) > 5 && a[5] == "wait" {
			return "", context.DeadlineExceeded
		}
		return execute(ctx, a, e)
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || !r.CleanupComplete || r.CheckpointPassed {
		t.Fatalf("pack interruption wrong: %+v %v", r, err)
	}
	count := 0
	for _, event := range f.events {
		if event == "create-pack" {
			count++
		}
		if event == "create-post-pack" {
			t.Fatal("boot after interrupted pack")
		}
	}
	if count != 1 {
		t.Fatal("maintenance retried")
	}
	assertReport(t, r)
}

func TestBoundaryRecheckedBeforeEachInvocation(t *testing.T) {
	f := fixture(t)
	execute, stat := f.d.execute, f.d.stat
	unsafe := false
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		unsafe = true
		return out, err
	}
	f.d.stat = func(p string) (node, error) {
		n, e := stat(p)
		if unsafe && p == "/usr/bin/docker" {
			n.mode |= 0020
		}
		return n, e
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || len(f.calls) != 1 {
		t.Fatal("changed executable boundary ignored")
	}
	assertReport(t, r)
}

func TestOutputLimitsAndErrorRedaction(t *testing.T) {
	canceled := false
	b := limitedOutput{cancel: func() { canceled = true }}
	if _, err := b.Write(make([]byte, outputLimit)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("x")); err == nil || !canceled || b.buffer.Len() != outputLimit {
		t.Fatal("output was not bounded")
	}
	canceled = false
	b.buffer.Reset()
	// LimitReader hides WriterTo, exercising the same io.Copy fallback that
	// could otherwise discover an accidentally promoted bytes.Buffer.ReadFrom.
	source := io.LimitReader(strings.NewReader(strings.Repeat("x", outputLimit+1)), outputLimit+1)
	if _, err := io.Copy(&b, source); err == nil || !canceled || b.buffer.Len() > outputLimit {
		t.Fatal("streaming output bypassed the cap")
	}
	f := fixture(t)
	f.d.execute = func(context.Context, []string, []string) (string, error) {
		return strings.Repeat("x", outputLimit+1), errors.New("SECRET-OUTPUT")
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatal("error output leaked")
	}
	assertReport(t, r)
}

func TestReportWriteFailureCannotPass(t *testing.T) {
	f := fixture(t)
	execute := f.d.execute
	blocked := false
	f.d.execute = func(ctx context.Context, a, e []string) (string, error) {
		out, err := execute(ctx, a, e)
		if !blocked && len(a) > 5 && a[5] == "rm" {
			blocked = true
			path := filepath.Join(filepath.Dir(a[3]), "report.json")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return out, err
	}
	r, err := run(context.Background(), nil, f.d)
	if err == nil || r.CheckpointPassed || !strings.Contains(err.Error(), "report") {
		t.Fatalf("report persistence ignored: %+v %v", r, err)
	}
}

func TestLockReallyUsesNonblockingFlock(t *testing.T) {
	f := fixture(t)
	q := &qualifier{d: f.d}
	if _, _, err := q.prevalidate(); err != nil {
		t.Fatal(err)
	}
	unlock, err := q.lockStorage()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(q.base, ".lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		t.Fatal("not locked")
	}
	unlock()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal("lock leaked")
	}
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
