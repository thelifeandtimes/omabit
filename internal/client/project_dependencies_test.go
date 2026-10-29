package client

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestProjectDependenciesClientPinsAndNoRetry(t *testing.T) {
	for _, mode := range []string{"ok", "unsupported", "observer", "runner", "image", "pill", "lost", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			var posts atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.ProjectDependenciesFeature}}
					if mode == "unsupported" {
						h.Features = nil
					}
					if mode == "observer" {
						h.AccessRole = "observer"
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				posts.Add(1)
				if q.URL.Path != protocol.ProjectDependenciesPath || q.Method != "POST" {
					t.Error(q.URL.Path, q.Method)
				}
				var req protocol.ProjectDependenciesRequest
				if json.NewDecoder(q.Body).Decode(&req) != nil || !req.ConfirmDownload {
					t.Error("missing consent")
				}
				if mode == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
					} else {
						conn.Close()
					}
					return
				}
				v := protocol.ProjectDependenciesResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", Image: candidate.Image, Pill: protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes}}
				if mode == "runner" {
					v.RunnerID = "other"
				}
				if mode == "image" {
					v.Image = "unpinned"
				}
				if mode == "pill" {
					v.Pill.Bytes++
				}
				json.NewEncoder(w).Encode(v)
			})
			ctx := context.Background()
			if mode == "lookup" {
				ctx = WithCreationReplayOnly(ctx)
			}
			err := c.PrepareProjectDependencies(ctx, Identity{RunnerID: "runner", Mode: "simulation"})
			if (err == nil) != (mode == "ok") {
				t.Fatal(err)
			}
			want := int32(1)
			if mode == "unsupported" || mode == "observer" || mode == "lookup" {
				want = 0
			}
			if posts.Load() != want {
				t.Fatal("unexpected dispatch/retry", posts.Load())
			}
		})
	}
}
