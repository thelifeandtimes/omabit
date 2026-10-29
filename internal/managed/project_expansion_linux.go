//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectExpansionBackend interface {
	ProjectShipAction(context.Context, *runtimequalify.ProjectExpansion, string, string, string, func(runtimequalify.ProjectExpansion) error) error
	AddProjectMember(context.Context, *runtimequalify.ProjectExpansion, string, func(runtimequalify.ProjectExpansion) error) error
	ExpandedProjectFleet(context.Context, runtimequalify.ProjectExpansion) ([]protocol.Instance, error)
	ExpandedProjectCode(context.Context, runtimequalify.ProjectExpansion, string, string) (string, error)
}

func (realBackend) AddProjectMember(ctx context.Context, s *runtimequalify.ProjectExpansion, id string, save func(runtimequalify.ProjectExpansion) error) error {
	return runtimequalify.RunProjectAddition(ctx, s, id, save)
}
func (realBackend) ExpandedProjectFleet(ctx context.Context, s runtimequalify.ProjectExpansion) ([]protocol.Instance, error) {
	return runtimequalify.ReadExpandedProjectFleet(ctx, s)
}
func (realBackend) ExpandedProjectCode(ctx context.Context, s runtimequalify.ProjectExpansion, id, c string) (string, error) {
	return runtimequalify.ReadExpandedProjectCode(ctx, s, id, c)
}

func (realBackend) ProjectShipAction(ctx context.Context, s *runtimequalify.ProjectExpansion, id, action, op string, save func(runtimequalify.ProjectExpansion) error) error {
	return runtimequalify.RunProjectShipAction(ctx, s, id, action, op, save)
}

