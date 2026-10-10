package collection

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestWatchReusesTransportRefreshesRoleAndRejectsReplacement(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var connections, active, snapshots atomic.Int32
	var observer, replacement atomic.Bool
	server := &http.Server{ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
			active.Add(1)
		}
		if state == http.StateClosed {
			active.Add(-1)
		}
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			id, role := "runner-original", "operator"
			if observer.Load() {
				role = "observer"
			}
			if replacement.Load() {
				id = "runner-replacement"
			}
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "simulation", AccessRole: role})
			return
		}
		if snapshots.Add(1) == 3 {
			w.Header().Set("Connection", "close")
		}
		json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner-original", Mode: "simulation"})
	})}
	go server.Serve(listener)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates := 0
	err = Watch(ctx, []Connection{{Name: "test", Transport: "unix", Socket: socket, RunnerID: "runner-original", Mode: "simulation"}}, 100*time.Millisecond, func(o Observation) error {
		updates++
		if updates <= 3 {
			if o.Status != "available" || connections.Load() != 1 {
				t.Errorf("healthy polls opened new connections: %d %+v", connections.Load(), o)
			}
			if updates == 1 {
				observer.Store(true)
			} else if o.AccessRole != "observer" {
				t.Error("kept stale role on reused transport")
			}
			if updates == 3 {
				replacement.Store(true)
			}
		} else {
			if o.Status != "identity_mismatch" || o.Snapshot != nil || connections.Load() != 2 {
				t.Errorf("replacement accepted: %+v, connections %d", o, connections.Load())
			}
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || updates != 4 {
		t.Fatal(updates, err)
	}
	deadline := time.Now().Add(time.Second)
	for active.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if active.Load() != 0 {
		t.Fatal("watch cancellation left a live transport")
	}
}

func TestWatchCancellationClosesHealthyPersistentTransport(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var opened, closed atomic.Int32
	server := &http.Server{ConnState: func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			opened.Add(1)
		}
		if s == http.StateClosed {
			closed.Add(1)
		}
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Mode: "simulation"})
			return
		}
		json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner-test", Mode: "simulation"})
	})}
	go server.Serve(listener)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	polls := 0
	err = Watch(ctx, []Connection{{Name: "test", Transport: "unix", Socket: socket, RunnerID: "runner-test", Mode: "simulation"}}, 100*time.Millisecond, func(o Observation) error {
		if o.Status != "available" {
			t.Error(o)
		}
		polls++
		if polls == 5 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || polls != 5 || opened.Load() != 1 {
		t.Fatal("transport not reused", polls, opened.Load(), err)
	}
	deadline := time.Now().Add(time.Second)
	for closed.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if closed.Load() != 1 {
		t.Fatal("SSH-equivalent connection not closed on cancellation")
	}
}
