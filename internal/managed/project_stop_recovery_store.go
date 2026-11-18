package managed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type projectStopResolution struct {
	Request protocol.ProjectStopReconcileRequest `json:"request"`
	Result  protocol.ProjectStopReconcileResult  `json:"result"`
}

func stopEvidenceDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
func loadProjectStopResolution(q querier, original string) (projectStopResolution, error) {
	var out projectStopResolution
	var body []byte
	var id, project string
	if err := q.QueryRow("SELECT id,project_id,body FROM project_stop_reconciliations WHERE original_operation_id=?", original).Scan(&id, &project, &body); err != nil {
		return out, err
	}
	if json.Unmarshal(body, &out) != nil || out.Request.Review.Validate() != nil || !out.Request.ConfirmAction || !protocol.ValidID(out.Request.RequestID) {
		return out, errors.New("invalid project stop resolution")
	}
	review, result := out.Request.Review, out.Result
	op, err := loadOperation(q, id)
	if err != nil {
		return out, err
	}
	var storedActor, storedRequest, fingerprint string
	if err = q.QueryRow("SELECT actor,request_id,fingerprint FROM operations WHERE id=?", id).Scan(&storedActor, &storedRequest, &fingerprint); err != nil {
		return out, err
	}
	if op.ID != id || op.Actor != storedActor || op.RequestID != storedRequest || fingerprint != stopReconcileFingerprint(project, out.Request) {
		return out, errors.New("project stop reconciliation request ancestry changed")
	}
	previous, err := loadOperation(q, original)
	if err != nil {
		return out, err
	}
	var originalBody []byte
	var originalActor, originalRequest, originalFingerprint string
	if err = q.QueryRow("SELECT body,actor,request_id,fingerprint FROM operations WHERE id=?", original).Scan(&originalBody, &originalActor, &originalRequest, &originalFingerprint); err != nil {
		return out, err
	}
	if previous.ID != original || previous.Actor != originalActor || previous.RequestID != originalRequest || stopEvidenceDigest([]byte(originalFingerprint)) != review.OriginalRequestDigest {
		return out, errors.New("original stop request ancestry changed")
	}
	if !reflect.DeepEqual(op, result.Operation) || result.SchemaVersion != 1 || result.RunnerID != review.RunnerID || result.Mode != review.Mode || result.OriginalOperationID != original || result.ReviewDigest != review.ReviewDigest || result.VerifiedState != "stopped" || op.ProjectID != project || project != review.ProjectID || op.InstanceID != review.InstanceID || op.Actor != review.Actor || op.RequestID != out.Request.RequestID || op.Kind != "reconcile-project-ship-stop" || op.Status != "succeeded" || op.ConfigRevision != review.ProjectRevision || op.CreatedAt.IsZero() || op.FinishedAt == nil || !op.FinishedAt.Equal(op.CreatedAt) || review.OriginalOperationID != original || stopEvidenceDigest(originalBody) != review.OperationDigest || previous.Status != "interrupted" || previous.Kind != "stop-project-ship" || previous.Actor != op.Actor || previous.ProjectID != project || previous.InstanceID != op.InstanceID {
		return out, errors.New("project stop resolution ancestry changed")
	}
	return out, nil
}

