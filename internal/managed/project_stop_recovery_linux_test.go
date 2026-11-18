//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type stopRecoveryFixtureBackend struct {
	*expansionFixtureBackend
	observation  protocol.ProjectStopObservation
	observations int
	onObserve    func()
}

func (b *stopRecoveryFixtureBackend) ExpandedProjectStopObservation(_ context.Context, _ runtimequalify.ProjectExpansion, _ string) (protocol.ProjectStopObservation, error) {
	b.observations++
	if b.onObserve != nil {
		b.onObserve()
	}
	o := b.observation
	o.ObservedAt = time.Now().UTC()
	return o, nil
}

func interruptedExpandedStopFixture(t *testing.T, retained bool) (*Server, *stopRecoveryFixtureBackend, protocol.Operation) {
	return interruptedExpandedStopFixtureActor(t, retained, "actor")
}
func interruptedExpandedStopFixtureActor(t *testing.T, retained bool, actor string) (*Server, *stopRecoveryFixtureBackend, protocol.Operation) {
	t.Helper()
	r, original, _, _ := projectQueueFixture(t)
	p, reviewed, _ := readyStopProject(t, r.s)
	selection, err := r.s.projectFleetSelection(p.ID, "")
	g02Check(t, err)
	state, err := runtimequalify.SeedProjectExpansion(selection)
	g02Check(t, err)
	m := state.Members[0]
	pier := memberReadinessProof(t, r.s, p.ID, m.Member.InstanceID).Storage
	started := time.Now().UTC().Add(-time.Minute)
	b := &stopRecoveryFixtureBackend{expansionFixtureBackend: &expansionFixtureBackend{queuedProjectBackend: original}}
	b.shipAction = func(_ context.Context, s *runtimequalify.ProjectExpansion, id, action, op string, save func(runtimequalify.ProjectExpansion) error) error {
		if action != "stop" || id != m.Member.InstanceID {
			t.Error("unselected runtime action")
		}
		for i := range s.Members {
			if s.Members[i].Member.InstanceID == id {
				s.Members[i].Initialized = pier
				s.Members[i].Phase = "stop-dispatched"
				if retained {
					s.Members[i].StopIntent = &runtimequalify.ProjectExpandedStopIntent{OperationID: op, ContainerID: m.Creator.Container.ID, StartedAt: started, RecordedAt: time.Now().UTC()}
				}
			}
		}
		if err := save(*s); err != nil {
			return err
		}
		return errors.New("fixture interrupted exit observation")
	}
	r.b = b
	req := protocol.ProjectActionRequest{Kind: "member-stop", InstanceID: m.Member.InstanceID, ContainerID: m.Creator.Container.ID, ExpectedRevision: p.Revision, ReviewDigest: reviewed.ReviewDigest, RequestID: "stop-for-recovery", ConfirmAction: true}
	op, _, err := r.queueProjectAddition(actor, p.ID, req, false)
	g02Check(t, err)
	waitProjectWorker(t, r)
	op, err = loadOperation(r.s.db, op.ID)
	g02Check(t, err)
	ns := state.Anchor.Namespace
	b.observation = protocol.ProjectStopObservation{ContainerID: m.Creator.Container.ID, ContractDigest: m.Creator.Container.ContractDigest, EngineID: state.Anchor.Prepared.EngineID, AnchorID: state.Anchor.Prepared.Anchor.ID, BridgeID: state.Anchor.Prepared.Bridge.ID, BootID: ns.BootID, NamespaceDevice: ns.Device, NamespaceInode: ns.Inode, Storage: pier, StartedAt: started, FinishedAt: *op.FinishedAt}
	return r, b, op
}

