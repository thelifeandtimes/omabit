// Package shipaccess loads collector-owned application credentials for clients.
package shipaccess

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/credentials"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
)

func Store(r *collection.Registry, s collection.Ship) (credentials.Store, error) {
	if s.Credentials == "secret-service" {
		return credentials.NewSecretService(), nil
	}
	return credentials.NewFile(r.CredentialsDir())
}
func Key(s collection.Ship) string { return credentials.Scope(s.ID, s.Origin, s.SessionRef) }
func Open(ctx context.Context, r *collection.Registry, s collection.Ship) (*eyre.Client, error) {
	if s.SessionRef == "" {
		return nil, credentials.ErrMissing
	}
	store, err := Store(r, s)
	if err != nil {
		return nil, err
	}
	data, err := store.Load(ctx, Key(s))
	if err != nil {
		return nil, err
	}
	c, err := eyre.New(s.Origin, s.Ship)
	if err != nil {
		clear(data)
		return nil, err
	}
	if err := c.RestoreSession(data); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}
