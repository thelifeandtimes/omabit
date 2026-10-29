//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type queuedAnchorBackend struct{ *queuedProjectBackend }

func (b *queuedAnchorBackend) OpenProjectAnchorStart(ctx context.Context, v runtimequalify.ProjectPreparationSelection, p projectnetwork.Preparation) (projectnetwork.AnchorStartEngine, error) {
	if v.OperationID != p.OperationID || v.ProjectID != p.ProjectID || p.Sequence != 6 {
		b.t.Fatal("Start session lost creation binding")
	}
	b.opened <- v
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fail {
		return nil, errors.New("synthetic setup failure")
	}
	return &storedAnchorEngine{t: b.t, s: b.s, id: v.ProjectID}, nil
}
func anchorQueueFixture(t *testing.T) (*Server, *queuedAnchorBackend, string, projectAnchorStartRequest) {
	t.Helper()
	r, b, id, preq := projectQueueFixture(t)
	prep, _, err := r.s.acceptProjectPreparation("actor", id, preq)
	if err != nil {
		t.Fatal(err)
	}
	completeProjectPreparation(t, r.s, prep)
	backend := &queuedAnchorBackend{b}
	r.b = backend
	return r, backend, id, projectAnchorStartRequest{RequestID: "start-anchor", ExpectedRevision: preq.ExpectedRevision, ReviewDigest: preq.ReviewDigest}
}
func TestProjectAnchorQueueSharesResourceAndUpdateFences(t *testing.T) {
	r, b, id, req := anchorQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectAnchorStart("actor", id, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	replay, hit, err := r.queueProjectAnchorStart("actor", id, req)
	if err != nil || !hit || replay.ID != op.ID {
		t.Fatal(replay, hit, err)
	}
	select {
	case <-b.opened:
		t.Fatal("Start setup bypassed resource gate")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 || !strings.Contains(w.Body.String(), "update_busy") {
		t.Fatal(w.Code, w.Body.String())
	}
	<-r.workGate
	select {
	case v := <-b.opened:
		if v.RunnerID != r.s.runnerID || v.ProjectID != id || v.OperationID == op.ID {
			t.Fatal("creation identity replaced by Start ID", v)
		}
	case <-time.After(time.Second):
		t.Fatal("setup not dispatched")
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
		t.Fatal(w.Code)
	}
	close(b.release)
	waitProjectWorker(t, r)
	actual, err := loadOperation(r.s.db, op.ID)
	if err != nil || actual.Status != "succeeded" {
		t.Fatal(actual, err)
	}
	select {
	case <-b.opened:
		t.Fatal("duplicate dispatched")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "second"
	_, _, err = r.queueProjectAnchorStart("actor", id, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectAnchorQueueFailuresKeepOriginalJournal(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, id, req := anchorQueueFixture(t)
			if mode == "cancel-waiting" {
				r.workGate <- struct{}{}
			}
			if mode == "retained-maintenance" {
				if _, _, err := r.s.register("actor", registration("legacy", "legacy", 18580)); err != nil {
					t.Fatal(err)
				}
				v, err := loadInstance(r.s.db, "legacy")
				if err != nil {
					t.Fatal(err)
				}
				v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, runtimequalify.ContainerRecord{Kind: "pack", CreateUncertain: true})
				tx, err := r.s.db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if err = saveInstance(tx, v); err != nil {
					tx.Rollback()
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			op, _, err := r.queueProjectAnchorStart("actor", id, req)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "cancel-waiting":
				r.cancel()
			case "cancel-setup", "setup-failure":
				select {
				case <-b.opened:
				case <-time.After(time.Second):
					t.Fatal("setup not reached")
				}
				if mode == "cancel-setup" {
					r.cancel()
				} else {
					b.fail = true
					close(b.release)
				}
			}
			waitProjectWorker(t, r)
			receipt, err := loadOperation(r.s.db, op.ID)
			if err != nil || receipt.Status != "interrupted" {
				t.Fatal(receipt, err)
			}
			state, err := loadProjectAnchorStart(r.s.db, id)
			if err != nil || state.Phase != "planned" {
				t.Fatal("setup failure advanced Start", state, err)
			}
			replay, hit, err := r.s.acceptProjectAnchorStart("actor", id, req)
			if err != nil || !hit || replay.ID != op.ID || replay.Status != "interrupted" {
				t.Fatal(replay, hit, err)
			}
			r.mu.Lock()
			workers := r.updateWorkers
			r.mu.Unlock()
			if workers != 0 || r.failed.Load() {
				t.Fatal("worker leaked fence or failed store", workers)
			}
		})
	}
}
func TestProjectAnchorQueueRefusesUnavailableRunner(t *testing.T) {
	for _, mode := range []string{"closing", "failed", "cancelled", "unsupported"} {
		t.Run(mode, func(t *testing.T) {
			r, _, id, req := anchorQueueFixture(t)
			switch mode {
			case "closing":
				r.closing = true
			case "failed":
				r.failed.Store(true)
			case "cancelled":
				r.cancel()
			case "unsupported":
				r.b = &fakeBackend{}
			}
			if _, _, err := r.queueProjectAnchorStart("actor", id, req); err == nil {
				t.Fatal("unavailable runner admitted Start")
			}
			var count int
			if err := r.s.db.QueryRow("SELECT count(*) FROM project_anchor_starts").Scan(&count); err != nil || count != 0 {
				t.Fatal("refusal accepted work", count, err)
			}
		})
	}
}
