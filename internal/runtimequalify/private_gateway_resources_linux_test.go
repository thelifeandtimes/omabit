//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

type privateInputStub struct {
	inspection    privatechain.RuntimeInputInspection
	err           error
	programErr    error
	calls, closes int
}

func (i *privateInputStub) Verify(context.Context) (privatechain.RuntimeInputInspection, error) {
	i.calls++
	return i.inspection, i.err
}
func (i *privateInputStub) Close() error { i.closes++; return nil }
func (i *privateInputStub) VerifyGatewayInitializer(_ context.Context, pin string) error {
	if pin != privateGatewayInitializerSHA256 {
		return errors.New("unexpected initializer pin")
	}
	return i.programErr
}

type privateBootstrapStub struct {
	t             *testing.T
	saved         *privatechain.Gateway
	initialized   bool
	err           error
	initializeErr error
	calls, closes int
}

func (b *privateBootstrapStub) VerifyEmpty(context.Context) error {
	if b.initialized {
		return errors.New("already initialized")
	}
	return b.err
}
func (b *privateBootstrapStub) Initialize(_ context.Context, g privatechain.Gateway) error {
	b.calls++
	if g.Phase != "start-dispatched" || !reflect.DeepEqual(g, *b.saved) {
		b.t.Fatal("bootstrap preceded retained Start")
	}
	b.initialized = true
	return b.initializeErr
}
func (b *privateBootstrapStub) Close() error { b.closes++; return nil }

type privateResourceFixture struct {
	t                                        *testing.T
	f                                        *fakeEngine
	e                                        *privateGatewayResources
	p                                        privatechain.Plan
	w                                        privatechain.Workspace
	g, saved                                 privatechain.Gateway
	input                                    *privateInputStub
	boot                                     *privateBootstrapStub
	network, container                       map[string]any
	inventory                                map[string][]mount
	networkCalls, containerCalls, startCalls int
	events                                   []string
	mode                                     string
	cancel                                   context.CancelFunc
}

