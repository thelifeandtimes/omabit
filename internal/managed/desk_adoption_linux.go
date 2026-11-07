//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (r *Server) inspectDeskAdoption(ctx context.Context, q protocol.DeskDeploymentRequest, actor string) (protocol.DeskAdoptionReview, deskwork.AdoptionPlan, error) {
	var review protocol.DeskAdoptionReview
	var plan deskwork.AdoptionPlan
	prepared, err := r.inspectDeskDeployment(ctx, q, actor)
	if err != nil {
		return review, plan, err
	}
	p := prepared.plan
	plan, err = deskwork.PlanAdoption(p.Previous, p.Current, p.Source, p.Known)
	if err != nil {
		return review, plan, conflict("adoption_unavailable", "stage the selected existing files and inspect their Clay agreement")
	}
	review = protocol.NewDeskAdoptionReview(prepared.review, plan)
	return review, plan, review.Validate()
}
func (r *Server) serveDeskAdoptionReview(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "source adoption reviews require operator access"}})
		return
	}
	var q protocol.DeskDeploymentRequest
	if err := decode(w, req, &q); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	review, _, err := r.inspectDeskAdoption(ctx, q, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, review)
}
func (r *Server) serveDeskAdoption(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "source adoption requires operator access"}})
		return
	}
	var input deskwork.AdoptionIntent
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	if err := input.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	if receipt, err := loadDeskOperation(r.s.db, input.Intent.ID); err == nil {
		prior, e := loadDeskAdoption(r.s.db, input.Intent.ID)
		if e != nil || prior.Actor != actor || prior.Intent != input {
			r.respondError(w, conflict("desk_request_conflict", "adoption ID already records another review"))
			return
		}
		respond(w, 200, r.deskReply(receipt))
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		r.respondError(w, err)
		return
	}
	if !r.routineAuthorized(actor) {
		r.respondError(w, conflict("access_changed", "operator access changed"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	q := protocol.DeskDeploymentRequest{Selection: deskSelection(input.Intent), StageID: input.StageID, SettingsRevision: input.SettingsRevision}
	v, plan, err := r.inspectDeskAdoption(ctx, q, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if v.Observation != input.Intent.Baseline || v.PolicyRevision != input.Intent.PolicyRevision || v.SourceName != input.SourceName || v.Continuity != input.Continuity || v.SourceDigest != input.Intent.SourceDigest || v.PlanDigest != input.PlanDigest {
		r.respondError(w, conflict("adoption_changed", "source or ownership changed after review; inspect again"))
		return
	}
	if v.ConflictCount != 0 {
		r.respondError(w, conflict("source_conflict", "adoption requires every selected file to already match Clay"))
		return
	}
	if v.NewCount+v.RefreshCount == 0 {
		r.respondError(w, conflict("source_unchanged", "the selected source baseline is already owned"))
		return
	}
	record := deskwork.AdoptionRecord{Version: 1, Intent: input, Actor: actor, Plan: plan, CreatedAt: time.Now().UTC()}
	receipt, err := r.s.adoptDeskSource(record)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, r.deskReply(receipt))
}
