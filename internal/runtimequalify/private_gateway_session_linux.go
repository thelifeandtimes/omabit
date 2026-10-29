//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// These factories are for the retained internal runner scheduler. No public
// route accepts paths or gateway descriptors. Guest-only input programs still
// require a fresh no-NIC VM qualification before supported runtime release.
type PrivateGatewayDriver interface {
	privatechain.GatewayDriver
	io.Closer
}
type PrivateGatewayObserver interface {
	privatechain.GatewayObserver
	io.Closer
}

func validPrivateGatewayOperation(id string) bool { return protocol.ValidID(id) }

func privateGatewayInputPath(stateRoot, manifest string) (string, error) {
	if !privateGatewayMountPath(stateRoot) || !privateGatewayID(manifest) {
		return "", errPrivateGatewayResource
	}
	return filepath.Join(stateRoot, "private-runtime-inputs", manifest), nil
}

func privateGatewaySession(ctx context.Context, stateRoot string, d dependencies) (*qualifier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !privateGatewayMountPath(d.home) || stateRoot != filepath.Join(d.home, ".local/share/urbit-runner/managed") {
		return nil, errPrivateGatewayResource
	}
	q := &qualifier{d: d, base: stateRoot, phase: "private-gateway"}
	if err := q.boundary(); err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(stateRoot)
	if err != nil || real != stateRoot {
		return nil, errPrivateGatewayResource
	}
	for _, p := range q.storagePaths() {
		if err = q.owned(p, os.ModeDir, p == stateRoot); err != nil {
			return nil, err
		}
	}
	return q, nil
}

// Read-only admission: no client directory, pull, network/container, chain or key
// is created. Input delivery is separate and fixed to a manifest-named directory.
func InspectPrivateGatewayInputs(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, manifest string) (string, privatechain.RuntimeInputInspection, error) {
	root, err := privateGatewayInputPath(stateRoot, manifest)
	if err != nil {
		return "", privatechain.RuntimeInputInspection{}, err
	}
	return inspectPrivateGatewayInputs(ctx, stateRoot, root, p, w, manifest, defaults())
}
func inspectPrivateGatewayInputs(ctx context.Context, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, manifest string, d dependencies) (engine string, result privatechain.RuntimeInputInspection, err error) {
	bad := func() (string, privatechain.RuntimeInputInspection, error) {
		return "", privatechain.RuntimeInputInspection{}, errPrivateGatewayResource
	}
	if w.Validate(p) != nil || w.Phase != "observed" || !privateGatewayID(manifest) {
		return bad()
	}
	q, err := privateGatewaySession(ctx, stateRoot, d)
	if err != nil {
		return "", result, err
	}
	v, err := privatechain.OpenRuntimeInputVerifier(ctx, inputRoot, manifest, p.FoundationDigest)
	if err != nil {
		return "", result, err
	}
	defer v.Close()
	result, err = v.Verify(ctx)
	if err != nil {
		return bad()
	}
	if v.VerifyGatewayInitializer(ctx, privateGatewayInitializerSHA256) != nil {
		return bad()
	}
	wv, err := privatechain.OpenWorkspaceVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w)
	if err != nil {
		return bad()
	}
	defer wv.Close()
	if q.verifyEngine(ctx) != nil || !hostcompat.ValidEngineIdentity(q.engineEvidence.ID) {
		return bad()
	}
	engine = q.engineEvidence.ID
	if wv.VerifyEmpty(ctx) != nil {
		return bad()
	}
	after, err := v.Verify(ctx)
	if err != nil || after != result || v.VerifyGatewayInitializer(ctx, privateGatewayInitializerSHA256) != nil || ctx.Err() != nil {
		return bad()
	}
	return engine, result, nil
}

type privateSessionDriver struct {
	privatechain.GatewayDriver
	driver   io.Closer
	client   *privateGatewayClient
	once     sync.Once
	closeErr error
}

func (d *privateSessionDriver) Close() error {
	d.once.Do(func() { d.closeErr = errors.Join(d.driver.Close(), d.client.Close()) })
	return d.closeErr
}

type privateSessionObserver struct {
	privatechain.GatewayObserver
	observer io.Closer
	client   *privateGatewayClient
	once     sync.Once
	closeErr error
}

func (o *privateSessionObserver) Close() error {
	o.once.Do(func() { o.closeErr = errors.Join(o.observer.Close(), o.client.Close()) })
	return o.closeErr
}

func OpenPrivateGateway(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (PrivateGatewayDriver, error) {
	inputRoot, err := privateGatewayInputPath(stateRoot, g.Selection.Inputs.ManifestSHA256)
	if err != nil {
		return nil, err
	}
	return openManagedPrivateGateway(ctx, stateRoot, inputRoot, p, w, g, defaults())
}
func openManagedPrivateGateway(ctx context.Context, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, d dependencies) (PrivateGatewayDriver, error) {
	if g.Validate(p, w) != nil || g.Phase != "planned" {
		return nil, errPrivateGatewayResource
	}
	engine, inputs, err := inspectPrivateGatewayInputs(ctx, stateRoot, inputRoot, p, w, g.Selection.Inputs.ManifestSHA256, d)
	if err != nil {
		return nil, err
	}
	if engine != g.Selection.EngineID || inputs != g.Selection.Inputs {
		return nil, errPrivateGatewayResource
	}
	q, err := privateGatewaySession(ctx, stateRoot, d)
	if err != nil {
		return nil, err
	}
	client, err := openPrivateGatewayClient(ctx, q, g, true)
	if err != nil {
		return nil, err
	}
	if err = q.inspectImage(ctx); err != nil {
		client.Close()
		return nil, err
	}
	driver, err := openPrivateGatewayDriver(ctx, q, stateRoot, inputRoot, p, w, g)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &privateSessionDriver{GatewayDriver: driver, driver: driver, client: client}, nil
}

func OpenPrivateGatewayObserver(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, operation string) (PrivateGatewayObserver, error) {
	inputRoot, err := privateGatewayInputPath(stateRoot, g.Selection.Inputs.ManifestSHA256)
	if err != nil {
		return nil, err
	}
	return openManagedPrivateGatewayObserver(ctx, stateRoot, inputRoot, p, w, g, operation, defaults())
}
func openManagedPrivateGatewayObserver(ctx context.Context, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, operation string, d dependencies) (PrivateGatewayObserver, error) {
	if g.Validate(p, w) != nil || (g.Phase != "start-dispatched" && g.Phase != "started") || !validPrivateGatewayOperation(operation) || operation == g.Selection.OperationID || operation == w.OperationID {
		return nil, errPrivateGatewayResource
	}
	q, err := privateGatewaySession(ctx, stateRoot, d)
	if err != nil {
		return nil, err
	}
	client, err := openPrivateGatewayClient(ctx, q, g, false)
	if err != nil {
		return nil, err
	}
	if err = q.inspectImage(ctx); err != nil {
		client.Close()
		return nil, err
	}
	o, err := openPrivateGatewayObserver(ctx, q, stateRoot, inputRoot, p, w, g)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &privateSessionObserver{GatewayObserver: o, observer: o, client: client}, nil
}
