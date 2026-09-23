package demo

import (
	"bytes"
	"context"
	"encoding/json"
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
)

func newRunner(t *testing.T, delay time.Duration) *Runner {
	t.Helper()
	runner := New(delay)
	t.Cleanup(runner.Close)
	return runner
}

func request(t *testing.T, runner *Runner, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	runner.ServeHTTP(w, req)
	return w
}

func readJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func requireError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) protocol.APIError {
	t.Helper()
	problem := readJSON[protocol.ErrorResponse](t, w).Error
	if w.Code != status || problem.Code != code {
		t.Fatalf("got HTTP %d %s, want HTTP %d %s", w.Code, w.Body, status, code)
	}
	return problem
}

func TestHelloAndSnapshot(t *testing.T) {
	runner := newRunner(t, time.Hour)
	w := request(t, runner, "GET", "/hello", nil)
	hello := readJSON[protocol.Hello](t, w)
	if w.Code != 200 || hello.ProtocolVersion != 1 || hello.Mode != "simulation" || hello.RunnerID == "" {
		t.Fatalf("unexpected hello: %+v", hello)
	}
	snapshot := readJSON[protocol.Snapshot](t, request(t, runner, "GET", "/v1/snapshot", nil))
	if snapshot.RunnerID != hello.RunnerID || snapshot.Mode != "simulation" || len(snapshot.Instances) != 2 ||
		snapshot.Operations == nil || len(snapshot.Incidents) != 1 || !snapshot.Incidents[0].Active {
		t.Fatalf("unexpected initial snapshot: %+v", snapshot)
	}
	for _, version := range []string{"", "2", "01"} {
		req := httptest.NewRequest("GET", "/v1/snapshot", nil)
		req.Header.Set(protocol.Header, version)
		w := httptest.NewRecorder()
		runner.ServeHTTP(w, req)
		requireError(t, w, 426, "protocol_mismatch")
	}
}

