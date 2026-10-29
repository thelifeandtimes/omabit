package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectActionClientBindingsAndLostReply(t *testing.T) {
	for _, scenario := range []string{"continue-start", "valid", "lookup", "unsupported", "observer", "actor", "runner", "member", "request", "kind", "revision", "digest", "unfinished", "lost"} {
		t.Run(scenario, func(t *testing.T) {
			req := protocol.ProjectActionRequest{Kind: "launch", InstanceID: "member", RequestID: "request", ExpectedRevision: 1, ReviewDigest: strings.Repeat("a", 64), ConfirmAction: true}
			if scenario == "continue-start" {
				req.Kind = "member-continue"
				req.AfterOperationID = "op-interrupted"
			}
			var posts atomic.Int32
			feature, kind, _ := protocol.ProjectAction(req.Kind)
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{feature, protocol.CreationReplayFeature}}
					if scenario == "unsupported" {
						h.Features = nil
					}
					if scenario == "observer" {
						h.AccessRole = "observer"
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				posts.Add(1)
				if q.URL.Path != protocol.ProjectActionPath("project-one") || q.Method != "POST" {
					t.Error(q.URL.Path, q.Method)
				}
				if scenario == "lookup" && q.Header.Get(protocol.CreationReplayHeader) != "true" {
					t.Error("lookup guard absent")
				}
				if scenario == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
					} else {
						conn.Close()
					}
					return
				}
				result := protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, Operation: protocol.Operation{ID: "op-one", ProjectID: "project-one", InstanceID: req.InstanceID, Kind: kind, RequestID: req.RequestID, ConfigRevision: 1, Actor: "actor", Status: "queued", CreatedAt: time.Now().UTC()}}
				if scenario == "continue-start" {
					result.Operation.Kind = "start-project-ship"
				}
				switch scenario {
				case "actor":
					result.Operation.Actor = "other"
				case "runner":
					result.RunnerID = "other"
				case "member":
					result.Operation.InstanceID = "other"
				case "request":
					result.Operation.RequestID = "other"
				case "kind":
					result.Operation.Kind = "stop-project-member"
				case "revision":
					result.Operation.ConfigRevision = 2
				case "digest":
					result.ReviewDigest = "changed"
				case "unfinished":
					result.Operation.Status = "succeeded"
				}
				json.NewEncoder(w).Encode(result)
			})
			ctx := context.Background()
			if scenario == "lookup" {
				ctx = WithCreationReplayOnly(ctx)
			}
			result, err := c.ProjectOperate(ctx, Identity{RunnerID: "runner", Mode: "simulation"}, "project-one", req)
			if scenario == "valid" || scenario == "lookup" || scenario == "continue-start" {
				if err != nil || result.Operation.ID != "op-one" || posts.Load() != 1 {
					t.Fatal(result, err, posts.Load())
				}
				return
			}
			if err == nil || result.Operation.ID != "" {
				t.Fatal(result, err)
			}
			if scenario == "unsupported" || scenario == "observer" {
				if posts.Load() != 0 {
					t.Fatal("preflight sent effect")
				}
			} else {
				var uncertain *OutcomeUncertainError
				if !errors.As(err, &uncertain) || posts.Load() != 1 {
					t.Fatal(err, posts.Load())
				}
			}
		})
	}
}
