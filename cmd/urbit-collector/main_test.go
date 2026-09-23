package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func fakeRunner(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("unexpected credentials in request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session=private-cookie")
		if r.URL.Path != "/hello" {
			if r.Header.Get(protocol.Header) != strconv.Itoa(protocol.Version) {
				t.Errorf("missing protocol header: %v", r.Header)
			}
			w.Header().Set(protocol.Header, strconv.Itoa(protocol.Version))
		}
		handler(w, r)
	})}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
		if err := <-served; !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve: %v", err)
		}
	})
	return socket
}

func reply(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func errorProblem(t *testing.T, output string, code string) protocol.APIError {
	t.Helper()
	var response protocol.ErrorResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("stderr must be a single JSON ErrorResponse: %q: %v", output, err)
	}
	if response.Error.Code != code || response.Error.Message == "" {
		t.Fatalf("problem = %+v; want code %s and a message", response.Error, code)
	}
	return response.Error
}

func TestCommands(t *testing.T) {
	hello := protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "simulator", Mode: "demo", Features: []string{"snapshot"}}
	config := protocol.Configuration{HTTPPort: 8080, AmesPort: 34343, Loom: 32}
	instance := protocol.Instance{ID: "alpha", Label: "Alpha", Config: config, ConfigRevision: 2, AppliedRevision: 1, DesiredState: "stopped", ObservedState: "stopped"}
	snapshot := protocol.Snapshot{RunnerID: hello.RunnerID, Mode: hello.Mode, Instances: []protocol.Instance{instance}, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{}}
	operation := protocol.Operation{ID: "op-1", RequestID: "walkthrough-start", InstanceID: "alpha", Kind: "start", Actor: "local", Status: "pending", ConfigRevision: 2, CreatedAt: time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)}
	stop := operation
	stop.Kind, stop.RequestID = "stop", "Stop_09-"
	for _, tt := range []struct {
		name     string
		args     []string
		route    string
		payload  any
		response any
	}{
		{"hello", []string{"hello"}, "GET /hello", nil, hello},
		{"snapshot", []string{"snapshot"}, "GET /v1/snapshot", nil, snapshot},
		{"configure", []string{"configure", "alpha", "--revision", "1", "--http-port", "8080", "--ames-port", "34343", "--loom", "32"}, "PUT /v1/instances/alpha/configuration", protocol.ConfigureRequest{ExpectedRevision: 1, Config: config}, instance},
		{"start", []string{"operate", "alpha", "start", "--revision", "2", "--request-id", "walkthrough-start"}, "POST /v1/instances/alpha/operations", protocol.OperationRequest{ExpectedRevision: 2, Kind: "start", RequestID: "walkthrough-start"}, operation},
		{"stop", []string{"operate", "alpha", "stop", "--revision", "2", "--request-id", "Stop_09-"}, "POST /v1/instances/alpha/operations", protocol.OperationRequest{ExpectedRevision: 2, Kind: "stop", RequestID: "Stop_09-"}, stop},
		{"lower_bounds", []string{"configure", "alpha", "--revision=1", "--http-port=1", "--ames-port=1", "--loom=31"}, "PUT /v1/instances/alpha/configuration", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 1, AmesPort: 1, Loom: 31}}, instance},
		{"upper_bounds", []string{"configure", "alpha", "--revision=18446744073709551615", "--http-port=65535", "--ames-port=65535", "--loom=33"}, "PUT /v1/instances/alpha/configuration", protocol.ConfigureRequest{ExpectedRevision: ^uint64(0), Config: protocol.Configuration{HTTPPort: 65535, AmesPort: 65535, Loom: 33}}, instance},
		{"long_id", []string{"operate", "alpha", "start", "--revision=2", "--request-id=" + strings.Repeat("a", 64)}, "POST /v1/instances/alpha/operations", protocol.OperationRequest{ExpectedRevision: 2, Kind: "start", RequestID: strings.Repeat("a", 64)}, operation},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if r.URL.Path == "/hello" {
					if n != 1 || r.Method != http.MethodGet {
						t.Errorf("Hello must be the first and only handshake: call %d, method %s", n, r.Method)
					}
					if tt.name != "hello" {
						reply(t, w, hello)
						return
					}
				} else if n != 2 {
					t.Errorf("operation must immediately follow Hello: call %d", n)
				}
				if got := r.Method + " " + r.RequestURI; got != tt.route {
					t.Errorf("route = %s; want %s", got, tt.route)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				var wantBody []byte
				if tt.payload != nil {
					wantBody, err = json.Marshal(tt.payload)
					if err != nil {
						t.Error(err)
					}
					if r.Header.Get("Content-Type") != "application/json" {
						t.Error("mutation must send JSON")
					}
				}
				if !bytes.Equal(body, wantBody) {
					t.Errorf("payload = %s; want %s", body, wantBody)
				}
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusAccepted)
				}
				reply(t, w, tt.response)
			})
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), append([]string{"--socket", socket}, tt.args...), &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 {
				t.Fatalf("exit = %d, stderr = %s", code, &stderr)
			}
			want, err := json.MarshalIndent(tt.response, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != string(want)+"\n" {
				t.Errorf("stdout = %s; want a single indented result: %s", &stdout, want)
			}
			wantCalls := int32(2)
			if tt.name == "hello" {
				wantCalls = 1
			}
			if calls.Load() != wantCalls {
				t.Errorf("requests = %d; want %d, without revision fetches or retries", calls.Load(), wantCalls)
			}
		})
	}
}

