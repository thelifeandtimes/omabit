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
)

type memberLaunchHarness struct {
	t                                           *testing.T
	p                                           Plan
	w                                           Workspace
	saved                                       MemberLaunch
	events                                      []string
	fail                                        string
	lostCommit, noIdentity, badIdentity, mutate bool
	cancelAt                                    string
	cancel                                      context.CancelFunc
	keyChange                                   func(*MemberKeyAcceptance)
	runningChange                               func(*MemberRunningObservation)
	artifacts, creates, starts                  int
}

func newMemberLaunchHarness(t *testing.T, peer bool) (*memberLaunchHarness, MemberLaunch) {
	t.Helper()
	p, w, g := gatewayFixture(t)
	g.Phase = "ready"
	g.NetworkID = strings.Repeat("2", 64)
	g.ContainerID = strings.Repeat("3", 64)
	o := gatewayObservation(p, w, g)
	g.Observation = &o
	var peers []MemberPeer
	if peer {
		peers = []MemberPeer{{InstanceID: p.Members[1].InstanceID, RuntimeID: strings.Repeat("e", 32), OperationID: "peer-launch", ContainerID: strings.Repeat("7", 64), ContractDigest: strings.Repeat("8", 64), ReadinessDigest: strings.Repeat("9", 64), Generation: 1, StartedAt: workspaceNow.Add(time.Second), Pier: pierstore.Identity{Device: 12, Inode: 200, UrbDevice: 12, UrbInode: 201, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}}
	}
	s, err := NewMemberLaunch(p, w, g, p.Members[0].InstanceID, strings.Repeat("d", 32), "member-launch", strings.Repeat("6", 64), peers)
	if err != nil {
		t.Fatal(err)
	}
	return &memberLaunchHarness{t: t, p: p, w: w, saved: s.Clone()}, s
}
func (h *memberLaunchHarness) now() time.Time { return workspaceNow.Add(4 * time.Second) }
func (h *memberLaunchHarness) event(name string) error {
	h.events = append(h.events, name)
	if h.cancelAt == name {
		h.cancel()
	}
	if h.fail == name {
		return errors.New("PRIVATE driver detail must not escape")
	}
	return nil
}
func (h *memberLaunchHarness) selected(s MemberLaunch, phase string) {
	h.t.Helper()
	if s.Phase != phase || !reflect.DeepEqual(s, h.saved) {
		h.t.Fatal("effect preceded durable exact dispatch", s.Phase, phase)
	}
	if h.mutate {
		s.Gateway.Observation.PublicKeys[0].Auth = "changed-driver-copy"
		if len(s.Peers) > 0 {
			s.Peers[0].RuntimeID = "changed-driver-peer"
		}
	}
}
func (h *memberLaunchHarness) VerifyPreparation(_ context.Context, s MemberLaunch) error {
	if s.Validate(h.p, h.w) != nil {
		h.t.Fatal("invalid preparation")
	}
	h.selected(s, s.Phase)
	return h.event("verify-" + s.Phase)
}
func (h *memberLaunchHarness) PrepareArtifact(_ context.Context, s MemberLaunch) (pierstore.DirectoryIdentity, error) {
	h.selected(s, "artifact-dispatched")
	h.artifacts++
	err := h.event("artifact")
	if h.noIdentity {
		return pierstore.DirectoryIdentity{}, err
	}
	if h.badIdentity {
		return workspaceID(10), err
	}
	return workspaceID(100), err
}
func (h *memberLaunchHarness) VerifyKey(_ context.Context, s MemberLaunch) (MemberKeyAcceptance, error) {
	h.selected(s, "artifact-created")
	g := h.saved.Gateway.Observation.Clone()
	g.ObservedAt = workspaceNow.Add(2 * time.Second)
	k := MemberKeyAcceptance{CheckedAt: g.ObservedAt, ContractDigest: h.saved.ContractDigest, KeySHA256: strings.Repeat("a", 64), Gateway: g, Verified: true}
	if h.keyChange != nil {
		h.keyChange(&k)
	}
	return k, h.event("key")
}
func (h *memberLaunchHarness) CreateMember(_ context.Context, s MemberLaunch) (string, error) {
	h.selected(s, "create-dispatched")
	h.creates++
	err := h.event("create")
	if h.noIdentity {
		return "", err
	}
	if h.badIdentity {
		return h.saved.Gateway.ContainerID, err
	}
	return strings.Repeat("5", 64), err
}
func (h *memberLaunchHarness) InspectCreated(_ context.Context, s MemberLaunch) error {
	h.selected(s, "created")
	return h.event("inspect-created")
}
func (h *memberLaunchHarness) StartMember(_ context.Context, s MemberLaunch) error {
	h.selected(s, "start-dispatched")
	h.starts++
	return h.event("start")
}
func (h *memberLaunchHarness) ObserveRunning(_ context.Context, s MemberLaunch) (MemberRunningObservation, error) {
	h.selected(s, "started")
	g := h.saved.Gateway.Observation.Clone()
	g.ObservedAt = h.now()
	o := MemberRunningObservation{ObservedAt: h.now(), ContainerID: h.saved.ContainerID, ContractDigest: h.saved.ContractDigest, StartedAt: workspaceNow.Add(3 * time.Second), Namespace: g.Namespace, Gateway: g, Peers: slices.Clone(h.saved.Peers), CompleteContract: true, Running: true}
	process := MemberBirthProcess{ObservedAt: h.now(), StartedAt: o.StartedAt, ContainerID: o.ContainerID, ContractDigest: o.ContractDigest, Namespace: o.Namespace, CompleteContract: true, Running: true}
	o.Birth = &MemberBirthEvidence{InstanceID: s.InstanceID, OperationID: s.OperationID, RuntimeID: s.RuntimeID, ContainerID: s.ContainerID, ContractDigest: s.ContractDigest, EmptyObservedAt: s.Key.CheckedAt, CapturedAt: h.now(), Before: process, After: process, Pier: pierstore.Identity{Device: 12, Inode: 300, UrbDevice: 12, UrbInode: 301, Birth: "1000000000:1", UrbBirth: "1000000000:2"}}
	if h.runningChange != nil {
		h.runningChange(&o)
	}
	return o, h.event("observe")
}
func (h *memberLaunchHarness) commit(expected string, next MemberLaunch) error {
	if expected != h.saved.Phase || next.Validate(h.p, h.w) != nil {
		h.t.Fatal("invalid commit")
	}
	err := h.event("commit-" + next.Phase)
	if err == nil || h.lostCommit {
		h.saved = next.Clone()
	}
	if h.mutate {
		next.Gateway.Observation.PublicKeys[0].Auth = "changed-commit-copy"
		if next.Key != nil {
			next.Key.Gateway.Mounts[0].Inode++
		}
	}
	return err
}
func (h *memberLaunchHarness) execute(ctx context.Context, s MemberLaunch) (MemberLaunch, error) {
	return ExecuteMemberLaunch(ctx, h.p, h.w, s, h, h.commit, h.now)
}

func TestPrivateMemberLaunchSerialFreshDispatch(t *testing.T) {
	for _, peer := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "second"}[peer], func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, peer)
			before := s.Clone()
			got, err := h.execute(context.Background(), s)
			if err != nil || got.Phase != "running" || got.Running == nil || got.Validate(h.p, h.w) != nil || !reflect.DeepEqual(h.saved, got) {
				t.Fatal(got.Phase, err)
			}
			if !reflect.DeepEqual(before, s) || h.artifacts != 1 || h.creates != 1 || h.starts != 1 {
				t.Fatal("input changed or duplicate effect")
			}
			want := []string{"verify-planned", "commit-artifact-dispatched", "artifact", "commit-artifact-created", "verify-artifact-created", "key", "commit-key-verified", "verify-key-verified", "commit-create-dispatched", "create", "commit-created", "inspect-created", "verify-created", "commit-start-dispatched", "start", "commit-started", "observe", "commit-running"}
			if !reflect.DeepEqual(h.events, want) {
				t.Fatal(h.events)
			}
			n := len(h.events)
			if _, err = h.execute(context.Background(), h.saved); err == nil || len(h.events) != n {
				t.Fatal("completed boot replayed")
			}
		})
	}
}