func stopDiagnosisCall(t *testing.T, r *Server, op protocol.Operation, probe bool) protocol.ProjectMemberDiagnosis {
	t.Helper()
	w := httptest.NewRecorder()
	q := httptest.NewRequest("GET", protocol.ProjectMemberDiagnosisPath(op.ProjectID, op.InstanceID, op.ID, probe), nil)
	r.serveProjectMemberDiagnosis(w, q, "actor", "operator", op.ProjectID, op.InstanceID, op.ID, probe)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out protocol.ProjectMemberDiagnosis
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}
func stopReconcileCall(t *testing.T, r *Server, project string, req protocol.ProjectStopReconcileRequest, replay bool) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	g02Check(t, err)
	if len(body) > 4096 {
		t.Fatal("review exceeds restricted SSH request bound", len(body))
	}
	w := httptest.NewRecorder()
	q := httptest.NewRequest("POST", protocol.ProjectStopReconcilePath(project), bytes.NewReader(body))
	q.Header.Set("Content-Type", "application/json")
	if replay {
		q.Header.Set(protocol.CreationReplayHeader, "true")
	}
	r.serveProjectStopReconcile(w, q, "actor", "operator", project)
	return w
}

func TestProjectStopReconciliationPreservesOriginalAndUnlocksOnlyItsFence(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "retained"}[retained], func(t *testing.T) {
			r, b, original := interruptedExpandedStopFixture(t, retained)
			before, err := loadProjectExpansion(r.s.db, original.ProjectID)
			g02Check(t, err)
			saved := stopDiagnosisCall(t, r, original, false)
			if b.observations != 0 || saved.RuntimeObservation != "not-performed" || saved.Phase != "stop-dispatched" || saved.Review != nil || saved.Operation.Message != "" {
				t.Fatal("saved diagnosis probed or exposed private text", saved)
			}
			diagnosis := stopDiagnosisCall(t, r, original, true)
			if diagnosis.Review == nil {
				t.Fatal("no supported clean-stop review", diagnosis)
			}
			g02Check(t, diagnosis.Review.Validate())
			req := protocol.ProjectStopReconcileRequest{RequestID: "reconcile-exact-stop", Review: *diagnosis.Review, ConfirmAction: true}
			w := stopReconcileCall(t, r, original.ProjectID, req, false)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			var result protocol.ProjectStopReconcileResult
			g02Check(t, json.Unmarshal(w.Body.Bytes(), &result))
			if result.Operation.Kind != "reconcile-project-ship-stop" || result.OriginalOperationID != original.ID || result.Operation.Status != "succeeded" || b.observations != 2 {
				t.Fatal(result, b.observations)
			}
			after, err := loadProjectExpansion(r.s.db, original.ProjectID)
			g02Check(t, err)
			for i := range before.Members {
				if before.Members[i].Member.InstanceID == original.InstanceID {
					before.Members[i].Phase = "stopped"
				}
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("reconciliation altered another binding")
			}
			unchanged, err := loadOperation(r.s.db, original.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(unchanged, original) {
				t.Fatal("original interruption was rewritten")
			}
			count, err := projectPendingOperationCount(r.s.db, original.ProjectID, "")
			g02Check(t, err)
			if count != 0 {
				t.Fatal("exact resolved fence remained active", count)
			}
			status, err := r.s.projectStatus(context.Background(), original.ProjectID)
			g02Check(t, err)
			if status.Operations[0].ID != result.Operation.ID {
				t.Fatal("resolution missing from public history")
			}
			resolved := stopDiagnosisCall(t, r, original, false)
			if resolved.ResolvedBy != result.Operation.ID || resolved.ReasonCode != "stop_reconciled" {
				t.Fatal(resolved)
			}
			r.b = nil
			r.failed.Store(true)
			repeated := stopReconcileCall(t, r, original.ProjectID, req, true)
			if repeated.Code != 200 || repeated.Body.String() != w.Body.String() || b.observations != 2 {
				t.Fatal("lost response lookup changed effects", repeated.Code, repeated.Body.String())
			}
			r.failed.Store(false)
			r.b = b
			// Exercise ordinary successor admission through the normal action path.
			// The backend is inert; real same-pier replacement remains a separate gate.
			starts := 0
			b.shipAction = func(_ context.Context, state *runtimequalify.ProjectExpansion, id, action, _ string, save func(runtimequalify.ProjectExpansion) error) error {
				if action != "start" || id != original.InstanceID {
					t.Error("unselected action after reconciliation")
				}
				starts++
				for i := range state.Members {
					if state.Members[i].Member.InstanceID == id {
						state.Members[i].Phase = "ready"
					}
				}
				return save(*state)
			}
			digest, err := after.Project.Plan.Digest()
			g02Check(t, err)
			var cid string
			for _, m := range after.Members {
				if m.Member.InstanceID == original.InstanceID {
					cid = m.Creator.Container.ID
				}
			}
			start, _, err := r.queueProjectAddition("actor", original.ProjectID, protocol.ProjectActionRequest{Kind: "member-start", InstanceID: original.InstanceID, ContainerID: cid, RequestID: "ordinary-start-after-reconciliation", ExpectedRevision: after.Project.Revision, ReviewDigest: digest, ConfirmAction: true}, false)
			g02Check(t, err)
			waitProjectWorker(t, r)
			completed, err := loadOperation(r.s.db, start.ID)
			g02Check(t, err)
			if starts != 1 || completed.Status != "succeeded" {
				t.Fatal("normal Start remained fenced", completed)
			}
			g02Check(t, r.Close())
			s, err := testOpenStore(r.s.directory)
			g02Check(t, err)
			defer s.close()
			replay, hit, err := s.replayProjectStopResolution("actor", original.ProjectID, req)
			if err != nil || !hit || !reflect.DeepEqual(replay, result) {
				t.Fatal("restart lost resolution", err)
			}
		})
	}
}

