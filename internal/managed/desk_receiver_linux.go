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

type deskReceiverRuntime interface {
	deskRuntime
	deskApplicationRuntime
	SourceIdentity(context.Context) (string, error)
	Subscription(context.Context, string) (deskwork.Subscription, error)
	ReceiverSourceReady(context.Context, string) (bool, error)
	InstallRemote(context.Context, string, deskwork.Publisher) error
}

func (r *Server) serveDeskReceiverStatus(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "receiver status requires operator access"}})
		return
	}
	var selection protocol.DeskSelection
	if err := decode(w, req, &selection); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	base, err := r.openDesk(ctx, selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	runtime, ok := base.(deskReceiverRuntime)
	if !ok {
		r.respondError(w, bad("receiver adapter unavailable"))
		return
	}
	unavailable := func() {
		r.respondError(w, &fault{502, protocol.APIError{Code: "receiver_status_unavailable", Message: "receiver state unavailable or changed during inspection"}})
	}
	identity, err := runtime.SourceIdentity(ctx)
	if err != nil {
		unavailable()
		return
	}
	subscription, err := runtime.Subscription(ctx, selection.Target.Desk)
	if err != nil {
		unavailable()
		return
	}
	ready, err := runtime.ReceiverSourceReady(ctx, selection.Target.Desk)
	if err != nil {
		unavailable()
		return
	}
	out := protocol.DeskReceiverStatus{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Selection: selection, Subscription: subscription, SourceReady: ready, ReadAt: time.Now().UTC()}
	if ready {
		before, err := runtime.Observe(ctx, selection.Target.Desk)
		if err != nil {
			unavailable()
			return
		}
		app, err := runtime.Application(ctx, selection.Target.Desk)
		if err != nil {
			unavailable()
			return
		}
		after, err := runtime.Observe(ctx, selection.Target.Desk)
		if err != nil || before != after {
			unavailable()
			return
		}
		out.Observation, out.Application = &after, &app
	}
	current, err := runtime.Subscription(ctx, selection.Target.Desk)
	if err != nil || current != subscription {
		unavailable()
		return
	}
	continuity, err := runtime.SourceIdentity(ctx)
	if err != nil || continuity != identity || out.Validate() != nil {
		unavailable()
		return
	}
	respond(w, 200, out)
}

