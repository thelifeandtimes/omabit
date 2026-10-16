package collection

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func logServer(t *testing.T, handler http.HandlerFunc) (Connection, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "logs.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var opened, active atomic.Int32
	srv := &http.Server{Handler: handler, ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
			active.Add(1)
		} else if state == http.StateClosed || state == http.StateHijacked {
			active.Add(-1)
		}
	}}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return Connection{Name: "test", RunnerID: "runner-logs", Mode: "development-managed", Transport: "unix", Socket: socket}, &opened, &active
}

func logHello(w http.ResponseWriter, r *http.Request, id string) bool {
	w.Header().Set(protocol.Header, "1")
	if r.URL.Path != "/hello" {
		return false
	}
	json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "development-managed", AccessRole: "operator"})
	return true
}

func logTail() protocol.RuntimeLogs {
	return protocol.RuntimeLogs{RunnerID: "runner-logs", InstanceID: "alpha", ConfigRevision: 2, Source: "docker", ContainerID: strings.Repeat("a", 64), ObservedAt: time.Now().UTC(), Lines: []string{"boot: redacted fixture"}}
}

func awaitLogClose(t *testing.T, active *atomic.Int32) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for active.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if active.Load() != 0 {
		t.Fatal("watch left a live connection")
	}
}

func TestLogWatchReusesConnectionAndReplacesWholeTail(t *testing.T) {
	t.Parallel()
	var polls atomic.Int32
	entry, opened, active := logServer(t, func(w http.ResponseWriter, r *http.Request) {
		if logHello(w, r, "runner-logs") {
			return
		}
		var request protocol.LogsRequest
		if r.Method != "POST" || r.URL.Path != "/v1/instances/alpha/logs" || json.NewDecoder(r.Body).Decode(&request) != nil || request.ExpectedRevision != 2 {
			t.Error("lost pinned log selection")
		}
		tail := logTail()
		switch polls.Add(1) {
		case 2:
			tail.Source = "retained"
		case 3:
			tail.ContainerID = strings.Repeat("b", 64)
			tail.Lines = []string{"new runtime"}
		case 4:
			tail.Source, tail.ContainerID, tail.Lines = "none", "", []string{}
		}
		json.NewEncoder(w).Encode(tail)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	count := 0
	err := WatchLogs(ctx, entry, "alpha", 2, time.Second, func(update LogUpdate) error {
		count++
		if update.Logs == nil || update.Error != nil || update.RetryAfterMS != 0 || opened.Load() != 1 {
			t.Fatalf("successful polls did not reuse one transport: %+v connections=%d", update, opened.Load())
		}
		if count == 3 && update.Logs.Lines[0] != "new runtime" || count == 4 && len(update.Logs.Lines) != 0 {
			t.Error("tail retained old runtime data")
		}
		if count == 4 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || count != 4 {
		t.Fatal(count, err)
	}
	awaitLogClose(t, active)
}

func TestLogWatchStopsOnPermanentErrorsWithoutRetry(t *testing.T) {
	for _, scenario := range []string{"identity", "handshake-denied", "revoked", "revision", "deleted", "selection", "oversized"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			var polls atomic.Int32
			entry, opened, active := logServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if r.URL.Path == "/hello" && scenario == "handshake-denied" {
					w.WriteHeader(403)
					io.WriteString(w, "PRIVATE remote diagnostic")
					return
				}
				id := "runner-logs"
				if scenario == "identity" {
					id = "runner-replacement"
				}
				if logHello(w, r, id) {
					return
				}
				polls.Add(1)
				tail := logTail()
				switch scenario {
				case "revoked", "revision", "deleted":
					w.WriteHeader(map[string]int{"revoked": 403, "revision": 409, "deleted": 404}[scenario])
					json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "revision_conflict", Message: "PRIVATE remote diagnostic"}})
					return
				case "selection":
					tail.InstanceID = "other"
				case "oversized":
					tail.Lines = []string{strings.Repeat("x", protocol.MaxLogBytes)}
				}
				json.NewEncoder(w).Encode(tail)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			updates := 0
			err := WatchLogs(ctx, entry, "alpha", 2, time.Second, func(update LogUpdate) error {
				updates++
				if update.Error == nil || update.Logs != nil || update.RetryAfterMS != 0 || strings.Contains(update.Error.Message, "PRIVATE") {
					t.Fatalf("unsafe terminal error: %+v", update)
				}
				if scenario == "identity" && update.Error.Code != "runner_identity_mismatch" || (scenario == "revoked" || scenario == "handshake-denied") && update.Error.Code != "logs_access_denied" || scenario == "revision" && update.Error.Code != "revision_conflict" {
					t.Error("failure not distinguished", update.Error)
				}
				return nil
			})
			if err == nil || ctx.Err() != nil || updates != 1 || opened.Load() != 1 || polls.Load() > 1 {
				t.Fatal("terminal error retried", err, updates, opened.Load(), polls.Load())
			}
			awaitLogClose(t, active)
		})
	}
}

