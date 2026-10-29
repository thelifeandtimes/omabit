package projectnetwork

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// InitialRuntimeBinding reconstructs the complete original generation from the
// immutable serial launch receipts. It conveys no current runtime authority.
func InitialRuntimeBinding(plan Plan, launches []MemberLaunch) (RuntimeBinding, error) {
	fail := func() (RuntimeBinding, error) {
		return RuntimeBinding{}, errors.New("complete consistent initial member launch receipts are required")
	}
	if len(launches) != len(plan.Members) || len(launches) == 0 {
		return fail()
	}
	ordered := slices.Clone(launches)
	slices.SortFunc(ordered, func(a, b MemberLaunch) int { return len(a.Peers) - len(b.Peers) })
	anchor := ordered[0].Anchor
	var previous []RuntimeMember
	operations := map[string]bool{anchor.OperationID: true, anchor.Prepared.OperationID: true}
	for i, s := range ordered {
		if s.Validate(plan) != nil || s.Phase != "running" || !reflect.DeepEqual(s.Anchor, anchor) || len(s.Peers) != i {
			return fail()
		}
		if operations[s.OperationID] || operations[s.Storage.OperationID] || s.OperationID == s.Storage.OperationID {
			return fail()
		}
		operations[s.OperationID], operations[s.Storage.OperationID] = true, true
		for k, peer := range previous {
			if s.Peers[k] != peer {
				return fail()
			}
		}
		previous = append(previous, RuntimeMember{s.Storage.InstanceID, s.Storage.RuntimeID, s.Container})
		slices.SortFunc(previous, func(a, b RuntimeMember) int {
			if a.InstanceID < b.InstanceID {
				return -1
			}
			if a.InstanceID > b.InstanceID {
				return 1
			}
			return 0
		})
	}
	p := anchor.Prepared
	b := RuntimeBinding{ProjectID: p.ProjectID, Revision: p.ProjectRevision, Generation: p.Generation, PlanDigest: p.PlanDigest, EngineID: p.EngineID, Bridge: ResourceIdentity{p.Bridge.ID, p.Bridge.ContractDigest}, Anchor: ResourceIdentity{p.Anchor.ID, p.Anchor.ContractDigest}, Namespace: *anchor.Namespace, Members: previous}
	if b.Validate(plan) != nil {
		return fail()
	}
	return b, nil
}

// InitialMemberReadiness is historical evidence, not cached permission to mutate
// a pier or a promise that a ship is still online. Original storage and launch
// receipts remain immutable. Persistence and runtime collection are separate.
type InitialMemberReadiness struct {
	InstanceID        string             `json:"instance_id"`
	LaunchOperationID string             `json:"launch_operation_id"`
	Binding           RuntimeBinding     `json:"binding"`
	Storage           pierstore.Location `json:"storage"`
	Ship              string             `json:"ship"`
	Endpoint          string             `json:"endpoint"`
	Before            Observation        `json:"before"`
	After             Observation        `json:"after"`
	CapturedAt        time.Time          `json:"captured_at"`
}

