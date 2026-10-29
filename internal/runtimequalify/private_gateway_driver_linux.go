//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateGatewayReadinessHandle interface {
	privatechain.GatewayObserver
	io.Closer
}

// The first-start driver combines journal-gated effects with bounded read-only
// waiting. It is intentionally private until the guest runtime is qualified.
// Recovery opens the separate observer and never gains these creation methods.
type privateGatewayDriver struct {
	*privateGatewayResources
	waitMu                               sync.Mutex
	openObserver                         func(context.Context, privatechain.Gateway) (privateGatewayReadinessHandle, error)
	readyTimeout, poll                   time.Duration
	startSucceeded, observationAttempted bool
}

var _ privatechain.GatewayDriver = (*privateGatewayDriver)(nil)

func openPrivateGatewayDriver(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (*privateGatewayDriver, error) {
	if q == nil || q.d.times.ready <= 0 || q.d.times.ready > 10*time.Minute || q.d.times.poll <= 0 || q.d.times.poll > q.d.times.ready {
		return nil, errPrivateGatewayResource
	}
	r, err := openPrivateGatewayResources(ctx, q, stateRoot, inputRoot, p, w, g)
	if err != nil {
		return nil, err
	}
	d := &privateGatewayDriver{privateGatewayResources: r, readyTimeout: q.d.times.ready, poll: q.d.times.poll}
	d.openObserver = func(ctx context.Context, selected privatechain.Gateway) (privateGatewayReadinessHandle, error) {
		// Use the resource handle's owned copies, not caller-owned plan slices.
		o, err := openPrivateGatewayObserver(ctx, q, stateRoot, inputRoot, r.plan, r.workspace, selected)
		if err != nil {
			return nil, err
		}
		return o, nil
	}
	return d, nil
}

func (d *privateGatewayDriver) StartGateway(ctx context.Context, g privatechain.Gateway) error {
	d.waitMu.Lock()
	defer d.waitMu.Unlock()
	if err := d.privateGatewayResources.StartGateway(ctx, g); err != nil {
		return err
	}
	d.startSucceeded = true
	return nil
}

func (d *privateGatewayDriver) Close() error {
	d.waitMu.Lock()
	defer d.waitMu.Unlock()
	return d.privateGatewayResources.Close()
}

// Check invariants that cannot legitimately be pending during initialization.
// Input replacement, a different engine/image, unexpected participants, a stopped
// gateway or a competing writer terminates waiting immediately. No bootstrap or
// port-availability check applies after Start has already been dispatched.
func (d *privateGatewayDriver) checkWaiting(ctx context.Context, g privatechain.Gateway) error {
	r := d.privateGatewayResources
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.selected(ctx, g, "started"); err != nil {
		return err
	}
	if !r.startAttempted || !d.startSucceeded {
		return errPrivateGatewayResource
	}
	if err := r.inputsCurrent(ctx); err != nil {
		return err
	}
	if err := r.engine(ctx); err != nil {
		return err
	}
	if err := r.q.inspectImage(ctx); err != nil {
		return err
	}
	if r.q.r.ImageID != r.contract.imageID {
		return errPrivateGatewayResource
	}
	if _, err := r.inspectRunning(ctx); err != nil {
		return err
	}
	return r.checkMountsMode(ctx, true)
}

func (d *privateGatewayDriver) ObserveGateway(ctx context.Context, g privatechain.Gateway) (result privatechain.GatewayObservation, err error) {
	d.waitMu.Lock()
	defer d.waitMu.Unlock()
	bad := func() (privatechain.GatewayObservation, error) {
		// Never expose raw engine/probe diagnostics or return partial authority.
		return privatechain.GatewayObservation{}, errors.Join(privatechain.ErrGatewayUnresolved, ctx.Err())
	}
	if d.observationAttempted || d.openObserver == nil || d.readyTimeout <= 0 || d.readyTimeout > 10*time.Minute || d.poll <= 0 || d.poll > d.readyTimeout {
		return bad()
	}
	d.observationAttempted = true
	ctx, cancel := context.WithTimeout(ctx, d.readyTimeout)
	defer cancel()
	var observer privateGatewayReadinessHandle
	defer func() {
		if observer != nil && observer.Close() != nil {
			result, err = privatechain.GatewayObservation{}, privatechain.ErrGatewayUnresolved
		}
	}()
	for {
		if d.checkWaiting(ctx, g) != nil {
			return bad()
		}
		if observer == nil {
			opened, openErr := d.openObserver(ctx, g.Clone())
			if openErr != nil {
				if opened != nil {
					_ = opened.Close()
					return bad()
				}
			} else {
				if opened == nil {
					return bad()
				}
				observer = opened
			}
		}
		if observer != nil {
			observed, observeErr := observer.ObserveGateway(ctx, g.Clone())
			if observeErr == nil {
				if ctx.Err() != nil || d.checkWaiting(ctx, g) != nil {
					return bad()
				}
				// The controller additionally checks the observation's freshness and
				// complete selected contract before committing its ready receipt.
				return observed.Clone(), nil
			}
			// Retain the same state/input handles across attempts: a subsequent
			// directory replacement must not become a newly accepted baseline.
		}
		timer := time.NewTimer(d.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return bad()
		case <-timer.C:
		}
	}
}
