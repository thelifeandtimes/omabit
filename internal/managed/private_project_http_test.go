package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func privateRegistrationPublicFixture(t *testing.T) privatechain.RegisterRequest {
	q := privateProjectRequest(t, "register-one", "a", 18381)
	out := privatechain.RegisterRequest{RequestID: q.RequestID, Label: q.Plan.Label, Foundation: q.Plan.Foundation, ConfirmSharedAdministration: true, ConfirmReservation: true, ReviewDigest: q.ReviewDigest}
	for _, m := range q.Plan.Members {
		out.Members = append(out.Members, privatechain.MemberRequest{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	return out
}
func TestPrivateProjectPublicRegistrationPublicReceiptAndReplay(t *testing.T) {
	r, _, socket, _ := testEndpointSocket(t, privateDir(t), &fakeBackend{})
	probes := 0
	r.s.portAvailable = func(string, uint16) bool { probes++; return true }
	pin := client.Identity{RunnerID: r.s.runnerID, Mode: r.s.mode}
	c := client.NewPinned(socket, pin)
	defer c.Close()
	req := privateRegistrationPublicFixture(t)
	saved, err := c.RegisterPrivateProject(context.Background(), pin, req)
	if err != nil || !saved.ReservationOnly {
		t.Fatal(saved, err)
	}
	if probes != 2 {
		t.Fatal("unexpected port probes", probes)
	}
	for _, table := range []string{"instances", "project_jobs", "project_member_storage", "project_member_launches"} {
		var count int
		if err = r.s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal(table, count, err)
		}
	}
	for _, ctx := range []context.Context{context.Background(), client.WithCreationReplayOnly(context.Background())} {
		replay, err := c.RegisterPrivateProject(ctx, pin, req)
		if err != nil || !reflect.DeepEqual(saved, replay) || probes != 2 {
			t.Fatal("replay changed or probed", replay, err, probes)
		}
	}
	req.RequestID = "missing-request"
	_, err = c.RegisterPrivateProject(client.WithCreationReplayOnly(context.Background()), pin, req)
	var remote *client.RemoteError
	if !errors.As(err, &remote) || remote.Problem.Code != "replay_not_found" {
		t.Fatal(err)
	}
	if probes != 2 {
		t.Fatal("missing lookup performed host checks")
	}
	status, err := c.PrivateProjects(context.Background(), pin)
	if err != nil || len(status.Projects) != 1 || status.Projects[0].RegistrationOperationID != saved.Operation.ID {
		t.Fatal(status, err)
	}
	req.RequestID = "register-one"
	req.ReviewDigest = "changed"
	_, err = c.RegisterPrivateProject(context.Background(), pin, req)
	if err == nil {
		t.Fatal("unreviewed plan accepted")
	}
	var projects int
	r.s.db.QueryRow("SELECT count(*) FROM private_projects").Scan(&projects)
	if projects != 1 {
		t.Fatal(projects)
	}
}
func TestPrivateProjectPublicRegistrationPublicRefusals(t *testing.T) {
	for _, scenario := range []string{"observer", "anonymous", "origin", "quiesced", "consent", "shared", "digest", "unknown-field", "bad-replay-header", "other-actor", "oversized", "duplicate", "case-alias"} {
		t.Run(scenario, func(t *testing.T) {
			r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			expectedCount := 0
			if scenario == "other-actor" {
				if _, _, err = r.s.registerPrivateProject("original-actor", privateProjectRequest(t, "register-one", "a", 18381)); err != nil {
					t.Fatal(err)
				}
				expectedCount = 1
			}
			r.s.portAvailable = func(string, uint16) bool { t.Error("refused request probed host"); return true }
			body := privateRegistrationPublicFixture(t)
			if scenario == "consent" {
				body.ConfirmReservation = false
			}
			if scenario == "shared" {
				body.ConfirmSharedAdministration = false
			}
			if scenario == "digest" {
				body.ReviewDigest = "changed"
			}
			raw, _ := json.Marshal(body)
			if scenario == "unknown-field" {
				raw = append(raw[:len(raw)-1], []byte(`,"start":true}`)...)
			}
			if scenario == "duplicate" {
				raw = append(raw[:len(raw)-1], []byte(`,"request_id":"other"}`)...)
			}
			if scenario == "case-alias" {
				raw = append(raw[:len(raw)-1], []byte(`,"Request_ID":"other"}`)...)
			}
			if scenario == "oversized" {
				raw = append(raw, bytes.Repeat([]byte(" "), 64*1024)...)
			}
			q := httptest.NewRequest("POST", protocol.PrivateChainProjectsPath, bytes.NewReader(raw))
			q.Header.Set(protocol.Header, "1")
			q.Header.Set("Content-Type", "application/json")
			if scenario != "anonymous" {
				q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
			}
			if scenario == "origin" {
				q.Header.Set("Origin", "https://untrusted.example")
			}
			if scenario == "observer" || scenario == "other-actor" {
				role := "observer"
				if scenario == "other-actor" {
					role = "operator"
					q.Header.Set(protocol.CreationReplayHeader, "true")
				}
				line, key := grantLine(t, role)
				r.keys = filepath.Join(privateDir(t), "keys")
				if err = os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				q.Header.Set(access.Header, key)
			}
			if scenario == "quiesced" {
				r.updateFence.Held = true
			}
			if scenario == "bad-replay-header" {
				q.Header.Set(protocol.CreationReplayHeader, "false")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, q)
			want := 400
			if scenario == "observer" || scenario == "anonymous" || scenario == "origin" {
				want = 403
			}
			if scenario == "quiesced" || scenario == "other-actor" {
				want = 409
			}
			if w.Code != want {
				t.Fatal("wrong refusal", w.Code, w.Body.String())
			}
			if scenario == "other-actor" || scenario == "quiesced" {
				var response protocol.ErrorResponse
				if json.Unmarshal(w.Body.Bytes(), &response) != nil {
					t.Fatal(w.Body.String())
				}
				code := "replay_not_found"
				if scenario == "quiesced" {
					code = "management_quiesced"
				}
				if response.Error.Code != code {
					t.Fatal(response)
				}
			}
			var count int
			r.s.db.QueryRow("SELECT count(*) FROM private_projects").Scan(&count)
			if count != expectedCount {
				t.Fatal("reservation created despite refusal")
			}
		})
	}
}
