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

// Internal store request, reconstructed by the explicit public action route.
type projectPrepareRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

type projectJob struct {
	ProjectID   string                      `json:"project_id"`
	OperationID string                      `json:"operation_id"`
	Generation  uint64                      `json:"generation"`
	Preparation *projectnetwork.Preparation `json:"preparation,omitempty"`
}

func initializeProjectJobs(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_jobs (project_id TEXT PRIMARY KEY REFERENCES projects(id), operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), body BLOB NOT NULL);`)
	return err
}

func loadProject(q querier, id string) (storedProject, error) {
	var data []byte
	var p storedProject
	if err := q.QueryRow("SELECT body FROM projects WHERE id=?", id).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, conflict("project_not_found", "project is not registered")
		}
		return p, err
	}
	if json.Unmarshal(data, &p) != nil || p.ID != id {
		return p, errors.New("invalid project record")
	}
	return p, nil
}

func loadProjectJob(q querier, id string) (projectJob, error) {
	var data []byte
	var opID string
	var j projectJob
	if err := q.QueryRow("SELECT operation_id,body FROM project_jobs WHERE project_id=?", id).Scan(&opID, &data); err != nil {
		return j, err
	}
	if json.Unmarshal(data, &j) != nil || j.ProjectID != id || j.OperationID != opID || j.Generation != 1 {
		return j, errors.New("invalid project runtime journal")
	}
	return j, nil
}

func saveProjectJob(tx *sql.Tx, j projectJob) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE project_jobs SET body=? WHERE project_id=? AND operation_id=?", data, j.ProjectID, j.OperationID)
	return err
}

func validateProjectJobs(q querier) error {
	return validateSnapshot(q, "validateProjectJobs", validateProjectJobsUncached)
}

func validateProjectJobsUncached(q querier) error {
	rows, err := q.Query("SELECT project_id FROM project_jobs")
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
		j, err := loadProjectJob(q, id)
		if err != nil {
			return err
		}
		p, err := loadProject(q, id)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.ProjectID != id || op.InstanceID != "" || op.Kind != "prepare-project" || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("project runtime receipt differs from ownership")
		}
		if op.Status != "queued" && op.Status != "running" && op.Status != "interrupted" && op.Status != "succeeded" {
			return errors.New("invalid project runtime receipt status")
		}
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
			return err
		}
		if j.Preparation == nil {
			if op.Status != "queued" && op.Status != "interrupted" || len(mirror) != 0 {
				return errors.New("project runtime receipt lacks its journal")
			}
		} else {
			v := j.Preparation
			if v.ProjectID != id || v.ProjectRevision != p.Revision || v.Generation != j.Generation || v.OperationID != op.ID || v.Validate(p.Plan) != nil {
				return errors.New("invalid bound project preparation")
			}
			data, _ := json.Marshal(v)
			if string(mirror) != string(data) || op.Status == "queued" || op.Status == "succeeded" && v.Anchor.Phase != "observed" {
				return errors.New("project journal and receipt disagree")
			}
		}
	}
	// A missing journal cannot release a project for another preparation attempt.
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='prepare-project' AND id NOT IN (SELECT operation_id FROM project_jobs)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("project operation is missing its journal")
	}
	return nil
}

func (s *store) acceptProjectPreparation(actor, id string, req projectPrepareRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(id) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, actor, request and revision are required")
	}
	body, _ := json.Marshal(struct {
		Kind, ProjectID string
		Request         projectPrepareRequest
	}{"prepare-project", id, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(body)); found || err != nil {
		return op, found, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching project preparation receipt exists")
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
	p, err := loadProject(tx, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("project preparation must match the reviewed plan")
	}
	if _, err = loadProjectJob(tx, id); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return protocol.Operation{}, false, err
		}
		return protocol.Operation{}, false, conflict("project_recovery_required", "observe the retained project operation before further preparation")
	}
	random, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + random, ProjectID: id, RequestID: req.RequestID, Actor: actor, Kind: "prepare-project", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(body)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectJob{ProjectID: id, OperationID: op.ID, Generation: 1}
	data, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_jobs(project_id,operation_id,body) VALUES(?,?,?)", id, op.ID, data); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

// Engine/config input is produced internally by the future driver, never by RPC.
// Binding both planned resources is a transaction before any dispatch can occur.
func (s *store) beginProjectPreparation(id, opID string, prepared projectnetwork.Preparation) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectJobs(tx); err != nil {
		return err
	}
	p, err := loadProject(tx, id)
	if err != nil {
		return err
	}
	j, err := loadProjectJob(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.OperationID != opID || op.Status != "queued" || j.Preparation != nil || prepared.ProjectID != id || prepared.ProjectRevision != p.Revision || prepared.Generation != j.Generation || prepared.OperationID != opID || prepared.Sequence != 0 || prepared.Validate(p.Plan) != nil {
		return conflict("stale_project_operation", "operation no longer owns project preparation")
	}
	j.Preparation = &prepared
	op.Status = "running"
	if err = saveProjectJob(tx, j); err != nil {
		return err
	}
	data, _ := json.Marshal(prepared)
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, opID); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *store) advanceProjectPreparation(id, opID string, sequence uint64, kind, phase, resourceID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectJobs(tx); err != nil {
		return err
	}
	p, err := loadProject(tx, id)
	if err != nil {
		return err
	}
	j, err := loadProjectJob(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.OperationID != opID || op.Status != "running" || j.Preparation == nil {
		return conflict("stale_project_operation", "operation no longer owns project preparation")
	}
	next, err := j.Preparation.Advance(p.Plan, sequence, kind, phase, resourceID)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(next, *j.Preparation) {
		return errors.New("project preparation did not advance")
	}
	j.Preparation = &next
	if err = saveProjectJob(tx, j); err != nil {
		return err
	}
	data, _ := json.Marshal(next)
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", data, opID); err != nil {
		return err
	}
	// Preparing the anchor is not whole-project readiness; members do not exist yet.
	if next.Anchor.Phase == "observed" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "project resources prepared; no member was started"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func interruptProjectOperation(tx *sql.Tx, op protocol.Operation) error {
	j, err := loadProjectJob(tx, op.ProjectID)
	if err != nil {
		return err
	}
	if j.OperationID != op.ID || op.Kind != "prepare-project" || op.InstanceID != "" {
		return errors.New("project interruption ownership mismatch")
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "project preparation interrupted; retain resources and inspect its original journal before recovery"
	return saveOperation(tx, op)
}

func reconcileProjectJobs(tx *sql.Tx) error {
	if err := validateProjectJobs(tx); err != nil {
		return err
	}
	rows, err := tx.Query("SELECT operation_id FROM project_jobs")
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
		op, err := loadOperation(tx, id)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" {
			if err = interruptProjectOperation(tx, op); err != nil {
				return err
			}
		}
	}
	return nil
}
