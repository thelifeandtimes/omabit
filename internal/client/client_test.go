package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "rpc.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socketPath, 0600); err != nil {
		listener.Close()
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	c := New(socketPath)
	t.Cleanup(func() {
		c.Close()
		if err := server.Close(); err != nil {
			t.Error(err)
		}
		if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve: %v", err)
		}
	})
	return c
}

func TestClientDefaults(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("timeout = %v; want 5s", c.httpClient.Timeout)
	}
	transport, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", c.httpClient.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("transport must not use a proxy")
	}
	conn, err := transport.DialContext(context.Background(), "tcp", "unused.invalid:80")
	if err != nil {
		t.Fatalf("dial must ignore network and address: %v", err)
	}
	defer conn.Close()
	if conn.RemoteAddr().Network() != "unix" {
		t.Errorf("connection network = %s; want unix", conn.RemoteAddr().Network())
	}
}

func TestClientRoutesAndJSON(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	config := protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 31}
	hello := protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "runner", Mode: "demo", Features: []string{"snapshot"}}
	instance := protocol.Instance{
		ID: "demo-1_A", Label: "Demo", Config: config, ConfigRevision: 2, AppliedRevision: 1,
		DesiredState: "running", ObservedState: "starting", ActiveOperationID: "op-1",
		EpochHistory: []protocol.EpochCreation{{Name: "0i100", RolledAt: now}},
	}
	operation := protocol.Operation{
		ID: "op-1", RequestID: "request-1", InstanceID: instance.ID, Kind: "start", Actor: "local",
		Status: "succeeded", ConfigRevision: 2, CreatedAt: now, FinishedAt: &now,
	}
	snapshot := protocol.Snapshot{
		RunnerID: hello.RunnerID, Mode: hello.Mode, Instances: []protocol.Instance{instance},
		Operations: []protocol.Operation{operation},
		Incidents:  []protocol.Incident{{ID: "incident-1", InstanceID: instance.ID, Code: "demo", Message: "demo only", Active: true, ObservedAt: now}},
	}
	configureReq := protocol.ConfigureRequest{ExpectedRevision: 1, Config: config}
	operateReq := protocol.OperationRequest{RequestID: operation.RequestID, Kind: operation.Kind, ExpectedRevision: 2}
	var calls, operationCalls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/hello" {
			if r.Header.Get(protocol.Header) != "" {
				t.Errorf("hello must not require a version header: %q", r.Header.Get(protocol.Header))
			}
		} else {
			if r.Header.Get(protocol.Header) != strconv.Itoa(protocol.Version) {
				t.Errorf("protocol header = %q", r.Header.Get(protocol.Header))
			}
			w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
		}
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
			}
		}
		var response any
		switch r.Method + " " + r.RequestURI {
		case "GET /hello":
			response = hello
		case "GET /v1/snapshot":
			response = snapshot
		case "PUT /v1/instances/demo-1_A/configuration":
			var got protocol.ConfigureRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode configuration: %v", err)
			}
			if got != configureReq {
				t.Errorf("configure request = %+v; want %+v", got, configureReq)
			}
			response = instance
		case "POST /v1/instances/demo-1_A/operations":
			var got protocol.OperationRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode operation: %v", err)
			}
			if got != operateReq {
				t.Errorf("operation request = %+v; want %+v", got, operateReq)
			}
			if operationCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusAccepted)
			}
			response = operation
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	})
	ctx := context.Background()
	if got, err := c.Hello(ctx); err != nil || !reflect.DeepEqual(got, hello) {
		t.Fatalf("Hello = %+v, %v; want %+v", got, err, hello)
	}
	if got, err := c.Snapshot(ctx); err != nil || !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("Snapshot = %+v, %v; want %+v", got, err, snapshot)
	}
	if got, err := c.Configure(ctx, instance.ID, configureReq); err != nil || !reflect.DeepEqual(got, instance) {
		t.Fatalf("Configure = %+v, %v; want %+v", got, err, instance)
	}
	for i := 0; i < 2; i++ {
		if got, err := c.Operate(ctx, instance.ID, operateReq); err != nil || !reflect.DeepEqual(got, operation) {
			t.Fatalf("Operate = %+v, %v; want %+v", got, err, operation)
		}
	}
	if calls.Load() != 5 {
		t.Errorf("requests = %d; want 5", calls.Load())
	}
}

