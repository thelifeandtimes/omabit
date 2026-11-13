package collection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

var ErrApplicationRoute = errors.New("no registered ship URL is reachable from this network")

// FleetApplicationRequest identifies the standing user choice, never a URL or
// secret. Process restarts can keep the same pier binding; replacement piers and
// revoked/replaced grants require a new explicit choice.
type FleetApplicationRequest struct {
	Version       int               `json:"version"`
	Binding       shiproute.Binding `json:"binding"`
	Target        ApplicationTarget `json:"target"`
	GrantRevision uint64            `json:"grant_revision"`
	LANProfiles   []string          `json:"lan_profiles,omitempty"`
}

func (u FleetApplicationRequest) Validate() error {
	if u.Version != 1 || u.Binding.Validate() != nil || !u.Target.Valid() || u.GrantRevision == 0 || u.GrantRevision > apppolicy.MaxRevision || len(u.LANProfiles) > 8 {
		return ErrApplicationConflict
	}
	seen := map[string]bool{}
	for _, id := range u.LANProfiles {
		if !protocol.ValidID(id) || seen[id] {
			return ErrApplicationConflict
		}
		seen[id] = true
	}
	return nil
}

func DecodeFleetApplicationRequest(data []byte) (FleetApplicationRequest, error) {
	var u FleetApplicationRequest
	if len(data) > 16<<10 || !uniqueApplicationJSON(data) {
		return u, ErrApplicationConflict
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&u) != nil || d.Decode(new(any)) != io.EOF || u.Validate() != nil {
		return FleetApplicationRequest{}, ErrApplicationConflict
	}
	return u, nil
}

func fleetEnvironment(entry Connection, tailnet string, lan []string) shiproute.Environment {
	env := shiproute.Environment{TailnetSuffix: tailnet, LANProfiles: append([]string(nil), lan...)}
	// This scope is enabled only after the pinned local enrollment has answered.
	// SSH reachability never makes the remote machine's localhost local to us.
	if entry.Transport == "unix" && entry.SSH == nil {
		env.HostID = entry.RunnerID
	}
	return env
}

// SelectFleetApplication is read-only. It bounds all candidate probes together,
// rechecks the chosen route/runtime and local network, and obtains no credential.
// ApproveFleetApplication must still publish under a fresh locked grant check.
func (r *Registry) SelectFleetApplication(ctx context.Context, name string, u FleetApplicationRequest) (FleetApplication, []shiproute.Attempt, error) {
	return r.selectFleetApplication(ctx, name, u, shiproute.LocalTailnet, shiproute.Probe)
}

func (r *Registry) selectFleetApplication(ctx context.Context, name string, u FleetApplicationRequest, network func(context.Context) string, probe func(context.Context, string, string) error) (FleetApplication, []shiproute.Attempt, error) {
	if u.Validate() != nil || !protocol.ValidID(name) || network == nil || probe == nil {
		return FleetApplication{}, nil, ErrApplicationConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	state, err := r.ApplicationShipApprovals(ctx)
	if err != nil {
		return FleetApplication{}, nil, err
	}
	allowed := false
	for _, grant := range state.Entries {
		allowed = allowed || grant.Allowed && grant.Binding == u.Binding && grant.Target == u.Target && grant.Revision == u.GrantRevision
	}
	if !allowed {
		return FleetApplication{}, nil, ErrApplicationPermission
	}
	entry, err := r.deskConnection(ctx, name, u.Binding.RunnerID)
	if err != nil {
		return FleetApplication{}, nil, err
	}
	routes, err := r.ShipRoutes(ctx, name, u.Binding.RunnerID, u.Binding.Target.InstanceID)
	if err != nil {
		return FleetApplication{}, nil, err
	}
	if routes.Record.Binding != u.Binding {
		return FleetApplication{}, nil, ErrApplicationConflict
	}
	env := fleetEnvironment(entry, network(ctx), u.LANProfiles)
	probeCtx, stop := context.WithTimeout(ctx, 15*time.Second)
	selected, attempts, err := routes.Registry.Select(probeCtx, env, probe)
	stop()
	if err != nil {
		return FleetApplication{}, attempts, ErrApplicationRoute
	}
	if selected.Network == "tailnet" && network(ctx) != selected.Scope {
		return FleetApplication{}, attempts, ErrApplicationRoute
	}
	current, err := r.Get(ctx, name)
	if err != nil || !reflect.DeepEqual(current, entry) {
		return FleetApplication{}, attempts, ErrApplicationConflict
	}
	c := entry.Client(true)
	defer c.Close()
	fresh, err := c.ShipRoutes(ctx, entry.Identity(), routes.Selection)
	if err != nil {
		return FleetApplication{}, attempts, err
	}
	if fresh.Record.Binding != u.Binding || fresh.Record.Revision != routes.Record.Revision || fresh.Selection != routes.Selection {
		return FleetApplication{}, attempts, ErrApplicationConflict
	}
	found := false
	for _, route := range fresh.Registry.Routes {
		found = found || route == selected
	}
	if !found || ctx.Err() != nil {
		return FleetApplication{}, attempts, ErrApplicationConflict
	}
	f := FleetApplication{Version: 1, RunnerName: name, GrantRevision: u.GrantRevision, Request: protocol.ShipRouteCodeRequest{Selection: fresh.Selection, Binding: u.Binding, RegistryRevision: fresh.Registry.Revision, Route: selected}}
	if err = r.locked(ctx, func() error { return r.currentFleetGrant(f, u.Target) }); err != nil {
		return FleetApplication{}, attempts, err
	}
	return f, attempts, nil
}
