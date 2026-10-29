package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Private initial Start only. Preparation remains immutable; neither reopening
// nor replay schedules work. Recovery/replacement and member lifecycle are separate.
type projectAnchorStartRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

func initializeProjectAnchorStarts(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_anchor_starts (project_id TEXT PRIMARY KEY REFERENCES projects(id), operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), body BLOB NOT NULL);`)
	return err
}

func loadProjectAnchorStart(q querier, id string) (projectnetwork.AnchorStart, error) {
	var state projectnetwork.AnchorStart
	var data []byte
	var opID string
	if err := q.QueryRow("SELECT operation_id,body FROM project_anchor_starts WHERE project_id=?", id).Scan(&opID, &data); err != nil {
		return state, err
	}
	if json.Unmarshal(data, &state) != nil || state.Prepared.ProjectID != id || state.OperationID != opID {
		return state, errors.New("invalid project anchor Start journal")
	}
	return state, nil
}

func validateProjectAnchorStarts(q querier) error {
	return validateSnapshot(q, "validateProjectAnchorStarts", validateProjectAnchorStartsUncached)
}

func validateProjectAnchorStartsUncached(q querier) error {
	rows, err := q.Query("SELECT project_id FROM project_anchor_starts")
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	for _, id := range ids {
		state, err := loadProjectAnchorStart(q, id)
		if err != nil {
			return err
		}
		p, err := loadProject(q, id)
		if err != nil {
			return err
		}
		prepared, err := loadProjectJob(q, id)
		if err != nil {
			return err
		}
		preparationOp, err := loadOperation(q, prepared.OperationID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, state.OperationID)
		if err != nil {
			return err
		}
		if state.Validate(p.Plan) != nil || prepared.Preparation == nil || !reflect.DeepEqual(state.Prepared, *prepared.Preparation) || preparationOp.Status != "succeeded" || op.ProjectID != id || op.InstanceID != "" || op.Kind != "start-project-anchor" || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("project anchor Start differs from retained preparation or ownership")
		}
		if op.Status != "queued" && op.Status != "running" && op.Status != "interrupted" && op.Status != "succeeded" {
			return errors.New("invalid project anchor Start status")
		}
		if op.Status == "queued" && state.Phase != "planned" || (op.Status == "succeeded") != (state.Phase == "observed") {
			return errors.New("project anchor Start phase and receipt disagree")
		}
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
			return err
		}
		data, _ := json.Marshal(state)
		if string(data) != string(mirror) {
			return errors.New("project anchor Start journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='start-project-anchor' AND id NOT IN (SELECT operation_id FROM project_anchor_starts)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("project anchor Start receipt is missing its journal")
	}
	return nil
}

func (s *store) acceptProjectAnchorStart(actor, id string, req projectAnchorStartRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(id) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, actor, request and revision are required")
	}
	body, _ := json.Marshal(struct {
		Kind, ProjectID string
		Request         projectAnchorStartRequest
	}{"start-project-anchor", id, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(body)); found || err != nil {
		return op, found, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching anchor Start receipt exists")
	}
	if err = projectExpansionClear(tx, id); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, id, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = projectReservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectJobs(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectAnchorStarts(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	p, err := loadProject(tx, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || req.ReviewDigest != digest {
		return protocol.Operation{}, false, bad("anchor Start must match the reviewed project plan")
	}
	if _, err = loadProjectAnchorStart(tx, id); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return protocol.Operation{}, false, err
		}
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original anchor Start receipt; observe before recovery")
	}
	prepared, err := loadProjectJob(tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, conflict("project_not_prepared", "complete project preparation before starting its anchor")
	}
	if err != nil {
		return protocol.Operation{}, false, err
	}
	preparationOp, err := loadOperation(tx, prepared.OperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if preparationOp.Status != "succeeded" || prepared.Preparation == nil {
		return protocol.Operation{}, false, conflict("project_not_prepared", "complete project preparation before starting its anchor")
	}
	random, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + random, ProjectID: id, RequestID: req.RequestID, Actor: actor, Kind: "start-project-anchor", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	state, err := projectnetwork.NewAnchorStart(p.Plan, *prepared.Preparation, op.ID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if err = insertOperation(tx, op, string(body)); err != nil {
		return protocol.Operation{}, false, err
	}
	data, _ := json.Marshal(state)
	if _, err = tx.Exec("INSERT INTO project_anchor_starts(project_id,operation_id,body) VALUES(?,?,?)", id, op.ID, data); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, op.ID); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

func (s *store) beginProjectAnchorStart(id, opID string) (projectnetwork.AnchorStart, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectnetwork.AnchorStart{}, err
	}
	defer tx.Rollback()
	if err = validateProjectJobs(tx); err != nil {
		return projectnetwork.AnchorStart{}, err
	}
	if err = validateProjectAnchorStarts(tx); err != nil {
		return projectnetwork.AnchorStart{}, err
	}
	state, err := loadProjectAnchorStart(tx, id)
	if err != nil {
		return state, err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return state, err
	}
	if state.OperationID != opID || op.Status != "queued" || state.Phase != "planned" {
		return state, conflict("stale_project_operation", "operation no longer owns initial anchor Start")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return state, err
	}
	return state, tx.Commit()
}

func (s *store) advanceProjectAnchorStart(id, opID, expected, phase string, namespace *projectnetwork.NamespaceIdentity) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectJobs(tx); err != nil {
		return err
	}
	if err = validateProjectAnchorStarts(tx); err != nil {
		return err
	}
	state, err := loadProjectAnchorStart(tx, id)
	if err != nil {
		return err
	}
	p, err := loadProject(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if state.OperationID != opID || op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns initial anchor Start")
	}
	next, err := state.Advance(p.Plan, expected, phase, namespace)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(next)
	if _, err = tx.Exec("UPDATE project_anchor_starts SET body=? WHERE project_id=? AND operation_id=?", data, id, opID); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, opID); err != nil {
		return err
	}
	if next.Phase == "observed" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "project anchor started and observed; no member was started"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func interruptProjectAnchorStart(tx *sql.Tx, id, opID string) error {
	state, err := loadProjectAnchorStart(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if state.OperationID != opID || op.ProjectID != id || op.Kind != "start-project-anchor" || op.InstanceID != "" {
		return errors.New("anchor Start interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "anchor Start did not finish; retain the original receipt and inspect before recovery"
	return saveOperation(tx, op)
}

func reconcileProjectAnchorStarts(tx *sql.Tx) error {
	if err := validateProjectAnchorStarts(tx); err != nil {
		return err
	}
	rows, err := tx.Query("SELECT project_id,operation_id FROM project_anchor_starts")
	if err != nil {
		return err
	}
	var entries [][2]string
	for rows.Next() {
		var v [2]string
		if err = rows.Scan(&v[0], &v[1]); err != nil {
			break
		}
		entries = append(entries, v)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	for _, v := range entries {
		if err = interruptProjectAnchorStart(tx, v[0], v[1]); err != nil {
			return err
		}
	}
	return nil
}

// The caller must hold the runner-wide resource/update fence. This remains a
// private worker with no RPC, automatic scheduling or recovery retry.
func (s *store) runProjectAnchorStart(ctx context.Context, id, opID string, engine projectnetwork.AnchorStartEngine) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if engine == nil {
		return errors.New("anchor Start engine is required")
	}
	state, err := s.beginProjectAnchorStart(id, opID)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, id)
	if err == nil {
		_, err = projectnetwork.ExecuteAnchorStart(ctx, p.Plan, state, engine, func(expected, phase string, namespace *projectnetwork.NamespaceIdentity) error {
			return s.advanceProjectAnchorStart(id, opID, expected, phase, namespace)
		}, time.Now)
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptAnchorStart(id, opID))
}

// Setup failure/cancellation before begin is also retained by the server worker.
func (s *store) interruptAnchorStart(id, opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectAnchorStart(tx, id, opID); err != nil {
		return err
	}
	return tx.Commit()
}
