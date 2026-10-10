package managed

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCometRegistrationModeReplayAndDowngradeGuard(t *testing.T) {
	s := testStore(t)
	req := registration("comet", "comet-one", 18087)
	if _, _, err := s.registerKind("actor", req, nil, true); err == nil {
		t.Fatal("fake-only mode accepted comet")
	}
	s.mode = DevelopmentMode
	op, replayed, err := s.registerKind("actor", req, nil, true)
	if err != nil || replayed || op.Kind != "register-comet" || op.Status != "succeeded" {
		t.Fatal("comet registration", err)
	}
	v, err := loadInstance(s.db, "comet")
	if err != nil || v.Public.Mode != "comet" || v.Public.Ship != "" || v.Public.ObservedState != "uninitialized" || v.Runtime.Prepared || v.Runtime.Comet == nil {
		t.Fatal("registration dispatched or invented identity", err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal("missing comet downgrade guard")
	}
	if duplicate, hit, err := s.registerKind("actor", req, nil, true); err != nil || !hit || duplicate.ID != op.ID {
		t.Fatal("registration replay changed")
	}
	for _, kind := range []string{"pack", "meld", "roll", "inspect-epochs", "prune-epochs"} {
		if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1}); err == nil {
			t.Fatal("unqualified comet action admitted")
		}
	}
	start, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "start", RequestID: "first-start", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	v, _ = loadInstance(s.db, "comet")
	now := time.Now().UTC()
	v.Runtime.Initialized = true
	v.Runtime.Comet.Ship = "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet"
	v.Runtime.Comet.StartedAt = &now
	v.Runtime.Comet.FinishedAt = &now
	if err := s.runtimeUpdate("comet", start.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	v, _ = loadInstance(s.db, "comet")
	if !v.Runtime.UncertainProvisioning() || !v.Public.RecoveryRequired {
		t.Fatal("interrupted first boot adopted identity")
	}
	if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "start", RequestID: "another-start", ExpectedRevision: 1}); err == nil {
		t.Fatal("interrupted identity creation retried")
	}
}

func TestDevelopmentModeHelloAndExplicitNetworkConsent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := newServerMode(dir, &fakeBackend{}, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(protocol.Header, "1")
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := call("GET", "/hello", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mode":"development-managed"`) || !strings.Contains(w.Body.String(), "create-comet") || !strings.Contains(w.Body.String(), "comet-compaction") || !strings.Contains(w.Body.String(), "comet-epochs") || !strings.Contains(w.Body.String(), "comet-prune-prefix") {
		t.Fatal("wrong development capabilities")
	}
	base := registration("new-comet", "reg", 18087)
	body, _ := json.Marshal(protocol.CreateCometRequest{CreateFakeRequest: base})
	if w = call("POST", "/v1/comets", string(body)); w.Code != 400 {
		t.Fatal("network consent missing")
	}
	body, _ = json.Marshal(protocol.CreateCometRequest{CreateFakeRequest: base, ConfirmNetwork: true})
	if w = call("POST", "/v1/comets", string(body)); w.Code != 201 {
		t.Fatal("comet not registered", w.Body)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if old, err := newServer(dir, &fakeBackend{}, time.Hour); err == nil {
		old.Close()
		t.Fatal("fake-only server reopened comet state")
	}
}
