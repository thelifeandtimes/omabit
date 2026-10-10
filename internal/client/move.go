package client

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MoveStopped coordinates a single forward-only handoff. Both clients must use
// identity-verified connections. Phase receipts live on the runners, so a lost
// desktop process can continue the same move without retaining a historical pier.
// An error never unfences a source or retries a mutation automatically.
func MoveStopped(ctx context.Context, source, destination *Client, sourceRunner, instance string, req protocol.PrepareHandoffRequest, httpPort, amesPort uint16) (protocol.Handoff, error) {
	var zero protocol.Handoff
	if !protocol.ValidID(req.RequestID) || len(req.RequestID) > 40 || sourceRunner == req.DestinationRunner {
		return zero, errors.New("require distinct pinned runners and a request ID of at most 40 characters")
	}
	for _, endpoint := range []struct {
		c  *Client
		id string
	}{{source, sourceRunner}, {destination, req.DestinationRunner}} {
		h, err := endpoint.c.Hello(ctx)
		if err != nil {
			return zero, err
		}
		if h.RunnerID != endpoint.id || h.AccessRole != "operator" || !slices.Contains(h.Features, "host-handoff") {
			return zero, errors.New("both selected runners must provide operator host-handoff access")
		}
	}
	// Refuse predictable destination collisions before fencing the source.
	s, err := destination.Snapshot(ctx)
	if err != nil {
		return zero, err
	}
	if s.RunnerID != req.DestinationRunner {
		return zero, ErrIdentityMismatch
	}
	for _, i := range s.Instances {
		if i.ID == req.DestinationInstance {
			if i.Handoff == nil || i.Handoff.SourceRunner != sourceRunner || i.Handoff.SourceInstance != instance {
				return zero, errors.New("destination instance already exists")
			}
		}
	}
	base := req.RequestID
	req.RequestID = base + "-prepare"
	h, err := source.PrepareHandoff(ctx, instance, req)
	if err != nil {
		return zero, err
	}
	if h.SourceRunner != sourceRunner || h.SourceInstance != instance || h.DestinationRunner != req.DestinationRunner || h.DestinationInstance != req.DestinationInstance || h.Direction != "out" {
		return zero, errors.New("source handoff selection mismatch")
	}
	h, err = waitHandoff(ctx, source, h, "ready", "moved")
	if err != nil {
		return h, err
	}
	var incoming protocol.Handoff
	if h.State == "moved" {
		incoming, err = destination.Handoff(ctx, h.ID)
	} else {
		incoming, err = destination.ReceiveHandoff(ctx, protocol.ReceiveHandoffRequest{Offer: h, HTTPPort: httpPort, AmesPort: amesPort, RequestID: base + "-receive"})
	}
	if err != nil {
		return h, err
	}
	if !incoming.SameTransfer(h) || incoming.Direction != "in" {
		return h, errors.New("destination handoff selection mismatch")
	}
	if incoming.State == "complete" {
		if !incoming.ValidCommitment(incoming.Commitment) {
			return h, errors.New("invalid completed handoff receipt")
		}
		return incoming, nil
	}
	if incoming.State == "reserved" {
		if err = RelayHandoff(ctx, source, destination, h); err != nil {
			return h, err
		}
	} else if incoming.State == "receiving" && incoming.Error != "" {
		_, err = destination.FinishHandoff(ctx, h.ID, "retry", protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: 1, RequestID: base + "-retry"})
		if err != nil {
			return h, err
		}
	}
	incoming, err = waitHandoff(ctx, destination, h, "received")
	if err != nil {
		return h, err
	}
	if h.State != "moved" {
		h, err = source.FinishHandoff(ctx, h.ID, "commit", protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: req.ExpectedRevision, RequestID: base + "-commit"})
		if err != nil {
			return h, err
		}
		h, err = waitHandoff(ctx, source, incoming, "moved")
		if err != nil {
			return h, err
		}
	}
	if !incoming.ValidCommitment(h.Commitment) {
		return h, errors.New("source commitment does not authorize this destination")
	}
	result, err := destination.FinishHandoff(ctx, h.ID, "activate", protocol.FinishHandoffRequest{SHA256: h.SHA256, ExpectedRevision: 1, RequestID: base + "-activate", Commitment: h.Commitment})
	if err == nil && (!result.SameTransfer(h) || result.Direction != "in" || result.State != "complete" || !result.ValidCommitment(result.Commitment)) {
		return h, errors.New("invalid destination activation receipt")
	}
	return result, err
}

func waitHandoff(ctx context.Context, c *Client, expected protocol.Handoff, states ...string) (protocol.Handoff, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	for {
		h, err := c.Handoff(ctx, expected.ID)
		if err != nil {
			return h, err
		}
		if expected.SHA256 != "" && !expected.SameTransfer(h) || h.ID != expected.ID || h.SourceRunner != expected.SourceRunner || h.SourceInstance != expected.SourceInstance || h.DestinationRunner != expected.DestinationRunner || h.DestinationInstance != expected.DestinationInstance || h.VerifyKey != expected.VerifyKey {
			return h, errors.New("handoff identity changed while waiting")
		}
		s, err := c.Snapshot(ctx)
		if err != nil {
			return h, err
		}
		id := h.SourceInstance
		runner := h.SourceRunner
		if h.Direction == "in" {
			id = h.DestinationInstance
			runner = h.DestinationRunner
		}
		if s.RunnerID != runner {
			return h, ErrIdentityMismatch
		}
		idle := false
		for _, i := range s.Instances {
			if i.ID == id {
				idle = i.ActiveOperationID == ""
			}
		}
		if idle {
			if slices.Contains(states, h.State) {
				return h, nil
			}
			if h.Error != "" {
				return h, fmt.Errorf("%s; source remains fenced; continue this same move with a new request ID", h.Error)
			}
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return h, errors.New("handoff wait interrupted; continue the same move from its recorded phase")
		case <-timer.C:
		}
	}
}
