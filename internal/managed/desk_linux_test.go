//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type deskBackendFixture struct {
	fakeBackend
	calls                        atomic.Int32
	reads                        atomic.Int32
	scenario                     string
	before                       func()
	effectStarted, effectRelease chan struct{}
}

func (b *deskBackendFixture) OpenManagedDesk(context.Context, runtimequalify.ManagedState, string) (deskRuntime, error) {
	if b.scenario == "runtime-changed" {
		return nil, errors.New("changed runtime")
	}
	return deskRuntimeFixture{b}, nil
}
func (b *deskBackendFixture) OpenProjectDesk(context.Context, runtimequalify.ProjectCurrentReadinessSelection, string, string) (deskRuntime, error) {
	return deskRuntimeFixture{b}, nil
}
func (b *deskBackendFixture) OpenExpandedProjectDesk(context.Context, runtimequalify.ProjectExpansion, string, string) (deskRuntime, error) {
	return deskRuntimeFixture{b}, nil
}
func (b *deskBackendFixture) Observe(_ context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	return state, runtimequalify.ManagedObservation{}, nil
}

// A wrapper avoids colliding with the lifecycle backend's Observe method.
type deskRuntimeFixture struct{ b *deskBackendFixture }

func (d deskRuntimeFixture) ListDesks(context.Context) ([]string, error) {
	d.b.reads.Add(1)
	if d.b.scenario == "read-fails" {
		return nil, errors.New("unavailable")
	}
	return []string{"base", "example"}, nil
}

