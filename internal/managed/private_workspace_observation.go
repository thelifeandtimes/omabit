package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateWorkspaceObservationKind = "observe-private-project-workspace"

type privateWorkspaceObservationRequest struct {
	RequestID         string `json:"request_id"`
	ExpectedRevision  uint64 `json:"expected_revision"`
	ReviewDigest      string `json:"review_digest"`
	SourceOperationID string `json:"source_operation_id"`
	SourceDigest      string `json:"source_digest"`
	ReplayOnly        bool   `json:"-"`
}

type privateWorkspaceObservation struct {
	Version           int                     `json:"version"`
	ProjectID         string                  `json:"project_id"`
	OperationID       string                  `json:"operation_id"`
	SourceOperationID string                  `json:"source_operation_id"`
	SourceDigest      string                  `json:"source_digest"`
	Phase             string                  `json:"phase"`
	Result            *privatechain.Workspace `json:"result,omitempty"`
}

func privateWorkspaceSourceDigest(w privatechain.Workspace) string {
	raw, _ := json.Marshal(w)
	sum := sha256.Sum256(append([]byte("omabit-private-workspace-observation-source-v1\x00"), raw...))
	return hex.EncodeToString(sum[:])
}
func privateWorkspaceObservationFingerprint(project string, req privateWorkspaceObservationRequest) string {
	raw, _ := json.Marshal(struct {
		Kind, Project string
		Request       privateWorkspaceObservationRequest
	}{privateWorkspaceObservationKind, project, req})
	return string(raw)
}
func initializePrivateWorkspaceObservations(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_workspace_observations(operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES private_project_workspaces(project_id),source_operation_id TEXT NOT NULL REFERENCES operations(id),body BLOB NOT NULL);`)
	return err
}
func loadPrivateWorkspaceObservation(q querier, id string) (privateWorkspaceObservation, error) {
	var j privateWorkspaceObservation
	var project, source string
	var raw []byte
	err := q.QueryRow("SELECT project_id,source_operation_id,body FROM private_workspace_observations WHERE operation_id=?", id).Scan(&project, &source, &raw)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(raw, &j) != nil || j.ProjectID != project || j.OperationID != id || j.SourceOperationID != source {
		return j, errors.New("invalid private workspace observation ownership")
	}
	canonical, _ := json.Marshal(j)
	if !bytes.Equal(raw, canonical) {
		return j, errors.New("invalid private workspace observation encoding")
	}
	return j, nil
}
func privateWorkspaceObservations(q querier) ([]privateWorkspaceObservation, error) {
	rows, err := q.Query("SELECT operation_id FROM private_workspace_observations ORDER BY operation_id")
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
	var out []privateWorkspaceObservation
	for _, id := range ids {
		j, err := loadPrivateWorkspaceObservation(q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

func privateWorkspaceObservedResult(plan privatechain.Plan, before, after privatechain.Workspace) error {
	if before.Validate(plan) != nil || (before.Phase != "layout-dispatched" && before.Phase != "layout-created") || after.Phase != "observed" || after.Observation == nil || after.Validate(plan) != nil {
		return errors.New("workspace result is not a complete observation of dispatched layout")
	}
	want := before.Clone()
	want.Phase = "observed"
	o := after.Clone()
	want.Observation = o.Observation
	if !reflect.DeepEqual(want, after) {
		return errors.New("workspace observation changed original ownership")
	}
	return nil
}

func validatePrivateWorkspaceObservations(q querier) error {
	return validateSnapshot(q, "validatePrivateWorkspaceObservations", validatePrivateWorkspaceObservationsUncached)
}

func validatePrivateWorkspaceObservationsUncached(q querier) error {
	if err := validatePrivateWorkspaces(q); err != nil {
		return err
	}
	fail := func() error { return errors.New("private workspace observation, source or operation receipt differs") }
	jobs, err := privateWorkspaceObservations(q)
	if err != nil {
		return err
	}
	if _, err = privateWorkspaceOwnedObjects(q); err != nil {
		return err
	}
	active, complete := map[string]bool{}, map[string]bool{}
	for _, j := range jobs {
		if j.Version != 1 || !protocol.ValidID(j.OperationID) || j.OperationID == j.SourceOperationID {
			return fail()
		}
		w, err := loadPrivateWorkspace(q, j.ProjectID)
		if err != nil {
			return err
		}
		p, err := loadPrivateProject(q, j.ProjectID)
		if err != nil {
			return err
		}
		source, err := loadOperation(q, j.SourceOperationID)
		if err != nil {
			return err
		}
		if w.OperationID != j.SourceOperationID || j.SourceDigest != privateWorkspaceSourceDigest(w) || w.Validate(p.Plan) != nil || (w.Phase != "layout-dispatched" && w.Phase != "layout-created") || source.Status != "interrupted" || source.FinishedAt == nil {
			return fail()
		}
		var actor, request, fingerprint string
		var body, mirror []byte
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", j.OperationID).Scan(&actor, &request, &fingerprint, &body, &mirror); err != nil {
			return err
		}
		var op protocol.Operation
		if json.Unmarshal(body, &op) != nil || op.ID != j.OperationID || op.Kind != privateWorkspaceObservationKind || op.ProjectID != j.ProjectID || op.InstanceID != "" || actor == "" || op.Actor != actor || op.RequestID != request || !protocol.ValidID(request) || op.ConfigRevision != w.Revision || op.CreatedAt.IsZero() || op.CreatedAt.Before(*source.FinishedAt) {
			return fail()
		}
		canonical, _ := json.Marshal(op)
		runtime, _ := json.Marshal(j)
		req := privateWorkspaceObservationRequest{RequestID: request, ExpectedRevision: w.Revision, ReviewDigest: w.PlanDigest, SourceOperationID: j.SourceOperationID, SourceDigest: j.SourceDigest}
		if !bytes.Equal(body, canonical) || !bytes.Equal(mirror, runtime) || fingerprint != privateWorkspaceObservationFingerprint(j.ProjectID, req) {
			return fail()
		}
		switch j.Phase {
		case "planned":
			if j.Result != nil || (op.Status != "queued" && op.Status != "interrupted") {
				return fail()
			}
		case "observing":
			if j.Result != nil || (op.Status != "running" && op.Status != "interrupted") {
				return fail()
			}
		case "observed":
			if op.Status != "succeeded" || j.Result == nil || privateWorkspaceObservedResult(p.Plan, w, *j.Result) != nil {
				return fail()
			}
			if complete[j.ProjectID] {
				return fail()
			}
			complete[j.ProjectID] = true
		default:
			return fail()
		}
		if op.Status == "queued" || op.Status == "running" {
			if op.FinishedAt != nil || active[j.ProjectID] {
				return fail()
			}
			active[j.ProjectID] = true
		} else {
			if op.FinishedAt == nil || op.FinishedAt.Before(op.CreatedAt) {
				return fail()
			}
		}
	}
	for project := range active {
		if complete[project] {
			return fail()
		}
	}
	var orphan int
	err = q.QueryRow(`SELECT count(*) FROM operations WHERE (json_extract(body,'$.kind')=? OR fingerprint LIKE ?) AND id NOT IN (SELECT operation_id FROM private_workspace_observations)`, privateWorkspaceObservationKind, `{"Kind":"`+privateWorkspaceObservationKind+`",%`).Scan(&orphan)
	if err != nil {
		return err
	}
	if orphan != 0 {
		return fail()
	}
	return nil
}

// Directory identity, unlike a shared foundation or ship name, must remain
// exclusive to one project. A recovered result supplements its original root
// claim; it must not claim another project's pier parents as its own.
func claimPrivateObjects(owned map[pierstore.DirectoryIdentity]string, project string, objects ...pierstore.DirectoryIdentity) error {
	for _, object := range objects {
		if previous, exists := owned[object]; exists && previous != project {
			return errors.New("private projects claim the same filesystem object")
		}
		owned[object] = project
	}
	return nil
}

func privateWorkspaceOwnedObjects(q querier) (map[pierstore.DirectoryIdentity]string, error) {
	owned := make(map[pierstore.DirectoryIdentity]string)
	workspaces, err := privateWorkspaceClaims(q)
	if err != nil {
		return nil, err
	}
	add := func(w privatechain.Workspace) error {
		if w.Root != (pierstore.DirectoryIdentity{}) {
			if err := claimPrivateObjects(owned, w.Binding.ProjectID, w.Root); err != nil {
				return err
			}
		}
		if w.Observation != nil {
			objects := append([]pierstore.DirectoryIdentity{w.Observation.PiersRoot}, w.Observation.Mounts...)
			return claimPrivateObjects(owned, w.Binding.ProjectID, objects...)
		}
		return nil
	}
	for _, w := range workspaces {
		if err = add(w); err != nil {
			return nil, err
		}
	}
	jobs, err := privateWorkspaceObservations(q)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if j.Result != nil {
			if err = add(*j.Result); err != nil {
				return nil, err
			}
		}
	}
	return owned, nil
}

func privateWorkspaceObservationState(q querier) error {
	_, err := projectReservations(q)
	return err
}
func savePrivateWorkspaceObservation(tx *sql.Tx, j privateWorkspaceObservation, op protocol.Operation) error {
	raw, _ := json.Marshal(j)
	result, err := tx.Exec("UPDATE private_workspace_observations SET body=? WHERE operation_id=? AND project_id=? AND source_operation_id=?", raw, j.OperationID, j.ProjectID, j.SourceOperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("private workspace observation lost ownership")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID); err != nil {
		return err
	}
	return saveOperation(tx, op)
}

func (s *store) acceptPrivateWorkspaceObservation(actor, project string, req privateWorkspaceObservationRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.SourceOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("workspace observation requires an actor and reviewed source operation")
	}
	fingerprint := privateWorkspaceObservationFingerprint(project, req)
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if err = privateWorkspaceObservationState(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if op, hit, e := replay(tx, actor, req.RequestID, fingerprint); hit || e != nil {
		return op, hit, e
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private workspace observation exists for this actor")
	}
	w, err := loadPrivateWorkspace(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	source, err := loadOperation(tx, w.OperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if req.SourceOperationID != w.OperationID || req.SourceDigest != privateWorkspaceSourceDigest(w) || req.ReviewDigest != w.PlanDigest || req.ExpectedRevision != w.Revision {
		return protocol.Operation{}, false, conflict("revision_conflict", "review the unchanged original workspace operation before observation")
	}
	if source.Status != "interrupted" || (w.Phase != "layout-dispatched" && w.Phase != "layout-created") {
		return protocol.Operation{}, false, conflict("project_recovery_required", "observation requires an interrupted layout with a recorded root")
	}
	jobs, err := privateWorkspaceObservations(tx)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, j := range jobs {
		if j.ProjectID == project {
			op, e := loadOperation(tx, j.OperationID)
			if e != nil {
				return protocol.Operation{}, false, e
			}
			if op.Status == "queued" || op.Status == "running" || op.Status == "succeeded" {
				return protocol.Operation{}, false, conflict("project_operation_pending", "retain the existing workspace observation receipt")
			}
		}
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, Kind: privateWorkspaceObservationKind, ProjectID: project, RequestID: req.RequestID, Actor: actor, ConfigRevision: w.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	j := privateWorkspaceObservation{Version: 1, ProjectID: project, OperationID: op.ID, SourceOperationID: w.OperationID, SourceDigest: req.SourceDigest, Phase: "planned"}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("INSERT INTO private_workspace_observations(operation_id,project_id,source_operation_id,body) VALUES(?,?,?,?)", op.ID, project, w.OperationID, []byte("{}")); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = savePrivateWorkspaceObservation(tx, j, op); err != nil {
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

func (s *store) beginPrivateWorkspaceObservation(id string) (privateWorkspaceObservation, error) {
	var j privateWorkspaceObservation
	tx, err := s.db.Begin()
	if err != nil {
		return j, err
	}
	defer tx.Rollback()
	if err = privateWorkspaceObservationState(tx); err != nil {
		return j, err
	}
	j, err = loadPrivateWorkspaceObservation(tx, id)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return j, err
	}
	if j.Phase != "planned" || op.Status != "queued" {
		return j, conflict("stale_project_operation", "private workspace observation is no longer freshly queued")
	}
	j.Phase = "observing"
	op.Status = "running"
	if err = savePrivateWorkspaceObservation(tx, j, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}

func (s *store) completePrivateWorkspaceObservation(id string, result privatechain.Workspace) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateWorkspaceObservationState(tx); err != nil {
		return err
	}
	j, err := loadPrivateWorkspaceObservation(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	w, err := loadPrivateWorkspace(tx, j.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" {
		return conflict("stale_project_operation", "private workspace observation changed")
	}
	if err = privateWorkspaceObservedResult(p.Plan, w, result); err != nil {
		return err
	}
	now := time.Now().UTC()
	at := result.Observation.ObservedAt
	if now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
		return errors.New("private workspace observation expired before commit")
	}
	result = result.Clone()
	j.Result = &result
	j.Phase = "observed"
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "retained private workspace observed; original interrupted operation preserved; no directories or runtime created"
	if err = savePrivateWorkspaceObservation(tx, j, op); err != nil {
		return err
	}
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func interruptPrivateWorkspaceObservation(tx *sql.Tx, id string) error {
	j, err := loadPrivateWorkspaceObservation(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.ProjectID != j.ProjectID || op.Kind != privateWorkspaceObservationKind {
		return errors.New("workspace observation interruption ownership differs")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "workspace observation interrupted; retain original directories and request a new inspection"
	return saveOperation(tx, op)
}
func reconcilePrivateWorkspaceObservations(tx *sql.Tx) error {
	if err := validatePrivateWorkspaceObservations(tx); err != nil {
		return err
	}
	jobs, err := privateWorkspaceObservations(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptPrivateWorkspaceObservation(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptPrivateWorkspaceObservation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateWorkspaceObservationState(tx); err != nil {
		return err
	}
	if err = interruptPrivateWorkspaceObservation(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// The observer has no directory-creation or runtime methods. Original workspace
// journal/operation bytes remain unchanged, including after observation success.
func (s *store) runPrivateWorkspaceObservation(ctx context.Context, id string, observer privatechain.WorkspaceObserver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if observer == nil {
		return errors.New("private workspace observer required")
	}
	j, err := s.beginPrivateWorkspaceObservation(id)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(s.db, j.ProjectID)
	if err == nil {
		w, e := loadPrivateWorkspace(s.db, j.ProjectID)
		err = e
		if err == nil {
			_, err = privatechain.ObserveWorkspace(ctx, p.Plan, w, observer, func(expected string, result privatechain.Workspace) error {
				if expected != w.Phase {
					return errors.New("workspace observation source changed")
				}
				return s.completePrivateWorkspaceObservation(id, result)
			}, time.Now)
		}
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptPrivateWorkspaceObservation(id))
}
