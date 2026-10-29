//go:build linux && amd64

package managed

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"slices"
)

func (r *Server) projectRecoveryActionFeatures() []string {
	var features []string
	if _, ok := r.b.(projectAnchorRecoveryBackend); ok {
		features = append(features, "project-anchor-recovery-v1")
	}
	if _, ok := r.b.(projectRecoveryObservationBackend); ok {
		features = append(features, "project-recovery-observe-v1")
	}
	if _, ok := r.b.(projectRecoveryContinuationBackend); ok {
		features = append(features, "project-recovery-continue-v1")
	}
	return features
}
func (r *Server) serveProjectRecoveryAction(w http.ResponseWriter, q *http.Request, actor, role, project, parent, kind string) {
	if r.projectExpanded(project) {
		r.respondError(w, conflict("project_expanded", "expanded project recovery requires its full member ledger; initial history cannot mutate it"))
		return
	}

	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "recovery actions require operator access"}})
		return
	}
	replay, err := protocol.CreationReplayOnly(q.Header)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	var body protocol.ProjectRecoveryActionRequest
	if err = decode(w, q, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !protocol.ValidProjectRecoveryAction(kind, parent, body) {
		r.respondError(w, bad("require an explicitly confirmed recovery action and exact reviewed history/checkpoint"))
		return
	}
	feature, _ := protocol.ProjectRecoveryAction(kind)
	if !slices.Contains(r.projectRecoveryActionFeatures(), feature) {
		r.respondError(w, conflict("project_recovery_unsupported", "runner backend does not support this recovery action"))
		return
	}
	var op protocol.Operation
	var hit bool
	if kind == "start" {
		op, hit, err = r.queueProjectAnchorRecovery(actor, project, projectAnchorRecoveryRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, HistoryDigest: body.HistoryDigest, ReplayOnly: replay})
	} else {
		req := projectRecoveryObservationRequest{RequestID: body.RequestID, RecoveryOperationID: parent, ExpectedRevision: body.ExpectedRevision, ExpectedSequence: *body.ExpectedSequence, ReviewDigest: body.ReviewDigest, HistoryDigest: body.HistoryDigest, CheckpointDigest: body.CheckpointDigest, ReplayOnly: replay}
		if kind == "observe" {
			op, hit, err = r.queueProjectRecoveryObservation(actor, project, req)
		} else {
			op, hit, err = r.queueProjectRecoveryContinuation(actor, project, req)
		}
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := http.StatusAccepted
	if hit {
		status = http.StatusOK
	}
	if kind == "start" {
		parent = op.ID
	}
	respond(w, status, protocol.ProjectRecoveryActionResult{ProjectActionResult: protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, ReviewDigest: body.ReviewDigest, Operation: op}, RecoveryOperationID: parent, HistoryDigest: body.HistoryDigest, ExpectedSequence: body.ExpectedSequence, CheckpointDigest: body.CheckpointDigest})
}
