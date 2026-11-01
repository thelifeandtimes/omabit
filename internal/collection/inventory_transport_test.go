package collection

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// The current inventory requests snapshot, projects and host evidence together.
// The older transport fixtures advertised at most one optional report and could
// not detect a three-stream workload evicting the default two idle connections.
func TestWatchThreeReportsKeepWarmTransports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	var opened, closed, requests atomic.Int32
	var mu sync.Mutex
	arrived := 0
	ready := make(chan struct{})
	features := []string{protocol.HostCompatibilityFeature, protocol.ProjectInventoryFeature}
	report := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
	srv := &http.Server{ConnState: func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
		if state == http.StateClosed {
			closed.Add(1)
		}
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: features})
			return
		}
		// Make all reports overlap deterministically, as real SSH latency can do.
		mu.Lock()
		gate := ready
		arrived++
		if arrived == 3 {
			close(gate)
			ready = make(chan struct{})
			arrived = 0
		}
		mu.Unlock()
		select {
		case <-gate:
		case <-r.Context().Done():
			return
		}
		switch r.URL.Path {
		case "/v1/snapshot":
			json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner", Mode: "simulation"})
		case protocol.ProjectInventoryPath:
			json.NewEncoder(w).Encode(protocol.ProjectInventory{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: time.Now().UTC(), Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{}})
		case protocol.HostCompatibilityPath:
			json.NewEncoder(w).Encode(protocol.HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.HostCompatibilityFeature}, Report: report, ExpiresAt: report.ObservedAt.Add(protocol.HostCompatibilityMaxAge)})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	polls := 0
	err = Watch(ctx, []Connection{{Name: "one", RunnerID: "runner", Mode: "simulation", Transport: "unix", Socket: path}}, 100*time.Millisecond, func(v Observation) error {
		polls++
		if v.Status != "available" || v.ProjectInventoryStatus != "available" || v.HostCompatibilityStatus != "available" {
			t.Errorf("lost independent report: %+v", v)
		}
		if polls == 10 {
			cancel()
		}
		return nil
	})
	t.Logf("polls=%d connections=%d requests=%d", polls, opened.Load(), requests.Load())
	if !errors.Is(err, context.Canceled) || polls != 10 {
		t.Fatal(err, polls)
	}
	if requests.Load() > 43 {
		t.Errorf("ten observations repeated capability RPCs: got %d requests, budget 43", requests.Load())
	}
	if opened.Load() != 3 {
		t.Errorf("warm three-stream watch reopened transports: got %d, want 3", opened.Load())
	}
	deadline := time.Now().Add(time.Second)
	for closed.Load() != opened.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if closed.Load() != opened.Load() {
		t.Fatal("cancellation retained transports", opened.Load(), closed.Load())
	}
}
