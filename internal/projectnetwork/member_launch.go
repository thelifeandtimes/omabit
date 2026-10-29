package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberLaunch is initial container execution only. Running is not Urbit
// readiness, initialized storage, or permission to resume/recreate a pier.
// The original empty-storage and anchor-Start receipts remain immutable.
// Schema 27 persists this model; it is not exposed as an RPC capability.
type MemberLaunch struct {
	Anchor      AnchorStart                 `json:"anchor"`
	Storage     MemberStorage               `json:"storage"`
	OperationID string                      `json:"operation_id"`
	Phase       string                      `json:"phase"`
	Workspace   pierstore.DirectoryIdentity `json:"workspace"`
	Container   ResourceIdentity            `json:"container"`
	Peers       []RuntimeMember             `json:"peers"`
	Namespace   *NamespaceIdentity          `json:"namespace,omitempty"`
}

func NewMemberLaunch(plan Plan, anchor AnchorStart, storage MemberStorage, operation, contract string, peers []RuntimeMember) (MemberLaunch, error) {
	s := MemberLaunch{Anchor: anchor, Storage: storage, OperationID: operation, Phase: "planned", Container: ResourceIdentity{ContractDigest: contract}, Peers: slices.Clone(peers)}
	if anchor.Namespace != nil {
		v := *anchor.Namespace
		s.Anchor.Namespace = &v
	}
	return s, s.Validate(plan)
}
func validMemberDirectory(v pierstore.DirectoryIdentity) bool {
	return v.Device != 0 && v.Inode != 0 && v.Birth != ""
}
func (s MemberLaunch) Validate(plan Plan) error {
	fail := func() error { return errors.New("invalid initial member launch binding") }
	p := s.Anchor.Prepared
	if s.Anchor.Validate(plan) != nil || s.Anchor.Phase != "observed" || s.Storage.Validate(plan) != nil || s.Storage.Phase != "observed" || s.Storage.ProjectID != p.ProjectID || s.Storage.ProjectRevision != p.ProjectRevision || s.Storage.PlanDigest != p.PlanDigest || !protocol.ValidID(s.OperationID) || s.OperationID == s.Storage.OperationID || s.OperationID == s.Anchor.OperationID || s.OperationID == p.OperationID || !digestPattern.MatchString(s.Container.ContractDigest) {
		return fail()
	}
	ids := map[string]bool{p.Anchor.ID: true, p.Bridge.ID: true}
	instances := map[string]bool{s.Storage.InstanceID: true}
	runtimes := map[string]bool{s.Storage.RuntimeID: true}
	for i, m := range s.Peers {
		found := false
		for _, planned := range plan.Members {
			if planned.InstanceID == m.InstanceID {
				found = true
			}
		}
		if !found || !m.Container.valid() || !runtimePattern.MatchString(m.RuntimeID) || ids[m.Container.ID] || instances[m.InstanceID] || runtimes[m.RuntimeID] || i > 0 && s.Peers[i-1].InstanceID >= m.InstanceID {
			return fail()
		}
		ids[m.Container.ID], instances[m.InstanceID], runtimes[m.RuntimeID] = true, true, true
	}
	phases := map[string]int{"planned": 0, "workspace-dispatched": 1, "workspace-created": 2, "create-dispatched": 3, "created": 4, "start-dispatched": 5, "started": 6, "running": 7}
	stage, ok := phases[s.Phase]
	if !ok {
		return fail()
	}
	if stage < 2 {
		if s.Workspace != (pierstore.DirectoryIdentity{}) {
			return fail()
		}
	} else if !validMemberDirectory(s.Workspace) {
		return fail()
	}
	if stage < 4 {
		if s.Container.ID != "" {
			return fail()
		}
	} else if !s.Container.valid() || ids[s.Container.ID] {
		return fail()
	}
	if stage < 7 {
		if s.Namespace != nil {
			return fail()
		}
	} else if s.Namespace == nil || *s.Namespace != *s.Anchor.Namespace {
		return fail()
	}
	return nil
}

