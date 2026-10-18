package managed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/access"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
)

func routineKind(kind string) bool { return kind == "routine-pack" || kind == "routine-meld" }

func loadRoutinePolicy(q querier, id string) (*routines.Policy, error) {
	var data []byte
	if err := q.QueryRow("SELECT body FROM maintenance_policies WHERE instance_id=?", id).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var p routines.Policy
	if json.Unmarshal(data, &p) != nil || p.InstanceID != id || !protocol.ValidID(p.ID) || p.Revision == 0 || p.InstanceRevision == 0 || p.Owner == "" || p.UpdatedAt.IsZero() || routines.Validate(&p.Config) != nil {
		return nil, errors.New("invalid stored maintenance policy")
	}
	return &p, nil
}

func saveRoutinePolicy(tx *sql.Tx, p routines.Policy) error {
	p.NextSlot = nil
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO maintenance_policies(instance_id,body) VALUES(?,?) ON CONFLICT(instance_id) DO UPDATE SET body=excluded.body", p.InstanceID, data)
	return err
}

func (s *store) putRoutine(actor, id string, req routines.Update) (routines.Policy, error) {
	var empty routines.Policy
	if req.ExpectedRevision >= math.MaxInt64 || req.ExpectedInstanceRevision == 0 || req.ExpectedInstanceRevision > math.MaxInt64 {
		return empty, bad("require valid policy and instance revisions")
	}
	if err := routines.Validate(&req.Config); err != nil {
		return empty, bad(err.Error())
	}
	tx, err := s.db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	v, err := loadInstance(tx, id)
	if err != nil {
		return empty, err
	}
	if v.Public.ConfigRevision != req.ExpectedInstanceRevision {
		return empty, conflict("revision_conflict", "instance configuration changed")
	}
	if v.Public.ActiveOperationID != "" {
		return empty, conflict("instance_busy", "cannot edit routine policy during active work")
	}
	if req.Config.Enabled && (!v.Runtime.MaintenanceQualified() || v.Public.RecoveryRequired || v.Runtime.UncertainMaintenance() || v.Runtime.UncertainProvisioning()) {
		return empty, conflict("invalid_state", "enable a routine only on an initialized instance without recovery requirements")
	}
	previous, err := loadRoutinePolicy(tx, id)
	if err != nil {
		return empty, err
	}
	var p routines.Policy
	if previous != nil {
		p = *previous
	} else {
		p.ID, err = randomID()
		if err != nil {
			return empty, err
		}
		p.InstanceID = id
	}
	if p.Revision != req.ExpectedRevision {
		return empty, conflict("policy_revision_conflict", "routine policy changed; reload before editing")
	}
	p.Revision++
	p.InstanceRevision = req.ExpectedInstanceRevision
	p.Config = req.Config
	p.Owner = actor
	p.UpdatedAt = time.Now().UTC()
	if err := saveRoutinePolicy(tx, p); err != nil {
		return empty, err
	}
	key, err := randomID()
	if err != nil {
		return empty, err
	}
	now := p.UpdatedAt
	op := protocol.Operation{ID: "op-" + key, RequestID: "server-policy-" + key, InstanceID: id, Kind: "configure-routine", Actor: actor, Status: "succeeded", ConfigRevision: v.Public.ConfigRevision, CreatedAt: now, FinishedAt: &now}
	if err := insertOperation(tx, op, "routine-policy:"+p.ID); err != nil {
		return empty, err
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	p.NextSlot = routines.Next(p, time.Now().UTC())
	return p, nil
}

func (r *Server) routineAuthorized(actor string) bool {
	if actor == fmt.Sprintf("local-uid:%d", r.uid) || actor == "local-uid:0" {
		return true
	}
	if !strings.HasPrefix(actor, "ssh-key:") || r.keys == "" {
		return false
	}
	g, err := access.Load(r.keys, strings.TrimPrefix(actor, "ssh-key:"))
	return err == nil && g.Role == "operator"
}

func (s *store) claimRoutine(id string, now time.Time, authorized func(string) bool) (protocol.Operation, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	p, err := loadRoutinePolicy(tx, id)
	if err != nil || p == nil {
		return protocol.Operation{}, false, err
	}
	slot, due := routines.Due(*p, now)
	if !due {
		return protocol.Operation{}, false, nil
	}
	v, err := loadInstance(tx, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	reason := ""
	deadline := slot.Add(time.Duration(p.Config.WindowMinutes) * time.Minute)
	switch {
	case !authorized(p.Owner):
		reason = "policy owner no longer has operator access"
	case now.After(deadline):
		reason = "dispatch window was missed; no catch-up execution"
	case v.Public.ConfigRevision != p.InstanceRevision:
		reason = "configuration changed; policy needs reapproval"
	case v.Public.ActiveOperationID != "":
		reason = "instance is busy"
	case !v.Runtime.MaintenanceQualified():
		reason = "instance maintenance profile is not qualified"
	case v.Public.RecoveryRequired || v.Runtime.UncertainMaintenance() || v.Runtime.UncertainProvisioning():
		reason = "instance requires recovery review"
	case v.Public.ObservedAt == nil || now.Sub(*v.Public.ObservedAt) > 30*time.Second || v.Public.ObservedAt.After(now.Add(5*time.Second)):
		reason = "runtime observation is stale"
	case v.Public.ObservedState != "running" && v.Public.ObservedState != "stopped":
		reason = "runtime state is not settled"
	case v.Public.ObservedState != v.Public.DesiredState:
		reason = "desired and observed runtime states differ"
	case v.Public.ObservedState == "running" && (!p.Config.StopRunning || v.Public.AppliedRevision != v.Public.ConfigRevision):
		reason = "running runtime lacks downtime consent or has configuration drift"
	}
	key, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	op := protocol.Operation{ID: "op-" + key, RequestID: fmt.Sprintf("schedule-%s-%d", p.ID, slot.Unix()), InstanceID: id, Kind: "routine-" + p.Config.Action, Actor: p.Owner, ConfigRevision: v.Public.ConfigRevision, CreatedAt: now, Status: "queued"}
	if reason != "" {
		op.Status = "skipped"
		op.Message = reason
		op.FinishedAt = &now
	} else {
		v.Public.ActiveOperationID = op.ID
		v.Runtime.Routine = &routines.Run{PolicyID: p.ID, PolicyRevision: p.Revision, Slot: slot, Deadline: deadline, Action: p.Config.Action, StopRunning: p.Config.StopRunning, Resume: p.Config.Resume, WasRunning: v.Public.ObservedState == "running", Stage: "queued"}
		if !v.Runtime.Routine.WasRunning || !p.Config.Resume {
			v.Public.DesiredState = "stopped"
		}
		v.Public.Routine = v.Runtime.Routine
		if err := saveInstance(tx, v); err != nil {
			return op, false, err
		}
	}
	p.LastSlot = &slot
	p.LastOperationID = op.ID
	if err := insertOperation(tx, op, "routine-slot:"+p.ID+":"+slot.Format(time.RFC3339)); err != nil {
		return op, false, err
	}
	if err := saveRoutinePolicy(tx, *p); err != nil {
		return op, false, err
	}
	return op, reason == "", tx.Commit()
}

func (r *Server) scheduleLoop() {
	defer r.wg.Done()
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-timer.C:
			r.dispatchRoutines(time.Now().UTC())
		}
	}
}

func (r *Server) dispatchRoutines(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing || r.failed.Load() || r.updateFence.Held {
		return
	}
	all, err := instances(r.s.db)
	if err != nil {
		r.failed.Store(true)
		return
	}
	for _, v := range all {
		op, queued, err := r.s.claimRoutine(v.Public.ID, now, r.routineAuthorized)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if queued {
			r.launch(op)
		}
	}
}

func (r *Server) serveRoutine(w http.ResponseWriter, req *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "routine policy requires operator access"}})
		return
	}
	if req.Method == http.MethodGet {
		if _, err := loadInstance(r.s.db, id); err != nil {
			r.respondError(w, err)
			return
		}
		p, err := loadRoutinePolicy(r.s.db, id)
		if err != nil {
			r.respondError(w, err)
			return
		}
		if p != nil {
			p.NextSlot = routines.Next(*p, time.Now().UTC())
		}
		respond(w, 200, protocol.RoutineResponse{RunnerID: r.s.runnerID, InstanceID: id, Policy: p})
		return
	}
	if req.Method != http.MethodPut {
		r.respondError(w, bad("routine endpoint supports GET or PUT"))
		return
	}
	var body routines.Update
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	p, err := r.s.putRoutine(actor, id, body)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, p)
}
