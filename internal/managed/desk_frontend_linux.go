//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type deskFrontendRuntime interface {
	deskRuntime
	deskApplicationRuntime
	Ship() string
	SourceIdentity(context.Context) (string, error)
	ClayFiles(context.Context, string, uint64) (deskwork.Manifest, error)
	Frontend(context.Context, string) (deskwork.FrontendState, error)
	FrontendHash(context.Context, deskwork.FrontendManifest, []deskwork.SourceFile) (string, error)
	UploadFrontend(context.Context, deskwork.FrontendReview, deskwork.FrontendManifest, []deskwork.SourceFile) (deskwork.FrontendTransfer, error)
}
type preparedDeskFrontend struct {
	review   protocol.DeskFrontendReview
	runtime  deskFrontendRuntime
	artifact deskwork.FrontendManifest
	files    []deskwork.SourceFile
}

// Caller holds the runner mutation lock. Stages are immutable and discard takes
// the same lock. External native edits cannot be locked by Omabit: inspect both
// sides and never describe this as an atomic native revision precondition.
func (r *Server) inspectDeskFrontend(ctx context.Context, q protocol.DeskFrontendRequest, actor string) (preparedDeskFrontend, error) {
	var out preparedDeskFrontend
	if q.Validate() != nil {
		return out, bad("invalid frontend selection")
	}
	desk, err := r.s.deskSetting(q.Selection.Target)
	if err != nil {
		return out, err
	}
	ship, err := r.s.deskSetting(q.Selection.Target.Ship())
	if err != nil {
		return out, err
	}
	if desk.Revision != q.SettingsRevision || desk.Desk == nil || desk.Desk.Workflow != "mounted" {
		return out, conflict("desk_settings_changed", "review the configured Unix workflow")
	}
	stage, directory, err := deskwork.ReadStage(ctx, filepath.Join(r.s.directory, "desk-stages-v1"), q.StageID, actor)
	if err != nil || stage.Receipt.Intent.Target != q.Selection.Target || stage.Receipt.Intent.SettingsRevision != q.SettingsRevision {
		return out, conflict("source_stage_changed", "verify the selected frontend stage and target")
	}
	artifact, files, err := deskwork.ReadFrontendFiles(ctx, directory, stage.Manifest)
	if err != nil {
		return out, conflict("frontend_artifact_unavailable", err.Error())
	}
	base, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		return out, err
	}
	runtime, ok := base.(deskFrontendRuntime)
	if !ok {
		return out, bad("frontend adapter unavailable")
	}
	continuity, err := runtime.SourceIdentity(ctx)
	if err != nil {
		return out, err
	}
	before, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || !before.Exists {
		return out, conflict("frontend_unavailable", "selected desk is unavailable")
	}
	app, err := runtime.Application(ctx, q.Selection.Target.Desk)
	if err != nil {
		return out, err
	}
	frontend, err := runtime.Frontend(ctx, q.Selection.Target.Desk)
	if err != nil {
		return out, err
	}
	clay, err := runtime.ClayFiles(ctx, q.Selection.Target.Desk, before.Revision)
	if err != nil {
		return out, err
	}
	sourceDigest, err := deskwork.FrontendSourceDigest(clay)
	if err != nil {
		return out, err
	}
	expected, err := runtime.FrontendHash(ctx, artifact, files)
	if err != nil {
		return out, err
	}
	after, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || after != before {
		return out, conflict("frontend_changed", "desk changed during frontend inspection")
	}
	current, err := runtime.SourceIdentity(ctx)
	if err != nil || current != continuity {
		return out, conflict("runtime_changed", "selected pier changed during frontend inspection")
	}
	freshDesk, e := r.s.deskSetting(q.Selection.Target)
	if e != nil || freshDesk.Revision != desk.Revision {
		return out, conflict("desk_settings_changed", "workflow changed during frontend review")
	}
	freshShip, e := r.s.deskSetting(q.Selection.Target.Ship())
	if e != nil || freshShip.Revision != ship.Revision {
		return out, conflict("desk_settings_changed", "ship instructions changed during frontend review")
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		return out, err
	}
	v := deskwork.FrontendReview{Version: 1, Ship: runtime.Ship(), Observation: after, Application: app, Frontend: frontend, ShipSetting: ship, DeskSetting: desk, Continuity: continuity, Stage: stage.Receipt.Intent, ArtifactDigest: artifact.Digest, ExpectedGlobHash: expected, UnchangedSourceDigest: sourceDigest}
	if err = v.Seal(); err != nil {
		return out, conflict("frontend_changed", "source and frontend observations do not agree")
	}
	out = preparedDeskFrontend{review: protocol.DeskFrontendReview{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: q, PolicyRevision: policy.Revision, Review: v, ReadAt: time.Now().UTC()}, runtime: runtime, artifact: artifact, files: files}
	return out, out.review.Validate()
}
func (r *Server) serveDeskFrontendReview(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "frontend review requires operator access"}})
		return
	}
	var input protocol.DeskFrontendRequest
	if err := decode(w, req, &input); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskDeploymentTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskDeploymentTimeout)
	defer cancel()
	prepared, err := r.inspectDeskFrontend(ctx, input, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, prepared.review)
}
func (r *Server) serveDeskFrontend(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "frontend uploads require operator access"}})
		return
	}
	var input protocol.DeskFrontendSubmission
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
		prior, e := loadDeskFrontend(r.s.db, i.Intent.ID)
		if e != nil || prior.Actor != actor || prior.Intent != i {
			r.respondError(w, conflict("desk_request_conflict", "frontend ID already records another review"))
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
	prepared, err := r.inspectDeskFrontend(ctx, protocol.DeskFrontendRequest{Selection: deskSelection(i.Intent), StageID: i.StageID, SettingsRevision: i.SettingsRevision}, actor)
	if err != nil {
		r.respondError(w, err)
		return
	}
	v := prepared.review
	if i.CheckReview(v.Review, input.Checks) != nil || i.Intent.PolicyRevision != v.PolicyRevision {
		r.respondError(w, conflict("frontend_changed", "review the current artifact, source and release checks again"))
		return
	}
	if err = v.Review.CheckConfiguredUpload(); err != nil {
		r.respondError(w, conflict("frontend_not_configured", err.Error()))
		return
	}
	if v.Review.Frontend.Declared.GlobHash == v.Review.ExpectedGlobHash {
		r.respondError(w, conflict("frontend_unchanged", "the selected artifact is already advertised; inspect its frontend status"))
		return
	}
	record := deskwork.FrontendRecord{Version: 1, Intent: i, Review: v.Review, Checks: input.Checks, Actor: actor, CreatedAt: time.Now().UTC()}
	receipt, fresh, err := r.s.createDeskFrontend(record)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if fresh {
		r.queueDeskFrontend(receipt, record)
	}
	respond(w, 202, r.deskReply(receipt))
}
func (r *Server) queueDeskFrontend(receipt deskwork.Receipt, record deskwork.FrontendRecord) {
	r.updateWorkers++
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		defer func() { r.updateWorkers-- }()
		step := func(s deskwork.Step) bool {
			s.Kind = receipt.Intent.Action
			s.At = time.Now().UTC()
			next, err := r.s.appendDeskStep(receipt, s)
			if err != nil {
				r.failed.Store(true)
				return false
			}
			receipt = next
			return true
		}
		if r.closing || r.ctx.Err() != nil {
			step(deskwork.Step{State: "interrupted"})
			return
		}
		if r.failed.Load() || r.updateFence.Held || !r.routineAuthorized(receipt.Actor) {
			step(deskwork.Step{State: "blocked"})
			return
		}
		ctx, cancel := context.WithTimeout(r.ctx, protocol.DeskDeploymentTimeout)
		defer cancel()
		i := record.Intent
		prepared, err := r.inspectDeskFrontend(ctx, protocol.DeskFrontendRequest{Selection: deskSelection(i.Intent), StageID: i.StageID, SettingsRevision: i.SettingsRevision}, receipt.Actor)
		if err != nil || i.CheckReview(prepared.review.Review, record.Checks) != nil || prepared.review.Review.CheckConfiguredUpload() != nil {
			step(deskwork.Step{State: "blocked"})
			return
		}
		policy, err := loadAppPolicy(r.s.db)
		if err != nil {
			r.failed.Store(true)
			return
		}
		if i.Intent.Check(policy, prepared.review.Review.Observation) != nil {
			step(deskwork.Step{State: "blocked"})
			return
		}
		if !step(deskwork.Step{State: "dispatched"}) {
			return
		}
		transfer, _ := prepared.runtime.UploadFrontend(ctx, record.Review, prepared.artifact, prepared.files)
		if transfer.Validate() != nil {
			step(deskwork.Step{State: "uncertain"})
			return
		}
		if !transfer.Acknowledged {
			state := "uncertain"
			if !transfer.Attempted {
				state = "rejected"
			}
			step(deskwork.Step{State: state, FrontendTransfer: &transfer})
			return
		}
		if !step(deskwork.Step{State: "acknowledged", FrontendTransfer: &transfer}) {
			return
		}
		for end := time.Now().Add(30 * time.Second); ; {
			observed, e := observeDeskFrontend(ctx, prepared.runtime, record)
			if e == nil {
				step(observed)
				return
			}
			if time.Now().After(end) || ctx.Err() != nil {
				step(deskwork.Step{State: "uncertain"})
				return
			}
			select {
			case <-ctx.Done():
				step(deskwork.Step{State: "uncertain"})
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}()
}
func observeDeskFrontend(ctx context.Context, runtime deskFrontendRuntime, record deskwork.FrontendRecord) (deskwork.Step, error) {
	fail := func() (deskwork.Step, error) {
		return deskwork.Step{}, errors.New("frontend result remains unverified")
	}
	desk := record.Intent.Intent.Target.Desk
	identity, err := runtime.SourceIdentity(ctx)
	if err != nil || identity != record.Review.Continuity || runtime.Ship() != record.Review.Ship {
		return fail()
	}
	before, err := runtime.Observe(ctx, desk)
	if err != nil || !before.Exists {
		return fail()
	}
	app, err := runtime.Application(ctx, desk)
	if err != nil {
		return fail()
	}
	frontend, err := runtime.Frontend(ctx, desk)
	if err != nil {
		return fail()
	}
	clay, err := runtime.ClayFiles(ctx, desk, before.Revision)
	if err != nil {
		return fail()
	}
	digest, err := deskwork.FrontendSourceDigest(clay)
	if err != nil {
		return fail()
	}
	after, err := runtime.Observe(ctx, desk)
	if err != nil || before != after {
		return fail()
	}
	identity, err = runtime.SourceIdentity(ctx)
	if err != nil || identity != record.Review.Continuity {
		return fail()
	}
	if deskwork.VerifyFrontendResult(record.Review, after, app, frontend, digest) != nil {
		return fail()
	}
	return deskwork.Step{State: "observed", Observation: &after, Application: &app, Frontend: &frontend, VerifiedSourceDigest: record.Review.ArtifactDigest, FrontendSourceDigest: digest}, nil
}
func (r *Server) queueDeskFrontendReconciliation(receipt deskwork.Receipt, actor string) {
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
		record, err := loadDeskFrontend(r.s.db, current.Intent.ID)
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
		runtime, ok := base.(deskFrontendRuntime)
		if !ok {
			return
		}
		step, err := observeDeskFrontend(ctx, runtime, record)
		if err != nil {
			return
		}
		step.Kind = current.Intent.Action
		step.Actor = actor
		step.At = time.Now().UTC()
		if _, err = r.s.appendDeskStep(current, step); err != nil {
			r.failed.Store(true)
		}
	}()
}
