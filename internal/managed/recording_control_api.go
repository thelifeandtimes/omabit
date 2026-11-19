package managed

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"net/http"
	"reflect"
	"time"
)

func (r *Server) serveRecordingControls(w http.ResponseWriter, req *http.Request, actor, role string) {
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "recording controls require operator access"}})
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	switch req.URL.Path {
	case protocol.RecordingControlStatusPath:
		var in recording.ControlReadRequest
		if err := decode(w, req, &in); err != nil {
			r.respondError(w, err)
			return
		}
		if in.Validate() != nil || in.RunnerID != r.s.runnerID {
			r.respondError(w, bad("require exact runner and explicit private-data choice"))
			return
		}
		state, err := r.s.recordingControls(ctx)
		if err != nil {
			r.respondError(w, conflict("recording_control_unavailable", "preserve original control receipts"))
			return
		}
		out := recording.ControlStatus{Version: 1, State: state, PilotEnabled: r.recording != nil, CapacityStatus: "pilot-disabled", ReadAt: time.Now().UTC(), Basis: "new launches use saved byte/job/reserve limits; commitments charge full maxima even for closed jobs; age and automatic retirement are not implemented; policy changes do not change existing workers"}
		if r.recording != nil {
			out.CapacityStatus = "unknown"
		}
		if r.recordingCapacity != nil {
			capacity, err := r.recordingCapacity(ctx, r.s.runnerID)
			if err == nil {
				out.Capacity = &capacity
				out.CapacityStatus = "available"
			}
		}
		after, err := r.s.recordingControls(ctx)
		if err != nil || !reflect.DeepEqual(state, after) {
			r.respondError(w, conflict("recording_control_changed", "refresh recording controls"))
			return
		}
		respond(w, 200, out)
	case protocol.RecordingControlLookupPath:
		var in recording.ControlLookupRequest
		if err := decode(w, req, &in); err != nil {
			r.respondError(w, err)
			return
		}
		out, err := r.s.lookupRecordingControl(ctx, actor, in)
		if err != nil {
			r.respondError(w, conflict("recording_control_unavailable", "lookup requires the original runner, authenticated actor and request ID"))
			return
		}
		respond(w, 200, out)
	case protocol.RecordingControlApplyPath:
		var in recording.ControlRequest
		if err := decode(w, req, &in); err != nil {
			r.respondError(w, err)
			return
		}
		if in.Validate() != nil || in.RunnerID != r.s.runnerID {
			r.respondError(w, bad("require exact runner, request and control revision/digest"))
			return
		}
		if in.Policy != nil && in.Policy.AutoRetire {
			r.respondError(w, &fault{501, protocol.APIError{Code: "recording_retirement_unsupported", Message: "automatic retirement is not implemented"}})
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		// Exact lookup/replay remains usable even while other work is active. A new
		// control revision waits for the selected launch worker to finish.
		saved, err := r.s.lookupRecordingControl(ctx, actor, recording.ControlLookupRequest{RunnerID: in.RunnerID, RequestID: in.RequestID, IncludePrivateData: true})
		if err != nil {
			r.respondError(w, conflict("recording_control_unavailable", "preserve original control receipts"))
			return
		}
		if r.failed.Load() {
			r.respondError(w, conflict("state_store_unavailable", "store mutations are disabled; inspect saved receipts"))
			return
		}
		if (r.closing || r.updateFence.Held) && !saved.Found {
			r.respondError(w, conflict("recording_control_quiesced", "management activation or shutdown holds admission; inspect saved receipts"))
			return
		}
		if (r.updateWorkers > 0 || r.updateBulk > 0) && !saved.Found {
			r.respondError(w, conflict("recording_control_busy", "wait for active management operations before changing recording controls"))
			return
		}
		receipt, replay, err := r.s.applyRecordingControl(ctx, actor, in)
		if errors.Is(err, errRecordingControl) {
			r.respondError(w, conflict("recording_control_changed", "lookup the original request and refresh exact revision/digest"))
			return
		}
		if err != nil {
			r.respondError(w, &fault{500, protocol.APIError{Code: "app_journal_uncertain", Message: "control commit was not confirmed; lookup the exact original request before any retry"}})
			return
		}
		respond(w, 200, recording.ControlApplyResult{Receipt: receipt, Replayed: replay})
	}
}
