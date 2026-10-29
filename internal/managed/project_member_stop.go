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

type projectMemberStopRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedRevision     uint64 `json:"expected_revision"`
	ReviewDigest         string `json:"review_digest"`
	ReadinessOperationID string `json:"readiness_operation_id"`
	ReplayOnly           bool   `json:"-"`
}
type projectMemberStopJob struct {
	ProjectID   string                    `json:"project_id"`
	InstanceID  string                    `json:"instance_id"`
	ContainerID string                    `json:"container_id"`
	OperationID string                    `json:"operation_id"`
	Stop        projectnetwork.MemberStop `json:"stop"`
}

func initializeProjectMemberStops(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_stops (operation_id TEXT PRIMARY KEY REFERENCES operations(id), project_id TEXT NOT NULL REFERENCES projects(id), instance_id TEXT NOT NULL REFERENCES project_members(instance_id), container_id TEXT NOT NULL, body BLOB NOT NULL, UNIQUE(instance_id,container_id));`)
	return err
}
func loadProjectMemberStop(q querier, id string) (projectMemberStopJob, error) {
	var j projectMemberStopJob
	var project, instance, container string
	var body []byte
	err := q.QueryRow("SELECT project_id,instance_id,container_id,body FROM project_member_stops WHERE operation_id=?", id).Scan(&project, &instance, &container, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.ProjectID != project || j.InstanceID != instance || j.ContainerID != container || j.OperationID != id {
		return j, errors.New("invalid member Stop journal")
	}
	return j, nil
}
func projectMemberStopJobs(q querier) ([]projectMemberStopJob, error) {
	rows, err := q.Query("SELECT operation_id FROM project_member_stops ORDER BY operation_id")
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
	var jobs []projectMemberStopJob
	for _, id := range ids {
		j, err := loadProjectMemberStop(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
func initialMemberStopSelection(q querier, project, instance, readiness, operation string) (storedProject, projectnetwork.MemberStop, string, error) {
	p, launches, _, _, err := projectReadinessPrerequisites(q, project, instance)
	if err != nil {
		return p, projectnetwork.MemberStop{}, "", err
	}
	ready, err := loadProjectMemberReadiness(q, readiness)
	if err != nil {
		return p, projectnetwork.MemberStop{}, "", err
	}
	op, err := loadOperation(q, readiness)
	if err != nil {
		return p, projectnetwork.MemberStop{}, "", err
	}
	if ready.ProjectID != project || ready.InstanceID != instance || ready.Phase != "observed" || ready.Evidence == nil || op.Status != "succeeded" {
		return p, projectnetwork.MemberStop{}, "", conflict("project_not_observed", "complete an initial member readiness observation before this Stop")
	}
	state, err := projectnetwork.NewInitialMemberStop(p.Plan, launches, *ready.Evidence, readiness, operation)
	if err != nil {
		return p, state, "", err
	}
	for _, m := range state.Binding.Members {
		if m.InstanceID == instance {
			return p, state, m.Container.ID, nil
		}
	}
	return p, state, "", errors.New("member Stop lost its container binding")
}
func validateProjectMemberStops(q querier) error {
	return validateSnapshot(q, "validateProjectMemberStops", validateProjectMemberStopsUncached)
}

func validateProjectMemberStopsUncached(q querier) error {
	q = projectRecoveryReadOnly(q)
	if err := validateProjectMemberReadiness(q); err != nil {
		return err
	}
	jobs, err := projectMemberStopJobs(q)
	if err != nil {
		return err
	}
	pending := map[string]bool{}
	for _, j := range jobs {
		p, expected, container, err := memberStopSelection(q, j.ProjectID, j.InstanceID, j.Stop.ReadinessOperationID, j.OperationID, false)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		baseline := j.Stop
		baseline.Phase = "planned"
		baseline.Before = nil
		baseline.After = nil
		if container != j.ContainerID || j.Stop.Validate(p.Plan) != nil || !reflect.DeepEqual(baseline, expected) || op.Kind != "stop-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("member Stop receipt ownership mismatch")
		}
		if err = validateContextMemberStopProcess(q, j.Stop); err != nil {
			return err
		}
		switch op.Status {
		case "queued":
			if j.Stop.Phase != "planned" {
				return errors.New("queued member Stop already dispatched")
			}
		case "running", "interrupted":
		case "succeeded":
			if j.Stop.Phase != "stopped" || !j.Stop.Clean() {
				return errors.New("member Stop success lacks clean exit evidence")
			}
		case "failed":
			if j.Stop.Phase != "stopped" || j.Stop.Clean() {
				return errors.New("member Stop failure lacks unclean exit evidence")
			}
		default:
			return errors.New("unsupported member Stop status")
		}
		resumes, err := completedProjectMemberResumes(q, j.ProjectID)
		if err != nil {
			return err
		}
		if op.Status == "queued" || op.Status == "running" || op.Status == "interrupted" && j.Stop.Phase != "stopped" && !stopWasReplaced(resumes, j.Stop) {
			if pending[j.ProjectID] {
				return errors.New("overlapping unresolved project Stops")
			}
			pending[j.ProjectID] = true
		}
		body, _ := json.Marshal(j)
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", j.OperationID).Scan(&mirror); err != nil {
			return err
		}
		if string(body) != string(mirror) {
			return errors.New("member Stop journal and receipt disagree")
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='stop-project-member' AND id NOT IN (SELECT operation_id FROM project_member_stops)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member Stop receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberStop(tx *sql.Tx, j projectMemberStopJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	result, err := tx.Exec("UPDATE project_member_stops SET body=? WHERE operation_id=? AND project_id=? AND instance_id=? AND container_id=?", body, j.OperationID, j.ProjectID, j.InstanceID, j.ContainerID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("member Stop lost its journal")
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberStop(actor, project, instance string, req projectMemberStopRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.ReadinessOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request, readiness and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberStopRequest
	}{"stop-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching member Stop receipt exists")
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
	if err = projectReceiptAdmission(tx, project, instance, req.ReadinessOperationID, "observe-project-member", "observe-resumed-project-member", "observe-current-project-member"); err != nil {
		return protocol.Operation{}, false, err
	}

	var active int
	if err = tx.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND json_extract(body,'$.status') IN ('queued','running')", project).Scan(&active); err != nil {
		return protocol.Operation{}, false, err
	}
	if active != 0 {
		return protocol.Operation{}, false, conflict("project_operation_pending", "finish the project's active work before Stop")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	id = "op-" + id
	p, initial, container, err := memberStopSelection(tx, project, instance, req.ReadinessOperationID, id, true)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("member Stop must match the reviewed project plan")
	}
	jobs, err := projectMemberStopJobs(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, j := range jobs {
		if j.InstanceID == instance && j.ContainerID == container {
			return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original Stop receipt before any further action on this container")
		}
		if j.ProjectID == project {
			op, err := loadOperation(tx, j.OperationID)
			if err != nil {
				return protocol.Operation{}, false, err
			}
			resumes, err := completedProjectMemberResumes(tx, project)
			if err != nil {
				return protocol.Operation{}, false, err
			}
			if op.Status == "queued" || op.Status == "running" || op.Status == "interrupted" && j.Stop.Phase != "stopped" && !stopWasReplaced(resumes, j.Stop) {
				return protocol.Operation{}, false, conflict("project_recovery_required", "resolve the project's pending Stop before another lifecycle action")
			}
		}
	}
	op := protocol.Operation{ID: id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "stop-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberStopJob{ProjectID: project, InstanceID: instance, ContainerID: container, OperationID: id, Stop: initial}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_stops(operation_id,project_id,instance_id,container_id,body) VALUES(?,?,?,?,?)", id, project, instance, container, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = saveProjectMemberStop(tx, j); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberStop(id string) (projectnetwork.MemberStop, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return projectnetwork.MemberStop{}, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberContextReadiness(tx); err != nil {
		return projectnetwork.MemberStop{}, err
	}
	j, err := loadProjectMemberStop(tx, id)
	if err != nil {
		return j.Stop, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return j.Stop, err
	}
	if op.Status != "queued" || j.Stop.Phase != "planned" {
		return j.Stop, conflict("stale_project_operation", "operation no longer owns this initial member Stop")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return j.Stop, err
	}
	return j.Stop, tx.Commit()
}
func (s *store) advanceProjectMemberStop(id, expected string, next projectnetwork.MemberStop) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberStops(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberStop(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" {
		return conflict("stale_project_operation", "operation no longer owns member Stop progression")
	}
	p, err := loadProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	var observation *projectnetwork.MemberStopObservation
	switch next.Phase {
	case "stop-dispatched":
		observation = next.Before
	case "stopped":
		observation = next.After
	}
	wanted, err := j.Stop.Advance(p.Plan, expected, next.Phase, observation)
	if err != nil || !reflect.DeepEqual(wanted, next) {
		return errors.New("member Stop transition changed its accepted binding")
	}
	if observation != nil {
		now := time.Now().UTC()
		for _, at := range []time.Time{observation.CapturedAt, observation.Runtime.ObservedAt} {
			if at.IsZero() || now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
				return errors.New("member Stop transition evidence is stale")
			}
		}
	}
	if err = validateContextMemberStopProcess(tx, wanted); err != nil {
		return err
	}
	j.Stop = wanted
	if err = saveProjectMemberStop(tx, j); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) finishProjectMemberStop(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberStops(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberStop(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Stop.Phase != "stopped" || !j.Stop.Clean() {
		return conflict("stale_project_operation", "member Stop lacks a clean acknowledged completion")
	}
	now := time.Now().UTC()
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "member stopped cleanly; pier, container and project network retained"
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func interruptProjectMemberStop(tx *sql.Tx, id string) error {
	j, err := loadProjectMemberStop(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.Kind != "stop-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("member Stop interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "member Stop is unresolved; retain dispatch and exit evidence without automatic re-signal or removal"
	if j.Stop.Phase == "stopped" && !j.Stop.Clean() {
		op.Status = "failed"
		op.Message = "member exited uncleanly; retain pier and exact exit evidence for review"
	}
	return saveOperation(tx, op)
}
func reconcileProjectMemberStops(tx *sql.Tx) error {
	if err := validateProjectMemberStops(tx); err != nil {
		return err
	}
	jobs, err := projectMemberStopJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberStop(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberStop(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberStop(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberStop(ctx context.Context, id string, d projectnetwork.MemberStopDriver) error {
	initial, err := s.beginProjectMemberStop(id)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, initial.Binding.ProjectID)
	if err == nil {
		_, err = projectnetwork.ExecuteMemberStop(ctx, p.Plan, initial, d, func(expected string, next projectnetwork.MemberStop) error {
			return s.advanceProjectMemberStop(id, expected, next)
		}, time.Now)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = s.finishProjectMemberStop(id)
	}
	if err != nil {
		return errors.Join(err, s.interruptMemberStop(id))
	}
	return nil
}
