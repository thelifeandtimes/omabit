package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type fakeBackend struct {
	mu      sync.Mutex
	running map[string]bool
	apply   int
	entered chan struct{}
	release chan struct{}
}

func (b *fakeBackend) Apply(ctx context.Context, action string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	b.mu.Lock()
	b.apply++
	b.mu.Unlock()
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	if b.entered != nil {
		b.entered <- struct{}{}
	}
	if b.release != nil {
		select {
		case <-ctx.Done():
			return state, runtimequalify.ManagedObservation{}, ctx.Err()
		case <-b.release:
		}
	}
	b.mu.Lock()
	if b.running == nil {
		b.running = map[string]bool{}
	}
	b.running[state.ID] = action == "start"
	b.mu.Unlock()
	if action == "start" {
		state.Prepared, state.Initialized = true, true
	}
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	_, obs, err := b.Observe(ctx, state)
	return state, obs, err
}

func (b *fakeBackend) Observe(_ context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	b.mu.Lock()
	running := b.running[state.ID]
	b.mu.Unlock()
	if running {
		return state, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Endpoint: fmt.Sprintf("http://127.0.0.1:%d", state.Port)}, nil
	}
	if !state.Prepared {
		return state, runtimequalify.ManagedObservation{State: "uninitialized", Health: "unknown"}, nil
	}
	return state, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func testEndpoint(t *testing.T, dir string, b backend) (*Server, *client.Client, func()) {
	t.Helper()
	r, err := newServer(dir, b, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(privateDir(t), "runner.sock")
	listener, err := localrpc.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	httpServer := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: time.Second}
	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	c := client.New(socket)
	var once sync.Once
	shutdown := func() {
		once.Do(func() {
			c.Close()
			_ = httpServer.Close()
			<-done
			if err := r.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(shutdown)
	return r, c, shutdown
}

func waitOperation(t *testing.T, c *client.Client, id, status string) protocol.Snapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		s, err := c.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, op := range s.Operations {
			if op.ID == id && op.Status == status {
				return s
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("operation %s did not reach %s", id, status)
	return protocol.Snapshot{}
}

func TestRealUnixClientsAndRunnerRestart(t *testing.T) {
	dir := privateDir(t)
	b := &fakeBackend{}
	_, c, close := testEndpoint(t, dir, b)
	hello, err := c.Hello(context.Background())
	if err != nil || hello.Mode != mode {
		t.Fatalf("hello: %+v %v", hello, err)
	}
	reg, err := c.CreateFake(context.Background(), registration("alpha", "reg", 18080))
	if err != nil {
		t.Fatal(err)
	}
	if reg.Actor != fmt.Sprintf("local-uid:%d", os.Geteuid()) {
		t.Fatalf("actor was not the Unix peer: %+v", reg)
	}
	op, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
	close() // stopping management does not request a ship stop
	_, c, close = testEndpoint(t, dir, b)
	defer close()
	next, err := c.Hello(context.Background())
	if err != nil || next.RunnerID != hello.RunnerID {
		t.Fatal("runner identity did not survive")
	}
	replay, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1})
	if err != nil || replay.ID != op.ID || replay.Status != "succeeded" {
		t.Fatalf("persistent replay: %+v %v", replay, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 1 {
		t.Fatal("restart replayed a completed action")
	}
	stop, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{RequestID: "stop", Kind: "stop", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	final := waitOperation(t, c, stop.ID, "succeeded")
	if final.Instances[0].ObservedState != "stopped" {
		t.Fatalf("stop: %+v", final.Instances[0])
	}
}

func TestClientDisconnectDoesNotCancelJob(t *testing.T) {
	b := &fakeBackend{entered: make(chan struct{}, 1), release: make(chan struct{})}
	_, c, shutdown := testEndpoint(t, privateDir(t), b)
	defer shutdown()
	if _, err := c.CreateFake(context.Background(), registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1}
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	c.Close()
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("job not started")
	}
	replay, err := c.Operate(context.Background(), "alpha", req)
	if err != nil || replay.ID != op.ID {
		t.Fatal("active replay duplicated action")
	}
	_, err = c.Configure(context.Background(), "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}})
	var remote *client.RemoteError
	if !errors.As(err, &remote) || remote.Problem.Code != "instance_busy" {
		t.Fatalf("active config write allowed: %v", err)
	}
	close(b.release)
	final := waitOperation(t, c, op.ID, "succeeded")
	if final.Instances[0].AppliedRevision != 1 {
		t.Fatal("applied revision missing")
	}
}

func TestPeerAndBodyCannotForgeActor(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, tc := range []struct {
		actor        *uint32
		origin, body string
		status       int
	}{
		{nil, "", `{}`, 403},
		{ptrUID(uint32(os.Geteuid() + 100)), "", `{}`, 403},
		{ptrUID(uint32(os.Geteuid())), "https://untrusted.invalid", `{}`, 403},
		{ptrUID(uint32(os.Geteuid())), "", `{"id":"alpha","request_id":"r","actor":"local-uid:0","config":{"http_port":18080,"ames_port":34343,"loom":31}}`, 400},
	} {
		req := httptest.NewRequest("POST", "/v1/instances", strings.NewReader(tc.body))
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-Actor", "local-uid:0")
		if tc.actor != nil {
			req = req.WithContext(context.WithValue(req.Context(), peerKey{}, *tc.actor))
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("response %d: %s", w.Code, w.Body)
		}
	}
	snapshot, err := r.s.snapshot()
	if err != nil || len(snapshot.Instances) != 0 || len(snapshot.Operations) != 0 {
		t.Fatal("invalid requests changed state")
	}
}

func ptrUID(uid uint32) *uint32 { return &uid }

func TestStateStoreErrorFailsClosed(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.failed.Store(true)
	body, _ := json.Marshal(registration("alpha", "reg", 18080))
	req := httptest.NewRequest("POST", "/v1/instances", strings.NewReader(string(body))).WithContext(context.WithValue(context.Background(), peerKey{}, uint32(os.Geteuid())))
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Fatalf("mutated failed journal: %d", w.Code)
	}
}

func TestServerShutdownJournalsInterruptedWork(t *testing.T) {
	dir := privateDir(t)
	b := &fakeBackend{entered: make(chan struct{}, 1), release: make(chan struct{})}
	_, c, shutdown := testEndpoint(t, dir, b)
	if _, err := c.CreateFake(context.Background(), registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	req := protocol.OperationRequest{RequestID: "start", Kind: "start", ExpectedRevision: 1}
	op, err := c.Operate(context.Background(), "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("operation did not begin")
	}
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(context.Background(), "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "interrupted" {
		t.Fatalf("interrupted replay: %+v %v", replay, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 1 {
		t.Fatal("interrupted operation was re-executed")
	}
}
