//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProjectRecoveryCheckpointPreservesUncertaintyAndEffects(t *testing.T) {
	for _, where := range []string{"signal", "remove", "start", "create", "create-unknown"} {
		t.Run(where, func(t *testing.T) {
			r, b, p, j := interruptedObservationFixture(t, where)
			before := append([]string(nil), b.calls...)
			var changes int
			r.s.db.QueryRow("SELECT total_changes()").Scan(&changes)
			out, err := r.s.projectRecoveryCheckpoint(context.Background(), p.ID, j.OperationID)
			g02Check(t, err)
			want := "observation"
			if where == "create" {
				want = "continuation"
			}
			if where == "create-unknown" {
				want = "review"
			}
			if out.Next.Kind != want || out.Sequence != j.Journal.Sequence || out.CheckpointDigest != recoveryCheckpointDigest(j.Journal) || out.HistoryDigest != recoveryHistoryDigest(j.Source) || out.Operation.Status != "interrupted" || out.ActionAuthority != "not-granted" || out.RuntimeObservation != "not-performed" {
				t.Fatal(out)
			}
			var after int
			r.s.db.QueryRow("SELECT total_changes()").Scan(&after)
			if changes != after || !reflect.DeepEqual(before, b.calls) || r.updateWorkers != 0 {
				t.Fatal("read changed state/effects")
			}
			kept, err := loadProjectAnchorRecovery(r.s.db, j.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(j, kept) {
				t.Fatal("checkpoint changed")
			}
			raw, _ := json.Marshal(out)
			for _, secret := range []string{"container_id", "contract_digest", "runtime_id", "cookie", "/srv/", "/home/", "original_readiness"} {
				if strings.Contains(string(raw), secret) {
					t.Fatal("private journal exposed", secret)
				}
			}
		})
	}
}
func TestProjectRecoveryCheckpointPendingAndComplete(t *testing.T) {
	r, b, p, req := recoveryQueueFixture(t)
	r.workGate <- struct{}{}
	op, _, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	prepareQueuedRecoveryReference(t, r, b, op.ID)
	before := append([]string(nil), b.calls...)
	v, err := r.s.projectRecoveryCheckpoint(context.Background(), p.ID, op.ID)
	g02Check(t, err)
	if v.Next.Kind != "pending" || v.Sequence != 0 || !reflect.DeepEqual(before, b.calls) {
		t.Fatal(v, b.calls)
	}
	<-r.workGate
	close(b.release)
	waitRecoveryQueue(t, r)
	before = append([]string(nil), b.calls...)
	v, err = r.s.projectRecoveryCheckpoint(context.Background(), p.ID, op.ID)
	g02Check(t, err)
	if v.Next.Kind != "complete" || v.Operation.Status != "succeeded" || !reflect.DeepEqual(before, b.calls) {
		t.Fatal(v, b.calls)
	}
}
func TestProjectRecoveryCheckpointObserverAndMissingSelection(t *testing.T) {
	r, _, p, j := interruptedObservationFixture(t, "signal")
	r.uid = uint32(os.Geteuid())
	line, key := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	g02Check(t, os.WriteFile(r.keys, []byte(line+"\n"), 0600))
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", protocol.ProjectRecoveryCheckpointPath(p.ID, j.OperationID), 200}, {"GET", protocol.ProjectRecoveryCheckpointPath(p.ID, "op-missing"), 409}, {"GET", protocol.ProjectRecoveryCheckpointPath("project-other", j.OperationID), 409}, {"POST", protocol.ProjectRecoveryCheckpointPath(p.ID, j.OperationID), 403}, {"GET", protocol.ProjectRecoveryCheckpointPath(p.ID, j.OperationID) + "/extra", 403}} {
		q := httptest.NewRequest(tc.method, tc.path, nil)
		q = q.WithContext(context.WithValue(q.Context(), peerKey{}, uint32(os.Geteuid())))
		q.Header.Set(access.Header, key)
		q.Header.Set(protocol.Header, "1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, q)
		if w.Code != tc.status || r.failed.Load() {
			t.Fatal(tc, w.Code, w.Body.String(), r.failed.Load())
		}
	}
	if _, err := r.s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", []byte(`{}`), j.OperationID); err != nil {
		t.Fatal(err)
	}
	value, err := r.s.projectRecoveryCheckpoint(context.Background(), p.ID, j.OperationID)
	if err == nil || value.Operation.ID != "" {
		t.Fatal("corrupt mirror accepted", value, err)
	}
}
