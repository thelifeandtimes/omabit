package client

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProjectRecoveryCheckpointClientValidatesBindings(t *testing.T) {
	for _, scenario := range []string{"valid", "unsupported", "runner", "project", "operation", "authority", "health", "history", "checkpoint", "member", "target", "finished", "phase"} {
		t.Run(scenario, func(t *testing.T) {
			var gets atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.Method != "GET" {
					t.Error("mutation")
				}
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectRecoveryCheckpointFeature}}
					if scenario == "unsupported" {
						h.Features = nil
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				gets.Add(1)
				if q.URL.Path != protocol.ProjectRecoveryCheckpointPath("project-one", "op-one") {
					t.Error(q.URL.Path)
				}
				inv := projectInventoryFixture(t)
				ended := inv.ReadAt.Add(-time.Second)
				v := protocol.ProjectRecoveryCheckpoint{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: inv.ReadAt, Basis: "stored-recovery-checkpoint", RuntimeObservation: "not-performed", ActionAuthority: "not-granted", Project: inv.Projects[0], Operation: protocol.ProjectOperationSummary{ID: "op-one", RequestID: "request-one", Kind: "recover-project-anchor", Actor: "actor", Status: "interrupted", ConfigRevision: 1, CreatedAt: inv.ReadAt.Add(-time.Minute), FinishedAt: &ended}, HistoryDigest: strings.Repeat("a", 64), CheckpointDigest: strings.Repeat("b", 64), Sequence: 4, Phase: "draining", Next: protocol.ProjectRecoveryHint{Kind: "observation", TargetKind: "member-exit", InstanceID: "demo-zod"}}
				switch scenario {
				case "runner":
					v.RunnerID = "other"
				case "project":
					v.Project.ID = "other"
				case "operation":
					v.Operation.ID = "other"
				case "authority":
					v.ActionAuthority = "granted"
				case "health":
					v.RuntimeObservation = "healthy"
				case "history":
					v.HistoryDigest = "bad"
				case "checkpoint":
					v.CheckpointDigest = strings.Repeat("z", 64)
				case "member":
					v.Next.InstanceID = "foreign"
				case "target":
					v.Next.TargetKind = "repeat-create"
				case "finished":
					v.Operation.FinishedAt = nil
				case "phase":
					v.Phase = "invented"
				}
				json.NewEncoder(w).Encode(v)
			})
			v, err := c.ProjectRecoveryCheckpoint(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, "project-one", "op-one")
			if scenario == "valid" {
				if err != nil || v.Operation.ID != "op-one" || gets.Load() != 1 {
					t.Fatal(v, err, gets.Load())
				}
			} else if err == nil || v.Operation.ID != "" {
				t.Fatal(v, err)
			}
			if scenario == "unsupported" && gets.Load() != 0 {
				t.Fatal("unsupported endpoint contacted")
			}
		})
	}
}
