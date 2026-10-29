//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/bootpill"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

var errPrivateMemberLaunch = errors.New("private member launch is unresolved; preserve its journal and all retained objects")

type privateMemberHeld interface {
	Verify(context.Context) error
	io.Closer
}
type privateMemberEmpty interface {
	VerifyEmpty(context.Context) error
	io.Closer
}
type privateMemberBirthHandle interface {
	Arm(context.Context, privatechain.MemberLaunch) error
	Capture(context.Context, privatechain.MemberLaunch, privatechain.MemberBirthObserver) (privatechain.MemberBirthEvidence, error)
	Verify(context.Context) error
	io.Closer
}

// Hooks are private to package tests; the concrete constructor binds all of them
// to held filesystem checks and the exact selected rootless engine. No public
// route, startup resumption, dependency transfer or second-member admission.
type privateMemberLaunchOps struct {
	prepare  func(context.Context) (pierstore.DirectoryIdentity, error)
	pill     func(context.Context, pierstore.DirectoryIdentity) (privateMemberHeld, error)
	key      func(context.Context, string) (privateMemberHeld, error)
	proof    func(context.Context) (privateMemberKeyProofObservation, error)
	gateway  func(context.Context, []privateGatewayMember, []privatechain.GatewayMemberPier) (privatechain.GatewayObservation, error)
	create   func(context.Context) (string, error)
	start    func(context.Context, string) error
	process  func(context.Context, string, bool) (privatechain.MemberBirthProcess, error)
	waitPier func(context.Context) error
}

type privateMemberLaunchDriver struct {
	mu            sync.Mutex
	contract      privateMemberContract
	initial       privatechain.MemberLaunch
	ops           privateMemberLaunchOps
	mount         privateMemberEmpty
	birth         privateMemberBirthHandle
	pill, key     privateMemberHeld
	artifact      pierstore.DirectoryIdentity
	acceptedKey   *privatechain.MemberKeyAcceptance
	created       string
	birthEvidence *privatechain.MemberBirthEvidence
	now           func() time.Time

	artifactAttempted  bool
	keyAttempted       bool
	createAttempted    bool
	createAcknowledged bool
	startAttempted     bool
	startAcknowledged  bool
	captureAttempted   bool
	observeAttempted   bool
	closed             bool
}