func TestRequestFailures(t *testing.T) {
	revision := uint64(7)
	problem := protocol.APIError{Code: "revision_conflict", Message: "configuration changed", CurrentRevision: &revision}
	for _, mode := range []string{"conflict", "hello_mismatch", "header_mismatch"} {
		for _, args := range [][]string{
			{"hello"}, {"snapshot"},
			{"configure", "alpha", "--revision=1", "--http-port=8080", "--ames-port=34343", "--loom=32"},
			{"operate", "alpha", "start", "--revision=2", "--request-id=walkthrough-start"},
			{"watch", "--interval=100ms"},
		} {
			if mode == "header_mismatch" && args[0] == "hello" {
				continue
			}
			t.Run(mode+"/"+args[0], func(t *testing.T) {
				var calls atomic.Int32
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Path == "/hello" {
						if mode == "hello_mismatch" {
							reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version + 1})
							return
						}
						if args[0] != "hello" {
							reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version})
							return
						}
					}
					if mode == "header_mismatch" {
						w.Header().Set(protocol.Header, "999")
					}
					w.WriteHeader(http.StatusConflict)
					reply(t, w, protocol.ErrorResponse{Error: problem})
				})
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var stdout, stderr bytes.Buffer
				code := run(ctx, append([]string{"--socket", socket}, args...), &stdout, &stderr)
				if code != 1 || stdout.Len() != 0 {
					t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, &stdout, &stderr)
				}
				if mode == "conflict" {
					if got := errorProblem(t, stderr.String(), problem.Code); !reflect.DeepEqual(got, problem) {
						t.Errorf("problem = %+v; want %+v", got, problem)
					}
				} else {
					got := errorProblem(t, stderr.String(), "client_error")
					if !strings.Contains(got.Message, "incompatible runner protocol") || got.CurrentRevision != nil {
						t.Errorf("untrusted protocol response: %+v", got)
					}
				}
				wantCalls := int32(2)
				if mode == "hello_mismatch" || args[0] == "hello" {
					wantCalls = 1
				}
				if calls.Load() != wantCalls {
					t.Errorf("requests = %d; want %d, without follow-up requests or retries", calls.Load(), wantCalls)
				}
			})
		}
	}
}

