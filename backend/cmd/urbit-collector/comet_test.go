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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestExplicitModeTransitionAndCometRegistration(t *testing.T) {
	var advanced, replaced atomic.Bool
	var posts atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		mode := "development-managed-fake"
		features := []string{"create-fake"}
		id := "runner-comet"
		if advanced.Load() {
			mode = "development-managed"
			features = append(features, "create-comet")
		}
		if replaced.Load() {
			id = "runner-replacement"
		}
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: id, Mode: mode, AccessRole: "operator", Features: features})
			return
		}
		if r.URL.Path != "/v1/comets" || r.Method != "POST" {
			t.Error("unexpected comet request", r.URL.Path)
		}
		var body protocol.CreateCometRequest
		if json.NewDecoder(r.Body).Decode(&body) != nil || !body.ConfirmNetwork || body.ID != "comet" || body.Config.HTTPPort != 18087 {
			t.Error("bad comet request")
		}
		posts.Add(1)
		reply(t, w, protocol.Operation{ID: "op-comet", InstanceID: body.ID, RequestID: body.RequestID, Kind: "register-comet", Status: "succeeded", ConfigRevision: 1})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) string {
		t.Helper()
		var out, errs bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
		if code != want {
			t.Fatalf("code=%d want=%d %s", code, want, &errs)
		}
		return out.String()
	}
	invoke(0, "enroll", "host", "--socket", socket)
	base := []string{"--runner", "host", "--expected-runner-id", "runner-comet", "create-comet", "comet", "--http-port", "18087", "--request-id", "comet-reg"}
	invoke(2, base...)
	invoke(1, append(base, "--confirm-network")...)
	advanced.Store(true)
	invoke(1, "runner-edit", "host", "--runner-id", "runner-comet", "--revision", "1", "--name", "host", "--socket", socket)
	modeArgs := []string{"runner-accept-mode", "host", "--runner-id", "runner-comet", "--revision", "1", "--mode", "development-managed"}
	replaced.Store(true)
	invoke(1, modeArgs...)
	replaced.Store(false)
	if list := invoke(0, "runners"); !strings.Contains(list, `"mode": "development-managed-fake"`) {
		t.Fatal("pin changed on failed verification")
	}
	result := invoke(0, modeArgs...)
	if !strings.Contains(result, `"mode": "development-managed"`) || !strings.Contains(result, `"revision": 2`) {
		t.Fatal("mode transition not versioned")
	}
	invoke(1, modeArgs...)
	invoke(0, append(base, "--confirm-network")...)
	if posts.Load() != 1 {
		t.Fatal("comet registration retried")
	}
}
