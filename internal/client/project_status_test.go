package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectStatusRejectsUnboundOrMisleadingServerResults(t *testing.T) {
	for _, scenario := range []string{"valid", "older", "runner", "project", "live-claim", "actor", "member", "unfinished-terminal", "duplicate", "untruthful-truncation"} {
		t.Run(scenario, func(t *testing.T) {
			reads := 0
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.Method != "GET" {
					t.Error("mutation")
				}
				if q.URL.Path == "/hello" {
					features := []string{protocol.ProjectStatusFeature}
					if scenario == "older" {
						features = nil
					}
					json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: features})
					return
				}
				reads++
				p := projectInventoryFixture(t)
				when := p.Projects[0].CreatedAt
				s := protocol.ProjectStatus{SchemaVersion: 1, RunnerID: p.RunnerID, Mode: p.Mode, ReadAt: p.ReadAt, Basis: "stored-operation-history", RuntimeObservation: "not-performed", Project: p.Projects[0], Operations: []protocol.ProjectOperationSummary{{ID: "op-one", RequestID: "request-one", Kind: "register-project", Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: when, FinishedAt: &when}}}
				switch scenario {
				case "runner":
					s.RunnerID = "other"
				case "project":
					s.Project.ID = "other"
				case "live-claim":
					s.RuntimeObservation = "running"
				case "actor":
					s.Operations[0].Actor = ""
				case "member":
					s.Operations[0].Kind = "launch-project-member"
					s.Operations[0].InstanceID = "foreign"
				case "unfinished-terminal":
					s.Operations[0].FinishedAt = nil
				case "duplicate":
					s.Operations = append(s.Operations, s.Operations[0])
				case "untruthful-truncation":
					s.HasOlderOperations = true
				}
				json.NewEncoder(w).Encode(s)
			})
			result, err := c.ProjectStatus(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, "project-one")
			if scenario == "valid" {
				if err != nil || len(result.Operations) != 1 {
					t.Fatal(result, err)
				}
				return
			}
			if err == nil || result.Operations != nil {
				t.Fatal("unbound result accepted", result, err)
			}
			if scenario == "older" && (!errors.Is(err, ErrProjectStatusUnsupported) || reads != 0) {
				t.Fatal("unsupported route queried", err, reads)
			}
		})
	}
}
