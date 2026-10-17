package managed

// Journal, authorization and host-gate regressions adapted from wundrellama's
// PR #3, including its seed-before-poll fix (d512fdc).
import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func seedRecovery(t *testing.T, s *store) storedInstance {
	t.Helper()
	if _, _, err := s.register("operator", registration("one", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "one")
	if err != nil {
		t.Fatal(err)
	}
	v.Public.ObservedState, v.Public.RecoveryRequired, v.Public.IdentityPending = "unknown", true, true
	v.Runtime.Prepared = true
	v.Runtime.Bootstrap = nil
	v.Runtime.ExternalImport = &runtimequalify.ManagedExternalImport{Stage: "enrolling"}
	v.Runtime.Storage = &pierstore.Location{Path: "fakeships/one/pier", Identity: pierstore.Identity{Inode: 1, UrbInode: 2}, MountIdentity: pierstore.DirectoryIdentity{Inode: 3}}
	v.Runtime.Adoption = &runtimequalify.ManagedAdoption{Uncertain: true, Fingerprint: pierstore.Fingerprint(v.Runtime.Storage.Identity)}
	saveRecoveryFixture(t, s, v)
	return v
}
func saveRecoveryFixture(t *testing.T, s *store, v storedInstance) {
	t.Helper()
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = saveInstance(tx, v); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryRPC(t *testing.T) {
	b := &storageActionBackend{home: privateDir(t)}
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedRecovery(t, s)
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	r, err := newServer(dir, b, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	line, observer := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err = os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, grant string, peer bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if peer {
			req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(os.Geteuid())))
		}
		if grant != "" {
			req.Header.Set(access.Header, grant)
		}
		req.Header.Set(protocol.Header, "1")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	var hello protocol.Hello
	if err = json.Unmarshal(request("GET", "/hello", "", "", true).Body.Bytes(), &hello); err != nil || !slices.Contains(hello.Features, "recover-import") {
		t.Fatal("missing capability", err)
	}
	body := `{"kind":"recover-import","request_id":"recover","expected_revision":1}`
	path := "/v1/instances/one/operations"
	if w := request("POST", path, body, "", false); w.Code != 403 {
		t.Fatal("unauthenticated recovery", w.Code)
	}
	if w := request("POST", path, body, observer, true); w.Code != 403 {
		t.Fatal("observer recovery", w.Code)
	}
	if w := request("POST", path, strings.Replace(body, ":1", ":2", 1), "", true); w.Code != 409 {
		t.Fatal("stale revision", w.Code)
	}
	r.workGate <- struct{}{}
	first := request("POST", path, body, "", true)
	if first.Code != 202 {
		t.Fatal(first.Code, first.Body.String())
	}
	again := request("POST", path, body, "", true)
	if again.Code != 200 || first.Body.String() != again.Body.String() {
		t.Fatal("replay changed", again.Body.String())
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("bypassed host gate")
	}
}

func TestRecoveryJournalAndActorReplay(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	snap, err := s.snapshot()
	if err != nil || !snap.Instances[0].RuntimeStatus.ImportRecoveryAvailable {
		t.Fatal("missing recovery hint", err)
	}
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1}
	op, replayed, err := s.accept("operator", "one", req)
	if err != nil || replayed {
		t.Fatal(err)
	}
	again, replayed, err := s.accept("operator", "one", req)
	if err != nil || !replayed || again.ID != op.ID {
		t.Fatal("queued replay", err)
	}
	if _, replayed, err = s.accept("other-operator", "one", req); replayed {
		t.Fatal("actor obtained another queued receipt")
	} else {
		requireFault(t, err, "instance_busy")
	}
	changed := req
	changed.ExpectedRevision = 2
	_, _, err = s.accept("operator", "one", changed)
	requireFault(t, err, "request_conflict")
	op, v, err := s.beginOperation(op.ID)
	if err != nil || v.Public.DesiredState != "stopped" || v.Public.ObservedState != "enrolling" {
		t.Fatal(v.Public, err)
	}
	_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	requireFault(t, err, "instance_busy")
	completed := completedRecovery(v)
	if err = s.runtimeUpdate("one", op.ID, completed); err != nil {
		t.Fatal(err)
	}
	still, err := loadInstance(s.db, "one")
	if err != nil || !still.Runtime.ImportRecoveryEligible() {
		t.Fatal("incremental write cleared fence", err)
	}
	if err = s.finish(op, completed, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	again, replayed, err = s.accept("operator", "one", req)
	if err != nil || !replayed || again.ID != op.ID || again.Status != "succeeded" {
		t.Fatal(again, err)
	}
	snap, err = s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	got := snap.Instances[0]
	if got.DesiredState != "stopped" || got.ObservedState != "stopped" || got.RecoveryRequired || !got.IdentityPending || got.ActiveOperationID != "" || got.RuntimeStatus.ImportRecoveryAvailable || got.RuntimeStatus.LastOperation.ID != op.ID {
		t.Fatal("completion publication", got)
	}
	req.RequestID = "new-recovery"
	_, _, err = s.accept("operator", "one", req)
	requireFault(t, err, "invalid_state")
}

func TestRecoveryInterrupted(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "first", ExpectedRevision: 1}
	op, _, err := s.accept("operator-a", "one", req)
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "unknown", RecoveryRequired: true}, context.Canceled, true); err != nil {
		t.Fatal(err)
	}
	again, replayed, err := s.accept("operator-a", "one", req)
	if err != nil || !replayed || again.ID != op.ID || again.Status != "interrupted" {
		t.Fatal(again, err)
	}
	v, err = loadInstance(s.db, "one")
	if err != nil || !v.Runtime.ImportRecoveryEligible() || !v.Public.RecoveryRequired {
		t.Fatal("lost fence", err)
	}
	_, _, err = s.accept("operator-a", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	requireFault(t, err, "recovery_required")
	op, replayed, err = s.accept("operator-b", "one", req)
	if err != nil || replayed || op.Actor != "operator-b" || op.ID == again.ID {
		t.Fatal("actor replay namespace", op, err)
	}
}

