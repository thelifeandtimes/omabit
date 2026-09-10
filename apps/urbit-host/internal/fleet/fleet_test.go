package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	r := t.TempDir()
	p := Paths{filepath.Join(r, "config"), filepath.Join(r, "data"), filepath.Join(r, "state"), filepath.Join(r, "run")}
	if e := p.Ensure(); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestGalaxyDirectory(t *testing.T) {
	if len(suffixes) != 768 {
		t.Fatal(len(suffixes))
	}
	seen := map[string]bool{}
	for i := 0; i < 256; i++ {
		name := suffixes[i*3 : i*3+3]
		n, e := Galaxy("~" + name)
		if e != nil || n != i || seen[name] {
			t.Fatalf("%d %s %v", i, name, e)
		}
		seen[name] = true
	}
	if n, _ := Galaxy("nec"); n != 1 {
		t.Fatal(n)
	}
	if _, e := Galaxy("../../etc"); e == nil {
		t.Fatal("accepted path")
	}
}
func TestPathSafety(t *testing.T) {
	p := testPaths(t)
	for _, s := range []string{"", ".", "../secret", "/etc/passwd"} {
		if _, e := CheckPath(p.Data, s); e == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	os.Symlink(t.TempDir(), filepath.Join(p.Data, "link"))
	if _, e := CheckPath(p.Data, "link/pier"); e == nil {
		t.Fatal("followed symlink")
	}
	if _, e := CheckPath(p.Data, "piers/dev/ok/pier"); e != nil {
		t.Fatal(e)
	}
}
func TestAtomicPersistenceAndLock(t *testing.T) {
	p := testPaths(t)
	file := filepath.Join(p.State, "db.json")
	if e := AtomicJSON(file, map[string]int{"schema": 1}); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(file)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	l, e := AcquireLock(filepath.Join(p.State, "lock"))
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	if b, e := AcquireLock(filepath.Join(p.State, "lock")); e == nil {
		b.Close()
		t.Fatal("double lock")
	}
}
func TestVersionManifest(t *testing.T) {
	digest := strings.Repeat("a", 64)
	b := []byte(`{"groundseg":{"latest":{"vere":{"repo":"example/urbit","tag":"v1","amd64_sha256":"` + digest + `","arm64_sha256":"` + strings.Repeat("b", 64) + `"}}}}`)
	for _, arch := range []string{"amd64", "arm64"} {
		ref, e := ParseVersion(b, "latest", arch)
		if e != nil || !strings.Contains(ref, "@sha256:") {
			t.Fatal(ref, e)
		}
	}
	for _, c := range []string{"live", "edge", "canary"} {
		if _, e := ParseVersion(b, c, "amd64"); e == nil {
			t.Fatal("silent fallback")
		}
	}
	if _, e := ParseVersion(b, "latest", "riscv64"); e == nil {
		t.Fatal("arch")
	}
}
func TestPermissions(t *testing.T) {
	real := Instance{ID: ID(), Kind: "keyed"}
	fake := Instance{ID: ID(), Kind: "fake"}
	for _, method := range []string{"ship.start", "ship.stop", "ship.restart", "ship.archive", "ship.purge", "ship.code", "ship.exec", "ship.logs", "ship.dojo"} {
		q := Request{Method: method, Confirm: real.ID}
		if e := authorize("agent", q, &real); e == nil {
			t.Fatalf("agent real access %s", method)
		}
		if e := authorize("agent", q, &fake); e != nil {
			t.Fatalf("agent fake %s: %v", method, e)
		}
	}
	if e := authorize("owner", Request{Method: "ship.stop"}, &real); e == nil {
		t.Fatal("missing confirmation")
	}
	if e := authorize("agent", Request{Method: "ship.create", Kind: "comet"}, nil); e == nil {
		t.Fatal("agent comet")
	}
}
func TestCreateValidation(t *testing.T) {
	q := Request{Label: "test", Kind: "fake", Group: "demo", Identity: "~zod"}
	if e := validateCreate(&q); e != nil {
		t.Fatal(e)
	}
	for _, image := range []string{"$(rm -rf /)", "repo image", "--privileged", "repo\n"} {
		if e := ValidateImage(image); e == nil {
			t.Fatalf("accepted %s", image)
		}
	}
	q = Request{Label: "test", Kind: "keyed", Identity: "sampel-palnet"}
	if e := validateCreate(&q); e == nil {
		t.Fatal("missing key fell back")
	}
	q = Request{Label: "test", Kind: "comet", Key: "private"}
	if e := validateCreate(&q); e == nil {
		t.Fatal("key accepted for comet")
	}
}
func TestDockerIsolationPlan(t *testing.T) {
	account, group := AccountFiles(1000, 1000)
	if !strings.Contains(account, "runner:x:1000:1000:") || !strings.Contains(group, "runner:x:1000:") {
		t.Fatal("missing numeric-user account mapping")
	}
	p := testPaths(t)
	e := Engine{Paths: p, Host: HostConfig{ID: ID()}}
	i := Instance{ID: ID(), Kind: "fake", Container: "test", Image: "example/urbit", HTTPPort: 8080, AmesPort: 31337, Loom: 31}
	g := Group{ID: ID(), Namespace: "namespace"}
	a := e.CreateArgs(i, &g, "/data/dev/test", "")
	s := strings.Join(a, " ")
	if !strings.Contains(s, "--network container:namespace") || strings.Contains(s, "--publish") || strings.Contains(s, "--privileged") {
		t.Fatal(s)
	}
	i.Kind = "keyed"
	i.HostPort = 41000
	s = strings.Join(e.CreateArgs(i, nil, "/data/real/test", "/private/keys"), " ")
	if !strings.Contains(s, "127.0.0.1:41000:8080/tcp") || strings.Contains(s, "network.key") || strings.Contains(s, "docker.sock") {
		t.Fatal(s)
	}
}
func TestBudgetWarnings(t *testing.T) {
	v := BudgetMetrics([]map[string]string{{"Name": "ship", "CPUPerc": "125.00%", "MemUsage": "1.5GiB / 128GiB"}}, HostConfig{CPUWarning: 100, MemoryWarning: 1 << 30})
	if v["fleet_memory_bytes"].(uint64) != 1610612736 || len(v["warnings"].([]string)) != 2 || v["hard_limits"] != false {
		t.Fatal(v)
	}
}

// fakeDocker exercises real manager and Docker argv without a Docker daemon.
// It is a test double, not evidence of a real Vere boot.
type fakeDocker struct {
	mu         sync.Mutex
	containers map[string]map[string]any
	networks   map[string]string
	calls      [][]string
	host       string
	failCreate bool
}

func newFake(host string) *fakeDocker {
	return &fakeDocker{containers: map[string]map[string]any{}, networks: map[string]string{}, host: host}
}
func flagValue(a []string, k string) string {
	for i, v := range a {
		if v == k && i+1 < len(a) {
			return a[i+1]
		}
	}
	return ""
}
func (d *fakeDocker) Run(ctx context.Context, a []string, in []byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, append([]string{}, a...))
	out := func(v any) ([]byte, error) { b, e := json.Marshal(v); return b, e }
	if len(a) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	switch a[0] {
	case "info":
		return []byte(`"test-docker"`), nil
	case "image":
		if flagValue(a, "--format") == "{{.Id}}" {
			return []byte("sha256:" + strings.Repeat("a", 64)), nil
		}
		return []byte("[]"), nil
	case "pull", "update":
		return []byte("ok"), nil
	case "network":
		switch a[1] {
		case "ls":
			name := strings.TrimSuffix(strings.TrimPrefix(flagValue(a, "--filter"), "name=^"), "$")
			if _, ok := d.networks[name]; ok {
				return []byte(name), nil
			}
			return nil, nil
		case "create":
			d.networks[a[len(a)-1]] = d.host
			return []byte("netid"), nil
		case "inspect":
			return []byte(d.host), nil
		}
	case "container":
		if a[1] == "ls" {
			name := strings.TrimSuffix(strings.TrimPrefix(flagValue(a, "--filter"), "name=^/"), "$")
			if _, ok := d.containers[name]; ok {
				return []byte(name), nil
			}
			return nil, nil
		}
		if a[1] == "inspect" {
			m, ok := d.containers[a[len(a)-1]]
			if !ok {
				return nil, fmt.Errorf("missing")
			}
			return out([]any{m})
		}
	case "run", "create":
		if d.failCreate {
			return nil, fmt.Errorf("injected create failure")
		}
		name := flagValue(a, "--name")
		labels := map[string]any{}
		for n, v := range a {
			if v == "--label" && n+1 < len(a) {
				parts := strings.SplitN(a[n+1], "=", 2)
				labels[parts[0]] = parts[1]
			}
		}
		state := "created"
		if a[0] == "run" {
			state = "running"
		}
		d.containers[name] = map[string]any{"State": map[string]any{"Status": state}, "Config": map[string]any{"Labels": labels}}
		return []byte("containerid"), nil
	case "start", "kill":
		name := a[len(a)-1]
		m, ok := d.containers[name]
		if !ok {
			return nil, fmt.Errorf("missing")
		}
		state := "running"
		if a[0] == "kill" {
			state = "exited"
		}
		m["State"].(map[string]any)["Status"] = state
		return nil, nil
	case "rm":
		delete(d.containers, a[len(a)-1])
		return nil, nil
	case "exec":
		if strings.Contains(string(in), `"dojo":"our"`) {
			return out("~sampel-palnet")
		}
		return out("sampel-palnet-sampel-palnet")
	case "stats":
		return []byte(`{"Name":"test","CPUPerc":"0.2%","MemUsage":"1MiB / 8GiB"}`), nil
	case "logs":
		return []byte("test runtime log"), nil
	}
	return nil, fmt.Errorf("unhandled fake Docker command: %v", a)
}
func testManager(t *testing.T) (*Manager, *fakeDocker) {
	t.Helper()
	p := testPaths(t)
	h := HostConfig{Schema: 1, ID: ID(), Name: "workstation", DataRoot: p.Data, Channel: "latest", ImageOverride: "example/urbit:test"}
	d := newFake(h.ID)
	m, e := NewManager(&Engine{Paths: p, Host: h, Exec: d})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(m.Close)
	return m, d
}
func request(t *testing.T, m *Manager, role string, q Request) Response {
	t.Helper()
	q.Schema = 1
	if q.ID == "" {
		q.ID = ID()
	}
	return m.Handle(context.Background(), role, q)
}
func accepted(t *testing.T, v Response) Operation {
	t.Helper()
	if !v.OK {
		t.Fatalf("request failed: %+v", v.Error)
	}
	var o Operation
	if e := json.Unmarshal(v.Result, &o); e != nil {
		t.Fatal(e)
	}
	return o
}
func await(t *testing.T, m *Manager, id string) Operation {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		o := m.DB.Operations[id]
		m.mu.Unlock()
		if o.State == "succeeded" || o.State == "failed" {
			return o
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("operation timed out")
	return Operation{}
}
func bootFake(t *testing.T, m *Manager, label, group string) Operation {
	t.Helper()
	v := request(t, m, "agent", Request{Method: "group.create", Label: group})
	if !v.OK {
		t.Fatal(v.Error)
	}
	o := accepted(t, request(t, m, "agent", Request{Method: "ship.create", Label: label, Kind: "fake", Identity: "zod", Group: group}))
	o = await(t, m, o.ID)
	if o.State != "succeeded" {
		t.Fatal(o.Message)
	}
	return o
}
func TestLifecycleDedupArchivePurge(t *testing.T) {
	m, d := testManager(t)
	request(t, m, "agent", Request{Method: "group.create", Label: "demo"})
	q := Request{ID: "retry-token", Method: "ship.create", Label: "demo-zod", Kind: "fake", Identity: "zod", Group: "demo"}
	first := accepted(t, request(t, m, "agent", q))
	same := accepted(t, request(t, m, "agent", q))
	if same.ID != first.ID {
		t.Fatal("duplicate operation")
	}
	q.Label = "another"
	if v := request(t, m, "agent", q); v.OK || v.Error.Code != "conflict" {
		t.Fatal(v)
	}
	if o := await(t, m, first.ID); o.State != "succeeded" {
		t.Fatal(o)
	}
	stop := accepted(t, request(t, m, "agent", Request{Method: "ship.stop", Target: first.Instance}))
	if o := await(t, m, stop.ID); o.State != "succeeded" {
		t.Fatal(o)
	}
	d.mu.Lock()
	seenDisable, seenTerm := false, false
	for _, a := range d.calls {
		if strings.Join(a, " ") == "update --restart=no ou-ship-"+first.Instance {
			seenDisable = true
		}
		if a[0] == "kill" {
			seenTerm = true
			if !seenDisable {
				t.Fatal("signal before disabling restart")
			}
		}
	}
	d.mu.Unlock()
	if !seenTerm {
		t.Fatal("no graceful signal")
	}
	start := accepted(t, request(t, m, "agent", Request{Method: "ship.start", Target: first.Instance}))
	if o := await(t, m, start.ID); o.State != "succeeded" {
		t.Fatal(o)
	}
	m.mu.Lock()
	i := m.DB.Instances[first.Instance]
	m.mu.Unlock()
	pier := filepath.Join(m.Engine.Host.DataRoot, i.RelativeDir(), "pier")
	os.MkdirAll(pier, 0700)
	os.WriteFile(filepath.Join(pier, "keep-me"), []byte("state"), 0600)
	if v := request(t, m, "agent", Request{Method: "ship.purge", Target: i.ID, Confirm: i.ID}); v.OK {
		t.Fatal("purged unarchived")
	}
	archive := accepted(t, request(t, m, "agent", Request{Method: "ship.archive", Target: i.ID}))
	if o := await(t, m, archive.ID); o.State != "succeeded" {
		t.Fatal(o)
	}
	dst := filepath.Join(m.Engine.Host.DataRoot, "archive", i.ID, "pier", "keep-me")
	if b, e := os.ReadFile(dst); e != nil || string(b) != "state" {
		t.Fatal("pier lost", e)
	}
	if v := request(t, m, "agent", Request{Method: "ship.purge", Target: i.ID}); v.OK {
		t.Fatal("purge without confirmation")
	}
	purge := accepted(t, request(t, m, "agent", Request{Method: "ship.purge", Target: i.ID, Confirm: i.ID}))
	if o := await(t, m, purge.ID); o.State != "succeeded" {
		t.Fatal(o)
	}
	if _, e := os.Stat(dst); !os.IsNotExist(e) {
		t.Fatal("purge failed")
	}
}
func TestIndependentFakeGroupsAndRealGate(t *testing.T) {
	m, _ := testManager(t)
	one := bootFake(t, m, "one-zod", "one")
	two := bootFake(t, m, "two-zod", "two")
	m.mu.Lock()
	a := m.DB.Instances[one.Instance]
	b := m.DB.Instances[two.Instance]
	m.mu.Unlock()
	if a.Group == b.Group || a.ID == b.ID {
		t.Fatal("groups conflated")
	}
	if v := request(t, m, "agent", Request{Method: "ship.create", Kind: "fake", Label: "duplicate", Identity: "zod", Group: "one"}); v.OK {
		t.Fatal("duplicate galaxy")
	}
	if v := request(t, m, "owner", Request{Method: "ship.create", Kind: "comet", Label: "real"}); v.OK || v.Error.Code != "pilot_gate" {
		t.Fatal(v)
	}
}
func TestRestartMarksInterrupted(t *testing.T) {
	m, _ := testManager(t)
	m.DB.Operations["test"] = Operation{ID: "test", State: "running"}
	if e := m.persist(); e != nil {
		t.Fatal(e)
	}
	next, e := NewManager(m.Engine)
	if e != nil {
		t.Fatal(e)
	}
	defer next.Close()
	if next.DB.Operations["test"].State != "interrupted" {
		t.Fatal("forgot interruption")
	}
}
func TestDockerFailureNotReportedAsStopped(t *testing.T) {
	p := testPaths(t)
	e := Engine{Paths: p, Host: HostConfig{ID: ID()}, Exec: failingExecutor{}}
	o := e.Observe(context.Background(), Instance{ID: ID(), Container: "x", Kind: "fake"}, nil)
	if o.Container != "unknown" || o.Runtime != "unknown" || o.GUIEnabled {
		t.Fatal(o)
	}
}

type failingExecutor struct{}

func (failingExecutor) Run(context.Context, []string, []byte) ([]byte, error) {
	return nil, fmt.Errorf("unreachable daemon")
}
func TestFailedCreationRecoverableByArchive(t *testing.T) {
	m, d := testManager(t)
	d.failCreate = true
	request(t, m, "agent", Request{Method: "group.create", Label: "demo"})
	o := accepted(t, request(t, m, "agent", Request{Method: "ship.create", Kind: "fake", Label: "failed", Identity: "zod", Group: "demo"}))
	if o = await(t, m, o.ID); o.State != "failed" {
		t.Fatal(o)
	}
	a := accepted(t, request(t, m, "agent", Request{Method: "ship.archive", Target: o.Instance}))
	if a = await(t, m, a.ID); a.State != "succeeded" {
		t.Fatal(a)
	}
}

func TestKeyedSecretNotJournaledAndRemovedAfterIdentityCheck(t *testing.T) {
	m, d := testManager(t)
	m.Engine.Host.AllowReal = true
	secret := "test-only-not-an-actual-network-key-3fb968ed"
	q := Request{Method: "ship.create", ID: "keyed-test", Kind: "keyed", Identity: "sampel-palnet", Label: "disposable", Key: secret}
	o := accepted(t, request(t, m, "owner", q))
	o = await(t, m, o.ID)
	if o.State != "succeeded" {
		t.Fatal(o)
	}
	m.mu.Lock()
	i := m.DB.Instances[o.Instance]
	m.mu.Unlock()
	key := filepath.Join(m.Engine.Host.DataRoot, i.RelativeDir(), ".boot-secret", "network.key")
	b, e := os.ReadFile(key)
	if e != nil || string(b) != secret {
		t.Fatal("boot secret absent", e)
	}
	st, _ := os.Stat(key)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	state, e := os.ReadFile(m.dbPath)
	if e != nil || strings.Contains(string(state), secret) {
		t.Fatal("secret in journal", e)
	}
	d.mu.Lock()
	for _, a := range d.calls {
		if strings.Contains(strings.Join(a, " "), secret) {
			t.Fatal("key in Docker argv")
		}
	}
	d.mu.Unlock()
	m.CompleteBirth(context.Background())
	if _, e = os.Stat(key); !os.IsNotExist(e) {
		t.Fatal("key not removed after verified identity", e)
	}
	if v := request(t, m, "agent", Request{Method: "ship.code", Target: i.ID}); v.OK {
		t.Fatal("agent read real code")
	}
}
func TestMetricsFailRatherThanReportZeroForUnavailableDocker(t *testing.T) {
	e := Engine{Exec: failingExecutor{}}
	if _, err := e.Stats(context.Background(), []Instance{{ID: ID(), Kind: "fake", Container: "x"}}); err == nil {
		t.Fatal("unavailable Docker became zero metrics")
	}
}
