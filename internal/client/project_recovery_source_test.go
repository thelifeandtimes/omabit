package client

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProjectRecoverySourceClientBindings(t *testing.T) {
	for _, scenario := range []string{"valid", "unsupported", "runner", "project", "authority", "health", "history", "member", "generation", "missing-original", "duplicate-ref", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			var reads atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.Method != "GET" {
					t.Error("mutation")
				}
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectRecoverySourceFeature}}
					if scenario == "unsupported" {
						h.Features = nil
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				reads.Add(1)
				if q.URL.Path != protocol.ProjectRecoverySourcePath("project-one") {
					t.Error(q.URL.Path)
				}
				inv := projectInventoryFixture(t)
				v := protocol.ProjectRecoverySource{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: inv.ReadAt, Basis: "stored-recovery-source", RuntimeObservation: "not-performed", ActionAuthority: "not-granted", Project: inv.Projects[0], HistoryDigest: strings.Repeat("a", 64), Members: []protocol.ProjectRecoverySourceMember{{InstanceID: "demo-zod", CreatorOperationID: "creator", ContainerGeneration: 1, OriginalReadinessOperationID: "original", CurrentReadinessOperationID: "current"}}}
				switch scenario {
				case "runner":
					v.RunnerID = "foreign"
				case "project":
					v.Project.ID = "foreign"
				case "authority":
					v.ActionAuthority = "granted"
				case "health":
					v.RuntimeObservation = "healthy"
				case "history":
					v.HistoryDigest = strings.Repeat("z", 64)
				case "member":
					v.Members[0].InstanceID = "foreign"
				case "generation":
					v.Members[0].ContainerGeneration = 0
				case "missing-original":
					v.Members[0].OriginalReadinessOperationID = ""
				case "duplicate-ref":
					v.Members[0].CurrentReadinessOperationID = "original"
				case "empty":
					v.Members = nil
				}
				json.NewEncoder(w).Encode(v)
			})
			v, err := c.ProjectRecoverySource(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, "project-one")
			if scenario == "valid" {
				if err != nil || len(v.Members) != 1 || reads.Load() != 1 {
					t.Fatal(v, err, reads.Load())
				}
			} else if err == nil || v.Project.ID != "" {
				t.Fatal(v, err)
			}
			if scenario == "unsupported" && reads.Load() != 0 {
				t.Fatal("unsupported read sent")
			}
		})
	}
}
