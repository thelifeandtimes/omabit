//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"path/filepath"
	"time"
)

type deskDeploymentRuntime interface {
	deskwork.DeploymentReader
	SourceIdentity(context.Context) (string, error)
	CommitFiles(context.Context, string, deskwork.Observation, []deskwork.DeltaFile) error
}
type preparedDeskDeployment struct {
	runtime deskDeploymentRuntime
	source  string
	review  protocol.DeskDeploymentReview
	plan    deskwork.DeploymentPlan
}

// Caller holds r.mu, serializing configuration, policy and lifecycle admission.
// Stage discard takes that same lock; uploads publish immutable entries only.
func (r *Server) inspectDeskDeployment(ctx context.Context, q protocol.DeskDeploymentRequest, actor string) (preparedDeskDeployment, error) {
	var out preparedDeskDeployment
	if q.Validate() != nil {
		return out, bad("invalid staged deployment selection")
	}
	setting, err := r.s.deskSetting(q.Selection.Target)
	if err != nil {
		return out, err
	}
	if setting.Revision != q.SettingsRevision || setting.Desk == nil || setting.Desk.Workflow != "mounted" {
		return out, conflict("desk_settings_changed", "inspect the selected Unix workflow again")
	}
	stage, source, err := deskwork.ReadStage(ctx, filepath.Join(r.s.directory, "desk-stages-v1"), q.StageID, actor)
	if err != nil || stage.Receipt.Intent.Target != q.Selection.Target || stage.Receipt.Intent.SettingsRevision != q.SettingsRevision {
		return out, conflict("source_stage_changed", "verify the original stage and its target before deployment")
	}
	runtime, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		return out, err
	}
	adapter, ok := runtime.(deskDeploymentRuntime)
	if !ok {
		return out, bad("staged commit adapter unavailable")
	}
	continuity, err := adapter.SourceIdentity(ctx)
	if err != nil {
		return out, conflict("runtime_changed", "selected pier identity unavailable")
	}
	previous, err := r.s.sourceOwnership(q.Selection.Target, setting.Desk.SourceName, continuity)
	if err != nil {
		return out, err
	}
	inspected, err := deskwork.InspectDeployment(ctx, adapter, q.Selection.Target.Desk, source, stage.Manifest, previous)
	if err != nil {
		return out, conflict("deployment_unavailable", "source/Clay comparison unavailable or changed; inspect the staged source and desk")
	}
	after, err := adapter.SourceIdentity(ctx)
	if err != nil || after != continuity {
		return out, conflict("runtime_changed", "selected pier changed during review")
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		return out, err
	}
	review := protocol.NewDeskDeploymentReview(r.s.runnerID, r.s.mode, q, setting.Desk.SourceName, continuity, policy.Revision, inspected)
	if err = review.Validate(); err != nil {
		return out, err
	}
	return preparedDeskDeployment{runtime: adapter, source: source, review: review, plan: inspected.Plan}, nil
}
func (r *Server) serveDeskDeploymentReview(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "deployment reviews require operator access"}})
		return
	}
	var input protocol.DeskDeploymentRequest
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	prepared, err := r.inspectDeskDeployment(ctx, input, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, prepared.review)
}
func (r *Server) serveDeskCommit(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "desk commits require operator access"}})
		return
	}
	var input deskwork.CommitIntent
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	if err := input.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	// Replays return the exact original receipt even after a completed stage was
	// discarded. They must not need another runtime read or enqueue another effect.
	if receipt, err := loadDeskOperation(r.s.db, input.Intent.ID); err == nil {
		prior, e := loadDeskDeployment(r.s.db, input.Intent.ID)
		if e != nil || prior.Actor != actor || prior.Intent != input {
			r.respondError(w, conflict("desk_request_conflict", "commit ID already records another reviewed intent"))
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
	q := protocol.DeskDeploymentRequest{Selection: deskSelection(input.Intent), StageID: input.StageID, SettingsRevision: input.SettingsRevision}
	prepared, err := r.inspectDeskDeployment(ctx, q, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	v := prepared.review
	if v.Observation != input.Intent.Baseline || v.PolicyRevision != input.Intent.PolicyRevision || v.SourceName != input.SourceName || v.Continuity != input.Continuity || v.SourceDigest != input.Intent.SourceDigest || v.PlanDigest != input.PlanDigest {
		r.respondError(w, conflict("deployment_changed", "staged deployment changed after review; inspect again"))
		return
	}
	if v.ConflictCount != 0 {
		r.respondError(w, conflict("source_conflict", "resolve source conflicts before committing"))
		return
	}
	if v.ChangeCount == 0 {
		r.respondError(w, conflict("source_unchanged", "source already matches; no commit is needed"))
		return
	}
	record := deskwork.DeploymentRecord{Version: 1, Intent: input, Actor: actor, Plan: prepared.plan, CreatedAt: time.Now().UTC()}
	receipt, fresh, err := r.s.createDeskDeployment(record)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if fresh {
		r.queueDeskCommit(receipt, record)
	}
	respond(w, 202, r.deskReply(receipt))
}
func (r *Server) queueDeskCommit(receipt deskwork.Receipt, record deskwork.DeploymentRecord) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		step := func(state string) bool {
			next, err := r.s.appendDeskStep(receipt, deskwork.Step{Kind: "commit", State: state, At: time.Now().UTC()})
			if err != nil {
				r.failed.Store(true)
				return false
			}
			receipt = next
			return true
		}
		if r.closing || r.ctx.Err() != nil {
			step("interrupted")
			return
		}
		if r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(receipt.Actor) {
			step("blocked")
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskDeploymentTimeout)
		defer cancel()
		i := record.Intent
		prepared, err := r.inspectDeskDeployment(ctx, protocol.DeskDeploymentRequest{Selection: deskSelection(i.Intent), StageID: i.StageID, SettingsRevision: i.SettingsRevision}, receipt.Actor)
		if err != nil || prepared.review.PlanDigest != i.PlanDigest || prepared.review.Continuity != i.Continuity || prepared.review.SourceName != i.SourceName {
			step("blocked")
			return
		}
		policy, err := loadAppPolicy(r.s.db)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if err = i.Intent.Check(policy, prepared.review.Observation); err != nil {
			step("blocked")
			return
		}
		files, err := deskwork.DeploymentFiles(ctx, prepared.source, record.Plan)
		if err != nil {
			step("blocked")
			return
		}
		if !step("dispatched") {
			return
		}
		if err = prepared.runtime.CommitFiles(ctx, i.Intent.Target.Desk, i.Intent.Baseline, files); err != nil {
			step("uncertain")
			return
		}
		if !step("acknowledged") {
			return
		}
		after, ownership, err := deskwork.ObserveDeploymentResult(ctx, prepared.runtime, i.Intent.Target.Desk, prepared.source, i.Intent.Baseline, record.Plan)
		if err != nil {
			step("uncertain")
			return
		}
		continuity, err := prepared.runtime.SourceIdentity(ctx)
		if err != nil || continuity != i.Continuity {
			step("uncertain")
			return
		}
		if _, err = r.s.completeDeskDeployment(receipt, after, ownership, ""); err != nil {
			r.failed.Store(true)
		}
	}()
}

