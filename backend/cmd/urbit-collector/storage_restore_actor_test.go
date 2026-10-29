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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRestorationCLIActorBindingBeforeDispatch(t *testing.T) {
	var previews, restores atomic.Int32
	actor := "local-uid:1000"
	digest := strings.Repeat("a", 64)
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-restore", Mode: "development-managed", AccessRole: "operator", Actor: actor, Features: []string{"restore-import"}})
			return
		}
		switch r.URL.Path {
		case "/v1/instances/zod/restore-import-preview":
			previews.Add(1)
			reply(t, w, protocol.ImportRestorePreview{ImportRestoreResult: protocol.ImportRestoreResult{Source: "/source/held/pier", RestoreTo: "/source/zod", Destination: "/managed/fakeships/zod", RetainAt: "/managed/archive/import-id", ReviewDigest: digest}, RunnerID: "runner-restore", InstanceID: "zod", ConfigRevision: 2, SourceFiles: 3, SourceBytes: 100, DestinationFiles: 1, DestinationBytes: 20, Stage: "review"})
		case "/v1/instances/zod/operations":
			var req protocol.OperationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				return
			}
			if req.Kind != "restore-import" || req.ExpectedRevision != 2 || req.RestoreDigest != digest || !req.ConfirmRestore {
				t.Error("lost restoration approval", req)
			}
			restores.Add(1)
			reply(t, w, protocol.Operation{ID: "op-restore", InstanceID: "zod", RequestID: req.RequestID, Kind: req.Kind, Actor: actor, ConfigRevision: 2, Status: "queued", CreatedAt: time.Now().UTC()})
		default:
			t.Error("unexpected restoration request", r.URL.Path)
		}
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
	invoke(0, "enroll", "lab", "--socket", socket)
	pinned := []string{"--runner", "lab", "--expected-runner-id", "runner-restore"}
	preview := []string{"preview-import-restore", "zod", "--revision", "2"}
	restore := []string{"restore-import", "zod", "--revision", "2", "--restore-digest", digest, "--confirm-restore", "--request-id", "restore-request"}
	for _, args := range [][]string{preview, restore} {
		invoke(2, append(append(append([]string{}, pinned...), args...), "--expected-actor", "ssh-key:other")...)
	}
	if previews.Load() != 0 || restores.Load() != 0 {
		t.Fatal("actor mismatch reached the preview or mutation endpoint")
	}
	output := invoke(0, append(append(append([]string{}, pinned...), preview...), "--expected-actor", actor)...)
	var result struct {
		protocol.ImportRestorePreview
		Actor string `json:"actor"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil || result.Actor != actor || result.ReviewDigest != digest {
		t.Fatal("preview omitted its authenticated actor", output, err)
	}
	invoke(0, append(append(append([]string{}, pinned...), restore...), "--expected-actor", actor)...)
	if previews.Load() != 1 || restores.Load() != 1 {
		t.Fatal("expected one reviewed preview and one mutation", previews.Load(), restores.Load())
	}
	// Existing explicit CLI workflows may omit the new optional UI binding.
	invoke(0, append(append([]string{}, pinned...), preview...)...)
	if previews.Load() != 2 {
		t.Fatal("optional actor binding broke ordinary CLI review")
	}
}
