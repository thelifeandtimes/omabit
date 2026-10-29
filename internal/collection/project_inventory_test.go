package collection

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectInventoryIndependentOfOrdinaryFleet(t *testing.T) {
	for _, scenario := range []string{"available", "unsupported", "error", "identity", "snapshot-error"} {
		t.Run(scenario, func(t *testing.T) {
			plan, err := projectnetwork.Build(projectnetwork.Request{Label: "demo", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "demo-nec", Ship: "~nec", HTTPPort: 18382}}}, projectnetwork.Reservations{})
			if err != nil {
				t.Fatal(err)
			}
			digest, _ := plan.Digest()
			now := time.Now().UTC()
			p := protocol.ProjectReservation{ID: "project-demo", Revision: 1, Label: plan.Label, CreatedAt: now, ReviewDigest: digest}
			for _, m := range plan.Members {
				p.Members = append(p.Members, protocol.ProjectMemberReservation{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort, InternalHTTPPort: m.InternalHTTPPort})
			}
			v := protocol.ProjectInventory{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: now, Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{p}}
			socket := filepath.Join(t.TempDir(), "r.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				switch q.URL.Path {
				case "/hello":
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation"}
					if scenario != "unsupported" {
						h.Features = []string{protocol.ProjectInventoryFeature}
					}
					json.NewEncoder(w).Encode(h)
				case "/v1/snapshot":
					if scenario == "snapshot-error" {
						w.WriteHeader(503)
						return
					}
					json.NewEncoder(w).Encode(protocol.Snapshot{RunnerID: "runner", Mode: "simulation", Instances: []protocol.Instance{{ID: "ordinary"}}})
				case protocol.ProjectInventoryPath:
					if scenario == "error" {
						w.WriteHeader(500)
						w.Write([]byte("untrusted remote diagnostic"))
						return
					}
					reply := v
					if scenario == "identity" {
						reply.RunnerID = "different"
					}
					json.NewEncoder(w).Encode(reply)
				default:
					t.Errorf("unexpected route %s", q.URL.Path)
				}
			})}
			go srv.Serve(ln)
			defer srv.Close()
			got := Collect(context.Background(), []Connection{{Name: "lab", RunnerID: "runner", Mode: "simulation", Transport: "unix", Socket: socket}}).Connections[0]
			if (got.Snapshot != nil) != (scenario != "snapshot-error") {
				t.Fatal("project inventory changed ordinary availability", got)
			}
			want := scenario == "available" || scenario == "snapshot-error"
			if (got.ProjectInventory != nil) != want {
				t.Fatal(got.ProjectInventoryStatus, got.ProjectInventory)
			}
			if want && (len(got.ProjectInventory.Projects) != 1 || got.ProjectInventory.RuntimeObservation != "not-performed") {
				t.Fatal(got.ProjectInventory)
			}
			if scenario == "unsupported" && got.ProjectInventoryStatus != "unsupported" {
				t.Fatal(got)
			}
		})
	}
}
