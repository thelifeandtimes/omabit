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

type projectMemberRetirementRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ExitOperationID  string `json:"exit_operation_id"`
	ReplayOnly       bool   `json:"-"`
}
type projectMemberRetirementJob struct {
	ProjectID       string                          `json:"project_id"`
	InstanceID      string                          `json:"instance_id"`
	ExitOperationID string                          `json:"exit_operation_id"`
	ContainerID     string                          `json:"container_id"`
	OperationID     string                          `json:"operation_id"`
	Retirement      projectnetwork.MemberRetirement `json:"retirement"`
}

func initializeProjectMemberRetirements(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_retirements (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), container_id TEXT NOT NULL, exit_operation_id TEXT NOT NULL REFERENCES project_member_stop_exits(operation_id), body BLOB NOT NULL, UNIQUE(instance_id,container_id));`)
	return err
}
func loadProjectMemberRetirement(q querier, id string) (projectMemberRetirementJob, error) {
	var j projectMemberRetirementJob
	var project, instance, container, exit string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,container_id,exit_operation_id,body FROM project_member_retirements WHERE operation_id=?", id).Scan(&project, &instance, &container, &exit, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.ProjectID != project || j.InstanceID != instance || j.ContainerID != container || j.ExitOperationID != exit || j.OperationID != id {
		return j, errors.New("invalid member Retirement journal")
	}
	return j, nil
}
func projectMemberRetirementJobs(q querier) ([]projectMemberRetirementJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_retirements ORDER BY operation_id")
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
	var jobs []projectMemberRetirementJob
	for _, id := range ids {
		j, err := loadProjectMemberRetirement(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func initialMemberRetirementSelection(q querier, project, instance, exitID, operation string) (storedProject, projectnetwork.MemberRetirement, string, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, projectnetwork.MemberRetirement{}, "", err
	}
	observed, err := loadProjectMemberStopExit(q, exitID)
	if err != nil {
		return p, projectnetwork.MemberRetirement{}, "", err
	}
	op, err := loadOperation(q, exitID)
	if err != nil {
		return p, projectnetwork.MemberRetirement{}, "", err
	}
	if observed.ProjectID != project || observed.InstanceID != instance || observed.Phase != "observed" || observed.Evidence == nil || op.Status != "succeeded" {
		return p, projectnetwork.MemberRetirement{}, "", conflict("project_exit_not_observed", "retirement requires a complete clean exit observation")
	}
	state, err := projectnetwork.NewMemberRetirement(p.Plan, *observed.Evidence, operation)
	if err != nil {
		return p, state, "", err
	}
	for _, m := range state.Exit.Stop.Binding.Members {
		if m.InstanceID == instance {
			return p, state, m.Container.ID, nil
		}
	}
	return p, state, "", errors.New("retirement lost original container binding")
}
func validateProjectMemberRetirements(q querier) error {
	return validateSnapshot(q, "validateProjectMemberRetirements", validateProjectMemberRetirementsUncached)
}

func validateProjectMemberRetirementsUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberStopExit(q); err != nil {
		return err
	}
	jobs, err := projectMemberRetirementJobs(q)
	if err != nil {
		return err
	}
	pending := map[string]bool{}
	for _, j := range jobs {
		p, expected, container, err := initialMemberRetirementSelection(q, j.ProjectID, j.InstanceID, j.ExitOperationID, j.OperationID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		baseline := j.Retirement
		baseline.Phase = "planned"
		baseline.Before = nil
		baseline.After = nil
		if container != j.ContainerID || j.Retirement.Validate(p.Plan) != nil || !reflect.DeepEqual(baseline, expected) || op.Kind != "retire-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("member Retirement receipt ownership mismatch")
		}
		switch op.Status {
		case "queued":
			if j.Retirement.Phase != "planned" {
				return errors.New("queued member Retirement already dispatched")
			}
		case "running", "interrupted":
		case "succeeded":
			if j.Retirement.Phase != "absent" {
				return errors.New("member retirement success lacks complete absence evidence")
			}
		default:
			return errors.New("unsupported member Retirement status")
		}
		if op.Status == "queued" || op.Status == "running" || op.Status == "interrupted" {
			if pending[j.ProjectID] {
				return errors.New("overlapping unresolved project Retirements")
			}
			pending[j.ProjectID] = true
		}
		body, _ := json.Marshal(j)
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", j.OperationID).Scan(&mirror); err != nil {
			return err
		}
		if string(body) != string(mirror) {
			return errors.New("member Retirement journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='retire-project-member' AND id NOT IN (SELECT operation_id FROM project_member_retirements)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member Retirement receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberRetirement(tx *sql.Tx, j projectMemberRetirementJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_retirements SET body=? WHERE operation_id=? AND project_id=? AND instance_id=? AND container_id=?", body, j.OperationID, j.ProjectID, j.InstanceID, j.ContainerID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("member Retirement lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberRetirement(actor, project, instance string, req projectMemberRetirementRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.ExitOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request, exit observation and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberRetirementRequest
	}{"retire-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching member Retirement receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = validateProjectMemberRetirements(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectReceiptAdmission(tx, project, instance, req.ExitOperationID, "observe-project-member-stop-exit"); err != nil {
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
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish the project's active work before retirement")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	id = "op-" + id
	p, initial, container, err := initialMemberRetirementSelection(tx, project, instance, req.ExitOperationID, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("member Retirement must match the reviewed project plan")
	}
	jobs, err := projectMemberRetirementJobs(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, j := range jobs {
		if j.InstanceID == instance && j.ContainerID == container {
			return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original Retirement receipt before any further action on this container")
		}
		if j.ProjectID == project {
			op, err := loadOperation(tx, j.OperationID)
			if err != nil {
				return protocol.Operation{}, false, err
			}
			if op.Status == "queued" || op.Status == "running" || op.Status == "interrupted" {
				return protocol.Operation{}, false, conflict("project_recovery_required", "resolve the project's pending Retirement before another lifecycle action")
			}
		}
	}
	// Another unresolved Stop is not retired incidentally. The selected original
	// Stop may remain interrupted; its separate clean exit observation is explicit.
	if _, err = memberResumeGeneration(tx, initial, true); err != nil {
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
		if j.ProjectID == project && j.OperationID != initial.Exit.Stop.OperationID && j.Stop.Phase != "stopped" && !stopWasReplaced(resumes, j.Stop) {
			return protocol.Operation{}, false, conflict("project_recovery_required", "another member Stop remains unresolved")
		}
	}
	op := protocol.Operation{ID: id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "retire-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberRetirementJob{ProjectID: project, InstanceID: instance, ContainerID: container, ExitOperationID: req.ExitOperationID, OperationID: id, Retirement: initial}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_retirements(operation_id,project_id,instance_id,container_id,exit_operation_id,body) VALUES(?,?,?,?,?,?)", id, project, instance, container, req.ExitOperationID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberRetirement(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberRetirement(id string) (projectnetwork.MemberRetirement, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectnetwork.MemberRetirement{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberRetirements(tx); err != nil {
		return projectnetwork.MemberRetirement{}, err
	}
	j, err := loadProjectMemberRetirement(tx, id)
	if err != nil {
		return j.Retirement, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return j.Retirement, err
	}
	if op.Status != "queued" || j.Retirement.Phase != "planned" {
		return j.Retirement, conflict("stale_project_operation", "operation no longer owns this initial member Retirement")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return j.Retirement, err
	}
	return j.Retirement, tx.Commit()
}
func (s *store) advanceProjectMemberRetirement(id, expected string, next projectnetwork.MemberRetirement) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberRetirements(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberRetirement(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns member Retirement progression")
	}
	p, err := loadProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	var before *projectnetwork.MemberStopObservation
	var after *projectnetwork.MemberAbsenceObservation
	var times []time.Time
	switch next.Phase {
	case "remove-dispatched":
		before = next.Before
		if before != nil {
			times = []time.Time{before.CapturedAt, before.Runtime.ObservedAt}
		}
	case "absent":
		after = next.After
		if after != nil {
			times = []time.Time{after.CapturedAt, after.Runtime.ObservedAt}
		}
	}
	wanted, err := j.Retirement.Advance(p.Plan, expected, next.Phase, before, after)
	if err != nil || !reflect.DeepEqual(wanted, next) {
		return errors.New("retirement transition changed accepted binding")
	}
	now := time.Now().UTC()
	for _, at := range times {
		if at.IsZero() || now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
			return errors.New("retirement transition evidence is stale")
		}
	}
	j.Retirement = wanted
	if err = saveProjectMemberRetirement(tx, j); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) finishProjectMemberRetirement(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberRetirements(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberRetirement(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Retirement.Phase != "absent" {
		return conflict("stale_project_operation", "member retirement lacks an acknowledged complete absence")
	}
	now := time.Now().UTC()
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "member container retired; initialized pier, history and project network retained"
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberRetirement(tx *sql.Tx, id string) error {
	j, err := loadProjectMemberRetirement(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Kind != "retire-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("member Retirement interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "member retirement is unresolved; retain dispatch, pier and history without automatic removal retry or resume"
	return saveOperation(tx, op)
}
func reconcileProjectMemberRetirements(tx *sql.Tx) error {
	if err := validateProjectMemberRetirements(tx); err != nil {
		return err
	}
	jobs, err := projectMemberRetirementJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberRetirement(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberRetirement(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberRetirement(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberRetirement(ctx context.Context, id string, d projectnetwork.MemberRetirementDriver) error {
	initial, err := s.beginProjectMemberRetirement(id)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, initial.Exit.Stop.Binding.ProjectID)
	if err == nil {
		_, err = projectnetwork.ExecuteMemberRetirement(ctx, p.Plan, initial, d, func(expected string, next projectnetwork.MemberRetirement) error {
			return s.advanceProjectMemberRetirement(id, expected, next)
		}, time.Now)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = s.finishProjectMemberRetirement(id)
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberRetirement(id))
	}
	return nil
}

// Unresolved removal fences every other project worker, including one queued
// earlier. The retiring worker may exclude only its own already-validated intent.
func projectRetirementClear(q querier, project, except string) error {
	if err := projectRecoveryClear(q, project, except); err != nil {
		return err
	}
	if err := projectResumeClear(q, project, except); err != nil {
		return err
	}
	var count int
	if err := q.QueryRow("SELECT count(*) FROM project_member_retirements j JOIN operations o ON j.operation_id=o.id WHERE j.project_id=? AND j.operation_id!=? AND json_extract(o.body,'$.status')!='succeeded'", project, except).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return conflict("project_recovery_required", "retain the unresolved member retirement before another project action")
	}
	return nil
}
