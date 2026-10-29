package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// AnchorReplacement retains serial creation/Start receipts after complete old
// container retirement. Running containers alone do not mean recovered apps:
// fresh identity/readiness evidence is still required before releasing the fence.
type AnchorReplacement struct {
	OperationID string            `json:"operation_id"`
	Sequence    uint64            `json:"sequence"`
	Retirement  AnchorRetirement  `json:"retirement"`
	Members     []RecoveredMember `json:"members"`
}
type RecoveredMember struct {
	Creation          RecoveryMemberCreation       `json:"creation"`
	Container         ResourceIdentity             `json:"container"`
	Phase             string                       `json:"phase"`
	StartAcknowledged bool                         `json:"start_acknowledged"`
	Before            *AnchorRetirementObservation `json:"before,omitempty"`
	Created           *AnchorRetirementObservation `json:"created,omitempty"`
	After             *AnchorRetirementObservation `json:"after,omitempty"`
}

func (s AnchorReplacement) Clone() AnchorReplacement {
	s.Retirement = s.Retirement.Clone()
	s.Members = slices.Clone(s.Members)
	for i := range s.Members {
		m := &s.Members[i]
		m.Creation = m.Creation.Clone()
		for _, p := range []**AnchorRetirementObservation{&m.Before, &m.Created, &m.After} {
			if *p != nil {
				v := cloneAnchorRetirementObservation(**p)
				*p = &v
			}
		}
	}
	return s
}
func NewAnchorReplacement(h AnchorRecoveryHistory, r AnchorRetirement, operation string, members []RecoveredMember) (AnchorReplacement, error) {
	s := (AnchorReplacement{OperationID: operation, Retirement: r, Members: members}).Clone()
	for i := range s.Members {
		if s.Members[i].Phase != "planned" {
			return AnchorReplacement{}, errors.New("replacement requires fresh member contracts")
		}
	}
	return s, s.Validate(h)
}
func (s AnchorReplacement) next() int {
	for i, m := range s.Members {
		if m.Phase != "running" {
			return i
		}
	}
	return -1
}
func (s AnchorReplacement) Running() bool { return len(s.Members) > 0 && s.next() < 0 }
func (s AnchorReplacement) checkObservation(index int, o AnchorRetirementObservation, stage string, now time.Time) error {
	fail := func() error { return errors.New("replacement lost exact successor, peer process or pier evidence") }
	if index < 0 || index >= len(s.Members) || len(s.Retirement.Members) == 0 {
		return fail()
	}
	base := s.Retirement.Members[len(s.Retirement.Members)-1].After
	if index > 0 {
		base = s.Members[index-1].After
	}
	if base == nil || !freshNamespaceObservation(AnchorDrainObservation{Runtime: o.Runtime, CapturedAt: o.CapturedAt}, now, base.CapturedAt) {
		return fail()
	}
	expected := cloneAnchorRetirementObservation(*base)
	expected.CapturedAt = o.CapturedAt
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	m := s.Members[index]
	if stage != "absent" {
		if stage != "created" && stage != "running" || m.Before == nil || o.CapturedAt.Before(m.Before.CapturedAt) || o.Runtime.ObservedAt.Before(m.Before.Runtime.ObservedAt) {
			return fail()
		}
		running := stage == "running"
		state := RecoveryProcessState{}
		if running {
			if m.Created == nil || o.CapturedAt.Before(m.Created.CapturedAt) || o.Runtime.ObservedAt.Before(m.Created.Runtime.ObservedAt) {
				return fail()
			}
			found := false
			for _, p := range o.Processes {
				if p.InstanceID == m.Creation.InstanceID {
					if found {
						return fail()
					}
					found = true
					state = p.Process
				}
			}
			if !found || !recoveryProcessValid(state, true, o.CapturedAt) || state.StartedAt.Before(m.Created.CapturedAt.Add(-5*time.Second)) {
				return fail()
			}
		}
		member := MemberObservation{InstanceID: m.Creation.InstanceID, ResourceObservation: ResourceObservation{Resource: m.Container, Present: true}, Running: running, AnchorID: expected.Runtime.Anchor.Resource.ID}
		if running {
			member.Namespace = m.Creation.Namespace
		}
		expected.Runtime.Members = append(expected.Runtime.Members, member)
		expected.Processes = append(expected.Processes, AnchorDrainProcess{InstanceID: m.Creation.InstanceID, Storage: m.Creation.Storage, Process: state})
	}
	if !equalEvidence(sortedRetirementObservation(expected), sortedRetirementObservation(o)) {
		return fail()
	}
	return nil
}
func (s AnchorReplacement) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid coordinated replacement intent") }
	r := s.Retirement
	if r.Validate(h) != nil || !r.Complete() || !protocol.ValidID(s.OperationID) || recoveryOperationExists(h, s.OperationID) || s.OperationID == r.OperationID || s.OperationID == r.Namespace.OperationID || s.OperationID == r.Namespace.Drain.OperationID || len(s.Members) != len(r.Members) {
		return fail()
	}
	usedOps := map[string]bool{s.OperationID: true}
	usedIDs := map[string]bool{r.Namespace.After.Runtime.Anchor.Resource.ID: true, r.Namespace.After.Runtime.Bridge.Resource.ID: true}
	for id := range recoveryHistoryContainerIDs(h) {
		usedIDs[id] = true
	}
	pending := false
	var sequence uint64
	for i, m := range s.Members {
		expected, err := recoveryMemberCreationFromValidatedRetirement(h, r, m.Creation.InstanceID, m.Creation.OperationID)
		if m.Creation.InstanceID != r.Members[i].InstanceID || err != nil || !equalEvidence(m.Creation, expected) || usedOps[m.Creation.OperationID] || !digestPattern.MatchString(m.Container.ContractDigest) {
			return fail()
		}
		usedOps[m.Creation.OperationID] = true
		n, ok := map[string]uint64{"planned": 0, "create-dispatched": 1, "created": 2, "start-dispatched": 3, "start-acknowledged": 4, "running": 4}[m.Phase]
		if !ok || pending && n > 0 {
			return fail()
		}
		if n == 0 {
			if m.Before != nil || m.Created != nil || m.After != nil || m.Container.ID != "" || m.StartAcknowledged {
				return fail()
			}
			pending = true
			continue
		}
		if m.Before == nil || s.checkObservation(i, *m.Before, "absent", m.Before.CapturedAt) != nil {
			return fail()
		}
		if n < 2 {
			if m.Container.ID != "" {
				return fail()
			}
		} else {
			if !m.Container.valid() || usedIDs[m.Container.ID] {
				return fail()
			}
			usedIDs[m.Container.ID] = true
		}
		if n < 3 {
			if m.Created != nil {
				return fail()
			}
		} else {
			if m.Created == nil || s.checkObservation(i, *m.Created, "created", m.Created.CapturedAt) != nil {
				return fail()
			}
		}
		if m.Phase == "running" {
			if m.After == nil || s.checkObservation(i, *m.After, "running", m.After.CapturedAt) != nil {
				return fail()
			}
			if m.StartAcknowledged {
				n++
			}
		} else {
			if m.After != nil {
				return fail()
			}
			pending = true
		}
		if m.Phase == "start-acknowledged" && !m.StartAcknowledged || m.Phase != "start-acknowledged" && m.Phase != "running" && m.StartAcknowledged {
			return fail()
		}
		sequence += n
	}
	if sequence != s.Sequence {
		return fail()
	}
	return nil
}

