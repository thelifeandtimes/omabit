package managed

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

const privateMemberLaunchKind = "launch-private-project-member"

type privateMemberLaunchRequest struct {
	RequestID                  string `json:"request_id"`
	ExpectedRevision           uint64 `json:"expected_revision"`
	ReviewDigest               string `json:"review_digest"`
	GatewayEvidenceOperationID string `json:"gateway_evidence_operation_id"`
	GatewayDigest              string `json:"gateway_digest"`
	ReplayOnly                 bool   `json:"-"`
}

// This internal boundary admits only a project's first member. A second member
// needs a separate durable readiness receipt with held pier identities. Merely
// reaching the running phase cannot supply that authority. There is no public
// route, startup dispatcher or concrete engine adapter here.
func initializePrivateMemberLaunches(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS private_member_launches(instance_id TEXT PRIMARY KEY REFERENCES private_project_members(instance_id),project_id TEXT NOT NULL UNIQUE REFERENCES private_projects(id),operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id),body BLOB NOT NULL);`)
	return err
}

func privateMemberLaunchFingerprint(project, instance string, req privateMemberLaunchRequest) string {
	raw, _ := json.Marshal(struct {
		Kind, Project, Instance string
		Request                 privateMemberLaunchRequest
	}{privateMemberLaunchKind, project, instance, req})
	return string(raw)
}

func loadPrivateMemberLaunch(q querier, instance string) (privatechain.MemberLaunch, error) {
	var j privatechain.MemberLaunch
	var project, operation string
	var raw []byte
	err := q.QueryRow("SELECT project_id,operation_id,body FROM private_member_launches WHERE instance_id=?", instance).Scan(&project, &operation, &raw)
	if err != nil {
		return j, err
	}
	canonical := []byte(nil)
	if json.Unmarshal(raw, &j) == nil {
		canonical, _ = json.Marshal(j)
	}
	if j.InstanceID != instance || j.OperationID != operation || j.Gateway.Selection.Binding.ProjectID != project || !bytes.Equal(raw, canonical) {
		return j, errors.New("invalid private member launch ownership or encoding")
	}
	return j, nil
}

func privateMemberLaunches(q querier) ([]privatechain.MemberLaunch, error) {
	rows, err := q.Query("SELECT instance_id FROM private_member_launches ORDER BY instance_id")
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
	var out []privatechain.MemberLaunch
	for _, id := range ids {
		j, e := loadPrivateMemberLaunch(q, id)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, nil
}

func privateMemberRuntime(p storedPrivateProject, instance string) string {
	for _, m := range p.Members {
		if m.InstanceID == instance {
			return m.RuntimeID
		}
	}
	return ""
}

// Database reads only. Every acceptance and commit checks both immutable gateway
// attribution and the journal/operation mirror. Historical timestamps do not
// expire; freshness is checked at new evidence commits instead.
func validatePrivateMemberLaunches(q querier) error {
	return validateSnapshot(q, "validatePrivateMemberLaunches", validatePrivateMemberLaunchesUncached)
}

func validatePrivateMemberLaunchesUncached(q querier) error {
	if err := validatePrivateGatewayObservations(q); err != nil {
		return err
	}
	jobs, err := privateMemberLaunches(q)
	if err != nil {
		return err
	}
	objects, err := privateWorkspaceOwnedObjects(q)
	if err != nil {
		return err
	}
	gateways, err := privateGatewayClaims(q)
	if err != nil {
		return err
	}
	resources := map[string]bool{}
	for _, g := range gateways {
		for _, id := range []string{g.NetworkID, g.ContainerID} {
			if id != "" {
				resources[id] = true
			}
		}
		if g.Phase == "ready" {
			objects[g.Observation.Chain] = g.Selection.Binding.ProjectID
			objects[g.Observation.Secrets] = g.Selection.Binding.ProjectID
		}
	}
	observations, err := privateGatewayObservations(q)
	if err != nil {
		return err
	}
	for _, j := range observations {
		if j.Result != nil {
			objects[j.Result.Observation.Chain] = j.ProjectID
			objects[j.Result.Observation.Secrets] = j.ProjectID
		}
	}
	fail := func() error {
		return errors.New("private member launch ownership, journal or operation receipt differs")
	}
	projects := map[string]bool{}
	for _, j := range jobs {
		project := j.Gateway.Selection.Binding.ProjectID
		p, e := loadPrivateProject(q, project)
		if e != nil {
			return e
		}
		w, e := readyPrivateWorkspace(q, project)
		if e != nil {
			return e
		}
		g, source, e := readyPrivateGatewayEvidence(q, project)
		if e != nil {
			return e
		}
		if j.Validate(p.Plan, w) != nil || len(j.Peers) != 0 || projects[project] || j.RuntimeID != privateMemberRuntime(p, j.InstanceID) || !reflect.DeepEqual(j.Gateway, g) {
			return fail()
		}
		projects[project] = true
		if j.ContainerID != "" {
			if resources[j.ContainerID] {
				return fail()
			}
			resources[j.ContainerID] = true
		}
		if j.Artifact != (pierstore.DirectoryIdentity{}) {
			for prior := range objects {
				if j.Artifact.SameObject(prior) {
					return fail()
				}
			}
			objects[j.Artifact] = project
		}
		birth := j.Capture
		if birth == nil && j.Running != nil {
			birth = j.Running.Birth
		}
		if birth != nil {
			b := birth.Pier
			for _, id := range []pierstore.DirectoryIdentity{{Device: b.Device, Inode: b.Inode, Birth: b.Birth}, {Device: b.UrbDevice, Inode: b.UrbInode, Birth: b.UrbBirth}} {
				for prior := range objects {
					if id.SameObject(prior) {
						return fail()
					}
				}
				objects[id] = project
			}
		}
		var actor, request, fingerprint string
		var body, mirror []byte
		if err = q.QueryRow("SELECT actor,request_id,fingerprint,body,runtime FROM operations WHERE id=?", j.OperationID).Scan(&actor, &request, &fingerprint, &body, &mirror); err != nil {
			return err
		}
		var op protocol.Operation
		if json.Unmarshal(body, &op) != nil || op.ID != j.OperationID || op.ProjectID != project || op.InstanceID != j.InstanceID || op.Kind != privateMemberLaunchKind || actor == "" || op.Actor != actor || op.RequestID != request || !protocol.ValidID(request) || op.ConfigRevision != p.Revision || op.CreatedAt.IsZero() || source.FinishedAt == nil || op.CreatedAt.Before(*source.FinishedAt) {
			return fail()
		}
		canonical, _ := json.Marshal(op)
		runtime, _ := json.Marshal(j)
		req := privateMemberLaunchRequest{RequestID: request, ExpectedRevision: p.Revision, ReviewDigest: g.Selection.PlanDigest, GatewayEvidenceOperationID: source.ID, GatewayDigest: privateGatewaySourceDigest(g)}
		if !bytes.Equal(body, canonical) || !bytes.Equal(mirror, runtime) || fingerprint != privateMemberLaunchFingerprint(project, j.InstanceID, req) {
			return fail()
		}
		if j.Key != nil && j.Key.CheckedAt.Before(op.CreatedAt) {
			return fail()
		}
		switch op.Status {
		case "queued":
			if j.Phase != "planned" || op.FinishedAt != nil {
				return fail()
			}
		case "running":
			if j.Phase == "running" || op.FinishedAt != nil {
				return fail()
			}
		case "interrupted":
			if j.Phase == "running" || op.FinishedAt == nil {
				return fail()
			}
		case "succeeded":
			if j.Phase != "running" || op.FinishedAt == nil || op.FinishedAt.Before(j.Running.ObservedAt) {
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
	err = q.QueryRow(`SELECT count(*) FROM operations WHERE (json_extract(body,'$.kind')=? OR fingerprint LIKE ?) AND id NOT IN (SELECT operation_id FROM private_member_launches)`, privateMemberLaunchKind, `{"Kind":"`+privateMemberLaunchKind+`",%`).Scan(&orphan)
	if err != nil {
		return err
	}
	if orphan != 0 {
		return fail()
	}
	return nil
}

// The pure backend contract builder is invoked only for a fresh accepted request,
// after replay and ownership checks. It receives copies and may perform no effects.
type privateMemberLaunchContract func(privatechain.Plan, privatechain.Workspace, privatechain.Gateway, string, string, string) (string, error)

func (s *store) acceptPrivateMemberLaunch(actor, project, instance string, req privateMemberLaunchRequest, contract privateMemberLaunchContract) (protocol.Operation, bool, error) {
	return s.acceptPrivateMemberLaunchMode(actor, project, instance, req, contract, false)
}

// Successor admission remains internal until its concrete adapter is qualified.
func (s *store) acceptCapturedPrivateMemberLaunch(actor, project, instance string, req privateMemberLaunchRequest, contract privateMemberLaunchContract) (protocol.Operation, bool, error) {
	return s.acceptPrivateMemberLaunchMode(actor, project, instance, req, contract, true)
}

func (s *store) acceptPrivateMemberLaunchMode(actor, project, instance string, req privateMemberLaunchRequest, contract privateMemberLaunchContract, capture bool) (protocol.Operation, bool, error) {
	if actor == "" || !protocol.ValidID(project) || !protocol.ValidID(instance) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 {
		return protocol.Operation{}, false, bad("private member launch requires actor, project, member and reviewed revision")
	}
	fingerprint := privateMemberLaunchFingerprint(project, instance, req)
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
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching private member launch exists for this actor")
	}
	p, err := loadPrivateProject(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	w, err := readyPrivateWorkspace(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	g, source, err := readyPrivateGateway(tx, project)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	runtime := privateMemberRuntime(p, instance)
	if runtime == "" {
		return protocol.Operation{}, false, bad("member is not reserved by this private project")
	}
	if req.ExpectedRevision != p.Revision || req.ReviewDigest != g.Selection.PlanDigest || req.GatewayEvidenceOperationID != source.ID || req.GatewayDigest != privateGatewaySourceDigest(g) {
		return protocol.Operation{}, false, conflict("revision_conflict", "review the current private project and attributed gateway readiness")
	}
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM private_member_launches WHERE project_id=?", project).Scan(&count); err != nil {
		return protocol.Operation{}, false, err
	}
	if count != 0 {
		return protocol.Operation{}, false, conflict("project_recovery_required", "retain the original member launch; further launch requires separately recorded member readiness")
	}
	if contract == nil {
		return protocol.Operation{}, false, bad("private member launch contract builder is required")
	}
	id, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + id, Kind: privateMemberLaunchKind, ProjectID: project, InstanceID: instance, Actor: actor, RequestID: req.RequestID, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	plan := p.Plan
	plan.Members = slices.Clone(plan.Members)
	plan.Foundation.Contracts = slices.Clone(plan.Foundation.Contracts)
	digest, err := contract(plan, w.Clone(), g.Clone(), instance, runtime, op.ID)
	if err != nil {
		return protocol.Operation{}, false, bad("private member launch contract could not be established")
	}
	constructor := privatechain.NewMemberLaunch
	if capture {
		constructor = privatechain.NewCapturedMemberLaunch
	}
	j, err := constructor(p.Plan, w, g, instance, runtime, op.ID, digest, nil)
	if err != nil {
		return protocol.Operation{}, false, bad("private member launch contract is invalid")
	}
	if err = insertOperation(tx, op, fingerprint); err != nil {
		return protocol.Operation{}, false, err
	}
	raw, _ := json.Marshal(j)
	if _, err = tx.Exec("INSERT INTO private_member_launches(instance_id,project_id,operation_id,body) VALUES(?,?,?,?)", instance, project, op.ID, raw); err != nil {
		return protocol.Operation{}, false, err
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = privateGatewayState(tx); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	return op, false, nil
}

func (s *store) beginPrivateMemberLaunch(project, instance, operation string) (privatechain.MemberLaunch, error) {
	var j privatechain.MemberLaunch
	tx, err := s.db.Begin()
	if err != nil {
		return j, err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return j, err
	}
	j, err = loadPrivateMemberLaunch(tx, instance)
	if err != nil {
		return j, err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return j, err
	}
	if j.OperationID != operation || j.Gateway.Selection.Binding.ProjectID != project || op.Status != "queued" || j.Phase != "planned" || (j.Version != 2 && j.Version != 3) {
		return j, conflict("stale_project_operation", "private member launch is no longer freshly queued")
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return j, err
	}
	return j, tx.Commit()
}

func (s *store) advancePrivateMemberLaunch(project, instance, operation, expected string, next privatechain.MemberLaunch) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	j, err := loadPrivateMemberLaunch(tx, instance)
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
	if j.OperationID != operation || j.Gateway.Selection.Binding.ProjectID != project || op.Status != "running" || (j.Version != 2 && j.Version != 3) {
		return conflict("stale_project_operation", "private member launch ownership changed")
	}
	result := privatechain.MemberLaunchResult{}
	switch next.Phase {
	case "artifact-created":
		result.Artifact = next.Artifact
	case "key-verified":
		result.Key = next.Key
	case "created":
		result.ContainerID = next.ContainerID
	case "captured":
		result.Capture = next.Capture
	case "running":
		result.Running = next.Running
	}
	want, err := j.Advance(p.Plan, w, expected, next.Phase, result)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, next) {
		return errors.New("private member callback changed immutable selection or prior evidence")
	}
	now := time.Now().UTC()
	fresh := func(at time.Time) bool {
		return !at.IsZero() && now.Sub(at) <= 30*time.Second && at.Sub(now) <= 5*time.Second
	}
	if next.Phase == "key-verified" && !fresh(next.Key.CheckedAt) || next.Phase == "captured" && !fresh(next.Capture.CapturedAt) || next.Phase == "running" && !fresh(next.Running.ObservedAt) {
		return errors.New("private member evidence expired before commit")
	}
	raw, _ := json.Marshal(next)
	updated, err := tx.Exec("UPDATE private_member_launches SET body=? WHERE instance_id=? AND project_id=? AND operation_id=?", raw, instance, project, operation)
	if err != nil {
		return err
	}
	n, err := updated.RowsAffected()
	if err != nil || n != 1 {
		return errors.New("private member journal lost ownership")
	}
	if _, err = tx.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, operation); err != nil {
		return err
	}
	if next.Phase == "running" {
		op.Status = "succeeded"
		op.FinishedAt = &now
		op.Message = "private member process observed; ship readiness has not been established"
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func interruptPrivateMemberLaunch(tx *sql.Tx, project, instance, operation string) error {
	j, err := loadPrivateMemberLaunch(tx, instance)
	if err != nil {
		return err
	}
	op, err := loadOperation(tx, operation)
	if err != nil {
		return err
	}
	if j.OperationID != operation || j.Gateway.Selection.Binding.ProjectID != project || op.Kind != privateMemberLaunchKind || op.InstanceID != instance || op.ProjectID != project {
		return errors.New("private member interruption ownership differs")
	}
	if op.Status != "queued" && op.Status != "running" {
		return nil
	}
	now := time.Now().UTC()
	op.Status = "interrupted"
	op.FinishedAt = &now
	op.Message = "private member outcome requires inspection; retain its original intent, artifacts, container and partial pier without repeating boot"
	return saveOperation(tx, op)
}
func reconcilePrivateMemberLaunches(tx *sql.Tx) error {
	if err := validatePrivateMemberLaunches(tx); err != nil {
		return err
	}
	jobs, err := privateMemberLaunches(tx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err = interruptPrivateMemberLaunch(tx, j.Gateway.Selection.Binding.ProjectID, j.InstanceID, j.OperationID); err != nil {
			return err
		}
	}
	return nil
}
func (s *store) interruptPrivateMemberLaunch(project, instance, operation string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = privateGatewayState(tx); err != nil {
		return err
	}
	if err = interruptPrivateMemberLaunch(tx, project, instance, operation); err != nil {
		return err
	}
	return tx.Commit()
}

// The caller must hold the common runner work/update fence and close its driver.
// Opening/reconciling the database never invokes this method or resumes a boot.
func (s *store) runPrivateMemberLaunch(ctx context.Context, project, instance, operation string, d privatechain.MemberLaunchDriver) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil {
		return errors.New("private member launch driver is required")
	}
	j, err := s.beginPrivateMemberLaunch(project, instance, operation)
	if err != nil {
		return err
	}
	p, err := loadPrivateProject(s.db, project)
	if err == nil {
		w, e := readyPrivateWorkspace(s.db, project)
		err = e
		if err == nil {
			commit := func(expected string, next privatechain.MemberLaunch) error {
				return s.advancePrivateMemberLaunch(project, instance, operation, expected, next)
			}
			if j.Version == 3 {
				if captureDriver, ok := d.(privatechain.CapturedMemberLaunchDriver); ok {
					_, err = privatechain.ExecuteCapturedMemberLaunch(ctx, p.Plan, w, j, captureDriver, commit, time.Now)
				} else {
					err = errors.New("private member capture driver is required")
				}
			} else {
				_, err = privatechain.ExecuteMemberLaunch(ctx, p.Plan, w, j, d, commit, time.Now)
			}
		}
	}
	if err == nil {
		return nil
	}
	return errors.Join(err, s.interruptPrivateMemberLaunch(project, instance, operation))
}
