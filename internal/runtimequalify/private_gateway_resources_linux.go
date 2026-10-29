//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateGatewayInputHandle interface {
	Verify(context.Context) (privatechain.RuntimeInputInspection, error)
	VerifyGatewayInitializer(context.Context, string) error
	io.Closer
}
type privateGatewayBootstrapHandle interface {
	VerifyEmpty(context.Context) error
	Initialize(context.Context, privatechain.Gateway) error
	io.Closer
}

var errPrivateGatewayResource = errors.New("private gateway resources differ from the selected initial operation; preserve retained state")

// Resource effects for the private gateway controller. Construction requires an
// already-owned Docker client workspace, not ambient Docker defaults. The caller
// holds the runner work/update fence and commits each dispatch before calling its
// matching method. No public factory or readiness authority is supplied here;
// a complete driver must add fresh chain/runtime observation before acceptance.
// The selected input program still requires the disposable-QEMU guest boundary.
type privateGatewayResources struct {
	mu                                                           sync.Mutex
	q                                                            *qualifier
	plan                                                         privatechain.Plan
	workspace                                                    privatechain.Workspace
	contract                                                     privateGatewayContainerContract
	inputs                                                       privateGatewayInputHandle
	bootstrap                                                    privateGatewayBootstrapHandle
	networkID, containerID                                       string
	networkAttempted, containerAttempted, startAttempted, closed bool
}

