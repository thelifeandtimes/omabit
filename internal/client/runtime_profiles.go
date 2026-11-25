package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
)

func (c *Client) requireRuntimeProfile(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	_, err := c.selectedRuntimeProfile(ctx, id)
	return err
}

func (c *Client) selectedRuntimeProfile(ctx context.Context, id string) (runtimeprofile.Profile, error) {
	want, err := runtimeprofile.Resolve(id)
	if err != nil && !runtimeprofile.IsLocalID(id) {
		return want, err
	}
	h, err := c.Hello(ctx)
	if err != nil {
		return want, err
	}
	if !slices.Contains(h.Features, protocol.RuntimeProfilesFeature) {
		return want, errors.New("update this runner for runtime-profiles-v1 before selecting or transferring this runtime")
	}
	for _, p := range h.RuntimeProfiles {
		if runtimeprofile.IsLocalID(id) && p.ID == id && runtimeprofile.ValidateLocal(p) == nil {
			return p, nil
		}
		if p == want {
			return p, nil
		}
	}
	return want, errors.New("runner runtime catalogue differs from the exact selected artifact; install it on this runner first")
}

// Preflight proves selected-runner artifact availability without opening a pier.
func (c *Client) RuntimeProfilePreflight(ctx context.Context, id string) (protocol.RuntimeProfilePreflight, error) {
	var out protocol.RuntimeProfilePreflight
	if id == "" {
		return out, errors.New("an explicit runtime profile ID is required")
	}
	want, err := c.selectedRuntimeProfile(ctx, id)
	if err != nil {
		return out, err
	}
	h, err := c.Hello(ctx)
	if err != nil {
		return out, err
	}
	if err = c.do(ctx, http.MethodGet, "/v1/runtime-profiles/"+id, nil, &out, true); err != nil {
		return out, err
	}
	if out.RunnerID != h.RunnerID || !out.Available || out.Profile != want {
		return out, errors.New("runtime artifact preflight identity or profile mismatch")
	}
	return out, nil
}