func TestProjectStopResolutionCorruptionRefusesAdmissionAndReopen(t *testing.T) {
	for _, mode := range []string{"missing-record", "missing-table", "original-rewritten", "request-rebound", "review-rebound", "stop-intent-rebound"} {
		t.Run(mode, func(t *testing.T) {
			r, _, original := interruptedExpandedStopFixture(t, true)
			d := stopDiagnosisCall(t, r, original, true)
			if d.Review == nil {
				t.Fatal(d)
			}
			req := protocol.ProjectStopReconcileRequest{RequestID: "corruption-fixture-resolution", Review: *d.Review, ConfirmAction: true}
			w := stopReconcileCall(t, r, original.ProjectID, req, false)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			var result protocol.ProjectStopReconcileResult
			g02Check(t, json.Unmarshal(w.Body.Bytes(), &result))
			var err error
			switch mode {
			case "missing-record":
				_, err = r.s.db.Exec("DELETE FROM project_stop_reconciliations")
			case "missing-table":
				_, err = r.s.db.Exec("DROP TABLE project_stop_reconciliations")
			case "original-rewritten":
				_, err = r.s.db.Exec("UPDATE operations SET body=json_set(body,'$.status','succeeded') WHERE id=?", original.ID)
			case "request-rebound":
				_, err = r.s.db.Exec("UPDATE operations SET fingerprint='changed' WHERE id=?", result.Operation.ID)
			case "review-rebound":
				_, err = r.s.db.Exec("UPDATE project_stop_reconciliations SET body=json_set(body,'$.request.review.ledger_digest',?)", strings.Repeat("f", 64))
			case "stop-intent-rebound":
				_, err = r.s.db.Exec("UPDATE project_expansions SET body=json_set(body,'$.members[0].stop_intent.container_id',?) WHERE project_id=?", strings.Repeat("f", 64), original.ProjectID)
			}
			g02Check(t, err)
			if mode != "missing-record" && mode != "stop-intent-rebound" {
				if _, err = projectPendingOperationCount(r.s.db, original.ProjectID, ""); err == nil {
					t.Fatal("corrupt receipt cleared admission")
				}
			}
			dir := r.s.directory
			g02Check(t, r.Close())
			if s, err := testOpenStore(dir); err == nil {
				s.close()
				t.Fatal("corrupt stop history accepted", mode)
			}
		})
	}
}

