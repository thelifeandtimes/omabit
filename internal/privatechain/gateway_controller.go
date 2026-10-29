package privatechain

import (
	"context"
	"errors"
	"slices"
	"time"
)

// A driver has no implicit defaults. It is opened against the selected private
// runner/engine and held workspace/input objects. VerifyPreparation freshly
// checks the exact owned empty workspace, immutable input generation, compatible
// rootless engine, pinned image and port reservations before each effect. Each
// mutation must repeat its relevant checks immediately before dispatch. Inspection
// must reject foreign/replaced resources and unexpected network participants.
// Create uses exclusive names/labels; known IDs survive an accompanying error.
// InspectGateway requires a never-started container. Start must not retry.
// None of these operations may regenerate, reset, adopt by name, or delete state.
type GatewayDriver interface {
	VerifyPreparation(context.Context, Gateway) error
	CreateNetwork(context.Context, Gateway) (string, error)
	InspectNetwork(context.Context, Gateway) error
	CreateGateway(context.Context, Gateway) (string, error)
	InspectGateway(context.Context, Gateway) error
	StartGateway(context.Context, Gateway) error
	GatewayObserver
}

// Recovery receives only observation authority, with no Create/Start methods.
// ObserveGateway brackets fresh runtime/chain checks with matching object and
// engine identities. It never initializes, mines or repairs retained state.
type GatewayObserver interface {
	ObserveGateway(context.Context, Gateway) (GatewayObservation, error)
}

type GatewayCommit func(expected string, next Gateway) error

var ErrGatewayUnresolved = errors.New("private gateway outcome unresolved; preserve the original operation, resources and partial chain; do not repeat initialization")

func gatewayInputs(plan Plan, w Workspace) (Plan, Workspace) {
	plan.Foundation.Contracts = slices.Clone(plan.Foundation.Contracts)
	plan.Members = slices.Clone(plan.Members)
	return plan, w.Clone()
}

// ExecuteGateway admits only a fresh journal. A committed dispatch fences a
// possibly executed effect even if the response/checkpoint is lost. The caller
// must reload its durable receipt after commit failure; this returned value is
// not permission to retry. There is deliberately no automatic cleanup/restart.
func ExecuteGateway(ctx context.Context, plan Plan, w Workspace, g Gateway, driver GatewayDriver, commit GatewayCommit, now func() time.Time) (Gateway, error) {
	plan, w = gatewayInputs(plan, w)
	g = g.Clone()
	if g.Validate(plan, w) != nil || g.Phase != "planned" || driver == nil || commit == nil || now == nil {
		return g, ErrGatewayJournal
	}
	step := func(phase, network, container string, o *GatewayObservation) error {
		next, err := g.advance(plan, w, phase, network, container, o)
		if err != nil {
			return err
		}
		if err = commit(g.Phase, next.Clone()); err != nil {
			return errors.Join(ErrGatewayUnresolved, ctx.Err())
		}
		g = next
		return nil
	}
	check := func(f func(context.Context, Gateway) error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f(ctx, g.Clone()) != nil {
			return errors.Join(ErrGatewayUnresolved, ctx.Err())
		}
		return ctx.Err()
	}
	create := func(dispatch, created string, effect func(context.Context, Gateway) (string, error), network bool) error {
		if err := check(driver.VerifyPreparation); err != nil {
			return err
		}
		if err := step(dispatch, g.NetworkID, g.ContainerID, nil); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		id, effectErr := effect(ctx, g.Clone())
		if id != "" {
			n, c := g.NetworkID, id
			if network {
				n, c = id, g.ContainerID
			}
			if err := step(created, n, c, nil); err != nil {
				return err
			}
		}
		if effectErr != nil || g.Phase != created {
			return errors.Join(ErrGatewayUnresolved, ctx.Err())
		}
		return ctx.Err()
	}
	if err := create("network-dispatched", "network-created", driver.CreateNetwork, true); err != nil {
		return g, err
	}
	if err := check(driver.InspectNetwork); err != nil {
		return g, err
	}
	if err := create("gateway-dispatched", "gateway-created", driver.CreateGateway, false); err != nil {
		return g, err
	}
	if err := check(driver.InspectGateway); err != nil {
		return g, err
	}
	if err := check(driver.VerifyPreparation); err != nil {
		return g, err
	}
	if err := step("start-dispatched", g.NetworkID, g.ContainerID, nil); err != nil {
		return g, err
	}
	if err := check(driver.StartGateway); err != nil {
		return g, err
	}
	if err := step("started", g.NetworkID, g.ContainerID, nil); err != nil {
		return g, err
	}
	ready, err := ObserveGatewayReadiness(ctx, plan, w, g, driver, now)
	if err != nil {
		return g, err
	}
	if err = step("ready", g.NetworkID, g.ContainerID, ready.Observation); err != nil {
		return g, err
	}
	return g, nil
}

// ObserveGatewayReadiness can reconcile an uncertain Start against its retained
// exact IDs. It performs no mutation and does not rewrite the original operation.
// A recovery caller must retain this result in a separately attributed receipt
// before granting member launch authority. A stopped/missing/partial gateway is
// a refusal; it is never implicitly started or initialized again.
func ObserveGatewayReadiness(ctx context.Context, plan Plan, w Workspace, g Gateway, observer GatewayObserver, now func() time.Time) (Gateway, error) {
	plan, w = gatewayInputs(plan, w)
	g = g.Clone()
	if g.Validate(plan, w) != nil || (g.Phase != "start-dispatched" && g.Phase != "started") || observer == nil || now == nil {
		return g, ErrGatewayJournal
	}
	if err := ctx.Err(); err != nil {
		return g, err
	}
	o, err := observer.ObserveGateway(ctx, g.Clone())
	if err != nil {
		return g, errors.Join(ErrGatewayUnresolved, ctx.Err())
	}
	o = o.Clone()
	if err = g.checkObservation(plan, w, o, now()); err != nil {
		return g, err
	}
	if err = ctx.Err(); err != nil {
		return g, err
	}
	g.Phase, g.Observation = "ready", &o
	return g, g.Validate(plan, w)
}