func TestUsageErrors(t *testing.T) {
	var calls atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version})
	})
	tests := []struct {
		name string
		args []string
	}{
		{"no_command", nil},
		{"unknown_command", []string{"daemon"}},
		{"unknown_global", []string{"--unknown", "hello"}},
		{"hello_extra", []string{"hello", "extra"}},
		{"snapshot_extra", []string{"snapshot", "extra"}},
		{"watch_extra", []string{"watch", "--interval=100ms", "extra"}},
		{"late_global", []string{"snapshot", "--socket", socket}},
		{"hello_flag", []string{"hello", "--unknown"}},
		{"snapshot_flag", []string{"snapshot", "--revision=1"}},
		{"watch_flag", []string{"watch", "--unknown"}},
		{"configure_empty", []string{"configure"}},
		{"configure_no_flags", []string{"configure", "alpha"}},
		{"operate_empty", []string{"operate"}},
		{"operate_no_kind", []string{"operate", "alpha"}},
		{"operate_no_flags", []string{"operate", "alpha", "start"}},
		{"operate_no_revision", []string{"operate", "alpha", "start", "--request-id=r"}},
		{"operate_no_request_id", []string{"operate", "alpha", "start", "--revision=1"}},
		{"operate_zero_revision", []string{"operate", "alpha", "start", "--revision=0", "--request-id=r"}},
		{"operate_bad_kind", []string{"operate", "alpha", "restart", "--revision=1", "--request-id=r"}},
		{"operate_extra", []string{"operate", "alpha", "start", "--revision=1", "--request-id=r", "extra"}},
		{"operate_flag", []string{"operate", "alpha", "start", "--revision=1", "--request-id=r", "--unknown"}},
	}
	configuration := []string{"configure", "alpha", "--revision=1", "--http-port=8080", "--ames-port=34343", "--loom=32"}
	for i, name := range []string{"revision", "http-port", "ames-port", "loom"} {
		index := i + 2
		missing := append([]string{}, configuration[:index]...)
		missing = append(missing, configuration[index+1:]...)
		tests = append(tests, struct {
			name string
			args []string
		}{"missing_" + name, missing})
		values := []string{"0", "-1", "abc", "1.5", "18446744073709551616"}
		if name == "http-port" || name == "ames-port" {
			values = append(values, "65536")
		}
		if name == "loom" {
			values = append(values, "30", "34", "287")
		}
		for _, value := range values {
			args := append([]string{}, configuration...)
			args[index] = "--" + name + "=" + value
			tests = append(tests, struct {
				name string
				args []string
			}{name + "_" + value, args})
		}
	}
	for _, extra := range []string{"extra", "--unknown", "--loom"} {
		tests = append(tests, struct {
			name string
			args []string
		}{"configure_" + extra, append(append([]string{}, configuration...), extra)})
	}
	for _, id := range []string{"", "bad/id", "bad id", "a.b", "a%2fb", "a?b", "a#b", "a\nb", "a\x00b", "caf\u00e9", strings.Repeat("a", 65)} {
		for _, field := range []string{"instance", "request"} {
			args := []string{"operate", "alpha", "start", "--revision=1", "--request-id=r"}
			if field == "instance" {
				args[1] = id
			} else {
				args[4] = "--request-id=" + id
			}
			tests = append(tests, struct {
				name string
				args []string
			}{field + "_id_" + id, args})
		}
	}
	for _, interval := range []string{"0", "-1s", "99ms", "99999999ns", "abc", "1", "999999999999999999999s"} {
		tests = append(tests, struct {
			name string
			args []string
		}{"interval_" + interval, []string{"watch", "--interval=" + interval}})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), append([]string{"--socket", socket}, tt.args...), &stdout, &stderr)
			if code != 2 || stdout.Len() != 0 {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, &stdout, &stderr)
			}
			errorProblem(t, stderr.String(), "client_error")
		})
	}
	if calls.Load() != 0 {
		t.Errorf("invalid arguments made %d requests", calls.Load())
	}
}

func TestSocketAndLocalFailure(t *testing.T) {
	for _, args := range [][]string{nil, {"hello"}, {"--socket", "", "hello"}, {"--socket", "relative.sock", "hello"}, {"--socket"}, {"hello", "--socket", "/unused.sock"}} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, &stdout, &stderr); code != 2 || stdout.Len() != 0 {
			t.Fatalf("args = %q: exit = %d, stdout = %q, stderr = %q", args, code, &stdout, &stderr)
		}
		errorProblem(t, stderr.String(), "client_error")
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--socket", filepath.Join(t.TempDir(), "missing.sock"), "hello"}, &stdout, &stderr)
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, &stdout, &stderr)
	}
	errorProblem(t, stderr.String(), "client_error")
}