func (r *Server) queueDeskCommitReconciliation(receipt deskwork.Receipt, actor string) {
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
		record, err := loadDeskDeployment(r.s.db, current.Intent.ID)
		if err != nil {
			r.failed.Store(true)
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskDeploymentTimeout)
		defer cancel()
		runtime, err := r.openDesk(ctx, deskSelection(current.Intent))
		if err != nil {
			return
		}
		adapter, ok := runtime.(deskDeploymentRuntime)
		if !ok {
			return
		}
		continuity, err := adapter.SourceIdentity(ctx)
		if err != nil || continuity != record.Intent.Continuity {
			return
		}
		stage, source, err := deskwork.ReadStage(ctx, filepath.Join(r.s.directory, "desk-stages-v1"), record.Intent.StageID, record.Actor)
		if err != nil || stage.Manifest.Digest != record.Plan.Source.Digest {
			return
		}
		after, ownership, err := deskwork.ObserveDeploymentResult(ctx, adapter, current.Intent.Target.Desk, source, current.Intent.Baseline, record.Plan)
		if err != nil {
			return
		}
		continuity, err = adapter.SourceIdentity(ctx)
		if err != nil || continuity != record.Intent.Continuity {
			return
		}
		if _, err = r.s.completeDeskDeployment(current, after, ownership, actor); err != nil {
			r.failed.Store(true)
		}
	}()
}