// Read-only construction. The scheduler must hold its common work/update fence
// throughout construction, ExecuteMemberLaunch, durable capture and Close. This
// first implementation refuses peers until their durable readiness is available.
func openPrivateMemberLaunch(ctx context.Context, q *qualifier, stateRoot, inputRoot string, p privatechain.Plan, w privatechain.Workspace, s privatechain.MemberLaunch) (*privateMemberLaunchDriver, error) {
	if q == nil || s.Validate(p, w) != nil || (s.Version != 2 && s.Version != 3) || s.Phase != "planned" || len(s.Peers) != 0 {
		return nil, errPrivateMemberLaunch
	}
	c, err := newPrivateMemberContract(privateMemberSelection{Plan: p, Workspace: w, Gateway: s.Gateway, InstanceID: s.InstanceID, RuntimeID: s.RuntimeID, OperationID: s.OperationID, ContainerGeneration: 1, Mode: "boot"}, stateRoot, q.r.ImageID)
	if err != nil || c.digest != s.ContractDigest {
		return nil, errPrivateMemberLaunch
	}
	if _, err = os.Lstat(filepath.Dir(c.pillPath)); !errors.Is(err, os.ErrNotExist) {
		return nil, errPrivateMemberLaunch
	}
	if err = bootpill.Verify(ctx, bootpill.CachePath(stateRoot, s.Pill), s.Pill); err != nil {
		return nil, errPrivateMemberLaunch
	}
	dir := filepath.Join(stateRoot, "private-projects")
	mount, err := privatechain.OpenMemberMountVerifier(ctx, dir, p, w, s.InstanceID)
	if err != nil {
		return nil, errPrivateMemberLaunch
	}
	birth, err := privatechain.OpenMemberBirthCapture(ctx, dir, p, w, s)
	if err != nil {
		mount.Close()
		return nil, errPrivateMemberLaunch
	}
	d := &privateMemberLaunchDriver{contract: c, initial: s.Clone(), mount: mount, birth: birth, now: time.Now}
	d.ops = privateMemberLaunchOps{
		prepare: func(ctx context.Context) (pierstore.DirectoryIdentity, error) {
			return preparePrivateMemberArtifact(ctx, c, d.initial.Pill)
		},
		pill: func(ctx context.Context, id pierstore.DirectoryIdentity) (privateMemberHeld, error) {
			return openPrivateMemberPill(ctx, c, id)
		},
		key: func(ctx context.Context, hash string) (privateMemberHeld, error) {
			return privatechain.OpenMemberKeyVerifier(ctx, dir, c.selection.Plan, c.selection.Workspace, c.selection.Gateway, c.selection.InstanceID, hash)
		},
		proof: func(ctx context.Context) (privateMemberKeyProofObservation, error) {
			return q.privateMemberKeyProof(ctx, c)
		},
		gateway: func(ctx context.Context, m []privateGatewayMember, piers []privatechain.GatewayMemberPier) (privatechain.GatewayObservation, error) {
			o, err := openPrivateMemberGatewayObserver(ctx, q, stateRoot, inputRoot, c.selection.Plan, c.selection.Workspace, c.selection.Gateway, m, piers)
			if err != nil {
				return privatechain.GatewayObservation{}, err
			}
			defer o.Close()
			got, err := o.ObserveGateway(ctx, c.selection.Gateway)
			if err == nil {
				err = checkPrivateMemberArtifactMounts(ctx, o.runtime.mountInventory, c, m)
			}
			return got, err
		},
		create: func(ctx context.Context) (string, error) {
			return q.dockerCreateID(ctx, q.d.times.command, c.createArgs()...)
		},
		start: func(ctx context.Context, id string) error {
			out, err := q.docker(ctx, q.d.times.command, "container", "start", id)
			if err != nil || strings.TrimSpace(out) != id {
				return errPrivateMemberLaunch
			}
			return nil
		},
		process: func(ctx context.Context, id string, running bool) (privatechain.MemberBirthProcess, error) {
			return observePrivateMemberProcess(ctx, q, c, id, running, time.Now)
		},
		waitPier: func(ctx context.Context) error {
			return waitPrivateMemberPier(ctx, c, func(ctx context.Context) error {
				if err := mount.Verify(ctx); err != nil {
					return err
				}
				_, err := observePrivateMemberProcess(ctx, q, c, d.created, true, time.Now)
				return err
			})
		},
	}
	if err = d.VerifyPreparation(ctx, s); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func (d *privateMemberLaunchDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	var errs []error
	for _, v := range []io.Closer{d.key, d.pill, d.birth, d.mount} {
		if v != nil {
			errs = append(errs, v.Close())
		}
	}
	return errors.Join(errs...)
}

func (d *privateMemberLaunchDriver) selected(ctx context.Context, s privatechain.MemberLaunch, phases ...string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if d.closed || s.Validate(d.contract.selection.Plan, d.contract.selection.Workspace) != nil {
		return errPrivateMemberLaunch
	}
	allowed := false
	for _, p := range phases {
		if s.Phase == p {
			allowed = true
		}
	}
	if !allowed || s.Artifact != d.artifact || s.ContainerID != d.created || !reflect.DeepEqual(s.Key, d.acceptedKey) || s.Running != nil {
		return errPrivateMemberLaunch
	}
	if s.Capture != nil && !reflect.DeepEqual(s.Capture, d.birthEvidence) {
		return errPrivateMemberLaunch
	}
	base := s.Clone()
	base.Capture = nil
	base.Phase = "planned"
	base.Artifact = pierstore.DirectoryIdentity{}
	base.ContainerID = ""
	base.Key = nil
	if !reflect.DeepEqual(base, d.initial) {
		return errPrivateMemberLaunch
	}
	return nil
}

func (d *privateMemberLaunchDriver) gateway(ctx context.Context, running *privatechain.MemberBirthProcess, pier *pierstore.Identity) (privatechain.GatewayObservation, error) {
	var members []privateGatewayMember
	var piers []privatechain.GatewayMemberPier
	if d.created != "" {
		m := privateGatewayMember{Contract: d.contract, ContainerID: d.created, State: "created"}
		if running != nil {
			m.State = "running"
			m.StartedAt = running.StartedAt
		}
		members = append(members, m)
	}
	if pier != nil {
		piers = append(piers, privatechain.GatewayMemberPier{InstanceID: d.initial.InstanceID, Identity: *pier})
	}
	o, err := d.ops.gateway(ctx, members, piers)
	if err != nil {
		return privatechain.GatewayObservation{}, errPrivateMemberLaunch
	}
	want := *d.initial.Gateway.Observation
	got := o
	got.ObservedAt = want.ObservedAt
	now := d.now().UTC()
	if !reflect.DeepEqual(want, got) || o.ObservedAt.IsZero() || now.Sub(o.ObservedAt) > 30*time.Second || o.ObservedAt.Sub(now) > 5*time.Second || ctx.Err() != nil {
		return privatechain.GatewayObservation{}, errPrivateMemberLaunch
	}
	return o, nil
}

func (d *privateMemberLaunchDriver) preparation(ctx context.Context) error {
	if d.startAttempted || d.mount.VerifyEmpty(ctx) != nil {
		return errPrivateMemberLaunch
	}
	if d.artifact != (pierstore.DirectoryIdentity{}) && (d.pill == nil || d.pill.Verify(ctx) != nil) {
		return errPrivateMemberLaunch
	}
	if d.acceptedKey != nil && (d.key == nil || d.key.Verify(ctx) != nil) {
		return errPrivateMemberLaunch
	}
	if _, err := d.gateway(ctx, nil, nil); err != nil {
		return err
	}
	if d.mount.VerifyEmpty(ctx) != nil || ctx.Err() != nil {
		return errPrivateMemberLaunch
	}
	return nil
}
func (d *privateMemberLaunchDriver) VerifyPreparation(ctx context.Context, s privatechain.MemberLaunch) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.selected(ctx, s, "planned", "artifact-created", "key-verified", "created"); err != nil {
		return err
	}
	return d.preparation(ctx)
}
func (d *privateMemberLaunchDriver) PrepareArtifact(ctx context.Context, s privatechain.MemberLaunch) (pierstore.DirectoryIdentity, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected(ctx, s, "artifact-dispatched") != nil || d.artifactAttempted {
		return pierstore.DirectoryIdentity{}, errPrivateMemberLaunch
	}
	d.artifactAttempted = true
	if err := d.preparation(ctx); err != nil {
		return pierstore.DirectoryIdentity{}, err
	}
	id, err := d.ops.prepare(ctx)
	d.artifact = id
	if err != nil || ctx.Err() != nil || id == (pierstore.DirectoryIdentity{}) {
		return id, errPrivateMemberLaunch
	}
	d.pill, err = d.ops.pill(ctx, id)
	if err != nil || d.pill == nil {
		return id, errPrivateMemberLaunch
	}
	return id, nil
}

