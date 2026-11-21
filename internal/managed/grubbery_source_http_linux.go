//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Server) grubberyAppSourceAvailable() bool { _, ok := r.b.(deskBackend); return ok }

func decodeGrubberySource(w http.ResponseWriter, req *http.Request, out any) error {
	media, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return bad("require application/json")
	}
	req.Body = http.MaxBytesReader(w, req.Body, protocol.GrubberyAppRequestBytes)
	defer req.Body.Close()
	d := json.NewDecoder(req.Body)
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return bad("app request must match its bounded JSON schema")
	}
	return nil
}

func appCall(r *Server, p grubbery.Plan, actor, step string) grubbery.Call {
	return grubbery.Call{RunnerID: r.s.runnerID, Mode: r.s.mode, Actor: actor, Target: p.Binding.Target, AppID: p.Binding.AppID, RequestID: p.RequestID, PlanDigest: p.Digest, StepID: step}
}

func (r *Server) respondGrubberyError(w http.ResponseWriter, err error) {
	code, status, message := "app_source_unavailable", 409, "App source evidence or admission is unavailable; inspect the original saved receipt."
	switch {
	case errors.Is(err, grubbery.ErrMissing):
		code, status, message = "app_operation_not_found", 404, "App operation not found."
	case errors.Is(err, grubbery.ErrConflict), errors.Is(err, grubbery.ErrCAS):
		code, message = "app_operation_changed", "Original app operation binding changed."
	case errors.Is(err, grubbery.ErrPhase):
		code, status, message = "app_phase_changed", 400, "Select the exact original app phase."
	case errors.Is(err, grubbery.ErrSource):
		code, message = "app_source_changed", "Exact access-bound staged source is unavailable."
	case errors.Is(err, grubbery.ErrJournal):
		code, status, message = "app_journal_uncertain", 503, "Journal outcome is unconfirmed; inspect the original saved receipt."
	case errors.Is(err, grubbery.ErrLegacyPlan):
		code, message = "app_legacy_plan", "Original legacy plan is read-only; inspect recovery or preactivation abandonment."
	case errors.Is(err, grubbery.ErrNamespaceBusy):
		code, message = "app_namespace_busy", "Use the original unfinished app operation."
	case errors.Is(err, grubbery.ErrHistory):
		code, message = "app_history_changed", "Original app history requires review."
	}
	r.respondError(w, &fault{status, protocol.APIError{Code: code, Message: message}})
}