func openPrivateGatewayResources(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (*privateGatewayResources, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if q == nil || g.Phase != "planned" {
		return nil, errPrivateGatewayResource
	}
	c, err := newPrivateGatewayContainerContract(p, w, g, stateRoot, inputRoot, q.r.ImageID)
	if err != nil {
		return nil, err
	}
	// Do not create metadata, pull images, or repair input/workspace state here.
	b, err := privatechain.OpenGatewayBootstrap(ctx, filepath.Join(stateRoot, "private-projects"), p, w, g)
	if err != nil {
		return nil, err
	}
	i, err := privatechain.OpenRuntimeInputVerifier(ctx, inputRoot, g.Selection.Inputs.ManifestSHA256, g.Selection.Inputs.FoundationDigest)
	if err != nil {
		b.Close()
		return nil, err
	}
	p.Members = slices.Clone(p.Members)
	p.Foundation.Contracts = slices.Clone(p.Foundation.Contracts)
	e := &privateGatewayResources{q: q, plan: p, workspace: w.Clone(), contract: c, inputs: i, bootstrap: b}
	if err = e.VerifyPreparation(ctx, g); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

func (e *privateGatewayResources) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	return errors.Join(e.inputs.Close(), e.bootstrap.Close())
}
func (e *privateGatewayResources) selected(ctx context.Context, g privatechain.Gateway, phases ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.closed || e.q == nil || e.inputs == nil || e.bootstrap == nil || g.Validate(e.plan, e.workspace) != nil ||
		g.Selection != e.contract.network.selection || !slices.Contains(phases, g.Phase) || g.NetworkID != e.networkID || g.ContainerID != e.containerID {
		return errPrivateGatewayResource
	}
	return nil
}
func (e *privateGatewayResources) engine(ctx context.Context) error {
	if err := e.q.verifyEngine(ctx); err != nil {
		return err
	}
	return hostcompat.CheckEngineIdentity(e.q.engineEvidence, e.contract.network.selection.EngineID)
}
func (e *privateGatewayResources) inputsCurrent(ctx context.Context) error {
	got, err := e.inputs.Verify(ctx)
	if err != nil {
		return err
	}
	if got != e.contract.network.selection.Inputs {
		return errPrivateGatewayResource
	}
	// A reviewed manifest is not enough: creation and Start must use the same
	// implementation that the observer relies on for readiness and RPC policy.
	return e.inputs.VerifyGatewayInitializer(ctx, privateGatewayInitializerSHA256)
}
func (e *privateGatewayResources) verifyPreparation(ctx context.Context, g privatechain.Gateway) error {
	if err := e.selected(ctx, g, "planned", "network-dispatched", "network-created", "gateway-dispatched", "gateway-created", "start-dispatched"); err != nil {
		return err
	}
	if e.startAttempted {
		return errPrivateGatewayResource
	}
	if err := e.bootstrap.VerifyEmpty(ctx); err != nil {
		return err
	}
	if err := e.inputsCurrent(ctx); err != nil {
		return err
	}
	if err := e.engine(ctx); err != nil {
		return err
	}
	if err := e.q.inspectImage(ctx); err != nil {
		return err
	}
	if e.q.r.ImageID != e.contract.imageID {
		return errPrivateGatewayResource
	}
	for _, m := range e.plan.Members {
		if !e.q.d.portAvailable(e.plan.BindAddress, m.HTTPPort) {
			return errPrivateGatewayResource
		}
	}
	if err := e.checkMounts(ctx); err != nil {
		return err
	}
	if err := e.engine(ctx); err != nil {
		return err
	}
	return e.bootstrap.VerifyEmpty(ctx)
}
func (e *privateGatewayResources) VerifyPreparation(ctx context.Context, g privatechain.Gateway) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.verifyPreparation(ctx, g)
}
func (e *privateGatewayResources) CreateNetwork(ctx context.Context, g privatechain.Gateway) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.selected(ctx, g, "network-dispatched"); err != nil {
		return "", err
	}
	if e.networkAttempted {
		return "", errPrivateGatewayResource
	}
	if err := e.verifyPreparation(ctx, g); err != nil {
		return "", err
	}
	e.networkAttempted = true
	out, err := e.q.dockerCreateID(ctx, e.q.d.times.command, e.contract.network.networkArgs()...)
	id := strings.TrimSpace(out)
	if !privateGatewayID(id) {
		return "", errors.Join(errPrivateGatewayResource, err)
	}
	e.networkID = id
	return id, err // A complete ID survives a lost/error response for journalling.
}
func (e *privateGatewayResources) network(ctx context.Context, running bool) (privateGatewayAllocation, error) {
	allowed := ""
	if running {
		allowed = e.containerID
	}
	out, err := e.q.docker(ctx, e.q.d.times.command, "network", "inspect", e.networkID)
	if err != nil {
		return privateGatewayAllocation{}, err
	}
	return e.contract.network.inspectNetwork([]byte(out), e.networkID, allowed)
}
func (e *privateGatewayResources) InspectNetwork(ctx context.Context, g privatechain.Gateway) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.selected(ctx, g, "network-created"); err != nil {
		return err
	}
	if !e.networkAttempted {
		return errPrivateGatewayResource
	}
	if err := e.engine(ctx); err != nil {
		return err
	}
	if _, err := e.network(ctx, false); err != nil {
		return err
	}
	return e.engine(ctx)
}
func (e *privateGatewayResources) CreateGateway(ctx context.Context, g privatechain.Gateway) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.selected(ctx, g, "gateway-dispatched"); err != nil {
		return "", err
	}
	if !e.networkAttempted || e.containerAttempted {
		return "", errPrivateGatewayResource
	}
	if err := e.verifyPreparation(ctx, g); err != nil {
		return "", err
	}
	if _, err := e.network(ctx, false); err != nil {
		return "", err
	}
	args, err := e.contract.containerArgs(e.networkID)
	if err != nil {
		return "", err
	}
	e.containerAttempted = true
	out, err := e.q.dockerCreateID(ctx, e.q.d.times.command, args...)
	id := strings.TrimSpace(out)
	if !privateGatewayID(id) || id == e.networkID {
		return "", errors.Join(errPrivateGatewayResource, err)
	}
	e.containerID = id
	return id, err
}
func (e *privateGatewayResources) inspectCreated(ctx context.Context) error {
	if !privateGatewayID(e.containerID) {
		return errPrivateGatewayResource
	}
	before, err := e.network(ctx, false)
	if err != nil {
		return err
	}
	out, err := e.q.docker(ctx, e.q.d.times.command, "container", "inspect", e.containerID)
	if err != nil {
		return err
	}
	if _, err = e.contract.inspectContainer([]byte(out), e.networkID, e.containerID, before, false); err != nil {
		return err
	}
	after, err := e.network(ctx, false)
	if err != nil {
		return err
	}
	if before != after {
		return errPrivateGatewayResource
	}
	return nil
}
func (e *privateGatewayResources) InspectGateway(ctx context.Context, g privatechain.Gateway) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.selected(ctx, g, "gateway-created"); err != nil {
		return err
	}
	if !e.containerAttempted {
		return errPrivateGatewayResource
	}
	if err := e.engine(ctx); err != nil {
		return err
	}
	if err := e.inspectCreated(ctx); err != nil {
		return err
	}
	return e.engine(ctx)
}
func (e *privateGatewayResources) StartGateway(ctx context.Context, g privatechain.Gateway) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.selected(ctx, g, "start-dispatched"); err != nil {
		return err
	}
	if !e.containerAttempted || e.startAttempted {
		return errPrivateGatewayResource
	}
	if err := e.verifyPreparation(ctx, g); err != nil {
		return err
	}
	if err := e.inspectCreated(ctx); err != nil {
		return err
	}
	if err := e.engine(ctx); err != nil {
		return err
	}
	// Engine/container/mount observation can take time. Rehash the held input
	// generation again before the startup effect can execute any of its code.
	if err := e.inputsCurrent(ctx); err != nil {
		return err
	}
	// The durable start-dispatched receipt covers both metadata initialization and
	// container Start. Any failure after this point is preserved, never retried.
	e.startAttempted = true
	if err := e.bootstrap.Initialize(ctx, g); err != nil {
		return errors.Join(privatechain.ErrGatewayUnresolved, err)
	}
	out, err := e.q.docker(ctx, e.q.d.times.command, "container", "start", e.containerID)
	if err != nil || strings.TrimSpace(out) != e.containerID {
		return errors.Join(privatechain.ErrGatewayUnresolved, err)
	}
	return nil
}