func TestLogWatchBackoffReconnectRechecksIdentity(t *testing.T) {
	t.Parallel()
	var replacement atomic.Bool
	var polls atomic.Int32
	entry, opened, active := logServer(t, func(w http.ResponseWriter, r *http.Request) {
		id := "runner-logs"
		if replacement.Load() {
			id = "runner-other"
		}
		if logHello(w, r, id) {
			return
		}
		polls.Add(1)
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var first time.Time
	var delay int64
	updates := 0
	err := WatchLogs(ctx, entry, "alpha", 2, time.Second, func(update LogUpdate) error {
		updates++
		if updates == 1 {
			first, delay = time.Now(), update.RetryAfterMS
			if update.Error == nil || update.Error.Code != "connection_unavailable" || delay < 3000 || delay >= 6000 || update.Logs != nil {
				t.Fatalf("missing bounded initial backoff: %+v", update)
			}
			replacement.Store(true)
		} else if time.Since(first) < time.Duration(delay)*time.Millisecond || update.Error == nil || update.Error.Code != "runner_identity_mismatch" || update.RetryAfterMS != 0 {
			t.Fatalf("reconnect skipped delay or pin: %+v", update)
		}
		return nil
	})
	if err == nil || ctx.Err() != nil || updates != 2 || opened.Load() != 2 || polls.Load() != 1 {
		t.Fatal("replacement received logs or watch retried identity failure", err, updates, opened.Load(), polls.Load())
	}
	awaitLogClose(t, active)
}

func TestLogWatchCancellationDuringHandshakeRequestAndBackoff(t *testing.T) {
	for _, phase := range []string{"handshake", "request", "backoff", "output"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			entry, _, active := logServer(t, func(w http.ResponseWriter, r *http.Request) {
				if phase == "handshake" || phase == "request" && r.URL.Path != "/hello" {
					io.Copy(io.Discard, r.Body)
					cancel()
					<-r.Context().Done()
					return
				}
				if logHello(w, r, "runner-logs") {
					return
				}
				if phase == "backoff" {
					w.WriteHeader(503)
					json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "unavailable"}})
					return
				}
				json.NewEncoder(w).Encode(logTail())
			})
			updates := 0
			err := WatchLogs(ctx, entry, "alpha", 2, time.Second, func(update LogUpdate) error {
				updates++
				if phase == "output" {
					return io.ErrClosedPipe
				}
				cancel()
				return nil
			})
			if phase == "output" {
				if !errors.Is(err, io.ErrClosedPipe) || updates != 1 {
					t.Fatal(err, updates)
				}
			} else if !errors.Is(err, context.Canceled) || phase != "backoff" && updates != 0 {
				t.Fatal(err, updates)
			}
			awaitLogClose(t, active)
		})
	}
}

func TestLogProblemRetriesOnlyTransientFailures(t *testing.T) {
	for _, test := range []struct {
		err              error
		retry, reconnect bool
	}{
		{&client.RemoteError{Status: 502}, true, false},
		{&client.RemoteError{Status: 409, Problem: protocol.APIError{Code: "runtime_changed"}}, true, false},
		{&client.RemoteError{Status: 501}, false, true},
		{io.EOF, true, true},
		{errors.New("malformed response"), false, true},
	} {
		_, retry, reconnect := logProblem(test.err)
		if retry != test.retry || reconnect != test.reconnect {
			t.Fatal(test.err, retry, reconnect)
		}
	}
}
