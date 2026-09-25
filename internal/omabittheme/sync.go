package omabittheme

import (
	"context"
	"errors"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shipaccess"
)

func Sync(ctx context.Context, r *collection.Registry, store *Store, ship collection.Ship, palette Palette, force bool) error {
	if err := palette.Validate(); err != nil {
		return err
	}
	unlock, err := store.LockShip(ship.ID)
	if err != nil {
		if !force && errors.Is(err, ErrBusy) {
			return nil
		}
		return err
	}
	defer unlock()
	var intent Preference
	skipped := false
	passive := false
	err = store.Update(ctx, func(state *State) error {
		if state.Migration != nil && !state.Migration.Complete {
			for _, id := range state.Migration.Imported {
				if id == ship.ID {
					skipped = true
					return nil
				}
			}
		}
		p, ok := state.Ships[ship.ID]
		if !ok || (!p.Automatic && !force) {
			skipped = true
			return nil
		}
		if p.Origin != ship.Origin || p.Ship != ship.Ship {
			return collection.ErrShipConflict
		}
		if p.AuthenticationRequired && !force {
			return eyre.ErrAuth
		}
		if !force && !p.Pending && p.Fingerprint == palette.Fingerprint() {
			passive = true
			intent = p
			return nil
		}
		if p.AttemptFingerprint != palette.Fingerprint() {
			p.Attempts = 0
		}
		if !force && p.Attempts >= 3 {
			skipped = true
			return nil
		}
		p.Attempts++
		p.AttemptFingerprint = palette.Fingerprint()
		p.Pending = true
		p.Revision++
		state.Ships[p.ID] = p
		intent = p
		return nil
	})
	if err != nil || skipped {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	client, err := shipaccess.Open(ctx, r, ship)
	cleanup := true
	if err == nil {
		if passive {
			_, err = client.Scry(ctx, "settings", "/desk/talon")
		} else {
			cleanup, err = Publish(ctx, client, palette)
		}
		client.Close()
	}
	publishErr := err
	// Use a new bounded context to record timeout/failure as pending work.
	finish, cancelFinish := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFinish()
	err = store.Update(finish, func(state *State) error {
		p, ok := state.Ships[ship.ID]
		if !ok || p.Revision != intent.Revision {
			return nil
		}
		if publishErr != nil {
			p.LastError = publishErr.Error()
			if len(p.LastError) > 512 {
				p.LastError = "Theme publication failed"
			}
			if errors.Is(publishErr, eyre.ErrAuth) {
				p.Automatic = false
				p.Pending = false
				p.AuthenticationRequired = true
			}
		} else if passive {
			p.LastError = ""
		} else {
			p.Pending = false
			p.Attempts = 0
			p.AuthenticationRequired = false
			p.Fingerprint = palette.Fingerprint()
			p.LastTheme = palette.Name
			p.LastPublished = time.Now().UTC().Format(time.RFC3339)
			p.LastError = ""
			if !cleanup {
				p.LastError = "Published; channel cleanup was not confirmed"
			}
		}
		state.Ships[p.ID] = p
		return nil
	})
	return errors.Join(publishErr, err)
}
