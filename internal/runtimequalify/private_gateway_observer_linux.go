//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"golang.org/x/sys/unix"
)

// Source-qualified M392 initializer. Runtime delivery still requires the new
// sealed generation and disposable-VM gate; this is not a production release pin.
const privateGatewayInitializerSHA256 = "b3f19ed5bdf468d41349e72c50514beaa140d7a00a694580a6570d90f68040fe"

type privateGatewayObserverInputs interface {
	privateGatewayInputHandle
}
type privateGatewayStateHandle interface {
	Verify(context.Context) (privatechain.GatewayStateObservation, error)
	io.Closer
}
type privateGatewayRunning struct {
	container  container
	allocation privateGatewayAllocation
}

func (r *privateGatewayResources) inspectRunning(ctx context.Context) (privateGatewayRunning, error) {
	if !privateGatewayID(r.networkID) || !privateGatewayID(r.containerID) {
		return privateGatewayRunning{}, errPrivateGatewayResource
	}
	before, err := r.network(ctx, true)
	if err != nil {
		return privateGatewayRunning{}, err
	}
	raw, err := r.q.docker(ctx, r.q.d.times.command, "container", "inspect", r.containerID)
	if err != nil {
		return privateGatewayRunning{}, err
	}
	c, err := r.contract.inspectContainer([]byte(raw), r.networkID, r.containerID, before, true)
	if err != nil {
		return privateGatewayRunning{}, err
	}
	after, err := r.network(ctx, true)
	if err != nil || before != after {
		return privateGatewayRunning{}, errPrivateGatewayResource
	}
	return privateGatewayRunning{container: c, allocation: after}, nil
}

type privateGatewayObserver struct {
	mu                sync.Mutex
	runtime           *privateGatewayResources
	inputs            privateGatewayObserverInputs
	state             privateGatewayStateHandle
	readBootID        func() (string, error)
	readHostNamespace func() (projectnetwork.NamespaceIdentity, error)
	now               func() time.Time
	closed            bool
	memberGateway     *privatechain.Gateway
	members           []privateGatewayMember
}

func gatewayHostNamespace(readBoot func() (string, error)) (projectnetwork.NamespaceIdentity, error) {
	before, err := readBoot()
	if err != nil || !projectBootIDPattern.MatchString(before) {
		return projectnetwork.NamespaceIdentity{}, errPrivateGatewayResource
	}
	var st unix.Stat_t
	if unix.Stat("/proc/self/ns/net", &st) != nil || st.Dev == 0 || st.Ino == 0 {
		return projectnetwork.NamespaceIdentity{}, errPrivateGatewayResource
	}
	after, err := readBoot()
	if err != nil || before != after {
		return projectnetwork.NamespaceIdentity{}, errPrivateGatewayResource
	}
	return projectnetwork.NamespaceIdentity{BootID: before, Device: uint64(st.Dev), Inode: st.Ino}, nil
}

