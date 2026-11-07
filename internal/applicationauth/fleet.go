package applicationauth

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

// issueFleet creates exactly one new origin-bound application session. It does
// not enroll a parent login, read any saved owner cookie, or fall back after an
// authentication effect. Its caller has already journaled issuing durably.
func (e *Engine) issueFleet(ctx context.Context, a collection.ApplicationAttempt) ([]byte, error) {
	client, err := eyre.New(a.Origin, a.Ship)
	if err != nil {
		return nil, errors.Join(ErrNotIssued, err)
	}
	defer client.Close()
	if err = client.Probe(ctx); err != nil {
		return nil, errors.Join(ErrNotIssued, err)
	}
	code, err := e.registry.FleetApplicationCode(ctx, a)
	if err != nil {
		return nil, errors.Join(ErrNotIssued, err)
	}
	secret := []byte(code.Code)
	code.Code = ""
	defer clear(secret)
	// Login sends no cookie. An error here can follow a server-side effect, so
	// leave the normal uncertain-attempt fence instead of selecting another URL.
	if err = client.Login(ctx, secret); err != nil {
		return nil, err
	}
	return client.ExportSession()
}
