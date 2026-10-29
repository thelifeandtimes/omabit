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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateGatewayObservationKind = "observe-private-project-gateway"

type privateGatewayObservationRequest privateWorkspaceObservationRequest
type privateGatewayObservation struct {
	Version           int                   `json:"version"`
	ProjectID         string                `json:"project_id"`
	OperationID       string                `json:"operation_id"`
	SourceOperationID string                `json:"source_operation_id"`
	SourceDigest      string                `json:"source_digest"`
	Phase             string                `json:"phase"`
	Result            *privatechain.Gateway `json:"result,omitempty"`
}

func privateGatewaySourceDigest(g privatechain.Gateway) string {
	raw, _ := json.Marshal(g)
	sum := sha256.Sum256(append([]byte("omabit-private-gateway-observation-source-v1\x00"), raw...))
	return hex.EncodeToString(sum[:])
}
func privateGatewayObservationFingerprint(project string, req privateGatewayObservationRequest) string {
	raw, _ := json.Marshal(struct {
		Kind, Project string
		Request       privateGatewayObservationRequest
	}{privateGatewayObservationKind, project, req})
	return string(raw)
}
func initializePrivateGatewayObservations(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_gateway_observations(operation_id TEXT PRIMARY KEY REFERENCES operations(id),project_id TEXT NOT NULL REFERENCES private_project_gateways(project_id),source_operation_id TEXT NOT NULL REFERENCES operations(id),body BLOB NOT NULL);`)
	return err
}
func loadPrivateGatewayObservation(q querier, id string) (privateGatewayObservation, error) {
	var j privateGatewayObservation
	var project, source string
	var raw []byte
	err := q.QueryRow("SELECT project_id,source_operation_id,body FROM private_gateway_observations WHERE operation_id=?", id).Scan(&project, &source, &raw)
	if err != nil {
		return j, err
	}
	if json.Unmarshal(raw, &j) != nil || j.ProjectID != project || j.OperationID != id || j.SourceOperationID != source {
		return j, errors.New("invalid private gateway observation ownership")
	}
	canonical, _ := json.Marshal(j)
	if !bytes.Equal(raw, canonical) {
		return j, errors.New("invalid private gateway observation encoding")
	}
	return j, nil
}
func privateGatewayObservations(q querier) ([]privateGatewayObservation, error) {
	rows, err := q.Query("SELECT operation_id FROM private_gateway_observations ORDER BY operation_id")
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
	var out []privateGatewayObservation
	for _, id := range ids {
		j, err := loadPrivateGatewayObservation(q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

func privateGatewayObservedResult(plan privatechain.Plan, w privatechain.Workspace, before, after privatechain.Gateway) error {
	if before.Validate(plan, w) != nil || (before.Phase != "start-dispatched" && before.Phase != "started") || after.Phase != "ready" || after.Observation == nil || after.Validate(plan, w) != nil {
		return errors.New("gateway result is not complete readiness of a dispatched Start")
	}
	want := before.Clone()
	want.Phase = "ready"
	o := after.Clone()
	want.Observation = o.Observation
	if !reflect.DeepEqual(want, after) {
		return errors.New("gateway observation changed original selection or resources")
	}
	return nil
}

func validatePrivateGatewayObservations(q querier) error {
	return validateSnapshot(q, "validatePrivateGatewayObservations", validatePrivateGatewayObservationsUncached)
}

func validatePrivateGatewayObservationsUncached(q querier) error {
	if err := validatePrivateGateways(q); err != nil {
		return err
	}
	fail := func() error { return errors.New("private gateway observation, source or operation receipt differs") }
	jobs, err := privateGatewayObservations(q)
	if err != nil {
		return err
	}
	objects, err := privateWorkspaceOwnedObjects(q)
	if err != nil {
		return err
	}
	active, complete, keys := map[string]bool{}, map[string]bool{}, map[string]bool{}
	addKeys := func(g privatechain.Gateway) error {
		if err := claimPrivateObjects(objects, g.Selection.Binding.ProjectID, g.Observation.Chain, g.Observation.Secrets); err != nil {
			return err
		}
		for _, key := range g.Observation.PublicKeys {
			for _, value := range []string{key.Auth, key.Crypt} {
				if keys[value] {
					return fail()
				}
				keys[value] = true
			}
		}
		return nil
	}
	claims, err := privateGatewayClaims(q)
	if err != nil {
		return err
	}
	for _, g := range claims {
		if g.Phase == "ready" {
			if err = addKeys(g); err != nil {
				return err
			}
		}
	}
	for _, j := range jobs {
		if j.Version != 1 || !protocol.ValidID(j.OperationID) || j.OperationID == j.SourceOperationID {
			return fail()
		}
		g, err := loadPrivateGateway(q, j.ProjectID)
		if err != nil {
			return err
		}
		p, err := loadPrivateProject(q, j.ProjectID)
		if err != nil {
			return err
		}
		w, err := readyPrivateWorkspace(q, j.ProjectID)
		if err != nil {
			return err
		}
		source, err := loadOperation(q, j.SourceOperationID)
		if err != nil {
			return err
		}
		if g.Selection.OperationID != j.SourceOperationID || j.SourceDigest != privateGatewaySourceDigest(g) || (g.Phase != "start-dispatched" && g.Phase != "started") || source.Status != "interrupted" || source.FinishedAt == nil {
			return fail()
		}
		var actor, request, fingerprint string
		var body, mirror []byte
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", j.OperationID).Scan(&actor, &request, &fingerprint, &body, &mirror); err != nil {
			return err
		}
		var op protocol.Operation
		if json.Unmarshal(body, &op) != nil || op.ID != j.OperationID || op.Kind != privateGatewayObservationKind || op.ProjectID != j.ProjectID || op.InstanceID != "" || actor == "" || op.Actor != actor || op.RequestID != request || !protocol.ValidID(request) || op.ConfigRevision != g.Selection.Revision || op.CreatedAt.IsZero() || op.CreatedAt.Before(*source.FinishedAt) {
			return fail()
		}
		canonical, _ := json.Marshal(op)
		runtime, _ := json.Marshal(j)
		req := privateGatewayObservationRequest{RequestID: request, ExpectedRevision: g.Selection.Revision, ReviewDigest: g.Selection.PlanDigest, SourceOperationID: j.SourceOperationID, SourceDigest: j.SourceDigest}
		if !bytes.Equal(body, canonical) || !bytes.Equal(mirror, runtime) || fingerprint != privateGatewayObservationFingerprint(j.ProjectID, req) {
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
			if op.Status != "succeeded" || j.Result == nil || privateGatewayObservedResult(p.Plan, w, g, *j.Result) != nil || complete[j.ProjectID] {
				return fail()
			}
			complete[j.ProjectID] = true
			if err = addKeys(*j.Result); err != nil {
				return err
			}
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
	err = q.QueryRow(`SELECT count(*) FROM operations WHERE (json_extract(body,'$.kind')=? OR fingerprint LIKE ?) AND id NOT IN (SELECT operation_id FROM private_gateway_observations)`, privateGatewayObservationKind, `{"Kind":"`+privateGatewayObservationKind+`",%`).Scan(&orphan)
	if err != nil {
		return err
	}
	if orphan != 0 {
		return fail()
	}
	return nil
}

func savePrivateGatewayObservation(tx *sql.Tx, j privateGatewayObservation, op protocol.Operation) error {
	raw, _ := json.Marshal(j)
	result, err := tx.Exec("UPDATE private_gateway_observations SET body=? WHERE operation_id=? AND project_id=? AND source_operation_id=?", raw, j.OperationID, j.ProjectID, j.SourceOperationID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("private gateway observation lost ownership")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID); err != nil {
		return err
	}
	return saveOperation(tx, op)
}

func (s *store) acceptPrivateGatewayObservation(actor, project string, req privateGatewayObservationRequest) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || !protocol.ValidID(req.SourceOperationID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("gateway observation requires an actor and reviewed source operation")
	}
	fingerprint := privateGatewayObservationFingerprint(project, req)
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if op, hit, e := replay(tx, actor, req.RequestID, fingerprint); hit || e != nil {
		return op, hit, e
	}
	if req.ReplayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private gateway observation exists for this actor")
	}
	g, err := loadPrivateGateway(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	source, err := loadOperation(tx, g.Selection.OperationID)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if req.SourceOperationID != g.Selection.OperationID || req.SourceDigest != privateGatewaySourceDigest(g) || req.ReviewDigest != g.Selection.PlanDigest || req.ExpectedRevision != g.Selection.Revision {
		return protocol.Operation{}, false, conflict("revision_conflict", "review the unchanged original gateway operation before observation")
	}
	if source.Status != "interrupted" || (g.Phase != "start-dispatched" && g.Phase != "started") {
		return protocol.Operation{}, false, conflict("project_recovery_required", "observation requires a dispatched Start with recorded exact resources")
	}
	jobs, err := privateGatewayObservations(tx)
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
				return protocol.Operation{}, false, conflict("project_operation_pending", "retain the existing gateway observation receipt")
			}
		}
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, Kind: privateGatewayObservationKind, ProjectID: project, RequestID: req.RequestID, Actor: actor, ConfigRevision: g.Selection.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	j := privateGatewayObservation{Version: 1, ProjectID: project, OperationID: op.ID, SourceOperationID: g.Selection.OperationID, SourceDigest: req.SourceDigest, Phase: "planned"}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("INSERT INTO private_gateway_observations(operation_id,project_id,source_operation_id,body) VALUES(?,?,?,?)", op.ID, project, j.SourceOperationID, []byte("{}")); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = savePrivateGatewayObservation(tx, j, op); err != nil {
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

func (s *store) beginPrivateGatewayObservation(id string) (privateGatewayObservation, error) {
	var j privateGatewayObservation
	tx, err := s.db.Begin()
	if err != nil {
		return j, err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return j, err
	}
	j, err = loadPrivateGatewayObservation(tx, id)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return j, err
	}
	if j.Phase != "planned" || op.Status != "queued" {
		return j, conflict("stale_project_operation", "private gateway observation is no longer freshly queued")
	}
	j.Phase = "observing"
	op.Status = "running"
	if err = savePrivateGatewayObservation(tx, j, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}

func (s *store) completePrivateGatewayObservation(id string, result privatechain.Gateway) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	j, err := loadPrivateGatewayObservation(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	g, err := loadPrivateGateway(tx, j.ProjectID)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(tx, j.ProjectID)
	if err != nil {
		return err
	}
	w, err := readyPrivateWorkspace(tx, j.ProjectID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" {
		return conflict("stale_project_operation", "private gateway observation changed")
	}
	if err = privateGatewayObservedResult(p.Plan, w, g, result); err != nil {
		return err
	}
	now := time.Now().UTC()
	at := result.Observation.ObservedAt
	if now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
		return errors.New("private gateway observation expired before commit")
	}
	result = result.Clone()
	j.Result = &result
	j.Phase = "observed"
	op.Status = "succeeded"
	op.FinishedAt = &now
	op.Message = "retained private gateway readiness observed; original interrupted initialization preserved; no runtime started"
	if err = savePrivateGatewayObservation(tx, j, op); err != nil {
		return err
	}
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func interruptPrivateGatewayObservation(tx *sql.Tx, id string) error {
	j, err := loadPrivateGatewayObservation(tx, id)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, id)
	if err != nil {
		return err
	}
	if op.ProjectID != j.ProjectID || op.Kind != privateGatewayObservationKind {
		return errors.New("gateway observation interruption ownership differs")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "gateway observation interrupted; retain original resources and request a new inspection"
	return saveOperation(tx, op)
}
func reconcilePrivateGatewayObservations(tx *sql.Tx) error {
	if err := validatePrivateGatewayObservations(tx); err != nil {
		return err
	}
	jobs, err := privateGatewayObservations(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptPrivateGatewayObservation(tx, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptPrivateGatewayObservation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	if err = interruptPrivateGatewayObservation(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *store) runPrivateGatewayObservation(ctx context.Context, id string, observer privatechain.GatewayObserver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if observer == nil {
		return errors.New("private gateway observer required")
	}
	j, err := s.beginPrivateGatewayObservation(id)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(s.db, j.ProjectID)
	if err == nil {
		w, e := readyPrivateWorkspace(s.db, j.ProjectID)
		err = e
		if err == nil {
			g, e := loadPrivateGateway(s.db, j.ProjectID)
			err = e
			if err == nil {
				result, e := privatechain.ObserveGatewayReadiness(ctx, p.Plan, w, g, observer, time.Now)
				err = e
				if err == nil {
					err = s.completePrivateGatewayObservation(id, result)
				}
			}
		}
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptPrivateGatewayObservation(id))
}

// This is saved readiness, not a live observation or permission to launch a
// member. Runtime use must freshly verify these exact retained resources again.
func readyPrivateGateway(q querier, project string) (privatechain.Gateway, protocol.Operation, error) {
	if err := privateGatewayState(q); err != nil {
		return privatechain.Gateway{}, protocol.Operation{}, err
	}
	return readyPrivateGatewayEvidence(q, project)
}

// Reads only gateway ancestry, without recursively validating its dependents.
func readyPrivateGatewayEvidence(q querier, project string) (privatechain.Gateway, protocol.Operation, error) {
	var empty protocol.Operation
	if err := validatePrivateGatewayObservations(q); err != nil {
		return privatechain.Gateway{}, empty, err
	}
	g, err := loadPrivateGateway(q, project)
	if err != nil {
		return g, empty, err
	}
	op, err := loadOperation(q, g.Selection.OperationID)
	if err != nil {
		return g, empty, err
	}
	if g.Phase == "ready" && op.Status == "succeeded" {
		return g, op, nil
	}
	jobs, err := privateGatewayObservations(q)
	if err != nil {
		return g, empty, err
	}
	for _, j := range jobs {
		if j.ProjectID == project && j.Phase == "observed" && j.Result != nil {
			observed, err := loadOperation(q, j.OperationID)
			if err != nil {
				return g, empty, err
			}
			return j.Result.Clone(), observed, nil
		}
	}
	return g, empty, conflict("project_recovery_required", "complete an attributed observation of the retained gateway before using its readiness")
}
