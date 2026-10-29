package main

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublicRecoveryActionCLIExplicitInputsAndLookup(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		t.Run(kind, func(t *testing.T) {
			feature, operation := protocol.ProjectRecoveryAction(kind)
			parent := ""
			if kind != "start" {
				parent = "parent"
			}
			var posts atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
				if q.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{feature, protocol.CreationReplayFeature}})
					return
				}
				n := posts.Add(1)
				if q.Method != "POST" || q.URL.Path != protocol.ProjectRecoveryActionPath("project", parent, kind) {
					t.Error(q.Method, q.URL.Path)
				}
				if n == 2 && q.Header.Get(protocol.CreationReplayHeader) != "true" {
					t.Error("missing lookup guard")
				}
				var req protocol.ProjectRecoveryActionRequest
				if json.NewDecoder(q.Body).Decode(&req) != nil || !protocol.ValidProjectRecoveryAction(kind, parent, req) {
					t.Error(req)
				}
				if kind != "start" && *req.ExpectedSequence != 0 {
					t.Error("explicit zero lost", req)
				}
				owner := parent
				if kind == "start" {
					owner = "op-one"
				}
				reply(t, w, protocol.ProjectRecoveryActionResult{ProjectActionResult: protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, Operation: protocol.Operation{ID: "op-one", ProjectID: "project", RequestID: req.RequestID, Kind: operation, Actor: "actor", Status: "queued", ConfigRevision: 1, CreatedAt: time.Now().UTC()}}, RecoveryOperationID: owner, HistoryDigest: req.HistoryDigest, CheckpointDigest: req.CheckpointDigest, ExpectedSequence: req.ExpectedSequence})
			})
			registry := filepath.Join(t.TempDir(), "registry")
			var out, errs bytes.Buffer
			if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "lab", "--socket", socket}, &out, &errs); code != 0 {
				t.Fatal(code, errs.String())
			}
			base := []string{"--config-dir", registry, "--runner", "lab", "--expected-runner-id", "runner"}
			args := []string{"project-recover", "project", kind, "--revision", "1", "--review-digest", strings.Repeat("a", 64), "--history-digest", strings.Repeat("b", 64), "--request-id", "request", "--confirm-action"}
			if parent != "" {
				args = append(args, "--recovery", parent, "--sequence", "0", "--checkpoint-digest", strings.Repeat("c", 64))
			}
			for _, format := range []string{"text", "json"} {
				out.Reset()
				errs.Reset()
				command := append(append(append([]string{}, base...), "--format", format), args...)
				if format == "json" {
					command = append(command, "--replay-only")
				}
				if code := run(context.Background(), command, &out, &errs); code != 0 {
					t.Fatal(code, errs.String())
				}
				if format == "text" && !strings.Contains(out.String(), "Saved receipt only") {
					t.Fatal(out.String())
				}
			}
			if posts.Load() != 2 {
				t.Fatal(posts.Load())
			}
		})
	}
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"project-recover", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "without scheduling another worker") {
		t.Fatal(code, out.String(), errs.String())
	}
}
