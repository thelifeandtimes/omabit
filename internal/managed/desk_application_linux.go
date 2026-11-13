//go:build linux && amd64

package managed

import (
	"context"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type deskApplicationRuntime interface {
	Application(context.Context, string) (deskwork.AppState, error)
}

func (r *Server) serveDeskApplication(w http.ResponseWriter, req *http.Request, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "application observations require operator access"}})
		return
	}
	var selection protocol.DeskSelection
	if err := decode(w, req, &selection); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.DeskReadTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.DeskReadTimeout)
	defer cancel()
	runtime, err := r.openDesk(ctx, selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	adapter, ok := runtime.(deskApplicationRuntime)
	if !ok {
		r.respondError(w, bad("application observation adapter unavailable"))
		return
	}
	unavailable := func() {
		r.respondError(w, &fault{502, protocol.APIError{Code: "desk_application_unavailable", Message: "application observation unavailable, unsupported or changed during inspection"}})
	}
	before, err := runtime.Observe(ctx, selection.Target.Desk)
	if err != nil || !before.Exists || before.Validate() != nil {
		unavailable()
		return
	}
	app, err := adapter.Application(ctx, selection.Target.Desk)
	if err != nil || app.Validate() != nil {
		unavailable()
		return
	}
	after, err := runtime.Observe(ctx, selection.Target.Desk)
	if err != nil || before != after || app.SourceRevision != after.Revision {
		unavailable()
		return
	}
	policy, err := loadAppPolicy(r.s.db)
	if err != nil {
		r.respondError(w, err)
		return
	}
	setting, err := r.s.deskSetting(selection.Target)
	if err != nil {
		r.respondError(w, err)
		return
	}
	out := protocol.DeskApplication{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Selection: selection, PolicyRevision: policy.Revision, SettingsRevision: setting.Revision, Observation: after, Application: app, ReadAt: time.Now().UTC()}
	if err = out.Validate(); err != nil {
		unavailable()
		return
	}
	respond(w, 200, out)
}
