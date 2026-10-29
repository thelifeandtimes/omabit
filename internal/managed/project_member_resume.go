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

type projectMemberResumeRequest struct {
	RequestID             string `json:"request_id"`
	ExpectedRevision      uint64 `json:"expected_revision"`
	ReviewDigest          string `json:"review_digest"`
	RetirementOperationID string `json:"retirement_operation_id"`
	ReplayOnly            bool   `json:"-"`
}
type projectMemberResumeJob struct {
	ProjectID             string                          `json:"project_id"`
	InstanceID            string                          `json:"instance_id"`
	OperationID           string                          `json:"operation_id"`
	RetirementOperationID string                          `json:"retirement_operation_id"`
	Retirement            projectnetwork.MemberRetirement `json:"retirement"`
	Resume                *projectnetwork.MemberResume    `json:"resume,omitempty"`
}

// Each replacement consumes one retirement; later generations retain prior rows.
// It does not overwrite initial launch or historical lifecycle receipts.
func initializeProjectMemberResumes(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_resumes (operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES projects(id),instance_id TEXT NOT NULL REFERENCES project_members(instance_id),retirement_operation_id TEXT NOT NULL UNIQUE REFERENCES project_member_retirements(operation_id),body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberResume(q querier, id string) (projectMemberResumeJob, error) {
	var j projectMemberResumeJob
	var project, instance, retirement string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,retirement_operation_id,body FROM project_member_resumes WHERE operation_id=?", id).Scan(&project, &instance, &retirement, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.InstanceID != instance || j.RetirementOperationID != retirement {
		return j, errors.New("invalid member resume journal")
	}
	return j, nil
}
func projectMemberResumeJobs(q querier) ([]projectMemberResumeJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_resumes ORDER BY operation_id")
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
	var jobs []projectMemberResumeJob
	for _, id := range ids {
		j, err := loadProjectMemberResume(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func memberResumePrerequisites(q querier, project, instance, retirement string) (storedProject, projectnetwork.MemberRetirement, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, projectnetwork.MemberRetirement{}, err
	}
	j, err := loadProjectMemberRetirement(q, retirement)
	if err != nil {
		return p, j.Retirement, err
	}
	op, err := loadOperation(q, retirement)
	if err != nil {
		return p, j.Retirement, err
	}
	if j.ProjectID != project || j.InstanceID != instance || op.Status != "succeeded" || j.Retirement.Phase != "absent" || j.Retirement.Validate(p.Plan) != nil {
		return p, j.Retirement, conflict("project_not_retired", "resume requires its complete clean retirement receipt")
	}
	return p, j.Retirement, nil
}
func validateProjectMemberResumes(q querier) error {
	return validateSnapshot(q, "validateProjectMemberResumes", validateProjectMemberResumesUncached)
}

func validateProjectMemberResumesUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberRetirements(q); err != nil {
		return err
	}
	jobs, err := projectMemberResumeJobs(q)
	if err != nil {
		return err
	}
	pending := map[string]bool{}
	for _, j := range jobs {
		p, retired, err := memberResumePrerequisites(q, j.ProjectID, j.InstanceID, j.RetirementOperationID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(j.Retirement, retired) || op.Kind != "resume-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("member resume lost original retirement or receipt ownership")
		}
		switch op.Status {
		case "queued":
			if j.Resume != nil {
				return errors.New("queued resume already has bound dispatch")
			}
		case "running", "interrupted":
		case "succeeded":
			if j.Resume == nil || j.Resume.Phase != "running" {
				return errors.New("resume success lacks complete running evidence")
			}
		default:
			return errors.New("unsupported member resume status")
		}
		if j.Resume == nil {
			if op.Status != "queued" && op.Status != "interrupted" {
				return errors.New("resume receipt lacks its bound contract")
			}
		} else {
			previous, err := memberResumeGeneration(q, retired, false)
			if err != nil {
				return err
			}
			expected, err := projectnetwork.NewMemberResume(p.Plan, retired, previous, j.OperationID, j.Resume.Container.ContractDigest)
			if err != nil {
				return err
			}
			baseline := *j.Resume
			baseline.Phase = "planned"
			baseline.Container.ID = ""
			baseline.Before, baseline.Created, baseline.After = nil, nil, nil
			if j.Resume.Validate(p.Plan) != nil || !reflect.DeepEqual(expected, baseline) {
				return errors.New("resume contract differs from its original intent")
			}
		}
		if op.Status != "succeeded" {
			if pending[j.ProjectID] {
				return errors.New("overlapping unresolved member resumes")
			}
			pending[j.ProjectID] = true
		}
		body, _ := json.Marshal(j)
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", j.OperationID).Scan(&mirror); err != nil {
			return err
		}
		if string(body) != string(mirror) {
			return errors.New("resume journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='resume-project-member' AND id NOT IN (SELECT operation_id FROM project_member_resumes)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("resume receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberResume(tx *sql.Tx, j projectMemberResumeJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_resumes SET body=? WHERE operation_id=? AND project_id=? AND instance_id=? AND retirement_operation_id=?", body, j.OperationID, j.ProjectID, j.InstanceID, j.RetirementOperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("resume lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberResume(actor, project, instance string, req projectMemberResumeRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.RetirementOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request, retirement and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberResumeRequest
	}{"resume-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching member resume receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = validateProjectMemberResumes(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectReceiptAdmission(tx, project, instance, req.RetirementOperationID, "retire-project-member"); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectResumeClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish the project's active work before resume")
	}
	p, retired, err := memberResumePrerequisites(tx, project, instance, req.RetirementOperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("member resume must match the reviewed project plan")
	}
	var consumed int
	if err = tx.QueryRow("SELECT count(*) FROM project_member_resumes WHERE retirement_operation_id=?", req.RetirementOperationID).Scan(&consumed); err != nil {
		return protocol.Operation{}, false, err
	}
	if consumed != 0 {
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original resume receipt before another action on this member")
	}
	if _, err = memberResumeGeneration(tx, retired, true); err != nil {
		return protocol.Operation{}, false, err
	}
	resumes, err := completedProjectMemberResumes(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	stops, err := projectMemberStopJobs(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, j := range stops {
		if j.ProjectID == project && j.OperationID != retired.Exit.Stop.OperationID && j.Stop.Phase != "stopped" && !stopWasReplaced(resumes, j.Stop) {
			return protocol.Operation{}, false, conflict("project_recovery_required", "another member Stop remains unresolved")
		}
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	id = "op-" + id
	op := protocol.Operation{ID: id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "resume-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberResumeJob{ProjectID: project, InstanceID: instance, OperationID: id, RetirementOperationID: req.RetirementOperationID, Retirement: retired}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_resumes(operation_id,project_id,instance_id,retirement_operation_id,body) VALUES(?,?,?,?,?)", id, project, instance, req.RetirementOperationID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberResume(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberResume(id string, initial projectnetwork.MemberResume) (projectnetwork.MemberResume, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return initial, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberResumes(tx); err != nil {
		return initial, err
	}
	j, err := loadProjectMemberResume(tx, id)
	if err != nil {
		return initial, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return initial, err
	}
	if op.Status != "queued" || j.Resume != nil {
		return initial, conflict("stale_project_operation", "operation no longer owns a fresh member resume")
	}
	p, retired, err := memberResumePrerequisites(tx, j.ProjectID, j.InstanceID, j.RetirementOperationID)
	if err != nil {
		return initial, err
	}
	previous, err := memberResumeGeneration(tx, retired, true)
	if err != nil {
		return initial, err
	}
	expected, err := projectnetwork.NewMemberResume(p.Plan, retired, previous, id, initial.Container.ContractDigest)
	if err != nil || !reflect.DeepEqual(expected, initial) {
		return initial, errors.New("resume adapter changed accepted retirement binding")
	}
	j.Resume = &expected
	op.Status = "running"
	if err = saveProjectMemberResume(tx, j); err != nil {
		return initial, err
	}
	if err = saveOperation(tx, op); err != nil {
		return initial, err
	}
	return expected, tx.Commit()
}
func (s *store) advanceProjectMemberResume(id, expected string, next projectnetwork.MemberResume) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberResumes(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberResume(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Resume == nil {
		return conflict("stale_project_operation", "operation no longer owns resume progression")
	}
	p, err := loadProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	var result projectnetwork.MemberResumeResult
	var times []time.Time
	switch next.Phase {
	case "create-dispatched":
		result.Before = next.Before
		if next.Before != nil {
			times = []time.Time{next.Before.CapturedAt, next.Before.Runtime.ObservedAt}
		}
	case "created":
		result.ContainerID = next.Container.ID
	case "start-dispatched":
		result.Observation = next.Created
	case "running":
		result.Observation = next.After
	}
	if result.Observation != nil {
		times = []time.Time{result.Observation.CapturedAt, result.Observation.Runtime.ObservedAt}
	}
	wanted, err := j.Resume.Advance(p.Plan, expected, next.Phase, result)
	if err != nil || !reflect.DeepEqual(wanted, next) {
		return errors.New("resume transition changed accepted binding")
	}
	now := time.Now().UTC()
	for _, at := range times {
		if at.IsZero() || now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
			return errors.New("resume transition evidence is stale")
		}
	}
	j.Resume = &wanted
	if err = saveProjectMemberResume(tx, j); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) finishProjectMemberResume(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberResumes(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberResume(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Resume == nil || j.Resume.Phase != "running" {
		return conflict("stale_project_operation", "resume lacks complete running-container evidence")
	}
	now := time.Now().UTC()
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "replacement container started over the retained pier; ship readiness requires a separate observation"
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberResume(tx *sql.Tx, id string) error {
	j, err := loadProjectMemberResume(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Kind != "resume-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("resume interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "member resume is unresolved; retain returned identities, pier and dispatch without automatic retry or cleanup"
	return saveOperation(tx, op)
}
func reconcileProjectMemberResumes(tx *sql.Tx) error {
	if err := validateProjectMemberResumes(tx); err != nil {
		return err
	}
	jobs, err := projectMemberResumeJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberResume(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberResume(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberResume(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberResume(ctx context.Context, id string, initial projectnetwork.MemberResume, d projectnetwork.MemberResumeDriver) error {
	state, err := s.beginProjectMemberResume(id, initial)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, state.Retirement.Exit.Stop.Binding.ProjectID)
	if err == nil {
		_, err = projectnetwork.ExecuteMemberResume(ctx, p.Plan, state, d, func(expected string, next projectnetwork.MemberResume) error {
			return s.advanceProjectMemberResume(id, expected, next)
		}, time.Now)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = s.finishProjectMemberResume(id)
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberResume(id))
	}
	return nil
}
func projectResumeClear(q querier, project, except string) error {
	var n int
	if err := q.QueryRow("SELECT count(*) FROM project_member_resumes j JOIN operations o ON j.operation_id=o.id WHERE j.project_id=? AND j.operation_id!=? AND json_extract(o.body,'$.status')!='succeeded'", project, except).Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return conflict("project_recovery_required", "resolve the project's pending member resume before another lifecycle action")
	}
	return nil
}
