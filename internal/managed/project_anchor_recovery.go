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

type projectAnchorRecoveryRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	HistoryDigest    string `json:"history_digest"`
	ReplayOnly       bool   `json:"-"`
}
type projectAnchorRecoveryJob struct {
	OperationID       string                               `json:"operation_id"`
	ProjectID         string                               `json:"project_id"`
	OriginalReadiness []string                             `json:"original_readiness"`
	CurrentReadiness  []string                             `json:"current_readiness"`
	Source            projectnetwork.AnchorRecoveryHistory `json:"source"`
	Journal           projectnetwork.AnchorRecoveryJournal `json:"journal"`
}
type projectRecoveryMemberReceipt struct {
	ParentOperationID string                          `json:"parent_operation_id"`
	InstanceID        string                          `json:"instance_id"`
	OperationID       string                          `json:"operation_id"`
	Member            *projectnetwork.RecoveredMember `json:"member,omitempty"`
}

func initializeProjectAnchorRecoveries(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_anchor_recoveries(operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES projects(id),body BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS project_recovery_members(operation_id TEXT PRIMARY KEY REFERENCES operations(id),parent_operation_id TEXT NOT NULL REFERENCES project_anchor_recoveries(operation_id),instance_id TEXT NOT NULL REFERENCES project_members(instance_id),UNIQUE(parent_operation_id,instance_id));`)
	return err
}
func loadProjectAnchorRecovery(q querier, id string) (projectAnchorRecoveryJob, error) {
	var j projectAnchorRecoveryJob
	var project string
	var body []byte
	err := q.QueryRow("SELECT project_id,body FROM project_anchor_recoveries WHERE operation_id=?", id).Scan(&project, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.Journal.IDs.Replacement != id {
		return j, errors.New("invalid coordinated recovery journal ownership")
	}
	return j, nil
}
func projectAnchorRecoveryJobs(q querier) ([]projectAnchorRecoveryJob, error) {
	var exists int
	if err := q.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='project_anchor_recoveries'").Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	}
	rows, err := q.Query("SELECT operation_id FROM project_anchor_recoveries ORDER BY operation_id")
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
	var out []projectAnchorRecoveryJob
	for _, id := range ids {
		j, err := loadProjectAnchorRecovery(q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}
func recoveryMemberReceipt(j projectAnchorRecoveryJob, index int) projectRecoveryMemberReceipt {
	id := j.Journal.IDs.Members[index]
	r := projectRecoveryMemberReceipt{ParentOperationID: j.OperationID, InstanceID: id.InstanceID, OperationID: id.OperationID}
	replacement := j.Journal.Replacement
	if j.Journal.Readiness != nil {
		replacement = &j.Journal.Readiness.Replacement
	}
	if replacement != nil {
		m := replacement.Clone().Members[index]
		r.Member = &m
	}
	return r
}
func validateRecoveryReceipt(q querier, j projectAnchorRecoveryJob) error {
	if len(j.Source.Launches) == 0 || j.Journal.Validate(j.Source) != nil {
		return errors.New("invalid retained recovery source or phase")
	}
	p, err := loadProject(q, j.ProjectID)
	if err != nil {
		return err
	}
	op, err := loadOperation(q, j.OperationID)
	if err != nil {
		return err
	}
	if op.Kind != "recover-project-anchor" || op.ProjectID != j.ProjectID || op.InstanceID != "" || op.Actor == "" || op.ConfigRevision != p.Revision || !reflect.DeepEqual(p.Plan, j.Source.Plan) {
		return errors.New("recovery parent ownership changed")
	}
	var actor, request, fingerprint string
	if err = q.QueryRow("SELECT actor,request_id,fingerprint FROM operations WHERE id=?", op.ID).Scan(&actor, &request, &fingerprint); err != nil {
		return err
	}
	var intent struct {
		Kind, Project string
		Request       projectAnchorRecoveryRequest
	}
	digest, _ := p.Plan.Digest()
	if json.Unmarshal([]byte(fingerprint), &intent) != nil || actor != op.Actor || request != op.RequestID || intent.Kind != op.Kind || intent.Project != op.ProjectID || intent.Request.RequestID != op.RequestID || intent.Request.ExpectedRevision != op.ConfigRevision || intent.Request.ReviewDigest != digest || intent.Request.HistoryDigest != recoveryHistoryDigest(j.Source) {
		return errors.New("recovery source differs from accepted request")
	}
	switch op.Status {
	case "queued":
		if j.Journal.Phase != "review-planned" {
			return errors.New("queued recovery has already started")
		}
	case "running":
		if j.Journal.Phase == "review-planned" || j.Journal.Phase == "complete" {
			return errors.New("invalid running recovery")
		}
	case "interrupted":
		if j.Journal.Phase == "complete" {
			return errors.New("interrupted recovery has completed")
		}
	case "failed":
		if j.Journal.Phase != "review-planned" && j.Journal.Phase != "reviewing" {
			return errors.New("recovery effects cannot be retired as a failed review")
		}
	case "succeeded":
		if j.Journal.Phase != "complete" {
			return errors.New("successful recovery lacks complete readiness")
		}
	default:
		return errors.New("unsupported recovery status")
	}
	if err = projectRecoveryMirror(q, j.OperationID, j); err != nil {
		return err
	}
	var count int
	if err = q.QueryRow("SELECT count(*) FROM project_recovery_members WHERE parent_operation_id=?", j.OperationID).Scan(&count); err != nil {
		return err
	}
	if count != len(j.Journal.IDs.Members) {
		return errors.New("recovery child creator inventory changed")
	}
	for i, id := range j.Journal.IDs.Members {
		child, err := loadOperation(q, id.OperationID)
		if err != nil {
			return err
		}
		var parent, instance string
		if err = q.QueryRow("SELECT parent_operation_id,instance_id FROM project_recovery_members WHERE operation_id=?", id.OperationID).Scan(&parent, &instance); err != nil {
			return err
		}
		if parent != j.OperationID || instance != id.InstanceID || child.Kind != "recover-project-member" || child.ProjectID != j.ProjectID || child.InstanceID != id.InstanceID || child.Actor != op.Actor || child.ConfigRevision != op.ConfigRevision || child.Status != op.Status {
			return errors.New("recovery creator lost its parent ownership")
		}
		var childActor, childRequest, childFingerprint string
		if err = q.QueryRow("SELECT actor,request_id,fingerprint FROM operations WHERE id=?", child.ID).Scan(&childActor, &childRequest, &childFingerprint); err != nil {
			return err
		}
		expectedFingerprint, _ := json.Marshal(struct{ Parent, Instance string }{j.OperationID, id.InstanceID})
		if childActor != child.Actor || childRequest != child.RequestID || child.RequestID != child.ID || childFingerprint != string(expectedFingerprint) {
			return errors.New("recovery creator intent changed")
		}
		if err = projectRecoveryMirror(q, child.ID, recoveryMemberReceipt(j, i)); err != nil {
			return err
		}
	}
	return nil
}
func validateProjectAnchorRecoveries(q querier) error {
	return validateSnapshot(q, "validateProjectAnchorRecoveries", validateProjectAnchorRecoveriesUncached)
}

func validateProjectAnchorRecoveriesUncached(q querier) error {
	jobs, err := projectAnchorRecoveryJobs(q)
	if err != nil {
		return err
	}
	projects := map[string]bool{}
	pending := map[string]bool{}
	for _, j := range jobs {
		if err = validateRecoveryReceipt(q, j); err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Status != "succeeded" && op.Status != "failed" {
			if pending[j.ProjectID] {
				return errors.New("overlapping unresolved coordinated recovery")
			}
			pending[j.ProjectID] = true
		}
		projects[j.ProjectID] = true
	}
	for p := range projects {
		if _, _, _, err = projectRecoveryHistory(q, p, false); err != nil {
			return err
		}
	}
	var orphans int
	if err = q.QueryRow(`SELECT count(*) FROM operations WHERE
 (json_extract(body,'$.kind')='recover-project-anchor' AND id NOT IN (SELECT operation_id FROM project_anchor_recoveries)) OR
 (json_extract(body,'$.kind')='recover-project-member' AND id NOT IN (SELECT operation_id FROM project_recovery_members))`).Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("recovery operation is missing its owning journal")
	}
	return nil
}
func projectRecoveryClear(q querier, project, except string) error {
	jobs, err := projectAnchorRecoveryJobs(q)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.ProjectID != project || j.OperationID == except {
			continue
		}
		if err = validateRecoveryReceipt(q, j); err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Status != "succeeded" && op.Status != "failed" {
			return conflict("project_recovery_required", "coordinated recovery retains unresolved intent; inspect its journal before any other project action")
		}
	}
	return nil
}
func recoveryHistoryDigest(h projectnetwork.AnchorRecoveryHistory) string {
	b, _ := json.Marshal(h)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func saveProjectAnchorRecovery(tx *sql.Tx, j projectAnchorRecoveryJob, op protocol.Operation) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_anchor_recoveries SET body=? WHERE operation_id=? AND project_id=?", body, j.OperationID, j.ProjectID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("recovery lost its durable journal")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	for i, id := range j.Journal.IDs.Members {
		child, err := loadOperation(tx, id.OperationID)
		if err != nil {
			return err
		}
		child.Status, child.FinishedAt, child.Message = op.Status, op.FinishedAt, op.Message
		if err = saveOperation(tx, child); err != nil {
			return err
		}
		b, err := json.Marshal(recoveryMemberReceipt(j, i))
		if err != nil {
			return err
		}
		if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", b, child.ID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) acceptProjectAnchorRecovery(actor, project string, req projectAnchorRecoveryRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 || len(req.HistoryDigest) != 64 {
		return protocol.Operation{}, false, bad("recovery requires actor, project, request, revision and reviewed history")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project string
		Request       projectAnchorRecoveryRequest
	}{"recover-project-anchor", project, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching recovery receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = validateProjectMemberContextReadiness(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectAnchorRecoveries(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	h, originals, current, err := projectRecoveryHistory(tx, project, true)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, conflict("project_not_launched", "complete initial member launches and readiness before recovery")
	}
	if err != nil {
		return protocol.Operation{}, false, err
	}
	p, err := loadProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if req.ExpectedRevision != p.Revision || req.ReviewDigest != digest || req.HistoryDigest != recoveryHistoryDigest(h) {
		return protocol.Operation{}, false, conflict("revision_conflict", "project or recovery history changed; review the current selection")
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish active project work before coordinated recovery")
	}
	// An unresolved Stop on a current member cannot be absorbed into recovery.
	context, err := projectnetwork.SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, m := range context.Members {
		if err = currentContextObservationAllowed(tx, m.Context); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	fresh := func() (string, error) { id, err := randomID(); return "op-" + id, err }
	ids := projectnetwork.RecoveryJournalIDs{}
	for _, dst := range []*string{&ids.Drain, &ids.Namespace, &ids.Retirement, &ids.Replacement} {
		*dst, err = fresh()
		if err != nil {
			return protocol.Operation{}, false, err
		}
	}
	for _, m := range context.Members {
		id, err := fresh()
		if err != nil {
			return protocol.Operation{}, false, err
		}
		ids.Members = append(ids.Members, projectnetwork.RecoveryJournalMemberID{InstanceID: m.Context.Member.InstanceID, OperationID: id})
	}
	journal, err := projectnetwork.NewAnchorRecoveryJournal(h, ids)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: ids.Replacement, ProjectID: project, Actor: actor, RequestID: req.RequestID, Kind: "recover-project-anchor", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectAnchorRecoveryJob{OperationID: op.ID, ProjectID: project, OriginalReadiness: originals, CurrentReadiness: current, Source: h, Journal: journal}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_anchor_recoveries(operation_id,project_id,body) VALUES(?,?,?)", op.ID, project, body); err != nil {
		return protocol.Operation{}, false, err
	}
	for _, id := range ids.Members {
		child := protocol.Operation{ID: id.OperationID, ProjectID: project, InstanceID: id.InstanceID, Actor: actor, RequestID: id.OperationID, Kind: "recover-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: op.CreatedAt}
		b, _ := json.Marshal(struct{ Parent, Instance string }{op.ID, id.InstanceID})
		if err = insertOperation(tx, child, string(b)); err != nil {
			return protocol.Operation{}, false, err
		}
		if _, err = tx.Exec("INSERT INTO project_recovery_members(operation_id,parent_operation_id,instance_id) VALUES(?,?,?)", child.ID, op.ID, id.InstanceID); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	if err = saveProjectAnchorRecovery(tx, j, op); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, tx.Commit()
}
func (s *store) advanceProjectAnchorRecovery(id string, expected uint64, phase string, u projectnetwork.AnchorRecoveryJournal) (projectAnchorRecoveryJob, error) {
	var empty projectAnchorRecoveryJob
	tx, err := s.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	if err = projectRecoveryContinuationInactive(tx, id); err != nil {
		return empty, err
	}
	j, err := loadProjectAnchorRecovery(tx, id)
	if err != nil {
		return empty, err
	}
	if err = validateRecoveryReceipt(tx, j); err != nil {
		return empty, err
	}
	if err = recoverySourceReferences(tx, j); err != nil {
		return empty, err
	}
	if err = projectRetirementClear(tx, j.ProjectID, id); err != nil {
		return empty, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return empty, err
	}
	if op.Status != "running" && !(op.Status == "queued" && phase == "reviewing") {
		return empty, conflict("project_recovery_required", "recovery is not active; reopening does not authorize replay")
	}
	h, _, _, err := projectRecoveryHistory(tx, j.ProjectID, false)
	if err != nil {
		return empty, err
	}
	h.Current = j.Source.Current
	if !reflect.DeepEqual(h, j.Source) {
		return empty, errors.New("recovery source changed before commit")
	}
	next, err := j.Journal.Advance(j.Source, expected, phase, u)
	if err != nil {
		return empty, err
	}
	now := time.Now().UTC()
	op.Status = "running"
	if next.Phase == "complete" {
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "coordinated recovery completed with exact member identity and readiness"
	}
	j.Journal = next
	if err = saveProjectAnchorRecovery(tx, j, op); err != nil {
		return empty, err
	}
	return j, tx.Commit()
}
func interruptProjectAnchorRecovery(tx *sql.Tx, id string, reviewFailed bool) error {
	if err := projectRecoveryContinuationInactive(tx, id); err != nil {
		return err
	}
	j, err := loadProjectAnchorRecovery(tx, id)
	if err != nil {
		return err
	}
	if err = validateRecoveryReceipt(tx, j); err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status == "succeeded" || op.Status == "failed" {
		return nil
	}
	if reviewFailed && j.Journal.Phase != "review-planned" && j.Journal.Phase != "reviewing" {
		return errors.New("cannot release recovery after retained review or effects")
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "recovery interrupted; preserve its intent and inspect before continuing; no automatic effects were replayed"
	if reviewFailed {
		op.Status = "failed"
		op.Message = "read-only recovery review did not establish a supported fault; no recovery effects were admitted"
	}
	return saveProjectAnchorRecovery(tx, j, op)
}
func (s *store) interruptAnchorRecovery(id string, reviewFailed bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectAnchorRecovery(tx, id, reviewFailed); err != nil {
		return err
	}
	return tx.Commit()
}
func reconcileProjectAnchorRecoveries(tx *sql.Tx) error {
	if err := validateProjectAnchorRecoveries(tx); err != nil {
		return err
	}
	jobs, err := projectAnchorRecoveryJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		op, err := loadOperation(tx, j.OperationID)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" {
			if err = interruptProjectAnchorRecovery(tx, j.OperationID, false); err != nil {
				return err
			}
		}
	}
	return nil
}
