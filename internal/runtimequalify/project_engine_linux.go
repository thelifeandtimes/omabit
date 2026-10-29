//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// This adapter only creates/inspects a bridge and stopped anchor. Fresh session
// setup and private server fencing exist; public routes/members are pending. Its caller
// must execute it through the durable preparation controller, never directly.
type projectDockerEngine struct {
	q                  *qualifier
	contract           projectResourceContract
	dispatched         map[string]bool
	bridgeID, anchorID string
	bridgeVerified     bool
}

func (e *projectDockerEngine) VerifyEngine(ctx context.Context, id string) error {
	if e.q == nil || id != e.contract.Preparation.EngineID || e.q.r.ImageID != e.contract.ImageID {
		return errors.New("project runtime session does not match its prepared contract")
	}
	if err := e.q.verifyEngine(ctx); err != nil {
		return err
	}
	return hostcompat.CheckEngineIdentity(e.q.engineEvidence, id)
}
func (e *projectDockerEngine) selection(kind string, r projectnetwork.Creation, phase string) error {
	expected := e.contract.Preparation.Bridge
	if kind == "anchor" {
		expected = e.contract.Preparation.Anchor
	} else if kind != "bridge" {
		return errors.New("unknown project resource kind")
	}
	if r.Name != expected.Name || r.ContractDigest != expected.ContractDigest || r.ContractDigest != e.contract.digest(kind) || r.Phase != phase {
		return errors.New("project resource contract changed")
	}
	return nil
}
func (e *projectDockerEngine) Create(ctx context.Context, kind string, r projectnetwork.Creation, bridgeID string) (string, error) {
	if err := e.selection(kind, r, "dispatched"); err != nil {
		return "", err
	}
	if r.ID != "" || e.dispatched[kind] {
		return "", errors.New("project resource creation was already dispatched")
	}
	if err := e.VerifyEngine(ctx, e.contract.Preparation.EngineID); err != nil {
		return "", err
	}
	args := e.contract.bridgeArgs()
	if kind == "anchor" {
		if !e.bridgeVerified || bridgeID != e.bridgeID || !containerIDPattern.MatchString(bridgeID) {
			return "", errUntrustedNetwork
		}
		if err := e.inspectBridge(ctx, bridgeID, ""); err != nil {
			return "", err
		}
		args = e.contract.anchorArgs(bridgeID)
	} else if bridgeID != "" {
		return "", errUntrustedNetwork
	}
	if e.dispatched == nil {
		e.dispatched = map[string]bool{}
	}
	e.dispatched[kind] = true
	out, err := e.q.dockerCreateID(ctx, e.q.d.times.command, args...)
	id := strings.TrimSpace(out)
	if !containerIDPattern.MatchString(id) {
		if err != nil {
			return "", err
		}
		return "", errors.New("project creation returned no exact resource identity")
	}
	if kind == "bridge" {
		e.bridgeID = id
	} else {
		e.anchorID = id
	}
	return id, err
}
func (e *projectDockerEngine) inspectBridge(ctx context.Context, id, allowedAnchor string) error {
	if !containerIDPattern.MatchString(id) {
		return errUntrustedNetwork
	}
	out, err := e.q.docker(ctx, e.q.d.times.command, "network", "inspect", id)
	if err != nil {
		return err
	}
	var values []bridge
	if json.Unmarshal([]byte(out), &values) != nil || len(values) != 1 {
		return errUntrustedNetwork
	}
	return e.contract.validateBridge(values[0], id, allowedAnchor)
}
func (e *projectDockerEngine) Inspect(ctx context.Context, kind string, r projectnetwork.Creation, bridgeID string) (projectnetwork.ResourceIdentity, error) {
	fail := func(err error) (projectnetwork.ResourceIdentity, error) {
		return projectnetwork.ResourceIdentity{}, err
	}
	if err := e.selection(kind, r, "created"); err != nil {
		return fail(err)
	}
	if !containerIDPattern.MatchString(r.ID) {
		return fail(errors.New("created project resource has no exact ID"))
	}
	if err := e.VerifyEngine(ctx, e.contract.Preparation.EngineID); err != nil {
		return fail(err)
	}
	switch kind {
	case "bridge":
		if r.ID != e.bridgeID || bridgeID != "" {
			return fail(errUntrustedNetwork)
		}
		if err := e.inspectBridge(ctx, r.ID, ""); err != nil {
			return fail(err)
		}
		e.bridgeVerified = true
	case "anchor":
		if r.ID != e.anchorID || bridgeID != e.bridgeID || !e.bridgeVerified {
			return fail(errUntrustedContainer)
		}
		if err := e.inspectBridge(ctx, bridgeID, r.ID); err != nil {
			return fail(err)
		}
		out, err := e.q.docker(ctx, e.q.d.times.command, "container", "inspect", r.ID)
		if err != nil {
			return fail(err)
		}
		var values []container
		if json.Unmarshal([]byte(out), &values) != nil || len(values) != 1 {
			return fail(errUntrustedContainer)
		}
		if err = e.contract.validateStoppedAnchor(values[0], r.ID, bridgeID); err != nil {
			return fail(err)
		}
	default:
		return fail(errors.New("unknown project resource kind"))
	}
	// Recompute from the fixed contract after checking every admitted field; do not
	// echo an unchecked digest supplied by a caller or by container labels.
	return projectnetwork.ResourceIdentity{ID: r.ID, ContractDigest: e.contract.digest(kind)}, nil
}
