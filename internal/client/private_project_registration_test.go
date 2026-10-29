package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func privateClientRegistrationFixture(t *testing.T) privatechain.RegisterRequest {
	t.Helper()
	data, err := os.ReadFile("../privatechain/testdata/qualified-foundation.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := privatechain.DecodeFoundation(data)
	if err != nil {
		t.Fatal(err)
	}
	req := privatechain.RegisterRequest{RequestID: "request-one", Label: "Private demo", Foundation: f, Members: []privatechain.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "demo-nec", Ship: "~nec", HTTPPort: 18382}}, ConfirmSharedAdministration: true, ConfirmReservation: true}
	plan, err := privatechain.Build(privatechain.Request{Label: req.Label, Foundation: f, Members: req.Members, ConfirmSharedAdministration: true}, privatechain.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	req.ReviewDigest, _ = plan.Digest()
	return req
}

func TestPrivateProjectRegistrationLostReplyDoesNotRepeatMutation(t *testing.T) {
	req := privateClientRegistrationFixture(t)
	var posts atomic.Int32
	now := time.Now().UTC()
	saved := privatechain.RegistrationResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, ReservationOnly: true, Binding: privatechain.Binding{Mode: privatechain.Mode, RunnerID: "runner", ProjectID: "private-project-one", Generation: 1, FoundationDigest: privateFixtureFoundationDigest(t, req)}, Operation: protocol.Operation{ID: "op-one", ProjectID: "private-project-one", RequestID: req.RequestID, Kind: privatechain.RegistrationKind, Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}}
	c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if q.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.PrivateChainRegistrationFeature, protocol.CreationReplayFeature}})
			return
		}
		if q.URL.Path != protocol.PrivateChainProjectsPath || q.Method != "POST" {
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
	_, err := c.RegisterPrivateProject(context.Background(), pin, req)
	var uncertain *OutcomeUncertainError
	if !errors.As(err, &uncertain) || posts.Load() != 1 {
		t.Fatal(err, posts.Load())
	}
	got, err := c.RegisterPrivateProject(WithCreationReplayOnly(context.Background()), pin, req)
	if err != nil || got.Operation.ID != saved.Operation.ID || posts.Load() != 2 {
		t.Fatal(got, err, posts.Load())
	}
}
func TestPrivateProjectRegistrationRejectsUnboundResults(t *testing.T) {
	for _, scenario := range []string{"unsupported", "observer", "actor", "runner", "digest", "runtime", "request", "unfinished", "malformed", "binding-project", "binding-runner", "binding-generation", "binding-foundation", "authorized"} {
		t.Run(scenario, func(t *testing.T) {
			req := privateClientRegistrationFixture(t)
			var posts atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set(protocol.Header, "1")
				if q.URL.Path == "/hello" {
					h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.PrivateChainRegistrationFeature}}
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
				result := privatechain.RegistrationResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, ReservationOnly: true, Binding: privatechain.Binding{Mode: privatechain.Mode, RunnerID: "runner", ProjectID: "private-project-one", Generation: 1, FoundationDigest: privateFixtureFoundationDigest(t, req)}, Operation: protocol.Operation{ID: "op-one", ProjectID: "private-project-one", RequestID: req.RequestID, Kind: privatechain.RegistrationKind, Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}}
				switch scenario {
				case "binding-project":
					result.Binding.ProjectID = "private-project-other"
				case "binding-runner":
					result.Binding.RunnerID = "other"
				case "binding-generation":
					result.Binding.Generation++
				case "binding-foundation":
					result.Binding.FoundationDigest = "changed"
				case "authorized":
					result.RuntimeAuthorized = true
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
			got, err := c.RegisterPrivateProject(context.Background(), Identity{RunnerID: "runner", Mode: "simulation"}, req)
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

func privateFixtureFoundationDigest(t *testing.T, req privatechain.RegisterRequest) string {
	t.Helper()
	d, e := req.Foundation.Digest()
	if e != nil {
		t.Fatal(e)
	}
	return d
}
