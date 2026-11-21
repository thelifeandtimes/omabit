//go:build linux && amd64

package managed

import (
	"context"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Compiled typed API availability, including saved apply replay when a live
// adapter is unavailable. This is not native workflow qualification.
func (r *Server) grubberyRuntimeRecoveryAvailable() bool { return true }

func (r *Server) serveGrubberyRuntimeRecovery(w http.ResponseWriter, req *http.Request, actor, role string) {
	if !protocol.GrubberyRuntimeLiveEndpoint(req.Method, req.URL.Path) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "App runtime recovery requires operator access."}})
		return
	}
	var review protocol.GrubberyRuntimeReviewRequest
	var apply protocol.GrubberyRuntimeApplyRequest
	isReview := req.URL.Path == protocol.GrubberyRuntimeReviewPath
	var err error
	if isReview {
		err = decodeGrubberySource(w, req, &review)
		if err == nil {
			err = review.Validate()
		}
	} else {
		err = decodeGrubberySource(w, req, &apply)
		if err == nil {
			err = apply.Validate()
		}
	}
	if err != nil {
		r.respondError(w, bad("invalid app runtime recovery selection or confirmation"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.GrubberyAppTimeout + time.Second))
	base := r.ctx
	if isReview {
		base = req.Context()
	}
	ctx, cancel := context.WithTimeout(base, protocol.GrubberyAppTimeout)
	defer cancel()
	r.mu.Lock()
	defer r.mu.Unlock()
	if isReview {
		facts, err := r.reviewGrubberyRuntime(ctx, review.OriginalRequestID, review.Selection, actor, role)
		if err != nil {
			r.respondGrubberyError(w, err)
			return
		}
		original, err := (grubberyJournal{r.s}).Load(ctx, review.OriginalRequestID)
		if err != nil {
			r.respondGrubberyError(w, err)
			return
		}
		out := protocol.GrubberyRuntimeReview{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Original: original, Review: facts}
		if err = out.Validate(); err != nil {
			r.respondGrubberyError(w, err)
			return
		}
		respond(w, 200, out)
		return
	}
	if _, err = r.applyGrubberyRuntime(ctx, apply.OriginalRequestID, apply.RequestID, apply.Review, actor, role); err != nil {
		r.respondGrubberyError(w, err)
		return
	}
	out, err := r.grubberyRuntimeStatus(ctx, apply.OriginalRequestID, apply.RequestID)
	if err != nil {
		r.respondGrubberyError(w, err)
		return
	}
	respond(w, 200, out)
}
