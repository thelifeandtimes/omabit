package client

import (
	"context"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) ProjectAllocation(ctx context.Context, in projectusage.Request) (projectusage.Observation, error) {
	var out projectusage.Observation
	if err := in.Validate(); err != nil {
		return out, err
	}
	hc := *c.httpClient
	hc.Timeout = protocol.ProjectAllocationTimeout + 2*time.Second
	scoped := *c
	scoped.httpClient = &hc
	if err := scoped.do(ctx, "POST", protocol.ProjectAllocationPath, in, &out, true); err != nil {
		return projectusage.Observation{}, err
	}
	if out.Validate() != nil || out.Binding.RunnerID != in.RunnerID || out.Binding.ProjectID != in.ProjectID || out.Binding.Revision != in.ExpectedRevision || out.Binding.PlanDigest != in.ReviewDigest || out.Growth == nil || out.ReadAt.After(time.Now().Add(5*time.Second)) || time.Since(out.ReadAt) > protocol.ProjectAllocationTimeout+5*time.Second {
		return projectusage.Observation{}, projectusage.ErrInvalid
	}
	return out, nil
}