func TestConfigurationRevisionsAndPortReservations(t *testing.T) {
	runner := newRunner(t, time.Hour)
	body := protocol.ConfigureRequest{
		ExpectedRevision: 1,
		Config:           protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 32},
	}
	w := request(t, runner, "PUT", "/v1/instances/alpha/configuration", body)
	instance := readJSON[protocol.Instance](t, w)
	if w.Code != 200 || instance.ConfigRevision != 2 || instance.AppliedRevision != 1 || instance.Config.Loom != 32 {
		t.Fatalf("configuration was not staged: %s", w.Body)
	}
	problem := requireError(t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", body), 409, "revision_conflict")
	if problem.CurrentRevision == nil || *problem.CurrentRevision != 2 {
		t.Fatalf("missing authoritative revision: %+v", problem)
	}
	body.ExpectedRevision = 2
	unchanged := readJSON[protocol.Instance](t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", body))
	if unchanged.ConfigRevision != 2 {
		t.Fatal("no-op changed revision")
	}
	body.Config.HTTPPort = 8081
	requireError(t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", body), 409, "port_conflict")
	body.Config.HTTPPort, body.Config.AmesPort = 8080, 34344
	requireError(t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", body), 409, "port_conflict")
	body.Config.AmesPort = 34343
	requireError(t, request(t, runner, "PUT", "/v1/instances/missing/configuration", body), 404, "instance_not_found")
	for _, loom := range []uint8{0, 30, 34} {
		body.Config.Loom = loom
		requireError(t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", body), 400, "invalid_configuration")
	}
}

func TestConcurrentConfigurationOnlyOneWins(t *testing.T) {
	runner := newRunner(t, time.Hour)
	var wg sync.WaitGroup
	results := make(chan int, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(t, runner, "PUT", "/v1/instances/alpha/configuration", protocol.ConfigureRequest{
				ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 32},
			})
			results <- w.Code
		}()
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for code := range results {
		switch code {
		case 200:
			wins++
		case 409:
			conflicts++
		default:
			t.Fatalf("unexpected code %d", code)
		}
	}
	if wins != 1 || conflicts != 7 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
}

func TestOperationsIdempotencyAndExclusion(t *testing.T) {
	runner := newRunner(t, time.Hour)
	body := protocol.OperationRequest{RequestID: "first", Kind: "start", ExpectedRevision: 1}
	w := request(t, runner, "POST", "/v1/instances/alpha/operations", body)
	op := readJSON[protocol.Operation](t, w)
	if w.Code != 202 || op.Status != "running" || !strings.HasPrefix(op.Actor, "local-uid:") {
		t.Fatalf("unexpected operation: %s", w.Body)
	}
	w = request(t, runner, "POST", "/v1/instances/alpha/operations", body)
	if replay := readJSON[protocol.Operation](t, w); w.Code != 200 || replay.ID != op.ID {
		t.Fatalf("retry did not return same operation: %s", w.Body)
	}
	requireError(t, request(t, runner, "POST", "/v1/instances/bravo/operations", body), 409, "request_conflict")
	body.Kind = "stop"
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", body), 409, "request_conflict")
	body.RequestID = "second"
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", body), 409, "instance_busy")
	requireError(t, request(t, runner, "PUT", "/v1/instances/alpha/configuration", protocol.ConfigureRequest{
		ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 32},
	}), 409, "instance_busy")
	body.Kind = "start"
	w = request(t, runner, "POST", "/v1/instances/bravo/operations", body)
	if w.Code != 202 {
		t.Fatalf("unrelated instance was blocked: %s", w.Body)
	}
	// The idempotency record includes the original expected configuration revision.
	body.RequestID, body.ExpectedRevision = "first", 2
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", body), 409, "request_conflict")
}

func TestCompletedOperationSurvivesClientDisconnect(t *testing.T) {
	runner := newRunner(t, 30*time.Millisecond)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "runner.sock")
	listener, err := localrpc.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: runner, ReadHeaderTimeout: time.Second}
	defer server.Close()
	go server.Serve(listener)
	first := client.New(socket)
	ctx, cancel := context.WithCancel(context.Background())
	_, err = first.Configure(ctx, "alpha", protocol.ConfigureRequest{
		ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 32},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := protocol.OperationRequest{RequestID: "detached", Kind: "start", ExpectedRevision: 2}
	accepted, err := first.Operate(ctx, "alpha", body)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	cancel()
	second := client.New(socket)
	defer second.Close()
	deadline, done := context.WithTimeout(context.Background(), 3*time.Second)
	defer done()
	for {
		snapshot, err := second.Snapshot(deadline)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Instances[0].ObservedState == "running" {
			if snapshot.Instances[0].AppliedRevision != 2 || snapshot.Instances[0].ActiveOperationID != "" ||
				snapshot.Incidents[0].Active || len(snapshot.Operations) != 1 ||
				snapshot.Operations[0].FinishedAt == nil || snapshot.Operations[0].Status != "succeeded" {
				t.Fatalf("unexpected final snapshot: %+v", snapshot)
			}
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	replayed, err := second.Operate(deadline, "alpha", body)
	if err != nil || replayed.ID != accepted.ID || replayed.Status != "succeeded" {
		t.Fatalf("completed request replay: %+v, %v", replayed, err)
	}
	_, err = second.Operate(deadline, "alpha", protocol.OperationRequest{RequestID: "again", Kind: "start", ExpectedRevision: 2})
	if remote, ok := err.(*client.RemoteError); !ok || remote.Problem.Code != "invalid_state" {
		t.Fatalf("expected already-running conflict, got %v", err)
	}
	if _, err := second.Operate(deadline, "alpha", protocol.OperationRequest{RequestID: "stop", Kind: "stop", ExpectedRevision: 2}); err != nil {
		t.Fatal(err)
	}
	for {
		snapshot, err := second.Snapshot(deadline)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Instances[0].ObservedState == "stopped" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRejectMalformedOrForgedRequests(t *testing.T) {
	runner := newRunner(t, time.Hour)
	valid := `{"request_id":"one","kind":"start","expected_revision":1}`
	for _, tc := range []struct {
		name, body, contentType, origin, code string
		status                                int
	}{
		{"empty", "", "application/json", "", "invalid_json", 400},
		{"extra-value", valid + `{}`, "application/json", "", "invalid_json", 400},
		{"too-large", valid + strings.Repeat(" ", 4096), "application/json", "", "invalid_json", 400},
		{"unknown-field", strings.TrimSuffix(valid, "}") + `,"actor":"root"}`, "application/json", "", "invalid_json", 400},
		{"null", "null", "application/json", "", "invalid_operation", 400},
		{"wrong-type", `{"expected_revision":"1"}`, "application/json", "", "invalid_json", 400},
		{"media-type", valid, "text/plain", "", "invalid_content_type", 415},
		{"browser", valid, "application/json", "https://example.com", "origin_not_allowed", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/instances/alpha/operations", strings.NewReader(tc.body))
			req.Header.Set(protocol.Header, "1")
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			runner.ServeHTTP(w, req)
			requireError(t, w, tc.status, tc.code)
		})
	}
	for _, id := range []string{"", "../x", "a b", "a\n", strings.Repeat("x", 65)} {
		requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{
			RequestID: id, Kind: "start", ExpectedRevision: 1,
		}), 400, "invalid_operation")
	}
	for _, path := range []string{"/nope", "/v1/instances/alpha/operations/", "/v1/instances/alpha/do-anything"} {
		requireError(t, request(t, runner, "POST", path, nil), 404, "not_found")
	}
	snapshot := readJSON[protocol.Snapshot](t, request(t, runner, "GET", "/v1/snapshot", nil))
	if len(snapshot.Operations) != 0 {
		t.Fatal("rejected input mutated the runner")
	}
}

func TestOperationLimitsAndRevision(t *testing.T) {
	runner := newRunner(t, time.Hour)
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{
		RequestID: "stale", Kind: "start", ExpectedRevision: 2,
	}), 409, "revision_conflict")
	for i := range maxOperations {
		runner.operations = append(runner.operations, protocol.Operation{ID: fmt.Sprint(i), Status: "succeeded"})
	}
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{
		RequestID: "full", Kind: "start", ExpectedRevision: 1,
	}), 503, "demo_capacity")
	runner.Close()
	requireError(t, request(t, runner, "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{
		RequestID: "closed", Kind: "start", ExpectedRevision: 1,
	}), 503, "runner_stopping")
}