func TestClientProtocolMismatch(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hello  bool
		header string
		status int
		body   string
	}{
		{"hello_version", true, "", 200, `{"protocol_version":2}`},
		{"hello_missing", true, "", 200, `{}`},
		{"missing_header", false, "", 200, `{}`},
		{"wrong_header", false, "2", 200, `{}`},
		{"invalid_header", false, "01", 200, `{}`},
		{"error_header", false, "2", 409, `{"error":{"code":"conflict","message":"stale"}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.header != "" {
					w.Header().Set(protocol.Header, tt.header)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				io.WriteString(w, tt.body)
			})
			var err error
			if tt.hello {
				_, err = c.Hello(context.Background())
			} else {
				_, err = c.Snapshot(context.Background())
			}
			if err == nil || !strings.Contains(err.Error(), "protocol") {
				t.Fatalf("error = %v; want protocol mismatch", err)
			}
			var remote *RemoteError
			if errors.As(err, &remote) {
				t.Fatal("incompatible response must not be trusted as a remote error")
			}
		})
	}
}

func TestClientResponseValidation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"empty", 200, "", "decode"},
		{"malformed", 200, "{", "decode"},
		{"wrong_type", 200, `{"instances":"bad"}`, "decode"},
		{"extra_value", 200, "{} {}", "decode"},
		{"trailing_junk", 200, "{}x", "decode"},
		{"whitespace", 200, "{} \n\t", ""},
		{"at_limit", 200, "{}" + strings.Repeat(" ", maxResponseBytes-2), ""},
		{"over_limit", 200, "{}" + strings.Repeat(" ", maxResponseBytes-1), "exceeds"},
		{"bad_error", 500, "{", "decode"},
		{"extra_error", 500, `{"error":{"code":"oops"}} {}`, "decode"},
		{"large_error", 500, "{}" + strings.Repeat(" ", maxResponseBytes-1), "exceeds"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				io.WriteString(w, tt.body)
			})
			_, err := c.Snapshot(context.Background())
			if tt.want == "" {
				if err != nil {
					t.Fatalf("valid response: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v; want %q", err, tt.want)
			}
		})
	}
}

func TestClientRemoteErrors(t *testing.T) {
	revision := uint64(7)
	problem := protocol.APIError{Code: "revision_conflict", Message: "configuration has changed", CurrentRevision: &revision}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: problem})
	})
	ctx := context.Background()
	_, helloErr := c.Hello(ctx)
	_, snapshotErr := c.Snapshot(ctx)
	_, configureErr := c.Configure(ctx, "demo", protocol.ConfigureRequest{})
	_, operateErr := c.Operate(ctx, "demo", protocol.OperationRequest{})
	for _, err := range []error{helloErr, snapshotErr, configureErr, operateErr} {
		var remote *RemoteError
		if !errors.As(err, &remote) {
			t.Fatalf("error = %v; want *RemoteError", err)
		}
		if remote.Status != http.StatusConflict || !reflect.DeepEqual(remote.Problem, problem) {
			t.Errorf("remote error = %+v; want status 409 and %+v", remote, problem)
		}
		for _, text := range []string{"409", problem.Code, problem.Message} {
			if !strings.Contains(err.Error(), text) {
				t.Errorf("error %q does not include %q", err, text)
			}
		}
		if strings.Contains(err.Error(), "uncertain") {
			t.Errorf("structured rejection is not uncertain: %v", err)
		}
	}
}

func TestClientNoRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path == "/redirected" {
					t.Error("client followed a redirect")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Location", "http://unused.invalid/redirected")
				w.WriteHeader(status)
				io.WriteString(w, `{"error":{"code":"redirect","message":"do not follow"}}`)
			})
			_, getErr := c.Snapshot(context.Background())
			_, putErr := c.Configure(context.Background(), "demo", protocol.ConfigureRequest{})
			_, postErr := c.Operate(context.Background(), "demo", protocol.OperationRequest{})
			for _, err := range []error{getErr, putErr, postErr} {
				var remote *RemoteError
				if !errors.As(err, &remote) || remote.Status != status {
					t.Errorf("error = %v; want remote HTTP %d", err, status)
				}
			}
			if calls.Load() != 3 {
				t.Errorf("requests = %d; want 3 without redirects", calls.Load())
			}
		})
	}
}

func TestClientInstanceIDs(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{}`)
	})
	for _, id := range []string{
		"", strings.Repeat("a", 65), ".", "..", "../demo", "demo/other", "demo\\other",
		"%2e%2e", "demo%2fother", "demo%252fother", "demo?query", "demo#fragment", "demo space",
		"demo\x00", "demo\n", "demo\x7f", "demo\xff", "caf\u00e9", "\uff41", "demo+other", "demo:other",
	} {
		_, configureErr := c.Configure(context.Background(), id, protocol.ConfigureRequest{})
		_, operateErr := c.Operate(context.Background(), id, protocol.OperationRequest{})
		for _, err := range []error{configureErr, operateErr} {
			if err == nil || !strings.Contains(err.Error(), "invalid instance ID") || strings.Contains(err.Error(), "uncertain") {
				t.Errorf("ID %q: error = %v; want local validation error", id, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid IDs sent %d requests", calls.Load())
	}
	valid := []string{"a", "Z", "0", "_", "-", "aZ09_-", strings.Repeat("a", 64)}
	for _, id := range valid {
		if _, err := c.Configure(context.Background(), id, protocol.ConfigureRequest{}); err != nil {
			t.Errorf("valid ID %q: %v", id, err)
		}
		if _, err := c.Operate(context.Background(), id, protocol.OperationRequest{}); err != nil {
			t.Errorf("valid ID %q: %v", id, err)
		}
	}
	if calls.Load() != int32(2*len(valid)) {
		t.Errorf("valid requests = %d; want %d", calls.Load(), 2*len(valid))
	}
}

func TestClientTimeouts(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPost} {
		for _, phase := range []string{"headers", "body"} {
			t.Run(method+"_"+phase, func(t *testing.T) {
				var calls atomic.Int32
				c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					io.Copy(io.Discard, r.Body)
					if phase == "body" {
						w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, "{")
						w.(http.Flusher).Flush()
					}
					<-r.Context().Done()
				})
				c.httpClient.Timeout = 100 * time.Millisecond
				var err error
				switch method {
				case http.MethodGet:
					_, err = c.Snapshot(context.Background())
				case http.MethodPut:
					_, err = c.Configure(context.Background(), "demo", protocol.ConfigureRequest{})
				case http.MethodPost:
					_, err = c.Operate(context.Background(), "demo", protocol.OperationRequest{})
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("error = %v; want deadline exceeded", err)
				}
				if got := strings.Contains(err.Error(), "uncertain"); got != (method != http.MethodGet) {
					t.Errorf("uncertainty for %s: %v", method, err)
				}
				var uncertain *OutcomeUncertainError
				if errors.As(err, &uncertain) != (method != http.MethodGet) {
					t.Errorf("typed uncertainty for %s: %v", method, err)
				}
				if calls.Load() != 1 {
					t.Errorf("requests = %d; want 1 without retries", calls.Load())
				}
			})
		}
	}
}

