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

type queuedProjectBackend struct {
	fakeBackend
	t       *testing.T
	s       *store
	opened  chan runtimequalify.ProjectPreparationSelection
	release chan struct{}
	fail    bool
}

func (b *queuedProjectBackend) OpenProjectPreparation(ctx context.Context, v runtimequalify.ProjectPreparationSelection) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
	b.opened <- v
	select {
	case <-b.release:
	case <-ctx.Done():
		return projectnetwork.Preparation{}, nil, ctx.Err()
	}
	if b.fail {
		return projectnetwork.Preparation{}, nil, errors.New("synthetic setup failure")
	}
	p := projectPreparationFixture(b.t, b.s, v.ProjectID, v.OperationID)
	return p, &storedPreparationEngine{t: b.t, s: b.s, id: v.ProjectID}, nil
}
func projectQueueFixture(t *testing.T) (*Server, *queuedProjectBackend, string, projectPrepareRequest) {
	t.Helper()
	s := testStore(t)
	b := &queuedProjectBackend{t: t, s: s, opened: make(chan runtimequalify.ProjectPreparationSelection, 2), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), executableSHA256: strings.Repeat("f", 64)}
	t.Cleanup(func() { cancel(); r.wg.Wait() })
	reg, _, err := s.registerProject("actor", projectRequest(t, "register", "queued", 18481))
	if err != nil {
		t.Fatal(err)
	}
	p, err := loadProject(s.db, reg.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	return r, b, p.ID, projectPrepareRequest{RequestID: "prepare", ExpectedRevision: 1, ReviewDigest: digest}
}
func waitProjectWorker(t *testing.T, r *Server) {
	t.Helper()
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("project worker did not finish")
	}
}
func TestProjectQueueSharesHostGateAndManagementFence(t *testing.T) {
	r, b, id, req := projectQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectPreparation("actor", id, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	if replay, hit, err := r.queueProjectPreparation("actor", id, req); err != nil || !hit || replay.ID != op.ID {
		t.Fatal(replay, hit, err)
	}
	select {
	case <-b.opened:
		t.Fatal("setup bypassed host gate")
	default:
	}
	w := rc11Fence(r, "acquire", strings.Repeat("a", 64))
	if w.Code != 409 || !strings.Contains(w.Body.String(), "update_busy") {
		t.Fatal(w.Code, w.Body.String())
	}
	<-r.workGate
	select {
	case v := <-b.opened:
		if v.RunnerID != r.s.runnerID || v.ProjectID != id || v.OperationID != op.ID || v.Revision != 1 || v.Generation != 1 {
			t.Fatal(v)
		}
	case <-time.After(time.Second):
		t.Fatal("setup never dispatched")
	}
	w = rc11Fence(r, "acquire", strings.Repeat("a", 64))
	if w.Code != 409 {
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
	w = rc11Fence(r, "acquire", strings.Repeat("a", 64))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "another"
	_, _, err = r.queueProjectPreparation("actor", id, req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectQueueFailuresRetainQueuedReceiptWithoutRetry(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, id, req := projectQueueFixture(t)
			if mode == "cancel-waiting" {
				r.workGate <- struct{}{}
			}
			if mode == "retained-maintenance" {
				_, _, err := r.s.register("actor", registration("legacy", "legacy", 18580))
				if err != nil {
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
			op, _, err := r.queueProjectPreparation("actor", id, req)
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
					t.Fatal("setup not opened")
				}
				if mode == "cancel-setup" {
					r.cancel()
				} else {
					b.fail = true
					close(b.release)
				}
			}
			waitProjectWorker(t, r)
			actual, err := loadOperation(r.s.db, op.ID)
			if err != nil || actual.Status != "interrupted" {
				t.Fatal(actual, err)
			}
			j, err := loadProjectJob(r.s.db, id)
			if err != nil || j.Preparation != nil {
				t.Fatal(j, err)
			}
			replay, hit, err := r.s.acceptProjectPreparation("actor", id, req)
			if err != nil || !hit || replay.ID != op.ID || replay.Status != "interrupted" {
				t.Fatal(replay, hit, err)
			}
			if err = validateProjectJobs(r.s.db); err != nil || r.failed.Load() {
				t.Fatal("expected refusal damaged journal", err)
			}
			if mode == "cancel-waiting" || mode == "retained-maintenance" {
				select {
				case <-b.opened:
					t.Fatal("failure dispatched setup")
				default:
				}
			}
		})
	}
}
func TestProjectQueueRefusesUnavailableBeforeReceipt(t *testing.T) {
	for _, mode := range []string{"closing", "failed", "fenced", "unsupported"} {
		t.Run(mode, func(t *testing.T) {
			r, _, id, req := projectQueueFixture(t)
			switch mode {
			case "closing":
				r.closing = true
			case "failed":
				r.failed.Store(true)
			case "fenced":
				r.updateFence.Held = true
			case "unsupported":
				r.b = &fakeBackend{}
			}
			if _, _, err := r.queueProjectPreparation("actor", id, req); err == nil {
				t.Fatal("admission unexpectedly succeeded")
			}
			var n int
			if err := r.s.db.QueryRow("SELECT COUNT(*) FROM project_jobs").Scan(&n); err != nil || n != 0 {
				t.Fatal("refusal retained a new job", n, err)
			}
		})
	}
}
