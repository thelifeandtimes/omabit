//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

// RouteIdentity is read-only. It binds the native ship identity, actual host
// listener and filesystem object under the same process checks used by Lens.
func (s *DeskRuntime) RouteIdentity(ctx context.Context) (shiproute.Runtime, error) {
	var out shiproute.Runtime
	if s.hostOrigin == nil {
		return out, errors.New("verified host route unavailable")
	}
	before, err := s.SourceIdentity(ctx)
	if err != nil {
		return out, err
	}
	identity, err := s.post(ctx, []byte(`{"source":{"dojo":"our"},"sink":{"stdout":null}}`))
	if err != nil || strings.TrimSpace(identity) != s.ship {
		return out, errors.New("ship route identity unavailable or changed")
	}
	origin, err := s.hostOrigin(ctx)
	if err != nil {
		return out, err
	}
	after, err := s.SourceIdentity(ctx)
	if err != nil || before != after {
		return out, errors.New("ship route pier changed")
	}
	out = shiproute.Runtime{Ship: s.ship, Continuity: after, Origin: origin}
	return out, out.Validate()
}