// Re-prove the key before each container mutation. The accepted hash must stay
// identical and the whole current gateway observation follows the potentially
// slow cryptographic proof. Held files are checked again after that observation.
func (d *privateMemberLaunchDriver) proveKey(ctx context.Context) (privatechain.MemberKeyAcceptance, error) {
	bad := func() (privatechain.MemberKeyAcceptance, error) {
		return privatechain.MemberKeyAcceptance{}, errPrivateMemberLaunch
	}
	if d.preparation(ctx) != nil {
		return bad()
	}
	o, err := d.ops.proof(ctx)
	if err != nil || o.BindingSHA256 != d.contract.digest || !o.Verified || !privateGatewayID(o.KeySHA256) {
		return bad()
	}
	if d.acceptedKey != nil && d.acceptedKey.KeySHA256 != o.KeySHA256 {
		return bad()
	}
	if d.key == nil {
		d.key, err = d.ops.key(ctx, o.KeySHA256)
		if err != nil || d.key == nil {
			return bad()
		}
	}
	if d.key.Verify(ctx) != nil || d.pill == nil || d.pill.Verify(ctx) != nil {
		return bad()
	}
	g, err := d.gateway(ctx, nil, nil)
	if err != nil || d.key.Verify(ctx) != nil || d.pill.Verify(ctx) != nil || d.mount.VerifyEmpty(ctx) != nil || ctx.Err() != nil {
		return bad()
	}
	return privatechain.MemberKeyAcceptance{CheckedAt: d.now().UTC(), ContractDigest: d.contract.digest, KeySHA256: o.KeySHA256, Gateway: g, Verified: true}, nil
}
func (d *privateMemberLaunchDriver) VerifyKey(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberKeyAcceptance, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected(ctx, s, "artifact-created") != nil || d.keyAttempted {
		return privatechain.MemberKeyAcceptance{}, errPrivateMemberLaunch
	}
	d.keyAttempted = true
	k, err := d.proveKey(ctx)
	if err != nil {
		return privatechain.MemberKeyAcceptance{}, err
	}
	copy := k
	copy.Gateway = k.Gateway.Clone()
	d.acceptedKey = &copy
	return k, nil
}
func (d *privateMemberLaunchDriver) CreateMember(ctx context.Context, s privatechain.MemberLaunch) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected(ctx, s, "create-dispatched") != nil || d.createAttempted {
		return "", errPrivateMemberLaunch
	}
	d.createAttempted = true
	if _, err := d.proveKey(ctx); err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	id, err := d.ops.create(ctx)
	id = strings.TrimSpace(id)
	if !privateGatewayID(id) || id == s.Gateway.ContainerID || id == s.Gateway.NetworkID {
		return "", errPrivateMemberLaunch
	}
	d.created = id
	if err != nil || ctx.Err() != nil {
		return id, errPrivateMemberLaunch
	}
	d.createAcknowledged = true
	return id, nil
}
func (d *privateMemberLaunchDriver) InspectCreated(ctx context.Context, s privatechain.MemberLaunch) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected(ctx, s, "created") != nil || !d.createAcknowledged {
		return errPrivateMemberLaunch
	}
	if _, err := d.ops.process(ctx, d.created, false); err != nil {
		return errPrivateMemberLaunch
	}
	return d.preparation(ctx)
}
func (d *privateMemberLaunchDriver) StartMember(ctx context.Context, s privatechain.MemberLaunch) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected(ctx, s, "start-dispatched") != nil || !d.createAcknowledged || d.startAttempted {
		return errPrivateMemberLaunch
	}
	// Mark after preflight, but consume every attempted Start call even if it fails.
	defer func() { d.startAttempted = true }()
	if _, err := d.proveKey(ctx); err != nil {
		return err
	}
	if d.birth.Arm(ctx, s.Clone()) != nil || ctx.Err() != nil {
		return errPrivateMemberLaunch
	}
	if d.ops.start(ctx, d.created) != nil || ctx.Err() != nil {
		return errPrivateMemberLaunch
	}
	d.startAcknowledged = true
	return nil
}

