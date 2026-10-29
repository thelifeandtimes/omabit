package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

func memberLaunchFixture(t *testing.T) (Plan, MemberLaunch, Observation) {
	t.Helper()
	plan, anchor, o := anchorStartFixture(t)
	ns := NamespaceIdentity{BootID: "01234567-89ab-cdef-0123-456789abcdef", Device: 5, Inode: 77}
	anchor.Phase = "observed"
	anchor.Namespace = &ns
	storage, err := NewMemberStorage(plan, anchor.Prepared.ProjectID, 1, "zod", strings.Repeat("e", 32), "storage-zod")
	if err != nil {
		t.Fatal(err)
	}
	storage.Phase = "observed"
	storage.Location.MountIdentity = pierstore.DirectoryIdentity{Device: 8, Inode: 91, Birth: "1000:1"}
	peer := RuntimeMember{InstanceID: "nec", RuntimeID: strings.Repeat("d", 32), Container: ResourceIdentity{strings.Repeat("d", 64), strings.Repeat("4", 64)}}
	s, err := NewMemberLaunch(plan, anchor, storage, "launch-zod", strings.Repeat("3", 64), []RuntimeMember{peer})
	if err != nil {
		t.Fatal(err)
	}
	o.Anchor.Running = true
	o.Anchor.Namespace = ns
	o.Anchor.Active = append([]Publication(nil), o.Anchor.Requested...)
	o.BridgeAttachments = []string{anchor.Prepared.Anchor.ID}
	o.Members = []MemberObservation{{InstanceID: peer.InstanceID, ResourceObservation: ResourceObservation{Resource: peer.Container, Present: true}, Running: true, AnchorID: anchor.Prepared.Anchor.ID, Namespace: ns}}
	return plan, s, o
}

type memberLaunchDriverFixture struct {
	t                                                            *testing.T
	state                                                        *MemberLaunch
	base                                                         Observation
	fault                                                        string
	cancel                                                       context.CancelFunc
	prepares, creates, starts, preflights, inspections, verifies int
}

