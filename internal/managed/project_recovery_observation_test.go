//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func observationRequest(p storedProject, j projectAnchorRecoveryJob, id string) projectRecoveryObservationRequest {
	digest, _ := p.Plan.Digest()
	return projectRecoveryObservationRequest{RequestID: id, RecoveryOperationID: j.OperationID, ExpectedRevision: p.Revision, ExpectedSequence: j.Journal.Sequence, ReviewDigest: digest, HistoryDigest: recoveryHistoryDigest(j.Source), CheckpointDigest: recoveryCheckpointDigest(j.Journal)}
}
func interruptedObservationFixture(t *testing.T, where string) (*Server, *queuedRecoveryBackend, storedProject, projectAnchorRecoveryJob) {
	t.Helper()
	r, b, p, req := recoveryQueueFixture(t)
	r.workGate <- struct{}{}
	op, _, err := r.queueProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	prepareQueuedRecoveryReference(t, r, b, op.ID)
	b.fault = where
	close(b.release)
	<-r.workGate
	waitRecoveryQueue(t, r)
	j, err := loadProjectAnchorRecovery(r.s.db, op.ID)
	g02Check(t, err)
	got, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	if got.Status != "interrupted" {
		t.Fatal("fixture lacks retained uncertainty", where, got.Status)
	}
	b.fault = ""
	return r, b, p, j
}

type storeRecoveryObserver struct{ *queuedRecoveryDriver }

