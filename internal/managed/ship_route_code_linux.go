//go:build linux && amd64

package managed

import (
	"context"
	"net/http"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

type shipRouteCodeRuntime interface {
	RouteCode(context.Context, shiproute.Runtime) (string, error)
}

// The server holds the lifecycle mutex throughout this bounded read. It cannot
// change a route or replace the selected runtime between admission and +code.
func (r *Server) serveShipRouteCode(w http.ResponseWriter, req *http.Request, actor, role string) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if role != "operator" {
		r.respondError(w, &fault{403, protocol.APIError{Code: "operator_required", Message: "ship credentials require operator access"}})
		return
	}
	var u protocol.ShipRouteCodeRequest
	if err := decode(w, req, &u); err != nil {
		r.respondError(w, bad("invalid ship credential selection"))
		return
	}
	if u.Validate() != nil || u.Binding.RunnerID != r.s.runnerID {
		r.respondError(w, bad("invalid ship credential selection"))
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(protocol.ShipRoutesTimeout + time.Second))
	ctx, cancel := context.WithTimeout(req.Context(), protocol.ShipRoutesTimeout)
	defer cancel()
	current, automatic, err := r.shipRouteIdentity(ctx, u.Selection)
	if err != nil {
		r.respondError(w, conflict("ship_route_changed", "selected ship is unavailable or changed"))
		return
	}
	if current != u.Binding {
		r.respondError(w, conflict("ship_route_changed", "ship incarnation changed"))
		return
	}
	record, err := r.s.shipRoutes(current)
	if err != nil {
		r.respondError(w, conflict("ship_route_changed", "ship route settings unavailable"))
		return
	}
	if record.Revision != u.RegistryRevision {
		r.respondError(w, conflict("ship_route_changed", "ship route settings changed"))
		return
	}
	expected := shiproute.Runtime{Ship: current.Ship, Continuity: current.Continuity, Origin: automatic[0].Origin}
	if u.Route.Source == "tailscale" {
		b, ok := r.b.(shipRouteDiscoveryBackend)
		if !ok {
			r.respondError(w, conflict("ship_route_changed", "route discovery unavailable"))
			return
		}
		d := b.DiscoverShipRoutes(ctx, expected.Origin)
		if d.Validate() != nil || d.Route == nil {
			r.respondError(w, conflict("ship_route_changed", "route discovery changed"))
			return
		}
		automatic = append(automatic, *d.Route)
	}
	effective, err := record.Settings.Effective(current, record.Revision, automatic)
	matched := false
	if err == nil {
		for _, route := range effective.Routes {
			if route == u.Route && route.Enabled {
				matched = true
			}
		}
	}
	if !matched || ctx.Err() != nil {
		r.respondError(w, conflict("ship_route_changed", "selected route is disabled or changed"))
		return
	}
	runtime, err := r.openDesk(ctx, u.Selection.ForDesk("base"))
	if err != nil {
		r.respondError(w, conflict("ship_route_changed", "selected runtime changed"))
		return
	}
	reader, ok := runtime.(shipRouteCodeRuntime)
	if !ok {
		r.respondError(w, &fault{501, protocol.APIError{Code: "unsupported", Message: "bound credential retrieval unavailable"}})
		return
	}
	code, err := reader.RouteCode(ctx, expected)
	success := err == nil && ctx.Err() == nil && protocol.ValidLoginCode(code)
	if auditErr := r.s.auditCode(actor, current.Target.InstanceID, u.Selection.ConfigRevision, success, err, ctx.Err() == context.DeadlineExceeded); auditErr != nil {
		r.failed.Store(true)
		r.respondError(w, &fault{500, protocol.APIError{Code: "audit_unavailable", Message: "credential access could not be recorded"}})
		return
	}
	if !success {
		r.respondError(w, &fault{502, protocol.APIError{Code: "code_unavailable", Message: "selected ship credential unavailable"}})
		return
	}
	respond(w, 200, protocol.ShipRouteCode{Version: 1, RunnerID: r.s.runnerID, Mode: r.s.mode, Request: u, Code: code})
}
