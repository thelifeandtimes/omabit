package managed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Server) grubberyRuntimeStatus(ctx context.Context, original, audit string) (protocol.GrubberyRuntimeRecoveryStatus, error) {
	var out protocol.GrubberyRuntimeRecoveryStatus
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = validateGrubberyJournal(tx); err != nil {
		return out, err
	}
	record, err := loadGrubberyRecord(tx, original)
	if err != nil {
		return out, err
	}
	var receipt *grubbery.RuntimeRecoveryReceipt
	if audit == "" {
		head, err := loadAppRuntimeHead(tx, original)
		if err != nil {
			return out, err
		}
		if head.ordinal != 0 {
			if err = tx.QueryRow("SELECT request_id FROM grubbery_app_runtime_recoveries WHERE original_id=? AND ordinal=?", original, head.ordinal).Scan(&audit); err != nil {
				return out, grubbery.ErrHistory
			}
		}
	}
	if audit != "" {
		var parent string
		if err = tx.QueryRow("SELECT original_id FROM grubbery_app_runtime_recoveries WHERE request_id=?", audit).Scan(&parent); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return out, grubbery.ErrMissing
			}
			return out, err
		}
		if parent != original {
			return out, grubbery.ErrConflict
		}
		saved, err := readAppRuntimeRecovery(tx, audit, record)
		if err != nil {
			return out, err
		}
		receipt = &saved
	}
	out = protocol.GrubberyRuntimeRecoveryStatus{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Original: record, Receipt: receipt}
	if err = out.Validate(); err != nil {
		return protocol.GrubberyRuntimeRecoveryStatus{}, err
	}
	return out, tx.Commit()
}

// Saved-only and outside r.mu: validates journal snapshots without native,
// source, policy, management or active-process admission.
func (r *Server) serveGrubberyRuntimeReceipt(w http.ResponseWriter, req *http.Request, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "App runtime recovery receipts require operator access."}})
		return
	}
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	original, audit, ok := protocol.GrubberyRuntimeReceiptPath(req.URL.Path)
	if !ok {
		r.respondError(w, bad("invalid original app operation or recovery audit ID"))
		return
	}
	out, err := r.grubberyRuntimeStatus(req.Context(), original, audit)
	if errors.Is(err, grubbery.ErrMissing) {
		r.respondError(w, &fault{404, protocol.APIError{Code: "app_operation_not_found", Message: "Original app operation or recovery audit not found."}})
		return
	}
	if errors.Is(err, grubbery.ErrConflict) || errors.Is(err, grubbery.ErrCAS) {
		r.respondError(w, &fault{409, protocol.APIError{Code: "app_operation_changed", Message: "Original app operation or recovery audit binding changed."}})
		return
	}
	if errors.Is(err, grubbery.ErrHistory) {
		r.respondError(w, &fault{409, protocol.APIError{Code: "app_history_changed", Message: "Original app recovery history requires review."}})
		return
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, out)
}