type AnchorReplacementResult struct {
	ContainerID string
	Observation *AnchorRetirementObservation
}

func (s AnchorReplacement) Advance(h AnchorRecoveryHistory, index int, sequence uint64, phase string, result AnchorReplacementResult) (AnchorReplacement, error) {
	fail := func() (AnchorReplacement, error) {
		return AnchorReplacement{}, errors.New("stale or invalid coordinated replacement transition")
	}
	if s.Validate(h) != nil || sequence != s.Sequence || index != s.next() || index < 0 {
		return fail()
	}
	next := s.Clone()
	m := &next.Members[index]
	old := m.Phase
	switch {
	case old == "planned" && phase == "create-dispatched":
		if result.Observation == nil || result.ContainerID != "" {
			return fail()
		}
		v := cloneAnchorRetirementObservation(*result.Observation)
		m.Before = &v
	case old == "create-dispatched" && phase == "created":
		if result.Observation != nil || !digestPattern.MatchString(result.ContainerID) {
			return fail()
		}
		m.Container.ID = result.ContainerID
	case old == "created" && phase == "start-dispatched":
		if result.Observation == nil || result.ContainerID != "" {
			return fail()
		}
		v := cloneAnchorRetirementObservation(*result.Observation)
		m.Created = &v
	case old == "start-dispatched" && phase == "start-acknowledged":
		if result.Observation != nil || result.ContainerID != "" {
			return fail()
		}
		m.StartAcknowledged = true
	case (old == "start-dispatched" || old == "start-acknowledged") && phase == "running":
		if result.Observation == nil || result.ContainerID != "" {
			return fail()
		}
		v := cloneAnchorRetirementObservation(*result.Observation)
		m.After = &v
	default:
		return fail()
	}
	m.Phase = phase
	next.Sequence++
	return next, next.Validate(h)
}
func (s AnchorReplacement) ValidateObservation(h AnchorRecoveryHistory, index int, o AnchorRetirementObservation, stage string, now time.Time) error {
	if err := s.Validate(h); err != nil {
		return err
	}
	return s.checkObservation(index, o, stage, now)
}

