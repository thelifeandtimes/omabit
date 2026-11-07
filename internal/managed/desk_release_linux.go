//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type deskReleaseRuntime interface {
	deskRuntime
	deskApplicationRuntime
	SourceIdentity(context.Context) (string, error)
	Release(context.Context, string, string, deskwork.Observation) error
}

func (r *Server) inspectDeskRelease(ctx context.Context, q protocol.DeskReleaseRequest) (protocol.DeskReleaseReview, deskReleaseRuntime, error) {
	var out protocol.DeskReleaseReview
	if q.Validate() != nil {
		return out, nil, bad("invalid release selection")
	}
	base, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		return out, nil, err
	}
	runtime, ok := base.(deskReleaseRuntime)
	if !ok {
		return out, nil, bad("release adapter unavailable")
	}
	continuity, err := runtime.SourceIdentity(ctx)
	if err != nil {
		return out, nil, err
	}
	before, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || !before.Exists {
		return out, nil, conflict("release_unavailable", "selected desk is unavailable")
	}
	app, err := runtime.Application(ctx, q.Selection.Target.Desk)
	if err != nil {
		return out, nil, conflict("release_unavailable", "application stage observation unavailable")
	}
	after, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || before != after || app.SourceRevision != after.Revision {
		return out, nil, conflict("release_changed", "desk changed during release inspection")
	}
	current, err := runtime.SourceIdentity(ctx)
	if err != nil || current != continuity {
		return out, nil, conflict("runtime_changed", "selected pier changed during release inspection")
	}
	desk, err := r.s.deskSetting(q.Selection.Target)
	if err != nil {
		return out, nil, err
	}
	ship, err := r.s.deskSetting(q.Selection.Target.Ship())
	if err != nil {
		return out, nil, err
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		return out, nil, err
	}
	review := deskwork.ReleaseReview{Version: 1, Action: q.Action, Observation: after, Application: app, DeskSetting: desk, ShipSetting: ship, Continuity: continuity}
	if err = review.Seal(); err != nil {
		return out, nil, err
	}
	out = protocol.DeskReleaseReview{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: q, PolicyRevision: policy.Revision, Review: review, ReadAt: time.Now().UTC()}
	return out, runtime, out.Validate()
}
func (r *Server) serveDeskReleaseReview(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "release review requires operator access"}})
		return
	}
	var input protocol.DeskReleaseRequest
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	out, _, err := r.inspectDeskRelease(ctx, input)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, out)
}
func (r *Server) serveDeskRelease(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "release effects require operator access"}})
		return
	}
	var input protocol.DeskReleaseSubmission
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	if err := input.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	i := input.Intent
	if receipt, err := loadDeskOperation(r.s.db, i.Intent.ID); err == nil {
		prior, e := loadDeskRelease(r.s.db, i.Intent.ID)
		if e != nil || prior.Actor != actor || prior.Intent != i {
			r.respondError(w, conflict("desk_request_conflict", "release ID already records another intent"))
			return
		}
		respond(w, 200, r.deskReply(receipt))
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	prepared, _, err := r.inspectDeskRelease(ctx, protocol.DeskReleaseRequest{Selection: deskSelection(i.Intent), Action: i.Intent.Action})
	if err != nil {
		r.respondError(w, err)
		return
	}
	if i.CheckReview(prepared.Review, input.Checks) != nil || prepared.PolicyRevision != i.Intent.PolicyRevision {
		r.respondError(w, conflict("release_changed", "review the current source, workflow and check evidence again"))
		return
	}
	if err = prepared.Review.CheckConfiguredRelease(); err != nil {
		r.respondError(w, conflict("release_not_configured", err.Error()))
		return
	}
	if deskwork.VerifyRelease(i.Intent.Action, i.Intent.Baseline, prepared.Review.Observation, prepared.Review.Application) == nil {
		r.respondError(w, conflict("release_unchanged", "requested stage is already observed; inspect its status"))
		return
	}
	record := deskwork.ReleaseRecord{Version: 1, Intent: i, Review: prepared.Review, Checks: input.Checks, Actor: actor, CreatedAt: time.Now().UTC()}
	receipt, fresh, err := r.s.createDeskRelease(record)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if fresh {
		r.queueDeskRelease(receipt, record)
	}
	respond(w, 202, r.deskReply(receipt))
}

