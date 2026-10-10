package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestImportCLIRequiresBoundSelectionAndConfirmation(t *testing.T) {
	posts := 0
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-import", Mode: "simulation", AccessRole: "operator", Features: []string{"import-qualified-fake"}})
			return
		}
		if r.URL.Path != "/v1/imports" {
			t.Error("unexpected route")
		}
		if r.Method == "GET" {
			reply(t, w, protocol.ImportCatalogue{RunnerID: "runner-import", Sources: []protocol.ImportSource{}})
			return
		}
		posts++
		var req protocol.ImportFakeRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || !req.ConfirmStopped || req.SourceRunID != strings.Repeat("a", 32) || req.ExpectedDigest != strings.Repeat("b", 64) || req.Config.HTTPPort != 18086 {
			t.Error("selection changed")
		}
		reply(t, w, protocol.Operation{ID: "op-import", InstanceID: req.ID, RequestID: req.RequestID, Kind: "import-fake", Status: "queued", ConfigRevision: 1})
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
	base := []string{"--runner", "host", "--expected-runner-id", "runner-import"}
	invoke(0, append(base, "import-sources")...)
	args := append(append([]string{}, base...), "import-fake", "copied", "--source-run", strings.Repeat("a", 32), "--expected-digest", strings.Repeat("b", 64), "--http-port", "18086", "--request-id", "import-one")
	invoke(2, args...)
	if posts != 0 {
		t.Fatal("import dispatched without acknowledgement")
	}
	invoke(0, append(args, "--confirm-stopped")...)
	if posts != 1 {
		t.Fatal("import submission count changed")
	}
	invoke(2, append(append(args, "--confirm-stopped"), "--source-path", "/private/pier")...)
	if posts != 1 {
		t.Fatal("external path accepted")
	}
}
