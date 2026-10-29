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

type projectMemberReadinessRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

// Each attempt is retained, including interrupted read-only attempts. A new
// request may repeat observation, but cannot rewrite an old success or retry a
// launch. Historical readiness is never current mutation authority.
type projectMemberReadinessJob struct {
	OperationID       string                                 `json:"operation_id"`
	ProjectID         string                                 `json:"project_id"`
	InstanceID        string                                 `json:"instance_id"`
	LaunchOperationID string                                 `json:"launch_operation_id"`
	Phase             string                                 `json:"phase"`
	Binding           projectnetwork.RuntimeBinding          `json:"binding"`
	Evidence          *projectnetwork.InitialMemberReadiness `json:"evidence,omitempty"`
}

func initializeProjectMemberReadiness(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_readiness (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberReadiness(q querier, id string) (projectMemberReadinessJob, error) {
	var j projectMemberReadinessJob
	var project, instance string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,body FROM project_member_readiness WHERE operation_id=?", id).Scan(&project, &instance, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.InstanceID != instance {
		return j, errors.New("invalid member readiness journal")
	}
	return j, nil
}
func projectMemberReadinessJobs(q querier) ([]projectMemberReadinessJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_readiness ORDER BY operation_id")
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
	var jobs []projectMemberReadinessJob
	for _, id := range ids {
		j, err := loadProjectMemberReadiness(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func projectReadinessPrerequisites(q querier, project, instance string) (storedProject, []projectnetwork.MemberLaunch, projectnetwork.RuntimeBinding, string, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, nil, projectnetwork.RuntimeBinding{}, "", err
	}
	var launches []projectnetwork.MemberLaunch
	target := ""
	for _, m := range p.Members {
		j, err := loadProjectMemberLaunch(q, m.InstanceID)
		if err != nil {
			return p, nil, projectnetwork.RuntimeBinding{}, "", err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return p, nil, projectnetwork.RuntimeBinding{}, "", err
		}
		if j.ProjectID != project || j.Launch == nil || j.Launch.Phase != "running" || op.Status != "succeeded" {
			return p, nil, projectnetwork.RuntimeBinding{}, "", conflict("project_not_launched", "complete all initial member launches before observing readiness")
		}
		launches = append(launches, *j.Launch)
		if m.InstanceID == instance {
			target = j.OperationID
		}
	}
	binding, err := projectnetwork.InitialRuntimeBinding(p.Plan, launches)
	if err != nil || target == "" {
		return p, nil, binding, "", errors.New("incomplete readiness launch binding")
	}
	return p, launches, binding, target, nil
}
func validateProjectMemberReadiness(q querier) error {
	return validateSnapshot(q, "validateProjectMemberReadiness", validateProjectMemberReadinessUncached)
}

func validateProjectMemberReadinessUncached(q querier) error {
	if err := validateProjectMemberLaunches(q); err != nil {
		return err
	}
	jobs, err := projectMemberReadinessJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		p, launches, binding, target, err := projectReadinessPrerequisites(q, j.ProjectID, j.InstanceID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Kind != "observe-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" || target != j.LaunchOperationID || !reflect.DeepEqual(j.Binding, binding) {
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
			if op.Status != "succeeded" || j.Evidence == nil || j.Evidence.InstanceID != j.InstanceID || j.Evidence.LaunchOperationID != target || j.Evidence.Validate(p.Plan, launches) != nil {
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
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='observe-project-member' AND id NOT IN (SELECT operation_id FROM project_member_readiness)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member readiness receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberReadiness(tx *sql.Tx, j projectMemberReadinessJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_readiness SET body=? WHERE operation_id=? AND project_id=? AND instance_id=?", body, j.OperationID, j.ProjectID, j.InstanceID)
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
func (s *store) acceptProjectMemberReadiness(actor, project, instance string, req projectMemberReadinessRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberReadinessRequest
	}{"observe-project-member", project, instance, req})
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
	if err = validateProjectMemberReadiness(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectMemberAdmission(tx, project, instance); err != nil {
		return protocol.Operation{}, false, err
	}
	p, _, binding, target, err := projectReadinessPrerequisites(tx, project, instance)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, conflict("project_not_launched", "complete all initial member launches before observing readiness")
	}
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
	if err = tx.QueryRow("SELECT count(*) FROM project_member_readiness j JOIN operations o ON j.operation_id=o.id WHERE j.instance_id=? AND json_extract(o.body,'$.status') IN ('queued','running')", instance).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_observation_pending", "this member already has a pending readiness observation")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "observe-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberReadinessJob{OperationID: op.ID, ProjectID: project, InstanceID: instance, LaunchOperationID: target, Phase: "planned", Binding: binding}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_readiness(operation_id,project_id,instance_id,body) VALUES(?,?,?,?)", op.ID, project, instance, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberReadiness(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberReadiness(opID string) (projectMemberReadinessJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectMemberReadinessJob{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberReadiness(tx); err != nil {
		return projectMemberReadinessJob{}, err
	}
	j, err := loadProjectMemberReadiness(tx, opID)
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
	if err = saveProjectMemberReadiness(tx, j); err != nil {
		return j, err
	}
	if err = saveOperation(tx, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}
func (s *store) completeProjectMemberReadiness(opID string, proof projectnetwork.InitialMemberReadiness) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberReadiness(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberReadiness(tx, opID)
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
	p, launches, binding, target, err := projectReadinessPrerequisites(tx, j.ProjectID, j.InstanceID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if proof.InstanceID != j.InstanceID || proof.LaunchOperationID != target || !reflect.DeepEqual(binding, proof.Binding) || proof.Validate(p.Plan, launches) != nil || proof.CapturedAt.IsZero() || now.Sub(proof.CapturedAt) > 30*time.Second || proof.CapturedAt.Sub(now) > 5*time.Second {
		return errors.New("member readiness completion is stale or belongs to another selection")
	}
	j.Phase = "observed"
	j.Evidence = &proof
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "member identity and initialized storage observed; this receipt is historical evidence"
	if err = saveProjectMemberReadiness(tx, j); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberReadiness(tx *sql.Tx, opID string) error {
	j, err := loadProjectMemberReadiness(tx, opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if op.Kind != "observe-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
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
func reconcileProjectMemberReadiness(tx *sql.Tx) error {
	if err := validateProjectMemberReadiness(tx); err != nil {
		return err
	}
	jobs, err := projectMemberReadinessJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberReadiness(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberReadiness(opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberReadiness(tx, opID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberReadiness(ctx context.Context, opID string, d projectnetwork.InitialReadinessDriver) error {
	j, err := s.beginProjectMemberReadiness(opID)
	if err != nil {
		return err
	}
	p, launches, _, _, err := projectReadinessPrerequisites(s.db, j.ProjectID, j.InstanceID)
	if err == nil {
		var proof projectnetwork.InitialMemberReadiness
		proof, err = projectnetwork.CollectInitialMemberReadiness(ctx, p.Plan, launches, j.InstanceID, d, time.Now)
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = s.completeProjectMemberReadiness(opID, proof)
		}
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberReadiness(opID))
	}
	return nil
}
