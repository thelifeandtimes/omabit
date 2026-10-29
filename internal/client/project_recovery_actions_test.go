package client

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublicRecoveryActionClientBindingAndLostReply(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		for _, scenario := range []string{"valid", "lookup", "lost", "unsupported", "observer", "runner", "actor", "parent", "request", "history", "checkpoint", "sequence", "kind", "unfinished"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				req := protocol.ProjectRecoveryActionRequest{RequestID: "request", ExpectedRevision: 1, ReviewDigest: strings.Repeat("a", 64), HistoryDigest: strings.Repeat("b", 64), ConfirmAction: true}
				parent := ""
				if kind != "start" {
					seq := uint64(4)
					req.ExpectedSequence = &seq
					req.CheckpointDigest = strings.Repeat("c", 64)
					parent = "parent"
				}
				feature, operation := protocol.ProjectRecoveryAction(kind)
				var posts atomic.Int32
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
					if q.Method != "POST" || q.URL.Path != protocol.ProjectRecoveryActionPath("project", parent, kind) {
						t.Error(q.Method, q.URL.Path)
					}
					if scenario == "lookup" && q.Header.Get(protocol.CreationReplayHeader) != "true" {
						t.Error("missing lookup header")
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
					owner := parent
					if kind == "start" {
						owner = "op-one"
					}
					v := protocol.ProjectRecoveryActionResult{ProjectActionResult: protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, Operation: protocol.Operation{ID: "op-one", ProjectID: "project", RequestID: req.RequestID, Kind: operation, Actor: "actor", Status: "queued", ConfigRevision: 1, CreatedAt: time.Now().UTC()}}, RecoveryOperationID: owner, HistoryDigest: req.HistoryDigest, CheckpointDigest: req.CheckpointDigest, ExpectedSequence: req.ExpectedSequence}
					switch scenario {
					case "runner":
						v.RunnerID = "other"
					case "actor":
						v.Operation.Actor = "other"
					case "parent":
						v.RecoveryOperationID = "other"
					case "request":
						v.Operation.RequestID = "other"
					case "history":
						v.HistoryDigest = "changed"
					case "checkpoint":
						v.CheckpointDigest = "changed"
					case "sequence":
						bad := uint64(99)
						v.ExpectedSequence = &bad
					case "kind":
						v.Operation.Kind = "operate"
					case "unfinished":
						v.Operation.Status = "succeeded"
					}
					json.NewEncoder(w).Encode(v)
				})
				ctx := context.Background()
				if scenario == "lookup" {
					ctx = WithCreationReplayOnly(ctx)
				}
				v, err := c.ProjectRecover(ctx, Identity{RunnerID: "runner", Mode: "simulation"}, "project", parent, kind, req)
				if scenario == "valid" || scenario == "lookup" {
					if err != nil || v.Operation.ID != "op-one" || posts.Load() != 1 {
						t.Fatal(v, err, posts.Load())
					}
					return
				}
				if err == nil || v.Operation.ID != "" {
					t.Fatal(v, err)
				}
				if scenario == "unsupported" || scenario == "observer" {
					if posts.Load() != 0 {
						t.Fatal("unsupported effect")
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
}
