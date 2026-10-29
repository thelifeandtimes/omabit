package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// RecoveryPier remains observable after its old container is absent. Process
// evidence, in contrast, must contain only containers that are still present.
type RecoveryPier struct {
	InstanceID string             `json:"instance_id"`
	Storage    pierstore.Location `json:"storage"`
}
type AnchorRetirementObservation struct {
	Runtime       Observation          `json:"runtime"`
	AnchorProcess RecoveryProcessState `json:"anchor_process"`
	Piers         []RecoveryPier       `json:"piers"`
	Processes     []AnchorDrainProcess `json:"processes"`
	CapturedAt    time.Time            `json:"captured_at"`
}
type AnchorRetiredMember struct {
	InstanceID         string                       `json:"instance_id"`
	Phase              string                       `json:"phase"`
	RemoveAcknowledged bool                         `json:"remove_acknowledged"`
	Before             *AnchorRetirementObservation `json:"before,omitempty"`
	After              *AnchorRetirementObservation `json:"after,omitempty"`
}

// AnchorRetirement removes the exact cleanly exited old containers under an
// explicit successor namespace. It preserves every pier and all old receipts.
// No queue, persistent journal, replacement or public runtime authority is wired.
type AnchorRetirement struct {
	OperationID string                `json:"operation_id"`
	Sequence    uint64                `json:"sequence"`
	Namespace   AnchorNamespace       `json:"namespace"`
	Members     []AnchorRetiredMember `json:"members"`
}