func newPrivateResourceFixture(t *testing.T) *privateResourceFixture {
	t.Helper()
	f, _, _ := managedFixture(t)
	p, w, g, _, n := privateNetworkFixture(t)
	_, v, _ := privateContainerFixture(t, false)
	f.engine.ID = g.Selection.EngineID
	f.imageID = runtimecandidate.ConfigDigest
	q := &qualifier{d: f.d, base: filepath.Join(f.d.home, ".local/share/urbit-runner/managed"), phase: "private-gateway"}
	q.r.Directory = filepath.Join(q.base, "private-gateway-client")
	q.config = filepath.Join(q.r.Directory, "docker-config")
	q.r.ImageID = runtimecandidate.ConfigDigest
	if err := os.MkdirAll(q.config, 0700); err != nil {
		t.Fatal(err)
	}
	inputRoot := filepath.Join(f.d.home, "sealed-fixture")
	c, err := newPrivateGatewayContainerContract(p, w, g, q.base, inputRoot, q.r.ImageID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(c.root, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(inputRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var configured, actual []any
	for _, m := range c.binds() {
		if m.Source != c.root {
			if err = os.MkdirAll(m.Source, 0700); err != nil {
				t.Fatal(err)
			}
		}
		configured = append(configured, map[string]any{"Type": m.Type, "Source": m.Source, "Target": m.Destination, "ReadOnly": !m.RW, "BindOptions": map[string]string{"Propagation": m.Propagation}})
		actual = append(actual, map[string]any{"Type": m.Type, "Source": m.Source, "Destination": m.Destination, "RW": m.RW, "Propagation": m.Propagation})
	}
	v["HostConfig"].(map[string]any)["Mounts"] = configured
	v["Mounts"] = actual
	x := &privateResourceFixture{t: t, f: f, p: p, w: w, g: g, saved: g, network: n, container: v, inventory: map[string][]mount{}}
	x.input = &privateInputStub{inspection: g.Selection.Inputs}
	x.boot = &privateBootstrapStub{t: t, saved: &x.saved}
	x.e = &privateGatewayResources{q: q, plan: p, workspace: w, contract: c, inputs: x.input, bootstrap: x.boot}
	original := q.d.execute
	q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if _, ok := ctx.Deadline(); !ok || len(args) < 6 || args[0] != "--host" || args[1] != "unix://"+q.socket || args[2] != "--config" || args[3] != q.config {
			t.Fatal("unbounded or ambient Docker command")
		}
		a := args[4:]
		switch {
		case reflect.DeepEqual(a, []string{"image", "inspect", runtimecandidate.Image}):
			return original(ctx, args, env)
		case reflect.DeepEqual(a, []string{"container", "ls", "--all", "--quiet", "--no-trunc"}):
			ids := []string{}
			for id := range x.inventory {
				ids = append(ids, id)
			}
			slices.Sort(ids)
			if x.mode == "duplicate-inventory" {
				ids = append(ids, strings.Repeat("d", 64), strings.Repeat("d", 64))
			}
			return strings.Join(ids, "\n"), nil
		case len(a) >= 5 && reflect.DeepEqual(a[:4], []string{"container", "inspect", "--format", projectMountFormat}):
			var lines []string
			for _, id := range a[4:] {
				raw, err := json.Marshal(projectContainerMounts{ID: id, Mounts: x.inventory[id]})
				if err != nil {
					t.Fatal(err)
				}
				lines = append(lines, string(raw))
			}
			return strings.Join(lines, "\n"), nil
		case reflect.DeepEqual(a, c.network.networkArgs()):
			x.networkCalls++
			x.events = append(x.events, "network")
			if x.saved.Phase != "network-dispatched" {
				t.Fatal("network create preceded journal")
			}
			if x.mode == "lost-network-id" {
				return "", errors.New("lost reply")
			}
			if x.mode == "network-id-error" {
				return strings.Repeat("a", 64), errors.New("client disconnected")
			}
			if x.mode == "network-id-cancel" {
				x.cancel()
				return strings.Repeat("a", 64), nil
			}
			if x.mode == "engine-replaced" {
				f.engine.ID = "foreign-engine"
			}
			return strings.Repeat("a", 64) + "\n", nil
		case reflect.DeepEqual(a, []string{"network", "inspect", strings.Repeat("a", 64)}):
			if x.mode == "input-changed-during-start-checks" && x.saved.Phase == "start-dispatched" {
				x.input.inspection.Bytes++
			}
			if x.mode == "initializer-changed-during-start-checks" && x.saved.Phase == "start-dispatched" {
				x.input.programErr = errors.New("initializer changed after preflight")
			}
			return string(privateNetworkBytes(t, x.network)), nil
		case reflect.DeepEqual(a, []string{"container", "inspect", strings.Repeat("b", 64)}):
			return string(privateContainerData(t, x.container)), nil
		case len(a) >= 2 && a[0] == "container" && a[1] == "create":
			want, err := c.containerArgs(strings.Repeat("a", 64))
			if err != nil || !reflect.DeepEqual(a, want) {
				t.Fatal("container command differs")
			}
			x.containerCalls++
			x.events = append(x.events, "container")
			if x.saved.Phase != "gateway-dispatched" {
				t.Fatal("container create preceded journal")
			}
			x.inventory[strings.Repeat("b", 64)] = c.binds()
			if x.mode == "gateway-id-error" {
				return strings.Repeat("b", 64), errors.New("client disconnected")
			}
			if x.mode == "gateway-id-cancel" {
				x.cancel()
				return strings.Repeat("b", 64), nil
			}
			if x.mode == "container-replaced" {
				x.container["Config"].(map[string]any)["Labels"] = map[string]string{"foreign": "true"}
			}
			return strings.Repeat("b", 64) + "\n", nil
		case reflect.DeepEqual(a, []string{"container", "start", strings.Repeat("b", 64)}):
			x.startCalls++
			x.events = append(x.events, "start")
			if x.saved.Phase != "start-dispatched" || !x.boot.initialized {
				t.Fatal("Start preceded journal/bootstrap")
			}
			if x.mode == "lost-start-reply" {
				return "", errors.New("lost reply")
			}
			if x.mode == "wrong-start-id" {
				return strings.Repeat("e", 64), nil
			}
			if x.mode == "start-cancel" {
				x.cancel()
				return strings.Repeat("b", 64), nil
			}
			return strings.Repeat("b", 64) + "\n", nil
		default:
			t.Fatal("unexpected engine authority", a)
			return "", errors.New("unreachable")
		}
	}
	t.Cleanup(func() { _ = x.e.Close() })
	return x
}
func (x *privateResourceFixture) commit(expected string, next privatechain.Gateway) error {
	if expected != x.saved.Phase {
		x.t.Fatal("commit mismatch")
	}
	checked, err := x.saved.Advance(x.p, x.w, expected, next.Phase, next.NetworkID, next.ContainerID, next.Observation)
	if err != nil || !reflect.DeepEqual(checked, next) {
		x.t.Fatal("invalid controller transition", err)
	}
	if x.mode == "failed-start-commit" && next.Phase == "start-dispatched" {
		return errors.New("commit failed")
	}
	x.saved = next
	if x.mode == "lost-start-commit" && next.Phase == "start-dispatched" {
		return errors.New("commit reply lost")
	}
	if next.Phase == "start-dispatched" && x.mode == "bootstrap-failure" {
		x.boot.initializeErr = errors.New("bootstrap write failure")
	}
	return nil
}

// Readiness is deliberately absent from the resource adapter. The fixture does
// not manufacture a running/chain observation after simulating Start.
type privateResourceDriver struct{ *privateGatewayResources }

func (privateResourceDriver) ObserveGateway(context.Context, privatechain.Gateway) (privatechain.GatewayObservation, error) {
	return privatechain.GatewayObservation{}, errors.New("runtime observation not supplied")
}
func (x *privateResourceFixture) execute(ctx context.Context) error {
	_, err := privatechain.ExecuteGateway(ctx, x.p, x.w, x.g, privateResourceDriver{x.e}, x.commit, time.Now)
	return err
}

func TestPrivateGatewayResourcesJournalOrderingAndUncertainEffects(t *testing.T) {
	for _, mode := range []string{"started", "lost-network-id", "network-id-error", "network-id-cancel", "gateway-id-error", "gateway-id-cancel", "engine-replaced", "container-replaced", "failed-start-commit", "lost-start-commit", "bootstrap-failure", "input-changed-during-start-checks", "initializer-changed-during-start-checks", "lost-start-reply", "wrong-start-id", "start-cancel"} {
		t.Run(mode, func(t *testing.T) {
			x := newPrivateResourceFixture(t)
			x.mode = mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			x.cancel = cancel
			if err := x.execute(ctx); err == nil {
				t.Fatal("resource-only driver fabricated readiness")
			}
			switch mode {
			case "lost-network-id":
				if x.saved.Phase != "network-dispatched" || x.saved.NetworkID != "" || x.containerCalls != 0 {
					t.Fatal("unknown creation adopted")
				}
			case "network-id-error", "network-id-cancel", "engine-replaced":
				if x.saved.Phase != "network-created" || x.saved.NetworkID != strings.Repeat("a", 64) || x.containerCalls != 0 {
					t.Fatal("known network ID discarded or later work ran")
				}
			case "gateway-id-error", "gateway-id-cancel", "container-replaced", "failed-start-commit":
				if x.saved.Phase != "gateway-created" || x.saved.ContainerID != strings.Repeat("b", 64) || x.boot.calls != 0 || x.startCalls != 0 {
					t.Fatal("known container lost or startup ran")
				}
			case "lost-start-commit", "bootstrap-failure", "input-changed-during-start-checks", "initializer-changed-during-start-checks":
				if x.saved.Phase != "start-dispatched" || x.startCalls != 0 {
					t.Fatal("uncertain startup repeated")
				}
				if mode == "bootstrap-failure" && x.boot.calls != 1 || mode != "bootstrap-failure" && x.boot.calls != 0 {
					t.Fatal("bootstrap failure boundary differs")
				}
			default:
				phase := "start-dispatched"
				if mode == "started" {
					phase = "started"
				}
				if x.saved.Phase != phase || x.startCalls != 1 || x.boot.calls != 1 {
					t.Fatal("single-start contract differs", x.saved.Phase, x.events)
				}
			}
			n, c, s, b := x.networkCalls, x.containerCalls, x.startCalls, x.boot.calls
			if _, err := privatechain.ExecuteGateway(context.Background(), x.p, x.w, x.saved, privateResourceDriver{x.e}, x.commit, time.Now); err == nil || n != x.networkCalls || c != x.containerCalls || s != x.startCalls || b != x.boot.calls {
				t.Fatal("uncertain receipt repeated effects")
			}
			if err := x.e.Close(); err != nil {
				t.Fatal(err)
			}
			if err := x.e.Close(); err != nil {
				t.Fatal(err)
			}
			if x.input.closes != 1 || x.boot.closes != 1 {
				t.Fatal("owned handles not closed exactly once")
			}
			if err := x.e.VerifyPreparation(context.Background(), x.g); err == nil {
				t.Fatal("closed resource handle retained authority")
			}
		})
	}
}

func TestPrivateGatewayResourcesRefusesStalePreconditionsAndSelections(t *testing.T) {
	for _, kind := range []string{"input-error", "initializer", "input-mismatch", "input-authority", "workspace-error", "image", "engine", "port", "canceled", "operation", "phase", "foreign-resource"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateResourceFixture(t)
			g := x.g
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "initializer":
				x.input.programErr = errors.New("unqualified initializer")
			case "input-error":
				x.input.err = errors.New("input changed")
			case "input-mismatch":
				x.input.inspection.Bytes++
			case "input-authority":
				x.input.inspection.RuntimeAuthorized = true
			case "workspace-error":
				x.boot.err = errors.New("workspace changed")
			case "image":
				x.f.imageID = "sha256:" + strings.Repeat("e", 64)
			case "engine":
				x.f.engine.ID = "foreign-engine"
			case "port":
				x.e.q.d.portAvailable = func(string, uint16) bool { return false }
			case "canceled":
				cancel()
			case "operation":
				g.Selection.OperationID = "other-operation"
			case "phase":
				g.Phase = "started"
				g.NetworkID = strings.Repeat("a", 64)
				g.ContainerID = strings.Repeat("b", 64)
			case "foreign-resource":
				g.Phase = "network-created"
				g.NetworkID = strings.Repeat("d", 64)
			}
			if err := x.e.VerifyPreparation(ctx, g); err == nil {
				t.Fatal("invalid precondition admitted")
			}
			if x.networkCalls+x.containerCalls+x.startCalls+x.boot.calls != 0 {
				t.Fatal("refusal performed mutation")
			}
		})
	}
}

