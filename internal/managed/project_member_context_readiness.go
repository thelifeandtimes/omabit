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

type projectMemberContextReadinessRequest struct {
	OriginalReadinessOperationID string `json:"original_readiness_operation_id"`
	CreatorOperationID           string `json:"creator_operation_id"`
	RequestID                    string `json:"request_id"`
	ExpectedRevision             uint64 `json:"expected_revision"`
	ReviewDigest                 string `json:"review_digest"`
	ReplayOnly                   bool   `json:"-"`
}
type projectMemberContextReadinessJob struct {
	OperationID                  string                                 `json:"operation_id"`
	ProjectID                    string                                 `json:"project_id"`
	InstanceID                   string                                 `json:"instance_id"`
	OriginalReadinessOperationID string                                 `json:"original_readiness_operation_id"`
	Context                      projectnetwork.MemberContext           `json:"context"`
	Phase                        string                                 `json:"phase"`
	Evidence                     *projectnetwork.CurrentMemberReadiness `json:"evidence,omitempty"`
}
type projectContextReadinessPrerequisites struct {
	RecoveryHistory *projectnetwork.AnchorRecoveryHistory
	Project         storedProject
	Launches        []projectnetwork.MemberLaunch
	Resumes         []projectnetwork.MemberResume
	Original        projectnetwork.InitialMemberReadiness
	Context         projectnetwork.MemberContext
}

