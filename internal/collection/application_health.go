package collection

import (
	"context"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

// CheckFleetApplicationGrant requires the exact connected attempt and a current
// standing grant. It changes no state and makes no runner/credential requests.
func (r *Registry) CheckFleetApplicationGrant(ctx context.Context, a ApplicationAttempt) error {
	if !a.valid() || a.Fleet.Version != 1 || a.Phase != "connected" {
		return ErrApplicationConflict
	}
	return r.CheckFleetApplicationPermission(ctx, a)
}

// CheckFleetApplicationPermission checks current permission for an exact durable
// attempt, including preparation. It does not authorize a lifecycle transition.
func (r *Registry) CheckFleetApplicationPermission(ctx context.Context, a ApplicationAttempt) error {
	if !a.valid() || a.Fleet.Version != 1 {
		return ErrApplicationConflict
	}
	return r.locked(ctx, func() error {
		state, err := r.readApplications()
		if err != nil {
			return err
		}
		for _, current := range state.Attempts {
			if current == a {
				return r.currentFleetGrant(a.Fleet, a.Target)
			}
		}
		return ErrApplicationConflict
	})
}

// FleetApplicationOriginEligible verifies the current runner/ship incarnation
// before a saved cookie can be checked. It never treats an old route or Tailnet
// scope as a permanent network grant, and never probes with credentials itself.
func (r *Registry) FleetApplicationOriginEligible(ctx context.Context, a ApplicationAttempt, lan []string) (bool, error) {
	return r.fleetApplicationOriginEligible(ctx, a, lan, shiproute.LocalTailnet)
}

func (r *Registry) fleetApplicationOriginEligible(ctx context.Context, a ApplicationAttempt, lan []string, network func(context.Context) string) (bool, error) {
	u := FleetApplicationRequest{Version: 1, Binding: a.Fleet.Request.Binding, Target: a.Target, GrantRevision: a.Fleet.GrantRevision, LANProfiles: lan}
	if u.Validate() != nil || network == nil {
		return false, ErrApplicationConflict
	}
	ctx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	if err := r.CheckFleetApplicationGrant(ctx, a); err != nil {
		return false, err
	}
	entry, err := r.deskConnection(ctx, a.Fleet.RunnerName, u.Binding.RunnerID)
	if err != nil {
		return false, err
	}
	routes, err := r.ShipRoutes(ctx, a.Fleet.RunnerName, u.Binding.RunnerID, u.Binding.Target.InstanceID)
	if err != nil {
		return false, err
	}
	if routes.Record.Binding != u.Binding {
		return false, ErrApplicationConflict
	}
	candidates, err := routes.Registry.Candidates(fleetEnvironment(entry, network(ctx), lan))
	if err != nil {
		return false, err
	}
	found := false
	for _, route := range candidates {
		if route.Origin == a.Origin && (route.Network != "tailnet" || network(ctx) == route.Scope) {
			found = true
		}
	}
	current, err := r.Get(ctx, a.Fleet.RunnerName)
	if err != nil || !reflect.DeepEqual(current, entry) {
		return false, ErrApplicationConflict
	}
	if err = r.CheckFleetApplicationGrant(ctx, a); err != nil {
		return false, err
	}
	return found, ctx.Err()
}
