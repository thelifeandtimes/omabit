package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Called under the normal mutation lock/update fence; never invokes a backend.
func (r *Server) servePrivateProjectRegistration(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "private-chain reservations require operator access"}})
		return
	}
	replayOnly, err := protocol.CreationReplayOnly(req.Header)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	media, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		r.respondError(w, &fault{415, protocol.APIError{Code: "invalid_content_type", Message: "require application/json"}})
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, privatechain.MaxFoundationBytes)
	defer req.Body.Close()
	data, err := io.ReadAll(req.Body)
	if err != nil {
		r.respondError(w, bad("private-chain registration exceeds its bounded JSON contract"))
		return
	}
	input, err := privatechain.DecodeRegistration(data)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	plan, err := input.ReviewedPlan()
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	op, replayed, err := r.s.registerPrivateProject(actor, privateProjectRegistration{RequestID: input.RequestID, Plan: plan, ReviewDigest: input.ReviewDigest, ReplayOnly: replayOnly})
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	respond(w, status, privatechain.RegistrationResult{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Binding: privatechain.Binding{Mode: privatechain.Mode, RunnerID: r.s.runnerID, ProjectID: op.ProjectID, Generation: 1, FoundationDigest: plan.FoundationDigest}, ReviewDigest: input.ReviewDigest, ReservationOnly: true, Operation: op})
}

func (s *store) privateProjectInventory(ctx context.Context) (privatechain.Inventory, error) {
	var result privatechain.Inventory
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if _, err = projectReservations(tx); err != nil {
		return result, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT body FROM private_projects ORDER BY id")
	if err != nil {
		return result, err
	}
	projects := []privatechain.Reservation{}
	for rows.Next() {
		var raw []byte
		var p storedPrivateProject
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = json.Unmarshal(raw, &p); err != nil {
			break
		}
		digest, _ := p.Plan.Digest()
		projects = append(projects, privatechain.Reservation{Binding: p.Binding, Revision: p.Revision, Phase: p.Phase, Plan: p.Plan, ReviewDigest: digest, CreatedAt: p.CreatedAt, RegistrationOperationID: p.RegistrationOperationID})
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return privatechain.Inventory{SchemaVersion: 1, RunnerID: s.runnerID, Mode: s.mode, ReadAt: time.Now().UTC(), Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: projects}, nil
}
