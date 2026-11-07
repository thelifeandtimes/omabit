//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
	"net/http"
	"time"
)

type shipRouteRuntime interface {
	RouteIdentity(context.Context) (shiproute.Runtime, error)
}

// Called with runner lifecycle admission held, including throughout authoring.
func (r *Server) shipRouteIdentity(ctx context.Context, selection protocol.ShipRouteSelection) (shiproute.Binding, []shiproute.Route, error) {
	var binding shiproute.Binding
	if err := selection.Validate(); err != nil {
		return binding, nil, bad(err.Error())
	}
	runtime, err := r.openDesk(ctx, selection.ForDesk("base"))
	if err != nil {
		return binding, nil, err
	}
	reader, ok := runtime.(shipRouteRuntime)
	if !ok {
		return binding, nil, bad("ship route runtime adapter unavailable")
	}
	identity, err := reader.RouteIdentity(ctx)
	if err != nil || identity.Validate() != nil {
		return binding, nil, &fault{502, protocol.APIError{Code: "ship_route_identity_unavailable", Message: "live ship identity or host listener changed; inspect the runner"}}
	}
	binding = shiproute.Binding{RunnerID: r.s.runnerID, Target: selection.Target, Ship: identity.Ship, Continuity: identity.Continuity}
	automatic := []shiproute.Route{{ID: "managed-http", Origin: identity.Origin, Network: "local", Scope: r.s.runnerID, Source: "managed", Enabled: true}}
	return binding, automatic, binding.Validate()
}

func (r *Server) serveShipRoutes(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost && req.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "ship URL inspection and configuration require operator access"}})
		return
	}
	var selection protocol.ShipRouteSelection
	var update protocol.ShipRoutesUpdate
	if req.Method == http.MethodPut {
		if err := decode(w, req, &update); err != nil {
			r.respondError(w, err)
			return
		}
		if err := update.Validate(); err != nil {
			r.respondError(w, bad(err.Error()))
			return
		}
		selection = update.Selection
	} else if err := decode(w, req, &selection); err != nil {
		r.respondError(w, err)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.ShipRoutesTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.ShipRoutesTimeout)
	defer cancel()
	binding, automatic, err := r.shipRouteIdentity(ctx, selection)
	if err != nil {
		r.respondError(w, err)
		return
	}
	var record shiproute.Record
	if req.Method == http.MethodPut {
		if update.Update.Binding != binding {
			r.respondError(w, conflict("ship_route_binding_changed", "ship incarnation changed; inspect before configuring URLs"))
			return
		}
		record, err = r.s.putShipRoutes(actor, update.Update)
	} else {
		record, err = r.s.shipRoutes(binding)
	}
	if err != nil {
		r.respondError(w, err)
		return
	}
	registry, err := record.Settings.Effective(binding, record.Revision, automatic)
	if err != nil {
		r.respondError(w, err)
		return
	}
	out := protocol.ShipRoutes{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Selection: selection, Record: record, Registry: registry, ReadAt: time.Now().UTC()}
	if err = out.Validate(); err != nil {
		r.respondError(w, err)
		return
	}
	respond(w, 200, out)
}
