//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type PrivateMemberHTTPDriver interface {
	privatechain.MemberHTTPReadinessDriver
	io.Closer
}

// Open is read-only. The caller must hold the common management/update fence
// through collection and Close. Retained birth identities are mandatory; this
// cannot discover/adopt an unbound pier or recreate Docker client metadata.
func OpenPrivateMemberHTTPReadiness(ctx context.Context, root string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch) (PrivateMemberHTTPDriver, error) {
	return openPrivateMemberHTTPReadiness(ctx, root, p, w, s, defaults())
}

func openPrivateMemberHTTPReadiness(ctx context.Context, root string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch, deps dependencies) (_ PrivateMemberHTTPDriver, resultErr error) {
	bad := privatechain.ErrMemberHTTPReadiness
	if ctx.Err() != nil || s.Validate(p, w) != nil || (s.Version != 2 && s.Version != 3) || s.Phase != "running" || s.Running == nil || s.Running.Birth == nil || len(s.Peers) != 0 {
		return nil, bad
	}
	c, err := newPrivateMemberContract(privateMemberSelection{Plan: p, Workspace: w, Gateway: s.Gateway, InstanceID: s.InstanceID, RuntimeID: s.RuntimeID, OperationID: s.OperationID, ContainerGeneration: 1, Mode: "boot"}, root, s.Gateway.Observation.ImageID)
	if err != nil || c.digest != s.ContractDigest {
		return nil, bad
	}
	inputRoot, err := privateGatewayInputPath(root, s.Gateway.Selection.Inputs.ManifestSHA256)
	if err != nil {
		return nil, bad
	}
	d := &privateMemberHTTPDriver{initial: s.Clone(), now: time.Now}
	defer func() {
		if resultErr != nil {
			d.Close()
		}
	}()
	q, err := privateGatewaySession(ctx, root, deps)
	if err != nil {
		return nil, bad
	}
	client, err := openPrivateGatewayClient(ctx, q, s.Gateway, false)
	if err != nil {
		return nil, bad
	}
	d.closers = append(d.closers, client)
	if q.inspectImage(ctx) != nil || q.r.ImageID != c.imageID {
		return nil, bad
	}
	dir := filepath.Join(root, "private-projects")
	pier, err := privatechain.OpenMemberPierVerifier(ctx, dir, p, w, s.InstanceID, s.Running.Birth.Pier)
	if err != nil {
		return nil, bad
	}
	d.held = append(d.held, pier)
	d.closers = append(d.closers, pier)
	pill, err := openPrivateMemberPill(ctx, c, s.Artifact)
	if err != nil {
		return nil, bad
	}
	d.held = append(d.held, pill)
	d.closers = append(d.closers, pill)
	key, err := privatechain.OpenMemberKeyVerifier(ctx, dir, p, w, s.Gateway, s.InstanceID, s.Key.KeySHA256)
	if err != nil {
		return nil, bad
	}
	d.held = append(d.held, key)
	d.closers = append(d.closers, key)
	members := []privateGatewayMember{{Contract: c, ContainerID: s.ContainerID, State: "running", StartedAt: s.Running.StartedAt}}
	piers := []privatechain.GatewayMemberPier{{InstanceID: s.InstanceID, Identity: s.Running.Birth.Pier}}
	observer, err := openPrivateMemberGatewayObserver(ctx, q, root, inputRoot, p, w, s.Gateway, members, piers)
	if err != nil {
		return nil, bad
	}
	d.closers = append(d.closers, observer)
	for _, m := range p.Members {
		if m.InstanceID == s.InstanceID {
			d.endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
			d.ship = m.Ship
		}
	}
	d.process = func(ctx context.Context) (privatechain.MemberBirthProcess, error) {
		return observePrivateMemberProcess(ctx, q, c, d.initial.ContainerID, true, time.Now)
	}
	d.gateway = func(ctx context.Context) (privatechain.GatewayObservation, error) {
		got, err := observer.ObserveGateway(ctx, d.initial.Gateway.Clone())
		if err == nil {
			err = checkPrivateMemberArtifactMounts(ctx, observer.runtime.mountInventory, c, members)
		}
		return got, err
	}
	d.host = q.d.hostIdentity
	if d.VerifyHeld(ctx, s) != nil {
		return nil, bad
	}
	return d, nil
}

