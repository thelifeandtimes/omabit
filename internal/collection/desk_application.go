package collection

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

type DeskApplicationInspection struct {
	Observation protocol.DeskApplication `json:"observation"`
	Permissions apppolicy.Effective      `json:"permissions"`
}

func (r *Registry) InspectDeskApplication(ctx context.Context, name, runner, instance, desk string) (DeskApplicationInspection, error) {
	var out DeskApplicationInspection
	if (apppolicy.Target{InstanceID: instance, Desk: desk}).Validate() != nil || instance == "" || desk == "" {
		return out, errors.New("select an instance and desk")
	}
	entry, err := r.deskConnection(ctx, name, runner)
	if err != nil {
		return out, err
	}
	c := entry.Client(true)
	defer c.Close()
	policy, err := c.AppPolicy(ctx, entry.Identity())
	if err != nil {
		return out, err
	}
	selection, err := selectDeskRuntime(ctx, c, entry.Identity(), policy, instance, desk)
	if err != nil {
		return out, err
	}
	observed, err := c.DeskApplication(ctx, entry.Identity(), selection)
	if err != nil {
		return out, err
	}
	if observed.PolicyRevision != policy.Policy.Revision {
		return out, errors.New("policy changed during application inspection; inspect again")
	}
	return DeskApplicationInspection{Observation: observed, Permissions: policy.Policy.Effective(selection.Target)}, nil
}