func cloneAnchorRetirementObservation(o AnchorRetirementObservation) AnchorRetirementObservation {
	o.Runtime = cloneReadinessObservation(o.Runtime)
	o.Piers = slices.Clone(o.Piers)
	o.Processes = slices.Clone(o.Processes)
	return o
}
func (s AnchorRetirement) Clone() AnchorRetirement {
	s.Namespace = s.Namespace.Clone()
	s.Members = slices.Clone(s.Members)
	for i := range s.Members {
		m := &s.Members[i]
		if m.Before != nil {
			o := cloneAnchorRetirementObservation(*m.Before)
			m.Before = &o
		}
		if m.After != nil {
			o := cloneAnchorRetirementObservation(*m.After)
			m.After = &o
		}
	}
	return s
}
func NewAnchorRetirement(h AnchorRecoveryHistory, namespace AnchorNamespace, operation string) (AnchorRetirement, error) {
	s := AnchorRetirement{OperationID: operation, Namespace: namespace.Clone()}
	for _, m := range namespace.Drain.Members {
		s.Members = append(s.Members, AnchorRetiredMember{InstanceID: m.InstanceID, Phase: "planned"})
	}
	return s, s.Validate(h)
}
func recoveryOperationExists(h AnchorRecoveryHistory, id string) bool {
	if recoveryEpochOperationExists(h, id) {
		return true
	}
	for _, l := range h.Launches {
		for _, old := range []string{l.OperationID, l.Storage.OperationID, l.Anchor.OperationID, l.Anchor.Prepared.OperationID} {
			if id == old {
				return true
			}
		}
	}
	for _, r := range h.Resumes {
		for _, old := range []string{r.OperationID, r.Retirement.OperationID, r.Retirement.Exit.OperationID, r.Retirement.Exit.Stop.OperationID, r.Retirement.Exit.Stop.ReadinessOperationID, r.Retirement.Exit.Stop.LaunchOperationID} {
			if id == old {
				return true
			}
		}
	}
	return false
}
func sortedRetirementObservation(o AnchorRetirementObservation) AnchorRetirementObservation {
	o = cloneAnchorRetirementObservation(o)
	sortRecoveryMembers(&o.Runtime)
	less := func(a, b string) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
	slices.SortFunc(o.Piers, func(a, b RecoveryPier) int { return less(a.InstanceID, b.InstanceID) })
	slices.SortFunc(o.Processes, func(a, b AnchorDrainProcess) int { return less(a.InstanceID, b.InstanceID) })
	// Both nil and [] denote the same empty observed set after the last removal.
	if len(o.Runtime.Members) == 0 {
		o.Runtime.Members = nil
	}
	if len(o.Processes) == 0 {
		o.Processes = nil
	}
	return o
}
func (s AnchorRetirement) checkObservation(index int, o AnchorRetirementObservation, absent bool, now time.Time) error {
	fail := func() error { return errors.New("retirement lost exact stopped-container, pier or successor evidence") }
	if index < 0 || index >= len(s.Members) || s.Namespace.After == nil {
		return fail()
	}
	source := s.Namespace.After
	if !freshNamespaceObservation(AnchorDrainObservation{Runtime: o.Runtime, CapturedAt: o.CapturedAt}, now, source.CapturedAt) {
		return fail()
	}
	expected := AnchorRetirementObservation{Runtime: cloneReadinessObservation(source.Runtime), AnchorProcess: source.AnchorProcess, Processes: slices.Clone(source.Processes), CapturedAt: o.CapturedAt}
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	for _, p := range source.Processes {
		expected.Piers = append(expected.Piers, RecoveryPier{InstanceID: p.InstanceID, Storage: p.Storage})
	}
	for i, m := range s.Members {
		if i < index || i == index && absent {
			expected.Runtime.Members = slices.DeleteFunc(expected.Runtime.Members, func(v MemberObservation) bool { return v.InstanceID == m.InstanceID })
			expected.Processes = slices.DeleteFunc(expected.Processes, func(v AnchorDrainProcess) bool { return v.InstanceID == m.InstanceID })
		}
	}
	if !equalEvidence(sortedRetirementObservation(expected), sortedRetirementObservation(o)) {
		return fail()
	}
	if index > 0 {
		prior := s.Members[index-1].After
		if prior == nil || o.CapturedAt.Before(prior.CapturedAt) || o.Runtime.ObservedAt.Before(prior.Runtime.ObservedAt) {
			return fail()
		}
	}
	if absent {
		before := s.Members[index].Before
		if before == nil || o.CapturedAt.Before(before.CapturedAt) || o.Runtime.ObservedAt.Before(before.Runtime.ObservedAt) {
			return fail()
		}
	}
	return nil
}
func (s AnchorRetirement) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid coordinated container retirement") }
	if s.Namespace.Validate(h) != nil || s.Namespace.Phase != "observed" || !protocol.ValidID(s.OperationID) || s.OperationID == s.Namespace.OperationID || s.OperationID == s.Namespace.Drain.OperationID || recoveryOperationExists(h, s.OperationID) || len(s.Members) != len(s.Namespace.Drain.Members) {
		return fail()
	}
	var sequence uint64
	pending := false
	for i, m := range s.Members {
		if m.InstanceID != s.Namespace.Drain.Members[i].InstanceID {
			return fail()
		}
		if m.Phase == "planned" {
			if m.Before != nil || m.After != nil || m.RemoveAcknowledged {
				return fail()
			}
			pending = true
			continue
		}
		if pending || m.Before == nil || s.checkObservation(i, *m.Before, false, m.Before.CapturedAt) != nil {
			return fail()
		}
		switch m.Phase {
		case "remove-dispatched":
			if m.After != nil || m.RemoveAcknowledged {
				return fail()
			}
			sequence++
			pending = true
		case "remove-acknowledged":
			if m.After != nil || !m.RemoveAcknowledged {
				return fail()
			}
			sequence += 2
			pending = true
		case "absent":
			if m.After == nil || s.checkObservation(i, *m.After, true, m.After.CapturedAt) != nil {
				return fail()
			}
			sequence += 2
			if m.RemoveAcknowledged {
				sequence++
			}
		default:
			return fail()
		}
	}
	if s.Sequence != sequence {
		return fail()
	}
	return nil
}
func (s AnchorRetirement) next() int {
	for i, m := range s.Members {
		if m.Phase != "absent" {
			return i
		}
	}
	return -1
}

// Complete is a classification; consumers must first Validate against history.
func (s AnchorRetirement) Complete() bool { return len(s.Members) > 0 && s.next() < 0 }
func (s AnchorRetirement) Advance(h AnchorRecoveryHistory, index int, expected uint64, phase string, o *AnchorRetirementObservation) (AnchorRetirement, error) {
	if s.Validate(h) != nil || s.Sequence != expected || index < 0 || index != s.next() {
		return AnchorRetirement{}, errors.New("stale coordinated retirement")
	}
	next := s.Clone()
	m := &next.Members[index]
	switch {
	case m.Phase == "planned" && phase == "remove-dispatched":
		if o == nil {
			return AnchorRetirement{}, errors.New("retirement requires fresh stopped evidence")
		}
		v := cloneAnchorRetirementObservation(*o)
		m.Before = &v
	case m.Phase == "remove-dispatched" && phase == "remove-acknowledged":
		if o != nil {
			return AnchorRetirement{}, errors.New("removal acknowledgement cannot replace observation")
		}
		m.RemoveAcknowledged = true
	case (m.Phase == "remove-dispatched" || m.Phase == "remove-acknowledged") && phase == "absent":
		if o == nil {
			return AnchorRetirement{}, errors.New("retirement requires complete absence evidence")
		}
		v := cloneAnchorRetirementObservation(*o)
		m.After = &v
	default:
		return AnchorRetirement{}, errors.New("retirement cannot skip or replay a removal")
	}
	m.Phase = phase
	next.Sequence++
	return next, next.Validate(h)
}
func (s AnchorRetirement) ValidateObservation(h AnchorRecoveryHistory, index int, o AnchorRetirementObservation, absent bool, now time.Time) error {
	if err := s.Validate(h); err != nil {
		return err
	}
	return s.checkObservation(index, o, absent, now)
}