func TestPrivateGatewayResourcesMountOverlapAndSharedReadOnlyInputs(t *testing.T) {
	for _, kind := range []string{"none", "unrelated", "readonly-input", "writable-input", "mixed-input", "root", "child", "ancestor", "alias", "volume", "missing", "duplicate-inventory", "changed-inventory"} {
		t.Run(kind, func(t *testing.T) {
			x := newPrivateResourceFixture(t)
			source := filepath.Join(x.f.d.home, "unrelated")
			if err := os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			m := mount{Type: "bind", Source: source, Destination: "/elsewhere", RW: true, Propagation: "rprivate"}
			switch kind {
			case "readonly-input", "writable-input", "mixed-input":
				m.Source = x.e.contract.inputs
				m.RW = kind == "writable-input"
			case "root":
				m.Source = x.e.contract.root
			case "child":
				m.Source = filepath.Join(x.e.contract.root, "nested")
				if err := os.Mkdir(m.Source, 0700); err != nil {
					t.Fatal(err)
				}
			case "ancestor":
				m.Source = filepath.Dir(x.e.contract.root)
			case "alias":
				m.Source = filepath.Join(x.f.d.home, "root-alias")
				if err := os.Symlink(x.e.contract.root, m.Source); err != nil {
					t.Fatal(err)
				}
			case "volume":
				m.Type = "volume"
			case "missing":
				m.Source = filepath.Join(x.f.d.home, "missing")
			case "duplicate-inventory":
				x.mode = kind
			}
			if kind != "none" {
				x.inventory[strings.Repeat("d", 64)] = []mount{m}
			}
			if kind == "mixed-input" {
				m.RW = true
				m.Destination = "/write"
				x.inventory[strings.Repeat("d", 64)] = append(x.inventory[strings.Repeat("d", 64)], m)
			}
			if kind == "changed-inventory" {
				base := x.e.q.d.execute
				calls := 0
				x.e.q.d.execute = func(ctx context.Context, a, e []string) (string, error) {
					out, err := base(ctx, a, e)
					if len(a) > 8 && a[4] == "container" && a[5] == "inspect" && a[6] == "--format" {
						calls++
						if calls == 1 {
							x.inventory[strings.Repeat("e", 64)] = nil
						}
					}
					return out, err
				}
			}
			err := x.e.VerifyPreparation(context.Background(), x.g)
			pass := kind == "none" || kind == "unrelated" || kind == "readonly-input"
			if (err == nil) != pass {
				t.Fatal("mount admission differs", kind, err)
			}
			if x.networkCalls+x.containerCalls+x.startCalls+x.boot.calls != 0 {
				t.Fatal("mount preflight performed mutation")
			}
		})
	}
}