type MemberLaunchResult struct {
	Workspace   pierstore.DirectoryIdentity
	ContainerID string
	Namespace   *NamespaceIdentity
}

func (s MemberLaunch) Advance(plan Plan, expected, phase string, result MemberLaunchResult) (MemberLaunch, error) {
	refuse := func() (MemberLaunch, error) {
		return MemberLaunch{}, errors.New("stale or invalid member launch transition")
	}
	next := map[string]string{"planned": "workspace-dispatched", "workspace-dispatched": "workspace-created", "workspace-created": "create-dispatched", "create-dispatched": "created", "created": "start-dispatched", "start-dispatched": "started", "started": "running"}
	if s.Validate(plan) != nil || s.Phase != expected || next[expected] != phase {
		return refuse()
	}
	switch phase {
	case "workspace-created":
		if !validMemberDirectory(result.Workspace) || result.ContainerID != "" || result.Namespace != nil {
			return refuse()
		}
		s.Workspace = result.Workspace
	case "created":
		if !digestPattern.MatchString(result.ContainerID) || result.Workspace != (pierstore.DirectoryIdentity{}) || result.Namespace != nil {
			return refuse()
		}
		s.Container.ID = result.ContainerID
	case "running":
		if result.Namespace == nil || result.Workspace != (pierstore.DirectoryIdentity{}) || result.ContainerID != "" {
			return refuse()
		}
		ns := *result.Namespace
		s.Namespace = &ns
	default:
		if result.Workspace != (pierstore.DirectoryIdentity{}) || result.ContainerID != "" || result.Namespace != nil {
			return refuse()
		}
	}
	s.Phase = phase
	return s, s.Validate(plan)
}

// Driver methods must bind the fixed command/image/storage/pill contract and
// exact recorded peers. Preflight rejects unresolved filesystem use and requires
// the original empty parent, verified pinned pill and unchanged fresh namespace.
// Inspect checks full container contracts and a complete bounded inventory.
// No method may adopt by name, remove anything, or retry an uncertain mutation.
type MemberLaunchDriver interface {
	VerifyEngine(context.Context, string) error
	PrepareWorkspace(context.Context, MemberLaunch) (pierstore.DirectoryIdentity, error)
	Preflight(context.Context, MemberLaunch) error
	Inspect(context.Context, MemberLaunch) (Observation, error)
	Create(context.Context, MemberLaunch) (string, error)
	Start(context.Context, MemberLaunch) error
}
type MemberLaunchCommit func(expected, phase string, result MemberLaunchResult) error

func (s MemberLaunch) inspect(plan Plan, o Observation, now time.Time, present, running bool) error {
	anchorOnly := o
	anchorOnly.Members = nil
	if s.Anchor.inspect(plan, anchorOnly, now, true) != nil || o.Anchor.Namespace != *s.Anchor.Namespace {
		return ErrRuntimeObservation
	}
	expected := map[string]ResourceIdentity{}
	for _, m := range s.Peers {
		expected[m.InstanceID] = m.Container
	}
	if present {
		expected[s.Storage.InstanceID] = s.Container
	}
	if len(o.Members) != len(expected) {
		return ErrRuntimeObservation
	}
	for _, m := range o.Members {
		want, ok := expected[m.InstanceID]
		if !ok || !m.Present || !exactResource(want, m.ResourceObservation) || m.AnchorID != s.Anchor.Prepared.Anchor.ID {
			return ErrRuntimeObservation
		}
		if m.Running {
			if m.Namespace != *s.Anchor.Namespace {
				return ErrRuntimeObservation
			}
		} else if m.Namespace != (NamespaceIdentity{}) {
			return ErrRuntimeObservation
		}
		if m.InstanceID == s.Storage.InstanceID && m.Running != running {
			return ErrRuntimeObservation
		}
		delete(expected, m.InstanceID)
	}
	return nil
}

