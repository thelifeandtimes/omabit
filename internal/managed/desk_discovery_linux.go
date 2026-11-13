//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"time"
)

func (r *Server) serveDeskDiscovery(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "desk discovery requires operator access"}})
		return
	}
	var selection protocol.DeskDiscoverySelection
	if err := decode(w, req, &selection); err != nil {
		r.respondError(w, err)
		return
	}
	if err := selection.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	// The placeholder desk selects only a process; this does not inspect or mutate base.
	runtime, err := r.openDesk(ctx, selection.ForDesk("base"))
	if err != nil {
		r.respondError(w, err)
		return
	}
	names, err := runtime.ListDesks(ctx)
	if err != nil {
		r.respondError(w, &fault{502, protocol.APIError{Code: "desk_discovery_unavailable", Message: "desk discovery unavailable for this runtime"}})
		return
	}
	respond(w, 200, protocol.DeskDiscovery{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Selection: selection, Desks: names, ReadAt: time.Now().UTC()})
}
