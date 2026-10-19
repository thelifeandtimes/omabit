//go:build linux

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func expireHostEvidence(c *hostCompatibilityCache, native bool) {
	c.live.mu.Lock()
	c.live.expires = time.Time{}
	c.live.mu.Unlock()
	if native {
		c.native.mu.Lock()
		c.native.expires = time.Time{}
		c.native.mu.Unlock()
	}
}

func TestHostCompatibilityLivePrivateAPIAndSeparateNativeCache(t *testing.T) {
	if os.Geteuid() == 0 || os.Getegid() == 0 {
		t.Skip("rootless socket fixture requires an ordinary test owner")
	}
	t.Setenv("DOCKER_HOST", "unix:///untrusted/shared.sock")
	t.Setenv("DOCKER_CONTEXT", "untrusted")
	t.Setenv("HTTP_PROXY", "http://untrusted.invalid")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	dataRoot := filepath.Join(dir, ".local/share/urbit-runner/docker")
	if err := os.MkdirAll(dataRoot, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "engine.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var state, apiCalls, nativeCalls atomic.Int32
	var installedNew, receiptInvalid atomic.Bool
	engine := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		apiCalls.Add(1)
		if req.Method != "GET" || req.URL.Path != "/version" && req.URL.Path != "/v1.47/info" {
			t.Error("unexpected engine operation", req.Method, req.URL)
			http.Error(w, "forbidden", 400)
			return
		}
		if state.Load() == 5 {
			http.Error(w, "private-secret engine error", 503)
			return
		}
		version := "29.7.2"
		if state.Load() >= 2 {
			version = "29.9.1"
		}
		if state.Load() == 3 {
			version = "30.0.0"
		}
		if req.URL.Path == "/version" {
			json.NewEncoder(w).Encode(map[string]any{"Version": version, "ApiVersion": "1.54", "MinAPIVersion": "1.44"})
			return
		}
		root := dataRoot
		if state.Load() == 4 {
			root = "/private-secret/wrong-data-root"
		}
		json.NewEncoder(w).Encode(map[string]any{"ServerVersion": version, "SecurityOptions": []string{"name=rootless"}, "CgroupDriver": "systemd", "CgroupVersion": "2", "DockerRootDir": root})
	})}
	go engine.Serve(ln)
	defer engine.Close()
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.hostCompatibility.inspect = func() (hostcompat.Report, error) {
		nativeCalls.Add(1)
		report := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
		report.InstalledEngineVersion = "29.7.2"
		if installedNew.Load() {
			report.InstalledEngineVersion = "29.9.1"
		}
		// Package authentication itself is tested by hostcompat. This fixture supplies
		// verified evidence so the RPC must preserve Evaluate's actual mutation result.
		for i := range report.Packages {
			report.Packages[i].Provenance = "verified"
			report.Packages[i].Executables = []string{"/usr/bin/fixture"}
			if receiptInvalid.Load() {
				report.Packages[i].Provenance = "metadata-only"
			}
		}
		return report, nil
	}
	r.hostCompatibility.owner = func() (*ownerhost.Profile, error) {
		return &ownerhost.Profile{Version: 1, Name: "fixture-owner", UID: uint32(os.Geteuid()), GID: uint32(os.Getegid()), Home: dir, Piers: filepath.Join(dir, "piers")}, nil
	}
	r.hostCompatibility.probe = func(ctx context.Context, selectedSocket, selectedRoot string) (hostcompat.Engine, error) {
		if selectedSocket != fmt.Sprintf("/run/user/%d/urbit-docker.sock", os.Geteuid()) || selectedRoot != dataRoot {
			t.Error("probe selected an environment/untrusted path", selectedSocket, selectedRoot)
			return hostcompat.Engine{}, errors.New("invalid target")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > time.Second {
			t.Error("probe missing bounded deadline")
		}
		// Only the test redirects the fixed installed socket to a disposable Unix
		// listener. Ownership, peer credentials and both API calls remain real.
		return hostcompat.ProbeEngine(ctx, socket, selectedRoot)
	}
	request := func() protocol.HostCompatibilityReport {
		t.Helper()
		req := httptest.NewRequest("GET", protocol.HostCompatibilityPath, nil)
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
		req.Header.Set(protocol.Header, "1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var report protocol.HostCompatibilityReport
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &report) != nil || !protocol.ValidHostCompatibility(report) || r.failed.Load() {
			t.Fatalf("report failed or poisoned journal: %d %s", w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), dir) || strings.Contains(w.Body.String(), "private-secret") {
			t.Fatal("private host data leaked", w.Body.String())
		}
		if report.MetadataObservedAt == nil || report.ExpiresAt.Sub(*report.MetadataObservedAt) > 30*time.Second || report.ExpiresAt.Sub(report.Report.ObservedAt) > hostEngineMaxAge {
			t.Fatal("report overstated evidence freshness", report)
		}
		return report
	}
	first := request()
	if first.Report.Engine == nil || first.Report.Engine.Version != "29.7.2" || !first.Report.CanStop || !first.Report.CanMutate || first.Report.RestartPending {
		t.Fatalf("live result lost: %+v", first.Report)
	}
	request()
	if apiCalls.Load() != 2 || nativeCalls.Load() != 1 {
		t.Fatal("poll repeated expensive evidence", apiCalls.Load(), nativeCalls.Load())
	}
	installedNew.Store(true)
	expireHostEvidence(&r.hostCompatibility, false)
	request()
	if nativeCalls.Load() != 1 || apiCalls.Load() != 4 {
		t.Fatal("live refresh reran native inspection")
	}
	// Simulate reaching the native 30-second freshness bound after an OS update.
	expireHostEvidence(&r.hostCompatibility, true)
	updated := request()
	if updated.Report.InstalledEngineVersion != "29.9.1" || updated.Report.Engine.Version != "29.7.2" || !updated.Report.RestartPending || !updated.Report.CanMutate || !updated.Report.CanStop || nativeCalls.Load() != 2 {
		t.Fatalf("package/daemon drift lost: %+v", updated.Report)
	}
	state.Store(2)
	expireHostEvidence(&r.hostCompatibility, false)
	restarted := request()
	if restarted.Report.RestartPending || !restarted.Report.CanMutate || restarted.Report.Engine.Version != "29.9.1" || nativeCalls.Load() != 2 {
		t.Fatalf("new daemon not observed: %+v", restarted.Report)
	}
	receiptInvalid.Store(true)
	expireHostEvidence(&r.hostCompatibility, true)
	unverified := request()
	if unverified.Report.CanMutate || !unverified.Report.CanStop || unverified.Report.Provenance == "verified" {
		t.Fatalf("receipt change remained trusted: %+v", unverified.Report)
	}
	state.Store(3)
	expireHostEvidence(&r.hostCompatibility, false)
	unsupported := request()
	if unsupported.Report.Compatibility != "unsupported" || !unsupported.Report.CanStop || unsupported.Report.CanMutate || unsupported.Report.Engine == nil || unsupported.Report.Engine.DataRoot != "" {
		t.Fatalf("unsupported control/read or redaction lost: %+v", unsupported.Report)
	}
	for _, failure := range []int32{4, 5} {
		state.Store(failure)
		expireHostEvidence(&r.hostCompatibility, false)
		unavailable := request()
		if unavailable.Report.Engine != nil || unavailable.Report.CanMutate || unavailable.Report.CanStop || !strings.Contains(fmt.Sprint(unavailable.Report.Checks), "host.engine") {
			t.Fatalf("unsafe/unavailable engine admitted: %+v", unavailable.Report)
		}
		if _, err := r.s.snapshot(); err != nil || r.failed.Load() {
			t.Fatal("engine failure affected journal", err)
		}
	}
}

