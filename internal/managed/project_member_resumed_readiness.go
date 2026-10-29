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

type projectMemberResumedReadinessRequest struct {
	ResumeOperationID string `json:"resume_operation_id"`
	RequestID         string `json:"request_id"`
	ExpectedRevision  uint64 `json:"expected_revision"`
	ReviewDigest      string `json:"review_digest"`
	ReplayOnly        bool   `json:"-"`
}

// Each attempt is retained, including interrupted read-only attempts. A new
// request may repeat observation, but cannot rewrite an old success or retry a
// launch. Historical readiness is never current mutation authority.
type projectMemberResumedReadinessJob struct {
	OperationID       string                                 `json:"operation_id"`
	ProjectID         string                                 `json:"project_id"`
	InstanceID        string                                 `json:"instance_id"`
	ResumeOperationID string                                 `json:"resume_operation_id"`
	Phase             string                                 `json:"phase"`
	Binding           projectnetwork.RuntimeBinding          `json:"binding"`
	Evidence          *projectnetwork.ResumedMemberReadiness `json:"evidence,omitempty"`
}

func initializeProjectMemberResumedReadiness(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_resumed_readiness (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), resume_operation_id TEXT NOT NULL REFERENCES project_member_resumes(operation_id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberResumedReadiness(q querier, id string) (projectMemberResumedReadinessJob, error) {
	var j projectMemberResumedReadinessJob
	var project, instance, resume string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,resume_operation_id,body FROM project_member_resumed_readiness WHERE operation_id=?", id).Scan(&project, &instance, &resume, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.InstanceID != instance || j.ResumeOperationID != resume {
		return j, errors.New("invalid member readiness journal")
	}
	return j, nil
}
func projectMemberResumedReadinessJobs(q querier) ([]projectMemberResumedReadinessJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_resumed_readiness ORDER BY operation_id")
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
	var jobs []projectMemberResumedReadinessJob
	for _, id := range ids {
		j, err := loadProjectMemberResumedReadiness(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func projectResumedReadinessPrerequisites(q querier, project, instance, resumeID string) (storedProject, projectnetwork.MemberResume, projectnetwork.RuntimeBinding, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, projectnetwork.MemberResume{}, projectnetwork.RuntimeBinding{}, err
	}
	j, err := loadProjectMemberResume(q, resumeID)
	if err != nil {
		return p, projectnetwork.MemberResume{}, projectnetwork.RuntimeBinding{}, err
	}
	op, err := loadOperation(q, resumeID)
	if err != nil {
		return p, projectnetwork.MemberResume{}, projectnetwork.RuntimeBinding{}, err
	}
	if j.ProjectID != project || j.InstanceID != instance || op.Status != "succeeded" || j.Resume == nil || j.Resume.Phase != "running" {
		return p, projectnetwork.MemberResume{}, projectnetwork.RuntimeBinding{}, conflict("project_not_resumed", "readiness requires a completed exact replacement receipt")
	}
	b, err := projectnetwork.ResumedRuntimeBinding(p.Plan, *j.Resume)
	return p, *j.Resume, b, err
}
func validateProjectMemberResumedReadiness(q querier) error {
	return validateSnapshot(q, "validateProjectMemberResumedReadiness", validateProjectMemberResumedReadinessUncached)
}

func validateProjectMemberResumedReadinessUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberResumes(q); err != nil {
		return err
	}
	jobs, err := projectMemberResumedReadinessJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		p, resumed, binding, err := projectResumedReadinessPrerequisites(q, j.ProjectID, j.InstanceID, j.ResumeOperationID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Kind != "observe-resumed-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" || resumed.OperationID != j.ResumeOperationID || !reflect.DeepEqual(j.Binding, binding) {
			return errors.New("member readiness receipt ownership mismatch")
		}
		switch j.Phase {
		case "planned":
			if j.Evidence != nil || op.Status != "queued" && op.Status != "interrupted" {
				return errors.New("invalid unstarted readiness receipt")
			}
		case "observing":
			if j.Evidence != nil || op.Status != "running" && op.Status != "interrupted" {
				return errors.New("invalid pending readiness receipt")
			}
		case "observed":
			if op.Status != "succeeded" || j.Evidence == nil || j.Evidence.InstanceID != j.InstanceID || j.Evidence.ResumeOperationID != resumed.OperationID || j.Evidence.Validate(p.Plan, resumed) != nil {
				return errors.New("invalid historical readiness evidence")
			}
		default:
			return errors.New("unsupported member readiness phase")
		}
		if op.Status == "queued" || op.Status == "running" {
			if active[j.InstanceID] {
				return errors.New("overlapping member readiness attempts")
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
			return errors.New("member readiness journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='observe-resumed-project-member' AND id NOT IN (SELECT operation_id FROM project_member_resumed_readiness)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member readiness receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberResumedReadiness(tx *sql.Tx, j projectMemberResumedReadinessJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_resumed_readiness SET body=? WHERE operation_id=? AND project_id=? AND instance_id=? AND resume_operation_id=?", body, j.OperationID, j.ProjectID, j.InstanceID, j.ResumeOperationID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("readiness operation lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberResumedReadiness(actor, project, instance string, req projectMemberResumedReadinessRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.ResumeOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberResumedReadinessRequest
	}{"observe-resumed-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching readiness receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectMemberResumedReadiness(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectReceiptAdmission(tx, project, instance, req.ResumeOperationID, "resume-project-member"); err != nil {
		return protocol.Operation{}, false, err
	}

	p, resumed, binding, err := projectResumedReadinessPrerequisites(tx, project, instance, req.ResumeOperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("readiness must match the reviewed project plan")
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_observation_pending", "finish the project's active work before observing replacement readiness")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "observe-resumed-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberResumedReadinessJob{OperationID: op.ID, ProjectID: project, InstanceID: instance, ResumeOperationID: resumed.OperationID, Phase: "planned", Binding: binding}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_resumed_readiness(operation_id,project_id,instance_id,resume_operation_id,body) VALUES(?,?,?,?,?)", op.ID, project, instance, resumed.OperationID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberResumedReadiness(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberResumedReadiness(opID string) (projectMemberResumedReadinessJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectMemberResumedReadinessJob{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberResumedReadiness(tx); err != nil {
		return projectMemberResumedReadinessJob{}, err
	}
	j, err := loadProjectMemberResumedReadiness(tx, opID)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return j, err
	}
	if j.Phase != "planned" || op.Status != "queued" {
		return j, conflict("stale_project_operation", "operation no longer owns this fresh readiness observation")
	}
	j.Phase = "observing"
	op.Status = "running"
	if err = saveProjectMemberResumedReadiness(tx, j); err != nil {
		return j, err
	}
	if err = saveOperation(tx, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}
func (s *store) completeProjectMemberResumedReadiness(opID string, proof projectnetwork.ResumedMemberReadiness) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberResumedReadiness(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberResumedReadiness(tx, opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns readiness completion")
	}
	p, resumed, binding, err := projectResumedReadinessPrerequisites(tx, j.ProjectID, j.InstanceID, j.ResumeOperationID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if proof.InstanceID != j.InstanceID || proof.ResumeOperationID != resumed.OperationID || !reflect.DeepEqual(binding, proof.Binding) || proof.Validate(p.Plan, resumed) != nil || proof.CapturedAt.IsZero() || now.Sub(proof.CapturedAt) > 30*time.Second || proof.CapturedAt.Sub(now) > 5*time.Second {
		return errors.New("member readiness completion is stale or belongs to another selection")
	}
	j.Phase = "observed"
	j.Evidence = &proof
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "member identity and initialized storage observed; this receipt is historical evidence"
	if err = saveProjectMemberResumedReadiness(tx, j); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberResumedReadiness(tx *sql.Tx, opID string) error {
	j, err := loadProjectMemberResumedReadiness(tx, opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if op.Kind != "observe-resumed-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("readiness interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "readiness observation did not complete; a new read-only request may retry without relaunching the member"
	return saveOperation(tx, op)
}
func reconcileProjectMemberResumedReadiness(tx *sql.Tx) error {
	if err := validateProjectMemberResumedReadiness(tx); err != nil {
		return err
	}
	jobs, err := projectMemberResumedReadinessJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberResumedReadiness(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberResumedReadiness(opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberResumedReadiness(tx, opID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberResumedReadiness(ctx context.Context, opID string, d projectnetwork.ResumedReadinessDriver) error {
	j, err := s.beginProjectMemberResumedReadiness(opID)
	if err != nil {
		return err
	}
	p, resumed, _, err := projectResumedReadinessPrerequisites(s.db, j.ProjectID, j.InstanceID, j.ResumeOperationID)
	if err == nil {
		var proof projectnetwork.ResumedMemberReadiness
		proof, err = projectnetwork.CollectResumedMemberReadiness(ctx, p.Plan, resumed, d, time.Now)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = s.completeProjectMemberResumedReadiness(opID, proof)
		}
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberResumedReadiness(opID))
	}
	return nil
}
