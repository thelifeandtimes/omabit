package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRecoveryCLI(t *testing.T) {
	var supported, observer atomic.Bool
	supported.Store(true)
	var posts atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			features := []string{}
			if supported.Load() {
				features = append(features, "recover-import")
			}
			role := "operator"
			if observer.Load() {
				role = "observer"
			}
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-recovery", Mode: "development-managed", AccessRole: role, Features: features})
			return
		}
		if r.Method != "POST" || r.URL.Path != "/v1/instances/one/operations" {
			t.Error("unexpected recovery request", r.Method, r.URL.Path)
		}
		var req protocol.OperationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Kind != "recover-import" || req.ExpectedRevision != 3 || req.RequestID != "recover-once" {
			t.Error("bad recovery binding", req)
		}
		posts.Add(1)
		reply(t, w, protocol.Operation{ID: "op-recover", Kind: req.Kind, InstanceID: "one", RequestID: req.RequestID, ConfigRevision: 3, Status: "queued"})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) {
		t.Helper()
		var out, errs bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
		if code != want {
			t.Fatalf("code=%d want=%d %s", code, want, &errs)
		}
	}
	invoke(0, "enroll", "host", "--socket", socket)
	args := []string{"--runner", "host", "--expected-runner-id", "runner-recovery", "operate", "one", "recover-import", "--revision", "3", "--request-id", "recover-once"}
	invoke(0, args...)
	supported.Store(false)
	invoke(1, args...)
	supported.Store(true)
	observer.Store(true)
	invoke(1, args...)
	observer.Store(false)
	invoke(2, "--socket", socket, "operate", "one", "recover-import", "--revision", "3", "--request-id", "recover-once")
	invoke(2, "--runner", "host", "operate", "one", "recover-import", "--revision", "3", "--request-id", "recover-once")
	args[3] = "wrong-runner"
	invoke(1, args...)
	if posts.Load() != 1 {
		t.Fatal("recovery sent without authorization or retried", posts.Load())
	}
}