func (d *memberLaunchDriverFixture) check(s MemberLaunch, phase string) {
	d.t.Helper()
	if !reflect.DeepEqual(s, *d.state) || s.Phase != phase {
		d.t.Fatal("effect before committed phase", s.Phase, phase)
	}
	// Driver arguments cannot rewrite the retained anchor receipt or peer list.
	s.Anchor.Namespace.Inode++
	if len(s.Peers) > 0 {
		s.Peers[0].InstanceID = "changed"
	}
}
func (d *memberLaunchDriverFixture) VerifyEngine(context.Context, string) error {
	d.verifies++
	return nil
}
func (d *memberLaunchDriverFixture) PrepareWorkspace(_ context.Context, s MemberLaunch) (pierstore.DirectoryIdentity, error) {
	d.check(s, "workspace-dispatched")
	d.prepares++
	id := pierstore.DirectoryIdentity{Device: 8, Inode: 100, Birth: "1000:2"}
	if d.fault == "workspace-no-id" {
		return pierstore.DirectoryIdentity{}, nil
	}
	if d.fault == "workspace-late-error" {
		return id, errors.New("synthetic provider detail")
	}
	return id, nil
}
func (d *memberLaunchDriverFixture) Preflight(_ context.Context, s MemberLaunch) error {
	phase := "workspace-created"
	if d.preflights > 0 {
		phase = "created"
	}
	d.check(s, phase)
	d.preflights++
	if d.fault == "preflight" || d.fault == "preflight-created" && d.preflights == 2 {
		return errors.New("synthetic provider detail")
	}
	return nil
}
func (d *memberLaunchDriverFixture) Create(_ context.Context, s MemberLaunch) (string, error) {
	d.check(s, "create-dispatched")
	d.creates++
	if d.fault == "create-no-id" {
		return "", nil
	}
	id := strings.Repeat("c", 64)
	if d.fault == "create-late-error" {
		return id, errors.New("synthetic provider detail")
	}
	if d.fault == "cancel-after-create" {
		d.cancel()
	}
	return id, nil
}
func (d *memberLaunchDriverFixture) Start(_ context.Context, s MemberLaunch) error {
	d.check(s, "start-dispatched")
	d.starts++
	if d.fault == "lost-start-reply" {
		return errors.New("synthetic provider detail")
	}
	if d.fault == "cancel-after-start" {
		d.cancel()
	}
	return nil
}
func (d *memberLaunchDriverFixture) Inspect(_ context.Context, s MemberLaunch) (Observation, error) {
	d.inspections++
	if !reflect.DeepEqual(s, *d.state) {
		d.t.Fatal("inspection preceded commit")
	}
	o := d.base
	o.Members = append([]MemberObservation(nil), o.Members...)
	if s.Container.ID != "" {
		running := s.Phase == "started"
		m := MemberObservation{InstanceID: s.Storage.InstanceID, ResourceObservation: ResourceObservation{Resource: s.Container, Present: true}, AnchorID: s.Anchor.Prepared.Anchor.ID, Running: running}
		if running {
			m.Namespace = *s.Anchor.Namespace
		}
		o.Members = append(o.Members, m)
	}
	switch d.fault {
	case "anchor-changed":
		o.Anchor.Namespace.Inode++
	case "old-peer-namespace":
		o.Members[0].Namespace.Inode++
	case "unknown-peer":
		o.Members[0].InstanceID = "unknown"
	case "incomplete":
		o.Complete = false
	case "stale":
		o.ObservedAt = o.ObservedAt.Add(-time.Minute)
	case "wrong-member-contract":
		if len(o.Members) > 1 {
			o.Members[1].Resource.ContractDigest = strings.Repeat("f", 64)
		}
	case "member-not-running":
		if s.Phase == "started" {
			o.Members[1].Running = false
			o.Members[1].Namespace = NamespaceIdentity{}
		}
	}
	return o, nil
}
func TestMemberLaunchWriteAheadAndUncertainty(t *testing.T) {
	for fault, wantPhase := range map[string]string{
		"success": "running", "workspace-no-id": "workspace-dispatched", "workspace-late-error": "workspace-created", "preflight": "workspace-created", "preflight-created": "created", "create-no-id": "create-dispatched", "create-late-error": "created", "cancel-after-create": "created", "lost-start-reply": "start-dispatched", "cancel-after-start": "started", "anchor-changed": "planned", "old-peer-namespace": "planned", "unknown-peer": "planned", "incomplete": "planned", "stale": "planned", "wrong-member-contract": "created", "member-not-running": "started",
		"commit-workspace-dispatched": "planned", "commit-workspace-created": "workspace-dispatched", "commit-create-dispatched": "workspace-created", "commit-created": "create-dispatched", "commit-start-dispatched": "created", "commit-started": "start-dispatched", "commit-running": "started",
	} {
		t.Run(fault, func(t *testing.T) {
			plan, s, o := memberLaunchFixture(t)
			originalAnchor := s.Anchor
			originalStorage := s.Storage
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &memberLaunchDriverFixture{t: t, state: &s, base: o, fault: fault, cancel: cancel}
			commit := func(expected, phase string, r MemberLaunchResult) error {
				if fault == "commit-"+phase {
					return errors.New("synthetic durable write failure")
				}
				next, err := s.Advance(plan, expected, phase, r)
				if err == nil {
					s = next
				}
				if r.Namespace != nil {
					r.Namespace.Inode++
				} // callback storage cannot alias controller state.
				return err
			}
			got, err := ExecuteMemberLaunch(ctx, plan, s, d, commit, func() time.Time { return o.ObservedAt })
			if (err == nil) != (fault == "success") || !reflect.DeepEqual(got, s) || s.Phase != wantPhase {
				t.Fatalf("phase=%s want=%s error=%v", s.Phase, wantPhase, err)
			}
			if err != nil && strings.Contains(err.Error(), "synthetic provider detail") {
				t.Fatal("provider error leaked")
			}
			if !reflect.DeepEqual(originalAnchor, s.Anchor) || originalStorage != s.Storage {
				t.Fatal("launch rewrote earlier receipt")
			}
			if wantPhase == "planned" && (d.prepares+d.creates+d.starts) != 0 {
				t.Fatal("failed preflight/commit caused effects")
			}
			if wantPhase == "workspace-created" && (d.creates+d.starts) != 0 {
				t.Fatal("container dispatched without gate")
			}
			if wantPhase == "created" && d.starts != 0 {
				t.Fatal("Start dispatched after failed gate")
			}
			if fault == "success" && (d.prepares != 1 || d.creates != 1 || d.starts != 1 || s.Namespace == nil) {
				t.Fatal("incorrect dispatch counts")
			}
			if s.Phase != "planned" {
				calls := d.prepares + d.creates + d.starts + d.inspections + d.verifies
				if _, err = ExecuteMemberLaunch(context.Background(), plan, s, d, commit, time.Now); err == nil || calls != d.prepares+d.creates+d.starts+d.inspections+d.verifies {
					t.Fatal("receipt replayed effects")
				}
			}
		})
	}
}
func TestMemberLaunchRejectsCrossProjectAndDuplicateOwnership(t *testing.T) {
	for name, mutate := range map[string]func(*MemberLaunch){
		"project":              func(s *MemberLaunch) { s.Storage.ProjectID = "other" },
		"revision":             func(s *MemberLaunch) { s.Storage.ProjectRevision++ },
		"operation":            func(s *MemberLaunch) { s.OperationID = s.Storage.OperationID },
		"unobserved-storage":   func(s *MemberLaunch) { s.Storage.Phase = "created" },
		"unobserved-anchor":    func(s *MemberLaunch) { s.Anchor.Phase = "started"; s.Anchor.Namespace = nil },
		"target-peer":          func(s *MemberLaunch) { s.Peers[0].InstanceID = s.Storage.InstanceID },
		"peer-runtime":         func(s *MemberLaunch) { s.Peers[0].RuntimeID = s.Storage.RuntimeID },
		"peer-anchor":          func(s *MemberLaunch) { s.Peers[0].Container.ID = s.Anchor.Prepared.Anchor.ID },
		"unrecorded-workspace": func(s *MemberLaunch) { s.Workspace = pierstore.DirectoryIdentity{Device: 1, Inode: 2, Birth: "3:4"} },
		"unrecorded-container": func(s *MemberLaunch) { s.Container.ID = strings.Repeat("c", 64) },
		"unobserved-namespace": func(s *MemberLaunch) { v := *s.Anchor.Namespace; s.Namespace = &v },
	} {
		t.Run(name, func(t *testing.T) {
			plan, s, _ := memberLaunchFixture(t)
			mutate(&s)
			if s.Validate(plan) == nil {
				t.Fatal("invalid launch admitted")
			}
		})
	}
	plan, s, _ := memberLaunchFixture(t)
	if _, err := s.Advance(plan, "planned", "created", MemberLaunchResult{ContainerID: strings.Repeat("c", 64)}); err == nil {
		t.Fatal("skipped intent")
	}
	copy, err := NewMemberLaunch(plan, s.Anchor, s.Storage, s.OperationID, s.Container.ContractDigest, s.Peers)
	if err != nil {
		t.Fatal(err)
	}
	s.Anchor.Namespace.Inode++
	s.Peers[0].InstanceID = "changed"
	if copy.Validate(plan) != nil || copy.Anchor.Namespace.Inode == s.Anchor.Namespace.Inode || copy.Peers[0].InstanceID == s.Peers[0].InstanceID {
		t.Fatal("constructor retained mutable aliases")
	}
}
