package managed

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/ownerhost"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestHostCompatibilityAuthenticatedReadAndFailureIsolation(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var calls atomic.Int32
	r.hostCompatibility.inspect = func() (hostcompat.Report, error) {
		calls.Add(1)
		return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil
	}
	r.hostCompatibility.owner = func() (*ownerhost.Profile, error) { return nil, nil }
	observer, observerID := grantLine(t, "observer")
	operator, operatorID := grantLine(t, "operator")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err := os.WriteFile(r.keys, []byte(observer+"\n"+operator+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(privateDir(t), "rpc.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: r, ConnContext: PeerContext}
	go srv.Serve(ln)
	defer srv.Close()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", path)
	}}
	defer transport.CloseIdleConnections()
	c := &http.Client{Transport: transport, Timeout: time.Second}
	request := func(id, method, path string) (int, []byte) {
		t.Helper()
		req, _ := http.NewRequest(method, "http://runner"+path, nil)
		req.Header.Set(protocol.Header, "1")
		if id != "" {
			req.Header.Set(access.Header, id)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, body
	}
	for _, id := range []string{observerID, operatorID, ""} {
		status, body := request(id, "GET", "/hello")
		var hello protocol.Hello
		if status != 200 || json.Unmarshal(body, &hello) != nil || !slices.Contains(hello.Features, protocol.HostCompatibilityFeature) {
			t.Fatalf("hello: %d %s", status, body)
		}
		status, body = request(id, "GET", protocol.HostCompatibilityPath)
		var report protocol.HostCompatibilityReport
		if status != 200 || json.Unmarshal(body, &report) != nil || !protocol.ValidHostCompatibility(report) || report.RunnerID != hello.RunnerID || report.Mode != hello.Mode || report.Report.Provenance == "verified" {
			t.Fatalf("bound metadata-only report: %d %s", status, body)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("metadata was inspected per poll", calls.Load())
	}
	for _, id := range []string{observerID, operatorID} {
		for _, method := range []string{"POST", "PUT", "DELETE"} {
			if status, _ := request(id, method, protocol.HostCompatibilityPath); status < 400 {
				t.Fatal("host report accepted mutation", method)
			}
		}
		if status, _ := request(id, "GET", protocol.HostCompatibilityPath+"?verify=true"); status != 400 {
			t.Fatal("remote verification parameter accepted")
		}
	}
	unauth := httptest.NewRecorder()
	req := httptest.NewRequest("GET", protocol.HostCompatibilityPath, nil)
	req.Header.Set(protocol.Header, "1")
	r.ServeHTTP(unauth, req)
	if unauth.Code != 403 {
		t.Fatal("missing Unix peer accepted")
	}
	if err := os.WriteFile(r.keys, []byte(operator+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if status, _ := request(observerID, "GET", protocol.HostCompatibilityPath); status != 403 {
		t.Fatal("revoked observer read cached evidence")
	}
	r.hostCompatibility.live.mu.Lock()
	r.hostCompatibility.live.expires = time.Time{}
	r.hostCompatibility.live.mu.Unlock()
	r.hostCompatibility.native.mu.Lock()
	r.hostCompatibility.native.expires = time.Time{}
	r.hostCompatibility.inspect = func() (hostcompat.Report, error) {
		return hostcompat.Report{}, errors.New("secret repository credential and arbitrary stderr")
	}
	r.hostCompatibility.native.mu.Unlock()
	status, body := request(operatorID, "GET", protocol.HostCompatibilityPath)
	if status != 503 || strings.Contains(string(body), "secret") || !strings.Contains(string(body), "host_report_unavailable") || r.failed.Load() {
		t.Fatalf("host error poisoned journal or leaked: %d %s", status, body)
	}
	if status, body := request(observerID, "GET", "/v1/snapshot"); status != 403 {
		t.Fatalf("revocation lost: %d %s", status, body)
	}
	if status, body := request(operatorID, "GET", "/v1/snapshot"); status != 200 || strings.Contains(string(body), "state_store_error") {
		t.Fatalf("host failure broke inventory: %d %s", status, body)
	}
	// Exercise admission after the unrelated error, rather than only checking a flag.
	post := httptest.NewRequest("POST", "/v1/instances", strings.NewReader(`{"id":"alpha","request_id":"create","config":{"http_port":18080,"ames_port":34343,"loom":31}}`))
	post = post.WithContext(context.WithValue(post.Context(), peerKey{}, uint32(os.Geteuid())))
	post.Header.Set(protocol.Header, "1")
	post.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, post)
	if w.Code != 201 {
		t.Fatalf("host error fenced valid admission: %d %s", w.Code, w.Body.String())
	}
}

func TestHostCompatibilityCacheSingleFlightAndCanceledWait(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	cache := hostCompatibilityCache{owner: func() (*ownerhost.Profile, error) { return nil, nil }, inspect: func() (hostcompat.Report, error) {
		calls.Add(1)
		<-release
		return hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64"), nil
	}}
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			if _, err := cache.get(ctx, context.Background(), "runner", "development-managed"); !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("wait did not time out: %v", err)
			}
		}()
	}
	wg.Wait()
	close(release)
	if _, err := cache.get(context.Background(), context.Background(), "runner", "development-managed"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("timed-out polls created more inspectors", calls.Load())
	}
	cache.live.mu.Lock()
	cache.live.expires = time.Time{}
	cache.live.mu.Unlock()
	cache.native.mu.Lock()
	cache.native.expires = time.Time{}
	cache.native.mu.Unlock()
	if _, err := cache.get(context.Background(), context.Background(), "runner", "development-managed"); err != nil || calls.Load() != 2 {
		t.Fatal("expired cache not refreshed", err, calls.Load())
	}
}
