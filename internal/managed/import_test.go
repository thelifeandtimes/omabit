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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func sourceFixture() protocol.ImportSource {
	return protocol.ImportSource{RunID: strings.Repeat("a", 32), ContentDigest: strings.Repeat("b", 64), Files: 7, Bytes: 1024, Ship: "~zod", QualifiedAt: time.Now().UTC()}
}

func TestImportReservationReplayAndRestartQuarantine(t *testing.T) {
	s := testStore(t)
	source := sourceFixture()
	req := registration("imported", "import-one", 18086)
	op, replayed, err := s.registerInstance("operator", req, &source)
	if err != nil || replayed || op.Status != "queued" || op.Kind != "import-fake" {
		t.Fatal("import registration", err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal("missing downgrade guard")
	}
	copy, replayed, err := s.registerInstance("operator", req, &source)
	if err != nil || !replayed || copy.ID != op.ID {
		t.Fatal("replay changed")
	}
	other := registration("second", "import-two", 18087)
	if _, _, err := s.registerInstance("operator", other, &source); err == nil {
		t.Fatal("source used twice")
	}
	// Even a fully copied private checkpoint must be quarantined if the operation
	// itself was not durably completed before management disappeared.
	v0, _ := loadInstance(s.db, "imported")
	now := time.Now().UTC()
	v0.Runtime.Initialized = true
	v0.Runtime.Import.Uncertain = false
	v0.Runtime.Import.FinishedAt = &now
	if err := s.runtimeUpdate("imported", op.ID, v0.Runtime); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "imported")
	if err != nil || !v.Runtime.UncertainProvisioning() || !v.Public.RecoveryRequired {
		t.Fatal("restart lost quarantine")
	}
	if _, _, err := s.accept("operator", "imported", protocol.OperationRequest{Kind: "start", RequestID: "bad-start", ExpectedRevision: 1}); err == nil {
		t.Fatal("incomplete import accepted start")
	}
	finished, _, err := s.registerInstance("operator", req, &source)
	if err != nil || finished.Status != "interrupted" {
		t.Fatal("interrupted import replay changed")
	}
}

type completingImportBackend struct{ catalogueFixture }

func (b *completingImportBackend) Apply(ctx context.Context, kind string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if kind != "import-fake" {
		return b.fakeBackend.Apply(ctx, kind, state, persist)
	}
	now := time.Now().UTC()
	state.Prepared = true
	state.Import.StartedAt = &now
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	state.Initialized = true
	state.Import.FinishedAt = &now
	state.Import.Uncertain = false
	state.Import.CopiedFiles = state.Import.Source.Files
	state.Import.CopiedBytes = state.Import.Source.Bytes
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	return state, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func TestImportRPCQueuesCompletesAndReplaysWithoutStarting(t *testing.T) {
	backend := &completingImportBackend{catalogueFixture: catalogueFixture{sources: []protocol.ImportSource{sourceFixture()}}}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := newServer(dir, backend, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	source := backend.sources[0]
	body := protocol.ImportFakeRequest{ID: "imported", RequestID: "one-import", Config: protocol.Configuration{HTTPPort: 18086, AmesPort: 34343, Loom: 31}, SourceRunID: source.RunID, ExpectedDigest: source.ContentDigest, ConfirmStopped: true}
	call := func(body protocol.ImportFakeRequest) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/v1/imports", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(protocol.Header, "1")
		req = req.WithContext(context.WithValue(req.Context(), peerKey{}, r.uid))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	w := call(body)
	if w.Code != 202 {
		t.Fatalf("admission %d: %s", w.Code, w.Body)
	}
	var op protocol.Operation
	if err := json.Unmarshal(w.Body.Bytes(), &op); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		current, err := loadOperation(r.s.db, op.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("import did not complete")
		}
		time.Sleep(time.Millisecond)
	}
	v, err := loadInstance(r.s.db, "imported")
	if err != nil || v.Public.ObservedState != "stopped" || v.Public.AppliedRevision != 0 || v.Public.RecoveryRequired || v.Public.Import.Uncertain || v.Public.Import.CopiedFiles != 7 || len(v.Runtime.Runtime.Containers) != 0 {
		t.Fatal("import did not stay stopped", err)
	}
	w = call(body)
	if w.Code != 200 || backend.calls != 1 {
		t.Fatal("replay repeated source lookup")
	}
	body.ID = "second"
	body.RequestID = "second-import"
	body.Config.HTTPPort = 18087
	if w = call(body); w.Code != 409 {
		t.Fatal("source could be imported twice")
	}
}

type catalogueFixture struct {
	fakeBackend
	sources []protocol.ImportSource
	calls   int
}

func (b *catalogueFixture) ImportSources(context.Context) ([]protocol.ImportSource, error) {
	b.calls++
	return append([]protocol.ImportSource(nil), b.sources...), nil
}

func TestImportRPCReplayPrecedesCatalogueAndObserverRefused(t *testing.T) {
	s := testStore(t)
	source := sourceFixture()
	base := registration("imported", "request", 18086)
	op, _, err := s.registerInstance("actor", base, &source)
	if err != nil {
		t.Fatal(err)
	}
	b := &catalogueFixture{}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	body := protocol.ImportFakeRequest{ID: base.ID, Label: base.Label, Config: base.Config, RequestID: base.RequestID, SourceRunID: source.RunID, ExpectedDigest: source.ContentDigest, ConfirmStopped: true}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/imports", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.serveImports(w, req, "actor", "operator")
	if w.Code != 200 || b.calls != 0 || !strings.Contains(w.Body.String(), op.ID) {
		t.Fatal("replay consulted changed source catalogue")
	}
	w = httptest.NewRecorder()
	r.serveImports(w, httptest.NewRequest("GET", "/v1/imports", nil), "observer", "observer")
	if w.Code != 403 || b.calls != 0 {
		t.Fatal("observer read private catalogue")
	}
	b.sources = []protocol.ImportSource{source}
	w = httptest.NewRecorder()
	r.serveImports(w, httptest.NewRequest("GET", "/v1/imports", nil), "actor", "operator")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"claimed_by":"imported"`) {
		t.Fatal("source reservation missing")
	}
}
