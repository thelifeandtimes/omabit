package client

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var ErrPrivateChainRegistrationUnsupported = errors.New("runner does not advertise private-chain-registration-v1; private-chain reservations are unavailable")
var ErrPrivateChainInventoryUnsupported = errors.New("runner does not advertise private-chain-reservations-v1; private-chain inventory is unavailable")

func (c *Client) RegisterPrivateProject(ctx context.Context, expected Identity, req privatechain.RegisterRequest) (privatechain.RegistrationResult, error) {
	var out privatechain.RegistrationResult
	plan, err := req.ReviewedPlan()
	if err != nil {
		return out, err
	}
	h, err := c.Hello(ctx)
	if err != nil {
		return out, err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return out, err
	}
	if !slices.Contains(h.Features, protocol.PrivateChainRegistrationFeature) {
		return out, ErrPrivateChainRegistrationUnsupported
	}
	if h.AccessRole != "operator" || h.Actor == "" {
		return out, errors.New("private-chain reservation requires authenticated operator access")
	}
	if IsCreationReplayOnly(ctx) && !slices.Contains(h.Features, protocol.CreationReplayFeature) {
		return out, errors.New("runner does not support replay-only reservation lookup")
	}
	if err = c.do(ctx, http.MethodPost, protocol.PrivateChainProjectsPath, req, &out, true); err != nil {
		return privatechain.RegistrationResult{}, err
	}
	op := out.Operation
	wantBinding := privatechain.Binding{Mode: privatechain.Mode, RunnerID: expected.RunnerID, ProjectID: op.ProjectID, Generation: 1, FoundationDigest: plan.FoundationDigest}
	if err = expected.Check(out.RunnerID, out.Mode); err == nil {
		if out.SchemaVersion != 1 || !out.ReservationOnly || out.RuntimeAuthorized || out.ReviewDigest != req.ReviewDigest || out.Binding != wantBinding || out.Binding.Validate() != nil || !strings.HasPrefix(op.ProjectID, "private-project-") || !protocol.ValidID(op.ID) || op.InstanceID != "" || op.RequestID != req.RequestID || op.Actor != h.Actor || op.Kind != privatechain.RegistrationKind || op.Status != "succeeded" || op.ConfigRevision != 1 || op.CreatedAt.IsZero() || op.FinishedAt == nil || !op.FinishedAt.Equal(op.CreatedAt) || op.ImportRestoration != nil {
			err = errors.New("private-chain reservation response differs from the selected project")
		}
	}
	if err != nil {
		return privatechain.RegistrationResult{}, &OutcomeUncertainError{Method: http.MethodPost, Path: protocol.PrivateChainProjectsPath, Err: err}
	}
	return out, nil
}

func (c *Client) PrivateProjects(ctx context.Context, expected Identity) (privatechain.Inventory, error) {
	var out privatechain.Inventory
	h, err := c.Hello(ctx)
	if err != nil {
		return out, err
	}
	if err = expected.Check(h.RunnerID, h.Mode); err != nil {
		return out, err
	}
	if !slices.Contains(h.Features, protocol.PrivateChainInventoryFeature) {
		return out, ErrPrivateChainInventoryUnsupported
	}
	if err = c.do(ctx, http.MethodGet, protocol.PrivateChainProjectsPath, nil, &out, true); err != nil {
		return privatechain.Inventory{}, err
	}
	if err = expected.Check(out.RunnerID, out.Mode); err != nil {
		return privatechain.Inventory{}, err
	}
	if !validPrivateProjectInventory(out) {
		return privatechain.Inventory{}, errors.New("invalid private-chain reservation inventory")
	}
	return out, nil
}

func validPrivateProjectInventory(out privatechain.Inventory) bool {
	if out.SchemaVersion != 1 || out.ReadAt.IsZero() || out.Basis != "stored-reservations" || out.RuntimeObservation != "not-performed" || out.RuntimeAuthorized || out.Projects == nil {
		return false
	}
	previous := ""
	ids := map[string]bool{}
	ports := map[uint16]bool{}
	receipts := map[string]bool{}
	for _, p := range out.Projects {
		digest, err := p.Plan.Digest()
		if err != nil || digest != p.ReviewDigest || p.Binding.Validate() != nil || p.Binding.RunnerID != out.RunnerID || p.Binding.FoundationDigest != p.Plan.FoundationDigest || p.Binding.Generation != 1 || !strings.HasPrefix(p.Binding.ProjectID, "private-project-") || p.Binding.ProjectID <= previous || p.Revision != 1 || p.Phase != "planned" || p.CreatedAt.IsZero() || p.CreatedAt.After(out.ReadAt) || !protocol.ValidID(p.RegistrationOperationID) || receipts[p.RegistrationOperationID] {
			return false
		}
		for _, m := range p.Plan.Members {
			if !protocol.ValidPierName(m.InstanceID) || ids[m.InstanceID] || ports[m.HTTPPort] {
				return false
			}
			ids[m.InstanceID], ports[m.HTTPPort] = true, true
		}
		receipts[p.RegistrationOperationID] = true
		previous = p.Binding.ProjectID
	}
	return true
}
