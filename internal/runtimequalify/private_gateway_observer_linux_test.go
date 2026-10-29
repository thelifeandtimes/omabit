//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type privateObserverInputStub struct {
	privateInputStub
	programCalls int
	programError error
}

func (s *privateObserverInputStub) VerifyGatewayInitializer(_ context.Context, pin string) error {
	s.programCalls++
	if pin != privateGatewayInitializerSHA256 {
		return errors.New("wrong implementation pin")
	}
	return s.programError
}

type privateObserverStateStub struct {
	value         privatechain.GatewayStateObservation
	calls, closes int
	err           error
	changed       bool
}

func (s *privateObserverStateStub) Verify(context.Context) (privatechain.GatewayStateObservation, error) {
	s.calls++
	o := s.value
	o.Mounts = slices.Clone(o.Mounts)
	if s.changed && s.calls > 1 {
		o.Chain.Inode++
	}
	return o, s.err
}
func (s *privateObserverStateStub) Close() error { s.closes++; return nil }

type privateObserverFixture struct {
	x         *privateResourceFixture
	o         *privateGatewayObserver
	g         privatechain.Gateway
	inputs    *privateObserverInputStub
	state     *privateObserverStateStub
	rpc       privateGatewayRPCObservation
	mode      string
	execCalls int
	cancel    context.CancelFunc
}