// Open only observes a selected initialized workspace. It does not wait for an
// initializer, create client metadata, start a container or repair partial state.
// A first-start driver must wait read-only for readiness within its own deadline;
// recovery remains one attributed observation without replaying initialization.
func openPrivateGatewayObserver(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (*privateGatewayObserver, error) {
	return openPrivateGatewayObserverMode(ctx, q, stateRoot, inputRoot, p, w, g, nil)
}

func openPrivateGatewayObserverMode(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway, membership *privateGatewayMembership) (*privateGatewayObserver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if q == nil || g.Validate(p, w) != nil || (membership == nil && g.Phase != "start-dispatched" && g.Phase != "started") || (membership != nil && g.Phase != "ready") {
		return nil, errPrivateGatewayResource
	}
	c, err := newPrivateGatewayContainerContract(p, w, g, stateRoot, inputRoot, q.r.ImageID)
	if err != nil {
		return nil, err
	}
	i, err := privatechain.OpenRuntimeInputVerifier(ctx, inputRoot, g.Selection.Inputs.ManifestSHA256, g.Selection.Inputs.FoundationDigest)
	if err != nil {
		return nil, err
	}
	p.Members = slices.Clone(p.Members)
	p.Foundation.Contracts = slices.Clone(p.Foundation.Contracts)
	r := &privateGatewayResources{q: q, plan: p, workspace: w.Clone(), contract: c, inputs: i, networkID: g.NetworkID, containerID: g.ContainerID}
	var members []privateGatewayMember
	if membership != nil {
		members, err = r.bindGatewayMemberPiers(membership.members, membership.piers)
		if err != nil {
			i.Close()
			return nil, err
		}
	}
	err = r.inputsCurrent(ctx)
	if err == nil {
		err = r.engine(ctx)
	}
	if err == nil {
		err = q.inspectImage(ctx)
	}
	if err == nil && q.r.ImageID != c.imageID {
		err = errPrivateGatewayResource
	}
	if err == nil {
		_, err = r.inspectRunning(ctx)
	}
	if err != nil {
		i.Close()
		return nil, err
	}
	var s *privatechain.GatewayStateVerifier
	if membership == nil {
		s, err = privatechain.OpenGatewayStateVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w, g)
	} else {
		s, err = privatechain.OpenMemberGatewayStateVerifier(ctx, filepath.Join(stateRoot, "private-projects"), p, w, g, slices.Clone(membership.piers))
	}
	if err != nil {
		i.Close()
		return nil, err
	}
	boot := func() (string, error) {
		raw, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
		return strings.TrimSpace(string(raw)), err
	}
	o := &privateGatewayObserver{runtime: r, inputs: i, state: s, readBootID: boot, readHostNamespace: func() (projectnetwork.NamespaceIdentity, error) { return gatewayHostNamespace(boot) }, now: time.Now, members: members}
	if membership != nil {
		copy := g.Clone()
		o.memberGateway = &copy
	}
	return o, nil
}
func (o *privateGatewayObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	o.closed = true
	return errors.Join(o.inputs.Close(), o.state.Close())
}
func (o *privateGatewayObserver) namespace(ctx context.Context) (projectnetwork.NamespaceIdentity, error) {
	r := o.runtime
	// Share the existing independently bracketed exact-container namespace read.
	e := &projectAnchorEngine{base: &projectDockerEngine{q: r.q}, readBootID: o.readBootID}
	return e.namespaceFor(ctx, r.containerID)
}
func (o *privateGatewayObserver) ObserveGateway(ctx context.Context, g privatechain.Gateway) (privatechain.GatewayObservation, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	bad := func() (privatechain.GatewayObservation, error) {
		return privatechain.GatewayObservation{}, privatechain.ErrGatewayObservation
	}
	if err := ctx.Err(); err != nil {
		return privatechain.GatewayObservation{}, err
	}
	r := o.runtime
	if o.closed || r == nil || o.inputs == nil || o.state == nil || o.now == nil || o.readBootID == nil || o.readHostNamespace == nil || g.Validate(r.plan, r.workspace) != nil || g.Selection != r.contract.network.selection || g.NetworkID != r.networkID || g.ContainerID != r.containerID || !o.selectedGatewayPhase(g) {
		return bad()
	}
	// Hash the sealed input generation before starting the mutable-observation
	// freshness window. Cold verification may exceed that window by itself.
	// The same held inputs are fully reverified inside the window after RPC and
	// filesystem observation, so changed inputs still invalidate the snapshot.
	preparation, finishPreparation := context.WithTimeout(ctx, 2*time.Minute)
	preparationErr := r.inputsCurrent(preparation)
	preparationExpired := preparation.Err() != nil
	finishPreparation()
	if preparationErr != nil || preparationExpired {
		return bad()
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	at := o.now().UTC()
	if at.IsZero() || r.engine(ctx) != nil {
		return bad()
	}
	if r.q.inspectImage(ctx) != nil || r.q.r.ImageID != r.contract.imageID {
		return bad()
	}
	first, err := r.inspectRunning(ctx)
	if err != nil {
		return bad()
	}
	host, err := o.readHostNamespace()
	if err != nil {
		return bad()
	}
	ns, err := o.namespace(ctx)
	if err != nil || ns.BootID != host.BootID || (ns.Device == host.Device && ns.Inode == host.Inode) {
		return bad()
	}
	state, err := o.state.Verify(ctx)
	if err != nil || !state.Private {
		return bad()
	}
	membersBefore, err := o.observeMembers(ctx, ns)
	if err != nil {
		return bad()
	}
	chain, err := r.q.privateGatewayRPC(ctx, r.plan, r.containerID)
	if err != nil {
		return bad()
	}
	if chain.Namespace.Device != strconv.FormatUint(ns.Device, 10) || chain.Namespace.Inode != strconv.FormatUint(ns.Inode, 10) {
		return bad()
	}
	// Rehash the complete held input tree after the RPC, concurrently with the
	// independent final runtime/storage observations. Both must finish within
	// this same freshness window. Always cancel and join before returning, so
	// failure or Close can never leave a verifier using released handles.
	inputsDone := make(chan struct{})
	var inputsErr error
	go func() {
		defer close(inputsDone)
		inputsErr = r.inputsCurrent(ctx)
	}()
	defer func() { cancel(); <-inputsDone }()
	afterState, err := o.state.Verify(ctx)
	if err != nil || !reflect.DeepEqual(state, afterState) {
		return bad()
	}
	membersAfter, err := o.observeMembers(ctx, ns)
	if err != nil || !reflect.DeepEqual(membersBefore, membersAfter) {
		return bad()
	}
	afterNS, err := o.namespace(ctx)
	if err != nil || afterNS != ns {
		return bad()
	}
	afterHost, err := o.readHostNamespace()
	if err != nil || afterHost != host {
		return bad()
	}
	<-inputsDone
	if inputsErr != nil || r.engine(ctx) != nil {
		return bad()
	}
	if r.q.inspectImage(ctx) != nil || r.q.r.ImageID != r.contract.imageID {
		return bad()
	}
	last, err := r.inspectRunning(ctx)
	if err != nil || !reflect.DeepEqual(first, last) {
		return bad()
	}
	elapsed := o.now().Sub(at)
	if ctx.Err() != nil || elapsed < 0 || elapsed > 25*time.Second {
		return bad()
	}
	result := privatechain.GatewayObservation{Selection: g.Selection, ObservedAt: at, NetworkID: g.NetworkID, ContainerID: g.ContainerID, ImageID: r.contract.imageID, Namespace: ns, HostNamespace: host, Root: state.Root, Chain: state.Chain, Secrets: state.Secrets, Mounts: slices.Clone(state.Mounts), Foundation: chain.Foundation, Head: chain.Head, HeadHash: chain.HeadHash, PublicKeys: slices.Clone(chain.PublicKeys), CompleteContract: true, Running: true, Private: true, Synced: true, RPCReadOnly: true}
	for _, m := range r.plan.Members {
		result.Publications = append(result.Publications, projectnetwork.Publication{InternalPort: m.InternalHTTPPort, HostAddress: r.plan.BindAddress, HostPort: m.HTTPPort, Protocol: "tcp"})
	}
	ready := g.Clone()
	ready.Phase = "ready"
	ready.Observation = &result
	if ready.Validate(r.plan, r.workspace) != nil {
		return bad()
	}
	return result, nil
}
