//go:build linux && amd64

package managed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type projectStopRecoveryBackend interface {
	ExpandedProjectStopObservation(context.Context, runtimequalify.ProjectExpansion, string) (protocol.ProjectStopObservation, error)
}

func (realBackend) ExpandedProjectStopObservation(ctx context.Context, s runtimequalify.ProjectExpansion, id string) (protocol.ProjectStopObservation, error) {
	return runtimequalify.ObserveExpandedProjectStop(ctx, s, id)
}

type projectStopSelection struct {
	state      runtimequalify.ProjectExpansion
	member     runtimequalify.ProjectExpandedMember
	original   protocol.Operation
	ledgerBody []byte
	review     protocol.ProjectStopReview
}

func (s *store) projectStopSelection(q querier, actor, project, member, operation string) (projectStopSelection, error) {
	var out projectStopSelection
	op, err := loadOperation(q, operation)
	if err != nil {
		return out, err
	}
	if op.ProjectID != project || op.InstanceID != member {
		return out, conflict("stop_target_changed", "operation does not belong to this exact project member")
	}
	out.original = op
	if err = q.QueryRow("SELECT body FROM project_expansions WHERE project_id=?", project).Scan(&out.ledgerBody); err != nil {
		return out, err
	}
	out.state, err = loadProjectExpansion(q, project)
	if err != nil {
		return out, err
	}
	for _, m := range out.state.Members {
		if m.Member.InstanceID == member {
			out.member = m
		}
	}
	if op.Kind != "stop-project-ship" || op.Status != "interrupted" || out.member.Phase != "stop-dispatched" {
		return out, conflict("stop_recovery_unsupported_phase", "only an interrupted expanded-member stop-dispatched phase has this recovery path")
	}
	if op.Actor != actor {
		return out, conflict("stop_recovery_actor_mismatch", "reconciliation requires the operator that submitted this stop")
	}
	if out.member.Initialized.Identity.Inode == 0 || out.member.Initialized.Identity.UrbInode == 0 {
		return out, conflict("stop_recovery_evidence_missing", "original stop lacks a retained initialized-pier binding")
	}
	p, err := loadProject(q, project)
	if err != nil {
		return out, err
	}
	if p.Revision != op.ConfigRevision || out.state.Project.ProjectID != project || out.state.Project.RunnerID != s.runnerID || out.state.Project.Revision != p.Revision {
		return out, conflict("stop_target_changed", "project revision or runner binding changed")
	}
	planDigest, err := p.Plan.Digest()
	if err != nil {
		return out, err
	}
	ledgerPlanDigest, err := out.state.Project.Plan.Digest()
	if err != nil {
		return out, err
	}
	if planDigest != ledgerPlanDigest {
		return out, conflict("stop_target_changed", "saved project plan changed without a matching ledger")
	}
	var originalBody []byte
	var storedActor, storedRequest, fingerprint string
	if err = q.QueryRow("SELECT body,actor,request_id,fingerprint FROM operations WHERE id=?", operation).Scan(&originalBody, &storedActor, &storedRequest, &fingerprint); err != nil {
		return out, err
	}
	var input struct {
		Project string
		Request protocol.ProjectActionRequest
	}
	if json.Unmarshal([]byte(fingerprint), &input) != nil || !protocol.ValidProjectActionRequest(input.Request) || input.Project != project || input.Request.Kind != "member-stop" || input.Request.InstanceID != member || input.Request.ContainerID != out.member.Creator.Container.ID || input.Request.ExpectedRevision != p.Revision || input.Request.ReviewDigest != planDigest || input.Request.RequestID != op.RequestID || storedActor != actor || storedRequest != op.RequestID || op.ID != operation {
		return out, conflict("stop_original_request_changed", "original stop request no longer matches this exact container and operator")
	}
	if _, err = loadProjectStopResolution(q, operation); err == nil {
		return out, conflict("stop_already_reconciled", "original stop already has a reconciliation receipt")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	basis := "legacy-admission-window"
	if intent := out.member.StopIntent; intent != nil {
		if intent.OperationID != op.ID || intent.ContainerID != out.member.Creator.Container.ID || intent.RecordedAt.Before(op.CreatedAt) || intent.StartedAt.IsZero() || intent.RecordedAt.Before(intent.StartedAt) {
			return out, conflict("stop_recovery_evidence_changed", "retained stop dispatch binding changed")
		}
		basis = "retained-stop-intent"
	}
	out.review = protocol.ProjectStopReview{Version: 1, RunnerID: s.runnerID, Mode: s.mode, Actor: actor, ProjectID: project, ProjectRevision: p.Revision, ProjectGeneration: out.state.Project.Generation, InstanceID: member, ContainerGeneration: out.member.Creator.Generation, OriginalOperationID: op.ID, OperationDigest: stopEvidenceDigest(originalBody), OriginalRequestDigest: stopEvidenceDigest([]byte(fingerprint)), LedgerDigest: stopEvidenceDigest(out.ledgerBody), EvidenceBasis: basis}
	return out, nil
}

func (s projectStopSelection) observedReview(o protocol.ProjectStopObservation) (protocol.ProjectStopReview, error) {
	r := s.review
	if o.ContainerID != s.member.Creator.Container.ID || o.ContractDigest != s.member.Creator.Container.ContractDigest || !reflect.DeepEqual(o.Storage, s.member.Initialized) || o.EngineID != s.state.Anchor.Prepared.EngineID || o.AnchorID != s.state.Anchor.Prepared.Anchor.ID || o.BridgeID != s.state.Anchor.Prepared.Bridge.ID || s.state.Anchor.Namespace == nil || o.BootID != s.state.Anchor.Namespace.BootID || o.NamespaceDevice != s.state.Anchor.Namespace.Device || o.NamespaceInode != s.state.Anchor.Namespace.Inode {
		return r, conflict("stop_recovery_identity_changed", "fresh container, pier, engine or namespace does not match retained ownership")
	}
	if o.Running {
		return r, conflict("stop_member_running", "member is still running; reconciliation does not send another signal")
	}
	if o.OOMKilled || o.ExitCode != 0 {
		return r, conflict("stop_exit_abnormal", "OOM or nonzero exit needs a separately supported recovery policy")
	}
	if o.StartedAt.After(s.original.CreatedAt) || o.FinishedAt.Before(s.original.CreatedAt) || s.original.FinishedAt == nil || o.ObservedAt.Before(*s.original.FinishedAt) {
		return r, conflict("stop_process_changed", "process timestamps do not bracket the original stop admission")
	}
	if intent := s.member.StopIntent; intent != nil && (!o.StartedAt.Equal(intent.StartedAt) || o.FinishedAt.Before(intent.RecordedAt)) {
		return r, conflict("stop_process_changed", "process no longer matches the retained pre-signal stop intent")
	}
	r.Observation = o
	r.ReviewDigest = r.Digest()
	if r.Validate() != nil {
		return r, conflict("stop_recovery_evidence_missing", "fresh observation lacks complete clean-exit evidence")
	}
	return r, nil
}

func (r *Server) serveProjectMemberDiagnosis(w http.ResponseWriter, q *http.Request, actor, role, project, member, operation string, probe bool) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "exact member diagnosis requires operator access"}})
		return
	}
	if probe {
		if err := r.beginUpdateBulk(); err != nil {
			r.respondError(w, err)
			return
		}
		defer r.endUpdateBulk()
	}
	r.mu.Lock()
	selected, err := r.s.projectStopSelection(r.s.db, actor, project, member, operation)
	r.mu.Unlock()
	if selected.original.ID == "" {
		r.respondError(w, err)
		return
	}
	// The original free-form diagnostic remains in its immutable receipt. This
	// bounded public result uses fixed messages, never host-supplied text/paths.
	op := selected.original
	op.Message = ""
	out := protocol.ProjectMemberDiagnosis{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Actor: actor, ProjectID: project, InstanceID: member, Operation: op, Phase: selected.member.Phase, Fence: "original-interrupted-operation", ReasonCode: "stop_needs_observation", Message: "Review the exact stopped process before reconciling this interruption.", RuntimeObservation: "not-performed", ReadAt: time.Now().UTC()}
	if resolution, e := loadProjectStopResolution(r.s.db, operation); e == nil {
		out.ResolvedBy = resolution.Result.Operation.ID
		out.Fence = "resolved"
		out.ReasonCode = "stop_reconciled"
		out.Message = "Stopped state was verified by a separate receipt; ordinary member-start is a separate action."
		respond(w, 200, out)
		return
	} else if !errors.Is(e, sql.ErrNoRows) {
		r.respondError(w, e)
		return
	}
	setReason := func(err error) {
		var f *fault
		if errors.As(err, &f) {
			out.ReasonCode = f.problem.Code
			out.Message = f.problem.Message
		} else {
			out.ReasonCode = "stop_observation_unavailable"
			out.Message = "Exact ownership or process evidence is unavailable; no recovery authority was established."
		}
	}
	if err != nil {
		setReason(err)
		respond(w, 200, out)
		return
	}
	if !probe {
		respond(w, 200, out)
		return
	}
	b, ok := r.b.(projectStopRecoveryBackend)
	if !ok {
		out.ReasonCode = "stop_recovery_adapter_unavailable"
		out.Message = "This runner has no supported expanded-member stop observer."
		respond(w, 200, out)
		return
	}
	ctx, cancel := context.WithTimeout(q.Context(), 30*time.Second)
	defer cancel()
	o, err := b.ExpandedProjectStopObservation(ctx, selected.state, member)
	if err != nil {
		setReason(err)
		out.RuntimeObservation = "unavailable"
		respond(w, 200, out)
		return
	}
	out.Observation = &o
	out.RuntimeObservation = "fresh"
	review, err := selected.observedReview(o)
	if err != nil {
		setReason(err)
	} else {
		out.Review = &review
		out.ReasonCode = "stop_clean_exit_verified"
		out.Message = "Review these exact bindings, then explicitly reconcile; no signal or Start has been dispatched."
	}
	respond(w, 200, out)
}