func initializeProjectStopReconciliations(tx *sql.Tx, version int) error {
	if version < 60 {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_stop_reconciliations (
id TEXT PRIMARY KEY REFERENCES operations(id),
original_operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id),
project_id TEXT NOT NULL REFERENCES projects(id), body BLOB NOT NULL)`); err != nil {
			return err
		}
	}
	rows, err := tx.Query("SELECT original_operation_id FROM project_stop_reconciliations")
	if err != nil {
		return err
	}
	var originals []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		originals = append(originals, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return err
	}
	for _, id := range originals {
		if _, err = loadProjectStopResolution(tx, id); err != nil {
			return err
		}
	}
	var missing int
	if err = tx.QueryRow("SELECT count(*) FROM operations o LEFT JOIN project_stop_reconciliations r ON r.id=o.id WHERE json_extract(o.body,'$.kind')='reconcile-project-ship-stop' AND r.id IS NULL").Scan(&missing); err != nil {
		return err
	}
	if missing != 0 {
		return errors.New("project stop resolution evidence is missing")
	}
	all, err := allProjectExpansions(tx)
	if err != nil {
		return err
	}
	for _, state := range all {
		for _, member := range append(state.Members, state.History...) {
			if intent := member.StopIntent; intent != nil {
				op, err := loadOperation(tx, intent.OperationID)
				if err != nil {
					return err
				}
				if op.Kind != "stop-project-ship" || op.ProjectID != state.Project.ProjectID || op.InstanceID != member.Member.InstanceID || op.ConfigRevision != state.Project.Revision || intent.ContainerID != member.Creator.Container.ID || intent.StartedAt.IsZero() || intent.RecordedAt.Before(intent.StartedAt) || intent.RecordedAt.Before(op.CreatedAt) {
					return errors.New("retained project stop process ancestry changed")
				}
			}
		}
	}
	return nil
}

// Reads the original receipt before runtime availability checks. A lost reply
// cannot turn a repeated exact request into another observation or mutation.
func (s *store) replayProjectStopResolution(actor, project string, req protocol.ProjectStopReconcileRequest) (protocol.ProjectStopReconcileResult, bool, error) {
	fingerprint := stopReconcileFingerprint(project, req)
	op, hit, err := replay(s.db, actor, req.RequestID, fingerprint)
	if err != nil || !hit {
		return protocol.ProjectStopReconcileResult{}, hit, err
	}
	r, err := loadProjectStopResolution(s.db, req.Review.OriginalOperationID)
	if err != nil || r.Result.Operation.ID != op.ID {
		return protocol.ProjectStopReconcileResult{}, true, errors.New("project stop reconciliation receipt mismatch")
	}
	return r.Result, true, nil
}
func stopReconcileFingerprint(project string, req protocol.ProjectStopReconcileRequest) string {
	body, _ := json.Marshal(struct {
		Project string
		Request protocol.ProjectStopReconcileRequest
	}{project, req})
	return string(body)
}

func projectPendingOperationCount(q querier, project, excluded string) (int, error) {
	// A row cannot suppress a fence merely by naming an interrupted operation.
	// Validate its linked receipt before admission, including in the same process
	// that created it; reopening is not the only corruption detection boundary.
	rows, err := q.Query("SELECT original_operation_id FROM project_stop_reconciliations WHERE project_id=?", project)
	if err != nil {
		return 0, err
	}
	var originals []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		originals = append(originals, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return 0, err
	}
	for _, id := range originals {
		if _, err = loadProjectStopResolution(q, id); err != nil {
			return 0, err
		}
	}
	var count int
	err = q.QueryRow(`SELECT count(*) FROM operations o
WHERE json_extract(o.body,'$.project_id')=? AND o.id!=?
AND json_extract(o.body,'$.status') IN ('queued','running','interrupted')
AND NOT EXISTS(SELECT 1 FROM project_stop_reconciliations r WHERE r.original_operation_id=o.id)`, project, excluded).Scan(&count)
	return count, err
}

// The caller has just made a fresh bracketed observation under runner admission.
// This transaction changes only the selected phase and adds its linked receipt.
func (s *store) commitProjectStopResolution(actor, project string, req protocol.ProjectStopReconcileRequest, fresh protocol.ProjectStopReview) (protocol.ProjectStopReconcileResult, error) {
	var zero protocol.ProjectStopReconcileResult
	if fresh.Validate() != nil {
		return zero, bad("fresh clean-stop evidence is invalid")
	}
	reviewed := req.Review
	at := time.Now().UTC()
	if reviewed.Validate() != nil || !req.ConfirmAction || !protocol.ValidID(req.RequestID) || reviewed.Actor != actor || reviewed.ProjectID != project || at.Sub(reviewed.Observation.ObservedAt) > protocol.ProjectStopReviewMaxAge || reviewed.Observation.ObservedAt.After(at.Add(5*time.Second)) {
		return zero, conflict("stop_review_invalid", "require a current exact stop review and its original operator")
	}
	// A new observation time is expected; every other reviewed binding must match.
	fresh.Observation.ObservedAt = reviewed.Observation.ObservedAt
	fresh.ReviewDigest = fresh.Digest()
	if !reflect.DeepEqual(fresh, reviewed) {
		return zero, conflict("stop_review_changed", "stop evidence changed; review again before reconciling")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	selected, err := s.projectStopSelection(tx, actor, project, reviewed.InstanceID, reviewed.OriginalOperationID)
	if err != nil {
		return zero, err
	}
	if selected.review.LedgerDigest != reviewed.LedgerDigest || selected.review.OperationDigest != reviewed.OperationDigest {
		return zero, conflict("stop_review_changed", "retained stop state changed after observation")
	}
	if active, err := projectPendingOperationCount(tx, project, reviewed.OriginalOperationID); err != nil {
		return zero, err
	} else if active != 0 {
		return zero, conflict("project_busy", "another project operation needs review")
	}
	for i := range selected.state.Members {
		if selected.state.Members[i].Member.InstanceID == reviewed.InstanceID {
			selected.state.Members[i].Phase = "stopped"
		}
	}
	body, err := json.Marshal(selected.state)
	if err != nil {
		return zero, err
	}
	changed, err := tx.Exec("UPDATE project_expansions SET body=? WHERE project_id=? AND CAST(body AS BLOB)=?", body, project, selected.ledgerBody)
	if err != nil {
		return zero, err
	}
	if n, err := changed.RowsAffected(); err != nil || n != 1 {
		return zero, conflict("stop_review_changed", "member ledger changed before commit")
	}
	id, err := randomID()
	if err != nil {
		return zero, err
	}
	op := protocol.Operation{ID: "op-" + id, RequestID: req.RequestID, ProjectID: project, InstanceID: reviewed.InstanceID, Kind: "reconcile-project-ship-stop", Actor: actor, ConfigRevision: reviewed.ProjectRevision, Status: "succeeded", CreatedAt: at, FinishedAt: &at, Message: "Exact interrupted stop reconciled; clean stopped state verified. No signal or Start dispatched."}
	if err = insertOperation(tx, op, stopReconcileFingerprint(project, req)); err != nil {
		return zero, err
	}
	result := protocol.ProjectStopReconcileResult{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode, OriginalOperationID: reviewed.OriginalOperationID, ReviewDigest: reviewed.ReviewDigest, VerifiedState: "stopped", Operation: op}
	record, _ := json.Marshal(projectStopResolution{Request: req, Result: result})
	if _, err = tx.Exec("INSERT INTO project_stop_reconciliations(id,original_operation_id,project_id,body) VALUES(?,?,?,?)", op.ID, reviewed.OriginalOperationID, project, record); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return result, nil
}
