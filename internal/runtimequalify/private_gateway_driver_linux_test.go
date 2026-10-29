//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateWaitingObserver struct {
	inner         *privateGatewayObserver
	calls, closes int
	failures      int
	hook          func()
	closeErr      error
}

func (o *privateWaitingObserver) ObserveGateway(ctx context.Context, g privatechain.Gateway) (privatechain.GatewayObservation, error) {
	o.calls++
	if o.hook != nil {
		o.hook()
	}
	if o.calls <= o.failures {
		return privatechain.GatewayObservation{}, errors.New("SECRET pending initialization")
	}
	return o.inner.ObserveGateway(ctx, g)
}
func (o *privateWaitingObserver) Close() error { o.closes++; return o.closeErr }

// Use the actual combined observer with inert engine/filesystem adapters. The
// other resource fixture executes the full journal/controller path before it.
func waitingDriverFixture(t *testing.T) (*privateResourceFixture, *privateGatewayDriver, *privateWaitingObserver, *int) {
	t.Helper()
	x := newPrivateResourceFixture(t)
	f := newPrivateObserverFixture(t)
	o := &privateWaitingObserver{inner: f.o}
	d := &privateGatewayDriver{privateGatewayResources: x.e, readyTimeout: time.Second, poll: time.Millisecond}
	opens := new(int)
	d.openObserver = func(ctx context.Context, g privatechain.Gateway) (privateGatewayReadinessHandle, error) {
		*opens++
		if x.saved.Phase != "started" || x.startCalls != 1 || x.boot.calls != 1 {
			t.Fatal("observation preceded committed Start")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded readiness")
		}
		return o, nil
	}
	original := x.e.q.d.execute
	x.e.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		out, err := original(ctx, args, env)
		if reflect.DeepEqual(args[4:], []string{"container", "start", strings.Repeat("b", 64)}) && err == nil {
			_, running, _ := privateContainerFixture(t, true)
			x.container["State"], x.container["NetworkSettings"] = running["State"], running["NetworkSettings"]
			privateNetworkAttach(x.network)
			raw, marshalErr := json.Marshal(x.container["Mounts"])
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			var mounts []mount
			if json.Unmarshal(raw, &mounts) != nil {
				t.Fatal("invalid fixture mounts")
			}
			x.inventory[x.e.containerID] = mounts
		}
		return out, err
	}
	t.Cleanup(func() { d.Close() })
	return x, d, o, opens
}

