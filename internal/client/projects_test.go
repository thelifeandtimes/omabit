package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func projectInventoryFixture(t *testing.T) protocol.ProjectInventory {
	t.Helper()
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "Example", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	now := time.Now().UTC()
	return protocol.ProjectInventory{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: now, Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{{ID: "project-one", Revision: 1, Label: "Example", CreatedAt: now.Add(-time.Hour), ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381, InternalHTTPPort: 8080}}}}}
}
func TestProjectInventoryCapabilityIdentityAndClaimValidation(t *testing.T) {
	for _, scenario := range []string{"valid", "empty", "older", "identity", "mode", "schema", "basis", "runtime", "digest", "port", "order", "duplicate", "future"} {
		t.Run(scenario, func(t *testing.T) {
			var reads atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.Method != "GET" {
					t.Error("mutation")
				}
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectInventoryFeature}}
					if scenario == "older" {
						h.Features = nil
					}
					json.NewEncoder(w).Encode(h)
					return
				}
				reads.Add(1)
				if q.URL.Path != protocol.ProjectInventoryPath {
					t.Error(q.URL.Path)
				}
				r := projectInventoryFixture(t)
				switch scenario {
				case "empty":
					r.Projects = []protocol.ProjectReservation{}
				case "identity":
					r.RunnerID = "replacement"
				case "mode":
					r.Mode = "other"
				case "schema":
					r.SchemaVersion++
				case "basis":
					r.Basis = "live"
				case "runtime":
					r.RuntimeObservation = "ready"
				case "digest":
					r.Projects[0].ReviewDigest = "bad"
				case "port":
					r.Projects[0].Members[0].InternalHTTPPort++
				case "order":
					r.Projects = append(r.Projects, r.Projects[0])
				case "duplicate":
					p := r.Projects[0]
					p.ID = "project-two"
					r.Projects = append(r.Projects, p)
				case "future":
					r.Projects[0].CreatedAt = r.ReadAt.Add(time.Hour)
				}
				json.NewEncoder(w).Encode(r)
			})
			out, err := c.Projects(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"})
			if scenario == "valid" || scenario == "empty" {
				if err != nil || out.RunnerID != "runner" {
					t.Fatal(out, err)
				}
				return
			}
			if err == nil || out.Projects != nil {
				t.Fatal("invalid inventory accepted", out, err)
			}
			if scenario == "older" && (!errors.Is(err, ErrProjectInventoryUnsupported) || reads.Load() != 0) {
				t.Fatal("unsupported route queried", err)
			}
			if (scenario == "identity" || scenario == "mode") && !errors.Is(err, ErrIdentityMismatch) {
				t.Fatal(err)
			}
		})
	}
}
