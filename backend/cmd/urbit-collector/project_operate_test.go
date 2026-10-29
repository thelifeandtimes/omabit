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

func TestProjectOperateCLIExplicitActions(t *testing.T) {
	for _, kind := range []string{"prepare", "start-anchor", "prepare-storage", "launch", "observe", "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current"} {
		t.Run(kind, func(t *testing.T) {
			feature, operation, member := protocol.ProjectAction(kind)
			var posts atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
				if q.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{feature, protocol.CreationReplayFeature}})
					return
				}
				n := posts.Add(1)
				if q.Method != "POST" || q.URL.Path != protocol.ProjectActionPath("project-one") {
					t.Error(q.Method, q.URL.Path)
				}
				if n == 2 && q.Header.Get(protocol.CreationReplayHeader) != "true" {
					t.Error("missing lookup guard")
				}
				var req protocol.ProjectActionRequest
				if json.NewDecoder(q.Body).Decode(&req) != nil {
					t.Error("invalid body")
				}
				if req.Kind != kind || !req.ConfirmAction || req.ExpectedRevision != 1 || req.RequestID != "request" || member != (req.InstanceID == "member") {
					t.Error(req)
				}

				switch kind {
				case "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current":
					if req.AfterOperationID != "prerequisite" {
						t.Error(req)
					}
				}
				if kind == "observe-current" && req.OriginalReadinessOperationID != "original-ready" {
					t.Error(req)
				}
				reply(t, w, protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, Operation: protocol.Operation{ID: "op-one", ProjectID: "project-one", InstanceID: req.InstanceID, Kind: operation, RequestID: req.RequestID, ConfigRevision: 1, Actor: "actor", Status: "queued", CreatedAt: time.Now().UTC()}})
			})
			registry := filepath.Join(t.TempDir(), "registry")
			var out, errs bytes.Buffer
			if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "lab", "--socket", socket}, &out, &errs); code != 0 {
				t.Fatal(code, errs.String())
			}
			base := []string{"--config-dir", registry, "--runner", "lab", "--expected-runner-id", "runner"}
			args := []string{"project-operate", "project-one", kind, "--revision", "1", "--review-digest", strings.Repeat("a", 64), "--request-id", "request", "--confirm-action"}
			if member {
				args = append(args, "--member", "member")
			}

			switch kind {
			case "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current":
				args = append(args, "--after", "prerequisite")
			}
			if kind == "observe-current" {
				args = append(args, "--original-readiness", "original-ready")
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
				if format == "text" && !strings.Contains(out.String(), "does not prove current health") {
					t.Fatal(out.String())
				}
			}
			if posts.Load() != 2 {
				t.Fatal(posts.Load())
			}
		})
	}
}
func TestProjectOperateCLIRefusesUnreviewedSelection(t *testing.T) {
	for _, args := range [][]string{{"project-operate"}, {"project-operate", "p", "stop"}, {"project-operate", "p", "prepare", "--revision", "1"}} {
		var out, errs bytes.Buffer
		if code := run(context.Background(), args, &out, &errs); code != 2 {
			t.Fatal(code, errs.String())
		}
	}
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"project-operate", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "without starting another worker") {
		t.Fatal(code, out.String(), errs.String())
	}
}
