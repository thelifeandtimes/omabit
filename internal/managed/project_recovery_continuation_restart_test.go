//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// Record only retained DB bytes. Failed transactions must preserve all parent,
// child, continuation and original observation rows, not merely their statuses.
func continuationDatabaseSnapshot(t *testing.T, s *store) map[string][][]string {
	t.Helper()
	out := map[string][][]string{}
	for table, query := range map[string]string{
		"operations":    "SELECT id,body,coalesce(runtime,'') FROM operations ORDER BY id",
		"parents":       "SELECT operation_id,body FROM project_anchor_recoveries ORDER BY operation_id",
		"continuations": "SELECT operation_id,body FROM project_recovery_continuations ORDER BY operation_id",
		"observations":  "SELECT operation_id,body FROM project_recovery_observations ORDER BY operation_id",
	} {
		rows, err := s.db.Query(query)
		g02Check(t, err)
		columns, err := rows.Columns()
		g02Check(t, err)
		for rows.Next() {
			values := make([]string, len(columns))
			scan := make([]any, len(columns))
			for n := range values {
				scan[n] = &values[n]
			}
			g02Check(t, rows.Scan(scan...))
			out[table] = append(out[table], values)
		}
		g02Check(t, rows.Err())
		g02Check(t, rows.Close())
	}
	return out
}
func continuationNextDrainDispatch(t *testing.T, b *continuationQueueBackend, parent projectAnchorRecoveryJob) projectnetwork.AnchorRecoveryJournal {
	t.Helper()
	d := &queuedRecoveryDriver{b: b.queuedRecoveryBackend, index: 1}
	before, err := d.InspectDrain(context.Background(), *parent.Journal.Drain)
	g02Check(t, err)
	next, err := parent.Journal.Drain.Advance(parent.Source, 1, parent.Journal.Drain.Sequence, "stop-dispatched", &before)
	g02Check(t, err)
	return projectnetwork.AnchorRecoveryJournal{Drain: &next}
}
func TestRecoveryContinuationRestartPreservesCheckpointWithoutEffects(t *testing.T) {
	for _, cut := range []string{"queued", "begun", "dispatched"} {
		t.Run(cut, func(t *testing.T) {
			r, b, p, parent := prepareRecoveryContinuation(t, "signal")
			calls := append([]string(nil), b.calls...)
			req := observationRequest(p, parent, "continue-before-restart")
			op, _, err := r.s.acceptProjectRecoveryContinuation("second-operator", p.ID, req)
			g02Check(t, err)
			if cut != "queued" {
				_, err = r.s.beginProjectRecoveryContinuation(op.ID)
				g02Check(t, err)
			}
			if cut == "dispatched" {
				parent, err = r.s.advanceProjectRecoveryContinuation(op.ID, parent.Journal.Sequence, "draining", continuationNextDrainDispatch(t, b, parent))
				g02Check(t, err)
			}
			saved, err := loadProjectRecoveryContinuation(r.s.db, op.ID)
			g02Check(t, err)
			directory := r.s.directory
			g02Check(t, r.s.close())
			reopened, err := testOpenStore(directory)
			g02Check(t, err)
			r.s = reopened
			b.s = reopened
			g02Check(t, reopened.reconcileInterrupted())
			first := continuationDatabaseSnapshot(t, reopened)
			g02Check(t, reopened.reconcileInterrupted())
			if !reflect.DeepEqual(first, continuationDatabaseSnapshot(t, reopened)) {
				t.Fatal("repeated startup rewrote retained receipts")
			}
			kept, err := loadProjectRecoveryContinuation(reopened.db, op.ID)
			g02Check(t, err)
			owner, err := loadOperation(reopened.db, parent.OperationID)
			g02Check(t, err)
			child, err := loadOperation(reopened.db, op.ID)
			g02Check(t, err)
			actual, err := loadProjectAnchorRecovery(reopened.db, parent.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(saved, kept) || !reflect.DeepEqual(parent, actual) || owner.Status != "interrupted" || child.Status != "interrupted" {
				t.Fatal("restart lost atomic parent/continuation state")
			}
			replay, hit, err := r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(replay, child) {
				t.Fatal("restart replay changed receipt")
			}
			if err = r.executeProjectRecoveryContinuation(b, replay); err == nil {
				t.Fatal("interrupted continuation dispatched automatically")
			}
			waitRecoveryQueue(t, r)
			if !reflect.DeepEqual(calls, b.calls) {
				t.Fatal("restart or replay repeated runtime effects")
			}
			g02Check(t, validateProjectRecoveryContinuations(reopened.db))
		})
	}
}
func TestRecoveryContinuationCommitsRollbackAndRefuseOrdinaryCallbacks(t *testing.T) {
	r, b, p, parent := prepareRecoveryContinuation(t, "signal")
	op, _, err := r.s.acceptProjectRecoveryContinuation("second-operator", p.ID, observationRequest(p, parent, "continue-atomic"))
	g02Check(t, err)
	before := continuationDatabaseSnapshot(t, r.s)
	_, err = r.s.db.Exec("CREATE TRIGGER fail_continuation_begin BEFORE UPDATE ON operations WHEN NEW.id='" + op.ID + "' BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	g02Check(t, err)
	if _, err = r.s.beginProjectRecoveryContinuation(op.ID); err == nil {
		t.Fatal("begin ignored failed child commit")
	}
	if !reflect.DeepEqual(before, continuationDatabaseSnapshot(t, r.s)) {
		t.Fatal("begin partially changed parent or creators")
	}
	_, err = r.s.db.Exec("DROP TRIGGER fail_continuation_begin")
	g02Check(t, err)
	_, err = r.s.beginProjectRecoveryContinuation(op.ID)
	g02Check(t, err)
	update := continuationNextDrainDispatch(t, b, parent)
	before = continuationDatabaseSnapshot(t, r.s)
	if _, err = r.s.advanceProjectAnchorRecovery(parent.OperationID, parent.Journal.Sequence, "draining", update); err == nil {
		t.Fatal("ordinary callback bypassed continuation owner")
	}
	if err = r.s.interruptAnchorRecovery(parent.OperationID, false); err == nil {
		t.Fatal("ordinary interruption bypassed continuation owner")
	}
	_, err = r.s.db.Exec("CREATE TRIGGER fail_continuation_advance BEFORE UPDATE ON project_recovery_continuations BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	g02Check(t, err)
	if _, err = r.s.advanceProjectRecoveryContinuation(op.ID, parent.Journal.Sequence, "draining", update); err == nil {
		t.Fatal("advance ignored failed Latest commit")
	}
	if !reflect.DeepEqual(before, continuationDatabaseSnapshot(t, r.s)) {
		t.Fatal("advance partially changed parent or creators")
	}
	_, err = r.s.db.Exec("DROP TRIGGER fail_continuation_advance")
	g02Check(t, err)
	_, err = r.s.advanceProjectRecoveryContinuation(op.ID, parent.Journal.Sequence, "draining", update)
	g02Check(t, err)
	g02Check(t, r.s.interruptRecoveryContinuation(op.ID))
	g02Check(t, validateProjectRecoveryContinuations(r.s.db))
}
func TestRecoveryContinuationRejectsChangedCheckpointAndOwnership(t *testing.T) {
	for _, fault := range []string{"parent-mirror", "continuation-mirror", "missing-journal", "actor", "target", "parent-before", "before", "latest"} {
		t.Run(fault, func(t *testing.T) {
			r, _, p, parent := prepareRecoveryContinuation(t, "signal")
			op, _, err := r.s.acceptProjectRecoveryContinuation("second-operator", p.ID, observationRequest(p, parent, "continue-ownership"))
			g02Check(t, err)
			j, err := r.s.beginProjectRecoveryContinuation(op.ID)
			g02Check(t, err)
			switch fault {
			case "parent-mirror":
				_, err = r.s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", parent.OperationID)
			case "continuation-mirror":
				_, err = r.s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "missing-journal":
				_, err = r.s.db.Exec("DELETE FROM project_recovery_continuations WHERE operation_id=?", op.ID)
			case "actor":
				_, err = r.s.db.Exec("UPDATE operations SET body=json_set(body,'$.actor','foreign') WHERE id=?", op.ID)
			default:
				switch fault {
				case "target":
					j.Target.InstanceID = "foreign"
				case "parent-before":
					j.ParentBefore.Actor = "foreign"
				case "before":
					j.Before.Sequence++
				case "latest":
					j.Latest.Sequence++
				}
				raw, e := json.Marshal(j)
				g02Check(t, e)
				_, err = r.s.db.Exec("UPDATE project_recovery_continuations SET body=? WHERE operation_id=?", raw, op.ID)
				g02Check(t, err)
				_, err = r.s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID)
			}
			g02Check(t, err)
			if err = validateProjectRecoveryContinuations(r.s.db); err == nil {
				t.Fatal("changed ownership accepted", fault)
			}
			if err = r.s.reconcileInterrupted(); err == nil {
				t.Fatal("restart repaired invalid evidence", fault)
			}
		})
	}
}
func TestRecoveryContinuationAdmissionAndCancellation(t *testing.T) {
	for _, fault := range []string{"stale-sequence", "stale-digest", "quiesced", "unsupported", "cancel", "reconcile", "uncertain"} {
		t.Run(fault, func(t *testing.T) {
			r, b, p, parent := prepareRecoveryContinuation(t, "signal")
			if fault == "uncertain" {
				// The next signal intent exists but its outcome has not been observed.
				op, _, err := r.s.acceptProjectRecoveryContinuation("other", p.ID, observationRequest(p, parent, "first"))
				g02Check(t, err)
				_, err = r.s.beginProjectRecoveryContinuation(op.ID)
				g02Check(t, err)
				parent, err = r.s.advanceProjectRecoveryContinuation(op.ID, parent.Journal.Sequence, "draining", continuationNextDrainDispatch(t, b, parent))
				g02Check(t, err)
				g02Check(t, r.s.interruptRecoveryContinuation(op.ID))
			}
			calls := append([]string(nil), b.calls...)
			req := observationRequest(p, parent, "new-continuation")
			switch fault {
			case "stale-sequence":
				req.ExpectedSequence++
			case "stale-digest":
				req.CheckpointDigest = strings.Repeat("a", 64)
			case "quiesced":
				if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
					t.Fatal("fixture failed to quiesce")
				}
			case "unsupported":
				r.b = b.queuedRecoveryBackend
			}
			r.workGate <- struct{}{}
			op, _, err := r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
			if fault == "cancel" || fault == "reconcile" {
				g02Check(t, err)
				if fault == "cancel" {
					r.cancel()
				} else {
					g02Check(t, r.s.reconcileInterrupted())
				}
				<-r.workGate
				waitRecoveryQueue(t, r)
				got, err := loadOperation(r.s.db, op.ID)
				g02Check(t, err)
				if got.Status != "interrupted" || r.failed.Load() {
					t.Fatal("cancelled queue lost retained intent")
				}
			} else {
				<-r.workGate
				if err == nil {
					t.Fatal("invalid continuation admitted", fault)
				}
			}
			if !reflect.DeepEqual(calls, b.calls) {
				t.Fatal("rejected work reached runtime", fault)
			}
		})
	}
}

func TestRecoveryContinuationCannotAdoptUnknownCreate(t *testing.T) {
	r, base, p, parent := interruptedObservationFixture(t, "create-unknown")
	b := &continuationQueueBackend{&queuedRecoveryObservationBackend{queuedRecoveryBackend: base}}
	r.b = b
	calls := append([]string(nil), base.calls...)
	member := parent.Journal.Replacement.Members[0]
	if member.Phase != "create-dispatched" || member.Container.ID != "" {
		t.Fatal("fixture did not retain unknown Create")
	}
	before := continuationDatabaseSnapshot(t, r.s)
	if _, _, err := r.queueProjectRecoveryContinuation("other-operator", p.ID, observationRequest(p, parent, "continue-unknown-create")); err == nil {
		t.Fatal("unknown Create gained continuation authority")
	}
	waitRecoveryQueue(t, r)
	if !reflect.DeepEqual(calls, base.calls) || !reflect.DeepEqual(before, continuationDatabaseSnapshot(t, r.s)) {
		t.Fatal("unknown Create was adopted or replayed")
	}
}