func TestPrivateMemberLaunchCommitLossNeverReplays(t *testing.T) {
	for _, phase := range []string{"artifact-dispatched", "artifact-created", "key-verified", "create-dispatched", "created", "start-dispatched", "started", "running"} {
		for _, lost := range []bool{false, true} {
			t.Run(phase+map[bool]string{false: "/before", true: "/after"}[lost], func(t *testing.T) {
				h, s := newMemberLaunchHarness(t, true)
				h.fail = "commit-" + phase
				h.lostCommit = lost
				_, err := h.execute(context.Background(), s)
				if !errors.Is(err, ErrMemberLaunchUnresolved) || strings.Contains(err.Error(), "PRIVATE") || h.saved.Validate(h.p, h.w) != nil {
					t.Fatal(err)
				}
				if lost && h.saved.Phase != phase {
					t.Fatal("durable result erased")
				}
				if phase == "artifact-dispatched" && h.artifacts != 0 || phase == "create-dispatched" && h.creates != 0 || phase == "start-dispatched" && h.starts != 0 {
					t.Fatal("effect after uncertain dispatch commit")
				}
				if h.saved.Phase != "planned" {
					n := len(h.events)
					if _, err = h.execute(context.Background(), h.saved); err == nil || len(h.events) != n {
						t.Fatal("uncertain boot replayed")
					}
				}
			})
		}
	}
}