func newPrivateObserverFixture(t *testing.T) *privateObserverFixture {
	t.Helper()
	x := newPrivateResourceFixture(t)
	r := x.e
	_, running, _ := privateContainerFixture(t, true)
	x.container["State"] = running["State"]
	x.container["NetworkSettings"] = running["NetworkSettings"]
	privateNetworkAttach(x.network)
	r.networkID = strings.Repeat("a", 64)
	r.containerID = strings.Repeat("b", 64)
	raw, err := json.Marshal(x.container["Mounts"])
	if err != nil {
		t.Fatal(err)
	}
	var mounts []mount
	if err = json.Unmarshal(raw, &mounts); err != nil {
		t.Fatal(err)
	}
	x.inventory[r.containerID] = mounts
	g := x.g
	g.Phase = "started"
	g.NetworkID = r.networkID
	g.ContainerID = r.containerID
	i := &privateObserverInputStub{privateInputStub: privateInputStub{inspection: g.Selection.Inputs}}
	r.inputs = i
	id := func(n uint64) pierstore.DirectoryIdentity {
		return pierstore.DirectoryIdentity{Device: x.w.Root.Device, Inode: n, Birth: "1000000000:1"}
	}
	s := &privateObserverStateStub{value: privatechain.GatewayStateObservation{Root: x.w.Root, Chain: id(14), Secrets: id(15), Mounts: slices.Clone(x.w.Observation.Mounts), Private: true}}
	boot := "01234567-89ab-cdef-0123-456789abcdef"
	host := projectnetwork.NamespaceIdentity{BootID: boot, Device: 4, Inode: 100}
	at := time.Now().UTC()
	o := &privateGatewayObserver{runtime: r, inputs: i, state: s, readBootID: func() (string, error) { return boot, nil }, readHostNamespace: func() (projectnetwork.NamespaceIdentity, error) { return host, nil }, now: func() time.Time { return at }}
	_, rpc := privateRPCFixture(t)
	rpc.Namespace.Device = "4"
	rpc.Namespace.Inode = "101"
	f := &privateObserverFixture{x: x, o: o, g: g, inputs: i, state: s, rpc: rpc}
	original := r.q.d.execute
	r.q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		a := args[4:]
		if len(a) >= 4 && a[0] == "container" && a[1] == "exec" {
			f.execCalls++
			if a[2] != r.containerID {
				t.Fatal("wrong exact container")
			}
			if a[3] == "/bin/stat" {
				if !reflect.DeepEqual(a[3:], []string{"/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
					t.Fatal("unexpected namespace command")
				}
				if f.mode == "bad-namespace" {
					return "invalid", nil
				}
				if f.mode == "namespace-change" && f.execCalls > 2 {
					return "4:102", nil
				}
				return "4:101", nil
			}
			want, err := privateGatewayRPCArgs(r.plan, r.containerID)
			if err != nil || !reflect.DeepEqual(a, want) {
				t.Fatal("changed probe command")
			}
			switch f.mode {
			case "rpc-error":
				return "untrusted private diagnostic", errors.New("untrusted error")
			case "process-change":
				x.container["State"].(map[string]any)["Pid"] = 5678
			case "program-change":
				i.programError = errors.New("changed initializer")
			case "input-change":
				i.inspection.Files++
			case "engine-change":
				x.f.engine.ID = "engine-changed"
			case "mount-change":
				x.inventory[strings.Repeat("e", 64)] = []mount{{Type: "bind", Source: r.contract.root, Destination: "/other", RW: true}}
			case "cancel":
				f.cancel()
			}
			return string(privateRPCBytes(t, f.rpc)), nil
		}
		return original(ctx, args, env)
	}
	t.Cleanup(func() { o.Close() })
	return f
}
func TestPrivateGatewayObserverFreshCombinedReadiness(t *testing.T) {
	f := newPrivateObserverFixture(t)
	before := f.g.Clone()
	o, err := f.o.ObserveGateway(context.Background(), f.g)
	if err != nil || !o.CompleteContract || !o.Private || !o.Synced || !o.RPCReadOnly || !o.Running {
		t.Fatal(o, err)
	}
	if !reflect.DeepEqual(f.g, before) || f.inputs.calls != 2 || f.inputs.programCalls != 2 || f.state.calls != 2 || f.execCalls != 3 {
		t.Fatal("missing brackets or changed journal", f.inputs.calls, f.inputs.programCalls, f.state.calls, f.execCalls)
	}
	ready, err := privatechain.ObserveGatewayReadiness(context.Background(), f.x.p, f.x.w, f.g, f.o, f.o.now)
	if err != nil || ready.Phase != "ready" {
		t.Fatal(ready.Phase, err)
	}
	if _, ok := any(f.o).(privatechain.GatewayDriver); ok {
		t.Fatal("observer exposes mutation authority")
	}
}
func TestPrivateGatewayObserverRefusesIncompleteOrChangedEvidence(t *testing.T) {
	for _, kind := range []string{"input", "initializer", "engine", "image", "stopped", "host-namespace", "host-change", "boot-change", "bad-namespace", "namespace-change", "rpc-namespace", "rpc-error", "unready", "state", "state-change", "not-private", "mount-overlap", "process-change", "program-change", "input-change", "engine-change", "mount-change", "expired", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			f := newPrivateObserverFixture(t)
			f.mode = kind
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.cancel = cancel
			switch kind {
			case "input":
				f.inputs.err = errors.New("changed input")
			case "initializer":
				f.inputs.programError = errors.New("wrong initializer")
			case "engine":
				f.x.f.engine.ID = "engine-other"
			case "image":
				f.x.f.imageID = "sha256:" + strings.Repeat("f", 64)
			case "stopped":
				f.x.container["State"].(map[string]any)["Running"] = false
			case "host-namespace":
				f.o.readHostNamespace = func() (projectnetwork.NamespaceIdentity, error) {
					return projectnetwork.NamespaceIdentity{BootID: "01234567-89ab-cdef-0123-456789abcdef", Device: 4, Inode: 101}, nil
				}
			case "host-change":
				original := f.o.readHostNamespace
				calls := 0
				f.o.readHostNamespace = func() (projectnetwork.NamespaceIdentity, error) {
					calls++
					ns, err := original()
					if calls > 1 {
						ns.Inode++
					}
					return ns, err
				}
			case "boot-change":
				calls := 0
				f.o.readBootID = func() (string, error) {
					calls++
					if calls > 1 {
						return "abcdef01-89ab-cdef-0123-456789abcdef", nil
					}
					return "01234567-89ab-cdef-0123-456789abcdef", nil
				}
			case "rpc-namespace":
				f.rpc.Namespace.Inode = "102"
			case "unready":
				f.rpc.Startup.Ready = false
			case "state":
				f.state.err = errors.New("changed storage")
			case "state-change":
				f.state.changed = true
			case "not-private":
				f.state.value.Private = false
			case "mount-overlap":
				f.x.inventory[strings.Repeat("e", 64)] = []mount{{Type: "bind", Source: f.x.e.contract.root, Destination: "/other", RW: true}}
			case "expired":
				calls := 0
				at := time.Now().UTC()
				f.o.now = func() time.Time {
					calls++
					if calls > 1 {
						return at.Add(26 * time.Second)
					}
					return at
				}
			}
			got, err := f.o.ObserveGateway(ctx, f.g)
			if !errors.Is(err, privatechain.ErrGatewayObservation) || !reflect.DeepEqual(got, privatechain.GatewayObservation{}) {
				t.Fatal("invalid observation escaped", got, err)
			}
			if f.x.startCalls != 0 || f.x.containerCalls != 0 || f.x.networkCalls != 0 || f.x.boot.calls != 0 {
				t.Fatal("readiness mutated resources")
			}
		})
	}
}
func TestPrivateGatewayObserverSelectionAndClose(t *testing.T) {
	for _, kind := range []string{"phase", "selection", "container", "network", "closed", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			f := newPrivateObserverFixture(t)
			g := f.g
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "phase":
				g.Phase = "gateway-created"
			case "selection":
				g.Selection.OperationID = "other-operation"
			case "container":
				g.ContainerID = strings.Repeat("d", 64)
			case "network":
				g.NetworkID = strings.Repeat("d", 64)
			case "closed":
				if err := f.o.Close(); err != nil {
					t.Fatal(err)
				}
				if err := f.o.Close(); err != nil {
					t.Fatal(err)
				}
				if f.inputs.closes != 1 || f.state.closes != 1 {
					t.Fatal("close count")
				}
			case "cancelled":
				cancel()
			}
			if _, err := f.o.ObserveGateway(ctx, g); err == nil {
				t.Fatal("changed caller accepted")
			}
			if f.execCalls != 0 || f.inputs.calls != 0 {
				t.Fatal("invalid selection touched runtime")
			}
		})
	}
}
func TestPrivateGatewayObserverPinMatchesReviewedSource(t *testing.T) {
	raw, err := os.ReadFile("../../scripts/qualification/private-chains/docker_gateway.cjs")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != privateGatewayInitializerSHA256 {
		t.Fatal("initializer changed without observer pin review")
	}
}