func (r *Server) inspectDeskReceiver(ctx context.Context, q protocol.DeskReceiverRequest) (protocol.DeskReceiverReview, deskReceiverRuntime, error) {
	var out protocol.DeskReceiverReview
	if q.Validate() != nil {
		return out, nil, bad("invalid receiver selection")
	}
	// Resolve project publisher identity from this runner, not a caller's name.
	targets, err := appPolicyTargets(r.s.db)
	if err != nil {
		return out, nil, err
	}
	receiverFound, publisherFound := false, q.Selection.Target.ProjectID == ""
	for _, target := range targets {
		if target.Target == q.Selection.Target.Ship() && target.Ready {
			if target.Ship == q.Publisher.Ship {
				return out, nil, bad("receiver cannot subscribe to itself")
			}
			receiverFound = true
		}
		if target.Target == q.PublisherTarget && target.Ready && target.Ship == q.Publisher.Ship {
			publisherFound = true
		}
	}
	if !receiverFound || !publisherFound {
		return out, nil, conflict("receiver_selection_changed", "receiver or publisher project identity changed")
	}
	base, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		return out, nil, err
	}
	runtime, ok := base.(deskReceiverRuntime)
	if !ok {
		return out, nil, bad("receiver adapter unavailable")
	}
	continuity, err := runtime.SourceIdentity(ctx)
	if err != nil {
		return out, nil, err
	}
	subscription, err := runtime.Subscription(ctx, q.Selection.Target.Desk)
	if err != nil || subscription.Present {
		return out, nil, conflict("receiver_occupied", "receiver name already has a subscription or its state is unavailable")
	}
	before, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || before.Exists {
		return out, nil, conflict("receiver_occupied", "receiver desk must be confirmed absent")
	}
	current, err := runtime.SourceIdentity(ctx)
	if err != nil || current != continuity {
		return out, nil, conflict("runtime_changed", "selected pier changed during receiver inspection")
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
	review := deskwork.ReceiverReview{Version: 1, Publisher: q.Publisher, PublisherTarget: q.PublisherTarget, Observation: before, Subscription: subscription, DeskSetting: desk, ShipSetting: ship, Continuity: continuity}
	if err = review.Seal(); err != nil {
		return out, nil, err
	}
	out = protocol.DeskReceiverReview{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: q, PolicyRevision: policy.Revision, Review: review, ReadAt: time.Now().UTC()}
	return out, runtime, out.Validate()
}
func (r *Server) serveDeskReceiverReview(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "receiver review requires operator access"}})
		return
	}
	var input protocol.DeskReceiverRequest
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	out, _, err := r.inspectDeskReceiver(ctx, input)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, out)
}
func (r *Server) serveDeskReceiver(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "receiver installation requires operator access"}})
		return
	}
	var i deskwork.ReceiverIntent
	if err := decode(w, req, &i); err != nil {
		r.respondError(w, err)
		return
	}
	if err := i.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	if receipt, err := loadDeskOperation(r.s.db, i.Intent.ID); err == nil {
		prior, e := loadDeskReceiver(r.s.db, i.Intent.ID)
		if e != nil || prior.Actor != actor || prior.Intent != i {
			r.respondError(w, conflict("desk_request_conflict", "receiver installation ID already records another intent"))
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
	prepared, _, err := r.inspectDeskReceiver(ctx, protocol.DeskReceiverRequest{Selection: deskSelection(i.Intent), Publisher: i.Publisher, PublisherTarget: i.PublisherTarget})
	if err != nil {
		r.respondError(w, err)
		return
	}
	if i.CheckReview(prepared.Review) != nil || prepared.PolicyRevision != i.Intent.PolicyRevision {
		r.respondError(w, conflict("receiver_changed", "review the current publisher, receiver and workflow settings again"))
		return
	}
	if err = prepared.Review.CheckConfiguredReceiver(); err != nil {
		r.respondError(w, conflict("receiver_not_configured", err.Error()))
		return
	}
	record := deskwork.ReceiverRecord{Version: 1, Intent: i, Review: prepared.Review, Actor: actor, CreatedAt: time.Now().UTC()}
	receipt, fresh, err := r.s.createDeskReceiver(record)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if fresh {
		r.queueDeskReceiver(receipt, record)
	}
	respond(w, 202, r.deskReply(receipt))
}

func (r *Server) queueDeskReceiver(receipt deskwork.Receipt, record deskwork.ReceiverRecord) {
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
		current, runtime, err := r.inspectDeskReceiver(ctx, protocol.DeskReceiverRequest{Selection: deskSelection(i.Intent), Publisher: i.Publisher, PublisherTarget: i.PublisherTarget})
		if err != nil || i.CheckReview(current.Review) != nil || current.Review.CheckConfiguredReceiver() != nil {
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
		if runtime.InstallRemote(ctx, i.Intent.Target.Desk, i.Publisher) != nil {
			step("uncertain", nil, nil)
			return
		}
		if !step("acknowledged", nil, nil) {
			return
		}
		for end := time.Now().Add(90 * time.Second); ; {
			after, app, err := observeDeskReceiver(ctx, runtime, i)
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
func observeDeskReceiver(ctx context.Context, runtime deskReceiverRuntime, i deskwork.ReceiverIntent) (deskwork.Observation, deskwork.AppState, error) {
	var app deskwork.AppState
	ready, err := runtime.ReceiverSourceReady(ctx, i.Intent.Target.Desk)
	if err != nil || !ready {
		return deskwork.Observation{}, app, errors.New("receiver source is still pending or unavailable")
	}
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
		return after, app, errors.New("receiver changed during verification")
	}
	identity, err := runtime.SourceIdentity(ctx)
	if err != nil || identity != i.Continuity {
		return after, app, errors.New("receiver pier changed")
	}
	return after, app, deskwork.VerifyReceiver(i.Publisher, after, app)
}
func (r *Server) queueDeskReceiverReconciliation(receipt deskwork.Receipt, actor string) {
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
		record, err := loadDeskReceiver(r.s.db, receipt.Intent.ID)
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
		runtime, ok := base.(deskReceiverRuntime)
		if !ok {
			return
		}
		after, app, err := observeDeskReceiver(ctx, runtime, record.Intent)
		if err != nil {
			return
		}
		_, err = r.s.appendDeskStep(current, deskwork.Step{Kind: current.Intent.Action, State: "observed", Actor: actor, At: time.Now().UTC(), Observation: &after, Application: &app})
		if err != nil {
			r.failed.Store(true)
		}
	}()
}
