//go:build linux && amd64

package managed

import (
	"context"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type deskSourceRuntime interface {
	InspectSource(context.Context, string, string, *deskwork.Manifest) (deskwork.SourceStatus, error)
}

func (r *Server) serveDeskSource(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "live source comparison requires operator access"}})
		return
	}
	var q protocol.DeskSourceRequest
	if err := decode(w, req, &q); err != nil {
		r.respondError(w, err)
		return
	}
	if err := q.Validate(); err != nil {
		r.respondError(w, bad(err.Error()))
		return
	}
	setting, err := r.s.deskSetting(q.Selection.Target)
	if err != nil {
		r.respondError(w, err)
		return
	}
	if setting.Revision != q.SettingsRevision {
		r.respondError(w, conflict("desk_settings_changed", "workflow changed; read it again"))
		return
	}
	if setting.Desk == nil || setting.Desk.Workflow != "mounted" {
		r.respondError(w, bad("use Urgit's native Git/Clay status for this workflow"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	runtime, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	source, ok := runtime.(deskSourceRuntime)
	if !ok {
		r.respondError(w, bad("source comparison adapter unavailable"))
		return
	}
	binding := q.Binding(setting.Desk.SourceName)
	baseline, err := r.s.sourceBaseline(binding)
	if err != nil {
		r.respondError(w, err)
		return
	}
	var base *deskwork.Manifest
	if baseline != nil {
		base = &baseline.Manifest
	}
	status, err := source.InspectSource(ctx, q.Selection.Target.Desk, setting.Desk.SourceName, base)
	if err != nil || ctx.Err() != nil || status.Validate(base) != nil || time.Since(status.ReadAt) > protocol.DeskReadTimeout || status.ReadAt.After(time.Now().Add(5*time.Second)) || baseline != nil && baseline.ObservedAt.After(status.ReadAt) {
		r.respondError(w, &fault{502, protocol.APIError{Code: "desk_source_unavailable", Message: "source comparison unavailable; check the configured directory, runtime and mark converters"}})
		return
	}
	out := protocol.NewDeskSourceStatus(r.s.runnerID, r.s.mode, q, setting.Desk.SourceName, status, baseline)
	if err = out.Validate(); err != nil {
		r.respondError(w, err)
		return
	}
	if status.Comparison.State == "synchronized" {
		err = r.s.saveSourceBaseline(deskwork.SourceBaseline{Version: 1, Binding: binding, Manifest: status.Clay, ObservedAt: status.ReadAt})
		if err != nil {
			r.respondError(w, err)
			return
		}
	}
	respond(w, 200, out)
}
