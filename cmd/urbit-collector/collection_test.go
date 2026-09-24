package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestEnrolledCLIAndReplacementRefusal(t *testing.T) {
	var replaced atomic.Bool
	var mutations atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		id := "runner-original"
		if replaced.Load() {
			id = "runner-replacement"
		}
		switch r.URL.Path {
		case "/hello":
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: "simulation"})
		case "/v1/snapshot":
			reply(t, w, protocol.Snapshot{RunnerID: id, Mode: "simulation", Instances: []protocol.Instance{{ID: "alpha"}}})
		default:
			mutations.Add(1)
			w.WriteHeader(409)
			revision := uint64(2)
			reply(t, w, protocol.ErrorResponse{Error: protocol.APIError{Code: "revision_conflict", Message: "configuration changed", CurrentRevision: &revision}})
		}
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) (string, string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &stdout, &stderr)
		if code != want {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, &stdout, &stderr)
		}
		return stdout.String(), stderr.String()
	}
	enrollment, _ := invoke(0, "enroll", "workstation", "--socket", socket)
	var saved collection.Connection
	if err := json.Unmarshal([]byte(enrollment), &saved); err != nil || saved.RunnerID != "runner-original" {
		t.Fatalf("enrollment: %s %v", enrollment, err)
	}
	out, _ := invoke(0, "runners")
	if !strings.Contains(out, "workstation") {
		t.Fatal("enrollment did not persist between invocations")
	}
	invoke(0, "--runner", "workstation", "hello")
	invoke(0, "--runner", "workstation", "snapshot")
	_, problem := invoke(1, "--runner", "workstation", "configure", "alpha", "--revision=1", "--http-port=18081", "--ames-port=34343", "--loom=31")
	if got := errorProblem(t, problem, "revision_conflict"); got.CurrentRevision == nil || *got.CurrentRevision != 2 {
		t.Fatalf("conflict lost: %+v", got)
	}
	if mutations.Load() != 1 {
		t.Fatal("configuration retried")
	}
	invoke(0, "inventory")
	replaced.Store(true)
	for _, args := range [][]string{
		{"--runner", "workstation", "hello"},
		{"--runner", "workstation", "snapshot"},
		{"--runner", "workstation", "operate", "alpha", "start", "--revision=2", "--request-id=do-not-start"},
	} {
		out, problem := invoke(1, args...)
		if out != "" {
			t.Fatal("replacement returned trusted output")
		}
		errorProblem(t, problem, "runner_identity_mismatch")
	}
	if mutations.Load() != 1 {
		t.Fatal("replacement received a mutation")
	}
	out, problem = invoke(1, "inventory")
	if problem != "" || !strings.Contains(out, `"status": "identity_mismatch"`) || !strings.Contains(out, `"complete": false`) {
		t.Fatalf("partial inventory lost: %s %s", out, problem)
	}
	invoke(1, "enroll", "workstation", "--socket", socket)
	invoke(1, "forget", "workstation", "--runner-id", "runner-replacement")
	invoke(0, "forget", "workstation", "--runner-id", "runner-original")
	out, _ = invoke(0, "runners")
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("forget failed: %s", out)
	}
}

func TestCollectionUsageAndHelp(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "collector")
	for _, args := range [][]string{
		{"enroll"}, {"enroll", "name"}, {"enroll", "bad/name", "--socket=/tmp/runner.sock"},
		{"enroll", "name", "--socket=relative"}, {"enroll", "name", "--socket=/tmp/../tmp/r.sock"},
		{"forget", "name"}, {"forget", "name", "--runner-id=bad/id"},
		{"runners", "extra"}, {"inventory", "extra"}, {"inventory", "--interval=1s"},
		{"inventory", "--watch", "--interval=99ms"},
		{"--socket=/tmp/runner.sock", "enroll", "name"},
		{"--runner=one", "inventory"}, {"--runner=one", "--socket=/tmp/r.sock", "snapshot"},
		{"--config-dir=relative", "runners"}, {"--runner=bad/name", "hello"},
	} {
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &stdout, &stderr)
		if code != 2 || stdout.Len() != 0 {
			t.Fatalf("%q: code=%d out=%s err=%s", args, code, &stdout, &stderr)
		}
		errorProblem(t, stderr.String(), "client_error")
	}
	for _, command := range []string{"enroll", "runners", "forget", "inventory"} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), []string{command, "--help"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Usage:") {
			t.Fatalf("%s help: %d %s %s", command, code, &stdout, &stderr)
		}
	}
}

func TestInventoryCLIWatchAndDefaultDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-one", Mode: "simulation"})
		} else {
			reply(t, w, protocol.Snapshot{RunnerID: "runner-one", Mode: "simulation"})
		}
	})
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"enroll", "one", "--socket", socket}, &stdout, &stderr); code != 0 {
		t.Fatalf("enroll: %d %s", code, &stderr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	updates := 0
	output := writerFunc(func(p []byte) (int, error) {
		var got collection.Observation
		if err := json.Unmarshal(p, &got); err != nil || got.Name != "one" || got.Status != "available" || got.Snapshot == nil {
			t.Fatalf("watch: %s %v", p, err)
		}
		updates++
		if updates == 2 {
			cancel()
		}
		return len(p), nil
	})
	if code := run(ctx, []string{"inventory", "--watch", "--interval=100ms"}, output, &stderr); code != 0 || stderr.Len() != 0 || updates != 2 {
		t.Fatalf("watch: %d updates=%d %s", code, updates, &stderr)
	}
}

func TestEnrolledWatchCanceledBeforeReadingRegistry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := filepath.Join(t.TempDir(), "collector")
	for _, args := range [][]string{{"inventory", "--watch"}, {"--runner=one", "watch"}} {
		var stdout, stderr bytes.Buffer
		if code := run(ctx, append([]string{"--config-dir", dir}, args...), &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("canceled watch: %d stdout=%s stderr=%s", code, &stdout, &stderr)
		}
	}
}