func initializeProjectMemberContextReadiness(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_context_readiness (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), original_readiness_operation_id TEXT NOT NULL REFERENCES project_member_readiness(operation_id), creator_operation_id TEXT NOT NULL REFERENCES operations(id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberContextReadiness(q querier, id string) (projectMemberContextReadinessJob, error) {
	var j projectMemberContextReadinessJob
	var project, instance, original, creator string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,original_readiness_operation_id,creator_operation_id,body FROM project_member_context_readiness WHERE operation_id=?", id).Scan(&project, &instance, &original, &creator, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.OperationID != id || j.ProjectID != project || j.InstanceID != instance || j.OriginalReadinessOperationID != original || j.Context.Member.OperationID != creator {
		return j, errors.New("invalid current-context readiness journal")
	}
	return j, nil
}
func projectMemberContextReadinessJobs(q querier) ([]projectMemberContextReadinessJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_context_readiness ORDER BY operation_id")
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
	var jobs []projectMemberContextReadinessJob
	for _, id := range ids {
		j, err := loadProjectMemberContextReadiness(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// This raw prerequisite loader must not call the lifecycle validators: a future
// Stop may refer back to this readiness. Global validation checks every journal
// independently; historical reads resolve a complete, exact causal prefix.
func projectContextReadinessSource(q querier, project, instance, originalID string, binding *projectnetwork.RuntimeBinding) (projectContextReadinessPrerequisites, error) {
	var out projectContextReadinessPrerequisites
	p, launches, _, _, err := projectReadinessPrerequisites(q, project, instance)
	if err != nil {
		return out, err
	}
	j, err := loadProjectMemberReadiness(q, originalID)
	if err != nil {
		return out, err
	}
	op, err := loadOperation(q, originalID)
	if err != nil {
		return out, err
	}
	if j.ProjectID != project || j.InstanceID != instance || j.Phase != "observed" || j.Evidence == nil || j.Evidence.InstanceID != instance || op.Status != "succeeded" || op.Kind != "observe-project-member" || op.ProjectID != project || op.InstanceID != instance || op.ConfigRevision != p.Revision || op.Actor == "" || j.LaunchOperationID != j.Evidence.LaunchOperationID || !reflect.DeepEqual(j.Binding, j.Evidence.Binding) || j.Evidence.Validate(p.Plan, launches) != nil {
		return out, conflict("project_not_observed", "current context requires matching original initialized-pier evidence")
	}
	body, _ := json.Marshal(j)
	var mirror []byte
	if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
		return out, err
	}
	if string(body) != string(mirror) {
		return out, errors.New("original readiness mirror changed")
	}
	if recovered, err := recoveredContextSource(q, p, launches, *j.Evidence, binding); err != nil {
		return out, err
	} else if recovered != nil {
		return *recovered, nil
	}
	resumes, err := completedProjectMemberResumes(q, project)
	if err != nil {
		return out, err
	}
	if binding != nil {
		resumes, err = projectnetwork.MemberResumesAtBinding(p.Plan, launches, resumes, *binding)
		if err != nil {
			return out, err
		}
	}
	c, err := projectnetwork.SelectMemberContext(p.Plan, launches, resumes, *j.Evidence)
	if err != nil {
		return out, err
	}
	return projectContextReadinessPrerequisites{Project: p, Launches: launches, Resumes: resumes, Original: *j.Evidence, Context: c}, nil
}
func contextReadinessJobSource(q querier, j projectMemberContextReadinessJob) (projectContextReadinessPrerequisites, protocol.Operation, error) {
	src, err := projectContextReadinessSource(q, j.ProjectID, j.InstanceID, j.OriginalReadinessOperationID, &j.Context.Binding)
	if err != nil {
		return src, protocol.Operation{}, err
	}
	op, err := loadOperation(q, j.OperationID)
	if err != nil {
		return src, op, err
	}
	if op.Kind != "observe-current-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != src.Project.Revision || op.Actor == "" || !reflect.DeepEqual(j.Context, src.Context) {
		return src, op, errors.New("current readiness ownership mismatch")
	}
	switch j.Phase {
	case "planned":
		if j.Evidence != nil || op.Status != "queued" && op.Status != "interrupted" {
			return src, op, errors.New("invalid unstarted current readiness")
		}
	case "observing":
		if j.Evidence != nil || op.Status != "running" && op.Status != "interrupted" {
			return src, op, errors.New("invalid pending current readiness")
		}
	case "observed":
		if op.Status != "succeeded" || j.Evidence == nil || contextReadinessProofValid(src, *j.Evidence) != nil {
			return src, op, errors.New("invalid historical current readiness")
		}
	default:
		return src, op, errors.New("unsupported current readiness phase")
	}
	body, _ := json.Marshal(j)
	var mirror []byte
	if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
		return src, op, err
	}
	if string(body) != string(mirror) {
		return src, op, errors.New("current readiness journal and receipt disagree")
	}
	return src, op, nil
}
func validateProjectMemberContextReadiness(q querier) error {
	return validateSnapshot(q, "validateProjectMemberContextReadiness", validateProjectMemberContextReadinessUncached)
}

func validateProjectMemberContextReadinessUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberResumedReadiness(q); err != nil {
		return err
	}
	jobs, err := projectMemberContextReadinessJobs(q)
	if err != nil {
		return err
	}
	active := map[string]bool{}
	for _, j := range jobs {
		_, op, err := contextReadinessJobSource(q, j)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" {
			if active[j.ProjectID] {
				return errors.New("overlapping current readiness")
			}
			active[j.ProjectID] = true
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='observe-current-project-member' AND id NOT IN (SELECT operation_id FROM project_member_context_readiness)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("current readiness receipt is missing journal")
	}
	return nil
}
func saveProjectMemberContextReadiness(tx *sql.Tx, j projectMemberContextReadinessJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_context_readiness SET body=? WHERE operation_id=? AND project_id=? AND instance_id=? AND original_readiness_operation_id=? AND creator_operation_id=?", body, j.OperationID, j.ProjectID, j.InstanceID, j.OriginalReadinessOperationID, j.Context.Member.OperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("current readiness lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberContextReadiness(actor, project, instance string, req projectMemberContextReadinessRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.OriginalReadinessOperationID) || !protocol.ValidID(req.CreatorOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, original readiness, current creator, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberContextReadinessRequest
	}{"observe-current-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching current readiness receipt exists")
	}
	if err = projectExpansionClear(tx, project); err != nil {
		return protocol.Operation{}, false, err
	}

	if err = projectRetirementClear(tx, project, ""); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = validateProjectMemberContextReadiness(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = projectReceiptAdmission(tx, project, instance, req.OriginalReadinessOperationID, "observe-project-member"); err != nil {
		return protocol.Operation{}, false, err
	}

	src, err := projectContextReadinessSource(tx, project, instance, req.OriginalReadinessOperationID, nil)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if src.Project.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := src.Project.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("current readiness must match the reviewed project plan")
	}
	if err = currentContextObservationAllowed(tx, src.Context); err != nil {
		return protocol.Operation{}, false, err
	}
	if src.Context.Member.OperationID != req.CreatorOperationID {
		return protocol.Operation{}, false, conflict("project_recovery_required", "member creator changed; review current history")
	}
	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_observation_pending", "finish active project work before current readiness")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "observe-current-project-member", ConfigRevision: src.Project.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberContextReadinessJob{OperationID: op.ID, ProjectID: project, InstanceID: instance, OriginalReadinessOperationID: req.OriginalReadinessOperationID, Context: src.Context, Phase: "planned"}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_context_readiness(operation_id,project_id,instance_id,original_readiness_operation_id,creator_operation_id,body) VALUES(?,?,?,?,?,?)", op.ID, project, instance, req.OriginalReadinessOperationID, req.CreatorOperationID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberContextReadiness(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

// A read-only refresh cannot turn an unrecorded restart into readiness. This
// first all-running path rejects Stop intent for any current container; earlier
// retired generations remain historical and do not block a new creator.
func currentContextObservationAllowed(q querier, c projectnetwork.MemberContext) error {
	jobs, err := projectMemberStopJobs(q)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.ProjectID != c.Binding.ProjectID {
			continue
		}
		for _, m := range c.Binding.Members {
			if j.ContainerID == m.Container.ID {
				return conflict("project_recovery_required", "a current member has retained Stop intent; complete explicit lifecycle recovery before observing running readiness")
			}
		}
	}
	return nil
}
func currentContextUnchanged(q querier, j projectMemberContextReadinessJob) (projectContextReadinessPrerequisites, error) {
	src, err := projectContextReadinessSource(q, j.ProjectID, j.InstanceID, j.OriginalReadinessOperationID, nil)
	if err != nil {
		return src, err
	}
	if !reflect.DeepEqual(src.Context, j.Context) {
		return src, conflict("project_recovery_required", "readiness no longer targets current membership")
	}
	if err = currentContextObservationAllowed(q, src.Context); err != nil {
		return src, err
	}
	return src, nil
}
func (s *store) beginProjectMemberContextReadiness(id string) (projectMemberContextReadinessJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectMemberContextReadinessJob{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberContextReadiness(tx); err != nil {
		return projectMemberContextReadinessJob{}, err
	}
	j, err := loadProjectMemberContextReadiness(tx, id)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return j, err
	}
	if j.Phase != "planned" || op.Status != "queued" {
		return j, conflict("stale_project_operation", "current readiness is no longer queued")
	}
	if _, err = currentContextUnchanged(tx, j); err != nil {
		return j, err
	}
	j.Phase = "observing"
	op.Status = "running"
	if err = saveProjectMemberContextReadiness(tx, j); err != nil {
		return j, err
	}
	if err = saveOperation(tx, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}
func (s *store) completeProjectMemberContextReadiness(id string, proof projectnetwork.CurrentMemberReadiness) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberContextReadiness(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberContextReadiness(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" {
		return conflict("stale_project_operation", "current readiness no longer owns completion")
	}
	src, err := currentContextUnchanged(tx, j)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if contextReadinessProofValid(src, proof) != nil || proof.CapturedAt.IsZero() || now.Sub(proof.CapturedAt) > 30*time.Second || proof.CapturedAt.Sub(now) > 5*time.Second {
		return errors.New("current readiness completion is stale or changed")
	}
	j.Phase = "observed"
	j.Evidence = &proof
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "current member identity, process and initialized storage observed; historical evidence only"
	if err = saveProjectMemberContextReadiness(tx, j); err != nil {
		return err
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberContextReadiness(tx *sql.Tx, id string) error {
	j, err := loadProjectMemberContextReadiness(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Kind != "observe-current-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("current readiness interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "current readiness did not complete; a new read-only attempt may retry without relaunching"
	return saveOperation(tx, op)
}
func reconcileProjectMemberContextReadiness(tx *sql.Tx) error {
	if err := validateProjectMemberContextReadiness(tx); err != nil {
		return err
	}
	jobs, err := projectMemberContextReadinessJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberContextReadiness(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberContextReadiness(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberContextReadiness(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberContextReadiness(ctx context.Context, id string, d projectnetwork.CurrentReadinessDriver) error {
	j, err := s.beginProjectMemberContextReadiness(id)
	if err != nil {
		return err
	}
	src, err := currentContextUnchanged(s.db, j)
	if err == nil {
		var proof projectnetwork.CurrentMemberReadiness
		if src.RecoveryHistory != nil {
			proof, err = projectnetwork.CollectRecoveredCurrentMemberReadiness(ctx, *src.RecoveryHistory, j.InstanceID, d, time.Now)
		} else {
			proof, err = projectnetwork.CollectCurrentMemberReadiness(ctx, src.Project.Plan, src.Launches, src.Resumes, src.Original, d, time.Now)
		}
		if err == nil {
			err = ctx.Err()
		}
		if err == nil {
			err = s.completeProjectMemberContextReadiness(id, proof)
		}
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberContextReadiness(id))
	}
	return nil
}
