package main

import (
	"bytes"
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestHostPortSuggestionPinsRunnerAndRejectsChangedReply(t *testing.T) {
	var replaced atomic.Bool
	var probes atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-port", Mode: "development-managed", AccessRole: "operator", Features: []string{"host-port-suggestion"}})
			return
		}
		if r.Method != "GET" || r.URL.Path != "/v1/port-suggestion" {
			t.Error("unexpected port request", r.Method, r.URL.Path)
		}
		probes.Add(1)
		id := "runner-port"
		if replaced.Load() {
			id = "runner-other"
		}
		reply(t, w, protocol.PortSuggestion{RunnerID: id, HTTPPort: 18088, ContainerPort: 8080, ObservedAt: time.Now().UTC()})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) string {
		t.Helper()
		var out, errs bytes.Buffer
		if code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs); code != want {
			t.Fatalf("code %d want %d: %s", code, want, &errs)
		}
		return out.String()
	}
	invoke(0, "enroll", "host", "--socket", socket)
	invoke(2, "--runner", "host", "suggest-port")
	if probes.Load() != 0 {
		t.Fatal("unreviewed runner probed")
	}
	if out := invoke(0, "--runner", "host", "--expected-runner-id", "runner-port", "suggest-port"); !strings.Contains(out, `"http_port": 18088`) {
		t.Fatal(out)
	}
	replaced.Store(true)
	invoke(1, "--runner", "host", "--expected-runner-id", "runner-port", "suggest-port")
}
