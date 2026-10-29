//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func initialActionHTTP(t *testing.T, r *Server, id string, body protocol.ProjectActionRequest, replay bool) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	q := httptest.NewRequest("POST", protocol.ProjectActionPath(id), bytes.NewReader(raw))
	q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
	q.Header.Set(protocol.Header, "1")
	q.Header.Set("Content-Type", "application/json")
	if replay {
		q.Header.Set(protocol.CreationReplayHeader, "true")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, q)
	return w
}
func TestProjectActionPublicQueuesAndReplay(t *testing.T) {
	for _, kind := range []string{"prepare", "start-anchor", "prepare-storage", "launch", "observe"} {
		t.Run(kind, func(t *testing.T) {
			var r *Server
			var id, digest, instance string
			var release chan struct{}
			var ready *queuedReadinessBackend
			switch kind {
			case "prepare":
				server, b, project, q := projectQueueFixture(t)
				r, id, digest, release = server, project, q.ReviewDigest, b.release
			case "start-anchor":
				server, b, project, q := anchorQueueFixture(t)
				r, id, digest, release = server, project, q.ReviewDigest, b.release
			case "prepare-storage":
				server, b, project, q := memberStorageQueueFixture(t)
				r, id, digest, release, instance = server, project, q.ReviewDigest, b.release, "queued-zod"
			case "launch":
				server, b, project, q := memberLaunchQueueFixture(t)
				r, id, digest, release, instance = server, project.ID, q.ReviewDigest, b.release, project.Members[0].InstanceID
			case "observe":
				server, b, project, q := readinessQueueFixture(t)
				r, id, digest, release, instance, ready = server, project.ID, q.ReviewDigest, b.release, b.instance, b
			}
			r.uid = uint32(os.Geteuid())
			r.workGate <- struct{}{}
			feature, operation, _ := protocol.ProjectAction(kind)
			if !slices.Contains(r.projectActionFeatures(), feature) {
				t.Fatal("capability missing")
			}
			req := protocol.ProjectActionRequest{Kind: kind, InstanceID: instance, RequestID: "public-" + kind, ExpectedRevision: 1, ReviewDigest: digest, ConfirmAction: true}
			w := initialActionHTTP(t, r, id, req, false)
			if w.Code != 202 {
				t.Fatal(w.Code, w.Body.String())
			}
			var accepted protocol.ProjectActionResult
			if json.Unmarshal(w.Body.Bytes(), &accepted) != nil || accepted.Operation.Kind != operation || accepted.Operation.Status != "queued" || accepted.Operation.InstanceID != instance || accepted.Operation.ProjectID != id {
				t.Fatal(w.Body.String())
			}
			if ready != nil {
				ready.opID = accepted.Operation.ID
			}
			if r.updateWorkers != 1 {
				t.Fatal("queue missing worker fence", r.updateWorkers)
			}
			replay := initialActionHTTP(t, r, id, req, true)
			if replay.Code != 200 || replay.Body.String() != w.Body.String() {
				t.Fatal("replay changed", replay.Code, replay.Body.String())
			}
			missing := req
			missing.RequestID = "missing-lookup"
			replay = initialActionHTTP(t, r, id, missing, true)
			if replay.Code != 409 || !strings.Contains(replay.Body.String(), "replay_not_found") {
				t.Fatal(replay.Code, replay.Body.String())
			}
			if r.updateWorkers != 1 {
				t.Fatal("lookup scheduled extra worker")
			}
			<-r.workGate
			close(release)
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, accepted.Operation.ID)
			if err != nil || actual.Status != "succeeded" {
				t.Fatal(actual, err)
			}
			replay = initialActionHTTP(t, r, id, req, true)
			if replay.Code != 200 {
				t.Fatal(replay.Code, replay.Body.String())
			}
			var completed protocol.ProjectActionResult
			json.Unmarshal(replay.Body.Bytes(), &completed)
			if completed.Operation.ID != actual.ID || completed.Operation.Status != "succeeded" || r.updateWorkers != 0 {
				t.Fatal(completed, r.updateWorkers)
			}
		})
	}
}
func TestProjectActionPublicRefusesBeforeDispatch(t *testing.T) {
	for _, scenario := range []string{"observer", "unsupported", "member", "confirmation", "revision", "digest", "quiesced", "unknown", "missing"} {
		t.Run(scenario, func(t *testing.T) {
			r, _, id, prepared := projectQueueFixture(t)
			r.uid = uint32(os.Geteuid())
			q := protocol.ProjectActionRequest{Kind: "prepare", RequestID: "refused", ExpectedRevision: 1, ReviewDigest: prepared.ReviewDigest, ConfirmAction: true}
			switch scenario {
			case "unsupported":
				q.Kind = "start-anchor"
			case "member":
				q.InstanceID = "queued-zod"
			case "confirmation":
				q.ConfirmAction = false
			case "revision":
				q.ExpectedRevision = 2
			case "digest":
				q.ReviewDigest = strings.Repeat("0", 64)
			case "quiesced":
				r.updateFence.Held = true
			case "unknown":
				q.Kind = "delete-project"
			case "missing":
				id = "unknown-project"
			}
			raw, _ := json.Marshal(q)
			req := httptest.NewRequest("POST", protocol.ProjectActionPath(id), bytes.NewReader(raw))
			req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
			req.Header.Set(protocol.Header, "1")
			req.Header.Set("Content-Type", "application/json")
			if scenario == "observer" {
				line, key := grantLine(t, "observer")
				r.keys = filepath.Join(privateDir(t), "keys")
				if err := os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				req.Header.Set(access.Header, key)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			want := 400
			if scenario == "observer" {
				want = 403
			}
			if scenario == "unsupported" || scenario == "revision" || scenario == "quiesced" || scenario == "missing" {
				want = 409
			}
			if w.Code != want || r.updateWorkers != 0 {
				t.Fatal(scenario, w.Code, w.Body.String(), r.updateWorkers)
			}
			var n int
			r.s.db.QueryRow("SELECT count(*) FROM project_jobs").Scan(&n)
			if n != 0 {
				t.Fatal("refused action admitted", n)
			}
		})
	}
}