func (r *Server) serveGrubberyAppSource(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "App source operations require operator access."}})
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.GrubberyAppTimeout + time.Second))
	base := req.Context()
	if req.URL.Path != protocol.GrubberyAppReviewPath {
		// Exact journal/phase requests belong to the runner after admission.
		// Disconnecting a caller cancels its wait, not an already accepted phase.
		// The original IDs, durable dispatch and native four-minute bound remain.
		base = r.ctx
	}
	ctx, cancel := context.WithTimeout(base, protocol.GrubberyAppTimeout)
	defer cancel()
	// Preserve existing lifecycle/config/policy serialization. Saved GET lookup
	// has its separate unlocked path. Every effect starts with its exact record.
	r.mu.Lock()
	defer r.mu.Unlock()
	j := grubberyJournal{r.s}
	e := grubbery.Execution{Store: j}
	var out grubbery.Record
	var err error
	if req.URL.Path == protocol.GrubberyAppReviewPath {
		var q protocol.GrubberyAppReviewRequest
		if err = decodeGrubberySource(w, req, &q); err != nil {
			r.respondError(w, err)
			return
		}
		if q.Validate() != nil {
			r.respondError(w, bad("invalid app review selection"))
			return
		}
		review, err := r.reviewGrubberyAppSource(ctx, q, actor, role)
		if err != nil {
			r.respondGrubberyError(w, err)
			return
		}
		respond(w, 200, review)
		return
	}
	if req.URL.Path == protocol.GrubberyAppOperationsPath {
		var q protocol.GrubberyAppBeginRequest
		if err = decodeGrubberySource(w, req, &q); err != nil {
			r.respondError(w, err)
			return
		}
		call := appCall(r, q.Plan, actor, "")
		// Replay precedes cache, process, policy and management admission. A
		// missing record cannot be created by this backend-free attempt.
		out, err = e.Begin(ctx, call, q.Plan)
		if errors.Is(err, grubbery.ErrAdmission) {
			b, makeErr := r.nativeGrubberyBackend(ctx, q.Plan.Binding, actor, role)
			if makeErr != nil {
				r.respondGrubberyError(w, makeErr)
				return
			}
			out, err = b.Execution(j).Begin(ctx, call, q.Plan)
		}
	} else {
		id, action, ok := protocol.GrubberyAppPhasePath(req.URL.Path)
		if !ok {
			r.respondError(w, bad("invalid app phase path"))
			return
		}
		var q protocol.GrubberyAppPhaseRequest
		if err = decodeGrubberySource(w, req, &q); err != nil {
			r.respondError(w, err)
			return
		}
		if q.Validate() != nil {
			r.respondError(w, bad("invalid original app phase selection"))
			return
		}
		original, readErr := j.Load(ctx, id)
		if readErr != nil {
			r.respondGrubberyError(w, readErr)
			return
		}
		call := appCall(r, original.Plan, actor, q.StepID)
		call.PlanDigest = q.PlanDigest
		if _, err = e.Lookup(ctx, call); err != nil {
			r.respondGrubberyError(w, err)
			return
		}
		switch action {
		case "advance":
			out, err = e.Advance(ctx, call)
		case "reconcile":
			out, err = e.Reconcile(ctx, call)
		case "abandon":
			if original.Abandoned() {
				out = original
				break
			}
			// Even a zero-effect abandonment writes the journal and must respect
			// management quiescence and current transport authorization.
			if r.closing || r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(actor) {
				err = grubbery.ErrAdmission
				break
			}
			out, err = e.Abandon(ctx, call)
		}
		if errors.Is(err, grubbery.ErrAdmission) {
			b, makeErr := r.nativeGrubberyBackendFor(ctx, original, actor, role)
			if makeErr != nil {
				r.respondGrubberyError(w, makeErr)
				return
			}
			e = b.Execution(j)
			switch action {
			case "advance":
				out, err = e.Advance(ctx, call)
			case "reconcile":
				out, err = e.Reconcile(ctx, call)
			case "abandon":
				out, err = e.Abandon(ctx, call)
			}
		}
	}
	if err != nil {
		r.respondGrubberyError(w, err)
		return
	}
	respond(w, 200, protocol.GrubberyAppOperation{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Record: out})
}

func (r *Server) reviewGrubberyAppSource(ctx context.Context, q protocol.GrubberyAppReviewRequest, actor, role string) (protocol.GrubberyAppReview, error) {
	var out protocol.GrubberyAppReview
	if _, err := (grubberyJournal{r.s}).Load(ctx, q.RequestID); !errors.Is(err, grubbery.ErrMissing) {
		if err == nil {
			err = grubbery.ErrConflict
		}
		return out, err
	}
	framework, err := loadDeskDeployment(r.s.db, q.FrameworkReceiptID)
	if err != nil {
		return out, grubbery.ErrHistory
	}
	receipt, err := loadDeskOperation(r.s.db, q.FrameworkReceiptID)
	if err != nil || len(receipt.Steps) == 0 || receipt.Steps[len(receipt.Steps)-1].Observation == nil {
		return out, grubbery.ErrHistory
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		return out, err
	}
	b := grubbery.Binding{RunnerID: r.s.runnerID, Mode: r.s.mode, Actor: actor, Target: q.Selection.Target, RuntimeID: q.Selection.RuntimeID, ConfigRevision: q.Selection.ConfigRevision, PolicyRevision: policy.Revision, Continuity: framework.Intent.Continuity, FrameworkReceiptID: q.FrameworkReceiptID, FrameworkBaseline: *receipt.Steps[len(receipt.Steps)-1].Observation, AppID: q.AppID}
	backend, err := r.nativeGrubberyBackend(ctx, b, actor, role)
	if err != nil {
		return out, err
	}
	manifest, files, err := r.grubberyStageSource(ctx, b, q.StageID, actor)
	if err != nil {
		return out, err
	}
	before, err := backend.Inspect(ctx, b)
	if err != nil {
		return out, err
	}
	plan, err := grubbery.NewPlan(b, q.StageID, q.RequestID, manifest, files, before)
	if err != nil {
		return out, grubbery.ErrSource
	}
	wire, err := json.Marshal(protocol.GrubberyAppBeginRequest{Plan: plan})
	if err != nil || len(wire) > protocol.GrubberyAppRequestBytes {
		return out, grubbery.ErrSource
	}
	out = protocol.GrubberyAppReview{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: q, Plan: plan, ReadAt: time.Now().UTC()}
	return out, out.Validate()
}
