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

func clientRegistrationFixture(t *testing.T) protocol.ProjectRegisterRequest {
	p, err := projectnetwork.Build(projectnetwork.Request{Label: "Demo", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Digest()
	return protocol.ProjectRegisterRequest{RequestID: "request-one", Label: p.Label, Members: []protocol.ProjectMemberInput{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}}, ConfirmSharedAdministration: true, ConfirmReservation: true, ReviewDigest: digest}
}
func TestProjectRegistrationLostReplyDoesNotRepeatMutation(t *testing.T) {
	req := clientRegistrationFixture(t)
	var posts atomic.Int32
	now := time.Now().UTC()
	saved := protocol.ProjectRegistrationResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, ReservationOnly: true, Operation: protocol.Operation{ID: "op-one", ProjectID: "project-one", RequestID: req.RequestID, Kind: "register-project", Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}}
	c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if q.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.ProjectRegistrationFeature, protocol.CreationReplayFeature}})
			return
		}
		if q.URL.Path != protocol.ProjectInventoryPath || q.Method != "POST" {
			t.Error("unexpected route", q.Method, q.URL.Path)
			return
		}
		n := posts.Add(1)
		if n == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		if n != 2 || q.Header.Get(protocol.CreationReplayHeader) != "true" {
			t.Error("unsafe replay", n, q.Header)
		}
		json.NewEncoder(w).Encode(saved)
	})
	pin := Identity{RunnerID: "runner", Mode: "simulation"}
	_, err := c.RegisterProject(context.Background(), pin, req)
	var uncertain *OutcomeUncertainError
	if !errors.As(err, &uncertain) || posts.Load() != 1 {
		t.Fatal(err, posts.Load())
	}
	got, err := c.RegisterProject(WithCreationReplayOnly(context.Background()), pin, req)
	if err != nil || got.Operation.ID != saved.Operation.ID || posts.Load() != 2 {
		t.Fatal(got, err, posts.Load())
	}
}
func TestProjectRegistrationRejectsUnboundResults(t *testing.T) {
	for _, scenario := range []string{"unsupported", "observer", "actor", "runner", "digest", "runtime", "request", "unfinished", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			req := clientRegistrationFixture(t)
			var posts atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.ProjectRegistrationFeature}}
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
				now := time.Now().UTC()
				result := protocol.ProjectRegistrationResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, ReservationOnly: true, Operation: protocol.Operation{ID: "op-one", ProjectID: "project-one", RequestID: req.RequestID, Kind: "register-project", Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}}
				switch scenario {
				case "actor":
					result.Operation.Actor = "foreign"
				case "runner":
					result.RunnerID = "foreign"
				case "digest":
					result.ReviewDigest = "wrong"
				case "runtime":
					result.ReservationOnly = false
				case "request":
					result.Operation.RequestID = "other"
				case "unfinished":
					result.Operation.FinishedAt = nil
				case "malformed":
					w.Write([]byte(`{`))
					return
				}
				json.NewEncoder(w).Encode(result)
			})
			got, err := c.RegisterProject(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, req)
			if err == nil || got.Operation.ID != "" {
				t.Fatal(got, err)
			}
			if scenario == "unsupported" || scenario == "observer" {
				if posts.Load() != 0 {
					t.Fatal("preflight forwarded mutation")
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