func TestProjectStopReconciliationRefusesChangedOrAbnormalEvidence(t *testing.T) {
	for _, mode := range []string{"running", "nonzero", "oom", "container", "pier", "namespace", "process", "stale", "actor", "confirmation", "concurrent-ledger", "original-container", "other-operation", "missing-lookup", "rollback"} {
		t.Run(mode, func(t *testing.T) {
			r, b, original := interruptedExpandedStopFixture(t, true)
			d := stopDiagnosisCall(t, r, original, true)
			if d.Review == nil {
				t.Fatal(d)
			}
			req := protocol.ProjectStopReconcileRequest{RequestID: "refused-recovery", Review: *d.Review, ConfirmAction: true}
			before, err := loadProjectExpansion(r.s.db, original.ProjectID)
			g02Check(t, err)
			switch mode {
			case "running":
				b.observation.Running = true
			case "nonzero":
				b.observation.ExitCode = 1
			case "oom":
				b.observation.ExitCode = 137
				b.observation.OOMKilled = true
			case "container":
				b.observation.ContainerID = strings.Repeat("f", 64)
			case "pier":
				b.observation.Storage.Identity.Inode++
			case "namespace":
				b.observation.NamespaceInode++
			case "process":
				b.observation.StartedAt = b.observation.StartedAt.Add(time.Second)
			case "stale":
				req.Review.Observation.ObservedAt = time.Now().UTC().Add(-10 * time.Minute)
				req.Review.Observation.FinishedAt = req.Review.Observation.ObservedAt.Add(-time.Second)
				req.Review.Observation.StartedAt = req.Review.Observation.FinishedAt.Add(-time.Minute)
				req.Review.ReviewDigest = req.Review.Digest()
			case "actor":
				req.Review.Actor = "another-operator"
				req.Review.ReviewDigest = req.Review.Digest()
			case "confirmation":
				req.ConfirmAction = false
			case "concurrent-ledger":
				b.onObserve = func() {
					before.Members[1].Phase = "stopped"
					body, _ := json.Marshal(before)
					_, err := r.s.db.Exec("UPDATE project_expansions SET body=? WHERE project_id=?", body, original.ProjectID)
					g02Check(t, err)
				}
			case "other-operation":
				pending := original
				pending.ID = "op-other-interrupted"
				pending.RequestID = "other-interruption"
				tx, err := r.s.db.Begin()
				g02Check(t, err)
				g02Check(t, insertOperation(tx, pending, "other-input"))
				g02Check(t, tx.Commit())
			case "original-container":
				_, err = r.s.db.Exec("UPDATE operations SET fingerprint=json_set(fingerprint,'$.Request.container_id',?) WHERE id=?", strings.Repeat("f", 64), original.ID)
				g02Check(t, err)
			case "rollback":
				_, err = r.s.db.Exec("CREATE TRIGGER refuse_stop_resolution BEFORE INSERT ON project_stop_reconciliations BEGIN SELECT RAISE(ABORT,'fixture'); END")
				g02Check(t, err)
			}
			w := stopReconcileCall(t, r, original.ProjectID, req, mode == "missing-lookup")
			if w.Code == 200 {
				t.Fatal("unsafe recovery accepted", mode, w.Body.String())
			}
			expected := map[string]string{"running": "stop_member_running", "nonzero": "stop_exit_abnormal", "oom": "stop_exit_abnormal", "container": "stop_recovery_identity_changed", "pier": "stop_recovery_identity_changed", "namespace": "stop_recovery_identity_changed", "process": "stop_process_changed", "stale": "stop_review_invalid", "actor": "invalid_request", "confirmation": "invalid_request", "concurrent-ledger": "stop_review_changed", "original-container": "stop_original_request_changed", "other-operation": "project_busy", "missing-lookup": "replay_not_found"}[mode]
			if expected != "" && !strings.Contains(w.Body.String(), expected) {
				t.Fatal("wrong refusal", mode, w.Code, w.Body.String())
			}
			if mode == "rollback" && w.Code != 500 {
				t.Fatal("rollback did not reach transaction", w.Code, w.Body.String())
			}
			after, err := loadProjectExpansion(r.s.db, original.ProjectID)
			g02Check(t, err)
			if !reflect.DeepEqual(after, before) {
				t.Fatal("refused recovery altered ledger")
			}
			previous, err := loadOperation(r.s.db, original.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(previous, original) {
				t.Fatal("failure rewrote original")
			}
			var count int
			g02Check(t, r.s.db.QueryRow("SELECT count(*) FROM project_stop_reconciliations").Scan(&count))
			if count != 0 {
				t.Fatal("partial resolution survived")
			}
		})
	}
}