func (r *Server) queueDeskRelease(receipt deskwork.Receipt, record deskwork.ReleaseRecord) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		step := func(state string, observation *deskwork.Observation, app *deskwork.AppState) bool {
			next, err := r.s.appendDeskStep(receipt, deskwork.Step{Kind: receipt.Intent.Action, State: state, At: time.Now().UTC(), Observation: observation, Application: app})
			if err != nil {
				r.failed.Store(true)
				return false
			}
			receipt = next
			return true
		}
		if r.closing || r.ctx.Err() != nil {
			step("interrupted", nil, nil)
			return
		}
		if r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(receipt.Actor) {
			step("blocked", nil, nil)
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskDeploymentTimeout)
		defer cancel()
		i := record.Intent
		current, runtime, err := r.inspectDeskRelease(ctx, protocol.DeskReleaseRequest{Selection: deskSelection(i.Intent), Action: i.Intent.Action})
		if err != nil || i.CheckReview(current.Review, record.Checks) != nil || current.Review.CheckConfiguredRelease() != nil {
			step("blocked", nil, nil)
			return
		}
		policy, err := loadAppPolicy(r.s.db)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if i.Intent.Check(policy, current.Review.Observation) != nil {
			step("blocked", nil, nil)
			return
		}
		if !step("dispatched", nil, nil) {
			return
		}
		if runtime.Release(ctx, i.Intent.Target.Desk, i.Intent.Action, i.Intent.Baseline) != nil {
			step("uncertain", nil, nil)
			return
		}
		if !step("acknowledged", nil, nil) {
			return
		}
		for end := time.Now().Add(30 * time.Second); ; {
			after, app, err := observeDeskRelease(ctx, runtime, i)
			if err == nil {
				step("observed", &after, &app)
				return
			}
			if time.Now().After(end) || ctx.Err() != nil {
				step("uncertain", nil, nil)
				return
			}
			select {
			case <-ctx.Done():
				step("uncertain", nil, nil)
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}()
}
func observeDeskRelease(ctx context.Context, runtime deskReleaseRuntime, i deskwork.ReleaseIntent) (deskwork.Observation, deskwork.AppState, error) {
	var app deskwork.AppState
	before, err := runtime.Observe(ctx, i.Intent.Target.Desk)
	if err != nil {
		return before, app, err
	}
	app, err = runtime.Application(ctx, i.Intent.Target.Desk)
	if err != nil {
		return before, app, err
	}
	after, err := runtime.Observe(ctx, i.Intent.Target.Desk)
	if err != nil || before != after {
		return after, app, errors.New("release changed during verification")
	}
	identity, err := runtime.SourceIdentity(ctx)
	if err != nil || identity != i.Continuity {
		return after, app, errors.New("release pier changed")
	}
	return after, app, deskwork.VerifyRelease(i.Intent.Action, i.Intent.Baseline, after, app)
}
func (r *Server) queueDeskReleaseReconciliation(receipt deskwork.Receipt, actor string) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		if r.closing || r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(actor) {
			return
		}
		current, err := loadDeskOperation(r.s.db, receipt.Intent.ID)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if current.Steps[len(current.Steps)-1].State != "uncertain" {
			return
		}
		record, err := loadDeskRelease(r.s.db, receipt.Intent.ID)
		if err != nil {
			r.failed.Store(true)
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskDeploymentTimeout)
		defer cancel()
		base, err := r.openDesk(ctx, deskSelection(current.Intent))
		if err != nil {
			return
		}
		runtime, ok := base.(deskReleaseRuntime)
		if !ok {
			return
		}
		after, app, err := observeDeskRelease(ctx, runtime, record.Intent)
		if err != nil {
			return
		}
		_, err = r.s.appendDeskStep(current, deskwork.Step{Kind: current.Intent.Action, State: "observed", Actor: actor, At: time.Now().UTC(), Observation: &after, Application: &app})
		if err != nil {
			r.failed.Store(true)
		}
	}()
}