// CaptureBirth uses the handle held continuously since the empty pre-Start
// mount. The controller must durably save this result before ObserveRunning.
func (d *privateMemberLaunchDriver) CaptureBirth(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberBirthEvidence, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s.Version != 3 || d.selected(ctx, s, "started") != nil || !d.startAcknowledged || d.observeAttempted {
		return privatechain.MemberBirthEvidence{}, errPrivateMemberLaunch
	}
	return d.captureBirth(ctx, s)
}

// Called only with the driver mutex held. Even a failed wait/capture consumes
// this attempt; recovery must read the durable journal, never repeat first boot.
func (d *privateMemberLaunchDriver) captureBirth(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberBirthEvidence, error) {
	if d.captureAttempted {
		return privatechain.MemberBirthEvidence{}, errPrivateMemberLaunch
	}
	d.captureAttempted = true
	if d.ops.waitPier(ctx) != nil || ctx.Err() != nil {
		return privatechain.MemberBirthEvidence{}, errPrivateMemberLaunch
	}
	e, err := d.birth.Capture(ctx, s.Clone(), func(ctx context.Context, _ privatechain.MemberLaunch) (privatechain.MemberBirthProcess, error) {
		return d.ops.process(ctx, d.created, true)
	})
	if err != nil || ctx.Err() != nil || e.Validate(d.contract.selection.Plan, d.contract.selection.Workspace, s) != nil {
		return privatechain.MemberBirthEvidence{}, errPrivateMemberLaunch
	}
	saved := e
	d.birthEvidence = &saved
	return e, nil
}