func TestPrivateGatewayDriverWaitsWithoutRepeatingEffects(t *testing.T) {
	x, d, o, opens := waitingDriverFixture(t)
	open := d.openObserver
	pending := 2
	d.openObserver = func(ctx context.Context, g privatechain.Gateway) (privateGatewayReadinessHandle, error) {
		if pending > 0 {
			pending--
			return nil, errors.New("private state not complete")
		}
		return open(ctx, g)
	}
	o.failures = 2
	got, err := privatechain.ExecuteGateway(context.Background(), x.p, x.w, x.g, d, x.commit, time.Now)
	if err != nil || got.Phase != "ready" || x.saved.Phase != "ready" {
		t.Fatal(got.Phase, err)
	}
	if pending != 0 || *opens != 1 || o.calls != 3 || o.closes != 1 || x.networkCalls != 1 || x.containerCalls != 1 || x.startCalls != 1 || x.boot.calls != 1 {
		t.Fatal("repeated effects or reopened observation", *opens, o.calls, o.closes, x.events)
	}
	if _, err = d.ObserveGateway(context.Background(), got); err == nil || o.calls != 3 {
		t.Fatal("repeated first-start observation")
	}
	if err = d.StartGateway(context.Background(), got); err == nil || x.startCalls != 1 {
		t.Fatal("repeated Start")
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
	if err = d.Close(); err != nil || x.input.closes != 1 || x.boot.closes != 1 {
		t.Fatal("Close is not idempotent", err)
	}
}

func TestPrivateGatewayDriverUncertainStartNeverWaitsOrRetries(t *testing.T) {
	for _, mode := range []string{"lost-start-reply", "wrong-start-id", "bootstrap-failure", "failed-start-commit", "lost-start-commit"} {
		t.Run(mode, func(t *testing.T) {
			x, d, o, opens := waitingDriverFixture(t)
			x.mode = mode
			_, err := privatechain.ExecuteGateway(context.Background(), x.p, x.w, x.g, d, x.commit, time.Now)
			if err == nil || o.calls != 0 || *opens != 0 {
				t.Fatal("uncertain Start entered readiness")
			}
			calls := x.startCalls
			g := x.saved
			g.Phase = "started"
			if _, err = d.ObserveGateway(context.Background(), g); err == nil || *opens != 0 || x.startCalls != calls {
				t.Fatal("uncertain Start was retried/adopted")
			}
		})
	}
}

func TestPrivateGatewayDriverWaitRefusesChangedRuntime(t *testing.T) {
	for _, mode := range []string{"inputs", "initializer", "engine", "image", "stopped", "mount", "participant", "canceled", "after-success"} {
		t.Run(mode, func(t *testing.T) {
			x, d, o, opens := waitingDriverFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			o.failures = 1
			o.hook = func() {
				switch mode {
				case "inputs":
					x.input.err = errors.New("SECRET changed inputs")
				case "initializer":
					x.input.programErr = errors.New("SECRET changed program")
				case "engine":
					x.f.engine.ID = "changed-engine"
				case "image":
					x.f.imageID = "sha256:" + strings.Repeat("f", 64)
				case "stopped":
					x.container["State"].(map[string]any)["Running"] = false
				case "mount":
					x.inventory[strings.Repeat("e", 64)] = []mount{{Type: "bind", Source: x.e.contract.root, Destination: "/foreign", RW: true}}
				case "participant":
					x.network["Containers"].(map[string]any)[strings.Repeat("e", 64)] = map[string]any{}
				case "canceled":
					cancel()
				case "after-success":
					o.failures = 0
					x.input.err = errors.New("changed while observing")
				}
			}
			got, err := privatechain.ExecuteGateway(ctx, x.p, x.w, x.g, d, x.commit, time.Now)
			if err == nil || got.Phase != "started" || x.saved.Phase != "started" || got.Observation != nil || strings.Contains(err.Error(), "SECRET") {
				t.Fatal(got.Phase, err)
			}
			if *opens != 1 || o.calls != 1 || o.closes != 1 || x.startCalls != 1 || x.boot.calls != 1 {
				t.Fatal("retried changed runtime", *opens, o.calls, o.closes, x.events)
			}
		})
	}
}

func TestPrivateGatewayDriverDeadlineAndFailedObservationPreserveState(t *testing.T) {
	for _, mode := range []string{"opening", "observing", "before-open-deadline", "close-failure", "nil-open", "handle-and-error"} {
		t.Run(mode, func(t *testing.T) {
			x, d, o, opens := waitingDriverFixture(t)
			// Only deadline cases need a short budget. Error/cleanup cases must
			// reach their selected hook, independent of race-detector overhead.
			if mode == "opening" || mode == "observing" || mode == "before-open-deadline" {
				d.readyTimeout = 15 * time.Millisecond
			}
			ctx := context.Background()
			switch mode {
			case "opening":
				d.openObserver = func(context.Context, privatechain.Gateway) (privateGatewayReadinessHandle, error) {
					return nil, errors.New("not ready")
				}
			case "observing":
				o.failures = 10000
			case "before-open-deadline":
				execute := x.e.q.d.execute
				x.e.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
					if x.saved.Phase == "started" {
						<-ctx.Done()
						return "", ctx.Err()
					}
					return execute(ctx, args, env)
				}
			case "close-failure":
				o.closeErr = errors.New("SECRET close failure")
			case "nil-open":
				d.openObserver = func(context.Context, privatechain.Gateway) (privateGatewayReadinessHandle, error) { return nil, nil }
			case "handle-and-error":
				d.openObserver = func(context.Context, privatechain.Gateway) (privateGatewayReadinessHandle, error) {
					*opens++
					return o, errors.New("partial open")
				}
			}
			at := time.Now()
			got, err := privatechain.ExecuteGateway(ctx, x.p, x.w, x.g, d, x.commit, time.Now)
			if err == nil || got.Phase != "started" || got.Observation != nil || time.Since(at) > 5*time.Second || x.startCalls != 1 || x.boot.calls != 1 {
				t.Fatal(got.Phase, err, x.events)
			}
			if mode == "opening" || mode == "observing" || mode == "before-open-deadline" {
				// ExecuteGateway intentionally reports unresolved rather than exposing
				// an internal deadline; it retains the committed started receipt.
				if !errors.Is(err, privatechain.ErrGatewayUnresolved) {
					t.Fatal(err)
				}
			}
			// The budget may expire during invariant inspection, before a handle
			// is opened. A nonexistent handle is not a leak. Every actual handle
			// must close exactly once, including an open that returns an error.
			if o.closes != *opens || *opens > 1 {
				t.Fatal("observer handle accounting differs", *opens, o.closes)
			}
			if mode == "before-open-deadline" && *opens != 0 {
				t.Fatal("deadline-before-open unexpectedly created a handle")
			}
			if (mode == "close-failure" || mode == "handle-and-error") && *opens != 1 {
				t.Fatal("selected cleanup failure was not exercised")
			}
		})
	}
}

func TestPrivateGatewayDriverRejectsInvalidWaitConfiguration(t *testing.T) {
	for _, times := range [][2]time.Duration{{0, time.Second}, {time.Minute, 0}, {11 * time.Minute, time.Second}, {time.Second, 2 * time.Second}} {
		x := newPrivateResourceFixture(t)
		x.e.q.d.times.ready = times[0]
		x.e.q.d.times.poll = times[1]
		if _, err := openPrivateGatewayDriver(context.Background(), x.e.q, x.e.q.base, x.e.contract.inputs, x.p, x.w, x.g); err == nil {
			t.Fatal("unbounded wait admitted")
		}
		if x.networkCalls+x.containerCalls+x.startCalls != 0 {
			t.Fatal("invalid configuration dispatched effects")
		}
	}
}

func TestPrivateGatewayDriverCallerDeadlineBoundsWaiting(t *testing.T) {
	x, d, o, opens := waitingDriverFixture(t)
	// Complete the ordinary resource fixture's Start before applying the caller's
	// short deadline, so this tests readiness rather than machine setup speed.
	if err := x.execute(context.Background()); err == nil || x.saved.Phase != "started" || x.startCalls != 1 {
		t.Fatal("fixture did not retain its successful Start", err)
	}
	d.startSucceeded = true
	o.failures = 10000
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	at := time.Now()
	got, err := d.ObserveGateway(ctx, x.saved)
	if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(got, privatechain.GatewayObservation{}) || time.Since(at) > time.Second {
		t.Fatal("caller deadline not preserved", err)
	}
	if x.startCalls != 1 || x.boot.calls != 1 || o.closes != *opens || *opens > 1 || x.saved.Phase != "started" {
		t.Fatal("deadline repeated effects, leaked handles or changed receipt")
	}
}
