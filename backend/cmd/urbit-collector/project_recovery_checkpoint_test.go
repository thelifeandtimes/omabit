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

func TestProjectRecoveryCheckpointCLIExplainsSavedState(t *testing.T) {
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
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectRecoveryCheckpointFeature}})
			return
		}
		if q.URL.Path != protocol.ProjectRecoveryCheckpointPath("project-one", "op-one") {
			t.Error(q.URL.Path)
		}
		reply(t, w, protocol.ProjectRecoveryCheckpoint{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: now, Basis: "stored-recovery-checkpoint", RuntimeObservation: "not-performed", ActionAuthority: "not-granted", Project: protocol.ProjectReservation{ID: "project-one", Label: "Demo", Revision: 1, CreatedAt: now, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{{InstanceID: "demo-zod", Ship: "~zod", HTTPPort: 18381, InternalHTTPPort: 8080}}}, Operation: protocol.ProjectOperationSummary{ID: "op-one", RequestID: "request-one", Kind: "recover-project-anchor", Actor: "actor", Status: "queued", ConfigRevision: 1, CreatedAt: now}, HistoryDigest: strings.Repeat("a", 64), CheckpointDigest: strings.Repeat("b", 64), Phase: "review-planned", Next: protocol.ProjectRecoveryHint{Kind: "pending"}})

	})
	registry := filepath.Join(t.TempDir(), "collector")
	var enrolled, enrollErr bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "fixture", "--socket", socket}, &enrolled, &enrollErr); code != 0 {
		t.Fatal(code, enrollErr.String())
	}
	for _, format := range []string{"text", "json"} {
		var out, errs bytes.Buffer
		code := run(context.Background(), []string{"--config-dir", registry, "--runner", "fixture", "--expected-runner-id", "runner", "--format", format, "project-recovery-checkpoint", "project-one", "op-one"}, &out, &errs)
		if code != 0 || errs.Len() != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "text" && (!strings.Contains(out.String(), "Saved history only;") || !strings.Contains(out.String(), "no current health check or action permission")) {
			t.Fatal(out.String())
		}
		if format == "json" {
			var saved protocol.ProjectRecoveryCheckpoint
			if json.Unmarshal(out.Bytes(), &saved) != nil || saved.RuntimeObservation != "not-performed" {
				t.Fatal(out.String())
			}
		}
	}
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"project-recovery-checkpoint", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "no permission to retry") {
		t.Fatal(code, out.String(), errs.String())
	}
}
