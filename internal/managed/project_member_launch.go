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

type projectMemberLaunchRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}
type projectMemberLaunchJob struct {
	ProjectID   string                       `json:"project_id"`
	InstanceID  string                       `json:"instance_id"`
	OperationID string                       `json:"operation_id"`
	Launch      *projectnetwork.MemberLaunch `json:"launch,omitempty"`
}

func initializeProjectMemberLaunches(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_member_launches (instance_id TEXT PRIMARY KEY REFERENCES project_members(instance_id), project_id TEXT NOT NULL REFERENCES projects(id), operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), body BLOB NOT NULL);`)
	return err
}
func loadProjectMemberLaunch(q querier, id string) (projectMemberLaunchJob, error) {
	var j projectMemberLaunchJob
	var project, op string
	var body []byte
	err := q.QueryRow("SELECT project_id,operation_id,body FROM project_member_launches WHERE instance_id=?", id).Scan(&project, &op, &body)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(body, &j) != nil || j.InstanceID != id || j.ProjectID != project || j.OperationID != op {
		return j, errors.New("invalid member launch journal")
	}
	return j, nil
}
func memberLaunchPrerequisites(q querier, project, instance string) (storedProject, projectnetwork.AnchorStart, projectnetwork.MemberStorage, error) {
	p, err := loadProject(q, project)
	if err != nil {
		return p, projectnetwork.AnchorStart{}, projectnetwork.MemberStorage{}, err
	}
	a, err := loadProjectAnchorStart(q, project)
	if err != nil {
		return p, a, projectnetwork.MemberStorage{}, err
	}
	storage, err := loadProjectMemberStorage(q, instance)
	if err != nil {
		return p, a, storage, err
	}
	aop, err := loadOperation(q, a.OperationID)
	if err != nil {
		return p, a, storage, err
	}
	sop, err := loadOperation(q, storage.OperationID)
	if err != nil {
		return p, a, storage, err
	}
	if a.Validate(p.Plan) != nil || a.Phase != "observed" || aop.Status != "succeeded" || storage.Validate(p.Plan) != nil || storage.ProjectID != project || storage.ProjectRevision != p.Revision || storage.Phase != "observed" || sop.Status != "succeeded" {
		return p, a, storage, conflict("project_not_prepared", "complete this project's anchor and member storage operations before launch")
	}
	return p, a, storage, nil
}
func projectMemberLaunchJobs(q querier) ([]projectMemberLaunchJob, error) {
	rows, err := q.Query("SELECT instance_id FROM project_member_launches ORDER BY instance_id")
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
	var jobs []projectMemberLaunchJob
	for _, id := range ids {
		j, err := loadProjectMemberLaunch(q, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// Initial launches are admitted serially per project. Completed receipts supply
// historical peer ownership only; the driver must freshly inspect actual state.
func projectLaunchPeers(q querier, project, instance string) ([]projectnetwork.RuntimeMember, error) {
	jobs, err := projectMemberLaunchJobs(q)
	if err != nil {
		return nil, err
	}
	var peers []projectnetwork.RuntimeMember
	for _, j := range jobs {
		if j.ProjectID != project || j.InstanceID == instance {
			continue
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return nil, err
		}
		if j.Launch == nil || j.Launch.Phase != "running" || op.Status != "succeeded" {
			return nil, conflict("project_recovery_required", "another member launch is pending or unresolved; retain its original receipt")
		}
		peers = append(peers, projectnetwork.RuntimeMember{InstanceID: j.InstanceID, RuntimeID: j.Launch.Storage.RuntimeID, Container: j.Launch.Container})
	}
	return peers, nil
}
func validateProjectMemberLaunches(q querier) error {
	return validateSnapshot(q, "validateProjectMemberLaunches", validateProjectMemberLaunchesUncached)
}

func validateProjectMemberLaunchesUncached(q querier) error {
	if _, err := projectReservations(q); err != nil {
		return err
	}
	for _, validate := range []func(querier) error{validateProjectJobs, validateProjectAnchorStarts, validateProjectMemberStorage} {
		if err := validate(q); err != nil {
			return err
		}
	}

	jobs, err := projectMemberLaunchJobs(q)
	if err != nil {
		return err
	}
	byInstance := map[string]projectMemberLaunchJob{}
	for _, j := range jobs {
		byInstance[j.InstanceID] = j
	}
	for _, j := range jobs {
		p, a, storage, err := memberLaunchPrerequisites(q, j.ProjectID, j.InstanceID)
		if err != nil {
			return err
		}
		op, err := loadOperation(q, j.OperationID)
		if err != nil {
			return err
		}
		if op.Kind != "launch-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID || op.ConfigRevision != p.Revision || op.Actor == "" {
			return errors.New("member launch receipt ownership mismatch")
		}
		if op.Status != "queued" && op.Status != "running" && op.Status != "interrupted" && op.Status != "succeeded" {
			return errors.New("invalid member launch status")
		}
		var mirror []byte
		if err = q.QueryRow("SELECT runtime FROM operations WHERE id=?", op.ID).Scan(&mirror); err != nil {
			return err
		}
		if j.Launch == nil {
			if op.Status != "queued" && op.Status != "interrupted" || len(mirror) != 0 {
				return errors.New("member launch receipt lacks its bound journal")
			}
			continue
		}
		s := j.Launch
		if s.Validate(p.Plan) != nil || s.OperationID != op.ID || !reflect.DeepEqual(s.Anchor, a) || s.Storage != storage || op.Status == "queued" || (op.Status == "succeeded") != (s.Phase == "running") {
			return errors.New("member launch differs from original receipts or status")
		}
		body, _ := json.Marshal(s)
		if string(body) != string(mirror) {
			return errors.New("member launch journal and receipt disagree")
		}
		for _, peer := range s.Peers {
			previous, ok := byInstance[peer.InstanceID]
			if !ok || previous.ProjectID != j.ProjectID || previous.Launch == nil || previous.Launch.Phase != "running" || peer.RuntimeID != previous.Launch.Storage.RuntimeID || peer.Container != previous.Launch.Container {
				return errors.New("member launch lost its recorded peer ownership")
			}
		}
	}
	var orphans int
	if err = q.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.kind')='launch-project-member' AND id NOT IN (SELECT operation_id FROM project_member_launches)").Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return errors.New("member launch receipt is missing its journal")
	}
	return nil
}
func saveProjectMemberLaunch(tx *sql.Tx, j projectMemberLaunchJob) error {
	body, err := json.Marshal(j)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE project_member_launches SET body=? WHERE instance_id=? AND operation_id=?", body, j.InstanceID, j.OperationID); err != nil {
		return err
	}
	var mirror []byte
	if j.Launch != nil {
		mirror, err = json.Marshal(j.Launch)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", mirror, j.OperationID)
	return err
}
func (s *store) acceptProjectMemberLaunch(actor, project, instance string, req projectMemberLaunchRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("project, member, actor, request and revision are required")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 projectMemberLaunchRequest
	}{"launch-project-member", project, instance, req})
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, hit, err := replay(tx, actor, req.RequestID, string(fingerprint)); hit || err != nil {
		return op, hit, err
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching member launch receipt exists")
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
	for _, validate := range []func(querier) error{validateProjectJobs, validateProjectAnchorStarts, validateProjectMemberStorage, validateProjectMemberLaunches} {
		if err = validate(tx); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	if err = projectMemberAdmission(tx, project, instance); err != nil {
		return protocol.Operation{}, false, err
	}
	p, _, _, err := memberLaunchPrerequisites(tx, project, instance)
	if errors.Is(err, sql.ErrNoRows) {
		return protocol.Operation{}, false, conflict("project_not_prepared", "complete this project anchor and member storage before launch")
	}
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, conflict("revision_conflict", "project changed; review its current revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return protocol.Operation{}, false, bad("member launch must match the reviewed project plan")
	}
	if _, err = loadProjectMemberLaunch(tx, instance); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return protocol.Operation{}, false, err
		}
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original member launch receipt before recovery")
	}
	if _, err = projectLaunchPeers(tx, project, instance); err != nil {
		return protocol.Operation{}, false, err
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, ProjectID: project, InstanceID: instance, RequestID: req.RequestID, Actor: actor, Kind: "launch-project-member", ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if err = insertOperation(tx, op, string(fingerprint)); err != nil {
		return protocol.Operation{}, false, err
	}
	j := projectMemberLaunchJob{ProjectID: project, InstanceID: instance, OperationID: op.ID}
	body, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO project_member_launches(instance_id,project_id,operation_id,body) VALUES(?,?,?,?)", instance, project, op.ID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}
func (s *store) beginProjectMemberLaunch(instance, opID string, initial projectnetwork.MemberLaunch) (projectnetwork.MemberLaunch, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return initial, err
	}
	defer tx.Rollback()
	if err = validateProjectMemberLaunches(tx); err != nil {
		return initial, err
	}
	j, err := loadProjectMemberLaunch(tx, instance)
	if err != nil {
		return initial, err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return initial, err
	}
	if j.OperationID != opID || op.Status != "queued" || j.Launch != nil {
		return initial, conflict("stale_project_operation", "operation no longer owns initial member launch")
	}
	p, a, storage, err := memberLaunchPrerequisites(tx, j.ProjectID, instance)
	if err != nil {
		return initial, err
	}
	peers, err := projectLaunchPeers(tx, j.ProjectID, instance)
	if err != nil {
		return initial, err
	}
	expected, err := projectnetwork.NewMemberLaunch(p.Plan, a, storage, opID, initial.Container.ContractDigest, peers)
	if err != nil || !reflect.DeepEqual(initial, expected) {
		return initial, errors.New("member launch selection changed before dispatch")
	}
	j.Launch = &expected
	op.Status = "running"
	if err = saveProjectMemberLaunch(tx, j); err != nil {
		return initial, err
	}
	if err = saveOperation(tx, op); err != nil {
		return initial, err
	}
	return expected, tx.Commit()
}
func (s *store) advanceProjectMemberLaunch(instance, opID, expected, phase string, result projectnetwork.MemberLaunchResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = validateProjectMemberLaunches(tx); err != nil {
		return err
	}
	j, err := loadProjectMemberLaunch(tx, instance)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.OperationID != opID || op.Status != "running" || j.Launch == nil {
		return conflict("stale_project_operation", "operation no longer owns member launch")
	}
	p, err := loadProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	next, err := j.Launch.Advance(p.Plan, expected, phase, result)
	if err != nil {
		return err
	}
	j.Launch = &next
	if err = saveProjectMemberLaunch(tx, j); err != nil {
		return err
	}
	if phase == "running" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "member container running in its recorded namespace; ship readiness is not yet established"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func interruptProjectMemberLaunch(tx *sql.Tx, instance, opID string) error {
	j, err := loadProjectMemberLaunch(tx, instance)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if j.OperationID != opID || op.Kind != "launch-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != instance {
		return errors.New("member launch interruption ownership mismatch")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "member launch did not finish; retain original receipt, workspace, container and any pier data before recovery"
	return saveOperation(tx, op)
}
func reconcileProjectMemberLaunches(tx *sql.Tx) error {
	if err := validateProjectMemberLaunches(tx); err != nil {
		return err
	}
	jobs, err := projectMemberLaunchJobs(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptProjectMemberLaunch(tx, j.InstanceID, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptMemberLaunch(instance, opID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = interruptProjectMemberLaunch(tx, instance, opID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) runProjectMemberLaunch(ctx context.Context, instance, opID string, initial projectnetwork.MemberLaunch, d projectnetwork.MemberLaunchDriver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil {
		return errors.New("member launch driver required")
	}
	state, err := s.beginProjectMemberLaunch(instance, opID, initial)
	if err != nil {
		return err
	}
	p, err := loadProject(s.db, state.Storage.ProjectID)
	if err == nil {
		_, err = projectnetwork.ExecuteMemberLaunch(ctx, p.Plan, state, d, func(expected, phase string, r projectnetwork.MemberLaunchResult) error {
			return s.advanceProjectMemberLaunch(instance, opID, expected, phase, r)
		}, time.Now)
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptMemberLaunch(instance, opID))
}