func TestPrivateMemberLaunchEffectFailuresRetainKnownObjects(t *testing.T) {
	for _, stage := range []string{"verify-planned", "artifact", "verify-artifact-created", "key", "verify-key-verified", "create", "inspect-created", "verify-created", "start", "observe"} {
		t.Run(stage, func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, false)
			h.fail = stage
			got, err := h.execute(context.Background(), s)
			if err == nil || strings.Contains(err.Error(), "PRIVATE") || got.Validate(h.p, h.w) != nil || h.artifacts > 1 || h.creates > 1 || h.starts > 1 {
				t.Fatal(err)
			}
			if stage == "artifact" && (h.saved.Phase != "artifact-created" || h.saved.Artifact != workspaceID(100)) {
				t.Fatal("known artifact discarded")
			}
			if stage == "create" && (h.saved.Phase != "created" || h.saved.ContainerID != strings.Repeat("5", 64)) {
				t.Fatal("known container discarded")
			}
			if stage == "start" && h.saved.Phase != "start-dispatched" {
				t.Fatal("uncertain Start relabelled")
			}
			if h.saved.Phase != "planned" {
				n := len(h.events)
				if _, e := h.execute(context.Background(), h.saved); e == nil || len(h.events) != n {
					t.Fatal("failed boot replayed")
				}
			}
		})
	}
}

func TestPrivateMemberLaunchCancellationPreservesEffects(t *testing.T) {
	for _, at := range []string{"before", "verify-planned", "commit-artifact-dispatched", "artifact", "key", "commit-key-verified", "commit-create-dispatched", "create", "inspect-created", "commit-start-dispatched", "start", "observe", "commit-running"} {
		t.Run(at, func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h.cancel = cancel
			h.cancelAt = at
			if at == "before" {
				cancel()
			}
			got, err := h.execute(ctx, s)
			// A cancellation delivered by the successful terminal commit does not
			// erase completed evidence. Earlier cancellation fences later effects.
			if at == "commit-running" {
				if err != nil || got.Phase != "running" {
					t.Fatal(err)
				}
			} else if !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost", err)
			}
			if h.saved.Validate(h.p, h.w) != nil || h.artifacts > 1 || h.creates > 1 || h.starts > 1 {
				t.Fatal("invalid retained evidence")
			}
			if at == "commit-artifact-dispatched" && h.artifacts != 0 || at == "commit-create-dispatched" && h.creates != 0 || at == "commit-start-dispatched" && h.starts != 0 {
				t.Fatal("effect after cancellation")
			}
			if at == "artifact" && h.saved.Artifact != workspaceID(100) || at == "create" && h.saved.ContainerID != strings.Repeat("5", 64) {
				t.Fatal("known effect lost on cancellation")
			}
		})
	}
}

func TestPrivateMemberLaunchRejectsKeyAndRuntimeEvidence(t *testing.T) {
	for _, kind := range []string{"key-unverified", "key-digest", "key-contract", "key-stale", "key-future", "key-before-gateway", "key-chain", "key-namespace", "running-container", "running-contract", "running-stopped", "running-incomplete", "running-start", "running-stale", "running-future", "running-namespace", "running-peer", "running-chain"} {
		t.Run(kind, func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, true)
			h.keyChange = func(k *MemberKeyAcceptance) {
				switch kind {
				case "key-unverified":
					k.Verified = false
				case "key-digest":
					k.KeySHA256 = strings.Repeat("0", 64)
				case "key-contract":
					k.ContractDigest = strings.Repeat("f", 64)
				case "key-stale":
					k.CheckedAt = h.now().Add(-31 * time.Second)
				case "key-future":
					k.CheckedAt = h.now().Add(6 * time.Second)
					k.Gateway.ObservedAt = k.CheckedAt
				case "key-before-gateway":
					k.CheckedAt = workspaceNow.Add(-time.Second)
					k.Gateway.ObservedAt = k.CheckedAt
				case "key-chain":
					k.Gateway.HeadHash = "0x" + strings.Repeat("f", 64)
				case "key-namespace":
					k.Gateway.Namespace.Inode++
				}
			}
			h.runningChange = func(o *MemberRunningObservation) {
				switch kind {
				case "running-container":
					o.ContainerID = strings.Repeat("f", 64)
				case "running-contract":
					o.ContractDigest = strings.Repeat("f", 64)
				case "running-stopped":
					o.Running = false
				case "running-incomplete":
					o.CompleteContract = false
				case "running-start":
					o.StartedAt = workspaceNow
				case "running-stale":
					o.ObservedAt = h.now().Add(-31 * time.Second)
					o.Gateway.ObservedAt = o.ObservedAt
				case "running-future":
					o.ObservedAt = h.now().Add(6 * time.Second)
					o.Gateway.ObservedAt = o.ObservedAt
				case "running-namespace":
					o.Namespace.Inode++
				case "running-peer":
					o.Peers[0].ContainerID = strings.Repeat("f", 64)
				case "running-chain":
					o.Gateway.PublicKeys[0].Life++
				}
			}
			got, err := h.execute(context.Background(), s)
			if err == nil || got.Phase == "running" {
				t.Fatal("invalid authority accepted")
			}
			if strings.HasPrefix(kind, "key-") && (h.creates != 0 || h.starts != 0 || h.saved.Key != nil) {
				t.Fatal("invalid key admitted effects")
			}
		})
	}
}