func TestConcurrentDuplicateRequests(t *testing.T) {
	runner := newRunner(t, time.Hour)
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 8)
	body := protocol.OperationRequest{RequestID: "shared", Kind: "start", ExpectedRevision: 1}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- request(t, runner, "POST", "/v1/instances/alpha/operations", body)
		}()
	}
	wg.Wait()
	close(results)
	accepted := 0
	opID := ""
	for result := range results {
		if result.Code == 202 {
			accepted++
		} else if result.Code != 200 {
			t.Fatalf("unexpected result: %d %s", result.Code, result.Body)
		}
		op := readJSON[protocol.Operation](t, result)
		if opID != "" && opID != op.ID {
			t.Fatal("duplicate request executed twice")
		}
		opID = op.ID
	}
	if accepted != 1 || len(runner.operations) != 1 {
		t.Fatalf("accepted=%d history=%d", accepted, len(runner.operations))
	}
	// A replay identifies the already accepted request, even if current
	// configuration or capacity would prevent submitting new work.
	runner.mu.Lock()
	instance := runner.instances["alpha"]
	instance.ConfigRevision = 2
	runner.instances["alpha"] = instance
	for len(runner.operations) < maxOperations {
		runner.operations = append(runner.operations, protocol.Operation{Status: "succeeded"})
	}
	runner.mu.Unlock()
	w := request(t, runner, "POST", "/v1/instances/alpha/operations", body)
	if op := readJSON[protocol.Operation](t, w); w.Code != 200 || op.ID != opID {
		t.Fatalf("replay revalidated new work: %s", w.Body)
	}
}