func initializeProjectExpansions(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS project_expansions(project_id TEXT PRIMARY KEY REFERENCES projects(id),body BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS project_expansion_members(instance_id TEXT PRIMARY KEY,project_id TEXT NOT NULL REFERENCES project_expansions(project_id),http_port INTEGER NOT NULL UNIQUE);`)
	return err
}
func loadProjectExpansion(q querier, id string) (runtimequalify.ProjectExpansion, error) {
	var s runtimequalify.ProjectExpansion
	var body []byte
	err := q.QueryRow("SELECT body FROM project_expansions WHERE project_id=?", id).Scan(&body)
	if err != nil {
		return s, err
	}
	if json.Unmarshal(body, &s) != nil || s.Version != 1 || s.Project.ProjectID != id {
		return s, errors.New("invalid project expansion ledger")
	}
	return s, nil
}
func (r *Server) projectExpanded(id string) bool {
	var count int
	return r.s.db.QueryRow("SELECT count(*) FROM project_expansions WHERE project_id=?", id).Scan(&count) == nil && count > 0
}
func allProjectExpansions(q querier) ([]runtimequalify.ProjectExpansion, error) {
	rows, err := q.Query("SELECT project_id FROM project_expansions")
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
	var all []runtimequalify.ProjectExpansion
	for _, id := range ids {
		s, err := loadProjectExpansion(q, id)
		if err != nil {
			return nil, err
		}
		all = append(all, s)
	}
	return all, nil
}
func expansionReservations(q querier, reserved *projectnetwork.Reservations) error {
	all, err := allProjectExpansions(q)
	if err != nil {
		return err
	}
	type claim struct {
		project string
		port    uint16
	}
	expected := map[string]claim{}
	for _, s := range all {
		p, err := loadProject(q, s.Project.ProjectID)
		if err != nil {
			return err
		}
		digest, err := p.Plan.Digest()
		selected, e := s.Project.Plan.Digest()
		if err != nil || e != nil || digest != selected || p.Revision != s.Project.Revision {
			return errors.New("expansion plan changed")
		}
		ships := map[string]bool{}
		ports := map[uint16]bool{}
		original := map[string]projectnetwork.Member{}
		for _, m := range p.Plan.Members {
			original[m.InstanceID] = m
		}
		for _, m := range s.Members {
			if ships[m.Member.Ship] || ports[m.Member.InternalHTTPPort] || m.Creator.InstanceID != m.Member.InstanceID {
				return errors.New("conflicting expansion identity")
			}
			ships[m.Member.Ship] = true
			ports[m.Member.InternalHTTPPort] = true
			if !m.Added {
				if original[m.Member.InstanceID] != m.Member {
					return errors.New("original member changed")
				}
				delete(original, m.Member.InstanceID)
				continue
			}
			id, port := m.Member.InstanceID, m.Member.HTTPPort
			if !protocol.ValidPierName(id) || (!protocol.ValidFakeShip(m.Member.Ship) || len(m.Member.Ship) != 4) || port < 1024 || reserved.Instances[id] || reserved.HTTPPorts[port] {
				return errors.New("conflicting added project member reservation")
			}
			reserved.Instances[id] = true
			reserved.HTTPPorts[port] = true
			expected[id] = claim{s.Project.ProjectID, port}
		}
		if len(original) > 0 {
			return errors.New("expansion lost original member")
		}
	}
	rows, err := q.Query("SELECT instance_id,project_id,http_port FROM project_expansion_members")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, project string
		var port uint16
		if err = rows.Scan(&id, &project, &port); err != nil {
			return err
		}
		if c, ok := expected[id]; !ok || c != (claim{project, port}) {
			return errors.New("expansion claim differs from ledger")
		}
		delete(expected, id)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if len(expected) > 0 {
		return errors.New("expansion claim missing")
	}
	return nil
}
func projectStorageLocations(q querier) (map[string]pierstore.Location, error) {
	out := map[string]pierstore.Location{}
	initial, err := projectMemberStorageClaims(q)
	if err != nil {
		return nil, err
	}
	for _, m := range initial {
		out[m.InstanceID] = m.Location
	}
	all, err := allProjectExpansions(q)
	if err != nil {
		return nil, err
	}
	for _, s := range all {
		for _, m := range s.Members {
			if m.Initialized.Identity.Inode != 0 {
				out[m.Member.InstanceID] = m.Initialized
			} else {
				out[m.Member.InstanceID] = m.Creator.Storage
			}
		}
	}
	return out, nil
}
func addExpansionInventory(q querier, p *protocol.ProjectReservation) error {
	s, err := loadProjectExpansion(q, p.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, m := range s.Members {
		if m.Added {
			p.AdditionalMembers = append(p.AdditionalMembers, protocol.ProjectMemberReservation{InstanceID: m.Member.InstanceID, Ship: m.Member.Ship, HTTPPort: m.Member.HTTPPort, InternalHTTPPort: m.Member.InternalHTTPPort})
		}
	}
	return nil
}
func (r *Server) queueProjectAddition(actor, id string, req protocol.ProjectActionRequest, replayOnly bool) (protocol.Operation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.ctx.Err() != nil || r.updateFence.Held {
		return protocol.Operation{}, false, conflict("runner_unavailable", "runner is unavailable for project changes")
	}
	b, ok := r.b.(projectExpansionBackend)
	if !ok {
		return protocol.Operation{}, false, bad("runner does not support project additions")
	}
	data, _ := json.Marshal(struct {
		Project string
		Request protocol.ProjectActionRequest
	}{id, req})
	if op, found, err := replay(r.s.db, actor, req.RequestID, string(data)); found || err != nil {
		return op, found, err
	}
	if replayOnly {
		return protocol.Operation{}, false, conflict("replay_not_found", "no matching addition receipt")
	}
	p, err := loadProject(r.s.db, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	digest, err := p.Plan.Digest()
	if err != nil || digest != req.ReviewDigest || p.Revision != req.ExpectedRevision {
		return protocol.Operation{}, false, bad("project selection changed")
	}
	var active int
	if err = r.s.db.QueryRow("SELECT count(*) FROM operations WHERE json_extract(body,'$.project_id')=? AND id!=? AND json_extract(body,'$.status') IN ('queued','running','interrupted')", id, req.AfterOperationID).Scan(&active); err != nil || active > 0 {
		return protocol.Operation{}, false, conflict("project_busy", "finish or review the project's pending operation first")
	}
	state, err := loadProjectExpansion(r.s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		if len(p.Members) == 0 {
			a, e := loadProjectAnchorStart(r.s.db, id)
			if e != nil {
				return protocol.Operation{}, false, e
			}
			state = runtimequalify.ProjectExpansion{Version: 1, Project: runtimequalify.ProjectPreparationSelection{RunnerID: r.s.runnerID, ProjectID: id, OperationID: a.Prepared.OperationID, Revision: p.Revision, Generation: a.Prepared.Generation, Plan: p.Plan}, Anchor: a}
		} else {
			selected, e := r.s.projectFleetSelection(id, "")
			if e != nil {
				return protocol.Operation{}, false, e
			}
			state, e = runtimequalify.SeedProjectExpansion(selected)
			if e != nil {
				return protocol.Operation{}, false, e
			}
		}
	} else if err != nil {
		return protocol.Operation{}, false, err
	}
	for _, m := range state.Members {
		if req.Kind == "member-continue" && m.Member.InstanceID == req.InstanceID && recoverableExpansionMember(m) {
			continue
		}
		if m.Phase != "ready" && m.Phase != "stopped" {
			return protocol.Operation{}, false, conflict("project_busy", "an earlier member change needs recovery")
		}
	}
	tx, err := r.s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if req.Kind == "add-member" {
		reserved, err := projectReservations(tx)
		if err != nil {
			return protocol.Operation{}, false, err
		}
		all, err := instances(tx)
		if err != nil {
			return protocol.Operation{}, false, err
		}
		for _, v := range all {
			reserved.Instances[v.Public.ID] = true
			reserved.HTTPPorts[v.Public.Config.HTTPPort] = true
			if v.Runtime.Prepared {
				reserved.HTTPPorts[v.Runtime.Port] = true
			}
			for _, c := range v.Runtime.Runtime.Containers {
				if !c.Removed {
					reserved.HTTPPorts[v.Runtime.Port] = true
				}
			}
		}
		if reserved.Instances[req.InstanceID] || reserved.HTTPPorts[req.HTTPPort] {
			return protocol.Operation{}, false, conflict("project_conflict", "instance or port is already reserved")
		}
		var ordinary int
		if err = tx.QueryRow("SELECT count(*) FROM instances WHERE id=?", req.InstanceID).Scan(&ordinary); err != nil || ordinary > 0 {
			return protocol.Operation{}, false, conflict("instance_exists", "instance already exists")
		}
		if err = r.s.availablePort("tcp", req.HTTPPort); err != nil {
			return protocol.Operation{}, false, err
		}
	} else if req.Kind == "member-continue" {
		previous, e := loadOperation(tx, req.AfterOperationID)
		if e != nil || previous.Actor != actor || previous.ProjectID != id || previous.InstanceID != req.InstanceID || (previous.Kind != "add-project-member" && previous.Kind != "start-project-ship") || previous.Status != "interrupted" {
			return protocol.Operation{}, false, bad("continuation requires the exact interrupted preflight")
		}
		found := false
		for _, m := range state.Members {
			if m.Member.InstanceID == req.InstanceID && recoverableExpansionMember(m) && m.Creator.OperationID == previous.ID && ((m.Phase == "reserved" && previous.Kind == "add-project-member") || (m.Phase == "retired" && previous.Kind == "start-project-ship")) {
				found = true
			}
		}
		if !found {
			return protocol.Operation{}, false, bad("continuation requires a reserved addition or a retired start before successor creation")
		}
		previous.Status = "failed"
		previous.Message = "Preflight failed before dispatch; explicit continuation accepted as " + req.RequestID
		if err = saveOperation(tx, previous); err != nil {
			return protocol.Operation{}, false, err
		}
	} else {
		found := false
		for _, m := range state.Members {
			if m.Member.InstanceID == req.InstanceID && m.Creator.Container.ID == req.ContainerID {
				found = true
			}
		}
		if !found {
			return protocol.Operation{}, false, conflict("runtime_changed", "selected project container changed")
		}
	}
	runtimeID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + opID, RequestID: req.RequestID, ProjectID: id, InstanceID: req.InstanceID, Kind: map[string]string{"member-continue": "add-project-member", "add-member": "add-project-member", "member-start": "start-project-ship", "member-stop": "stop-project-ship"}[req.Kind], Actor: actor, ConfigRevision: p.Revision, Status: "queued", CreatedAt: time.Now().UTC()}
	if req.Kind == "add-member" {
		state, err = state.NewMember(projectnetwork.MemberRequest{InstanceID: req.InstanceID, Ship: req.Ship, HTTPPort: req.HTTPPort}, runtimeID, op.ID)
		if err != nil {
			return protocol.Operation{}, false, bad(err.Error())
		}
	}
	if req.Kind == "member-continue" {
		for i, m := range state.Members {
			if m.Member.InstanceID == req.InstanceID {
				state.History = append(state.History, m)
				state.Members[i].Creator.OperationID = op.ID
				if m.Phase == "retired" {
					op.Kind = "start-project-ship"
				}
			}
		}
	}
	body, _ := json.Marshal(state)
	if _, err = tx.Exec("INSERT INTO project_expansions(project_id,body) VALUES(?,?) ON CONFLICT(project_id) DO UPDATE SET body=excluded.body", id, body); err != nil {
		return protocol.Operation{}, false, err
	}
	if req.Kind == "add-member" {
		if _, err = tx.Exec("INSERT INTO project_expansion_members(instance_id,project_id,http_port) VALUES(?,?,?)", req.InstanceID, id, req.HTTPPort); err != nil {
			return protocol.Operation{}, false, err
		}
	}
	if err = insertOperation(tx, op, string(data)); err != nil {
		return protocol.Operation{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return protocol.Operation{}, false, err
	}
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { r.mu.Lock(); r.updateWorkers--; r.mu.Unlock() }()
		var failure error
		select {
		case r.workGate <- struct{}{}:
			defer func() { <-r.workGate }()
			save := func(next runtimequalify.ProjectExpansion) error { return r.s.saveProjectExpansion(op.ID, next) }
			if req.Kind == "member-continue" && op.Kind == "start-project-ship" {
				failure = b.ProjectShipAction(r.ctx, &state, req.InstanceID, "continue-start", op.ID, save)
			} else if req.Kind == "add-member" || req.Kind == "member-continue" {
				failure = b.AddProjectMember(r.ctx, &state, req.InstanceID, save)
			} else {
				failure = b.ProjectShipAction(r.ctx, &state, req.InstanceID, map[string]string{"member-start": "start", "member-stop": "stop"}[req.Kind], op.ID, save)
			}
		case <-r.ctx.Done():
			failure = r.ctx.Err()
		}
		if err := r.s.finishProjectExpansion(op.ID, failure); err != nil {
			r.failed.Store(true)
		}
	}()
	return op, false, nil
}
func (s *store) saveProjectExpansion(opID string, state runtimequalify.ProjectExpansion) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	if op.ProjectID != state.Project.ProjectID || (op.Status != "queued" && op.Status != "running") {
		return errors.New("addition journal is no longer active")
	}
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE project_expansions SET body=? WHERE project_id=?", body, op.ProjectID); err != nil {
		return err
	}
	op.Status = "running"
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *store) finishProjectExpansion(opID string, cause error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	op, err := loadOperation(tx, opID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	op.FinishedAt = &now
	op.Status = "succeeded"
	op.Message = "Project member ready"
	if op.Kind == "stop-project-ship" {
		op.Message = "Project member stopped"
	}
	if cause != nil {
		op.Status = "interrupted"
		op.Message = "Project member addition paused; retained state needs review: " + cause.Error()
		if len(op.Message) > 512 {
			op.Message = op.Message[:512]
		}
	}
	if err = saveOperation(tx, op); err != nil {
		return err
	}
	return tx.Commit()
}

func reconcileProjectExpansions(tx *sql.Tx) error {
	rows, err := tx.Query("SELECT id FROM operations WHERE json_extract(body,'$.kind') IN ('add-project-member','start-project-ship','stop-project-ship') AND json_extract(body,'$.status') IN ('queued','running')")
	if err != nil {
		return err
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
		return err
	}
	for _, id := range ids {
		op, err := loadOperation(tx, id)
		if err != nil {
			return err
		}
		op.Status = "interrupted"
		op.Message = "Project action interrupted; retained member phase requires review before another action."
		now := time.Now().UTC()
		op.FinishedAt = &now
		if err = saveOperation(tx, op); err != nil {
			return err
		}
	}
	return nil
}
func projectExpansionClear(q querier, id string) error {
	var count int
	if err := q.QueryRow("SELECT count(*) FROM project_expansions WHERE project_id=?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return conflict("project_expanded", "use current project ship controls; original lifecycle history is retained")
	}
	return nil
}

func recoverableExpansionMember(m runtimequalify.ProjectExpandedMember) bool {
	return m.Creator.Container.ID == "" && ((m.Added && m.Phase == "reserved" && m.Creator.Storage.MountIdentity.Inode == 0) || (m.Phase == "retired" && m.Creator.Mode == "resume" && m.Initialized.Identity.Inode != 0))
}