// One collection uses two full runtime snapshots surrounding one HTTP read.
// These package-private callbacks are bound above to the selected engine and
// held objects. Failure returns no observation and does not perform repairs.
type privateMemberHTTPDriver struct {
	mu                    sync.Mutex
	initial               privatechain.MemberLaunch
	endpoint, ship        string
	held                  []privateMemberHeld
	closers               []io.Closer
	process               func(context.Context) (privatechain.MemberBirthProcess, error)
	gateway               func(context.Context) (privatechain.GatewayObservation, error)
	host                  func(context.Context, string) (string, error)
	now                   func() time.Time
	last                  *privatechain.MemberHTTPObservation
	readAttempted, closed bool
	closeErr              error
}

func (d *privateMemberHTTPDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.closed {
		d.closed = true
		for i := len(d.closers) - 1; i >= 0; i-- {
			d.closeErr = errors.Join(d.closeErr, d.closers[i].Close())
		}
	}
	return d.closeErr
}
func (d *privateMemberHTTPDriver) verify(ctx context.Context, s privatechain.MemberLaunch) error {
	if ctx.Err() != nil || d.closed || !reflect.DeepEqual(s, d.initial) || len(d.held) != 3 {
		return privatechain.ErrMemberHTTPReadiness
	}
	for _, h := range d.held {
		if ctx.Err() != nil || h.Verify(ctx) != nil || ctx.Err() != nil {
			return privatechain.ErrMemberHTTPReadiness
		}
	}
	return nil
}
func (d *privateMemberHTTPDriver) VerifyHeld(ctx context.Context, s privatechain.MemberLaunch) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.verify(ctx, s)
}
func (d *privateMemberHTTPDriver) validProcess(o privatechain.MemberBirthProcess) bool {
	want := d.initial.Running.Birth.After
	at := o.ObservedAt
	o.ObservedAt = want.ObservedAt
	now := d.now()
	return o == want && !at.Before(d.initial.Running.ObservedAt) && !at.IsZero() && now.Sub(at) <= 30*time.Second && at.Sub(now) <= 5*time.Second
}
func (d *privateMemberHTTPDriver) ObserveRuntime(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberHTTPObservation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	bad := func() (privatechain.MemberHTTPObservation, error) {
		d.last = nil
		return privatechain.MemberHTTPObservation{}, privatechain.ErrMemberHTTPReadiness
	}
	if d.verify(ctx, s) != nil {
		return bad()
	}
	before, err := d.process(ctx)
	if err != nil || !d.validProcess(before) || d.verify(ctx, s) != nil {
		return bad()
	}
	g, err := d.gateway(ctx)
	g = g.Clone()
	if err != nil || d.verify(ctx, s) != nil {
		return bad()
	}
	after, err := d.process(ctx)
	if err != nil || !d.validProcess(after) || after.ObservedAt.Before(before.ObservedAt) || d.verify(ctx, s) != nil {
		return bad()
	}
	want, got := d.initial.Running.Gateway.Clone(), g.Clone()
	want.ObservedAt, got.ObservedAt = time.Time{}, time.Time{}
	now := d.now()
	if !reflect.DeepEqual(want, got) || g.ObservedAt.Before(before.ObservedAt) || g.ObservedAt.After(after.ObservedAt) || now.Sub(g.ObservedAt) > 30*time.Second || g.ObservedAt.Sub(now) > 5*time.Second {
		return bad()
	}
	o := privatechain.MemberHTTPObservation{Gateway: g, Process: after, Pier: d.initial.Running.Birth.Pier}
	copy := o.Clone()
	d.last = &copy
	return o, nil
}
func (d *privateMemberHTTPDriver) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	bad := func() (string, error) { return "", privatechain.ErrMemberHTTPReadiness }
	if endpoint != d.endpoint || d.readAttempted || d.last == nil || !d.validProcess(d.last.Process) || d.now().Sub(d.last.Gateway.ObservedAt) > 30*time.Second || d.verify(ctx, d.initial) != nil {
		return bad()
	}
	d.readAttempted = true
	identity, err := d.host(ctx, endpoint)
	if err != nil || identity != d.ship || d.verify(ctx, d.initial) != nil {
		return bad()
	}
	return identity, nil
}

var _ PrivateMemberHTTPDriver = (*privateMemberHTTPDriver)(nil)