// Fail writes only after a successful request has primed a reusable connection.
type failingWriteConn struct {
	net.Conn
	fail *atomic.Bool
}

func (c *failingWriteConn) Write(data []byte) (int, error) {
	if c.fail.Load() {
		return 0, io.ErrClosedPipe
	}
	return c.Conn.Write(data)
}

func TestClientNoMutationReplay(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			var calls, dials atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"protocol_version":1}`)
			})
			transport := c.httpClient.Transport.(*http.Transport)
			dial := transport.DialContext
			var fail atomic.Bool
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				dials.Add(1)
				conn, err := dial(ctx, network, address)
				if err != nil {
					return nil, err
				}
				return &failingWriteConn{Conn: conn, fail: &fail}, nil
			}
			if _, err := c.Hello(context.Background()); err != nil {
				t.Fatal(err)
			}
			fail.Store(true)
			var err error
			if method == http.MethodPut {
				_, err = c.Configure(context.Background(), "demo", protocol.ConfigureRequest{})
			} else {
				_, err = c.Operate(context.Background(), "demo", protocol.OperationRequest{})
			}
			if !errors.Is(err, io.ErrClosedPipe) || !strings.Contains(err.Error(), "uncertain") {
				t.Fatalf("error = %v; want uncertain write failure", err)
			}
			if dials.Load() != 1 || calls.Load() != 1 {
				t.Errorf("dials = %d, requests = %d; mutation must not be retried", dials.Load(), calls.Load())
			}
		})
	}
}

func TestClientLocalTransportErrors(t *testing.T) {
	c := New(filepath.Join(t.TempDir(), "missing.sock"))
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Snapshot(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled request: %v", err)
	}
	_, readErr := c.Hello(context.Background())
	_, putErr := c.Configure(context.Background(), "demo", protocol.ConfigureRequest{})
	_, postErr := c.Operate(context.Background(), "demo", protocol.OperationRequest{})
	for i, err := range []error{readErr, putErr, postErr} {
		var remote *RemoteError
		var op *net.OpError
		if errors.As(err, &remote) || !errors.As(err, &op) {
			t.Errorf("error = %v; want local network error", err)
		}
		if err == nil || strings.Contains(err.Error(), "uncertain") != (i > 0) {
			t.Errorf("unexpected transport uncertainty: %v", err)
		}
	}
}
