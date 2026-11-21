package collection

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/buildmeta"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AppDevelopmentPreflight contacts only the explicitly selected enrollment. It
// does not boot projects, author permissions, or modify runner/desktop state.
func (r *Registry) AppDevelopmentPreflight(ctx context.Context, req appdev.PreflightRequest) (appdev.Preflight, error) {
	if err := req.Validate(); err != nil {
		return appdev.Preflight{}, err
	}
	entry, err := r.deskConnection(ctx, req.Name, req.RunnerID)
	if err != nil {
		return appdev.Preflight{}, err
	}
	c := entry.Client(true)
	defer c.Close()
	status, h, err := c.AppPolicyWithHello(ctx, entry.Identity())
	var policy *protocol.AppPolicyStatus
	if err != nil {
		var unsupported *client.UnsupportedCapabilityError
		if !errors.As(err, &unsupported) || unsupported.Feature != protocol.AppPolicyFeature {
			return appdev.Preflight{}, err
		}
	} else {
		policy = &status
	}
	out, err := appdev.Assess(req, h, policy)
	out.ClientBuild = buildmeta.Current("")
	return out, err
}