func (d *privateMemberLaunchDriver) ObserveRunning(ctx context.Context, s privatechain.MemberLaunch) (privatechain.MemberRunningObservation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	bad := func() (privatechain.MemberRunningObservation, error) {
		return privatechain.MemberRunningObservation{}, errPrivateMemberLaunch
	}
	phase := "started"
	if s.Version == 3 {
		phase = "captured"
	}
	if d.selected(ctx, s, phase) != nil || !d.startAcknowledged || d.observeAttempted {
		return bad()
	}
	d.observeAttempted = true
	var e privatechain.MemberBirthEvidence
	if s.Version == 3 {
		if !d.captureAttempted || s.Capture == nil || d.birthEvidence == nil {
			return bad()
		}
		e = *d.birthEvidence
	} else {
		var err error
		e, err = d.captureBirth(ctx, s)
		if err != nil {
			return bad()
		}
	}
	g, err := d.gateway(ctx, &e.After, &e.Pier)
	if err != nil || d.birth.Verify(ctx) != nil || d.key.Verify(ctx) != nil || d.pill.Verify(ctx) != nil {
		return bad()
	}
	last, err := d.ops.process(ctx, d.created, true)
	want := e.After
	lastAt := last.ObservedAt
	last.ObservedAt = want.ObservedAt
	if err != nil || last != want || lastAt.Before(g.ObservedAt) || d.now().Sub(lastAt) > 30*time.Second || lastAt.Sub(d.now()) > 5*time.Second || ctx.Err() != nil {
		return bad()
	}
	return privatechain.MemberRunningObservation{ObservedAt: d.now().UTC(), ContainerID: d.created, ContractDigest: d.contract.digest, StartedAt: last.StartedAt, Namespace: last.Namespace, Gateway: g, CompleteContract: true, Running: true, Birth: &e}, nil
}

// Bracket one namespace observation with full container snapshots. Created
// containers are inspected only, never exec'd. This grants no pier authority.
func observePrivateMemberProcess(ctx context.Context, q *qualifier, c privateMemberContract, id string, running bool, now func() time.Time) (privatechain.MemberBirthProcess, error) {
	bad := func() (privatechain.MemberBirthProcess, error) {
		return privatechain.MemberBirthProcess{}, errPrivateMemberLaunch
	}
	if q == nil || now == nil || ctx.Err() != nil {
		return bad()
	}
	inspect := func() (container, error) {
		raw, err := q.docker(ctx, q.d.times.command, "container", "inspect", id)
		if err != nil {
			return container{}, err
		}
		return c.inspectContainer([]byte(raw), id, running)
	}
	before, err := inspect()
	if err != nil {
		return bad()
	}
	var ns projectnetwork.NamespaceIdentity
	if running {
		e := &projectAnchorEngine{base: &projectDockerEngine{q: q}, readBootID: func() (string, error) {
			raw, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
			return strings.TrimSpace(string(raw)), err
		}}
		ns, err = e.namespaceFor(ctx, id)
		if err != nil || ns != c.selection.Gateway.Observation.Namespace {
			return bad()
		}
	}
	after, err := inspect()
	if err != nil || !reflect.DeepEqual(before, after) || ctx.Err() != nil {
		return bad()
	}
	if !running {
		return privatechain.MemberBirthProcess{}, nil
	}
	started, err := time.Parse(time.RFC3339Nano, before.State.StartedAt)
	if err != nil {
		return bad()
	}
	return privatechain.MemberBirthProcess{ObservedAt: now().UTC(), StartedAt: started, ContainerID: id, ContractDigest: c.digest, Namespace: ns, CompleteContract: true, Running: true}, nil
}

var _ privatechain.CapturedMemberLaunchDriver = (*privateMemberLaunchDriver)(nil)

// A bounded read-only wait before the single capture attempt. Existence is only
// a scheduling hint; the held birth verifier and exact process observations are
// still mandatory. No payload, lock, peer, key or process control is accessed.
func waitPrivateMemberPier(ctx context.Context, c privateMemberContract, verify func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	ship := ""
	for _, m := range c.selection.Plan.Members {
		if m.InstanceID == c.selection.InstanceID {
			ship = strings.TrimPrefix(m.Ship, "~")
		}
	}
	if ship == "" || verify == nil {
		return errPrivateMemberLaunch
	}
	for {
		if ctx.Err() != nil || verify(ctx) != nil {
			return errPrivateMemberLaunch
		}
		info, err := os.Lstat(filepath.Join(c.mountSource, ship, ".urb"))
		if err == nil {
			if !info.IsDir() {
				return errPrivateMemberLaunch
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return errPrivateMemberLaunch
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errPrivateMemberLaunch
		case <-timer.C:
		}
	}
}
