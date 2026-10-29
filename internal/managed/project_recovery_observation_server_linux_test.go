//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type queuedRecoveryObservationBackend struct {
	*queuedRecoveryBackend
	opens   int
	failure bool
}

func (b *queuedRecoveryObservationBackend) OpenProjectRecoveryObservation(ctx context.Context, s projectRecoveryObservationSelection) (projectnetwork.AnchorRecoveryObservers, error) {
	b.opens++
	// This mock exposes only read operations, like the real factory wrappers.
	var phase string
	g02Check(b.t, b.s.db.QueryRow("SELECT json_extract(body,'$.phase') FROM project_recovery_observations ORDER BY rowid DESC LIMIT 1").Scan(&phase))
	if phase != "observing" {
		b.t.Fatal("observation opened before durable begin")
	}
	j, err := loadProjectAnchorRecovery(b.s.db, b.id)
	g02Check(b.t, err)
	if !reflect.DeepEqual(s.Checkpoint, j.Journal) || !reflect.DeepEqual(s.History, j.Source) {
		b.t.Fatal("factory authority differs from parent")
	}
	target, err := projectnetwork.RecoveryCheckpointObservationTarget(s.History, s.Checkpoint)
	g02Check(b.t, err)
	if target != s.Target {
		b.t.Fatal("factory target changed")
	}
	if b.failure {
		return projectnetwork.AnchorRecoveryObservers{}, errors.New("synthetic read failure")
	}
	if err := ctx.Err(); err != nil {
		return projectnetwork.AnchorRecoveryObservers{}, err
	}
	d := &storeRecoveryObserver{&queuedRecoveryDriver{b: b.queuedRecoveryBackend, index: 0}}
	return projectnetwork.AnchorRecoveryObservers{
		Drain: struct {
			projectnetwork.AnchorDrainObserver
		}{d}, Namespace: struct {
			projectnetwork.AnchorNamespaceObserver
		}{d},
		Retirement: struct {
			projectnetwork.AnchorRetirementObserver
		}{d}, Replacement: struct {
			projectnetwork.AnchorReplacementObserver
		}{d}, Readiness: d,
	}, nil
}
func TestRecoveryObservationQueueSharesFenceAndNeverReplaysEffects(t *testing.T) {
	for _, where := range []string{"signal", "remove", "start", "readiness"} {
		t.Run(where, func(t *testing.T) {
			r, base, p, parent := interruptedObservationFixture(t, where)
			b := &queuedRecoveryObservationBackend{queuedRecoveryBackend: base}
			r.b = b
			calls := append([]string(nil), base.calls...)
			r.workGate <- struct{}{}
			req := observationRequest(p, parent, "queue-observation")
			op, hit, err := r.queueProjectRecoveryObservation("observer", p.ID, req)
			g02Check(t, err)
			if hit {
				t.Fatal("unexpected replay")
			}
			replay, hit, err := r.queueProjectRecoveryObservation("observer", p.ID, req)
			g02Check(t, err)
			if !hit || replay.ID != op.ID {
				t.Fatal("queued replay changed")
			}
			if b.opens != 0 {
				t.Fatal("read crossed worker gate")
			}
			if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
				t.Fatal("update crossed worker", w.Code)
			}
			<-r.workGate
			waitRecoveryQueue(t, r)
			got, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if got.Status != "succeeded" || b.opens != 1 || r.failed.Load() {
				t.Fatal("observation did not complete", got.Status, b.opens)
			}
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			if current.Journal.Sequence != parent.Journal.Sequence+1 {
				t.Fatal("worker advanced more than one outcome")
			}
			g02Check(t, r.s.reconcileInterrupted())
			replay, hit, err = r.queueProjectRecoveryObservation("observer", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(got, replay) {
				t.Fatal("completed replay changed")
			}
			if err = r.executeProjectRecoveryObservation(b, got); err == nil {
				t.Fatal("completed observation executed again")
			}
			waitRecoveryQueue(t, r)
			if b.opens != 1 || !reflect.DeepEqual(base.calls, calls) {
				t.Fatal("observation repeated read/effects")
			}
			if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
				t.Fatal("worker retained update fence", w.Code)
			}
			req.RequestID = "quiesced"
			_, _, err = r.queueProjectRecoveryObservation("observer", p.ID, req)
			requireFault(t, err, "management_quiesced")
		})
	}
}
func TestRecoveryObservationQueueFailureAndRestartStayExplicit(t *testing.T) {
	for _, fault := range []string{"read", "cancel", "reconcile", "unsupported"} {
		t.Run(fault, func(t *testing.T) {
			r, base, p, parent := interruptedObservationFixture(t, "signal")
			calls := append([]string(nil), base.calls...)
			b := &queuedRecoveryObservationBackend{queuedRecoveryBackend: base, failure: fault == "read"}
			r.b = b
			req := observationRequest(p, parent, "observe-interrupted")
			if fault == "unsupported" {
				r.b = base
				if _, _, err := r.queueProjectRecoveryObservation("observer", p.ID, req); err == nil {
					t.Fatal("missing observer backend accepted")
				}
				return
			}
			r.workGate <- struct{}{}
			op, _, err := r.queueProjectRecoveryObservation("observer", p.ID, req)
			g02Check(t, err)
			if fault == "cancel" {
				r.cancel()
			}
			if fault == "reconcile" {
				g02Check(t, r.s.reconcileInterrupted())
			}
			<-r.workGate
			waitRecoveryQueue(t, r)
			got, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if got.Status != "interrupted" || r.failed.Load() {
				t.Fatal("failed read not preserved", got.Status)
			}
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			if !reflect.DeepEqual(current, parent) || !reflect.DeepEqual(base.calls, calls) {
				t.Fatal("failed read changed parent or effects")
			}
			if fault != "read" && b.opens != 0 {
				t.Fatal("old/cancelled work opened observer")
			}
			if err = r.executeProjectRecoveryObservation(b, got); err == nil {
				t.Fatal("interrupted observation automatically resumed")
			}
			if fault == "read" {
				b.failure = false
				replay, hit, err := r.queueProjectRecoveryObservation("observer", p.ID, req)
				g02Check(t, err)
				if !hit || replay.Status != "interrupted" || b.opens != 1 {
					t.Fatal("replay reopened failed read")
				}
				req.RequestID = "explicit-fresh-read"
				next, hit, err := r.queueProjectRecoveryObservation("observer", p.ID, req)
				g02Check(t, err)
				if hit {
					t.Fatal("fresh read was replay")
				}
				waitRecoveryQueue(t, r)
				got, err = loadOperation(r.s.db, next.ID)
				g02Check(t, err)
				if got.Status != "succeeded" || b.opens != 2 || !reflect.DeepEqual(base.calls, calls) {
					t.Fatal("fresh explicit read failed")
				}
			}
		})
	}
}