// Drivers verify immutable contracts, all retained piers, mount exclusivity and
// exact process inventories at each dispatch. No method may bootstrap or replay.
type AnchorReplacementObserver interface {
	VerifyEngine(context.Context, string) error
	InspectReplacement(context.Context, AnchorReplacement, int, string) (AnchorRetirementObservation, error)
}
type AnchorReplacementStartDriver interface {
	AnchorReplacementObserver
	StartReplacement(context.Context, AnchorReplacement, int) error
}
type AnchorReplacementDriver interface {
	AnchorReplacementStartDriver
	CreateReplacement(context.Context, AnchorReplacement, int) (string, error)
}
type AnchorReplacementCommit func(uint64, AnchorReplacement) error

func ExecuteNextAnchorReplacement(ctx context.Context, h AnchorRecoveryHistory, s AnchorReplacement, d AnchorReplacementDriver, commit AnchorReplacementCommit, now func() time.Time) (AnchorReplacement, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("replacement requires retained intent and durable dispatch")
	}
	h = h.Clone()
	s = s.Clone()
	index := s.next()
	if index < 0 || s.Members[index].Phase != "planned" {
		return s, errors.New("replacement has no fresh creation intent")
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Retirement.Namespace.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	advance := func(phase string, result AnchorReplacementResult) error {
		next, err := s.Advance(h, index, s.Sequence, phase, result)
		if err != nil {
			return err
		}
		if err = commit(s.Sequence, next.Clone()); err != nil {
			return err
		}
		s = next
		return nil
	}
	inspect := func(stage string) (AnchorRetirementObservation, error) {
		if !verify() {
			return AnchorRetirementObservation{}, fail("replacement engine unavailable")
		}
		o, err := d.InspectReplacement(ctx, s.Clone(), index, stage)
		o = cloneAnchorRetirementObservation(o)
		if err != nil || !verify() || s.checkObservation(index, o, stage, now()) != nil {
			return AnchorRetirementObservation{}, fail("replacement observation refused")
		}
		return o, nil
	}
	before, err := inspect("absent")
	if err != nil {
		return s, err
	}
	if err = advance("create-dispatched", AnchorReplacementResult{Observation: &before}); err != nil {
		return s, err
	}
	if !verify() || s.checkObservation(index, *s.Members[index].Before, "absent", now()) != nil {
		return s, fail("replacement Create preflight expired")
	}
	id, createErr := d.CreateReplacement(ctx, s.Clone(), index)
	if id != "" {
		if err = advance("created", AnchorReplacementResult{ContainerID: id}); err != nil {
			return s, err
		}
	}
	if createErr != nil || ctx.Err() != nil || s.Members[index].Phase != "created" {
		return s, fail("replacement Create unresolved; retain intent and returned ID")
	}
	return StartCreatedAnchorReplacement(ctx, h, s, d, commit, now)
}

// Pending Start is reconciled only against the recorded container. Missing IDs
// from uncertain Create are never inferred from names or followed by another Create.
func ObserveAnchorReplacementRunning(ctx context.Context, h AnchorRecoveryHistory, s AnchorReplacement, d AnchorReplacementObserver, commit AnchorReplacementCommit, now func() time.Time) (AnchorReplacement, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("replacement observation requires retained intent")
	}
	h = h.Clone()
	s = s.Clone()
	index := s.next()
	if index < 0 || s.Members[index].Phase != "start-dispatched" && s.Members[index].Phase != "start-acknowledged" {
		return s, errors.New("replacement has no pending Start to observe")
	}
	fail := func() (AnchorReplacement, error) {
		return s, errors.Join(errors.New("replacement running outcome unresolved"), ctx.Err())
	}
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Retirement.Namespace.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return fail()
	}
	o, err := d.InspectReplacement(ctx, s.Clone(), index, "running")
	o = cloneAnchorRetirementObservation(o)
	if err != nil || !verify() || s.checkObservation(index, o, "running", now()) != nil {
		return fail()
	}
	next, err := s.Advance(h, index, s.Sequence, "running", AnchorReplacementResult{Observation: &o})
	if err != nil {
		return fail()
	}
	if err = commit(s.Sequence, next.Clone()); err != nil {
		return s, err
	}
	return next, nil
}
