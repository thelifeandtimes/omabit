//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateHTTPHeldStub struct {
	h      *privateHTTPHarness
	name   string
	closes int
}

func (v *privateHTTPHeldStub) Verify(ctx context.Context) error { return v.h.event(v.name) }
func (v *privateHTTPHeldStub) Close() error                     { v.closes++; return nil }

type privateHTTPHarness struct {
	d                                     *privateMemberHTTPDriver
	p                                     privatechain.Plan
	w                                     privatechain.Workspace
	at                                    time.Time
	events                                []string
	fail, cancelAt                        string
	cancel                                context.CancelFunc
	processCalls, gatewayCalls, httpCalls int
	mutateProcess                         func(*privatechain.MemberBirthProcess, int)
	mutateGateway                         func(*privatechain.GatewayObservation, int)
	identity                              string
}

func (h *privateHTTPHarness) event(name string) error {
	h.events = append(h.events, name)
	if name == h.cancelAt {
		h.cancel()
	}
	if name == h.fail {
		return errors.New("PRIVATE detail")
	}
	return nil
}
func privateHTTPFixture(t *testing.T) *privateHTTPHarness {
	t.Helper()
	l := newPrivateLaunchHarness(t)
	s, err := l.run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h := &privateHTTPHarness{p: l.d.contract.selection.Plan, w: l.d.contract.selection.Workspace, at: l.at.Add(time.Second)}
	d := &privateMemberHTTPDriver{initial: s.Clone(), now: func() time.Time { return h.at }}
	h.d = d
	for _, m := range h.p.Members {
		if m.InstanceID == s.InstanceID {
			d.endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
			d.ship = m.Ship
		}
	}
	h.identity = d.ship
	for _, name := range []string{"pier", "pill", "key"} {
		v := &privateHTTPHeldStub{h: h, name: name}
		d.held = append(d.held, v)
		d.closers = append(d.closers, v)
	}
	d.process = func(ctx context.Context) (privatechain.MemberBirthProcess, error) {
		h.processCalls++
		o := s.Running.Birth.After
		o.ObservedAt = h.at
		if h.mutateProcess != nil {
			h.mutateProcess(&o, h.processCalls)
		}
		return o, h.event(fmt.Sprintf("process-%d", h.processCalls))
	}
	d.gateway = func(ctx context.Context) (privatechain.GatewayObservation, error) {
		h.gatewayCalls++
		o := s.Running.Gateway.Clone()
		o.ObservedAt = h.at
		if h.mutateGateway != nil {
			h.mutateGateway(&o, h.gatewayCalls)
		}
		return o, h.event(fmt.Sprintf("gateway-%d", h.gatewayCalls))
	}
	d.host = func(ctx context.Context, endpoint string) (string, error) {
		if endpoint != d.endpoint {
			t.Fatal("wrong HTTP endpoint")
		}
		h.httpCalls++
		return h.identity, h.event("http")
	}
	t.Cleanup(func() { d.Close() })
	return h
}
func (h *privateHTTPHarness) collect(ctx context.Context) (privatechain.MemberHTTPReadiness, error) {
	return privatechain.CollectMemberHTTPReadiness(ctx, h.p, h.w, h.d.initial, h.d, h.d.now)
}
func TestPrivateMemberHTTPDriverCollectsExactIdentity(t *testing.T) {
	h := privateHTTPFixture(t)
	r, err := h.collect(context.Background())
	if err != nil || r.Validate(h.p, h.w, h.d.initial) != nil || h.httpCalls != 1 || h.processCalls != 4 || h.gatewayCalls != 2 {
		t.Fatal("collection failed", err)
	}
	if r.Before.Pier != h.d.initial.Running.Birth.Pier || r.After.Process.StartedAt != h.d.initial.Running.StartedAt {
		t.Fatal("lost captured identities")
	}
	retained := h.d.last.Clone()
	r.After.Gateway.Mounts[0].Inode++
	if !reflect.DeepEqual(retained, *h.d.last) {
		t.Fatal("returned observation aliases driver")
	}
	if out, err := h.d.ReadHostIdentity(context.Background(), h.d.endpoint); err == nil || out != "" || h.httpCalls != 1 {
		t.Fatal("HTTP read repeated")
	}
}
func TestPrivateMemberHTTPDriverFailuresAndCancellation(t *testing.T) {
	for _, name := range []string{"pier", "pill", "key", "process-1", "gateway-1", "process-2", "http", "process-3", "gateway-2", "process-4"} {
		for _, cancel := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/cancel=%v", name, cancel), func(t *testing.T) {
				h := privateHTTPFixture(t)
				ctx, c := context.WithCancel(context.Background())
				defer c()
				h.cancel = c
				if cancel {
					h.cancelAt = name
				} else {
					h.fail = name
				}
				r, err := h.collect(ctx)
				if err == nil || !reflect.DeepEqual(r, privatechain.MemberHTTPReadiness{}) || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("failure returned partial/private result", err)
				}
				if (name == "pier" || name == "pill" || name == "key" || strings.HasSuffix(name, "-1") || name == "process-2") && h.httpCalls != 0 {
					t.Fatal("unsafe HTTP read")
				}
			})
		}
	}
}
func TestPrivateMemberHTTPDriverRejectsChangedObservations(t *testing.T) {
	for _, kind := range []string{"restart", "namespace", "container", "contract", "stopped", "incomplete", "stale", "future", "before-launch", "gateway-chain", "gateway-stale", "gateway-future", "wrong-ship"} {
		for _, late := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/after-http=%v", kind, late), func(t *testing.T) {
				h := privateHTTPFixture(t)
				h.mutateProcess = func(o *privatechain.MemberBirthProcess, n int) {
					if late && n < 3 {
						return
					}
					switch kind {
					case "restart":
						o.StartedAt = o.StartedAt.Add(time.Second)
					case "namespace":
						o.Namespace.Inode++
					case "container":
						o.ContainerID = strings.Repeat("a", 64)
					case "contract":
						o.ContractDigest = strings.Repeat("a", 64)
					case "stopped":
						o.Running = false
					case "incomplete":
						o.CompleteContract = false
					case "stale":
						o.ObservedAt = h.at.Add(-31 * time.Second)
					case "future":
						o.ObservedAt = h.at.Add(6 * time.Second)
					case "before-launch":
						o.ObservedAt = h.d.initial.Running.ObservedAt.Add(-time.Nanosecond)
					}
				}
				h.mutateGateway = func(o *privatechain.GatewayObservation, n int) {
					if late && n < 2 {
						return
					}
					switch kind {
					case "gateway-chain":
						o.HeadHash = strings.Repeat("a", 64)
					case "gateway-stale":
						o.ObservedAt = h.at.Add(-31 * time.Second)
					case "gateway-future":
						o.ObservedAt = h.at.Add(6 * time.Second)
					}
				}
				if kind == "wrong-ship" {
					h.identity = "PRIVATE wrong identity"
				}
				r, err := h.collect(context.Background())
				if err == nil || !reflect.DeepEqual(r, privatechain.MemberHTTPReadiness{}) || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("changed runtime accepted", err)
				}
				if !late && kind != "wrong-ship" && h.httpCalls != 0 {
					t.Fatal("changed runtime reached HTTP")
				}
			})
		}
	}
}
func TestPrivateMemberHTTPDriverRejectsUnobservedOrWrongEndpoint(t *testing.T) {
	for _, kind := range []string{"unobserved", "wrong-endpoint", "stale", "closed", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			h := privateHTTPFixture(t)
			ctx, c := context.WithCancel(context.Background())
			defer c()
			if kind != "unobserved" {
				if _, err := h.d.ObserveRuntime(ctx, h.d.initial); err != nil {
					t.Fatal(err)
				}
			}
			endpoint := h.d.endpoint
			switch kind {
			case "wrong-endpoint":
				endpoint = "http://127.0.0.1:9999/~/host"
			case "stale":
				h.at = h.at.Add(31 * time.Second)
			case "closed":
				h.d.Close()
			case "cancelled":
				c()
			}
			if out, err := h.d.ReadHostIdentity(ctx, endpoint); err == nil || out != "" || h.httpCalls != 0 {
				t.Fatal("unsafe HTTP call")
			}
		})
	}
}
func TestPrivateMemberHTTPDriverChangedSelectionAndClose(t *testing.T) {
	h := privateHTTPFixture(t)
	s := h.d.initial.Clone()
	s.OperationID = "other"
	if _, err := h.d.ObserveRuntime(context.Background(), s); err == nil || len(h.events) != 0 {
		t.Fatal("changed selection reached runtime")
	}
	a, b := errors.New("a"), errors.New("b")
	x, y := &privateMemberCloseStub{err: a}, &privateMemberCloseStub{err: b}
	h.d.closers = append(h.d.closers, []io.Closer{x, y}...)
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.d.Close(); !errors.Is(err, a) || !errors.Is(err, b) {
				t.Error("lost close errors")
			}
		}()
	}
	wg.Wait()
	for _, v := range h.d.held {
		if v.(*privateHTTPHeldStub).closes != 1 {
			t.Fatal("handle close repeated")
		}
	}
	if x.count != 1 || y.count != 1 {
		t.Fatal("close repeated")
	}
	if _, err := h.d.ObserveRuntime(context.Background(), h.d.initial); err == nil {
		t.Fatal("closed driver observed")
	}
}
func TestPrivateMemberHTTPFactoryRefusesInvalidBeforeIO(t *testing.T) {
	for _, kind := range []string{"legacy", "planned", "no-birth", "digest", "workspace", "root", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			h := privateHTTPFixture(t)
			s := h.d.initial.Clone()
			w := h.w.Clone()
			root := "/synthetic/private-state"
			q, _ := privateClientFixture(t)
			q.d.execute = func(context.Context, []string, []string) (string, error) {
				t.Fatal("invalid selection reached engine")
				return "", nil
			}
			ctx, c := context.WithCancel(context.Background())
			defer c()
			switch kind {
			case "legacy":
				s.Version = 1
				s.Running.Birth = nil
			case "planned":
				s.Phase = "planned"
			case "no-birth":
				s.Running.Birth = nil
			case "digest":
				s.ContractDigest = strings.Repeat("a", 64)
			case "workspace":
				w.Phase = "planned"
			case "root":
				root = "relative"
			case "cancel":
				c()
			}
			before, err := os.ReadDir(q.base)
			if err != nil {
				t.Fatal(err)
			}
			if d, err := openPrivateMemberHTTPReadiness(ctx, root, h.p, w, s, q.d); err == nil || d != nil {
				t.Fatal("invalid factory accepted")
			}
			after, err := os.ReadDir(q.base)
			if err != nil || len(before) != len(after) {
				t.Fatal("factory wrote metadata")
			}
		})
	}
}
