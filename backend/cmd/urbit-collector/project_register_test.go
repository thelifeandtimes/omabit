package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectRegisterCLIReviewedReservationAndLookup(t *testing.T) {
	var plan, errs bytes.Buffer
	if code := run(context.Background(), []string{"project-plan", "--label", "demo", "--member", "demo-zod:~zod:18381", "--shared-administration"}, &plan, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	var view projectPlanView
	if err := json.Unmarshal(plan.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(file, plan.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var posts atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "actor", AccessRole: "operator", Features: []string{protocol.ProjectRegistrationFeature, protocol.CreationReplayFeature}})
			return
		}
		if q.Method != "POST" || q.URL.Path != protocol.ProjectInventoryPath {
			t.Error(q.Method, q.URL.Path)
			return
		}
		n := posts.Add(1)
		if n == 2 && q.Header.Get(protocol.CreationReplayHeader) != "true" {
			t.Error("lookup header missing")
		}
		var req protocol.ProjectRegisterRequest
		if err := json.NewDecoder(q.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.RequestID != "request-one" || req.ReviewDigest != view.ReviewDigest || !req.ConfirmReservation || len(req.Members) != 1 {
			t.Error(req)
		}
		now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
		reply(t, w, protocol.ProjectRegistrationResult{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReviewDigest: req.ReviewDigest, ReservationOnly: true, Operation: protocol.Operation{ID: "op-one", ProjectID: "project-one", RequestID: req.RequestID, Kind: "register-project", Actor: "actor", Status: "succeeded", ConfigRevision: 1, CreatedAt: now, FinishedAt: &now}})
	})
	registry := filepath.Join(t.TempDir(), "registry")
	var out bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "lab", "--socket", socket}, &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	base := []string{"--config-dir", registry, "--runner", "lab", "--expected-runner-id", "runner"}
	action := []string{"project-register", "--plan", file, "--review-digest", view.ReviewDigest, "--request-id", "request-one", "--confirm-reservation"}
	for _, format := range []string{"text", "json"} {
		out.Reset()
		errs.Reset()
		args := append(append(append([]string{}, base...), "--format", format), action...)
		if format == "json" {
			args = append(args, "--replay-only")
		}
		if code := run(context.Background(), args, &out, &errs); code != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "text" && !strings.Contains(out.String(), "No ships started or piers created.") {
			t.Fatal(out.String())
		}
		if format == "json" {
			var result protocol.ProjectRegistrationResult
			if json.Unmarshal(out.Bytes(), &result) != nil || !result.ReservationOnly {
				t.Fatal(out.String())
			}
		}
	}
	for _, args := range [][]string{action, append(append([]string{}, base...), action[:len(action)-1]...), append(append([]string{}, base...), "project-register", "--plan", file, "--request-id", "request-one", "--review-digest", "wrong", "--confirm-reservation")} {
		out.Reset()
		errs.Reset()
		if code := run(context.Background(), args, &out, &errs); code != 2 {
			t.Fatal(code, errs.String())
		}
	}
	if posts.Load() != 2 {
		t.Fatal("invalid command dispatched", posts.Load())
	}
	out.Reset()
	errs.Reset()
	if code := run(context.Background(), []string{"project-register", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "never creates a missing reservation") {
		t.Fatal(code, out.String(), errs.String())
	}
}

func TestProjectRegistrationLostReplyGuidance(t *testing.T) {
	err := &client.OutcomeUncertainError{Method: "POST", Path: protocol.ProjectInventoryPath, Err: errors.New("lost response")}
	g := guidanceFor(err, protocol.APIError{})
	if g == nil || g.Kind != "reconcile_project_reservation" || !strings.Contains(g.Summary, "--replay-only") || !strings.Contains(g.Summary, "do not choose a new ID") {
		t.Fatal(g)
	}
	if len(g.Next) != 2 || g.Next[0].Name != "project-register --help" {
		t.Fatal(g)
	}
}
