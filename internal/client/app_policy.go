package client

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) AppPolicy(ctx context.Context, expected Identity) (protocol.AppPolicyStatus, error) {
	var out protocol.AppPolicyStatus
	h, err := c.Hello(ctx)
	if err != nil {
		return out, err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return out, err
	}
	if !slices.Contains(h.Features, protocol.AppPolicyFeature) {
		return out, errors.New("runner lacks app-development-policy-v2; desk creation, commit and publication unavailable")
	}
	if err = c.do(ctx, http.MethodGet, protocol.AppPolicyPath, nil, &out, true); err != nil {
		return out, err
	}
	if err = expected.Check(out.RunnerID, out.Mode); err != nil {
		return protocol.AppPolicyStatus{}, err
	}
	if out.Version != 1 || out.ReadAt.IsZero() || out.Policy.Validate() != nil || out.Targets == nil || out.DeskOperations {
		return protocol.AppPolicyStatus{}, errors.New("unsupported application policy response")
	}
	seen := map[string]bool{}
	for _, t := range out.Targets {
		if t.Target.Validate() != nil || t.Target.InstanceID == "" || t.Target.Desk != "" || seen[t.Target.InstanceID] || t.Ready && t.Target.ProjectID != "" && t.Target.Generation == 0 {
			return protocol.AppPolicyStatus{}, errors.New("invalid application policy target")
		}
		seen[t.Target.InstanceID] = true
	}
	return out, nil
}

// SetAppPolicy is reserved for the user-facing Setup authoring helper. The
// generic collector CLI intentionally has no permission-setting command.
func (c *Client) SetAppPolicy(ctx context.Context, expected Identity, u apppolicy.Update) (apppolicy.Policy, error) {
	var out apppolicy.Policy
	if err := u.Validate(); err != nil {
		return out, err
	}
	current, err := c.AppPolicy(ctx, expected)
	if err != nil {
		return out, err
	}
	if current.Policy.Revision != u.ExpectedRevision {
		return out, errors.New("application policy changed; refresh Setup before saving")
	}
	if err = c.do(ctx, http.MethodPut, protocol.AppPolicyPath, u, &out, true); err != nil {
		return out, err
	}
	if out.Validate() != nil || out.Revision != u.ExpectedRevision+1 {
		return apppolicy.Policy{}, errors.New("invalid application policy save response; refresh to reconcile")
	}
	return out, nil
}
