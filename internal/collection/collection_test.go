package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func fixture(t *testing.T, id string, handler http.HandlerFunc) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "r.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "simulation"})
			return
		}
		if handler != nil {
			handler(w, r)
			return
		}
		json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: id, Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha"}}})
	})}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	t.Cleanup(func() {
		srv.Close()
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	})
	return socket
}

func registry(t *testing.T) *Registry {
	t.Helper()
	r, err := Open(filepath.Join(t.TempDir(), "collector"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestEnrollmentPersistenceAndConflicts(t *testing.T) {
	ctx := context.Background()
	r := registry(t)
	socket := fixture(t, "runner-1", nil)
	entry, err := r.Enroll(ctx, "workstation", socket)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Open(r.dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r2.Enroll(ctx, "workstation", socket)
	if err != nil || got != entry {
		t.Fatalf("exact re-enrollment must preserve pin and timestamp: %+v %v", got, err)
	}
	otherSocket := fixture(t, "runner-2", nil)
	for _, pair := range [][2]string{{"workstation", otherSocket}, {"alias", socket}} {
		if _, err := r2.Enroll(ctx, pair[0], pair[1]); err == nil {
			t.Fatal("conflicting enrollment accepted")
		}
	}
	if err := r2.Forget(ctx, "workstation", "wrong-runner"); !errors.Is(err, client.ErrIdentityMismatch) {
		t.Fatalf("forget ignored expected ID: %v", err)
	}
	got, err = r.Get(ctx, "workstation")
	if err != nil || got != entry {
		t.Fatalf("failed writes changed enrollment: %+v %v", got, err)
	}
	if err := r2.Forget(ctx, "workstation", entry.RunnerID); err != nil {
		t.Fatal(err)
	}
	all, err := r.List(ctx)
	if err != nil || len(all) != 0 {
		t.Fatalf("forgotten registry: %+v %v", all, err)
	}
	// Forgetting enrollment has no runner-side effects.
	c := client.New(socket)
	defer c.Close()
	if _, err := c.Snapshot(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Enroll(ctx, "offline", filepath.Join(t.TempDir(), "missing.sock")); err == nil {
		t.Fatal("unreachable endpoint enrolled")
	}
}

func TestExpectedEnrollmentRejectsIdentityBeforeWriting(t *testing.T) {
	r := registry(t)
	socket := fixture(t, "runner-real", nil)
	entry := Connection{Name: "local", Transport: "unix", Socket: socket}
	for _, expected := range []client.Identity{{RunnerID: "runner-other", Mode: "simulation"}, {RunnerID: "runner-real", Mode: "development-managed"}} {
		if _, err := r.enrollExpected(context.Background(), entry, &expected); err == nil {
			t.Fatal("unexpected identity enrolled")
		}
		all, err := r.List(context.Background())
		if err != nil || len(all) != 0 {
			t.Fatal("failed handshake wrote enrollment", all, err)
		}
	}
	expected := client.Identity{RunnerID: "runner-real", Mode: "simulation"}
	if c, err := r.enrollExpected(context.Background(), entry, &expected); err != nil || c.Identity() != expected {
		t.Fatal("expected identity refused", c, err)
	}
}

func TestConcurrentEnrollmentNoLostUpdates(t *testing.T) {
	r := registry(t)
	var wg sync.WaitGroup
	for i := range 8 {
		name := fmt.Sprintf("runner-%d", i)
		socket := fixture(t, name, nil)
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Independent handles, as with separate CLI processes.
			other, err := Open(r.dir)
			if err == nil {
				_, err = other.Enroll(context.Background(), name, socket)
			}
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	all, err := r.List(context.Background())
	if err != nil || len(all) != 8 {
		t.Fatalf("lost concurrent enrollment: %d %v", len(all), err)
	}
	for i, entry := range all {
		if entry.Name != fmt.Sprintf("runner-%d", i) {
			t.Fatal("registry is not deterministic")
		}
	}
}

func TestPrivateDirectoryAndCanceledLock(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(filepath.Join(link, "new")); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := os.Stat(filepath.Join(target, "new")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created state through refused symlink")
	}
	if err := os.Chmod(target, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(target); err == nil {
		t.Fatal("public directory accepted")
	}
	r := registry(t)
	lock, err := privateFile(filepath.Join(r.dir, "registry.lock"), os.O_RDWR|os.O_CREATE)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := r.List(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock did not respect cancellation: %v", err)
	}
}

func TestRegistryRejectsUnsafeAndUnknownFiles(t *testing.T) {
	for _, kind := range []string{"public", "symlink", "hardlink", "fifo", "truncated", "future", "unknown-field", "extra-value", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			r := registry(t)
			path := filepath.Join(r.dir, "runners.json")
			data := []byte(`{"version":1,"connections":[]}`)
			switch kind {
			case "truncated":
				data = []byte(`{"version":1`)
			case "future":
				data = []byte(`{"version":2,"connections":[]}`)
			case "unknown-field":
				data = []byte(`{"version":1,"connections":[],"secret":"do-not-log"}`)
			case "extra-value":
				data = append(data, []byte(` {}`)...)
			case "oversized":
				data = []byte(strings.Repeat(" ", maxRegistryBytes+1))
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "public":
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink", "hardlink":
				original := filepath.Join(r.dir, "original")
				if err := os.Rename(path, original); err != nil {
					t.Fatal(err)
				}
				var err error
				if kind == "symlink" {
					err = os.Symlink(original, path)
				} else {
					err = os.Link(original, path)
				}
				if err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := r.List(context.Background()); err == nil || strings.Contains(err.Error(), "do-not-log") {
				t.Fatalf("unsafe registry accepted or leaked contents: %v", err)
			}
		})
	}
}

func TestInventoryIsolationIdentityAndRedaction(t *testing.T) {
	r := registry(t)
	ctx := context.Background()
	for _, name := range []string{"one", "two"} {
		if _, err := r.Enroll(ctx, name, fixture(t, "runner-"+name, nil)); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := r.List(ctx)
	entries = append(entries, Connection{Name: "offline", RunnerID: "runner-offline", Mode: "simulation", Socket: filepath.Join(t.TempDir(), "absent.sock")})
	entries = append(entries, Connection{Name: "swapped", RunnerID: "runner-original", Mode: "simulation", Socket: entries[0].Socket})
	entries = append(entries, Connection{Name: "bad-snapshot", RunnerID: "runner-bad", Mode: "simulation", Socket: fixture(t, "runner-bad", func(w http.ResponseWriter, req *http.Request) {
		json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "wrong", Mode: "simulation"})
	})})
	entries = append(entries, Connection{Name: "remote-error", RunnerID: "runner-error", Mode: "simulation", Socket: fixture(t, "runner-error", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(503)
		json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "error", Message: "private-credential"}})
	})})
	got := Collect(ctx, entries)
	if got.Complete || len(got.Connections) != 6 {
		t.Fatalf("partial inventory: %+v", got)
	}
	for i, v := range got.Connections {
		if i < 2 {
			if v.Status != "available" || v.Snapshot == nil || v.Snapshot.Instances[0].ID != "alpha" || v.RunnerID != entries[i].RunnerID {
				t.Fatalf("same-named instances were conflated: %+v", v)
			}
		} else if v.Snapshot != nil || v.Error == nil || v.Status == "available" {
			t.Fatalf("failure presented a current snapshot: %+v", v)
		}
	}
	if got.Connections[3].Status != "identity_mismatch" || got.Connections[4].Status != "identity_mismatch" {
		t.Fatal("identity mismatch not classified")
	}
	data, _ := json.Marshal(got)
	if strings.Contains(string(data), "private-credential") {
		t.Fatal("remote diagnostics leaked")
	}
}

func TestWatchHealthyRunnerNotBlockedAndReconnects(t *testing.T) {
	var slowCalls atomic.Int32
	slow := fixture(t, "runner-slow", func(w http.ResponseWriter, r *http.Request) {
		slowCalls.Add(1)
		<-r.Context().Done()
	})
	var unavailable atomic.Bool
	unavailable.Store(true)
	healthy := fixture(t, "runner-healthy", func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			w.WriteHeader(503)
			return
		}
		json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner-healthy", Mode: "simulation"})
	})
	entries := []Connection{{Name: "slow", RunnerID: "runner-slow", Mode: "simulation", Socket: slow}, {Name: "healthy", RunnerID: "runner-healthy", Mode: "simulation", Socket: healthy}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	updates := 0
	err := Watch(ctx, entries, 100*time.Millisecond, func(v Observation) error {
		if v.Name != "healthy" {
			t.Fatalf("slow runner unexpectedly completed: %+v", v)
		}
		updates++
		if updates == 1 {
			if v.Status != "unavailable" || v.Snapshot != nil {
				t.Fatalf("bad failure: %+v", v)
			}
			unavailable.Store(false)
		} else {
			if v.Status != "available" || v.Snapshot == nil {
				t.Fatalf("failed reconnect: %+v", v)
			}
			if updates == 3 {
				cancel()
			}
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || updates != 3 || slowCalls.Load() != 1 {
		t.Fatalf("independent watch failed: updates=%d slow=%d err=%v", updates, slowCalls.Load(), err)
	}
}
