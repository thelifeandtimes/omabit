package applicationauth

import (
	"context"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

// CheckFleet observes a connected attempt without login, delivery, logout or
// application-data access. The installed receiver generation, current grant,
// runner/ship incarnation and network eligibility precede any credential read.
// A nonhealthy result does not close the old attempt or authorize replacement.
func (e *Engine) CheckFleet(ctx context.Context, id string, lan []string, receiver Receiver) (string, error) {
	return e.checkFleet(ctx, id, lan, receiver, e.registry.FleetApplicationOriginEligible)
}

func (e *Engine) checkFleet(ctx context.Context, id string, lan []string, receiver Receiver, eligible func(context.Context, collection.ApplicationAttempt, []string) (bool, error)) (string, error) {
	ctx, stop := context.WithTimeout(ctx, 25*time.Second)
	defer stop()
	var state string
	err := e.registry.WithApplication(ctx, id, func(tx *collection.ApplicationTransaction) error {
		a := tx.Current()
		if a.Fleet.Version != 1 || a.Phase != "connected" || e.expectedRevision != 0 && e.expectedRevision != a.Revision || receiver == nil || receiver.Target() != a.Target {
			return collection.ErrApplicationConflict
		}
		if err := e.registry.CheckFleetApplicationGrant(ctx, a); err != nil {
			return err
		}
		if err := receiver.Verify(ctx); err != nil {
			return ErrArtifact
		}
		ok, err := eligible(ctx, a, lan)
		if err != nil {
			return err
		}
		if !ok {
			state = "route-unavailable"
			return nil
		}
		store, err := e.store(a)
		if err != nil {
			return credentials.ErrStore
		}
		data, err := store.Load(ctx, credentialKey(a))
		defer clear(data)
		if err != nil {
			return credentials.ErrStore // Missing/locked state is not an expired session.
		}
		if err = e.registry.CheckFleetApplicationGrant(ctx, a); err != nil {
			return err
		}
		state, err = eyre.StoredSessionHealth(ctx, a.Origin, a.Ship, data)
		if err != nil {
			return err
		}
		return e.registry.CheckFleetApplicationGrant(ctx, a)
	})
	if err != nil {
		return "", err
	}
	return state, nil
}
