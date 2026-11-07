//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

// RouteCode reads one credential from the held private Lens endpoint. It refuses
// another pier even when that pier has the same ~ship and container selection.
// It creates no session and returns no native error or partial response.
func (s *DeskRuntime) RouteCode(ctx context.Context, expected shiproute.Runtime) (string, error) {
	unavailable := errors.New("selected ship credential unavailable")
	if expected.Validate() != nil {
		return "", unavailable
	}
	before, err := s.RouteIdentity(ctx)
	if err != nil || before != expected {
		return "", unavailable
	}
	code, err := s.post(ctx, []byte(`{"source":{"dojo":"+code"},"sink":{"stdout":null}}`))
	if err != nil || !protocol.ValidLoginCode(strings.TrimSpace(code)) {
		return "", unavailable
	}
	after, err := s.RouteIdentity(ctx)
	if err != nil || after != expected || ctx.Err() != nil {
		return "", unavailable
	}
	return strings.TrimSpace(code), nil
}
