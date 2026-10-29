package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectStatusCLIExplainsHistoricalStatus(t *testing.T) {
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "Demo", ConfirmSharedAdministration: true, Members: []projectnetwork.MemberRequest{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381}}}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	now := time.Now().UTC()
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		if q.Method != "GET" {
			t.Error("mutation")
		}
		if q.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectStatusFeature}})
			return
		}
		if q.URL.Path != protocol.ProjectStatusPath("project-one") {
			t.Error(q.URL.Path)
		}
		reply(t, w, protocol.ProjectStatus{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: now, Basis: "stored-operation-history", RuntimeObservation: "not-performed", Project: protocol.ProjectReservation{ID: "project-one", Label: "Demo", Revision: 1, CreatedAt: now, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381, InternalHTTPPort: 8080}}}, Operations: []protocol.ProjectOperationSummary{{ID: "op-one", RequestID: "request-one", Kind: "prepare-project", Actor: "actor", Status: "running", ConfigRevision: 1, CreatedAt: now}}})
	})
	registry := filepath.Join(t.TempDir(), "collector")
	var enrolled, enrollErr bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "fixture", "--socket", socket}, &enrolled, &enrollErr); code != 0 {
		t.Fatal(code, enrollErr.String())
	}
	for _, format := range []string{"text", "json"} {
		var out, errs bytes.Buffer
		code := run(context.Background(), []string{"--config-dir", registry, "--runner", "fixture", "--expected-runner-id", "runner", "--format", format, "project-status", "project-one"}, &out, &errs)
		if code != 0 || errs.Len() != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "text" && (!strings.Contains(out.String(), "Saved history only.") || !strings.Contains(out.String(), "do not prove current ship health")) {
			t.Fatal(out.String())
		}
		if format == "json" {
			var saved protocol.ProjectStatus
			if json.Unmarshal(out.Bytes(), &saved) != nil || saved.RuntimeObservation != "not-performed" {
				t.Fatal(out.String())
			}
		}
	}
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"project-status", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "explicit recovery") {
		t.Fatal(code, out.String(), errs.String())
	}
}
