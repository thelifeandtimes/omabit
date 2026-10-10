package managed

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type codeBackend interface {
	LoginCode(context.Context, runtimequalify.ManagedState, string) (string, error)
}

func (realBackend) LoginCode(ctx context.Context, state runtimequalify.ManagedState, container string) (string, error) {
	return runtimequalify.ReadLoginCode(ctx, state, container)
}

var fullContainerID = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Caller holds the mutation mutex through the bounded read; lifecycle/config
// changes cannot race the selected instance. The code is never persisted.
func (r *Server) loginCode(w http.ResponseWriter, req *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "login codes require operator access"}})
		return
	}
	var body protocol.LoginCodeRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if body.ExpectedRevision == 0 || !fullContainerID.MatchString(body.ContainerID) {
		r.respondError(w, bad("require the selected revision and full runtime container ID"))
		return
	}
	v, err := loadInstance(r.s.db, id)
	if err != nil {
		r.respondError(w, err)
		return
	}
	p := v.Public
	if p.ConfigRevision != body.ExpectedRevision || p.ObservedState != "running" || p.ActiveOperationID != "" || p.RecoveryRequired || p.Resources == nil || p.Resources.ContainerID != body.ContainerID || p.ObservedAt == nil || time.Since(*p.ObservedAt) > 30*time.Second || p.ObservedAt.After(time.Now().Add(5*time.Second)) {
		r.respondError(w, conflict("runtime_changed", "select a fresh running instance and its current container"))
		return
	}
	b, ok := r.b.(codeBackend)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "runtime code retrieval unavailable"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 4*time.Second)
	defer cancel()
	code, err := b.LoginCode(ctx, v.Runtime, body.ContainerID)
	success := err == nil && ctx.Err() == nil && protocol.ValidLoginCode(code)
	if err := r.s.auditCode(actor, id, p.ConfigRevision, success); err != nil {
		r.failed.Store(true)
		r.respondError(w, err)
		return
	}
	if !success {
		r.respondError(w, &fault{502, protocol.APIError{Code: "code_unavailable", Message: "login code could not be read from the selected runtime"}})
		return
	}
	respond(w, 200, protocol.LoginCode{RunnerID: r.s.runnerID, InstanceID: id, ConfigRevision: p.ConfigRevision, ContainerID: body.ContainerID, Code: code})
}

func (s *store) auditCode(actor, id string, revision uint64, success bool) error {
	key, err := randomID()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	op := protocol.Operation{ID: "op-" + key, RequestID: "server-code-" + key, InstanceID: id, Kind: "login-code", Actor: actor, ConfigRevision: revision, CreatedAt: now, FinishedAt: &now, Status: "failed", Message: "login-code access failed"}
	if success {
		op.Status, op.Message = "succeeded", "login-code access completed (secret not retained)"
	}
	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO operations(id,actor,request_id,fingerprint,body) VALUES(?,?,?,?,?)", op.ID, actor, op.RequestID, "login-code-audit", data)
	return err
}
