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

type projectMemberStopExitRequest struct {
	RequestID        string `json:"request_id"`
	StopOperationID  string `json:"stop_operation_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

// Every read-only exit observation is separate from the original Stop. Restart
// interrupts this observation; it never acknowledges, retries or replaces Stop.
type projectMemberStopExitJob struct {
	OperationID     string                                 `json:"operation_id"`
	ProjectID       string                                 `json:"project_id"`
	InstanceID      string                                 `json:"instance_id"`
	StopOperationID string                                 `json:"stop_operation_id"`
	Phase           string                                 `json:"phase"`
	Stop            projectnetwork.MemberStop              `json:"stop"`
	Evidence        *projectnetwork.MemberStopExitEvidence `json:"evidence,omitempty"`
}

func initializeProjectMemberStopExit(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_stop_exits (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), stop_operation_id TEXT NOT NULL REFERENCES project_member_stops(operation_id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberStopExit(q querier, id string) (projectMemberStopExitJob, error) {
	var j projectMemberStopExitJob
	var project, instance, stop string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,stop_operation_id,body FROM project_member_stop_exits WHERE operation_id=?", id).Scan(&project, &instance, &stop, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.InstanceID != instance || j.StopOperationID != stop {
		return j, errors.New("invalid member Stop exit journal")
	}
	return j, nil
}
func projectMemberStopExitJobs(q querier) ([]projectMemberStopExitJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_stop_exits ORDER BY operation_id")
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
	var jobs []projectMemberStopExitJob
	for _, id := range ids {
		j, err := loadProjectMemberStopExit(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func projectStopExitPrerequisites(q querier, project, instance, stopID string) (storedProject, projectnetwork.MemberStop, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, projectnetwork.MemberStop{}, err
	}
	j, err := loadProjectMemberStop(q, stopID)
	if err != nil {
		return p, projectnetwork.MemberStop{}, err
	}
	op, err := loadOperation(q, stopID)
	if err != nil {
		return p, projectnetwork.MemberStop{}, err
	}
	if j.ProjectID != project || j.InstanceID != instance || j.Stop.Before == nil || j.Stop.Validate(p.Plan) != nil || op.Status != "interrupted" && op.Status != "succeeded" && op.Status != "failed" {
		return p, projectnetwork.MemberStop{}, conflict("project_stop_not_observable", "observe only a finished or interrupted Stop with retained process evidence")
	}
	return p, j.Stop, nil
}
func validateProjectMemberStopExit(q querier) error {
	return validateSnapshot(q, "validateProjectMemberStopExit", validateProjectMemberStopExitUncached)
}

func validateProjectMemberStopExitUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberStops(q); err != nil {
		return err
	}
	jobs, err := projectMemberStopExitJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		p, retained, err := projectStopExitPrerequisites(q, j.ProjectID, j.InstanceID, j.StopOperationID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Kind != "observe-project-member-stop-exit" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" || j.StopOperationID != retained.OperationID || !reflect.DeepEqual(j.Stop, retained) {
			return errors.New("member Stop exit receipt ownership mismatch")
		}
		switch j.Phase {
		case "planned":
			if j.Evidence != nil || op.Status != "queued" && op.Status != "interrupted" {
				return errors.New("invalid unstarted Stop exit receipt")
			}
		case "observing":
			if j.Evidence != nil || op.Status != "running" && op.Status != "interrupted" {
				return errors.New("invalid pending Stop exit receipt")
			}
		case "observed":
			if op.Status != "succeeded" || j.Evidence == nil || j.Evidence.OperationID != j.OperationID || !reflect.DeepEqual(j.Evidence.Stop, retained) || j.Evidence.Validate(p.Plan) != nil {
				return errors.New("invalid historical Stop exit evidence")
			}
		default:
			return errors.New("unsupported member Stop exit phase")
		}
		if op.Status == "queued" || op.Status == "running" {
			if active[j.InstanceID] {
				return errors.New("overlapping member Stop exit attempts")
			}
			active[j.InstanceID] = true
		}
		body, err := json.Marshal(j)
		if err != nil {
			return err
		}
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
			return err
		}
		if string(body) != string(mirror) {
			return errors.New("member Stop exit journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='observe-project-member-stop-exit' AND id NOT IN (SELECT operation_id FROM project_member_stop_exits)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member Stop exit receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberStopExit(tx *sql.Tx, j projectMemberStopExitJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_stop_exits SET body=? WHERE operation_id=? AND project_id=? AND instance_id=?", body, j.OperationID, j.ProjectID, j.InstanceID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("Stop exit operation lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberStopExit(actor, project, instance string, req projectMemberStopExitRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.StopOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberStopExitRequest
	}{"observe-project-member-stop-exit", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching Stop exit receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectMemberStopExit(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectReceiptAdmission(tx, project, instance, req.StopOperationID, "stop-project-member"); err != nil {
		return protocol.Operation{}, false, err
	}

	p, retained, err := projectStopExitPrerequisites(tx, project, instance, req.StopOperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("Stop exit must match the reviewed project plan")
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM project_member_stop_exits j JOIN operations o ON j.operation_id=o.id WHERE j.instance_id=? AND json_extract(o.body,'$.status') IN ('queued','running')", instance).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_observation_pending", "this member already has a pending Stop exit observation")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "observe-project-member-stop-exit", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberStopExitJob{OperationID: op.ID, ProjectID: project, InstanceID: instance, StopOperationID: req.StopOperationID, Phase: "planned", Stop: retained}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_stop_exits(operation_id,project_id,instance_id,stop_operation_id,body) VALUES(?,?,?,?,?)", op.ID, project, instance, req.StopOperationID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberStopExit(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberStopExit(opID string) (projectMemberStopExitJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectMemberStopExitJob{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberStopExit(tx); err != nil {
		return projectMemberStopExitJob{}, err
	}
	j, err := loadProjectMemberStopExit(tx, opID)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return j, err
	}
	if j.Phase != "planned" || op.Status != "queued" {
		return j, conflict("stale_project_operation", "operation no longer owns this fresh Stop exit observation")
	}
	j.Phase = "observing"
	op.Status = "running"
	if err = saveProjectMemberStopExit(tx, j); err != nil {
		return j, err
	}
	if err = saveOperation(tx, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}
func (s *store) completeProjectMemberStopExit(opID string, proof projectnetwork.MemberStopExitEvidence) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberStopExit(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberStopExit(tx, opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns Stop exit completion")
	}
	p, retained, err := projectStopExitPrerequisites(tx, j.ProjectID, j.InstanceID, j.StopOperationID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if proof.OperationID != j.OperationID || !reflect.DeepEqual(proof.Stop, retained) || !reflect.DeepEqual(j.Stop, retained) || proof.Validate(p.Plan) != nil {
		return errors.New("member Stop exit completion belongs to another retained selection")
	}
	for _, at := range []time.Time{proof.Observation.CapturedAt, proof.Observation.Runtime.ObservedAt} {
		if at.IsZero() || now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
			return errors.New("member Stop exit completion is stale")
		}
	}
	j.Phase = "observed"
	j.Evidence = &proof
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "member exit observed; original Stop receipt and status remain unchanged"
	if err = saveProjectMemberStopExit(tx, j); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberStopExit(tx *sql.Tx, opID string) error {
	j, err := loadProjectMemberStopExit(tx, opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if op.Kind != "observe-project-member-stop-exit" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("Stop exit interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "Stop exit observation did not complete; a new read-only request may observe again without re-signalling"
	return saveOperation(tx, op)
}
func reconcileProjectMemberStopExit(tx *sql.Tx) error {
	if err := validateProjectMemberStopExit(tx); err != nil {
		return err
	}
	jobs, err := projectMemberStopExitJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberStopExit(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberStopExit(opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberStopExit(tx, opID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberStopExit(ctx context.Context, opID string, d projectnetwork.MemberStopExitDriver) error {
	j, err := s.beginProjectMemberStopExit(opID)
	if err != nil {
		return err
	}
	p, retained, err := projectStopExitPrerequisites(s.db, j.ProjectID, j.InstanceID, j.StopOperationID)
	if err == nil {
		var proof projectnetwork.MemberStopExitEvidence
		proof, err = projectnetwork.CollectMemberStopExit(ctx, p.Plan, retained, j.OperationID, d, time.Now)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = s.completeProjectMemberStopExit(opID, proof)
		}
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberStopExit(opID))
	}
	return nil
}