func (d *storeRecoveryObserver) InspectDrain(ctx context.Context, s projectnetwork.AnchorDrain) (projectnetwork.AnchorDrainObservation, error) {
	return d.AwaitDrainExit(ctx, s, d.index)
}
func observationDrivers(b *queuedRecoveryBackend) projectnetwork.AnchorRecoveryObservers {
	d := &storeRecoveryObserver{&queuedRecoveryDriver{b: b, index: 0}}
	return projectnetwork.AnchorRecoveryObservers{Drain: d, Namespace: d, Retirement: d, Replacement: d, Readiness: d}
}
func observedRecoveryResult(t *testing.T, b *queuedRecoveryBackend, j projectAnchorRecoveryJob) projectnetwork.AnchorRecoveryJournal {
	t.Helper()
	result, err := projectnetwork.ObserveAnchorRecoveryCheckpoint(context.Background(), j.Source, j.Journal, observationDrivers(b), func(uint64, projectnetwork.AnchorRecoveryJournal) error { return nil }, func() time.Time { return time.Now().UTC() })
	g02Check(t, err)
	return result
}
func TestRecoveryObservationStoreAtomicCompletionAndReplay(t *testing.T) {
	for _, where := range []string{"signal", "remove", "start", "readiness"} {
		t.Run(where, func(t *testing.T) {
			r, b, p, parent := interruptedObservationFixture(t, where)
			calls := append([]string(nil), b.calls...)
			req := observationRequest(p, parent, "observe-uncertain")
			op, hit, err := r.s.acceptProjectRecoveryObservation("other-operator", p.ID, req)
			g02Check(t, err)
			if hit {
				t.Fatal("unexpected replay")
			}
			replay, hit, err := r.s.acceptProjectRecoveryObservation("other-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(op, replay) {
				t.Fatal("actor replay changed")
			}
			changed := req
			changed.RequestID = "overlap"
			if _, _, err = r.s.acceptProjectRecoveryObservation("other-operator", p.ID, changed); err == nil {
				t.Fatal("overlapping observation admitted")
			}
			j, err := r.s.beginProjectRecoveryObservation(op.ID)
			g02Check(t, err)
			after := observedRecoveryResult(t, b, parent)
			if err = r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence+1, after); err == nil {
				t.Fatal("stale sequence admitted")
			}
			g02Check(t, r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence, after))
			g02Check(t, validateProjectRecoveryObservations(r.s.db))
			g02Check(t, validateProjectAnchorRecoveries(r.s.db))
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(current.Source, parent.Source) || !reflect.DeepEqual(current.Journal, after) {
				t.Fatal("observation rewrote source or outcome")
			}
			owner, err := loadOperation(r.s.db, parent.OperationID)
			g02Check(t, err)
			if where == "readiness" {
				if owner.Status != "succeeded" || projectRecoveryClear(r.s.db, p.ID, "") != nil {
					t.Fatal("complete readiness did not release fence")
				}
			} else if owner.Status != "interrupted" || projectRecoveryClear(r.s.db, p.ID, "") == nil {
				t.Fatal("one observation released unresolved recovery")
			}
			completed, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			replay, hit, err = r.s.acceptProjectRecoveryObservation("other-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(replay, completed) {
				t.Fatal("completion replay changed")
			}
			if err = r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence, after); err == nil {
				t.Fatal("completion repeated")
			}
			if !reflect.DeepEqual(calls, b.calls) {
				t.Fatal("observation dispatched runtime effects")
			}
			directory := r.s.directory
			g02Check(t, r.s.close())
			reopened, err := testOpenStore(directory)
			g02Check(t, err)
			t.Cleanup(func() { _ = reopened.close() })
			r.s = reopened
			b.s = reopened
			g02Check(t, reopened.reconcileInterrupted())
			g02Check(t, validateProjectRecoveryObservations(reopened.db))
			replay, hit, err = reopened.acceptProjectRecoveryObservation("other-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(replay, completed) {
				t.Fatal("reopen lost observation")
			}
		})
	}
}
func TestRecoveryObservationStoreRestartAndUnknownCreate(t *testing.T) {
	for _, begin := range []bool{false, true} {
		t.Run(map[bool]string{false: "planned", true: "observing"}[begin], func(t *testing.T) {
			r, b, p, parent := interruptedObservationFixture(t, "signal")
			req := observationRequest(p, parent, "inspect")
			op, _, err := r.s.acceptProjectRecoveryObservation("actor", p.ID, req)
			g02Check(t, err)
			if begin {
				_, err = r.s.beginProjectRecoveryObservation(op.ID)
				g02Check(t, err)
			}
			before := parent.Journal.Clone()
			calls := append([]string(nil), b.calls...)
			g02Check(t, r.s.reconcileInterrupted())
			got, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if got.Status != "interrupted" {
				t.Fatal("observation not interrupted")
			}
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(current.Journal, before) || !reflect.DeepEqual(b.calls, calls) {
				t.Fatal("restart changed recovery or replayed effects")
			}
			if _, err = r.s.beginProjectRecoveryObservation(op.ID); err == nil {
				t.Fatal("restart resumed old observation")
			}
			req.RequestID = "fresh-observation"
			if _, hit, err := r.s.acceptProjectRecoveryObservation("actor", p.ID, req); err != nil || hit {
				t.Fatal("explicit fresh observation refused", err)
			}
		})
	}
	r, _, p, parent := interruptedObservationFixture(t, "create")
	if _, _, err := r.s.acceptProjectRecoveryObservation("actor", p.ID, observationRequest(p, parent, "unknown-create")); err == nil {
		t.Fatal("unknown Create adopted by observation")
	}
}
func TestRecoveryObservationStoreRejectsChangedAuthorityAndRollback(t *testing.T) {
	for _, fault := range []string{"mirror", "parent", "source", "checkpoint", "actor", "orphan", "rollback"} {
		t.Run(fault, func(t *testing.T) {
			r, b, p, parent := interruptedObservationFixture(t, "signal")
			req := observationRequest(p, parent, "inspect")
			op, _, err := r.s.acceptProjectRecoveryObservation("observer", p.ID, req)
			g02Check(t, err)
			j, err := r.s.beginProjectRecoveryObservation(op.ID)
			g02Check(t, err)
			switch fault {
			case "mirror":
				_, err = r.s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", []byte("{}"), op.ID)
			case "parent":
				j.RecoveryOperationID = "op-foreign"
			case "source":
				j.SourceDigest = "changed"
			case "checkpoint":
				j.Before.Sequence++
			case "actor":
				_, err = r.s.db.Exec("UPDATE operations SET actor='changed' WHERE id=?", op.ID)
			case "orphan":
				_, err = r.s.db.Exec("DELETE FROM project_recovery_observations WHERE operation_id=?", op.ID)
			case "rollback":
				_, err = r.s.db.Exec("CREATE TRIGGER reject_recovery_child BEFORE UPDATE OF runtime ON operations WHEN NEW.id='" + parent.Journal.IDs.Members[0].OperationID + "' BEGIN SELECT RAISE(ABORT,'synthetic child mirror failure'); END;")
			}
			g02Check(t, err)
			if fault == "parent" || fault == "source" || fault == "checkpoint" {
				raw, err := json.Marshal(j)
				g02Check(t, err)
				_, err = r.s.db.Exec("UPDATE project_recovery_observations SET body=? WHERE operation_id=?", raw, op.ID)
				g02Check(t, err)
				_, err = r.s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID)
				g02Check(t, err)
			}
			if fault != "rollback" {
				if validateProjectRecoveryObservations(r.s.db) == nil {
					t.Fatal("changed observation authority accepted")
				}
				return
			}
			after := observedRecoveryResult(t, b, parent)
			if err = r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence, after); err == nil {
				t.Fatal("partial mirror commit accepted")
			}
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(current, parent) {
				t.Fatal("failed transaction changed parent")
			}
			currentObservation, err := loadProjectRecoveryObservation(r.s.db, op.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(currentObservation, j) {
				t.Fatal("failed transaction changed observer")
			}
			_, err = r.s.db.Exec("DROP TRIGGER reject_recovery_child")
			g02Check(t, err)
			g02Check(t, validateProjectRecoveryObservations(r.s.db))
			g02Check(t, r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence, after))
		})
	}
}