func TestHostReportRequiresCurrentInstalledOwner(t *testing.T) {
	for _, scenario := range []string{"absent", "unreadable", "root", "different-uid"} {
		t.Run(scenario, func(t *testing.T) {
			c := hostCompatibilityCache{
				inspect: func() (hostcompat.Report, error) { return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil },
				owner: func() (*ownerhost.Profile, error) {
					if scenario == "absent" {
						return nil, nil
					}
					if scenario == "unreadable" {
						return nil, errors.New("private-secret owner error")
					}
					p := &ownerhost.Profile{Version: 1, Name: "owner", UID: uint32(os.Geteuid()) + 1, GID: 1000, Home: "/home/owner", Piers: "/srv/urbit/piers"}
					if scenario == "root" {
						p.Name, p.UID = "root", 0
					}
					return p, nil
				},
				probe: func(context.Context, string, string) (hostcompat.Engine, error) {
					t.Error("untrusted owner reached engine")
					return hostcompat.Engine{}, nil
				},
			}
			v, err := c.get(context.Background(), context.Background(), "runner", "development-managed")
			body, _ := json.Marshal(v)
			if err != nil || v.Report.Engine != nil || v.Report.CanStop || v.Report.CanMutate || strings.Contains(string(body), "private-secret") || !strings.Contains(string(body), "host.owner") {
				t.Fatal(string(body), err)
			}
		})
	}
}

