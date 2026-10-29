package managed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// The request binds the same reviewed parent/history/checkpoint fields as a read
// attempt, but is fingerprinted under a distinct mutating operation kind.
type projectRecoveryContinuationRequest = projectRecoveryObservationRequest

type projectRecoveryContinuationJob struct {
	OperationID         string                                    `json:"operation_id"`
	ProjectID           string                                    `json:"project_id"`
	RecoveryOperationID string                                    `json:"recovery_operation_id"`
	SourceDigest        string                                    `json:"source_digest"`
	Target              projectnetwork.RecoveryContinuationTarget `json:"target"`
	Phase               string                                    `json:"phase"`
	ParentBefore        protocol.Operation                        `json:"parent_before"`
	Before              projectnetwork.AnchorRecoveryJournal      `json:"before"`
	Latest              *projectnetwork.AnchorRecoveryJournal     `json:"latest,omitempty"`
}

func initializeProjectRecoveryContinuations(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_recovery_continuations(operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES projects(id),recovery_operation_id TEXT NOT NULL REFERENCES project_anchor_recoveries(operation_id),body BLOB NOT NULL);`)
	return err
}
func loadProjectRecoveryContinuation(q querier, id string) (projectRecoveryContinuationJob, error) {
	var j projectRecoveryContinuationJob
	var project, parent string
	var body []byte
	if err := q.QueryRow("SELECT project_id,recovery_operation_id,body FROM project_recovery_continuations WHERE operation_id=?", id).Scan(&project, &parent, &body); err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.RecoveryOperationID != parent {
		return j, errors.New("invalid recovery continuation ownership")
	}
	return j, nil
}
func projectRecoveryContinuationJobs(q querier) ([]projectRecoveryContinuationJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_recovery_continuations ORDER BY operation_id")
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	var jobs []projectRecoveryContinuationJob
	for _, id := range ids {
		j, err := loadProjectRecoveryContinuation(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func validateRecoveryContinuation(q querier, j projectRecoveryContinuationJob) error {
	fail := func() error {
		return errors.New("recovery continuation differs from its actual parent, actor or reviewed checkpoint")
	}
	parent, err := loadProjectAnchorRecovery(q, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	if err = validateRecoveryReceipt(q, parent); err != nil {
		return err
	}
	if err = recoverySourceReferences(q, parent); err != nil {
		return err
	}
	if parent.ProjectID != j.ProjectID || j.SourceDigest != recoveryHistoryDigest(parent.Source) || !projectnetwork.RecoveryJournalContainsCheckpoint(parent.Source, parent.Journal, j.Before) {
		return fail()
	}
	target, err := projectnetwork.RecoveryCheckpointContinuationTarget(parent.Source, j.Before)
	if err != nil || target != j.Target {
		return fail()
	}
	p, err := loadProject(q, j.ProjectID)
	if err != nil {
		return err
	}
	owner, err := loadOperation(q, parent.OperationID)
	if err != nil {
		return err
	}
	// Preserve the earlier interruption and all immutable original actor/request fields.
	old := owner
	old.Status = j.ParentBefore.Status
	old.Message = j.ParentBefore.Message
	old.FinishedAt = j.ParentBefore.FinishedAt
	if j.ParentBefore.Status != "interrupted" || j.ParentBefore.FinishedAt == nil || !reflect.DeepEqual(old, j.ParentBefore) {
		return fail()
	}
	op, err := loadOperation(q, j.OperationID)
	if err != nil {
		return err
	}
	var actor, request, fingerprint string
	if err = q.QueryRow("SELECT actor,request_id,fingerprint FROM operations WHERE id=?", op.ID).Scan(&actor, &request, &fingerprint); err != nil {
		return err
	}
	var intent struct {
		Kind, Project string
		Request       projectRecoveryContinuationRequest
	}
	digest, _ := p.Plan.Digest()
	if json.Unmarshal([]byte(fingerprint), &intent) != nil || actor == "" || actor != op.Actor || request != op.RequestID || op.Kind != "continue-project-recovery" || op.ProjectID != j.ProjectID || op.InstanceID != "" || op.ConfigRevision != p.Revision || intent.Kind != op.Kind || intent.Project != j.ProjectID || intent.Request.RequestID != op.RequestID || intent.Request.RecoveryOperationID != j.RecoveryOperationID || intent.Request.ExpectedRevision != p.Revision || intent.Request.ExpectedSequence != j.Before.Sequence || intent.Request.HistoryDigest != j.SourceDigest || intent.Request.CheckpointDigest != recoveryCheckpointDigest(j.Before) || intent.Request.ReviewDigest != digest {
		return fail()
	}
	if j.Latest != nil && (!projectnetwork.RecoveryJournalContainsCheckpoint(parent.Source, *j.Latest, j.Before) || !projectnetwork.RecoveryJournalContainsCheckpoint(parent.Source, parent.Journal, *j.Latest)) {
		return fail()
	}
	switch j.Phase {
	case "planned":
		if j.Latest != nil || op.Status != "queued" && op.Status != "interrupted" {
			return fail()
		}
		if op.Status == "queued" && (!reflect.DeepEqual(owner, j.ParentBefore) || !reflect.DeepEqual(parent.Journal, j.Before)) {
			return fail()
		}
	case "executing":
		if j.Latest == nil || op.Status != "running" && op.Status != "interrupted" {
			return fail()
		}
		if op.Status == "running" && (owner.Status != "running" || !reflect.DeepEqual(parent.Journal, *j.Latest)) {
			return fail()
		}
	case "complete":
		if j.Latest == nil || j.Latest.Phase != "complete" || op.Status != "succeeded" || owner.Status != "succeeded" {
			return fail()
		}
	default:
		return fail()
	}
	return projectRecoveryMirror(q, j.OperationID, j)
}
func validateProjectRecoveryContinuations(q querier) error {
	return validateSnapshot(q, "validateProjectRecoveryContinuations", validateProjectRecoveryContinuationsUncached)
}

func validateProjectRecoveryContinuationsUncached(q querier) error {
	jobs, err := projectRecoveryContinuationJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		if err = validateRecoveryContinuation(q, j); err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" {
			if active[j.ProjectID] {
				return errors.New("overlapping recovery continuations")
			}
			active[j.ProjectID] = true
		}
	}
	var orphan int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='continue-project-recovery' AND id NOT IN (SELECT operation_id FROM project_recovery_continuations)").Scan(&orphan); err != nil {
		return err
	}
	if orphan != 0 {
		return errors.New("recovery continuation is missing its owning journal")
	}
	return nil
}
func saveProjectRecoveryContinuation(tx *sql.Tx, j projectRecoveryContinuationJob, op protocol.Operation) error {
	raw, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_recovery_continuations SET body=? WHERE operation_id=? AND project_id=? AND recovery_operation_id=?", raw, j.OperationID, j.ProjectID, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("recovery continuation lost ownership")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID); err != nil {
		return err
	}
	return saveOperation(tx, op)
}
func (s *store) acceptProjectRecoveryContinuation(actor, project string, req projectRecoveryContinuationRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.RecoveryOperationID) || req.ExpectedRevision == 0 || len(req.CheckpointDigest) != 64 || len(req.HistoryDigest) != 64 {
		return protocol.Operation{}, false, bad("recovery continuation requires an exact reviewed parent checkpoint")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project string
		Request       projectRecoveryContinuationRequest
	}{"continue-project-recovery", project, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching recovery continuation exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = validateProjectAnchorRecoveries(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectRecoveryObservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectRecoveryContinuations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectRecoveryReceiptAdmission(tx, project, req.RecoveryOperationID); err != nil {
		return protocol.Operation{}, false, err
	}
	parent, err := loadProjectAnchorRecovery(tx, req.RecoveryOperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	owner, err := loadOperation(tx, parent.OperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	p, err := loadProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	digest, _ := p.Plan.Digest()
	if parent.ProjectID != project || owner.Status != "interrupted" || req.ExpectedRevision != p.Revision || req.ReviewDigest != digest || req.ExpectedSequence != parent.Journal.Sequence || req.CheckpointDigest != recoveryCheckpointDigest(parent.Journal) || req.HistoryDigest != recoveryHistoryDigest(parent.Source) {
		return protocol.Operation{}, false, conflict("revision_conflict", "review the current recovery checkpoint before continuing")
	}
	if err = projectRetirementClear(tx, project, parent.OperationID); err != nil {
		return protocol.Operation{}, false, err
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish the active project work before a continuation")
	}
	target, err := projectnetwork.RecoveryCheckpointContinuationTarget(parent.Source, parent.Journal)
	if err != nil {
		return protocol.Operation{}, false, conflict("project_recovery_review_required", err.Error())
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, Kind: "continue-project-recovery", ProjectID: project, Actor: actor, RequestID: req.RequestID, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectRecoveryContinuationJob{OperationID: op.ID, ProjectID: project, RecoveryOperationID: parent.OperationID, SourceDigest: req.HistoryDigest, Target: target, Phase: "planned", ParentBefore: owner, Before: parent.Journal.Clone()}
	if _, err = tx.Exec("INSERT INTO project_recovery_continuations(operation_id,project_id,recovery_operation_id,body) VALUES(?,?,?,?)", op.ID, project, parent.OperationID, []byte("{}")); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectRecoveryContinuation(tx, j, op); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, tx.Commit()
}
func (s *store) beginProjectRecoveryContinuation(id string) (projectRecoveryContinuationJob, error) {
	var empty projectRecoveryContinuationJob
	tx, err := s.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	if err = validateProjectRecoveryContinuations(tx); err != nil {
		return empty, err
	}
	j, err := loadProjectRecoveryContinuation(tx, id)
	if err != nil {
		return empty, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return empty, err
	}
	if op.Status != "queued" || j.Phase != "planned" {
		return empty, errors.New("continuation is not freshly queued")
	}
	if err = projectRetirementClear(tx, j.ProjectID, j.RecoveryOperationID); err != nil {
		return empty, err
	}
	parent, err := loadProjectAnchorRecovery(tx, j.RecoveryOperationID)
	if err != nil {
		return empty, err
	}
	owner, err := loadOperation(tx, parent.OperationID)
	if err != nil {
		return empty, err
	}
	owner.Status = "running"
	owner.FinishedAt = nil
	owner.Message = "explicit continuation admitted from retained checkpoint"
	if err = saveProjectAnchorRecovery(tx, parent, owner); err != nil {
		return empty, err
	}
	latest := j.Before.Clone()
	j.Latest = &latest
	j.Phase = "executing"
	op.Status = "running"
	if err = saveProjectRecoveryContinuation(tx, j, op); err != nil {
		return empty, err
	}
	if err = validateRecoveryContinuation(tx, j); err != nil {
		return empty, err
	}
	return j, tx.Commit()
}
func (s *store) advanceProjectRecoveryContinuation(id string, expected uint64, phase string, u projectnetwork.AnchorRecoveryJournal) (projectAnchorRecoveryJob, error) {
	var empty projectAnchorRecoveryJob
	tx, err := s.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	j, err := loadProjectRecoveryContinuation(tx, id)
	if err != nil {
		return empty, err
	}
	if err = validateRecoveryContinuation(tx, j); err != nil {
		return empty, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return empty, err
	}
	if op.Status != "running" || j.Phase != "executing" || j.Latest == nil || expected != j.Latest.Sequence {
		return empty, errors.New("stale or inactive continuation")
	}
	parent, err := loadProjectAnchorRecovery(tx, j.RecoveryOperationID)
	if err != nil {
		return empty, err
	}
	owner, err := loadOperation(tx, parent.OperationID)
	if err != nil {
		return empty, err
	}
	if err = projectRetirementClear(tx, j.ProjectID, parent.OperationID); err != nil {
		return empty, err
	}
	h, _, _, err := projectRecoveryHistory(tx, j.ProjectID, false)
	if err != nil {
		return empty, err
	}
	h.Current = parent.Source.Current
	if !reflect.DeepEqual(h, parent.Source) {
		return empty, errors.New("recovery history changed during continuation")
	}
	next, err := parent.Journal.Advance(parent.Source, expected, phase, u)
	if err != nil {
		return empty, err
	}
	parent.Journal = next
	latest := next.Clone()
	j.Latest = &latest
	if next.Phase == "complete" {
		now := time.Now().UTC()
		owner.Status = "succeeded"
		owner.FinishedAt = &now
		owner.Message = "explicit recovery continuation completed with fresh readiness"
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "recovery continued without repeating uncertain runtime effects"
		j.Phase = "complete"
	}
	if err = saveProjectAnchorRecovery(tx, parent, owner); err != nil {
		return empty, err
	}
	if err = saveProjectRecoveryContinuation(tx, j, op); err != nil {
		return empty, err
	}
	// Validate the bytes actually persisted; JSON intentionally omits Go's
	// process-local monotonic clock component from fresh readiness timestamps.
	persisted, err := loadProjectRecoveryContinuation(tx, j.OperationID)
	if err != nil {
		return empty, err
	}
	if err = validateRecoveryContinuation(tx, persisted); err != nil {
		return empty, err
	}
	return parent, tx.Commit()
}
func interruptProjectRecoveryContinuation(tx *sql.Tx, id string) error {
	j, err := loadProjectRecoveryContinuation(tx, id)
	if err != nil {
		return err
	}
	if err = validateRecoveryContinuation(tx, j); err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status == "succeeded" || op.Status == "interrupted" {
		return nil
	}
	now := time.Now().UTC()
	if op.Status == "running" {
		parent, err := loadProjectAnchorRecovery(tx, j.RecoveryOperationID)
		if err != nil {
			return err
		}
		owner, err := loadOperation(tx, parent.OperationID)
		if err != nil {
			return err
		}
		owner.Status = "interrupted"
		owner.FinishedAt = &now
		owner.Message = "continuation interrupted; preserve the latest checkpoint without automatic runtime replay"
		if err = saveProjectAnchorRecovery(tx, parent, owner); err != nil {
			return err
		}
	}
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "continuation interrupted; review its exact retained outcome before another request"
	return saveProjectRecoveryContinuation(tx, j, op)
}
func (s *store) interruptRecoveryContinuation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectRecoveryContinuation(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func reconcileProjectRecoveryContinuations(tx *sql.Tx) error {
	if err := validateProjectRecoveryContinuations(tx); err != nil {
		return err
	}
	jobs, err := projectRecoveryContinuationJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectRecoveryContinuation(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}

// Ordinary first-run callbacks cannot bypass the continuation's atomic Latest
// mirror. Both ordinary advancement and interruption check this ownership.
func projectRecoveryContinuationInactive(q querier, parent string) error {
	var active int
	if err := q.QueryRow("SELECT count(*) FROM project_recovery_continuations j JOIN operations o ON o.id=j.operation_id WHERE j.recovery_operation_id=? AND json_extract(o.body,'$.status') IN ('queued','running')", parent).Scan(&active); err != nil {
		return err
	}
	if active != 0 {
		return errors.New("explicit continuation owns recovery progress")
	}
	return nil
}
