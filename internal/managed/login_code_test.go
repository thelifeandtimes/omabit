package managed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type codeFixture struct {
	fakeBackend
	calls int
	fail  bool
}

func (b *codeFixture) LoginCode(context.Context, runtimequalify.ManagedState, string) (string, error) {
	b.calls++
	if b.fail {
		return "", errors.New("PRIVATE-SECRET")
	}
	return "lidlut-tabwed-pillex-ridrup", nil
}

func TestLoginCodeAdmissionAndRedactedAudit(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "running", Health: "healthy", Resources: resourceSample(time.Now().UTC())}); err != nil {
		t.Fatal(err)
	}
	b := &codeFixture{}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	call := func(revision uint64, container string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(protocol.LoginCodeRequest{ExpectedRevision: revision, ContainerID: container})
		req := httptest.NewRequest("POST", "/v1/instances/alpha/login-code", strings.NewReader(string(body)))
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	container := strings.Repeat("a", 64)
	for _, w := range []*httptest.ResponseRecorder{call(2, container), call(1, strings.Repeat("b", 64)), call(1, "short")} {
		if w.Code < 400 {
			t.Fatal("invalid selection accepted")
		}
	}
	if b.calls != 0 {
		t.Fatal("invalid selection reached backend")
	}
	good := call(1, container)
	if good.Code != 200 || good.Header().Get("Cache-Control") != "no-store" || !strings.Contains(good.Body.String(), "lidlut") {
		t.Fatal("secret response unavailable")
	}
	b.fail = true
	bad := call(1, container)
	if bad.Code != 502 || strings.Contains(bad.Body.String(), "PRIVATE") {
		t.Fatal("backend diagnostics leaked")
	}
	snapshot, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := json.Marshal(snapshot)
	if strings.Contains(string(wire), "lidlut") || strings.Contains(string(wire), "PRIVATE") {
		t.Fatal("secret in snapshot")
	}
	if snapshot.Operations[0].Kind != "login-code" || snapshot.Operations[0].Actor == "" {
		t.Fatal("missing attributed audit")
	}
	w := httptest.NewRecorder()
	r.loginCode(w, httptest.NewRequest("POST", "/", nil), "observer", "observer", "alpha")
	if w.Code != 403 || b.calls != 2 {
		t.Fatal("observer admitted")
	}
}

func TestLoginCodeAuditRecordsOnlySafeDeadlineDiagnostic(t *testing.T) {
	s := testStore(t)
	if err := s.auditCode("actor", "synthetic", 1, false, errors.New("PRIVATE response"), true); err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err := s.db.QueryRow("SELECT body FROM operations WHERE fingerprint='login-code-audit'").Scan(&body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "PRIVATE") || !strings.Contains(string(body), "stage=unavailable, deadline_expired=true") {
		t.Fatal("unsafe or missing diagnostic")
	}
}
