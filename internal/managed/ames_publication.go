package managed

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type amesPublicationBackend interface {
	ValidateAmesPublication(context.Context, runtimequalify.ManagedState) error
}

func (realBackend) ValidateAmesPublication(ctx context.Context, state runtimequalify.ManagedState) error {
	return runtimequalify.ValidateAmesPublication(ctx, state)
}

func (r *Server) servePublishAmes(w http.ResponseWriter, req *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "Ames publication requires operator access"}})
		return
	}
	b, ok := r.b.(amesPublicationBackend)
	if !ok || r.s.mode != DevelopmentMode {
		r.respondError(w, conflict("capability_required", "update this live runner for publish-ames support"))
		return
	}
	var body protocol.PublishAmesRequest
	if err := decode(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	op, replayed, err := r.s.publishAmes(req.Context(), actor, id, body, b.ValidateAmesPublication)
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	respond(w, status, op)
}

// The immediate transaction and Server.mu serialize this stopped-state change
// against configuration, runtime callbacks, operator work and scheduled work.
// Validation is read-only; cancellation/interruption before commit publishes
// neither a new launch contract nor a receipt. Replay precedes all live checks.
func (s *store) publishAmes(ctx context.Context, actor, id string, req protocol.PublishAmesRequest, validate func(context.Context, runtimequalify.ManagedState) error) (protocol.Operation, bool, error) {
	if !protocol.ValidID(id) || !protocol.ValidID(req.RequestID) || req.ExpectedRevision == 0 || req.ExpectedRevision > math.MaxInt64 || req.AmesPort < 1024 || !req.ConfirmNetwork {
		return protocol.Operation{}, false, bad("publish-ames requires instance/request IDs, a positive revision, an unprivileged Ames port and explicit network confirmation")
	}
	fingerprint, _ := json.Marshal(struct {
		Kind, ID string
		Request  protocol.PublishAmesRequest
	}{protocol.AmesPublicationFeature, id, req})
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	defer tx.Rollback()
	if op, found, err := replay(tx, actor, req.RequestID, string(fingerprint)); found || err != nil {
		return op, found, err
	}
	v, err := loadInstance(tx, id)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	if v.Public.ConfigRevision != req.ExpectedRevision {
		return protocol.Operation{}, false, &fault{409, protocol.APIError{Code: "revision_conflict", Message: "configuration changed; inspect a fresh snapshot", CurrentRevision: &v.Public.ConfigRevision}}
	}
	if req.ExpectedRevision == math.MaxInt64 {
		return protocol.Operation{}, false, conflict("revision_limit", "configuration revision exhausted")
	}
	if s.mode != DevelopmentMode || v.Public.AmesPublished ||
		v.Public.ActiveOperationID != "" || v.Public.DesiredState != "stopped" || v.Public.ObservedState != "stopped" || v.Public.RecoveryRequired {
		return protocol.Operation{}, false, conflict("invalid_state", "publish-ames requires a cleanly stopped legacy live enrollment with no active operation")
	}
	proposed, err := runtimequalify.AmesPublicationState(v.Runtime, req.AmesPort)
	if err != nil {
		return protocol.Operation{}, false, conflict("invalid_state", err.Error())
	}
	if v.Public.Mode != protocol.ShipClass(proposed.NativeShip) || v.Public.Ship != proposed.NativeShip {
		return protocol.Operation{}, false, conflict("invalid_state", "public enrollment identity differs from the validated live runtime")
	}
	if err := reserveAmes(tx, id, req.AmesPort); err != nil {
		return protocol.Operation{}, false, err
	}
	if validate == nil {
		return protocol.Operation{}, false, conflict("capability_required", "fresh stopped-runtime validation is unavailable")
	}
	if err := validate(ctx, v.Runtime); err != nil {
		return protocol.Operation{}, false, conflict("publication_unavailable", "Ames publication validation failed: "+err.Error())
	}
	// Do the host bind check after potentially slow provenance inspection. The
	// ordinary Start path will check it again before actually publishing UDP.
	if err := s.availablePort("udp", req.AmesPort); err != nil {
		return protocol.Operation{}, false, err
	}
	if err := ctx.Err(); err != nil {
		return protocol.Operation{}, false, conflict("publication_unavailable", err.Error())
	}
	opID, err := randomID()
	if err != nil {
		return protocol.Operation{}, false, err
	}
	prior, err := json.Marshal(v.Runtime)
	if err != nil {
		return protocol.Operation{}, false, err
	}
	now := time.Now().UTC()
	v.Runtime, v.RuntimeRevision = proposed, v.RuntimeRevision+1
	v.Public.Config.AmesPort, v.Public.AmesPublished = req.AmesPort, true
	v.Public.ConfigRevision++
	v.Public.ObservedAt = &now
	v.Startup = nil
	op := protocol.Operation{ID: "op-" + opID, InstanceID: id, RequestID: req.RequestID, Actor: actor, Kind: protocol.AmesPublicationFeature, Status: "succeeded", ConfigRevision: v.Public.ConfigRevision, CreatedAt: now, FinishedAt: &now,
		Message: "Ames publication configured for the next explicit Start; ship remains stopped"}
	if err := saveInstance(tx, v); err != nil {
		return op, false, err
	}
	if err := insertOperation(tx, op, string(fingerprint)); err != nil {
		return op, false, err
	}
	// Preserve the pre-migration private launch/provenance contract alongside
	// the success receipt; previous operation bodies and runtime history remain.
	if _, err := tx.Exec("UPDATE operations SET runtime=? WHERE id=?", prior, op.ID); err != nil {
		return op, false, err
	}
	return op, false, tx.Commit()
}
