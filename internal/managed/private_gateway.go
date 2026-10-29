package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateGatewayKind = "initialize-private-project-gateway"

type privateGatewayRequest struct {
	RequestID            string `json:"request_id"`
	ExpectedRevision     uint64 `json:"expected_revision"`
	ReviewDigest         string `json:"review_digest"`
	WorkspaceOperationID string `json:"workspace_operation_id"`
	WorkspaceDigest      string `json:"workspace_digest"`
	InputManifestSHA256  string `json:"input_manifest_sha256"`
	ReplayOnly           bool   `json:"-"`
}

func initializePrivateGateways(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_project_gateways(project_id TEXT PRIMARY KEY REFERENCES private_projects(id),operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id),body BLOB NOT NULL);`)
	return err
}

func privateGatewayFingerprint(project string, req privateGatewayRequest) string {
	raw, _ := json.Marshal(struct {
		Kind, Project string
		Request       privateGatewayRequest
	}{privateGatewayKind, project, req})
	return string(raw)
}

func loadPrivateGateway(q querier, project string) (privatechain.Gateway, error) {
	var g privatechain.Gateway
	var operation string
	var raw []byte
	err := q.QueryRow("SELECT operation_id,body FROM private_project_gateways WHERE project_id=?", project).Scan(&operation, &raw)
	if err != nil {
		return g, err
	}
	if json.Unmarshal(raw, &g) != nil || g.Selection.Binding.ProjectID != project || g.Selection.OperationID != operation {
		return g, errors.New("invalid private gateway journal")
	}
	canonical, _ := json.Marshal(g)
	if !bytes.Equal(raw, canonical) {
		return g, errors.New("invalid private gateway journal encoding")
	}
	return g, nil
}

func privateGatewayClaims(q querier) ([]privatechain.Gateway, error) {
	rows, err := q.Query("SELECT project_id FROM private_project_gateways ORDER BY project_id")
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
	var out []privatechain.Gateway
	for _, id := range ids {
		g, err := loadPrivateGateway(q, id)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// The original workspace receipt stays immutable. A recovered result comes from
// its separately attributed observation; the gateway is bound to that exact view.
func readyPrivateWorkspace(q querier, project string) (privatechain.Workspace, error) {
	w, _, err := readyPrivateWorkspaceEvidence(q, project)
	return w, err
}

func readyPrivateWorkspaceEvidence(q querier, project string) (privatechain.Workspace, protocol.Operation, error) {
	var empty protocol.Operation
	if err := validatePrivateWorkspaceObservations(q); err != nil {
		return privatechain.Workspace{}, empty, err
	}
	w, err := loadPrivateWorkspace(q, project)
	if err != nil {
		return w, empty, err
	}
	op, err := loadOperation(q, w.OperationID)
	if err != nil {
		return w, empty, err
	}
	if w.Phase == "observed" && op.Status == "succeeded" {
		return w, op, nil
	}
	jobs, err := privateWorkspaceObservations(q)
	if err != nil {
		return w, empty, err
	}
	for _, j := range jobs {
		if j.ProjectID == project && j.Phase == "observed" && j.Result != nil {
			observed, err := loadOperation(q, j.OperationID)
			if err != nil {
				return w, empty, err
			}
			return j.Result.Clone(), observed, nil
		}
	}
	return w, empty, conflict("project_recovery_required", "complete an attributed observation of the retained workspace before gateway initialization")
}

// This validator performs database reads only. Resource IDs and public key
// material may not be claimed by two private projects. Saved namespace inodes
// are intentionally not treated as permanent, globally unique identities.
func validatePrivateGateways(q querier) error {
	return validateSnapshot(q, "validatePrivateGateways", validatePrivateGatewaysUncached)
}

func validatePrivateGatewaysUncached(q querier) error {
	if err := validatePrivateWorkspaceObservations(q); err != nil {
		return err
	}
	objects, err := privateWorkspaceOwnedObjects(q)
	if err != nil {
		return err
	}
	fail := func() error { return errors.New("private gateway ownership, journal or operation receipt differs") }
	claims, err := privateGatewayClaims(q)
	if err != nil {
		return err
	}
	resources, keys := map[string]bool{}, map[string]bool{}
	for _, g := range claims {
		s := g.Selection
		p, err := loadPrivateProject(q, s.Binding.ProjectID)
		if err != nil {
			return err
		}
		w, workspaceOp, err := readyPrivateWorkspaceEvidence(q, s.Binding.ProjectID)
		if err != nil {
			return err
		}
		if g.Validate(p.Plan, w) != nil || s.Binding != p.Binding || s.Revision != p.Revision || s.OperationID == p.RegistrationOperationID {
			return fail()
		}
		for _, id := range []string{g.NetworkID, g.ContainerID} {
			if id != "" {
				if resources[id] {
					return fail()
				}
				resources[id] = true
			}
		}
		if g.Observation != nil {
			if err = claimPrivateObjects(objects, s.Binding.ProjectID, g.Observation.Chain, g.Observation.Secrets); err != nil {
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
		}
		var actor, request, fingerprint string
		var body, mirror []byte
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", s.OperationID).Scan(&actor, &request, &fingerprint, &body, &mirror); err != nil {
			return err
		}
		var op protocol.Operation
		if json.Unmarshal(body, &op) != nil || op.ID != s.OperationID || op.ProjectID != s.Binding.ProjectID || op.InstanceID != "" || op.Kind != privateGatewayKind || actor == "" || op.Actor != actor || op.RequestID != request || !protocol.ValidID(request) || op.ConfigRevision != s.Revision || op.CreatedAt.IsZero() || workspaceOp.FinishedAt == nil || op.CreatedAt.Before(*workspaceOp.FinishedAt) {
			return fail()
		}
		canonical, _ := json.Marshal(op)
		runtime, _ := json.Marshal(g)
		req := privateGatewayRequest{RequestID: request, ExpectedRevision: s.Revision, ReviewDigest: s.PlanDigest, WorkspaceOperationID: s.WorkspaceOperationID, WorkspaceDigest: s.WorkspaceDigest, InputManifestSHA256: s.Inputs.ManifestSHA256}
		if !bytes.Equal(body, canonical) || !bytes.Equal(mirror, runtime) || fingerprint != privateGatewayFingerprint(s.Binding.ProjectID, req) {
			return fail()
		}
		switch op.Status {
		case "queued":
			if g.Phase != "planned" || op.FinishedAt != nil {
				return fail()
			}
		case "running":
			if g.Phase == "ready" || op.FinishedAt != nil {
				return fail()
			}
		case "interrupted":
			if g.Phase == "ready" || op.FinishedAt == nil {
				return fail()
			}
		case "succeeded":
			if g.Phase != "ready" || op.FinishedAt == nil {
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
	err = q.QueryRow(`SELECT count(*) FROM operations WHERE (json_extract(body,'$.kind')=? OR fingerprint LIKE ?) AND id NOT IN (SELECT operation_id FROM private_project_gateways)`, privateGatewayKind, `{"Kind":"`+privateGatewayKind+`",%`).Scan(&orphan)
	if err != nil {
		return err
	}
	if orphan != 0 {
		return fail()
	}
	return nil
}

func privateGatewayState(q querier) error {
	_, err := projectReservations(q)
	return err
}

// Engine/input inspection is backend-selected, not a client runtime descriptor.
// Replay precedes its use and never substitutes a new engine/input generation.
func (s *store) acceptPrivateGateway(actor, project string, req privateGatewayRequest, engine string, inputs privatechain.RuntimeInputInspection) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("private gateway requires actor, project, request and reviewed revision")
	}
	fingerprint := privateGatewayFingerprint(project, req)
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
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private gateway receipt exists for this actor")
	}
	p, err := loadPrivateProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	w, err := readyPrivateWorkspace(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = loadPrivateGateway(tx, project); !errors.Is(err, sql.ErrNoRows) {
		if err != nil {
			return protocol.Operation{}, false, err
		}
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original gateway attempt; do not repeat chain initialization")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, Kind: privateGatewayKind, ProjectID: project, RequestID: req.RequestID, Actor: actor, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	g, err := privatechain.NewGateway(p.Plan, w, op.ID, engine, inputs)
	if err != nil {
		return protocol.Operation{}, false, bad("private gateway input selection or workspace is invalid")
	}
	selection := g.Selection
	if req.ExpectedRevision != p.Revision || req.ReviewDigest != selection.PlanDigest || req.WorkspaceOperationID != selection.WorkspaceOperationID || req.WorkspaceDigest != selection.WorkspaceDigest || req.InputManifestSHA256 != inputs.ManifestSHA256 {
		return protocol.Operation{}, false, conflict("revision_conflict", "review the current private project, workspace and input generation")
	}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	raw, _ := json.Marshal(g)
	if _, err = tx.Exec("INSERT INTO private_project_gateways(project_id,operation_id,body) VALUES(?,?,?)", project, op.ID, raw); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID); err != nil {
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

func (s *store) beginPrivateGateway(project, operation string) (privatechain.Gateway, error) {
	var g privatechain.Gateway
	tx, err := s.db.Begin()
	if err != nil {
		return g, err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return g, err
	}
	g, err = loadPrivateGateway(tx, project)
	if err != nil {
		return g, err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return g, err
	}
	if g.Selection.OperationID != operation || op.Status != "queued" || g.Phase != "planned" {
		return g, conflict("stale_project_operation", "private gateway is no longer freshly queued")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return g, err
	}
	return g, tx.Commit()
}

func (s *store) advancePrivateGateway(project, operation, expected string, next privatechain.Gateway) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	g, err := loadPrivateGateway(tx, project)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(tx, project)
	if err != nil {
		return err
	}
	w, err := readyPrivateWorkspace(tx, project)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return err
	}
	if g.Selection.OperationID != operation || op.Status != "running" {
		return conflict("stale_project_operation", "private gateway operation changed")
	}
	want, err := g.Advance(p.Plan, w, expected, next.Phase, next.NetworkID, next.ContainerID, next.Observation)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, next) {
		return errors.New("private gateway callback changed immutable selection")
	}
	if next.Observation != nil {
		at := next.Observation.ObservedAt
		now := time.Now().UTC()
		if now.Sub(at) > 30*time.Second || at.Sub(now) > 5*time.Second {
			return errors.New("private gateway observation expired before commit")
		}
	}
	raw, _ := json.Marshal(next)
	if _, err = tx.Exec("UPDATE private_project_gateways SET body=? WHERE project_id=? AND operation_id=?", raw, project, operation); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, operation); err != nil {
		return err
	}
	if next.Phase == "ready" {
		now := time.Now().UTC()
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "private chain gateway ready; member ships have not been started"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	if _, err = projectReservations(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func interruptPrivateGateway(tx *sql.Tx, project, operation string) error {
	g, err := loadPrivateGateway(tx, project)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return err
	}
	if g.Selection.OperationID != operation || op.ProjectID != project || op.Kind != privateGatewayKind {
		return errors.New("private gateway interruption ownership differs")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "private gateway outcome requires inspection; retain original resources, chain and keys without repeating initialization"
	return saveOperation(tx, op)
}

func reconcilePrivateGateways(tx *sql.Tx) error {
	if err := validatePrivateGateways(tx); err != nil {
		return err
	}
	claims, err := privateGatewayClaims(tx)
	if err != nil {
		return err
	}
	for _, g := range claims {
		if err = interruptPrivateGateway(tx, g.Selection.Binding.ProjectID, g.Selection.OperationID); err != nil {
			return err
		}
	}
	return nil
}

func (s *store) interruptPrivateGateway(project, operation string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	if err = interruptPrivateGateway(tx, project, operation); err != nil {
		return err
	}
	return tx.Commit()
}

// Internal only. The future queue must hold the common runner work/update fence
// and close the concrete driver. Neither public routes nor startup dispatch here.
func (s *store) runPrivateGateway(ctx context.Context, project, operation string, driver privatechain.GatewayDriver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if driver == nil {
		return errors.New("private gateway driver is required")
	}
	g, err := s.beginPrivateGateway(project, operation)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(s.db, project)
	if err == nil {
		w, e := readyPrivateWorkspace(s.db, project)
		err = e
		if err == nil {
			_, err = privatechain.ExecuteGateway(ctx, p.Plan, w, g, driver, func(expected string, next privatechain.Gateway) error {
				return s.advancePrivateGateway(project, operation, expected, next)
			}, time.Now)
		}
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptPrivateGateway(project, operation))
}
