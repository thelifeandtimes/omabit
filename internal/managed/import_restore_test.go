package managed

import (
	"context"
	"encoding/json"
	"errors"
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

func seedRestore(t *testing.T, s *store) storedInstance {
	t.Helper()
	v := seedRecovery(t, s)
	v.Runtime.Prepared = false
	v.Runtime.Runtime = runtimequalify.Result{}
	v.Runtime.Storage.Identity = pierstore.Identity{}
	v.Runtime.ExternalImport = &runtimequalify.ManagedExternalImport{Stage: "copying", Source: pierstore.ExternalSource{Path: "/synthetic/original", Fingerprint: strings.Repeat("a", 64), Identity: pierstore.Identity{Inode: 4, UrbInode: 5}}, HeldPath: "/synthetic/.omabit-import-" + v.Runtime.ID + "/pier"}
	v.Runtime.Adoption.Fingerprint = v.Runtime.ExternalImport.Source.Fingerprint
	saveRecoveryFixture(t, s, v)
	return v
}
func restoreRequest(id string) protocol.OperationRequest {
	return protocol.OperationRequest{Kind: protocol.ImportRestoreFeature, RequestID: id, ExpectedRevision: 1, RestoreDigest: strings.Repeat("d", 64), ConfirmRestore: true}
}
func restoreFixtureReview(s runtimequalify.ManagedState) runtimequalify.ImportRestoreReview {
	return runtimequalify.ImportRestoreReview{Version: 1, RuntimeID: s.ID, OriginalPath: s.ExternalImport.Source.Path, Source: pierstore.ExternalSource{Path: s.ExternalImport.HeldPath, Bytes: 100, Files: 2}, Destination: "/synthetic/mount", RetainedPath: "/synthetic/archive/import-" + s.ID, Digest: strings.Repeat("d", 64), DestinationTree: &pierstore.RemovalTree{Entries: []pierstore.RemovalEntry{{Path: ".", Directory: true}, {Path: "pier", Directory: true}, {Path: "pier/partial", Size: 20}}}}
}
func restorePhase(t *testing.T, s *store, op protocol.Operation, state runtimequalify.ManagedState, phase string) runtimequalify.ManagedState {
	t.Helper()
	state.ImportRestore = &runtimequalify.ManagedImportRestore{Digest: strings.Repeat("d", 64), Journal: &runtimequalify.ImportRestoreJournal{Stage: phase, Review: restoreFixtureReview(state)}}
	if err := s.runtimeUpdate(op.InstanceID, op.ID, state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestRestoreAdmissionJournalAndRetirement(t *testing.T) {
	for _, outcome := range []string{"crash-intent", "crash-retain", "crash-complete", "backend-error", "cancel", "terminal-write-failure", "success"} {
		t.Run(outcome, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { s.close() }()
			before := seedRestore(t, s)
			req := restoreRequest("restore")
			op, hit, err := s.accept("operator", "one", req)
			if err != nil || hit {
				t.Fatal(err)
			}
			if replay, hit, err := s.accept("operator", "one", req); err != nil || !hit || replay.ID != op.ID {
				t.Fatal("queued replay", err)
			}
			op, v, err := s.beginOperation(op.ID)
			if err != nil || v.Public.ObservedState != "restoring" {
				t.Fatal(err)
			}
			state := restorePhase(t, s, op, v.Runtime, "retaining-destination")
			if outcome != "crash-intent" {
				state = restorePhase(t, s, op, state, "restoring-source")
			}
			if outcome != "crash-intent" && outcome != "crash-retain" {
				state = restorePhase(t, s, op, state, "complete")
			}
			restored := runtimequalify.ManagedObservation{State: "restored", Health: "unknown", RecoveryRequired: true}
			switch outcome {
			case "success":
				err = s.finish(op, state, restored, nil, false)
			case "backend-error":
				err = s.finish(op, state, restored, errors.New("late validation failure"), false)
			case "cancel":
				err = s.finish(op, state, restored, nil, true)
			case "terminal-write-failure":
				if _, err := s.db.Exec(`CREATE TRIGGER fail_restore_receipt BEFORE UPDATE OF body ON operations WHEN json_extract(NEW.body,'$.status')='succeeded' BEGIN SELECT RAISE(ABORT,'terminal fault'); END`); err != nil {
					t.Fatal(err)
				}
				if err := s.finish(op, state, restored, nil, false); err == nil {
					t.Fatal("terminal failure not injected")
				}
				if _, err := s.db.Exec(`DROP TRIGGER fail_restore_receipt`); err != nil {
					t.Fatal(err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			receipt, hit, err := s.accept("operator", "one", req)
			if err != nil || !hit || receipt.ID != op.ID {
				t.Fatal("receipt lost", err)
			}
			if outcome == "success" {
				if receipt.Status != "succeeded" || receipt.ImportRestoration == nil || receipt.ImportRestoration.RestoreTo != "/synthetic/original" {
					t.Fatal("missing restoration result", receipt)
				}
				snapshot, err := s.snapshot()
				if err != nil || len(snapshot.Instances) != 0 {
					t.Fatal("retirement not atomic", err)
				}
				replacement, _, err := s.register("operator", registration("one", "new-registration", 18080))
				if err != nil || replacement.ConfigRevision != 2 {
					t.Fatal("revision not advanced", err)
				}
				if err = s.observe(before, before.Runtime, runtimequalify.ManagedObservation{State: "running"}); err != nil {
					t.Fatal(err)
				}
				if _, hit, err := s.accept("operator", "one", req); err != nil || !hit {
					t.Fatal("historical replay lost", err)
				}
				current, err := loadInstance(s.db, "one")
				if err != nil || current.Runtime.ID == before.Runtime.ID || current.Public.ObservedState != "uninitialized" {
					t.Fatal("old receipt affected replacement", err)
				}
				stale := req
				stale.RequestID = "stale-new-request"
				_, _, err = s.accept("operator", "one", stale)
				requireFault(t, err, "revision_conflict")
				return
			}
			got, err := loadInstance(s.db, "one")
			if err != nil || got.Runtime.ImportRestore == nil || got.Runtime.ImportRestore.Journal == nil || !got.Public.RecoveryRequired || got.Public.DesiredState != "stopped" {
				t.Fatal("lost journal or fence", err)
			}
			if outcome == "backend-error" {
				if receipt.Status != "failed" {
					t.Fatal(receipt)
				}
			} else if receipt.Status != "interrupted" {
				t.Fatal(receipt)
			}
			changed := restoreRequest("changed")
			changed.RestoreDigest = strings.Repeat("e", 64)
			_, _, err = s.accept("operator", "one", changed)
			requireFault(t, err, "restoration_pending")
			for _, kind := range []string{"start", "recover-import", "delete-pier"} {
				_, _, err = s.accept("operator", "one", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1, ConfirmDelete: kind == "delete-pier"})
				requireFault(t, err, "restoration_pending")
			}
			stop, _, err := s.accept("operator", "one", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			stop, stopped, err := s.beginOperation(stop.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.finish(stop, stopped.Runtime, runtimequalify.ManagedObservation{State: "stopped", RecoveryRequired: true}, nil, false); err != nil {
				t.Fatal(err)
			}
			continued, _, err := s.accept("operator", "one", restoreRequest("continue"))
			if err != nil || continued.Kind != protocol.ImportRestoreFeature {
				t.Fatal("continuation refused", err)
			}
		})
	}
}

func TestRestoreRejectsUnapprovedAndInvalidJournalUpdates(t *testing.T) {
	s := testStore(t)
	seedRestore(t, s)
	for _, fault := range []string{"confirmation", "digest", "unrelated"} {
		req := restoreRequest(fault)
		switch fault {
		case "confirmation":
			req.ConfirmRestore = false
		case "digest":
			req.RestoreDigest = ""
		case "unrelated":
			req.Kind = "stop"
		}
		_, _, err := s.accept("operator", "one", req)
		requireFault(t, err, "invalid_request")
	}
	op, _, err := s.accept("operator", "one", restoreRequest("approved"))
	if err != nil {
		t.Fatal(err)
	}
	op, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	forged := v.Runtime
	forged.ImportRestore = &runtimequalify.ManagedImportRestore{Digest: strings.Repeat("d", 64), Journal: &runtimequalify.ImportRestoreJournal{Stage: "complete", Review: restoreFixtureReview(v.Runtime)}}
	if err := s.runtimeUpdate("one", op.ID, forged); err == nil {
		t.Fatal("skipped durable phases")
	}
	if err := s.finish(op, forged, runtimequalify.ManagedObservation{State: "restored"}, nil, false); err == nil {
		t.Fatal("undurable completion retired instance")
	}
	state := restorePhase(t, s, op, v.Runtime, "retaining-destination")
	forged = state
	copy := *state.ImportRestore
	forged.ImportRestore = &copy
	j := *copy.Journal
	copy.Journal = &j
	j.Review.OriginalPath += "-changed"
	if err := s.runtimeUpdate("one", op.ID, forged); err == nil {
		t.Fatal("changed approved review")
	}
}

type restoreRPCBackend struct{ fakeBackend }

func (b *restoreRPCBackend) PreviewImportRestore(_ context.Context, s runtimequalify.ManagedState) (runtimequalify.ImportRestoreReview, error) {
	return restoreFixtureReview(s), nil
}

func TestRestorePreviewRPCIsPrivatePinnedAndReadOnly(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedRestore(t, s)
	if err = s.close(); err != nil {
		t.Fatal(err)
	}
	b := &restoreRPCBackend{}
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
	if err = json.Unmarshal(request("GET", "/hello", "", "", true).Body.Bytes(), &hello); err != nil || !slices.Contains(hello.Features, protocol.ImportRestoreFeature) {
		t.Fatal("feature missing", err)
	}
	path := "/v1/instances/one/restore-import-preview"
	for _, auth := range []struct {
		peer  bool
		grant string
	}{{false, ""}, {true, observer}} {
		if w := request("POST", path, `{"expected_revision":1}`, auth.grant, auth.peer); w.Code != 403 {
			t.Fatal("unauthorized preview", w.Code)
		}
	}
	if w := request("POST", path, `{"expected_revision":2}`, "", true); w.Code != 409 {
		t.Fatal("stale preview", w.Code)
	}
	w := request("POST", path, `{"expected_revision":1}`, "", true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var preview protocol.ImportRestorePreview
	if err = json.Unmarshal(w.Body.Bytes(), &preview); err != nil || preview.RunnerID != hello.RunnerID || preview.ConfigRevision != 1 || preview.DestinationBytes != 20 {
		t.Fatal("preview mismatch", err)
	}
	if strings.Contains(w.Body.String(), "entries") || strings.Contains(w.Body.String(), "identity") {
		t.Fatal("private manifest leaked")
	}
	r.workGate <- struct{}{}
	body, _ := json.Marshal(restoreRequest("rpc-restore"))
	w = request("POST", "/v1/instances/one/operations", string(body), "", true)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request("POST", path, `{"expected_revision":1}`, "", true); w.Code != 409 {
		t.Fatal("busy preview", w.Code)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("bypassed host gate")
	}
}
