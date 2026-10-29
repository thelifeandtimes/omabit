//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"sync"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type PrivateMemberLaunchDriver interface {
	privatechain.MemberLaunchDriver
	io.Closer
}

// Pure accepted-contract builder: no engine or filesystem access. The runner
// calls this only after replay and ownership checks inside its acceptance
// transaction. Paths and image identity come from its retained gateway selection.
func PrivateMemberLaunchContract(stateRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, instance, runtime, operation string) (string, error) {
	if g.Validate(p, w) != nil || g.Phase != "ready" {
		return "", errPrivateMemberLaunch
	}
	c, err := newPrivateMemberContract(privateMemberSelection{Plan: p, Workspace: w, Gateway: g, InstanceID: instance, RuntimeID: runtime, OperationID: operation, ContainerGeneration: 1, Mode: "boot"}, stateRoot, g.Observation.ImageID)
	if err != nil {
		return "", errPrivateMemberLaunch
	}
	return c.digest, nil
}

type privateMemberSessionDriver struct {
	privatechain.MemberLaunchDriver
	driver, client io.Closer
	once           sync.Once
	closeErr       error
}

// Preserve the capture capability through the session's base-interface wrapper.
// A legacy-only test/backend driver is refused without invoking any effect.
func (d *privateMemberSessionDriver) CaptureBirth(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberBirthEvidence, error) {
	driver, ok := d.MemberLaunchDriver.(privatechain.CapturedMemberLaunchDriver)
	if !ok {
		return privatechain.MemberBirthEvidence{}, errPrivateMemberLaunch
	}
	return driver.CaptureBirth(ctx, s.Clone())
}

func (d *privateMemberSessionDriver) Close() error {
	d.once.Do(func() { d.closeErr = errors.Join(d.driver.Close(), d.client.Close()) })
	return d.closeErr
}

// Read-only factory for the internal scheduler, never a startup continuation.
// Reuses only the retained gateway's owned Docker client metadata. The common
// runner work/update fence must cover Open, the controller and Close.
func OpenPrivateMemberLaunch(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch) (PrivateMemberLaunchDriver, error) {
	return openManagedPrivateMemberLaunch(ctx, stateRoot, p, w, s, defaults())
}
func openManagedPrivateMemberLaunch(ctx context.Context, stateRoot string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch, d dependencies) (PrivateMemberLaunchDriver, error) {
	if s.Validate(p, w) != nil || (s.Version != 2 && s.Version != 3) || s.Phase != "planned" || len(s.Peers) != 0 {
		return nil, errPrivateMemberLaunch
	}
	digest, err := PrivateMemberLaunchContract(stateRoot, p, w, s.Gateway, s.InstanceID, s.RuntimeID, s.OperationID)
	if err != nil || digest != s.ContractDigest {
		return nil, errPrivateMemberLaunch
	}
	inputRoot, err := privateGatewayInputPath(stateRoot, s.Gateway.Selection.Inputs.ManifestSHA256)
	if err != nil {
		return nil, err
	}
	q, err := privateGatewaySession(ctx, stateRoot, d)
	if err != nil {
		return nil, err
	}
	client, err := openPrivateGatewayClient(ctx, q, s.Gateway, false)
	if err != nil {
		return nil, err
	}
	if err = q.inspectImage(ctx); err != nil {
		client.Close()
		return nil, err
	}
	driver, err := openPrivateMemberLaunch(ctx, q, stateRoot, inputRoot, p, w, s)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &privateMemberSessionDriver{MemberLaunchDriver: driver, driver: driver, client: client}, nil
}
