//go:build linux && amd64

package managed

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type continuationQueueBackend struct {
	*queuedRecoveryObservationBackend
}

func (b *continuationQueueBackend) OpenProjectCreatedAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (projectnetwork.AnchorReplacementStartDriver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	j, err := loadProjectAnchorRecovery(b.s.db, b.id)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(*j.Journal.Replacement, s.Replacement) {
		b.t.Fatal("created continuation differs from exact parent")
	}
	index := 0
	for s.Replacement.Members[index].Phase == "running" {
		index++
	}
	return struct {
		projectnetwork.AnchorReplacementStartDriver
	}{&queuedRecoveryDriver{b: b.queuedRecoveryBackend, index: index}}, nil
}
func prepareRecoveryContinuation(t *testing.T, where string) (*Server, *continuationQueueBackend, storedProject, projectAnchorRecoveryJob) {
	t.Helper()
	r, base, p, parent := interruptedObservationFixture(t, where)
	b := &continuationQueueBackend{&queuedRecoveryObservationBackend{queuedRecoveryBackend: base}}
	r.b = b
	if where != "create" {
		op, _, err := r.s.acceptProjectRecoveryObservation("observer", p.ID, observationRequest(p, parent, "resolve-once"))
		g02Check(t, err)
		j, err := r.s.beginProjectRecoveryObservation(op.ID)
		g02Check(t, err)
		after := observedRecoveryResult(t, base, parent)
		g02Check(t, r.s.completeProjectRecoveryObservation(op.ID, j.Before.Sequence, after))
		parent, err = loadProjectAnchorRecovery(r.s.db, parent.OperationID)
		g02Check(t, err)
	}
	return r, b, p, parent
}
func TestRecoveryContinuationCompletesWithoutRepeatedEffects(t *testing.T) {
	for _, where := range []string{"signal", "remove", "start", "create"} {
		t.Run(where, func(t *testing.T) {
			r, b, p, parent := prepareRecoveryContinuation(t, where)
			before, err := loadOperation(r.s.db, parent.OperationID)
			g02Check(t, err)
			req := observationRequest(p, parent, "continue-explicitly")
			r.workGate <- struct{}{}
			op, hit, err := r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
			g02Check(t, err)
			if hit {
				t.Fatal("new continuation replayed")
			}
			replay, hit, err := r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(op, replay) {
				t.Fatal("queued replay changed")
			}
			if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
				t.Fatal("management crossed worker")
			}
			<-r.workGate
			waitRecoveryQueue(t, r)
			current, err := loadProjectAnchorRecovery(r.s.db, parent.OperationID)
			g02Check(t, err)
			owner, err := loadOperation(r.s.db, parent.OperationID)
			g02Check(t, err)
			done, err := loadOperation(r.s.db, op.ID)
			g02Check(t, err)
			if owner.Status != "succeeded" || done.Status != "succeeded" || current.Journal.Phase != "complete" || r.failed.Load() {
				t.Fatal("explicit continuation did not finish", owner.Status, done.Status, current.Journal.Phase)
			}
			j, err := loadProjectRecoveryContinuation(r.s.db, op.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(j.ParentBefore, before) || !reflect.DeepEqual(j.Before, parent.Journal) || !reflect.DeepEqual(*j.Latest, current.Journal) || owner.Actor != before.Actor || owner.RequestID != before.RequestID {
				t.Fatal("continuation changed original attribution or lost checkpoint")
			}
			counts := map[string]int{}
			for _, call := range b.calls {
				counts[call]++
			}
			for _, effect := range []string{"signal", "remove", "create", "start"} {
				if counts[effect] != 2 {
					t.Fatal("effect repeated or missing", where, effect, b.calls)
				}
			}
			g02Check(t, validateProjectRecoveryContinuations(r.s.db))
			g02Check(t, validateProjectRecoveryObservations(r.s.db))
			g02Check(t, validateProjectAnchorRecoveries(r.s.db))
			calls := append([]string(nil), b.calls...)
			replay, hit, err = r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
			g02Check(t, err)
			if !hit || !reflect.DeepEqual(done, replay) {
				t.Fatal("completed replay changed")
			}
			if err = r.executeProjectRecoveryContinuation(b, done); err == nil {
				t.Fatal("completed continuation executed again")
			}
			waitRecoveryQueue(t, r)
			if !reflect.DeepEqual(calls, b.calls) {
				t.Fatal("completed replay dispatched effects")
			}
		})
	}
}
func TestRecoveryContinuationSecondObservationAndFreshContinuation(t *testing.T) {
	r, b, p, parent := prepareRecoveryContinuation(t, "signal")
	b.fault = "remove"
	req := observationRequest(p, parent, "continue-until-uncertain")
	op, _, err := r.queueProjectRecoveryContinuation("second-operator", p.ID, req)
	g02Check(t, err)
	waitRecoveryQueue(t, r)
	first, err := loadProjectRecoveryContinuation(r.s.db, op.ID)
	g02Check(t, err)
	firstOp, err := loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	parent, err = loadProjectAnchorRecovery(r.s.db, parent.OperationID)
	g02Check(t, err)
	if firstOp.Status != "interrupted" || first.Latest == nil || !reflect.DeepEqual(*first.Latest, parent.Journal) {
		t.Fatal("second uncertainty lost")
	}
	if _, _, err = r.s.acceptProjectRecoveryContinuation("second-operator", p.ID, observationRequest(p, parent, "must-observe-first")); err == nil {
		t.Fatal("uncertain removal admitted new effects")
	}
	b.fault = ""
	observed, _, err := r.queueProjectRecoveryObservation("observer", p.ID, observationRequest(p, parent, "observe-removal"))
	g02Check(t, err)
	waitRecoveryQueue(t, r)
	read, err := loadOperation(r.s.db, observed.ID)
	g02Check(t, err)
	if read.Status != "succeeded" {
		t.Fatal("fresh observation failed")
	}
	parent, err = loadProjectAnchorRecovery(r.s.db, parent.OperationID)
	g02Check(t, err)
	next, _, err := r.queueProjectRecoveryContinuation("third-operator", p.ID, observationRequest(p, parent, "continue-after-observation"))
	g02Check(t, err)
	waitRecoveryQueue(t, r)
	done, err := loadOperation(r.s.db, next.ID)
	g02Check(t, err)
	if done.Status != "succeeded" {
		t.Fatal("second continuation failed")
	}
	g02Check(t, validateProjectRecoveryContinuations(r.s.db))
	g02Check(t, validateProjectRecoveryObservations(r.s.db))
	retained, err := loadProjectRecoveryContinuation(r.s.db, first.OperationID)
	g02Check(t, err)
	if !reflect.DeepEqual(retained, first) {
		t.Fatal("earlier continuation rewritten")
	}
	counts := map[string]int{}
	for _, call := range b.calls {
		counts[call]++
	}
	for _, effect := range []string{"signal", "remove", "create", "start"} {
		if counts[effect] != 2 {
			t.Fatal("uncertain effect repeated", effect, b.calls)
		}
	}
}
