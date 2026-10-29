//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateObserverOverlapInputs struct {
	base                       privateGatewayObserverInputs
	calls                      atomic.Int32
	closes                     atomic.Int32
	rpcReturned                *atomic.Bool
	beforeRPC                  atomic.Bool
	started, release, finished chan struct{}
	fail                       bool
}

func (s *privateObserverOverlapInputs) Verify(ctx context.Context) (privatechain.RuntimeInputInspection, error) {
	if s.calls.Add(1) == 2 {
		if !s.rpcReturned.Load() {
			s.beforeRPC.Store(true)
		}
		close(s.started)
		defer close(s.finished)
		select {
		case <-ctx.Done():
			return privatechain.RuntimeInputInspection{}, ctx.Err()
		case <-s.release:
		}
		if s.fail {
			return privatechain.RuntimeInputInspection{}, errors.New("synthetic input change")
		}
	}
	return s.base.Verify(ctx)
}
func (s *privateObserverOverlapInputs) VerifyGatewayInitializer(ctx context.Context, pin string) error {
	return s.base.VerifyGatewayInitializer(ctx, pin)
}
func (s *privateObserverOverlapInputs) Close() error { s.closes.Add(1); return s.base.Close() }

func TestPrivateGatewayObserverJoinsOverlappingVerification(t *testing.T) {
	for _, kind := range []string{"success", "input-failure", "runtime-failure", "cancel", "close"} {
		t.Run(kind, func(t *testing.T) {
			x, g, _, _, _ := privateMemberObserverFixture(t, "boot", "running", 1)
			o := x.f.o
			q := o.runtime.q
			var rpcReturned atomic.Bool
			input := &privateObserverOverlapInputs{base: o.inputs, rpcReturned: &rpcReturned, started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{}), fail: kind == "input-failure"}
			o.inputs = input
			o.runtime.inputs = input
			original := q.d.execute
			concurrentRuntime := make(chan struct{})
			var seen atomic.Bool
			q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if rpcReturned.Load() && seen.CompareAndSwap(false, true) {
					select {
					case <-input.started:
					case <-ctx.Done():
						return "", ctx.Err()
					}
					close(concurrentRuntime)
					if kind == "runtime-failure" {
						return "", errors.New("synthetic runtime failure")
					}
				}
				out, err := original(ctx, args, env)
				if len(a) >= 4 && a[0] == "container" && a[1] == "exec" && a[2] == o.runtime.containerID && a[3] != "/bin/stat" {
					rpcReturned.Store(true)
				}
				return out, err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			type result struct {
				observation privatechain.GatewayObservation
				err         error
			}
			done := make(chan result, 1)
			go func() { v, e := o.ObserveGateway(ctx, g); done <- result{v, e} }()
			select {
			case <-concurrentRuntime:
			case <-ctx.Done():
				t.Fatal("final runtime check did not overlap post-RPC input verification")
			}
			if input.beforeRPC.Load() {
				t.Fatal("input verification ran before RPC returned")
			}
			var closed chan error
			if kind == "close" {
				closed = make(chan error, 1)
				go func() { closed <- o.Close() }()
				select {
				case <-closed:
					t.Fatal("Close released handles during verification")
				default:
				}
			}
			switch kind {
			case "cancel":
				cancel()
			case "runtime-failure": // The observer must cancel/join without a manual release.
			default:
				select {
				case <-done:
					t.Fatal("returned before full input verification completed")
				default:
				}
				close(input.release)
			}
			var got result
			select {
			case got = <-done:
			case <-ctx.Done():
				// Cancellation is itself one tested outcome; allow the joined return.
				select {
				case got = <-done:
				case <-time.After(time.Second):
					t.Fatal("observer did not join verifier")
				}
			}
			select {
			case <-input.finished:
			default:
				t.Fatal("observer returned with live input verifier")
			}
			pass := kind == "success" || kind == "close"
			if pass {
				if got.err != nil || !got.observation.CompleteContract {
					t.Fatal(got.err)
				}
			} else if got.err == nil || !reflect.DeepEqual(got.observation, privatechain.GatewayObservation{}) {
				t.Fatal("partial observation escaped")
			}
			if closed != nil {
				select {
				case err := <-closed:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(time.Second):
					t.Fatal("Close did not complete")
				}
				if input.closes.Load() != 1 {
					t.Fatal("handles closed more than once")
				}
			}
		})
	}
}
