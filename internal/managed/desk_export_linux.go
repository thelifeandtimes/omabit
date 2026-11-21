//go:build linux && amd64

package managed

import (
	"context"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type deskSourceExportRuntime interface {
	deskRuntime
	SourceIdentity(context.Context) (string, error)
	ExportSourceFiles(context.Context, string, uint64, []string) ([]deskwork.ExportedFile, error)
}

func (r *Server) serveDeskSourceExport(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "source export requires operator access"}})
		return
	}
	var q protocol.DeskSourceExportRequest
	if err := decode(w, req, &q); err != nil {
		r.respondError(w, err)
		return
	}
	if q.Validate() != nil {
		r.respondError(w, bad("invalid source export selection"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	base, err := r.openDesk(ctx, q.Selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	runtime, ok := base.(deskSourceExportRuntime)
	if !ok {
		r.respondError(w, bad("source export adapter unavailable"))
		return
	}
	identity, err := runtime.SourceIdentity(ctx)
	if err != nil {
		r.respondError(w, conflict("runtime_changed", "source identity unavailable"))
		return
	}
	before, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || before != q.Baseline {
		r.respondError(w, conflict("source_export_changed", "review the current desk baseline"))
		return
	}
	files, err := runtime.ExportSourceFiles(ctx, q.Selection.Target.Desk, q.Baseline.Revision, q.Paths)
	if err != nil || deskwork.ValidateExportedFiles(q.Paths, files) != nil {
		r.respondError(w, conflict("source_export_unavailable", "selected source could not be converted within bounds"))
		return
	}
	after, err := runtime.Observe(ctx, q.Selection.Target.Desk)
	if err != nil || after != before {
		r.respondError(w, conflict("source_export_changed", "desk changed during export"))
		return
	}
	current, err := runtime.SourceIdentity(ctx)
	if err != nil || identity == "" || current != identity {
		r.respondError(w, conflict("runtime_changed", "source identity changed during export"))
		return
	}
	respond(w, 200, protocol.DeskSourceExport{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: q, Files: files, ReadAt: time.Now().UTC()})
}