func TestHostReportDeadlineBoundsSingleFlightWorkers(t *testing.T) {
	lifetime, stop := context.WithCancel(context.Background())
	defer stop()
	release := make(chan struct{})
	var inspections atomic.Int32
	c := hostCompatibilityCache{inspect: func() (hostcompat.Report, error) {
		inspections.Add(1)
		<-release // simulate a filesystem read without a context-aware API
		return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil
	}}
	started := time.Now()
	if _, err := c.get(context.Background(), lifetime, "runner", "development-managed"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("shared refresh did not time out", err)
	}
	if time.Since(started) > hostReportTimeout+time.Second {
		t.Fatal("report deadline exceeded budget")
	}
	c.native.mu.Lock()
	flight := c.native.pending
	c.native.mu.Unlock()
	if flight == nil {
		t.Fatal("unresponsive inspector slot was released")
	}
	for range 3 {
		expireHostEvidence(&c, false)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, err := c.get(ctx, lifetime, "runner", "development-managed")
		cancel()
		if err == nil {
			t.Fatal("blocked inspection returned evidence")
		}
	}
	if inspections.Load() != 1 {
		t.Fatal("timed-out polls spawned more inspectors", inspections.Load())
	}
	stop()
	close(release)
	select {
	case <-flight.done:
	case <-time.After(time.Second):
		t.Fatal("inspector did not release its slot")
	}
	if _, err := c.get(context.Background(), lifetime, "runner", "development-managed"); !errors.Is(err, context.Canceled) {
		t.Fatal("server lifetime ignored", err)
	}
}

func TestHostReportProbeUsesServerLifetime(t *testing.T) {
	if os.Geteuid() == 0 || os.Getegid() == 0 {
		t.Skip("requires non-root owner")
	}
	lifetime, stop := context.WithCancel(context.Background())
	defer stop()
	entered, exited := make(chan struct{}), make(chan struct{})
	c := hostCompatibilityCache{
		inspect: func() (hostcompat.Report, error) { return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil },
		owner: func() (*ownerhost.Profile, error) {
			return &ownerhost.Profile{Version: 1, Name: "owner", UID: uint32(os.Geteuid()), GID: uint32(os.Getegid()), Home: "/home/owner", Piers: "/srv/urbit/piers"}, nil
		},
		probe: func(ctx context.Context, _, _ string) (hostcompat.Engine, error) {
			close(entered)
			<-ctx.Done()
			close(exited)
			return hostcompat.Engine{}, ctx.Err()
		},
	}
	done := make(chan error, 1)
	go func() { _, err := c.get(context.Background(), lifetime, "runner", "development-managed"); done <- err }()
	<-entered
	stop()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("probe outlived server")
	}
}

func TestHostEvidenceCacheBoundsNonCancelableLoader(t *testing.T) {
	var c hostEvidenceCache
	release := make(chan struct{})
	started := time.Now()
	_, err := c.get(context.Background(), context.Background(), func(context.Context) (hostEvidence, error) {
		<-release
		return hostEvidence{}, nil
	})
	// Release even on a failed assertion; no test worker may outlive its fixture.
	c.mu.Lock()
	flight := c.pending
	c.mu.Unlock()
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > hostReportTimeout+time.Second || flight == nil {
		t.Fatal("noncancelable loader escaped deadline or lost its worker slot", err)
	}
	select {
	case <-flight.done:
	case <-time.After(time.Second):
		t.Fatal("finished loader did not release its slot")
	}
}