type AnchorRetirementObserver interface {
	VerifyEngine(context.Context, string) error
	InspectRetirement(context.Context, AnchorRetirement) (AnchorRetirementObservation, error)
}
type AnchorRetirementDriver interface {
	AnchorRetirementObserver
	// Remove must recheck exact clean exit, full ownership, initialized storage,
	// lock absence and successor immediately before one unforced exact-ID removal.
	RemoveRetiredMember(context.Context, AnchorRetirement, int) error
}
type AnchorRetirementCommit func(expected uint64, next AnchorRetirement) error

func ExecuteNextAnchorRetirement(ctx context.Context, h AnchorRecoveryHistory, s AnchorRetirement, d AnchorRetirementDriver, commit AnchorRetirementCommit, now func() time.Time) (AnchorRetirement, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("retirement requires retained authority and driver")
	}
	h = h.Clone()
	s = s.Clone()
	index := s.next()
	if index < 0 || s.Members[index].Phase != "planned" {
		return s, errors.New("retirement has no fresh removal intent")
	}
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Namespace.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	advance := func(phase string, o *AnchorRetirementObservation) error {
		next, err := s.Advance(h, index, s.Sequence, phase, o)
		if err != nil {
			return err
		}
		if err = commit(s.Sequence, next.Clone()); err != nil {
			return err
		}
		s = next
		return nil
	}
	if !verify() {
		return s, fail("retirement engine unavailable")
	}
	before, err := d.InspectRetirement(ctx, s.Clone())
	before = cloneAnchorRetirementObservation(before)
	if err != nil || !verify() || s.checkObservation(index, before, false, now()) != nil {
		return s, fail("retirement stopped observation refused")
	}
	if err = advance("remove-dispatched", &before); err != nil {
		return s, err
	}
	if !verify() || s.checkObservation(index, *s.Members[index].Before, false, now()) != nil {
		return s, fail("retirement dispatch evidence expired or changed")
	}
	if err = d.RemoveRetiredMember(ctx, s.Clone(), index); err != nil {
		return s, fail("removal outcome unresolved; do not resend")
	}
	if err = advance("remove-acknowledged", nil); err != nil {
		return s, err
	}
	return ObserveAnchorRetirementAbsence(ctx, h, s, d, commit, now)
}

// ObserveAnchorRetirementAbsence resolves one pending removal without repeating it.
func ObserveAnchorRetirementAbsence(ctx context.Context, h AnchorRecoveryHistory, s AnchorRetirement, d AnchorRetirementObserver, commit AnchorRetirementCommit, now func() time.Time) (AnchorRetirement, error) {
	if s.Validate(h) != nil || d == nil || commit == nil || now == nil {
		return s, errors.New("retirement observation requires retained intent")
	}
	h = h.Clone()
	s = s.Clone()
	index := s.next()
	if index < 0 || (s.Members[index].Phase != "remove-dispatched" && s.Members[index].Phase != "remove-acknowledged") {
		return s, errors.New("retirement observation has no unresolved removal")
	}
	fail := func() (AnchorRetirement, error) {
		return s, errors.Join(errors.New("retirement absence remains unresolved"), ctx.Err())
	}
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, s.Namespace.Drain.Review.Context.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return fail()
	}
	after, err := d.InspectRetirement(ctx, s.Clone())
	after = cloneAnchorRetirementObservation(after)
	if err != nil || !verify() || s.checkObservation(index, after, true, now()) != nil {
		return fail()
	}
	next, err := s.Advance(h, index, s.Sequence, "absent", &after)
	if err != nil {
		return fail()
	}
	if err = commit(s.Sequence, next.Clone()); err != nil {
		return s, err
	}
	return next, nil
}