func (d deskRuntimeFixture) Observe(context.Context, string) (deskwork.Observation, error) {
	b := d.b
	b.reads.Add(1)
	if b.scenario == "absent" {
		return deskwork.Observation{}, nil
	}
	if b.before != nil {
		f := b.before
		b.before = nil
		f()
	}
	if b.scenario == "read-fails" || b.scenario == "post-read-fails" && b.calls.Load() > 0 {
		return deskwork.Observation{}, errors.New("read failed")
	}
	if b.scenario == "desk-exists" || b.calls.Load() > 0 {
		return deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}, nil
	}
	return deskwork.Observation{}, nil
}
func (d deskRuntimeFixture) Create(context.Context, string) error {
	b := d.b
	b.calls.Add(1)
	if b.effectStarted != nil {
		close(b.effectStarted)
		<-b.effectRelease
	}
	if b.scenario == "transport-lost" {
		return errors.New("reply lost")
	}
	return nil
}
func deskServerFixture(t *testing.T, b *deskBackendFixture) (*Server, deskwork.Intent) {
	t.Helper()
	s := testStore(t)
	_, _, err := s.register("fixture", registration("alpha", "desk-fixture", 18080))
	g02Check(t, err)
	v, err := loadInstance(s.db, "alpha")
	g02Check(t, err)
	v.Public.ObservedState = "running"
	v.Public.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
	saveRecoveryFixture(t, s, v)
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, uid: uint32(os.Geteuid())}
	t.Cleanup(func() { g02Check(t, r.Close()) })
	return r, deskIntentFixture()
}
func deskHTTPRequest(t *testing.T, r *Server, method, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if value != nil {
		var err error
		body, err = json.Marshal(value)
		g02Check(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(protocol.Header, "1")
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	return out
}
func deskWait(t *testing.T, r *Server, id string) deskwork.Receipt {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		out := deskHTTPRequest(t, r, http.MethodGet, protocol.DeskOperationsPath+"/"+id, nil)
		if out.Code != 200 {
			t.Fatal(out.Code, out.Body.String())
		}
		var reply protocol.DeskOperation
		g02Check(t, json.Unmarshal(out.Body.Bytes(), &reply))
		state := reply.Receipt.Steps[len(reply.Receipt.Steps)-1].State
		if state == "blocked" || state == "observed" || state == "uncertain" || state == "interrupted" {
			return reply.Receipt
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("desk operation did not finish")
	return deskwork.Receipt{}
}
func TestDeskAPIEnforcesPolicyAndSingleDispatch(t *testing.T) {
	for _, scenario := range []string{"ok", "default-block", "revoked", "runtime-changed", "desk-exists", "read-fails", "post-read-fails", "transport-lost"} {
		t.Run(scenario, func(t *testing.T) {
			b := &deskBackendFixture{scenario: scenario}
			r, intent := deskServerFixture(t, b)
			if scenario != "default-block" {
				p, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
				g02Check(t, err)
				intent.PolicyRevision = p.Revision
			}
			if scenario == "revoked" {
				b.before = func() {
					_, err := r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: intent.PolicyRevision, Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Blocked()})
					g02Check(t, err)
				}
			}
			out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
			if out.Code != 202 {
				t.Fatal(out.Code, out.Body.String())
			}
			receipt := deskWait(t, r, intent.ID)
			state := receipt.Steps[len(receipt.Steps)-1].State
			want := "blocked"
			effects := int32(0)
			switch scenario {
			case "ok":
				want = "observed"
				effects = 1
			case "transport-lost", "post-read-fails":
				want = "uncertain"
				effects = 1
			}
			if state != want || b.calls.Load() != effects {
				t.Fatal(state, want, b.calls.Load())
			}
			replay := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
			if replay.Code != 202 {
				t.Fatal(replay.Code, replay.Body.String())
			}
			r.mu.Lock()
			r.mu.Unlock()
			if b.calls.Load() != effects {
				t.Fatal("duplicate effect")
			}
			p, err := loadAppPolicy(r.s.db)
			g02Check(t, err)
			decision := p.Effective(intent.Target)
			if decision.Commit.Value != apppolicy.Block || decision.Publish.Value != apppolicy.Block {
				t.Fatal("creation granted commit/publish")
			}
		})
	}
}
func TestDeskReceiptReadableDuringEffect(t *testing.T) {
	b := &deskBackendFixture{effectStarted: make(chan struct{}), effectRelease: make(chan struct{})}
	r, intent := deskServerFixture(t, b)
	p, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
	g02Check(t, err)
	intent.PolicyRevision = p.Revision
	out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
	if out.Code != 202 {
		t.Fatal(out.Code, out.Body.String())
	}
	defer close(b.effectRelease)
	select {
	case <-b.effectStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("effect not dispatched")
	}
	out = deskHTTPRequest(t, r, http.MethodGet, protocol.DeskOperationsPath+"/"+intent.ID, nil)
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	var reply protocol.DeskOperation
	g02Check(t, json.Unmarshal(out.Body.Bytes(), &reply))
	if reply.Receipt.Steps[len(reply.Receipt.Steps)-1].State != "dispatched" {
		t.Fatal("dispatch was not durable before effect")
	}
	if reply.Receipt.Actor != fmt.Sprintf("local-uid:%d", os.Geteuid()) {
		t.Fatal("untrusted actor")
	}
}

func TestDeskReconciliationNeverRedispatchesAndIgnoresRevokedEffectGate(t *testing.T) {
	for _, after := range []string{"present", "absent", "read-fails", "runtime-changed"} {
		t.Run(after, func(t *testing.T) {
			b := &deskBackendFixture{scenario: "transport-lost"}
			r, intent := deskServerFixture(t, b)
			p, err := r.s.putAppPolicy("setup", apppolicy.Update{Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
			g02Check(t, err)
			intent.PolicyRevision = p.Revision
			out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
			if out.Code != 202 {
				t.Fatal(out.Code, out.Body.String())
			}
			receipt := deskWait(t, r, intent.ID)
			if receipt.Steps[len(receipt.Steps)-1].State != "uncertain" {
				t.Fatal("not uncertain")
			}
			r.mu.Lock()
			b.scenario = after
			_, err = r.s.putAppPolicy("setup", apppolicy.Update{ExpectedRevision: p.Revision, Scope: "ship", Target: intent.Target.Ship(), Gates: apppolicy.Blocked()})
			r.mu.Unlock()
			g02Check(t, err)
			path := protocol.DeskOperationsPath + "/" + intent.ID + "/reconcile"
			out = deskHTTPRequest(t, r, http.MethodPost, path, struct{}{})
			if out.Code != 202 {
				t.Fatal(out.Code, out.Body.String())
			}
			r.wg.Wait()
			got, err := loadDeskOperation(r.s.db, intent.ID)
			g02Check(t, err)
			state := got.Steps[len(got.Steps)-1].State
			want := "uncertain"
			if after == "present" {
				want = "observed"
				if got.Steps[len(got.Steps)-1].Actor != fmt.Sprintf("local-uid:%d", os.Geteuid()) {
					t.Fatal("reconciler identity missing")
				}
			}
			if state != want || b.calls.Load() != 1 || r.failed.Load() {
				t.Fatal("reconciliation dispatched or lost evidence", state, b.calls.Load(), r.failed.Load())
			}
		})
	}
}
func TestDeskAPIRejectsUnknownEffectsAndObserverWrites(t *testing.T) {
	b := &deskBackendFixture{}
	r, intent := deskServerFixture(t, b)
	for _, action := range []string{"commit", "publish-source", "publish-treaty", "dojo", "override-policy"} {
		intent.Action = action
		out := deskHTTPRequest(t, r, http.MethodPost, protocol.DeskOperationsPath, intent)
		if out.Code < 400 {
			t.Fatal("unsupported effect accepted", action)
		}
	}
	intent.Action = "create-desk"
	body, _ := json.Marshal(intent)
	req := httptest.NewRequest(http.MethodPost, protocol.DeskOperationsPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	r.serveDeskOperation(out, req, "fixture-observer", "observer")
	if out.Code != 403 || b.calls.Load() != 0 {
		t.Fatal("observer effect accepted")
	}
	ids, err := deskOperationIDs(r.s.db)
	g02Check(t, err)
	if len(ids) != 0 {
		t.Fatal("invalid effects were journaled")
	}
}

func TestDeskDiscoveryIsReadOnlyAndBoundToCurrentSelection(t *testing.T) {
	for _, scenario := range []string{"ok", "read-fails", "runtime-changed", "wrong-revision", "wrong-instance", "desk-selected", "observer", "method"} {
		t.Run(scenario, func(t *testing.T) {
			b := &deskBackendFixture{scenario: scenario}
			r, intent := deskServerFixture(t, b)
			selection := protocol.DeskDiscoverySelection{Target: intent.Target.Ship(), RuntimeID: intent.RuntimeID, ConfigRevision: intent.ConfigRevision}
			switch scenario {
			case "wrong-revision":
				selection.ConfigRevision++
			case "wrong-instance":
				selection.Target.InstanceID = "other"
			case "desk-selected":
				selection.Target.Desk = "base"
			}
			var out *httptest.ResponseRecorder
			if scenario == "observer" {
				body, _ := json.Marshal(selection)
				out = httptest.NewRecorder()
				r.serveDeskDiscovery(out, httptest.NewRequest("POST", protocol.DeskDiscoveryPath, bytes.NewReader(body)), "observer")
			} else {
				method := "POST"
				if scenario == "method" {
					method = "GET"
				}
				out = deskHTTPRequest(t, r, method, protocol.DeskDiscoveryPath, selection)
			}
			if scenario == "ok" {
				var result protocol.DeskDiscovery
				if out.Code != 200 || json.Unmarshal(out.Body.Bytes(), &result) != nil || result.Selection != selection || len(result.Desks) != 2 || b.reads.Load() != 1 {
					t.Fatal(out.Code, out.Body.String())
				}
			} else if out.Code < 400 {
				t.Fatal("invalid discovery accepted", out.Code, out.Body.String())
			}
			if b.calls.Load() != 0 {
				t.Fatal("discovery dispatched mutation")
			}
		})
	}
}