func TestHelp(t *testing.T) {
	for _, args := range [][]string{
		{"--help"}, {"-h"}, {"hello", "--help"}, {"snapshot", "--help"},
		{"configure", "--help"}, {"configure", "alpha", "--help"},
		{"operate", "--help"}, {"operate", "alpha", "--help"}, {"operate", "alpha", "start", "--help"},
		{"watch", "--help"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(context.Background(), args, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("exit = %d, stderr = %q", code, &stderr)
			}
			for _, text := range []string{"Usage:", "simulator checkpoint", "not a full collector daemon", "absolute path", "before the subcommand"} {
				if !strings.Contains(stdout.String(), text) {
					t.Errorf("help must include %q: %s", text, &stdout)
				}
			}
			if args[0] == "configure" && (!strings.Contains(stdout.String(), "31..33") || !strings.Contains(stdout.String(), "not universal runtime bounds")) {
				t.Errorf("help must explain demo loom bounds: %s", &stdout)
			}
		})
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestWatch(t *testing.T) {
	for _, stop := range []string{"cancel", "remote_error", "output_error"} {
		t.Run(stop, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var hellos, snapshots atomic.Int32
			snapshot := protocol.Snapshot{RunnerID: "simulator", Mode: "demo", Instances: []protocol.Instance{}, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{}}
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.RequestURI {
				case "GET /hello":
					hellos.Add(1)
					reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version})
				case "GET /v1/snapshot":
					if snapshots.Add(1) == 2 && stop == "remote_error" {
						w.WriteHeader(http.StatusServiceUnavailable)
						reply(t, w, protocol.ErrorResponse{Error: protocol.APIError{Code: "unavailable", Message: "simulator unavailable"}})
						return
					}
					reply(t, w, snapshot)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.RequestURI)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			var stdout, stderr bytes.Buffer
			writes := 0
			output := writerFunc(func(p []byte) (int, error) {
				if stop == "output_error" {
					return 0, io.ErrClosedPipe
				}
				n, err := stdout.Write(p)
				writes++
				if writes == 2 && stop == "cancel" {
					cancel()
				}
				return n, err
			})
			code := run(ctx, []string{"--socket", socket, "watch", "--interval=100ms"}, output, &stderr)
			wantLines, wantCalls := 2, int32(2)
			switch stop {
			case "cancel":
				if code != 0 || stderr.Len() != 0 {
					t.Fatalf("exit = %d, stderr = %q", code, &stderr)
				}
			case "remote_error":
				wantLines = 1
				if code != 1 {
					t.Fatalf("exit = %d; want 1", code)
				}
				errorProblem(t, stderr.String(), "unavailable")
			case "output_error":
				wantLines, wantCalls = 0, 1
				if code != 1 {
					t.Fatalf("exit = %d; want 1", code)
				}
				errorProblem(t, stderr.String(), "client_error")
			}
			line, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != strings.Repeat(string(line)+"\n", wantLines) {
				t.Errorf("stdout must be %d compact NDJSON snapshots: %q", wantLines, &stdout)
			}
			if hellos.Load() != 1 || snapshots.Load() != wantCalls {
				t.Errorf("requests: hello = %d, snapshot = %d; want 1, %d", hellos.Load(), snapshots.Load(), wantCalls)
			}
		})
	}
}

func TestWatchCanceledRequest(t *testing.T) {
	for _, phase := range []string{"before_hello", "/hello", "/v1/snapshot"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var calls atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path == phase {
					cancel()
					<-r.Context().Done()
					return
				}
				reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version})
			})
			if phase == "before_hello" {
				cancel()
			}
			var stdout, stderr bytes.Buffer
			if code := run(ctx, []string{"--socket", socket, "watch"}, &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, &stdout, &stderr)
			}
			want := map[string]int32{"before_hello": 0, "/hello": 1, "/v1/snapshot": 2}[phase]
			if calls.Load() != want {
				t.Errorf("requests = %d; want %d", calls.Load(), want)
			}
		})
	}
}

func TestNoSecretOutput(t *testing.T) {
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"protocol_version":1,"runner_id":"simulator","cookie":"private-cookie","secret":"private-secret"}`)
	})
	for _, command := range []string{"hello", "snapshot"} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), []string{"--socket", socket, command}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
			t.Fatalf("exit = %d, stderr = %q", code, &stderr)
		}
		if strings.Contains(stdout.String(), "private-") || strings.Contains(stdout.String(), "cookie") || strings.Contains(stdout.String(), "secret") {
			t.Errorf("response leaked credentials: %q", &stdout)
		}
	}
}
