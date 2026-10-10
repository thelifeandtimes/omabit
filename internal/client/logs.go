package client

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func (c *Client) RuntimeLogs(ctx context.Context, runner, id string, revision uint64) (protocol.RuntimeLogs, error) {
	var out protocol.RuntimeLogs
	if err := validateID(id); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", "/v1/instances/"+id+"/logs", protocol.LogsRequest{ExpectedRevision: revision}, &out, true); err != nil {
		return out, err
	}
	if out.RunnerID != runner || out.InstanceID != id || out.ConfigRevision != revision || out.Lines == nil || len(out.Lines) > protocol.MaxLogLines || out.ObservedAt.IsZero() {
		return protocol.RuntimeLogs{}, errors.New("invalid runtime log selection or response")
	}
	total := 0
	for _, line := range out.Lines {
		total += len(line) + 1
	}
	if total > protocol.MaxLogBytes {
		return protocol.RuntimeLogs{}, errors.New("oversized runtime log response")
	}
	return out, nil
}