func TestRecoveryPostGateMaintenance(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	if _, _, err := s.register("operator", registration("other", "register-other", 18081)); err != nil {
		t.Fatal(err)
	}
	b := &storageActionBackend{home: privateDir(t)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &Server{s: s, b: b, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1)}
	r.workGate <- struct{}{}
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1}
	op, _, err := s.accept("operator", "one", req)
	if err != nil {
		t.Fatal(err)
	}
	r.launch(op)
	other, err := loadInstance(s.db, "other")
	if err != nil {
		t.Fatal(err)
	}
	other.Runtime.MaintenanceUncertain = true
	other.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "meld", CreateDispatched: true, CreateUncertain: true}}
	saveRecoveryFixture(t, s, other)
	<-r.workGate
	r.wg.Wait()
	receipt, replayed, err := s.accept("operator", "one", req)
	if err != nil || !replayed || receipt.ID != op.ID || receipt.Status != "failed" || !strings.Contains(receipt.Message, "retained maintenance resources on other") || r.failed.Load() {
		t.Fatal(receipt, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("dispatched despite retained utility")
	}
	v, err := loadInstance(s.db, "one")
	if err != nil || !v.Runtime.ImportRecoveryEligible() || !v.Public.RecoveryRequired {
		t.Fatal("lost fence", err)
	}
}

func TestRecoveryGates(t *testing.T) {
	for _, fault := range []string{"revision", "busy", "running", "phase", "identity", "fingerprint", "container", "maintenance", "unprepared", "bootstrap", "key"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			v := seedRecovery(t, s)
			req := protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1}
			code := "invalid_state"
			switch fault {
			case "revision":
				req.ExpectedRevision = 2
				code = "revision_conflict"
			case "busy":
				v.Public.ActiveOperationID = "other"
				code = "instance_busy"
			case "running":
				v.Public.DesiredState = "running"
			case "phase":
				v.Runtime.ExternalImport.Stage = "copying"
			case "identity":
				v.Runtime.Storage.Identity.Inode = 0
			case "fingerprint":
				v.Runtime.Adoption.Fingerprint = "changed"
			case "container":
				v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{}}
			case "maintenance":
				v.Runtime.MaintenanceUncertain = true
			case "unprepared":
				v.Runtime.Prepared = false
			case "bootstrap":
				v.Runtime.Bootstrap = &protocol.Bootstrap{Kind: "bare"}
			case "key":
				v.Runtime.NativeKeyDigest = strings.Repeat("a", 64)
			}
			saveRecoveryFixture(t, s, v)
			_, _, err := s.accept("operator", "one", req)
			requireFault(t, err, code)
			_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
			if err == nil {
				t.Fatal("start bypassed fence")
			}
		})
	}
}
