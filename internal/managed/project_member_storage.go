package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type projectMemberStorageRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

func initializeProjectMemberStorage(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_storage (instance_id TEXT PRIMARY KEY REFERENCES project_members(instance_id), project_id TEXT NOT NULL REFERENCES projects(id), operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberStorage(q querier, id string) (projectnetwork.MemberStorage, error) {
	var state projectnetwork.MemberStorage
	var project, op string
	var body []byte
	err := q.QueryRow("SELECT project_id,operation_id,body FROM project_member_storage WHERE instance_id=?", id).Scan(&project, &op, &body)
	if err != nil {
		return state, err
	}
	if json.Unmarshal(body, &state) != nil || state.InstanceID != id || state.ProjectID != project || state.OperationID != op {
		return state, errors.New("invalid project member storage journal")
	}
	return state, nil
}
func projectMemberStorageClaims(q querier) ([]projectnetwork.MemberStorage, error) {
	rows, err := q.Query("SELECT instance_id FROM project_member_storage ORDER BY instance_id")
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
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	claims := make([]projectnetwork.MemberStorage, 0, len(ids))
	for _, id := range ids {
		s, err := loadProjectMemberStorage(q, id)
		if err != nil {
			return nil, err
		}
		p, err := loadProject(q, s.ProjectID)
		if err != nil {
			return nil, err
		}
		if s.Validate(p.Plan) != nil || s.ProjectRevision != p.Revision {
			return nil, errors.New("member storage differs from project plan")
		}
		matched := false
		for _, m := range p.Members {
			if m.InstanceID == id {
				matched = m.RuntimeID == s.RuntimeID
			}
		}
		if !matched {
			return nil, errors.New("member storage lost its reserved runtime identity")
		}
		claims = append(claims, s)
	}
	return claims, nil
}
func validateProjectMemberStorage(q querier) error {
	return validateSnapshot(q, "validateProjectMemberStorage", validateProjectMemberStorageUncached)
}

func validateProjectMemberStorageUncached(q querier) error {
	claims, err := projectMemberStorageClaims(q)
	if err != nil {
		return err
	}
	for _, s := range claims {
		op, err := loadOperation(q, s.OperationID)
		if err != nil {
			return err
		}
		if op.Kind != "prepare-project-member-storage" || op.ProjectID != s.ProjectID || op.InstanceID != s.InstanceID || op.ConfigRevision != s.ProjectRevision || op.Actor == "" {
			return errors.New("member storage receipt ownership mismatch")
		}
		if op.Status != "queued" && op.Status != "running" && op.Status != "interrupted" && op.Status != "succeeded" {
			return errors.New("unsupported member storage receipt status")
		}
		if op.Status == "queued" && s.Phase != "planned" || (op.Status == "succeeded") != (s.Phase == "observed") {
			return errors.New("member storage phase and receipt disagree")
		}
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
			return err
		}
		body, _ := json.Marshal(s)
		if string(body) != string(mirror) {
			return errors.New("member storage journal and receipt disagree")
		}
		// Materializing project members into standalone instances is intentionally
		// not supported by this storage-only slice. No legacy path can own it.
		if err = reserveStandalonePierPath(q, "", s.Location.Path); err != nil {
			return err
		}
	}
	var n int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='prepare-project-member-storage' AND id NOT IN (SELECT operation_id FROM project_member_storage)").Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return errors.New("member storage receipt is missing its journal")
	}
	return nil
}
func (s *store) acceptProjectMemberStorage(actor, project, instance string, req projectMemberStorageRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberStorageRequest
	}{"prepare-project-member-storage", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(fingerprint)); found || err != nil {
		return op, found, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching member storage receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = projectReservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectMemberStorage(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	p, err := loadProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("member storage must match the reviewed project plan")
	}
	if _, err = loadProjectMemberStorage(tx, instance); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return protocol.Operation{}, false, err
		}
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original member storage receipt before recovery")
	}
	runtime := ""
	for _, m := range p.Members {
		if m.InstanceID == instance {
			runtime = m.RuntimeID
		}
	}
	if runtime == "" {
		return protocol.Operation{}, false, bad("instance is not a reserved member of this project")
	}
	random, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + random, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "prepare-project-member-storage", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	state, err := projectnetwork.NewMemberStorage(p.Plan, project, p.Revision, instance, runtime, op.ID)
	if err != nil {
		return protocol.Operation{}, false, bad(err.Error())
	}
	if err = reservePierPath(tx, instance, state.Location.Path); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = s.checkStorage(state.Location.Path); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	body, _ := json.Marshal(state)
	if _, err = tx.Exec("INSERT INTO project_member_storage(instance_id,project_id,operation_id,body) VALUES(?,?,?,?)", instance, project, op.ID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, op.ID); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberStorage(instance, opID string) (projectnetwork.MemberStorage, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectnetwork.MemberStorage{}, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return projectnetwork.MemberStorage{}, err
	}
	if err = validateProjectMemberStorage(tx); err != nil {
		return projectnetwork.MemberStorage{}, err
	}
	state, err := loadProjectMemberStorage(tx, instance)
	if err != nil {
		return state, err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return state, err
	}
	if state.OperationID != opID || op.Status != "queued" || state.Phase != "planned" {
		return state, conflict("stale_project_operation", "operation no longer owns member storage preparation")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return state, err
	}
	return state, tx.Commit()
}
func (s *store) advanceProjectMemberStorage(instance, opID, expected, phase string, identity pierstore.DirectoryIdentity) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	if err = validateProjectMemberStorage(tx); err != nil {
		return err
	}
	state, err := loadProjectMemberStorage(tx, instance)
	if err != nil {
		return err
	}
	p, err := loadProject(tx, state.ProjectID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if state.OperationID != opID || op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns member storage preparation")
	}
	next, err := state.Advance(p.Plan, expected, phase, identity)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(next)
	if _, err = tx.Exec("UPDATE project_member_storage SET body=? WHERE instance_id=? AND operation_id=?", body, instance, opID); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, opID); err != nil {
		return err
	}
	if next.Phase == "observed" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "member mount prepared; no pier or ship was created"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func interruptProjectMemberStorage(tx *sql.Tx, instance, opID string) error {
	state, err := loadProjectMemberStorage(tx, instance)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if state.OperationID != opID || op.InstanceID != instance || op.ProjectID != state.ProjectID || op.Kind != "prepare-project-member-storage" {
		return errors.New("member storage interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "member storage preparation did not finish; retain original receipt and filesystem objects before recovery"
	return saveOperation(tx, op)
}
func reconcileProjectMemberStorage(tx *sql.Tx) error {
	if err := validateProjectMemberStorage(tx); err != nil {
		return err
	}
	claims, err := projectMemberStorageClaims(tx)
	if err != nil {
		return err
	}
	for _, v := range claims {
		if err = interruptProjectMemberStorage(tx, v.InstanceID, v.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) runProjectMemberStorage(ctx context.Context, instance, opID string, driver projectnetwork.MemberStorageDriver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if driver == nil {
		return errors.New("member storage driver is required")
	}
	state, err := s.beginProjectMemberStorage(instance, opID)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, state.ProjectID)
	if err == nil {
		_, err = projectnetwork.ExecuteMemberStorage(ctx, p.Plan, state, driver, func(expected, phase string, id pierstore.DirectoryIdentity) error {
			return s.advanceProjectMemberStorage(instance, opID, expected, phase, id)
		})
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptMemberStorage(instance, opID))
}

// Setup failure or cancellation before the worker claims its operation still
// retains the accepted storage claim and original planned receipt.
func (s *store) interruptMemberStorage(instance, opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberStorage(tx, instance, opID); err != nil {
		return err
	}
	return tx.Commit()
}
