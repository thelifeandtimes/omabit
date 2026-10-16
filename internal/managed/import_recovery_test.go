package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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
	// Seed before the server starts polling. A poll of the initial registration
	// can otherwise overwrite direct fixture writes with the same revision.
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedRecovery(t, s)
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	r, err := newServer(dir, b, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	line, observer := grantLine(t, "observer")
	r.keys = filepath.Join(privateDir(t), "keys")
	if err := os.WriteFile(r.keys, []byte(line+"\n"), 0600); err != nil {
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
	hello := request("GET", "/hello", "", "", true)
	var h protocol.Hello
	if err = json.Unmarshal(hello.Body.Bytes(), &h); err != nil || !slices.Contains(h.Features, "recover-import") {
		t.Fatal("missing recovery capability", err)
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
		t.Fatal("stale recovery", w.Code)
	}
	// Recovery must wait behind the same host-work gate as import and start.
	r.workGate <- struct{}{}
	first := request("POST", path, body, "", true)
	if first.Code != 202 {
		t.Fatal(first.Code, first.Body.String())
	}
	again := request("POST", path, body, "", true)
	if again.Code != 200 || first.Body.String() != again.Body.String() {
		t.Fatal("RPC replay changed", again.Body.String())
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("recovery bypassed host gate")
	}
}

func TestRecoveryJournal(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "recover", ExpectedRevision: 1}
	op, replayed, err := s.accept("operator", "one", req)
	if err != nil || replayed {
		t.Fatal("recovery not accepted", err)
	}
	again, replayed, err := s.accept("operator", "one", req)
	if err != nil || !replayed || again.ID != op.ID {
		t.Fatal("queued replay", err)
	}
	req.ExpectedRevision = 2
	_, _, err = s.accept("operator", "one", req)
	requireFault(t, err, "request_conflict")
	op, v, err := s.beginOperation(op.ID)
	if err != nil || v.Public.DesiredState != "stopped" || v.Public.ObservedState != "enrolling" {
		t.Fatal(v.Public, err)
	}
	_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start-busy", ExpectedRevision: 1})
	requireFault(t, err, "instance_busy")
	now := time.Now().UTC()
	v.Runtime.ExternalImport.Stage = "complete"
	v.Runtime.Initialized = true
	v.Runtime.Adoption.Uncertain, v.Runtime.Adoption.FinishedAt = false, &now
	if err = s.runtimeUpdate("one", op.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err = s.finish(op, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
		t.Fatal(err)
	}
	req.ExpectedRevision = 1
	again, replayed, err = s.accept("operator", "one", req)
	if err != nil || !replayed || again.ID != op.ID || again.Status != "succeeded" {
		t.Fatal("completed replay", again, err)
	}
	got, err := loadInstance(s.db, "one")
	if err != nil || got.Public.DesiredState != "stopped" || got.Public.ObservedState != "stopped" || got.Public.RecoveryRequired || !got.Public.IdentityPending || got.Public.ActiveOperationID != "" {
		t.Fatal(got.Public, err)
	}
	snapshot, err := s.snapshot()
	if err != nil || len(snapshot.Instances) != 1 || snapshot.Instances[0].RuntimeStatus == nil || snapshot.Instances[0].RuntimeStatus.LastOperation == nil || snapshot.Instances[0].RuntimeStatus.LastOperation.ID != op.ID {
		t.Fatal("snapshot lost recovery operation", snapshot, err)
	}
	req.RequestID = "recover-again"
	_, _, err = s.accept("operator", "one", req)
	requireFault(t, err, "invalid_state")
}

func TestRecoveryInterrupted(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "first", ExpectedRevision: 1}
	op, _, err := s.accept("operator", "one", req)
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
	again, replayed, err := s.accept("operator", "one", req)
	if err != nil || !replayed || again.ID != op.ID || again.Status != "interrupted" {
		t.Fatal("lost interrupted receipt", again, err)
	}
	v, err = loadInstance(s.db, "one")
	if err != nil || !v.Runtime.ImportRecoveryEligible() || !v.Public.RecoveryRequired || v.Public.DesiredState != "stopped" {
		t.Fatal("lost retained fence", v.Public, err)
	}
	_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	requireFault(t, err, "recovery_required")
	req.RequestID = "reviewed-retry"
	if _, replayed, err = s.accept("operator", "one", req); err != nil || replayed {
		t.Fatal("explicit retry refused", err)
	}
}

func TestRecoveryActorReplay(t *testing.T) {
	s := testStore(t)
	seedRecovery(t, s)
	req := protocol.OperationRequest{Kind: "recover-import", RequestID: "same-request", ExpectedRevision: 1}
	first, _, err := s.accept("operator-a", "one", req)
	if err != nil {
		t.Fatal(err)
	}
	if _, replayed, err := s.accept("operator-b", "one", req); replayed {
		t.Fatal("another actor obtained queued replay")
	} else {
		requireFault(t, err, "instance_busy")
	}
	first, v, err := s.beginOperation(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.finish(first, v.Runtime, runtimequalify.ManagedObservation{RecoveryRequired: true}, context.Canceled, true); err != nil {
		t.Fatal(err)
	}
	second, replayed, err := s.accept("operator-b", "one", req)
	if err != nil || replayed || second.ID == first.ID || second.Actor != "operator-b" {
		t.Fatal("actor request namespace not isolated", second, err)
	}
	again, replayed, err := s.accept("operator-a", "one", req)
	if err != nil || !replayed || again.ID != first.ID || again.Status != "interrupted" {
		t.Fatal("original actor lost receipt", again, err)
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
	// Another worker can leave unresolved resources after acceptance, while
	// recovery is queued behind the host gate. Recheck at worker admission.
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
		t.Fatal("post-gate admission not journaled", receipt, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("recovery dispatched despite retained utility")
	}
	v, err := loadInstance(s.db, "one")
	if err != nil || !v.Runtime.ImportRecoveryEligible() || !v.Public.RecoveryRequired {
		t.Fatal("admission failure lost fence", v.Public, err)
	}
}

func TestRecoveryGates(t *testing.T) {
	for _, fault := range []string{"revision", "busy", "running", "phase", "identity", "fingerprint", "container", "maintenance", "unprepared"} {
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
			}
			saveRecoveryFixture(t, s, v)
			_, _, err := s.accept("operator", "one", req)
			requireFault(t, err, code)
			_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
			if err == nil {
				t.Fatal("start bypassed import fence")
			}
		})
	}
}
