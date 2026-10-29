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

type queuedMemberStorageBackend struct {
	*queuedProjectBackend
	memberOpened chan projectnetwork.MemberStorage
}

func (b *queuedMemberStorageBackend) OpenProjectMemberStorage(ctx context.Context, p projectnetwork.Plan, s projectnetwork.MemberStorage) (projectnetwork.MemberStorageDriver, error) {
	if s.Validate(p) != nil || s.Phase != "planned" {
		b.t.Fatal("setup lost accepted intent")
	}
	b.memberOpened <- s
	select {
	case <-b.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if b.fail {
		return nil, errors.New("synthetic setup failure")
	}
	return &memberStorageStoreDriver{t: b.t, s: b.s}, nil
}
func memberStorageQueueFixture(t *testing.T) (*Server, *queuedMemberStorageBackend, string, projectMemberStorageRequest) {
	t.Helper()
	r, b, id, req := projectQueueFixture(t)
	backend := &queuedMemberStorageBackend{queuedProjectBackend: b, memberOpened: make(chan projectnetwork.MemberStorage, 2)}
	r.b = backend
	return r, backend, id, projectMemberStorageRequest{RequestID: "prepare-member", ExpectedRevision: req.ExpectedRevision, ReviewDigest: req.ReviewDigest}
}
func TestProjectMemberStorageQueueHonorsResourceAndUpdateFences(t *testing.T) {
	r, b, id, req := memberStorageQueueFixture(t)
	r.workGate <- struct{}{}
	op, hit, err := r.queueProjectMemberStorage("actor", id, "queued-zod", req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	repeated, hit, err := r.queueProjectMemberStorage("actor", id, "queued-zod", req)
	if err != nil || !hit || repeated.ID != op.ID {
		t.Fatal(repeated, hit, err)
	}
	select {
	case <-b.memberOpened:
		t.Fatal("setup bypassed resource gate")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 || !strings.Contains(w.Body.String(), "update_busy") {
		t.Fatal(w.Code, w.Body.String())
	}
	<-r.workGate
	select {
	case v := <-b.memberOpened:
		if v.OperationID != op.ID || v.ProjectID != id || v.InstanceID != "queued-zod" {
			t.Fatal(v)
		}
	case <-time.After(time.Second):
		t.Fatal("setup not dispatched")
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 409 {
		t.Fatal(w.Code)
	}
	close(b.release)
	waitProjectWorker(t, r)
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil || receipt.Status != "succeeded" {
		t.Fatal(receipt, err)
	}
	select {
	case <-b.memberOpened:
		t.Fatal("duplicate dispatched")
	default:
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	req.RequestID = "second"
	_, _, err = r.queueProjectMemberStorage("actor", id, "queued-nec", req)
	requireFault(t, err, "management_quiesced")
}
func TestProjectMemberStorageQueueFailuresRetainPlannedClaim(t *testing.T) {
	for _, mode := range []string{"cancel-waiting", "cancel-setup", "setup-failure", "retained-maintenance"} {
		t.Run(mode, func(t *testing.T) {
			r, b, id, req := memberStorageQueueFixture(t)
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
			op, _, err := r.queueProjectMemberStorage("actor", id, "queued-zod", req)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "cancel-waiting":
				r.cancel()
			case "cancel-setup", "setup-failure":
				select {
				case <-b.memberOpened:
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
			state, err := loadProjectMemberStorage(r.s.db, "queued-zod")
			if err != nil || state.Phase != "planned" {
				t.Fatal(state, err)
			}
			repeated, hit, err := r.s.acceptProjectMemberStorage("actor", id, "queued-zod", req)
			if err != nil || !hit || repeated.Status != "interrupted" || repeated.ID != op.ID {
				t.Fatal(repeated, hit, err)
			}
			r.mu.Lock()
			workers := r.updateWorkers
			r.mu.Unlock()
			if workers != 0 || r.failed.Load() {
				t.Fatal("leaked worker fence or failed store", workers)
			}
		})
	}
}
