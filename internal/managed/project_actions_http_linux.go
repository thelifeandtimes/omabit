//go:build linux && amd64

package managed

import (
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (r *Server) projectActionFeatures() []string {
	var out []string
	if _, ok := r.b.(projectExpansionBackend); ok {
		out = append(out, "project-add-member-v1")
	}
	if _, ok := r.b.(projectPreparationBackend); ok {
		out = append(out, "project-prepare-v1")
	}
	if _, ok := r.b.(projectAnchorStartBackend); ok {
		out = append(out, "project-anchor-start-v1")
	}
	if _, ok := r.b.(projectMemberStorageBackend); ok {
		out = append(out, "project-member-storage-v1")
	}
	if _, ok := r.b.(projectMemberLaunchBackend); ok {
		out = append(out, "project-member-launch-v1")
	}
	if _, ok := r.b.(projectMemberReadinessBackend); ok {
		out = append(out, "project-member-observe-v1")
	}

	if _, ok := r.b.(projectMemberStopBackend); ok {
		out = append(out, "project-member-stop-v1")
	}
	if _, ok := r.b.(projectMemberStopExitBackend); ok {
		out = append(out, "project-member-exit-v1")
	}
	if _, ok := r.b.(projectMemberRetirementBackend); ok {
		out = append(out, "project-member-retire-v1")
	}
	if _, ok := r.b.(projectMemberResumeBackend); ok {
		out = append(out, "project-member-resume-v1")
	}
	if _, ok := r.b.(projectMemberResumedReadinessBackend); ok {
		out = append(out, "project-member-resumed-observe-v1")
	}
	if _, ok := r.b.(projectMemberContextReadinessBackend); ok {
		out = append(out, "project-member-current-observe-v1")
	}

	return out
}

// Called before the ordinary mutation lock: each established queue performs its
// own lock, management fence, durable admission and worker accounting.
func (r *Server) serveProjectAction(w http.ResponseWriter, q *http.Request, actor, role, id string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "project actions require operator access"}})
		return
	}
	replay, err := protocol.CreationReplayOnly(q.Header)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	var body protocol.ProjectActionRequest
	if err = decode(w, q, &body); err != nil {
		r.respondError(w, err)
		return
	}
	feature, _, _ := protocol.ProjectAction(body.Kind)
	if !protocol.ValidProjectActionRequest(body) {
		r.respondError(w, bad("require a supported explicit project action and its exact member selection"))
		return
	}
	if !slices.Contains(r.projectActionFeatures(), feature) {
		r.respondError(w, &fault{409, protocol.APIError{Code: "project_action_unsupported", Message: "this runner backend does not support the selected project action"}})
		return
	}
	var op protocol.Operation
	var hit bool
	if r.projectExpanded(id) && body.Kind != "member-continue" && body.Kind != "add-member" && body.Kind != "member-start" && body.Kind != "member-stop" && !replay {
		r.respondError(w, conflict("project_expanded", "use the project ship controls for expanded membership; original history is retained"))
		return
	}
	switch body.Kind {
	case "member-continue", "add-member", "member-start", "member-stop":
		op, hit, err = r.queueProjectAddition(actor, id, body, replay)
	case "prepare":
		op, hit, err = r.queueProjectPreparation(actor, id, projectPrepareRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay})
	case "start-anchor":
		op, hit, err = r.queueProjectAnchorStart(actor, id, projectAnchorStartRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay})
	case "prepare-storage":
		op, hit, err = r.queueProjectMemberStorage(actor, id, body.InstanceID, projectMemberStorageRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay})
	case "launch":
		op, hit, err = r.queueProjectMemberLaunch(actor, id, body.InstanceID, projectMemberLaunchRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay})
	case "observe":
		op, hit, err = r.queueProjectMemberReadiness(actor, id, body.InstanceID, projectMemberReadinessRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay})

	case "stop":
		op, hit, err = r.queueProjectMemberStop(actor, id, body.InstanceID, projectMemberStopRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, ReadinessOperationID: body.AfterOperationID})
	case "observe-exit":
		op, hit, err = r.queueProjectMemberStopExit(actor, id, body.InstanceID, projectMemberStopExitRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, StopOperationID: body.AfterOperationID})
	case "retire":
		op, hit, err = r.queueProjectMemberRetirement(actor, id, body.InstanceID, projectMemberRetirementRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, ExitOperationID: body.AfterOperationID})
	case "resume":
		op, hit, err = r.queueProjectMemberResume(actor, id, body.InstanceID, projectMemberResumeRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, RetirementOperationID: body.AfterOperationID})
	case "observe-resumed":
		op, hit, err = r.queueProjectMemberResumedReadiness(actor, id, body.InstanceID, projectMemberResumedReadinessRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, ResumeOperationID: body.AfterOperationID})
	case "observe-current":
		op, hit, err = r.queueProjectMemberContextReadiness(actor, id, body.InstanceID, projectMemberContextReadinessRequest{RequestID: body.RequestID, ExpectedRevision: body.ExpectedRevision, ReviewDigest: body.ReviewDigest, ReplayOnly: replay, CreatorOperationID: body.AfterOperationID, OriginalReadinessOperationID: body.OriginalReadinessOperationID})

	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := http.StatusAccepted
	if hit {
		status = http.StatusOK
	}
	respond(w, status, protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, ReviewDigest: body.ReviewDigest, Operation: op})
}
