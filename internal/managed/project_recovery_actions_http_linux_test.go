//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

type publicRecoveryFixture struct {
	r               *Server
	b               *queuedRecoveryBackend
	project, parent string
	req             protocol.ProjectRecoveryActionRequest
	accepted        func(string)
	release         func()
}

func publicRecoveryActionFixture(t *testing.T, kind string) publicRecoveryFixture {
	var f publicRecoveryFixture
	if kind == "start" {
		r, b, p, q := recoveryQueueFixture(t)
		f = publicRecoveryFixture{r: r, b: b, project: p.ID, req: protocol.ProjectRecoveryActionRequest{RequestID: "public-recovery-start", ExpectedRevision: p.Revision, ReviewDigest: q.ReviewDigest, HistoryDigest: q.HistoryDigest, ConfirmAction: true}, accepted: func(id string) { prepareQueuedRecoveryReference(t, r, b, id) }, release: func() { close(b.release) }}
	} else {
		var p storedProject
		var parent projectAnchorRecoveryJob
		if kind == "observe" {
			r, b, project, j := interruptedObservationFixture(t, "signal")
			r.b = &queuedRecoveryObservationBackend{queuedRecoveryBackend: b}
			f.r, f.b, p, parent = r, b, project, j
		} else {
			r, b, project, j := prepareRecoveryContinuation(t, "signal")
			f.r, f.b, p, parent = r, b.queuedRecoveryBackend, project, j
		}
		q := observationRequest(p, parent, "public-recovery-"+kind)
		seq := q.ExpectedSequence
		f.project, f.parent = p.ID, parent.OperationID
		f.req = protocol.ProjectRecoveryActionRequest{RequestID: q.RequestID, ExpectedRevision: q.ExpectedRevision, ReviewDigest: q.ReviewDigest, HistoryDigest: q.HistoryDigest, ExpectedSequence: &seq, CheckpointDigest: q.CheckpointDigest, ConfirmAction: true}
		f.accepted = func(string) {}
		f.release = func() {}
	}
	f.r.uid = uint32(os.Geteuid())
	return f
}
func publicRecoveryHTTP(t *testing.T, f publicRecoveryFixture, kind string, replay bool) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(f.req)
	q := httptest.NewRequest("POST", protocol.ProjectRecoveryActionPath(f.project, f.parent, kind), bytes.NewReader(raw))
	q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
	q.Header.Set(protocol.Header, "1")
	q.Header.Set("Content-Type", "application/json")
	if replay {
		q.Header.Set(protocol.CreationReplayHeader, "true")
	}
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, q)
	return w
}
func TestPublicRecoveryActionQueuesAndLookup(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		t.Run(kind, func(t *testing.T) {
			f := publicRecoveryActionFixture(t, kind)
			before := append([]string(nil), f.b.calls...)
			f.r.workGate <- struct{}{}
			w := publicRecoveryHTTP(t, f, kind, false)
			if w.Code != 202 {
				t.Fatal(w.Code, w.Body.String())
			}
			var accepted protocol.ProjectRecoveryActionResult
			g02Check(t, json.Unmarshal(w.Body.Bytes(), &accepted))
			_, opKind := protocol.ProjectRecoveryAction(kind)
			if accepted.Operation.Kind != opKind || accepted.Operation.Status != "queued" || accepted.HistoryDigest != f.req.HistoryDigest || accepted.CheckpointDigest != f.req.CheckpointDigest || !reflect.DeepEqual(accepted.ExpectedSequence, f.req.ExpectedSequence) {
				t.Fatal(accepted)
			}
			expectedParent := f.parent
			if kind == "start" {
				expectedParent = accepted.Operation.ID
			}
			if accepted.RecoveryOperationID != expectedParent {
				t.Fatal(accepted)
			}
			f.accepted(accepted.Operation.ID)
			hit := publicRecoveryHTTP(t, f, kind, true)
			if hit.Code != 200 || hit.Body.String() != w.Body.String() || f.r.updateWorkers != 1 {
				t.Fatal(hit.Code, hit.Body.String())
			}
			absent := f
			absent.req.RequestID = "missing-lookup"
			hit = publicRecoveryHTTP(t, absent, kind, true)
			if hit.Code != 409 || !strings.Contains(hit.Body.String(), "replay_not_found") || f.r.updateWorkers != 1 {
				t.Fatal(hit.Code, hit.Body.String())
			}
			changed := f
			changed.req.HistoryDigest = strings.Repeat("a", 64)
			hit = publicRecoveryHTTP(t, changed, kind, true)
			if hit.Code != 409 || f.r.updateWorkers != 1 {
				t.Fatal("changed replay accepted", hit.Code, hit.Body.String())
			}
			<-f.r.workGate
			f.release()
			waitRecoveryQueue(t, f.r)
			actual, err := loadOperation(f.r.s.db, accepted.Operation.ID)
			g02Check(t, err)
			if actual.Status != "succeeded" || f.r.failed.Load() {
				t.Fatal(actual, f.r.failed.Load())
			}
			if kind == "observe" && !reflect.DeepEqual(before, f.b.calls) {
				t.Fatal("observation changed effects")
			}
			completedCalls := append([]string(nil), f.b.calls...)
			hit = publicRecoveryHTTP(t, f, kind, true)
			if hit.Code != 200 || f.r.updateWorkers != 0 || !reflect.DeepEqual(completedCalls, f.b.calls) {
				t.Fatal(hit.Code, hit.Body.String())
			}
			var completed protocol.ProjectRecoveryActionResult
			g02Check(t, json.Unmarshal(hit.Body.Bytes(), &completed))
			if completed.Operation.ID != actual.ID || completed.Operation.Status != "succeeded" {
				t.Fatal(completed)
			}
		})
	}
}
func TestPublicRecoveryActionMissingParentAndCorruption(t *testing.T) {
	for _, kind := range []string{"observe", "continue"} {
		for _, scenario := range []string{"missing", "wrong-kind", "corrupt"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := publicRecoveryActionFixture(t, kind)
				before := append([]string(nil), f.b.calls...)
				want := 409
				switch scenario {
				case "missing":
					f.parent = "unknown-parent"
				case "wrong-kind":
					g02Check(t, f.r.s.db.QueryRow("SELECT operation_id FROM project_member_readiness ORDER BY operation_id LIMIT 1").Scan(&f.parent))
				case "corrupt":
					_, err := f.r.s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", []byte(`{}`), f.parent)
					g02Check(t, err)
					want = 500
				}
				w := publicRecoveryHTTP(t, f, kind, false)
				if w.Code != want || f.r.failed.Load() != (scenario == "corrupt") || f.r.updateWorkers != 0 || !reflect.DeepEqual(before, f.b.calls) {
					t.Fatal(w.Code, w.Body.String(), f.r.failed.Load())
				}
			})
		}
	}
}
func TestPublicRecoveryActionUnknownCreateNeverAdoptsOrRepeats(t *testing.T) {
	for _, kind := range []string{"observe", "continue"} {
		t.Run(kind, func(t *testing.T) {
			r, b, p, j := interruptedObservationFixture(t, "create-unknown")
			r.b = &continuationQueueBackend{&queuedRecoveryObservationBackend{queuedRecoveryBackend: b}}
			r.uid = uint32(os.Geteuid())
			q := observationRequest(p, j, "unknown-create-action")
			seq := q.ExpectedSequence
			f := publicRecoveryFixture{r: r, b: b, project: p.ID, parent: j.OperationID, req: protocol.ProjectRecoveryActionRequest{RequestID: q.RequestID, ExpectedRevision: q.ExpectedRevision, ReviewDigest: q.ReviewDigest, HistoryDigest: q.HistoryDigest, ExpectedSequence: &seq, CheckpointDigest: q.CheckpointDigest, ConfirmAction: true}}
			before := append([]string(nil), b.calls...)
			w := publicRecoveryHTTP(t, f, kind, false)
			if w.Code != 409 || r.failed.Load() || r.updateWorkers != 0 || !reflect.DeepEqual(before, b.calls) {
				t.Fatal(w.Code, w.Body.String())
			}
			after, err := loadProjectAnchorRecovery(r.s.db, j.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(after, j) {
				t.Fatal("unknown Create intent changed")
			}
		})
	}
}
