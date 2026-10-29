package managed

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Called under the ordinary mutation/admission lock. Registration reserves only
// the complete membership and ports; no backend or worker is invoked.
func (r *Server) serveProjectRegistration(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "project reservations require operator access"}})
		return
	}
	replayOnly, err := protocol.CreationReplayOnly(req.Header)
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	var body protocol.ProjectRegisterRequest
	if err = decodeProjectRegistration(w, req, &body); err != nil {
		r.respondError(w, err)
		return
	}
	if !body.ConfirmReservation {
		r.respondError(w, bad("explicit project reservation acknowledgement is required"))
		return
	}
	request := projectnetwork.Request{Label: body.Label, ConfirmSharedAdministration: body.ConfirmSharedAdministration}
	for _, m := range body.Members {
		request.Members = append(request.Members, projectnetwork.MemberRequest{InstanceID: m.InstanceID, Ship: m.Ship, HTTPPort: m.HTTPPort})
	}
	plan, err := projectnetwork.Build(request, projectnetwork.Reservations{})
	if err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	op, replayed, err := r.s.registerProject(actor, projectRegistration{RequestID: body.RequestID, Plan: plan, ReviewDigest: body.ReviewDigest, ReplayOnly: replayOnly})
	if err != nil {
		r.respondError(w, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	respond(w, status, protocol.ProjectRegistrationResult{SchemaVersion: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, ReviewDigest: body.ReviewDigest, ReservationOnly: true, Operation: op})
}

// The complete 256-galaxy member set can exceed the ordinary 4 KiB action body.
func decodeProjectRegistration(w http.ResponseWriter, req *http.Request, out *protocol.ProjectRegisterRequest) error {
	media, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return &fault{415, protocol.APIError{Code: "invalid_content_type", Message: "require application/json"}}
	}
	req.Body = http.MaxBytesReader(w, req.Body, 64*1024)
	defer req.Body.Close()
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(out); err != nil {
		return bad("project reservation must match the JSON schema and 64 KiB limit")
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return bad("require exactly one bounded project reservation value")
	}
	return nil
}