func TestPrivateGatewayResourcesEffectMethodsCannotRepeatOrSkipPhases(t *testing.T) {
	x := newPrivateResourceFixture(t)
	if _, err := x.e.CreateNetwork(context.Background(), x.g); err == nil {
		t.Fatal("create before dispatch")
	}
	if _, err := x.e.CreateGateway(context.Background(), x.g); err == nil {
		t.Fatal("container before dispatch")
	}
	if err := x.e.StartGateway(context.Background(), x.g); err == nil {
		t.Fatal("Start before dispatch")
	}
	x.mode = "started"
	if err := x.execute(context.Background()); err == nil {
		t.Fatal("missing observer accepted")
	}
	if x.saved.Phase != "started" || x.startCalls != 1 {
		t.Fatal("fixture did not reach the single-start boundary")
	}
	starts := x.startCalls
	g := x.saved
	g.Phase = "start-dispatched"
	if err := x.e.StartGateway(context.Background(), g); err == nil || x.startCalls != starts {
		t.Fatal("Start repeated")
	}
	if _, err := x.e.CreateGateway(context.Background(), g); err == nil {
		t.Fatal("creation after Start")
	}
}

func TestPrivateGatewayResourcesUnknownCreationCannotRepeat(t *testing.T) {
	x := newPrivateResourceFixture(t)
	x.mode = "lost-network-id"
	if err := x.execute(context.Background()); err == nil || x.saved.Phase != "network-dispatched" {
		t.Fatal("fixture did not retain unknown creation")
	}
	if id, err := x.e.CreateNetwork(context.Background(), x.saved); err == nil || id != "" || x.networkCalls != 1 {
		t.Fatal("unknown creation repeated or adopted an ID")
	}
}

func TestPrivateGatewayResourcesCloseSerializesWithPreflight(t *testing.T) {
	x := newPrivateResourceFixture(t)
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { _ = x.e.VerifyPreparation(context.Background(), x.g) })
	}
	wg.Go(func() { _ = x.e.Close() })
	wg.Wait()
	if x.input.closes != 1 || x.boot.closes != 1 {
		t.Fatal("Close did not release both handles exactly once")
	}
	if err := x.e.VerifyPreparation(context.Background(), x.g); err == nil {
		t.Fatal("closed handle allowed preflight")
	}
}