func (r *Server) serveProjectStopReconcile(w http.ResponseWriter, q *http.Request, actor, role, project string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "stop reconciliation requires operator access"}})
		return
	}
	var req protocol.ProjectStopReconcileRequest
	if err := decode(w, q, &req); err != nil {
		r.respondError(w, err)
		return
	}
	replayOnly, err := protocol.CreationReplayOnly(q.Header)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	if !protocol.ValidID(req.RequestID) || !req.ConfirmAction || req.Review.Validate() != nil || req.Review.RunnerID != r.s.runnerID || req.Review.Mode != r.s.mode || req.Review.Actor != actor || req.Review.ProjectID != project {
		r.respondError(w, bad("require this operator's exact stop review, request ID and explicit confirmation"))
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if result, hit, err := r.s.replayProjectStopResolution(actor, project, req); hit || err != nil {
		if err != nil {
			r.respondError(w, err)
		} else {
			respond(w, 200, result)
		}
		return
	}
	if replayOnly {
		r.respondError(w, conflict("replay_not_found", "no matching stop reconciliation receipt"))
		return
	}
	if r.closing || r.failed.Load() || r.ctx.Err() != nil || r.updateFence.Held {
		r.respondError(w, conflict("runner_unavailable", "runner is unavailable for reconciliation"))
		return
	}
	b, ok := r.b.(projectStopRecoveryBackend)
	if !ok {
		r.respondError(w, conflict("stop_recovery_adapter_unavailable", "runner cannot observe the selected stop"))
		return
	}
	selected, err := r.s.projectStopSelection(r.s.db, actor, project, req.Review.InstanceID, req.Review.OriginalOperationID)
	if err != nil {
		r.respondError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(q.Context(), 30*time.Second)
	defer cancel()
	o, err := b.ExpandedProjectStopObservation(ctx, selected.state, req.Review.InstanceID)
	if err != nil {
		r.respondError(w, conflict("stop_observation_unavailable", "fresh exact ownership and exit evidence could not be established"))
		return
	}
	fresh, err := selected.observedReview(o)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	result, err := r.s.commitProjectStopResolution(actor, project, req, fresh)
	if err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, result)
}