// ExecuteMemberLaunch requires the common runner mutation/update fence and a
// durable compare-and-commit callback. Returned identities survive late errors.
// An uncertain receipt is never automatically resumed, replayed or cleaned up.
func ExecuteMemberLaunch(ctx context.Context, plan Plan, s MemberLaunch, d MemberLaunchDriver, commit MemberLaunchCommit, now func() time.Time) (MemberLaunch, error) {
	if s.Validate(plan) != nil || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, errors.New("member launch requires fresh intent and durable dispatch")
	}
	// Do not share mutable slices/pointers with adapters or commit callbacks.
	copyState := func() MemberLaunch {
		v := s
		v.Peers = slices.Clone(s.Peers)
		ns := *s.Anchor.Namespace
		v.Anchor.Namespace = &ns
		if s.Namespace != nil {
			n := *s.Namespace
			v.Namespace = &n
		}
		return v
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.VerifyEngine(ctx, s.Anchor.Prepared.EngineID) != nil {
			return fail("member launch engine identity could not be verified")
		}
		return ctx.Err()
	}
	advance := func(phase string, r MemberLaunchResult) error {
		next, err := s.Advance(plan, s.Phase, phase, r)
		if err != nil {
			return err
		}
		if r.Namespace != nil {
			ns := *r.Namespace
			r.Namespace = &ns
		}
		if err = commit(s.Phase, phase, r); err != nil {
			return err
		}
		s = next
		return nil
	}
	inspect := func(present, running bool) error {
		if err := verify(); err != nil {
			return err
		}
		o, err := d.Inspect(ctx, copyState())
		if err != nil {
			return fail("member launch ownership observation failed")
		}
		if err = verify(); err != nil {
			return err
		}
		return s.inspect(plan, o, now(), present, running)
	}
	preflight := func() error {
		if err := verify(); err != nil {
			return err
		}
		if d.Preflight(ctx, copyState()) != nil {
			return fail("member launch storage or artifact preflight failed")
		}
		return verify()
	}
	if err := inspect(false, false); err != nil {
		return s, err
	}
	if err := advance("workspace-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if err := verify(); err != nil {
		return s, err
	}
	workspace, prepareErr := d.PrepareWorkspace(ctx, copyState())
	if workspace != (pierstore.DirectoryIdentity{}) {
		if err := advance("workspace-created", MemberLaunchResult{Workspace: workspace}); err != nil {
			return s, err
		}
	}
	if prepareErr != nil {
		return s, fail("member workspace preparation is unresolved; retain its receipt")
	}
	if s.Phase != "workspace-created" {
		return s, fail("member workspace returned no exact identity")
	}
	if err := preflight(); err != nil {
		return s, err
	}
	if err := inspect(false, false); err != nil {
		return s, err
	}
	if err := advance("create-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if err := verify(); err != nil {
		return s, err
	}
	id, createErr := d.Create(ctx, copyState())
	if id != "" {
		if err := advance("created", MemberLaunchResult{ContainerID: id}); err != nil {
			return s, err
		}
	}
	if createErr != nil {
		return s, fail("member container creation is unresolved; retain its receipt")
	}
	if s.Phase != "created" {
		return s, fail("member creation returned no exact container ID")
	}
	if err := preflight(); err != nil {
		return s, err
	}
	if err := inspect(true, false); err != nil {
		return s, err
	}
	if err := advance("start-dispatched", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if err := verify(); err != nil {
		return s, err
	}
	if d.Start(ctx, copyState()) != nil {
		return s, fail("member Start outcome is unresolved; retain its receipt")
	}
	if err := advance("started", MemberLaunchResult{}); err != nil {
		return s, err
	}
	if err := inspect(true, true); err != nil {
		return s, err
	}
	if err := advance("running", MemberLaunchResult{Namespace: s.Anchor.Namespace}); err != nil {
		return s, err
	}
	return s, nil
}