// Full sealed-input qualification is preparation, not the first mutable runtime
// observation. A slow first pass must not consume snapshot freshness time.
type privateSlowObservationInputs struct {
	*privateObserverInputStub
	clock                               *time.Time
	calls                               int
	staleSnapshot                       bool
	preparationBounded, snapshotBounded bool
}

func (i *privateSlowObservationInputs) Verify(ctx context.Context) (privatechain.RuntimeInputInspection, error) {
	i.calls++
	deadline, ok := ctx.Deadline()
	if i.calls == 1 {
		i.preparationBounded = ok && time.Until(deadline) > 25*time.Second && time.Until(deadline) <= 2*time.Minute
		*i.clock = i.clock.Add(41 * time.Second)
	} else {
		i.snapshotBounded = ok && time.Until(deadline) <= 25*time.Second
		if i.staleSnapshot {
			*i.clock = i.clock.Add(26 * time.Second)
		}
	}
	return i.privateObserverInputStub.Verify(ctx)
}
func TestPrivateGatewayObserverPreparationOutsideFreshnessWindow(t *testing.T) {
	for _, stale := range []bool{false, true} {
		f := newPrivateObserverFixture(t)
		clock := time.Now().UTC().Add(-41 * time.Second)
		firstMutableAt := clock.Add(41 * time.Second)
		slow := &privateSlowObservationInputs{privateObserverInputStub: f.inputs, clock: &clock, staleSnapshot: stale}
		f.o.inputs = slow
		f.o.runtime.inputs = slow
		f.o.now = func() time.Time { return clock }
		got, err := f.o.ObserveGateway(context.Background(), f.g)
		if stale {
			if !errors.Is(err, privatechain.ErrGatewayObservation) || !reflect.DeepEqual(got, privatechain.GatewayObservation{}) {
				t.Fatal("expired mutable observation accepted")
			}
		} else if err != nil || !got.ObservedAt.Equal(firstMutableAt) {
			t.Fatalf("slow preparation incorrectly consumed snapshot freshness: %v", err)
		}
		if slow.calls != 2 || !slow.preparationBounded || !slow.snapshotBounded {
			t.Fatal("full input passes or stage deadlines changed", slow.calls, slow.preparationBounded, slow.snapshotBounded)
		}
	}
}
