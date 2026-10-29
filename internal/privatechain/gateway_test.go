package privatechain

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

func gatewayFixture(t *testing.T) (Plan, Workspace, Gateway) {
	t.Helper()
	p, w := workspaceFixture(t)
	w.Phase, w.Root = "observed", workspaceID(10)
	o := workspaceObservation()
	w.Observation = &o
	i := RuntimeInputInspection{Format: RuntimeInputFormat, ManifestSHA256: strings.Repeat("1", 64), FoundationDigest: p.FoundationDigest, Files: 12352, Bytes: 834035190}
	g, err := NewGateway(p, w, "gateway-operation", "engine-private-fixture", i)
	if err != nil {
		t.Fatal(err)
	}
	return p, w, g
}

func gatewayObservation(p Plan, w Workspace, g Gateway) GatewayObservation {
	ns := projectnetwork.NamespaceIdentity{BootID: "01234567-89ab-cdef-0123-456789abcdef", Device: 4, Inode: 101}
	host := ns
	host.Inode = 100
	o := GatewayObservation{Selection: g.Selection, ObservedAt: workspaceNow,
		NetworkID: g.NetworkID, ContainerID: g.ContainerID, ImageID: runtimecandidate.ConfigDigest,
		Namespace: ns, HostNamespace: host, Root: w.Root, Chain: workspaceID(14), Secrets: workspaceID(15),
		Mounts: slices.Clone(w.Observation.Mounts), Foundation: p.Foundation, Head: 0x50, HeadHash: "0x" + strings.Repeat("a", 64),
		CompleteContract: true, Running: true, Private: true, Synced: true, RPCReadOnly: true}
	o.Foundation.Contracts = slices.Clone(p.Foundation.Contracts)
	for n, m := range p.Members {
		o.PublicKeys = append(o.PublicKeys, GatewayPublicKey{Ship: m.Ship, Auth: "0x" + strings.Repeat(string(rune('b'+n*2)), 64), Crypt: "0x" + strings.Repeat(string(rune('c'+n*2)), 64), Life: 1, Suite: 1})
		o.Publications = append(o.Publications, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: "127.0.0.1", HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	return o
}

// The harness is an inert driver. It has no engine/filesystem/network access.
// Failures can occur after a known effect or after a durable commit whose reply
// was lost; the retained receipt is deliberately distinct from the return value.
type gatewayHarness struct {
	t                                                   *testing.T
	p                                                   Plan
	w                                                   Workspace
	saved                                               Gateway
	events                                              []string
	fail                                                string
	loseCommit                                          bool
	loseID                                              bool
	badID                                               bool
	idAt                                                string
	cancel                                              context.CancelFunc
	cancelAt                                            string
	mutateObservation                                   func(*GatewayObservation)
	mutateCommit                                        bool
	networkCalls, createCalls, startCalls, observations int
}

func newGatewayHarness(t *testing.T) (*gatewayHarness, Gateway) {
	p, w, g := gatewayFixture(t)
	return &gatewayHarness{t: t, p: p, w: w, saved: g.Clone()}, g
}

func (h *gatewayHarness) event(name string) error {
	h.events = append(h.events, name)
	if h.cancelAt == name {
		h.cancel()
	}
	if h.fail == name {
		return errors.New("SECRET driver diagnostic")
	}
	return nil
}
func (h *gatewayHarness) VerifyPreparation(_ context.Context, g Gateway) error {
	if g.Validate(h.p, h.w) != nil {
		h.t.Fatal("invalid driver selection")
	}
	return h.event("verify-" + g.Phase)
}
func (h *gatewayHarness) CreateNetwork(_ context.Context, g Gateway) (string, error) {
	h.networkCalls++
	if h.saved.Phase != "network-dispatched" || g != h.saved {
		h.t.Fatal("network effect preceded commit")
	}
	err := h.event("create-network")
	if h.loseID && h.idAt != "gateway" {
		return "", err
	}
	if h.badID && h.idAt != "gateway" {
		return "short", err
	}
	return strings.Repeat("2", 64), err
}
func (h *gatewayHarness) InspectNetwork(_ context.Context, g Gateway) error {
	if g.Phase != "network-created" || !hash(g.NetworkID, false) {
		h.t.Fatal("invalid network inspection")
	}
	return h.event("inspect-network")
}
func (h *gatewayHarness) CreateGateway(_ context.Context, g Gateway) (string, error) {
	h.createCalls++
	if h.saved.Phase != "gateway-dispatched" || g != h.saved {
		h.t.Fatal("container effect preceded commit")
	}
	err := h.event("create-gateway")
	if h.loseID && h.idAt != "network" {
		return "", err
	}
	if h.badID && h.idAt != "network" {
		return "short", err
	}
	return strings.Repeat("3", 64), err
}
func (h *gatewayHarness) InspectGateway(_ context.Context, g Gateway) error {
	if g.Phase != "gateway-created" || !hash(g.ContainerID, false) {
		h.t.Fatal("invalid created inspection")
	}
	return h.event("inspect-gateway")
}
func (h *gatewayHarness) StartGateway(_ context.Context, g Gateway) error {
	h.startCalls++
	if h.saved.Phase != "start-dispatched" || g != h.saved {
		h.t.Fatal("start preceded commit")
	}
	return h.event("start-gateway")
}
func (h *gatewayHarness) ObserveGateway(_ context.Context, g Gateway) (GatewayObservation, error) {
	h.observations++
	o := gatewayObservation(h.p, h.w, g)
	if h.mutateObservation != nil {
		h.mutateObservation(&o)
	}
	return o, h.event("observe-gateway")
}
func (h *gatewayHarness) commit(expected string, next Gateway) error {
	if expected != h.saved.Phase || next.Validate(h.p, h.w) != nil {
		h.t.Fatal("invalid commit", expected, next.Phase)
	}
	err := h.event("commit-" + next.Phase)
	if err == nil || h.loseCommit {
		h.saved = next.Clone()
	}
	if h.mutateCommit && next.Observation != nil {
		next.Observation.Mounts[0].Inode = 99
		next.Observation.PublicKeys[0].Auth = "changed"
		next.Observation.Foundation.Contracts[0].Name = "changed"
		next.Observation.Publications[0].HostAddress = "0.0.0.0"
	}
	return err
}
func (h *gatewayHarness) execute(ctx context.Context, g Gateway) (Gateway, error) {
	return ExecuteGateway(ctx, h.p, h.w, g, h, h.commit, func() time.Time { return workspaceNow })
}

func TestGatewayFreshStartupCommitsBeforeEffects(t *testing.T) {
	h, g := newGatewayHarness(t)
	h.mutateCommit = true
	original := g.Clone()
	got, err := h.execute(context.Background(), g)
	if err != nil || got.Validate(h.p, h.w) != nil || got.Phase != "ready" {
		t.Fatal(got.Phase, err)
	}
	if !reflect.DeepEqual(g, original) || !reflect.DeepEqual(got, h.saved) {
		t.Fatal("callback changed retained input/output")
	}
	if h.networkCalls != 1 || h.createCalls != 1 || h.startCalls != 1 || h.observations != 1 {
		t.Fatal("unexpected effects")
	}
	want := []string{"verify-planned", "commit-network-dispatched", "create-network", "commit-network-created", "inspect-network", "verify-network-created", "commit-gateway-dispatched", "create-gateway", "commit-gateway-created", "inspect-gateway", "verify-gateway-created", "commit-start-dispatched", "start-gateway", "commit-started", "observe-gateway", "commit-ready"}
	if !slices.Equal(h.events, want) {
		t.Fatal(h.events)
	}
	got.Observation.PublicKeys[0].Auth = "changed"
	if h.saved.Validate(h.p, h.w) != nil {
		t.Fatal("returned observation aliases durable receipt")
	}
}

func TestGatewayQualifiedDockerImageIDForms(t *testing.T) {
	// The qualified Docker adapters accept these two raw image-ID forms, while
	// still requiring the exact pinned image reference and full image inspection.
	for _, id := range []string{runtimecandidate.ConfigDigest, runtimecandidate.ManifestDigest} {
		h, g := newGatewayHarness(t)
		h.mutateObservation = func(o *GatewayObservation) { o.ImageID = id }
		if got, err := h.execute(context.Background(), g); err != nil || got.Phase != "ready" {
			t.Fatal("qualified image ID refused", id, err)
		}
	}
}

func TestGatewayEveryCommitFailureAndLostReplyPreservesIntent(t *testing.T) {
	for _, phase := range []string{"network-dispatched", "network-created", "gateway-dispatched", "gateway-created", "start-dispatched", "started", "ready"} {
		for _, after := range []bool{false, true} {
			name := phase + "/before"
			if after {
				name = phase + "/lost-reply"
			}
			t.Run(name, func(t *testing.T) {
				h, g := newGatewayHarness(t)
				h.fail, h.loseCommit = "commit-"+phase, after
				_, err := h.execute(context.Background(), g)
				if !errors.Is(err, ErrGatewayUnresolved) || strings.Contains(err.Error(), "SECRET") {
					t.Fatal(err)
				}
				if h.events[len(h.events)-1] != h.fail || h.saved.Validate(h.p, h.w) != nil {
					t.Fatal("continued after uncertain commit")
				}
				if after && h.saved.Phase != phase {
					t.Fatal("durable commit not retained")
				}
				if !after && h.saved.Phase == phase {
					t.Fatal("failed commit advanced")
				}
				calls := len(h.events)
				// Even uncertainty before the first commit requires the store to
				// interrupt the original operation, not call this fresh executor.
				if h.saved.Phase != "planned" {
					if _, err = h.execute(context.Background(), h.saved); err == nil || len(h.events) != calls {
						t.Fatal("replayed effects")
					}
				}
			})
		}
	}
}

func TestGatewayDriverFailuresNeverRepeatOrDiscardKnownIDs(t *testing.T) {
	for name, phase := range map[string]string{
		"verify-planned": "planned", "create-network": "network-created", "inspect-network": "network-created", "verify-network-created": "network-created",
		"create-gateway": "gateway-created", "inspect-gateway": "gateway-created", "verify-gateway-created": "gateway-created", "start-gateway": "start-dispatched", "observe-gateway": "started",
	} {
		t.Run(name, func(t *testing.T) {
			h, g := newGatewayHarness(t)
			h.fail = name
			got, err := h.execute(context.Background(), g)
			if !errors.Is(err, ErrGatewayUnresolved) || strings.Contains(err.Error(), "SECRET") || got.Phase != phase || h.saved.Phase != phase {
				t.Fatal(got.Phase, h.saved.Phase, err)
			}
			if h.networkCalls > 1 || h.createCalls > 1 || h.startCalls > 1 {
				t.Fatal("effect retried")
			}
		})
	}
	for _, resource := range []string{"network", "gateway"} {
		for _, bad := range []bool{false, true} {
			h, g := newGatewayHarness(t)
			h.loseID, h.badID, h.idAt = !bad, bad, resource
			got, err := h.execute(context.Background(), g)
			if err == nil || got.Phase != resource+"-dispatched" || h.startCalls != 0 {
				t.Fatal("missing/malformed resource adopted", resource)
			}
			if resource == "network" && h.createCalls != 0 {
				t.Fatal("continued after lost network identity")
			}
		}
	}
}

func TestGatewayCancellationFencesLaterEffects(t *testing.T) {
	for _, at := range []string{"before", "verify-planned", "commit-network-dispatched", "create-network", "inspect-network", "verify-network-created", "commit-gateway-dispatched", "create-gateway", "inspect-gateway", "verify-gateway-created", "commit-start-dispatched", "start-gateway", "observe-gateway"} {
		t.Run(at, func(t *testing.T) {
			h, g := newGatewayHarness(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h.cancel, h.cancelAt = cancel, at
			if at == "before" {
				cancel()
			}
			got, err := h.execute(ctx, g)
			if !errors.Is(err, context.Canceled) || got.Phase == "ready" || got.Validate(h.p, h.w) != nil {
				t.Fatal(got.Phase, err)
			}
			if at == "before" && len(h.events) != 0 {
				t.Fatal("called driver after cancellation")
			}
			if at == "create-network" && (got.Phase != "network-created" || h.createCalls != 0) {
				t.Fatal("known network lost or continued")
			}
			if at == "create-gateway" && (got.Phase != "gateway-created" || h.startCalls != 0) {
				t.Fatal("known container lost or continued")
			}
			if at == "commit-start-dispatched" && h.startCalls != 0 {
				t.Fatal("started after cancellation")
			}
		})
	}
}

type gatewayObserverOnly struct{ h *gatewayHarness }

func (o gatewayObserverOnly) ObserveGateway(ctx context.Context, g Gateway) (GatewayObservation, error) {
	return o.h.ObserveGateway(ctx, g)
}

func TestGatewayLostStartRecoveredByObservationOnly(t *testing.T) {
	h, g := newGatewayHarness(t)
	h.fail = "start-gateway"
	uncertain, err := h.execute(context.Background(), g)
	if !errors.Is(err, ErrGatewayUnresolved) || uncertain.Phase != "start-dispatched" {
		t.Fatal(err)
	}
	retained := h.saved.Clone()
	h.fail = ""
	got, err := ObserveGatewayReadiness(context.Background(), h.p, h.w, retained, gatewayObserverOnly{h}, func() time.Time { return workspaceNow })
	if err != nil || got.Phase != "ready" || got.Validate(h.p, h.w) != nil {
		t.Fatal(got.Phase, err)
	}
	if !reflect.DeepEqual(h.saved, retained) || h.startCalls != 1 || h.createCalls != 1 || h.networkCalls != 1 {
		t.Fatal("recovery altered original intent or repeated startup")
	}
	for _, phase := range []string{"planned", "network-dispatched", "network-created", "gateway-dispatched", "gateway-created", "ready"} {
		bad := retained.Clone()
		bad.Phase = phase
		count := h.observations
		if _, err = ObserveGatewayReadiness(context.Background(), h.p, h.w, bad, gatewayObserverOnly{h}, func() time.Time { return workspaceNow }); err == nil || h.observations != count {
			t.Fatal("invalid recovery admitted", phase)
		}
	}
}

func TestGatewayReadinessRejectsChangedStateWithoutReinitializing(t *testing.T) {
	cases := map[string]func(*GatewayObservation){
		"different-project":   func(o *GatewayObservation) { o.Selection.Binding.ProjectID += "-other" },
		"different-engine":    func(o *GatewayObservation) { o.Selection.EngineID += "-other" },
		"different-inputs":    func(o *GatewayObservation) { o.Selection.Inputs.ManifestSHA256 = strings.Repeat("f", 64) },
		"different-workspace": func(o *GatewayObservation) { o.Selection.WorkspaceDigest = strings.Repeat("f", 64) },
		"different-operation": func(o *GatewayObservation) { o.Selection.OperationID += "-other" },
		"network":             func(o *GatewayObservation) { o.NetworkID = strings.Repeat("4", 64) },
		"container":           func(o *GatewayObservation) { o.ContainerID = strings.Repeat("4", 64) },
		"image":               func(o *GatewayObservation) { o.ImageID = "sha256:" + strings.Repeat("f", 64) },
		"incomplete-contract": func(o *GatewayObservation) { o.CompleteContract = false },
		"stopped":             func(o *GatewayObservation) { o.Running = false },
		"permissions":         func(o *GatewayObservation) { o.Private = false },
		"unsynced":            func(o *GatewayObservation) { o.Synced = false },
		"rpc-mutation":        func(o *GatewayObservation) { o.RPCReadOnly = false },
		"stale":               func(o *GatewayObservation) { o.ObservedAt = o.ObservedAt.Add(-31 * time.Second) },
		"future":              func(o *GatewayObservation) { o.ObservedAt = o.ObservedAt.Add(6 * time.Second) },
		"no-time":             func(o *GatewayObservation) { o.ObservedAt = time.Time{} },
		"host-netns":          func(o *GatewayObservation) { o.Namespace = o.HostNamespace },
		"boot-changed":        func(o *GatewayObservation) { o.Namespace.BootID = "11234567-89ab-cdef-0123-456789abcdef" },
		"invalid-boot":        func(o *GatewayObservation) { o.Namespace.BootID = ""; o.HostNamespace.BootID = "" },
		"namespace-missing":   func(o *GatewayObservation) { o.Namespace.Inode = 0 },
		"root-replaced":       func(o *GatewayObservation) { o.Root.Inode++ },
		"chain-missing":       func(o *GatewayObservation) { o.Chain = pierstore.DirectoryIdentity{} },
		"chain-alias":         func(o *GatewayObservation) { o.Chain = o.Root },
		"secret-alias":        func(o *GatewayObservation) { o.Secrets = o.Chain },
		"cross-device":        func(o *GatewayObservation) { o.Chain.Device++ },
		"mount-replaced":      func(o *GatewayObservation) { o.Mounts[0].Birth = "1000000001:1" },
		"mount-missing":       func(o *GatewayObservation) { o.Mounts = o.Mounts[:1] },
		"foundation":          func(o *GatewayObservation) { o.Foundation.ChainID++ },
		"contract-code":       func(o *GatewayObservation) { o.Foundation.Contracts[0].CodeSHA256 = strings.Repeat("f", 64) },
		"head":                func(o *GatewayObservation) { o.Head++ },
		"no-head-hash":        func(o *GatewayObservation) { o.HeadHash = "" },
		"missing-key":         func(o *GatewayObservation) { o.PublicKeys = o.PublicKeys[:1] },
		"wrong-ship":          func(o *GatewayObservation) { o.PublicKeys[0].Ship = "~bud" },
		"key-life":            func(o *GatewayObservation) { o.PublicKeys[0].Life++ },
		"key-suite":           func(o *GatewayObservation) { o.PublicKeys[0].Suite++ },
		"key-alias":           func(o *GatewayObservation) { o.PublicKeys[1].Auth = o.PublicKeys[0].Crypt },
		"invalid-key":         func(o *GatewayObservation) { o.PublicKeys[0].Auth = "0x" + strings.Repeat("0", 64) },
		"host-binding":        func(o *GatewayObservation) { o.Publications[0].HostAddress = "0.0.0.0" },
		"host-port":           func(o *GatewayObservation) { o.Publications[0].HostPort++ },
		"internal-port":       func(o *GatewayObservation) { o.Publications[0].InternalPort = 25432 },
		"udp":                 func(o *GatewayObservation) { o.Publications[0].Protocol = "udp" },
		"unpublished":         func(o *GatewayObservation) { o.Publications = nil },
		"extra-publication": func(o *GatewayObservation) {
			o.Publications = append(o.Publications, projectnetwork.Publication{InternalPort: 25432, HostAddress: "127.0.0.1", HostPort: 25432, Protocol: "tcp"})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			h, g := newGatewayHarness(t)
			h.mutateObservation = mutate
			got, err := h.execute(context.Background(), g)
			if !errors.Is(err, ErrGatewayObservation) || got.Phase != "started" || h.saved.Phase != "started" {
				t.Fatal(got.Phase, err)
			}
			if _, err = ObserveGatewayReadiness(context.Background(), h.p, h.w, h.saved, gatewayObserverOnly{h}, func() time.Time { return workspaceNow }); !errors.Is(err, ErrGatewayObservation) {
				t.Fatal(err)
			}
			if h.startCalls != 1 || h.createCalls != 1 || h.networkCalls != 1 {
				t.Fatal("recovery repeated effects")
			}
		})
	}
}

func TestGatewaySelectionCannotAuthorizeFromInspectionAlone(t *testing.T) {
	for name, mutate := range map[string]func(*Gateway){
		"version":          func(g *Gateway) { g.Selection.Version++ },
		"engine":           func(g *Gateway) { g.Selection.EngineID = "/var/run/docker.sock" },
		"image":            func(g *Gateway) { g.Selection.Image = "latest" },
		"operation":        func(g *Gateway) { g.Selection.OperationID = g.Selection.WorkspaceOperationID },
		"revision":         func(g *Gateway) { g.Selection.Revision++ },
		"generation":       func(g *Gateway) { g.Selection.Binding.Generation++ },
		"workspace":        func(g *Gateway) { g.Selection.WorkspaceDigest = strings.Repeat("f", 64) },
		"input-format":     func(g *Gateway) { g.Selection.Inputs.Format = "unknown" },
		"input-hash":       func(g *Gateway) { g.Selection.Inputs.ManifestSHA256 = "" },
		"input-foundation": func(g *Gateway) { g.Selection.Inputs.FoundationDigest = strings.Repeat("f", 64) },
		"input-authority":  func(g *Gateway) { g.Selection.Inputs.RuntimeAuthorized = true },
		"input-files":      func(g *Gateway) { g.Selection.Inputs.Files = MaxRuntimeInputFiles + 1 },
		"input-bytes":      func(g *Gateway) { g.Selection.Inputs.Bytes = MaxRuntimeInputBytes + 1 },
		"adopt-network":    func(g *Gateway) { g.NetworkID = strings.Repeat("2", 64) },
		"adopt-container":  func(g *Gateway) { g.ContainerID = strings.Repeat("3", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			h, g := newGatewayHarness(t)
			mutate(&g)
			if _, err := h.execute(context.Background(), g); err == nil || len(h.events) != 0 {
				t.Fatal("invalid selection reached driver")
			}
		})
	}
	h, g := newGatewayHarness(t)
	h.w.Phase = "layout-created"
	h.w.Observation = nil
	if _, err := NewGateway(h.p, h.w, "new-gateway", g.Selection.EngineID, g.Selection.Inputs); err == nil {
		t.Fatal("unobserved workspace accepted")
	}
}
