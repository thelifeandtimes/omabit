//go:build linux && amd64

package runtimequalify

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// SourceIdentity exposes an opaque identity for the already selected pier, never
// a caller-chosen path. Stable births survive device renumbering across reboots;
// records without births keep strict device identity, matching SameObject.
func (s *DeskRuntime) SourceIdentity(ctx context.Context) (string, error) {
	if err := s.check(ctx); err != nil {
		return "", err
	}
	identity, err := pierstore.Identify(s.pier)
	if err != nil {
		return "", err
	}
	if err = s.check(ctx); err != nil {
		return "", err
	}
	if identity.Birth != "" && identity.UrbBirth != "" {
		identity.Device = 0
		identity.UrbDevice = 0
	}
	return pierstore.Fingerprint(identity), nil
}