func initialMember(plan Plan, launches []MemberLaunch, instance string) (MemberLaunch, Member, error) {
	for _, s := range launches {
		if s.Storage.InstanceID == instance {
			for _, m := range plan.Members {
				if m.InstanceID == instance {
					return s, m, nil
				}
			}
		}
	}
	return MemberLaunch{}, Member{}, errors.New("member is not in the original launch set")
}
func readinessStorage(s MemberLaunch, location pierstore.Location) bool {
	original := s.Storage.Location
	id := location.Identity
	return location.Path == original.Path && location.MountIdentity == original.MountIdentity && !location.Uncertain && id.Device != 0 && id.Inode != 0 && id.Birth != "" && id.UrbDevice != 0 && id.UrbInode != 0 && id.UrbBirth != "" && (id.Device != id.UrbDevice || id.Inode != id.UrbInode) && (id.Device != location.MountIdentity.Device || id.Inode != location.MountIdentity.Inode) && (id.UrbDevice != location.MountIdentity.Device || id.UrbInode != location.MountIdentity.Inode)
}
func cloneReadinessObservation(o Observation) Observation {
	o.BridgeAttachments = slices.Clone(o.BridgeAttachments)
	o.Members = slices.Clone(o.Members)
	o.Anchor.Requested = slices.Clone(o.Anchor.Requested)
	o.Anchor.Active = slices.Clone(o.Anchor.Active)
	return o
}
func (r InitialMemberReadiness) Validate(plan Plan, launches []MemberLaunch) error {
	fail := func() error { return errors.New("initial member readiness evidence is incomplete or changed") }
	b, err := InitialRuntimeBinding(plan, launches)
	if err != nil || !reflect.DeepEqual(b, r.Binding) || r.CapturedAt.IsZero() {
		return fail()
	}
	s, m, err := initialMember(plan, launches, r.InstanceID)
	if err != nil || r.LaunchOperationID != s.OperationID || r.Ship != m.Ship || r.Endpoint != fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort) || !readinessStorage(s, r.Storage) {
		return fail()
	}
	for _, o := range []Observation{r.Before, r.After} {
		a, err := AssessRuntime(plan, b, o, r.CapturedAt)
		if err != nil || a.State != "ready" || a.RecoveryRequired || len(a.Running) != len(plan.Members) || len(a.Stopped) != 0 {
			return fail()
		}
	}
	before, after := r.Before, r.After
	if after.ObservedAt.Before(before.ObservedAt) {
		return fail()
	}
	before.ObservedAt = after.ObservedAt
	if !reflect.DeepEqual(before, after) {
		return fail()
	}
	return nil
}

type InitialReadinessDriver interface {
	VerifyEngine(context.Context, string) error
	InspectProject(context.Context) (Observation, error)
	InspectPier(context.Context, MemberLaunch) (pierstore.Location, error)
	ReadHostIdentity(context.Context, string) (string, error)
}

// CollectInitialMemberReadiness performs one read-only attempt while the caller
// holds the common project fence. An unavailable HTTP endpoint can be retried by
// a bounded higher-level operation, but no effect or launch is retried here.
// Persist only the complete returned evidence; failure returns no partial proof.
func CollectInitialMemberReadiness(ctx context.Context, plan Plan, launches []MemberLaunch, instance string, d InitialReadinessDriver, now func() time.Time) (InitialMemberReadiness, error) {
	refuse := func() (InitialMemberReadiness, error) {
		return InitialMemberReadiness{}, errors.New("initial member readiness could not be established")
	}
	b, err := InitialRuntimeBinding(plan, launches)
	if err != nil || d == nil || now == nil {
		return refuse()
	}
	s, m, err := initialMember(plan, launches, instance)
	if err != nil {
		return refuse()
	}
	copyLaunch := func() MemberLaunch {
		v := s
		v.Peers = slices.Clone(s.Peers)
		a := *s.Anchor.Namespace
		v.Anchor.Namespace = &a
		n := *s.Namespace
		v.Namespace = &n
		return v
	}
	verify := func() bool { return ctx.Err() == nil && d.VerifyEngine(ctx, b.EngineID) == nil && ctx.Err() == nil }
	if !verify() {
		return refuse()
	}
	before, err := d.InspectProject(ctx)
	before = cloneReadinessObservation(before)
	if err != nil || !verify() {
		return refuse()
	}
	assessment, err := AssessRuntime(plan, b, before, now())
	if err != nil || assessment.State != "ready" || assessment.RecoveryRequired || len(assessment.Running) != len(plan.Members) || len(assessment.Stopped) != 0 {
		return refuse()
	}
	first, err := d.InspectPier(ctx, copyLaunch())
	if err != nil || !readinessStorage(s, first) || !verify() {
		return refuse()
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
	ship, err := d.ReadHostIdentity(ctx, endpoint)
	if err != nil || ship != m.Ship || !verify() {
		return refuse()
	}
	second, err := d.InspectPier(ctx, copyLaunch())
	if err != nil || first != second || !verify() {
		return refuse()
	}
	after, err := d.InspectProject(ctx)
	after = cloneReadinessObservation(after)
	if err != nil || !verify() {
		return refuse()
	}
	r := InitialMemberReadiness{InstanceID: instance, LaunchOperationID: s.OperationID, Binding: b, Storage: first, Ship: ship, Endpoint: endpoint, Before: before, After: after, CapturedAt: now()}
	if r.Validate(plan, launches) != nil || ctx.Err() != nil {
		return refuse()
	}
	return r, nil
}