func TestPrivateMemberLaunchSelectionAndPhaseValidation(t *testing.T) {
	for _, kind := range []string{"version", "runtime", "zero-runtime", "operation", "instance", "contract", "pill", "gateway-phase", "peer-instance", "peer-runtime", "peer-operation", "peer-container", "peer-pier", "peer-readiness", "phase", "early-artifact", "early-key", "early-container", "early-running"} {
		t.Run(kind, func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, true)
			switch kind {
			case "version":
				s.Version = 4 // Unknown after the capture-journal successor.
			case "runtime":
				s.RuntimeID = "invalid"
			case "zero-runtime":
				s.RuntimeID = strings.Repeat("0", 32)
			case "operation":
				s.OperationID = s.Gateway.Selection.OperationID
			case "instance":
				s.InstanceID = "unknown"
			case "contract":
				s.ContractDigest = "not-a-digest"
			case "pill":
				s.Pill.Bytes++
			case "gateway-phase":
				s.Gateway.Phase = "started"
				s.Gateway.Observation = nil
			case "peer-instance":
				s.Peers[0].InstanceID = s.InstanceID
			case "peer-runtime":
				s.Peers[0].RuntimeID = s.RuntimeID
			case "peer-operation":
				s.Peers[0].OperationID = s.OperationID
			case "peer-container":
				s.Peers[0].ContainerID = s.Gateway.ContainerID
			case "peer-pier":
				s.Peers[0].Pier.Inode = h.w.Root.Inode
			case "peer-readiness":
				s.Peers[0].ReadinessDigest = ""
			case "phase":
				s.Phase = "ready"
			case "early-artifact":
				s.Artifact = workspaceID(100)
			case "early-key":
				s.Key = &MemberKeyAcceptance{}
			case "early-container":
				s.ContainerID = strings.Repeat("5", 64)
			case "early-running":
				s.Running = &MemberRunningObservation{}
			}
			if s.Validate(h.p, h.w) == nil {
				t.Fatal("invalid history accepted")
			}
			if _, err := h.execute(context.Background(), s); err == nil || len(h.events) != 0 {
				t.Fatal("invalid intent reached adapter")
			}
		})
	}
}

func TestPrivateMemberLaunchImmutableCopiesAndBadEffectIdentities(t *testing.T) {
	t.Run("copies", func(t *testing.T) {
		h, s := newMemberLaunchHarness(t, true)
		h.mutate = true
		before := s.Clone()
		got, err := h.execute(context.Background(), s)
		if err != nil || got.Validate(h.p, h.w) != nil || !reflect.DeepEqual(before, s) {
			t.Fatal(err)
		}
	})
	for _, bad := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-artifact", true: "alias-artifact"}[bad], func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, false)
			h.noIdentity = !bad
			h.badIdentity = bad
			_, err := h.execute(context.Background(), s)
			if err == nil || h.saved.Phase != "artifact-dispatched" || h.creates != 0 {
				t.Fatal("bad artifact adopted")
			}
		})
	}
	for _, bad := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-container", true: "alias-container"}[bad], func(t *testing.T) {
			h, s := newMemberLaunchHarness(t, false)
			h.keyChange = func(*MemberKeyAcceptance) { h.noIdentity = !bad; h.badIdentity = bad }
			_, err := h.execute(context.Background(), s)
			if err == nil || h.saved.Phase != "create-dispatched" || h.creates != 1 || h.starts != 0 {
				t.Fatal("bad container adopted")
			}
		})
	}
	t.Run("skip", func(t *testing.T) {
		h, s := newMemberLaunchHarness(t, false)
		if _, err := s.Advance(h.p, h.w, "planned", "created", MemberLaunchResult{ContainerID: strings.Repeat("5", 64)}); err == nil {
			t.Fatal("phase skipped")
		}
		if _, err := s.Advance(h.p, h.w, "planned", "artifact-dispatched", MemberLaunchResult{ContainerID: strings.Repeat("5", 64)}); err == nil {
			t.Fatal("unrelated result admitted")
		}
	})
}
