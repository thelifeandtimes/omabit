//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
)

type privateLaunchHeldStub struct {
	h      *privateLaunchHarness
	name   string
	closes int
}

func (v *privateLaunchHeldStub) Verify(ctx context.Context) error      { return v.h.event(v.name) }
func (v *privateLaunchHeldStub) VerifyEmpty(ctx context.Context) error { return v.h.event(v.name) }
func (v *privateLaunchHeldStub) Close() error                          { v.closes++; return nil }

type privateLaunchBirthStub struct {
	privateLaunchHeldStub
	armed *privatechain.MemberLaunch
	at    time.Time
}

func (v *privateLaunchBirthStub) Arm(ctx context.Context, s privatechain.MemberLaunch) error {
	if err := v.h.event("arm"); err != nil {
		return err
	}
	copy := s.Clone()
	v.armed = &copy
	v.at = v.h.at
	return nil
}
func (v *privateLaunchBirthStub) Capture(ctx context.Context, s privatechain.MemberLaunch, observe privatechain.MemberBirthObserver) (privatechain.MemberBirthEvidence, error) {
	if err := v.h.event("capture"); err != nil {
		return privatechain.MemberBirthEvidence{}, err
	}
	if v.armed == nil {
		v.h.t.Fatal("capture before arm")
	}
	a, err := observe(ctx, s)
	if err != nil {
		return privatechain.MemberBirthEvidence{}, err
	}
	b, err := observe(ctx, s)
	if err != nil {
		return privatechain.MemberBirthEvidence{}, err
	}
	return privatechain.MemberBirthEvidence{InstanceID: s.InstanceID, OperationID: s.OperationID, RuntimeID: s.RuntimeID, ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, EmptyObservedAt: v.at, CapturedAt: v.h.at, Before: a, After: b, Pier: pierstore.Identity{Device: 12, Inode: 800, UrbDevice: 12, UrbInode: 801, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}, nil
}

type privateLaunchHarness struct {
	t                                 *testing.T
	d                                 *privateMemberLaunchDriver
	saved                             privatechain.MemberLaunch
	at, started                       time.Time
	fail, cancelAt                    string
	cancel                            context.CancelFunc
	events                            []string
	proofs, creates, starts, prepares int
	keyHash                           string
	mutateGateway                     func(*privatechain.GatewayObservation)
	mutateProcess                     func(*privatechain.MemberBirthProcess)
}

func (h *privateLaunchHarness) event(name string) error {
	h.events = append(h.events, name)
	if h.cancelAt == name {
		h.cancel()
	}
	if h.fail == name {
		return errors.New("PRIVATE synthetic detail")
	}
	return nil
}
func newPrivateLaunchHarness(t *testing.T) *privateLaunchHarness {
	c, proof := privateKeyProofFixture(t)
	s, err := privatechain.NewMemberLaunch(c.selection.Plan, c.selection.Workspace, c.selection.Gateway, c.selection.InstanceID, c.selection.RuntimeID, c.selection.OperationID, c.digest, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := &privateLaunchHarness{t: t, saved: s.Clone(), at: c.selection.Gateway.Observation.ObservedAt.Add(time.Second), keyHash: proof.KeySHA256}
	d := &privateMemberLaunchDriver{contract: c, initial: s.Clone(), now: func() time.Time { return h.at }}
	h.d = d
	d.mount = &privateLaunchHeldStub{h: h, name: "empty"}
	d.birth = &privateLaunchBirthStub{privateLaunchHeldStub: privateLaunchHeldStub{h: h, name: "birth-held"}}
	d.ops = privateMemberLaunchOps{
		prepare: func(ctx context.Context) (pierstore.DirectoryIdentity, error) {
			h.prepares++
			if h.saved.Phase != "artifact-dispatched" {
				t.Fatal("artifact before commit")
			}
			id := pierstore.DirectoryIdentity{Device: 12, Inode: 700, Birth: "1000000000:1"}
			return id, h.event("prepare")
		},
		pill: func(ctx context.Context, id pierstore.DirectoryIdentity) (privateMemberHeld, error) {
			if err := h.event("pill-open"); err != nil {
				return nil, err
			}
			return &privateLaunchHeldStub{h: h, name: "pill-held"}, nil
		},
		key: func(ctx context.Context, hash string) (privateMemberHeld, error) {
			if hash != h.keyHash {
				t.Fatal("wrong key hash")
			}
			if err := h.event("key-open"); err != nil {
				return nil, err
			}
			return &privateLaunchHeldStub{h: h, name: "key-held"}, nil
		},
		proof: func(ctx context.Context) (privateMemberKeyProofObservation, error) {
			h.proofs++
			o := proof
			o.KeySHA256 = h.keyHash
			return o, h.event("proof")
		},
		gateway: func(ctx context.Context, m []privateGatewayMember, p []privatechain.GatewayMemberPier) (privatechain.GatewayObservation, error) {
			if len(m) > 1 || len(p) > 1 {
				t.Fatal("unowned members")
			}
			if d.created != "" && (len(m) != 1 || m[0].ContainerID != d.created || !reflect.DeepEqual(m[0].Contract, c)) {
				t.Fatal("lost created member")
			}
			if d.startAcknowledged && (len(p) != 1 || m[0].State != "running" || m[0].StartedAt != h.started) {
				t.Fatal("running gateway lacks exact birth")
			}
			o := c.selection.Gateway.Observation.Clone()
			o.ObservedAt = h.at
			if h.mutateGateway != nil {
				h.mutateGateway(&o)
			}
			return o, h.event("gateway")
		},
		create: func(ctx context.Context) (string, error) {
			h.creates++
			if h.saved.Phase != "create-dispatched" {
				t.Fatal("create before commit")
			}
			return strings.Repeat("c", 64), h.event("create")
		},
		start: func(ctx context.Context, id string) error {
			h.starts++
			if h.saved.Phase != "start-dispatched" || id != d.created {
				t.Fatal("start before commit")
			}
			h.at = h.at.Add(time.Second)
			h.started = h.at
			return h.event("start")
		},
		process: func(ctx context.Context, id string, running bool) (privatechain.MemberBirthProcess, error) {
			if err := h.event("process"); err != nil {
				return privatechain.MemberBirthProcess{}, err
			}
			if !running {
				return privatechain.MemberBirthProcess{}, nil
			}
			o := privatechain.MemberBirthProcess{ObservedAt: h.at, StartedAt: h.started, ContainerID: id, ContractDigest: c.digest, Namespace: c.selection.Gateway.Observation.Namespace, CompleteContract: true, Running: true}
			if h.mutateProcess != nil {
				h.mutateProcess(&o)
			}
			return o, nil
		},
		waitPier: func(ctx context.Context) error { return h.event("wait-pier") },
	}
	t.Cleanup(func() { d.Close() })
	return h
}
func (h *privateLaunchHarness) run(ctx context.Context) (privatechain.MemberLaunch, error) {
	return privatechain.ExecuteMemberLaunch(ctx, h.d.contract.selection.Plan, h.d.contract.selection.Workspace, h.d.initial, h.d, func(expected string, next privatechain.MemberLaunch) error {
		if expected != h.saved.Phase {
			h.t.Fatal("wrong journal ancestry")
		}
		if err := h.event("commit-" + next.Phase); err != nil {
			return err
		}
		h.saved = next.Clone()
		return nil
	}, h.d.now)
}
func TestPrivateMemberDriverExactFirstLaunch(t *testing.T) {
	h := newPrivateLaunchHarness(t)
	got, err := h.run(context.Background())
	if err != nil || got.Phase != "running" || h.proofs != 3 || h.creates != 1 || h.starts != 1 || h.prepares != 1 || h.d.birthEvidence == nil {
		t.Fatal("incomplete launch", err, got.Phase, h.events)
	}
	if err = h.d.birthEvidence.Validate(h.d.contract.selection.Plan, h.d.contract.selection.Workspace, got); err != nil {
		t.Fatal(err)
	}
	if _, err = h.d.ObserveRunning(context.Background(), got); err == nil {
		t.Fatal("observed twice")
	}
	key := h.d.key.(*privateLaunchHeldStub)
	pill := h.d.pill.(*privateLaunchHeldStub)
	h.d.Close()
	h.d.Close()
	if key.closes != 1 || pill.closes != 1 {
		t.Fatal("close repeated")
	}
}
func TestPrivateMemberDriverFailureStopsEffects(t *testing.T) {
	for _, at := range []string{"empty", "gateway", "prepare", "pill-open", "proof", "key-open", "key-held", "pill-held", "create", "process", "arm", "start", "wait-pier", "capture", "birth-held", "commit-artifact-dispatched", "commit-artifact-created", "commit-key-verified", "commit-create-dispatched", "commit-created", "commit-start-dispatched", "commit-started", "commit-running"} {
		t.Run(at, func(t *testing.T) {
			h := newPrivateLaunchHarness(t)
			h.fail = at
			got, err := h.run(context.Background())
			if err == nil || strings.Contains(err.Error(), "PRIVATE") || h.creates > 1 || h.starts > 1 || h.prepares > 1 {
				t.Fatal("uncertainty not contained", err)
			}
			if at == "prepare" && (got.Phase != "artifact-created" || got.Artifact.Inode != 700) {
				t.Fatal("lost artifact identity")
			}
			if at == "create" && (got.Phase != "created" || got.ContainerID == "") {
				t.Fatal("lost container identity")
			}
			if at == "start" && (got.Phase != "start-dispatched" || h.d.startAcknowledged) {
				t.Fatal("lost Start reply promoted")
			}
			if at == "commit-created" && h.starts != 0 {
				t.Fatal("started after uncommitted create")
			}
			if at == "commit-started" && h.d.birthEvidence != nil {
				t.Fatal("captured after lost commit reply")
			}
		})
	}
}
func TestPrivateMemberDriverCancellationStopsEffects(t *testing.T) {
	for _, at := range []string{"prepare", "proof", "create", "arm", "start", "wait-pier", "capture", "gateway", "commit-create-dispatched", "commit-start-dispatched"} {
		t.Run(at, func(t *testing.T) {
			h := newPrivateLaunchHarness(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h.cancelAt = at
			h.cancel = cancel
			_, err := h.run(ctx)
			if err == nil {
				t.Fatal("canceled launch accepted")
			}
			if (at == "arm" || at == "commit-start-dispatched") && h.starts != 0 {
				t.Fatal("effect after cancellation")
			}
			if at == "commit-create-dispatched" && h.creates != 0 {
				t.Fatal("create after cancellation")
			}
		})
	}
}
func TestPrivateMemberDriverRefusesChangedGateway(t *testing.T) {
	for _, at := range []string{"head", "namespace", "secrets", "public-key", "old", "future", "incomplete"} {
		t.Run(at, func(t *testing.T) {
			h := newPrivateLaunchHarness(t)
			h.mutateGateway = func(g *privatechain.GatewayObservation) {
				switch at {
				case "head":
					g.HeadHash = "changed"
				case "namespace":
					g.Namespace.Inode++
				case "secrets":
					g.Secrets.Inode++
				case "public-key":
					g.PublicKeys[0].Auth = "changed"
				case "old":
					g.ObservedAt = h.at.Add(-31 * time.Second)
				case "future":
					g.ObservedAt = h.at.Add(6 * time.Second)
				case "incomplete":
					g.CompleteContract = false
				}
			}
			if _, err := h.run(context.Background()); err == nil || h.prepares != 0 {
				t.Fatal("changed gateway authorized effect")
			}
		})
	}
}
func TestPrivateMemberDriverRefusesChangedProcess(t *testing.T) {
	for _, at := range []string{"namespace", "container", "start", "contract", "incomplete"} {
		t.Run(at, func(t *testing.T) {
			h := newPrivateLaunchHarness(t)
			h.mutateProcess = func(p *privatechain.MemberBirthProcess) {
				switch at {
				case "namespace":
					p.Namespace.Inode++
				case "container":
					p.ContainerID = strings.Repeat("d", 64)
				case "start":
					p.StartedAt = h.at.Add(-time.Hour)
				case "contract":
					p.ContractDigest = strings.Repeat("f", 64)
				case "incomplete":
					p.CompleteContract = false
				}
			}
			if _, err := h.run(context.Background()); err == nil || h.d.birthEvidence != nil {
				t.Fatal("invalid birth accepted")
			}
		})
	}
}
func TestPrivateMemberDriverChangedKeyPreventsCreate(t *testing.T) {
	h := newPrivateLaunchHarness(t)
	proof := h.d.ops.proof
	h.d.ops.proof = func(ctx context.Context) (privateMemberKeyProofObservation, error) {
		if h.proofs == 1 {
			h.keyHash = strings.Repeat("d", 64)
		}
		return proof(ctx)
	}
	if _, err := h.run(context.Background()); err == nil || h.creates != 0 {
		t.Fatal("key changed after acceptance")
	}
}
func TestPrivateMemberDriverSelectionAndUnacknowledgedStart(t *testing.T) {
	h := newPrivateLaunchHarness(t)
	s := h.d.initial.Clone()
	s.OperationID = "different-operation"
	if h.d.VerifyPreparation(context.Background(), s) == nil || h.prepares != 0 {
		t.Fatal("changed selection")
	}
	h.fail = "start"
	got, err := h.run(context.Background())
	if err == nil {
		t.Fatal("lost start")
	}
	next, err := got.Advance(h.d.contract.selection.Plan, h.d.contract.selection.Workspace, "start-dispatched", "started", privatechain.MemberLaunchResult{})
	if err != nil {
		t.Fatal(err)
	}
	h.fail = ""
	if _, err = h.d.ObserveRunning(context.Background(), next); err == nil {
		t.Fatal("forged start acknowledgement")
	}
	if h.d.StartMember(context.Background(), got) == nil || h.starts != 1 {
		t.Fatal("retried start")
	}
}

func TestPrivateMemberDriverRetainsCaptureOnFinalFailure(t *testing.T) {
	for _, kind := range []string{"gateway", "key", "pill", "process"} {
		t.Run(kind, func(t *testing.T) {
			h := newPrivateLaunchHarness(t)
			base := h.d.ops.gateway
			h.d.ops.gateway = func(ctx context.Context, m []privateGatewayMember, p []privatechain.GatewayMemberPier) (privatechain.GatewayObservation, error) {
				if h.d.birthEvidence != nil {
					switch kind {
					case "gateway":
						h.fail = "gateway"
					case "key":
						h.fail = "key-held"
					case "pill":
						h.fail = "pill-held"
					case "process":
						h.fail = "process"
					}
				}
				return base(ctx, m, p)
			}
			got, err := h.run(context.Background())
			if err == nil || got.Phase != "started" || h.d.birthEvidence == nil {
				t.Fatal("capture discarded or failed readiness promoted", err, got.Phase)
			}
		})
	}
}

func TestPrivateMemberDriverLostCreateCannotStart(t *testing.T) {
	h := newPrivateLaunchHarness(t)
	h.fail = "create"
	got, err := h.run(context.Background())
	if err == nil || got.Phase != "created" {
		t.Fatal("fixture")
	}
	h.fail = ""
	if h.d.InspectCreated(context.Background(), got) == nil {
		t.Fatal("uncertain create reused")
	}
	next, err := got.Advance(h.d.contract.selection.Plan, h.d.contract.selection.Workspace, "created", "start-dispatched", privatechain.MemberLaunchResult{})
	if err != nil {
		t.Fatal(err)
	}
	if h.d.StartMember(context.Background(), next) == nil || h.starts != 0 {
		t.Fatal("started after lost create acknowledgement")
	}
}
func TestPrivateMemberProcessConcreteBracket(t *testing.T) {
	for _, kind := range []string{"created", "running", "changed", "namespace", "contract", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			c := privateMemberContractFixture(t, "sender", "boot")
			rawBoot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
			if err != nil {
				t.Fatal(err)
			}
			c.selection.Gateway.Observation.Namespace.BootID = strings.TrimSpace(string(rawBoot))
			c.selection.Gateway.Observation.HostNamespace.BootID = strings.TrimSpace(string(rawBoot))
			c, err = newPrivateMemberContract(c.selection, c.root, c.imageID)
			if err != nil {
				t.Fatal(err)
			}
			v := privateMemberContainerFixture(t, c)
			running := kind != "created"
			if running {
				*v.State.Running = true
				v.State.Status = "running"
				v.State.Pid = 42
				v.State.StartedAt = c.selection.Gateway.Observation.ObservedAt.Format(time.RFC3339Nano)
			}
			calls, execs := 0, 0
			q := newPrivateObserverFixture(t).o.runtime.q
			q.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				a := args[4:]
				if len(a) == 3 && a[0] == "container" && a[1] == "inspect" && a[2] == v.ID {
					calls++
					if kind == "changed" && calls > 1 {
						v.State.Pid++
					}
					if kind == "contract" {
						v.Config.Labels["unexpected"] = "yes"
					}
					return string(privateMemberContainerBytes(t, v)), nil
				}
				if len(a) == 8 && a[0] == "container" && a[1] == "exec" && a[2] == v.ID {
					execs++
					if kind == "namespace" {
						return "4:999", nil
					}
					return "4:101", nil
				}
				t.Fatalf("unexpected command %v", a)
				return "", nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "canceled" {
				cancel()
			}
			_, err = observePrivateMemberProcess(ctx, q, c, v.ID, running, func() time.Time { return c.selection.Gateway.Observation.ObservedAt })
			if (err == nil) != (kind == "created" || kind == "running") {
				t.Fatal(kind, err)
			}
			if kind == "created" && (execs != 0 || calls != 2) {
				t.Fatal("created container executed")
			}
			if kind == "running" && (execs != 1 || calls != 2) {
				t.Fatal("missing process bracket", calls, execs)
			}
		})
	}
}
