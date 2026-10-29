package client

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

// This idempotent content cache step creates no ship or network. A lost reply
// stops the guided workflow; a subsequent explicit invocation re-verifies pins.
func (c *Client) PrepareProjectDependencies(ctx context.Context, pin Identity) error {
	if IsCreationReplayOnly(ctx) {
		return errors.New("dependency download is not a reservation lookup")
	}
	h, err := c.Hello(ctx)
	if err != nil {
		return err
	}
	if err = pin.Check(h.RunnerID, h.Mode); err != nil {
		return err
	}
	if h.AccessRole != "operator" || !slices.Contains(h.Features, protocol.ProjectDependenciesFeature) {
		return errors.New("runner needs operator access and project-dependencies-v1")
	}
	hc := *c.httpClient
	hc.Timeout = 11 * time.Minute
	scoped := *c
	scoped.httpClient = &hc
	var result protocol.ProjectDependenciesResult
	if err = scoped.do(ctx, http.MethodPost, protocol.ProjectDependenciesPath, protocol.ProjectDependenciesRequest{ConfirmDownload: true}, &result, true); err != nil {
		return err
	}
	if err = pin.Check(result.RunnerID, result.Mode); err != nil {
		return err
	}
	if result.SchemaVersion != 1 || result.Image != candidate.Image || result.Pill != (protocol.BootPill{SHA256: candidate.PillSHA256, Bytes: candidate.PillBytes}) {
		return errors.New("project dependency acknowledgement differs from the selected release")
	}
	return nil
}
