//go:build linux && amd64

package managed

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func uncertainDeskAdmissionFixture(t *testing.T) (*Server, *deskBackendFixture, deskwork.Receipt) {
	t.Helper()
	b := &deskBackendFixture{scenario: "transport-lost"}
	r, intent := deskServerFixture(t, b)
	policy, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
	g02Check(t, err)
	intent.PolicyRevision = policy.Revision
	out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
	if out.Code != http.StatusAccepted {
		t.Fatal(out.Code, out.Body.String())
	}
	r.wg.Wait()
	receipt := deskWait(t, r, intent.ID)
	if receipt.Steps[len(receipt.Steps)-1].State != "uncertain" || b.calls.Load() != 1 {
		t.Fatal("expected one originally dispatched lost reply")
	}
	return r, b, receipt
}

func TestDeskReconciliationReturnsSavedBusyReceiptDuringHeldAdmission(t *testing.T) {
	r, b, original := uncertainDeskAdmissionFixture(t)
	r.mu.Lock()
	defer r.mu.Unlock()
	done := make(chan protocol.DeskOperation, 1)
	go func() {
		out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath+"/"+original.Intent.ID+"/reconcile", struct{}{})
		var reply protocol.DeskOperation
		if out.Code == http.StatusAccepted {
			_ = json.Unmarshal(out.Body.Bytes(), &reply)
		}
		done <- reply
	}()
	select {
	case reply := <-done:
		if reply.ReconciliationAdmission != "busy" || !reflect.DeepEqual(reply.Receipt, original) || r.updateWorkers != 0 || b.calls.Load() != 1 {
			t.Fatal("busy admission queued work or changed saved evidence", reply)
		}
	case <-time.After(time.Second):
		t.Fatal("reconciliation waited for the held native/lifecycle mutex")
	}
}

func TestDeskReconciliationAdmitsOnlyOneObserverAndPreservesOriginalDispatch(t *testing.T) {
	r, b, original := uncertainDeskAdmissionFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	r.mu.Lock()
	b.before = func() { close(started); <-release }
	r.mu.Unlock()
	// Always release a failed test's owned observer before Server.Close waits.
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	reads := b.reads.Load()
	path := protocol.DeskOperationsPath + "/" + original.Intent.ID + "/reconcile"
	out := deskHTTPRequest(t, r, http.MethodPost, path, struct{}{})
	var admitted protocol.DeskOperation
	g02Check(t, json.Unmarshal(out.Body.Bytes(), &admitted))
	if out.Code != http.StatusAccepted || admitted.ReconciliationAdmission != "queued" || !reflect.DeepEqual(admitted.Receipt, original) {
		t.Fatal("observer admission changed original receipt", out.Code, admitted)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("observer did not start")
	}
	for range 30 {
		out = deskHTTPRequest(t, r, http.MethodPost, path, struct{}{})
		var busy protocol.DeskOperation
		g02Check(t, json.Unmarshal(out.Body.Bytes(), &busy))
		if out.Code != http.StatusAccepted || busy.ReconciliationAdmission != "busy" || !reflect.DeepEqual(busy.Receipt, original) {
			t.Fatal("concurrent poll changed or duplicated reconciliation", out.Code, busy)
		}
	}
	close(release)
	released = true
	r.wg.Wait()
	current := deskWait(t, r, original.Intent.ID)
	if len(current.Steps) != len(original.Steps)+1 || !reflect.DeepEqual(current.Steps[:len(original.Steps)], original.Steps) || current.Steps[len(current.Steps)-1].State != "observed" || b.calls.Load() != 1 || b.reads.Load() != reads+1 {
		t.Fatal("polling redispatched or queued duplicate native observation", current, b.calls.Load(), b.reads.Load()-reads)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateWorkers != 0 {
		t.Fatal("observer admission leaked a worker")
	}
	out = deskHTTPRequest(t, r, http.MethodPost, path, struct{}{})
	var saved protocol.DeskOperation
	g02Check(t, json.Unmarshal(out.Body.Bytes(), &saved))
	if out.Code != http.StatusOK || saved.ReconciliationAdmission != "" || !reflect.DeepEqual(saved.Receipt, current) {
		t.Fatal("completed reconciliation did not replay outside admission")
	}
}

func TestDeskReconciliationStillHonorsUpdateAndFailureFences(t *testing.T) {
	for _, fence := range []string{"update", "failed", "closing", "pending-worker"} {
		t.Run(fence, func(t *testing.T) {
			r, b, original := uncertainDeskAdmissionFixture(t)
			r.mu.Lock()
			switch fence {
			case "update":
				r.updateFence.Held = true
			case "failed":
				r.failed.Store(true)
			case "closing":
				r.closing = true
			case "pending-worker":
				r.updateWorkers = 1
			}
			r.mu.Unlock()
			out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath+"/"+original.Intent.ID+"/reconcile", struct{}{})
			want := http.StatusServiceUnavailable
			if fence == "update" {
				want = http.StatusConflict
			} else if fence == "pending-worker" {
				want = http.StatusAccepted
			}
			if out.Code != want || b.calls.Load() != 1 {
				t.Fatal("fence bypassed", out.Code, out.Body.String())
			}
			current, err := loadDeskOperation(r.s.db, original.Intent.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(current, original) {
				t.Fatal("fenced reconciliation changed original evidence")
			}
		})
	}
}
