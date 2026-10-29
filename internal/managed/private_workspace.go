package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateWorkspaceKind = "prepare-private-project-workspace"

// Internal admission only. Parent identity is obtained by the backend, never
// accepted as public client input or substituted while replaying a saved receipt.
type privateWorkspaceRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision uint64 `json:"expected_revision"`
	ReviewDigest     string `json:"review_digest"`
	ReplayOnly       bool   `json:"-"`
}

func initializePrivateWorkspaces(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_project_workspaces(project_id TEXT PRIMARY KEY REFERENCES private_projects(id),operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id),body BLOB NOT NULL);`)
	return err
}
func loadPrivateProject(q querier, id string) (storedPrivateProject, error) {
	var p storedPrivateProject
	var raw []byte
	err := q.QueryRow("SELECT body FROM private_projects WHERE id=?", id).Scan(&raw)
	if err != nil {
		return p, err
	}
	if json.Unmarshal(raw, &p) != nil || p.Binding.ProjectID != id {
		return p, errors.New("invalid private project")
	}
	canonical, _ := json.Marshal(p)
	if !bytes.Equal(raw, canonical) {
		return p, errors.New("invalid private project encoding")
	}
	return p, nil
}
func privateWorkspaceFingerprint(project string, req privateWorkspaceRequest) string {
	raw, _ := json.Marshal(struct {
		Kind, Project string
		Request       privateWorkspaceRequest
	}{privateWorkspaceKind, project, req})
	return string(raw)
}
func loadPrivateWorkspace(q querier, project string) (privatechain.Workspace, error) {
	var w privatechain.Workspace
	var op string
	var raw []byte
	err := q.QueryRow("SELECT operation_id,body FROM private_project_workspaces WHERE project_id=?", project).Scan(&op, &raw)
	if err != nil {
		return w, err
	}
	if json.Unmarshal(raw, &w) != nil || w.Binding.ProjectID != project || w.OperationID != op {
		return w, errors.New("invalid private workspace journal")
	}
	canonical, _ := json.Marshal(w)
	if !bytes.Equal(raw, canonical) {
		return w, errors.New("invalid private workspace encoding")
	}
	return w, nil
}
func privateWorkspaceClaims(q querier) ([]privatechain.Workspace, error) {
	rows, err := q.Query("SELECT project_id FROM private_project_workspaces ORDER BY project_id")
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
	var out []privatechain.Workspace
	for _, id := range ids {
		w, err := loadPrivateWorkspace(q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

// Called inside the shared reservations validator. This must not recurse through
// projectReservations or inspect the filesystem. It validates the saved contract.
func validatePrivateWorkspaces(q querier) error {
	return validateSnapshot(q, "validatePrivateWorkspaces", validatePrivateWorkspacesUncached)
}

func validatePrivateWorkspacesUncached(q querier) error {
	fail := func() error { return errors.New("private workspace journal, ownership or operation receipt differs") }
	claims, err := privateWorkspaceClaims(q)
	if err != nil {
		return err
	}
	var parent pierstore.DirectoryIdentity
	var owned []pierstore.DirectoryIdentity
	for _, w := range claims {
		p, err := loadPrivateProject(q, w.Binding.ProjectID)
		if err != nil {
			return err
		}
		if w.Validate(p.Plan) != nil || w.Binding != p.Binding || w.Revision != p.Revision || w.OperationID == p.RegistrationOperationID {
			return fail()
		}
		if parent == (pierstore.DirectoryIdentity{}) {
			parent = w.Parent
		} else if !parent.SameObject(w.Parent) {
			return fail()
		}
		objects := []pierstore.DirectoryIdentity{}
		if w.Root != (pierstore.DirectoryIdentity{}) {
			objects = append(objects, w.Root)
		}
		if w.Observation != nil {
			objects = append(objects, w.Observation.PiersRoot)
			objects = append(objects, w.Observation.Mounts...)
		}
		for _, id := range objects {
			for _, previous := range owned {
				if id.SameObject(previous) {
					return fail()
				}
			}
			owned = append(owned, id)
		}
		var actor, request, fingerprint string
		var body, mirror []byte
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", w.OperationID).Scan(&actor, &request, &fingerprint, &body, &mirror); err != nil {
			return err
		}
		var op protocol.Operation
		if json.Unmarshal(body, &op) != nil || op.ID != w.OperationID || op.ProjectID != w.Binding.ProjectID || op.InstanceID != "" || op.Kind != privateWorkspaceKind || op.Actor != actor || actor == "" || op.RequestID != request || !protocol.ValidID(request) || op.ConfigRevision != w.Revision || op.CreatedAt.IsZero() || op.CreatedAt.Before(p.CreatedAt) {
			return fail()
		}
		canonical, _ := json.Marshal(op)
		runtime, _ := json.Marshal(w)
		if !bytes.Equal(body, canonical) || !bytes.Equal(mirror, runtime) || fingerprint != privateWorkspaceFingerprint(w.Binding.ProjectID, privateWorkspaceRequest{RequestID: request, ExpectedRevision: w.Revision, ReviewDigest: w.PlanDigest}) {
			return fail()
		}
		switch op.Status {
		case "queued":
			if w.Phase != "planned" || op.FinishedAt != nil {
				return fail()
			}
		case "running":
			if w.Phase == "observed" || op.FinishedAt != nil {
				return fail()
			}
		case "interrupted":
			if w.Phase == "observed" || op.FinishedAt == nil {
				return fail()
			}
		case "succeeded":
			if w.Phase != "observed" || op.FinishedAt == nil {
				return fail()
			}
		default:
			return fail()
		}
		if op.FinishedAt != nil && op.FinishedAt.Before(op.CreatedAt) {
			return fail()
		}
	}
	var orphan int
	err = q.QueryRow(`SELECT count(*) FROM operations WHERE (json_extract(body,'$.kind')=? OR fingerprint LIKE ?) AND id NOT IN (SELECT operation_id FROM private_project_workspaces)`, privateWorkspaceKind, `{"Kind":"`+privateWorkspaceKind+`",%`).Scan(&orphan)
	if err != nil {
		return err
	}
	if orphan != 0 {
		return fail()
	}
	return nil
}

// Review fresh intent before a backend can provision shared grouping metadata.
// The accepting transaction repeats this check before saving the workspace.
func reviewPrivateWorkspace(q querier, project string, req privateWorkspaceRequest) (storedPrivateProject, error) {
	p, err := loadPrivateProject(q, project)
	if err != nil {
		return storedPrivateProject{}, err
	}
	if p.Revision != req.ExpectedRevision {
		return storedPrivateProject{}, conflict("revision_conflict", "review the current private project revision")
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest {
		return storedPrivateProject{}, bad("workspace must match the reviewed private project")
	}
	if _, err = loadPrivateWorkspace(q, project); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return storedPrivateProject{}, err
		}
		return storedPrivateProject{}, conflict("project_recovery_required", "retain the original private workspace operation and objects before recovery")
	}
	return p, nil
}

func (s *store) acceptPrivateWorkspace(actor, project string, req privateWorkspaceRequest, parent pierstore.DirectoryIdentity) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("private workspace requires actor, project, request and reviewed revision")
	}
	fingerprint := privateWorkspaceFingerprint(project, req)
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if op, hit, e := replay(tx, actor, req.RequestID, fingerprint); hit || e != nil {
		return op, hit, e
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private workspace receipt exists for this actor")
	}
	p, err := reviewPrivateWorkspace(tx, project, req)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	random, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + random, ProjectID: project, RequestID: req.RequestID, Actor: actor, Kind: privateWorkspaceKind, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	w, err := privatechain.NewWorkspace(p.Plan, p.Binding, p.Revision, op.ID, parent)
	if err != nil {
		return protocol.Operation{}, false, bad("private workspace parent or plan is invalid")
	}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	body, _ := json.Marshal(w)
	if _, err = tx.Exec("INSERT INTO private_project_workspaces(project_id,operation_id,body) VALUES(?,?,?)", project, op.ID, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, op.ID); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = projectReservations(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

func (s *store) beginPrivateWorkspace(project, operation string) (privatechain.Workspace, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return privatechain.Workspace{}, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return privatechain.Workspace{}, err
	}
	w, err := loadPrivateWorkspace(tx, project)
	if err != nil {
		return w, err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return w, err
	}
	if w.OperationID != operation || op.Status != "queued" || w.Phase != "planned" {
		return w, conflict("stale_project_operation", "private workspace is no longer freshly queued")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return w, err
	}
	return w, tx.Commit()
}

func (s *store) advancePrivateWorkspace(project, operation, expected string, next privatechain.Workspace) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	w, err := loadPrivateWorkspace(tx, project)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(tx, project)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return err
	}
	if w.OperationID != operation || op.Status != "running" {
		return conflict("stale_project_operation", "private workspace operation changed")
	}
	want, err := w.Advance(p.Plan, expected, next.Phase, next.Root, next.Observation)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, next) {
		return errors.New("private workspace callback changed immutable ownership")
	}
	if next.Phase == "observed" && next.Observation != nil {
		now := time.Now().UTC()
		at := next.Observation.ObservedAt
		if now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
			return errors.New("private workspace observation expired before commit")
		}
	}
	body, _ := json.Marshal(next)
	if _, err = tx.Exec("UPDATE private_project_workspaces SET body=? WHERE project_id=? AND operation_id=?", body, project, operation); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", body, operation); err != nil {
		return err
	}
	if next.Phase == "observed" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "private project workspace prepared; no chain, key or ship was created"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func interruptPrivateWorkspace(tx *sql.Tx, project, operation string) error {
	w, err := loadPrivateWorkspace(tx, project)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return err
	}
	if w.OperationID != operation || op.ProjectID != project || op.Kind != privateWorkspaceKind {
		return errors.New("private workspace interruption ownership differs")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "private workspace outcome requires inspection; retain the original operation and partial directories"
	return saveOperation(tx, op)
}
func reconcilePrivateWorkspaces(tx *sql.Tx) error {
	if err := validatePrivateWorkspaces(tx); err != nil {
		return err
	}
	claims, err := privateWorkspaceClaims(tx)
	if err != nil {
		return err
	}
	for _, w := range claims {
		if err = interruptPrivateWorkspace(tx, w.Binding.ProjectID, w.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptPrivateWorkspace(project, operation string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	if err = interruptPrivateWorkspace(tx, project, operation); err != nil {
		return err
	}
	return tx.Commit()
}

// Private source entry only; no public route or startup queue invokes this. The
// worker must hold the common runner mutation/update fence and close its driver.
func (s *store) runPrivateWorkspace(ctx context.Context, project, operation string, driver privatechain.WorkspaceDriver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if driver == nil {
		return errors.New("private workspace driver is required")
	}
	w, err := s.beginPrivateWorkspace(project, operation)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(s.db, project)
	if err == nil {
		_, err = privatechain.ExecuteWorkspace(ctx, p.Plan, w, driver, func(expected string, next privatechain.Workspace) error {
			return s.advancePrivateWorkspace(project, operation, expected, next)
		}, time.Now)
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptPrivateWorkspace(project, operation))
}
