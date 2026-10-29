package managed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type projectRecoveryObservationRequest struct {
	RequestID           string `json:"request_id"`
	RecoveryOperationID string `json:"recovery_operation_id"`
	ExpectedRevision    uint64 `json:"expected_revision"`
	ExpectedSequence    uint64 `json:"expected_sequence"`
	ReviewDigest        string `json:"review_digest"`
	HistoryDigest       string `json:"history_digest"`
	CheckpointDigest    string `json:"checkpoint_digest"`
	ReplayOnly          bool   `json:"-"`
}
type projectRecoveryObservationJob struct {
	OperationID         string                                   `json:"operation_id"`
	ProjectID           string                                   `json:"project_id"`
	RecoveryOperationID string                                   `json:"recovery_operation_id"`
	SourceDigest        string                                   `json:"source_digest"`
	Target              projectnetwork.RecoveryObservationTarget `json:"target"`
	Phase               string                                   `json:"phase"`
	Before              projectnetwork.AnchorRecoveryJournal     `json:"before"`
	After               *projectnetwork.AnchorRecoveryJournal    `json:"after,omitempty"`
}

func recoveryCheckpointDigest(j projectnetwork.AnchorRecoveryJournal) string {
	b, _ := json.Marshal(j)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func initializeProjectRecoveryObservations(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_recovery_observations(operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES projects(id),recovery_operation_id TEXT NOT NULL REFERENCES project_anchor_recoveries(operation_id),body BLOB NOT NULL);`)
	return err
}
func loadProjectRecoveryObservation(q querier, id string) (projectRecoveryObservationJob, error) {
	var j projectRecoveryObservationJob
	var project, parent string
	var body []byte
	err := q.QueryRow("SELECT project_id,recovery_operation_id,body FROM project_recovery_observations WHERE operation_id=?", id).Scan(&project, &parent, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.RecoveryOperationID != parent {
		return j, errors.New("invalid recovery observation ownership")
	}
	return j, nil
}
func projectRecoveryObservationJobs(q querier) ([]projectRecoveryObservationJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_recovery_observations ORDER BY operation_id")
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
	var jobs []projectRecoveryObservationJob
	for _, id := range ids {
		j, err := loadProjectRecoveryObservation(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func validateRecoveryObservation(q querier, j projectRecoveryObservationJob) error {
	fail := func() error {
		return errors.New("recovery observation differs from its actual parent, actor or checkpoint")
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
	target, err := projectnetwork.RecoveryCheckpointObservationTarget(parent.Source, j.Before)
	if err != nil || target != j.Target {
		return fail()
	}
	p, err := loadProject(q, j.ProjectID)
	if err != nil {
		return err
	}
	op, err := loadOperation(q, j.OperationID)
	if err != nil {
		return err
	}
	var actor, request, fingerprint string
	if err = q.QueryRow("SELECT actor,request_id,fingerprint FROM operations WHERE id=?", j.OperationID).Scan(&actor, &request, &fingerprint); err != nil {
		return err
	}
	var intent struct {
		Kind, Project string
		Request       projectRecoveryObservationRequest
	}
	digest, _ := p.Plan.Digest()
	if json.Unmarshal([]byte(fingerprint), &intent) != nil || actor == "" || actor != op.Actor || request != op.RequestID || op.Kind != "observe-project-recovery" || op.ProjectID != j.ProjectID || op.InstanceID != "" || op.ConfigRevision != p.Revision || intent.Kind != op.Kind || intent.Project != j.ProjectID || intent.Request.RequestID != op.RequestID || intent.Request.RecoveryOperationID != j.RecoveryOperationID || intent.Request.ExpectedRevision != p.Revision || intent.Request.ExpectedSequence != j.Before.Sequence || intent.Request.HistoryDigest != j.SourceDigest || intent.Request.CheckpointDigest != recoveryCheckpointDigest(j.Before) || intent.Request.ReviewDigest != digest {
		return fail()
	}
	switch j.Phase {
	case "planned":
		if j.After != nil || op.Status != "queued" && op.Status != "interrupted" {
			return fail()
		}
	case "observing":
		if j.After != nil || op.Status != "running" && op.Status != "interrupted" {
			return fail()
		}
	case "observed":
		if op.Status != "succeeded" || j.After == nil || projectnetwork.RecoveryJournalObservedTransition(parent.Source, j.Before, *j.After) != nil || !projectnetwork.RecoveryJournalContainsCheckpoint(parent.Source, parent.Journal, *j.After) {
			return fail()
		}
	default:
		return fail()
	}
	if op.Status == "queued" || op.Status == "running" {
		owner, err := loadOperation(q, j.RecoveryOperationID)
		if err != nil {
			return err
		}
		if owner.Status != "interrupted" || !reflect.DeepEqual(parent.Journal, j.Before) {
			return fail()
		}
	}
	return projectRecoveryMirror(q, j.OperationID, j)
}
func validateProjectRecoveryObservations(q querier) error {
	return validateSnapshot(q, "validateProjectRecoveryObservations", validateProjectRecoveryObservationsUncached)
}

func validateProjectRecoveryObservationsUncached(q querier) error {
	jobs, err := projectRecoveryObservationJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		if err = validateRecoveryObservation(q, j); err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" {
			if active[j.ProjectID] {
				return errors.New("overlapping recovery observations")
			}
			active[j.ProjectID] = true
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='observe-project-recovery' AND id NOT IN (SELECT operation_id FROM project_recovery_observations)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("recovery observation is missing its owning journal")
	}
	return nil
}
func saveProjectRecoveryObservation(tx *sql.Tx, j projectRecoveryObservationJob, op protocol.Operation) error {
	b, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_recovery_observations SET body=? WHERE operation_id=? AND project_id=? AND recovery_operation_id=?", b, j.OperationID, j.ProjectID, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("recovery observation lost ownership")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", b, op.ID); err != nil {
		return err
	}
	return saveOperation(tx, op)
}
func (s *store) acceptProjectRecoveryObservation(actor, project string, req projectRecoveryObservationRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.RecoveryOperationID) || req.ExpectedRevision == 0 || len(req.CheckpointDigest) != 64 || len(req.HistoryDigest) != 64 {
		return protocol.Operation{}, false, bad("recovery observation requires a reviewed parent and exact checkpoint")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project string
		Request       projectRecoveryObservationRequest
	}{"observe-project-recovery", project, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching recovery observation exists")
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
	if err = projectRecoveryReceiptAdmission(tx, project, req.RecoveryOperationID); err != nil {
		return protocol.Operation{}, false, err
	}
	parent, err := loadProjectAnchorRecovery(tx, req.RecoveryOperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op, err := loadOperation(tx, parent.OperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	p, err := loadProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	digest, _ := p.Plan.Digest()
	if parent.ProjectID != project || op.Status != "interrupted" || req.ExpectedRevision != p.Revision || req.ReviewDigest != digest || req.ExpectedSequence != parent.Journal.Sequence || req.CheckpointDigest != recoveryCheckpointDigest(parent.Journal) || req.HistoryDigest != recoveryHistoryDigest(parent.Source) {
		return protocol.Operation{}, false, conflict("revision_conflict", "recovery or its checkpoint changed; review the current retained intent")
	}
	if err = projectRetirementClear(tx, project, parent.OperationID); err != nil {
		return protocol.Operation{}, false, err
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish the active project observation before another attempt")
	}
	target, err := projectnetwork.RecoveryCheckpointObservationTarget(parent.Source, parent.Journal)
	if err != nil {
		return protocol.Operation{}, false, conflict("project_recovery_review_required", err.Error())
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	observed := protocol.Operation{ID: "op-" + id, Kind: "observe-project-recovery", ProjectID: project, Actor: actor, RequestID: req.RequestID, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, observed, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectRecoveryObservationJob{OperationID: observed.ID, ProjectID: project, RecoveryOperationID: parent.OperationID, SourceDigest: req.HistoryDigest, Target: target, Phase: "planned", Before: parent.Journal.Clone()}
	if _, err = tx.Exec("INSERT INTO project_recovery_observations(operation_id,project_id,recovery_operation_id,body) VALUES(?,?,?,?)", j.OperationID, project, parent.OperationID, []byte("{}")); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectRecoveryObservation(tx, j, observed); err != nil {
		return protocol.Operation{}, false, err
	}
	return observed, false, tx.Commit()
}
func (s *store) beginProjectRecoveryObservation(id string) (projectRecoveryObservationJob, error) {
	var empty projectRecoveryObservationJob
	tx, err := s.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	j, err := loadProjectRecoveryObservation(tx, id)
	if err != nil {
		return empty, err
	}
	if err = validateRecoveryObservation(tx, j); err != nil {
		return empty, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return empty, err
	}
	if op.Status != "queued" || j.Phase != "planned" {
		return empty, errors.New("recovery observation is not freshly queued")
	}
	if err = projectRetirementClear(tx, j.ProjectID, j.RecoveryOperationID); err != nil {
		return empty, err
	}
	j.Phase, op.Status = "observing", "running"
	if err = saveProjectRecoveryObservation(tx, j, op); err != nil {
		return empty, err
	}
	return j, tx.Commit()
}
func (s *store) completeProjectRecoveryObservation(id string, expected uint64, after projectnetwork.AnchorRecoveryJournal) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	j, err := loadProjectRecoveryObservation(tx, id)
	if err != nil {
		return err
	}
	if err = validateRecoveryObservation(tx, j); err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	parent, err := loadProjectAnchorRecovery(tx, j.RecoveryOperationID)
	if err != nil {
		return err
	}
	owner, err := loadOperation(tx, parent.OperationID)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Phase != "observing" || owner.Status != "interrupted" || expected != j.Before.Sequence || !reflect.DeepEqual(parent.Journal, j.Before) {
		return errors.New("stale recovery observation completion")
	}
	if err = projectRetirementClear(tx, j.ProjectID, parent.OperationID); err != nil {
		return err
	}
	if err = projectnetwork.RecoveryJournalObservedTransition(parent.Source, j.Before, after); err != nil {
		return err
	}
	parent.Journal = after.Clone()
	owner.Message = "one retained recovery outcome observed; explicit continuation remains required"
	now := time.Now().UTC()
	if after.Phase == "complete" {
		owner.Status = "succeeded"
		owner.FinishedAt = &now
		owner.Message = "coordinated recovery completed by fresh read-only readiness"
	}
	if err = saveProjectAnchorRecovery(tx, parent, owner); err != nil {
		return err
	}
	copy := after.Clone()
	j.After = &copy
	j.Phase = "observed"
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "retained recovery outcome observed without replaying runtime actions"
	if err = saveProjectRecoveryObservation(tx, j, op); err != nil {
		return err
	}
	if err = validateProjectRecoveryObservations(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectRecoveryObservation(tx *sql.Tx, id string) error {
	j, err := loadProjectRecoveryObservation(tx, id)
	if err != nil {
		return err
	}
	if err = validateRecoveryObservation(tx, j); err != nil {
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
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "read-only recovery observation interrupted; original checkpoint remains fenced"
	return saveProjectRecoveryObservation(tx, j, op)
}
func (s *store) interruptRecoveryObservation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectRecoveryObservation(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func reconcileProjectRecoveryObservations(tx *sql.Tx) error {
	if err := validateProjectRecoveryObservations(tx); err != nil {
		return err
	}
	jobs, err := projectRecoveryObservationJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectRecoveryObservation(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
