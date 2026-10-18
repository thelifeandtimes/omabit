package managed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func rc11Request(r *Server, ctx context.Context, method, path string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, strings.NewReader(string(data))).WithContext(context.WithValue(ctx, peerKey{}, uint32(0)))
	req.Header.Set(protocol.Header, "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func rc11Fence(r *Server, action, token string) *httptest.ResponseRecorder {
	return rc11Request(r, context.Background(), "POST", protocol.ManagementUpdatePath, protocol.ManagementUpdateRequest{Action: action, RunnerID: r.s.runnerID, Token: token, ExpectedSHA256: r.executableSHA256})
}

func TestRC11QuiescenceRefusesActiveWorkersWithoutCancellation(t *testing.T) {
	b := &fakeBackend{entered: make(chan struct{}, 1), release: make(chan struct{})}
	r, c, shutdown := testEndpoint(t, privateDir(t), b)
	if _, err := c.CreateFake(context.Background(), registration("alpha", "reg", 18080)); err != nil {
		t.Fatal(err)
	}
	op, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("worker not dispatched")
	}
	defer shutdown()
	token := strings.Repeat("a", 64)
	w := rc11Fence(r, "acquire", token)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "update_busy") || r.ctx.Err() != nil {
		close(b.release)
		t.Fatal(w.Code, w.Body.String())
	}
	close(b.release)
	waitOperation(t, c, op.ID, "succeeded")
	deadline := time.Now().Add(time.Second)
	for {
		w = rc11Fence(r, "acquire", token)
		if w.Code == 200 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(w.Code, w.Body.String())
		}
		time.Sleep(time.Millisecond)
	}
	for _, path := range []string{"/v1/instances", "/v1/instances/alpha/operations", protocol.PillUploadPath} {
		w := rc11Request(r, context.Background(), "POST", path, map[string]string{})
		if w.Code != 409 || !strings.Contains(w.Body.String(), "management_quiesced") {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w := rc11Request(r, context.Background(), "GET", "/v1/snapshot", nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := rc11Fence(r, "release", token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if _, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
}

func TestRC11QuiescenceBulkAndDurableToken(t *testing.T) {
	dir := privateDir(t)
	r, err := newServer(dir, &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("c", 64)
	if err := r.beginUpdateBulk(); err != nil {
		t.Fatal(err)
	}
	if w := rc11Fence(r, "acquire", token); w.Code != 409 {
		t.Fatal(w.Body.String())
	}
	r.endUpdateBulk()
	if w := rc11Fence(r, "acquire", token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r, err = newServer(dir, &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if !r.updateFence.Held || r.updateFence.Token != token {
		t.Fatal("restart lost fence")
	}
	if err := r.beginUpdateBulk(); err == nil {
		r.endUpdateBulk()
		t.Fatal("bulk bypassed restart fence")
	}
	if w := rc11Fence(r, "acquire", strings.Repeat("d", 64)); w.Code != 409 {
		t.Fatal(w.Body.String())
	}
	if w := rc11Fence(r, "release", strings.Repeat("d", 64)); w.Code != 409 {
		t.Fatal(w.Body.String())
	}
	if w := rc11Fence(r, "release", token); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := rc11Fence(r, "release", token); w.Code != 200 {
		t.Fatal("lost ack replay", w.Body.String())
	}
	if w := rc11Fence(r, "acquire", token); w.Code != 409 {
		t.Fatal("reused released token", w.Body.String())
	}
}

func TestRC11QuiescenceRootLocalOnly(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, remote := range []bool{false, true} {
		uid := uint32(1000)
		if remote {
			uid = 0
		}
		r.uid = 1000
		req := httptest.NewRequest("GET", protocol.ManagementUpdatePath, nil).WithContext(context.WithValue(context.Background(), peerKey{}, uid))
		req.Header.Set(protocol.Header, "1")
		if remote {
			req.Header.Set(access.Header, "ssh-key")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestRC11QuiescenceSchedulerAndStartupRestart(t *testing.T) {
	t.Run("scheduler", func(t *testing.T) {
		s, _, now := routineFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		r := &Server{s: s, b: &routineBackendFixture{}, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), executableSHA256: strings.Repeat("f", 64)}
		defer r.Close()
		if w := rc11Fence(r, "acquire", strings.Repeat("a", 64)); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		r.dispatchRoutines(now)
		p, err := loadRoutinePolicy(s.db, "alpha")
		if err != nil || p.LastSlot != nil {
			t.Fatal("scheduler admitted while fenced", p, err)
		}
		if w := rc11Fence(r, "release", strings.Repeat("a", 64)); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		r.dispatchRoutines(now)
		p, err = loadRoutinePolicy(s.db, "alpha")
		if err != nil || p.LastSlot == nil {
			t.Fatal("scheduler did not resume", p, err)
		}
	})
	t.Run("startup", func(t *testing.T) {
		dir := privateDir(t)
		s, err := testOpenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		seedHealthyBoot(t, s)
		token := strings.Repeat("b", 64)
		r := &Server{s: s}
		if err := r.saveManagementFence(managementFence{Token: token, Held: true}); err != nil {
			t.Fatal(err)
		}
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
		b := &startupBackend{exitCode: 255}
		r, err = newServerWithStartup(dir, b, time.Millisecond, true, nextBootID)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		time.Sleep(30 * time.Millisecond)
		b.mu.Lock()
		calls := b.calls
		b.mu.Unlock()
		if calls != 0 {
			t.Fatal("startup bypassed fence", calls)
		}
		if w := rc11Fence(r, "release", token); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		startupOperations(t, r, 1, "succeeded")
	})
}

func TestRC11CanceledPortSuggestionDoesNotLatchControl(t *testing.T) {
	r, err := newServer(privateDir(t), &fakeBackend{}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	r.s.portAvailable = func(string, uint16) bool { cancel(); return false }
	w := rc11Request(r, ctx, "GET", "/v1/port-suggestion", nil)
	if w.Code != http.StatusRequestTimeout || r.failed.Load() {
		t.Fatal(w.Code, w.Body.String())
	}
	r.s.portAvailable = func(string, uint16) bool { return true }
	w = rc11Request(r, context.Background(), "POST", "/v1/instances", registration("alpha", "register", 18080))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = rc11Request(r, context.Background(), "POST", "/v1/instances/alpha/operations", protocol.OperationRequest{Kind: "stop", RequestID: "unrelated-stop", ExpectedRevision: 1})
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	// No state-store incident and normal scheduler dispatch remains available.
	r.dispatchRoutines(time.Now().UTC())
	if r.failed.Load() {
		t.Fatal("request cancellation disabled scheduling")
	}
}
