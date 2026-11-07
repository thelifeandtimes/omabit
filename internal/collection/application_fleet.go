package collection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

var ErrApplicationPermission = errors.New("standing application permission is absent or changed; review this ship's application access")

// FleetApplication is immutable attempt ancestry, not a reusable grant. The
// ordinary origin-bound journal owns issuance/delivery/cleanup. A replacement
// pier or another independent fake never shares this connection identity.
type FleetApplication struct {
	Version       int                           `json:"version"`
	RunnerName    string                        `json:"runner_name"`
	Request       protocol.ShipRouteCodeRequest `json:"request"`
	GrantRevision uint64                        `json:"grant_revision"`
}

func (f FleetApplication) valid() bool {
	return f.Version == 1 && protocol.ValidID(f.RunnerName) && f.Request.Validate() == nil && f.GrantRevision > 0 && f.GrantRevision <= apppolicy.MaxRevision
}

func FleetConnectionID(b shiproute.Binding) string {
	if b.Validate() != nil {
		return ""
	}
	data, _ := json.Marshal(b)
	digest := sha256.Sum256(append([]byte("omabit-fleet-connection-v1\x00"), data...))
	return hex.EncodeToString(digest[:16])
}

// currentFleetGrant must run under the same registry lock as publication. It
// compares the exact grant revision so revoke/reallow cannot revive old work.
func (r *Registry) currentFleetGrant(f FleetApplication, target ApplicationTarget) error {
	if !f.valid() || !target.Valid() {
		return ErrApplicationConflict
	}
	state, err := r.readApplicationShipApprovals()
	if err != nil {
		return err
	}
	for _, g := range state.Entries {
		if g.Allowed && g.Binding == f.Request.Binding && g.Target == target && g.Revision == f.GrantRevision {
			return nil
		}
	}
	return ErrApplicationPermission
}

// ApproveFleetApplication publishes a fresh attempt only under a current standing
// grant. The caller has already selected/probed a registered route and verified
// the installed receiver. Every credential read rechecks the live runner binding.
// This path creates no parent ship enrollment and never borrows a saved cookie.
func (r *Registry) ApproveFleetApplication(ctx context.Context, id string, target ApplicationTarget, f FleetApplication, credentials string) (ApplicationAttempt, error) {
	if !validRef(id) || !target.Valid() || !f.valid() || (credentials != "secret-service" && credentials != "file") {
		return ApplicationAttempt{}, ErrApplicationConflict
	}
	var result ApplicationAttempt
	err := r.locked(ctx, func() error {
		if _, archived, err := r.readApplicationArchive(id); err != nil {
			return err
		} else if archived {
			return ErrApplicationArchived
		}
		state, err := r.readApplications()
		if err != nil {
			return err
		}
		for _, a := range state.Attempts {
			if a.ID == id {
				if a.Fleet != f || a.Target != target || a.Credentials != credentials || a.Replacement.PreviousID != "" {
					return ErrApplicationConflict
				}
				result = a
				return nil // lookup only; issuance has its own fresh guard
			}
			if a.blocksApproval() && a.Target.PluginID == target.PluginID && a.Target.Profile == target.Profile &&
				(a.Origin == f.Request.Route.Origin || a.Fleet.Version != 0 && a.Fleet.Request.Binding == f.Request.Binding || a.Fleet.Version == 0 && a.Ship == f.Request.Binding.Ship) {
				return ErrApplicationConflict
			}
		}
		if err = r.currentFleetGrant(f, target); err != nil {
			return err
		}
		if len(state.Attempts) >= 64 {
			return ErrApplicationLimit
		}
		now := time.Now().UTC()
		result = ApplicationAttempt{ID: id, ConnectionID: FleetConnectionID(f.Request.Binding), ConnectionRevision: f.GrantRevision,
			Origin: f.Request.Route.Origin, Ship: f.Request.Binding.Ship, Credentials: credentials, SessionRef: NewSessionRef(), Target: target,
			Authority: "eyre-owner-session", Phase: "approved", Revision: 1, ApprovedAt: now, UpdatedAt: now, Fleet: f}
		if !result.valid() {
			return ErrApplicationConflict
		}
		if state.Version < 3 {
			state.Version = 3
		}
		state.Attempts = append(state.Attempts, result)
		return r.saveApplications(state, true)
	})
	if err != nil {
		return ApplicationAttempt{}, err
	}
	return result, nil
}

// FleetApplicationCode verifies the enrolled runner and its exact enabled route.
// It is private to the issuer; never expose this result in a status/CLI response.
func (r *Registry) FleetApplicationCode(ctx context.Context, a ApplicationAttempt) (protocol.ShipRouteCode, error) {
	unavailable := errors.New("selected fleet credential unavailable")
	if !a.valid() || a.Fleet.Version != 1 || a.Phase != "issuing" {
		return protocol.ShipRouteCode{}, unavailable
	}
	// Fresh local permission check before credential retrieval, in addition to
	// publication and the approved -> issuing transition's locked checks.
	if err := r.locked(ctx, func() error {
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
	}); err != nil {
		return protocol.ShipRouteCode{}, err
	}
	entry, err := r.Get(ctx, a.Fleet.RunnerName)
	if err != nil || entry.RunnerID != a.Fleet.Request.Binding.RunnerID {
		return protocol.ShipRouteCode{}, unavailable
	}
	c := entry.Client(true)
	defer c.Close()
	return c.ShipRouteCode(ctx, entry.Identity(), a.Fleet.Request)
}
