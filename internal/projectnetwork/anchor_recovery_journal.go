package projectnetwork

import (
	"errors"
	"slices"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// RecoveryJournalIDs reserves every operation identity before the first effect.
// Replacement is also the parent recovery operation; member creators remain
// distinct so later ordinary lifecycle receipts can reference them individually.
type RecoveryJournalIDs struct {
	Drain       string                    `json:"drain"`
	Namespace   string                    `json:"namespace"`
	Retirement  string                    `json:"retirement"`
	Replacement string                    `json:"replacement"`
	Members     []RecoveryJournalMemberID `json:"members"`
}
type RecoveryJournalMemberID struct {
	InstanceID  string `json:"instance_id"`
	OperationID string `json:"operation_id"`
}

// AnchorRecoveryJournal is a single serial recovery state. Exactly one nested
// payload is retained; later phases embed the complete earlier receipts. Source
// history is supplied separately and must remain immutable in the owning store.
// This pure model performs no I/O and does not change the runner schema.
type AnchorRecoveryJournal struct {
	IDs         RecoveryJournalIDs       `json:"ids"`
	Phase       string                   `json:"phase"`
	Sequence    uint64                   `json:"sequence"`
	Review      *AnchorRecoveryReview    `json:"review,omitempty"`
	Drain       *AnchorDrain             `json:"drain,omitempty"`
	Namespace   *AnchorNamespace         `json:"namespace,omitempty"`
	Retirement  *AnchorRetirement        `json:"retirement,omitempty"`
	Replacement *AnchorReplacement       `json:"replacement,omitempty"`
	Readiness   *AnchorRecoveryReadiness `json:"readiness,omitempty"`
}

func (j AnchorRecoveryJournal) Clone() AnchorRecoveryJournal {
	j.IDs.Members = slices.Clone(j.IDs.Members)
	if j.Review != nil {
		v := cloneAnchorRecoveryReview(*j.Review)
		j.Review = &v
	}
	if j.Drain != nil {
		v := j.Drain.Clone()
		j.Drain = &v
	}
	if j.Namespace != nil {
		v := j.Namespace.Clone()
		j.Namespace = &v
	}
	if j.Retirement != nil {
		v := j.Retirement.Clone()
		j.Retirement = &v
	}
	if j.Replacement != nil {
		v := j.Replacement.Clone()
		j.Replacement = &v
	}
	if j.Readiness != nil {
		v := j.Readiness.Clone()
		j.Readiness = &v
	}
	return j
}
func NewAnchorRecoveryJournal(h AnchorRecoveryHistory, ids RecoveryJournalIDs) (AnchorRecoveryJournal, error) {
	j := (AnchorRecoveryJournal{IDs: ids, Phase: "review-planned"}).Clone()
	return j, j.Validate(h)
}
func (j AnchorRecoveryJournal) payloads() int {
	n := 0
	for _, present := range []bool{j.Review != nil, j.Drain != nil, j.Namespace != nil, j.Retirement != nil, j.Replacement != nil, j.Readiness != nil} {
		if present {
			n++
		}
	}
	return n
}
func (j AnchorRecoveryJournal) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid coordinated recovery journal") }
	context, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return fail()
	}
	ids := []string{j.IDs.Drain, j.IDs.Namespace, j.IDs.Retirement, j.IDs.Replacement}
	if len(j.IDs.Members) != len(context.Members) {
		return fail()
	}
	for i, m := range j.IDs.Members {
		if m.InstanceID != context.Members[i].Context.Member.InstanceID {
			return fail()
		}
		ids = append(ids, m.OperationID)
	}
	used := map[string]bool{}
	for _, id := range ids {
		if !protocol.ValidID(id) || used[id] || recoveryOperationExists(h, id) {
			return fail()
		}
		used[id] = true
	}
	if j.Phase == "review-planned" || j.Phase == "reviewing" {
		sequence := uint64(0)
		if j.Phase == "reviewing" {
			sequence = 1
		}
		if j.Sequence != sequence || j.payloads() != 0 {
			return fail()
		}
		return nil
	}
	if j.payloads() != 1 {
		return fail()
	}
	var drain *AnchorDrain
	var namespace *AnchorNamespace
	var retirement *AnchorRetirement
	var replacement *AnchorReplacement
	var sequence uint64
	switch j.Phase {
	case "reviewed":
		if j.Review == nil || j.Review.validateContext(h.Plan, context) != nil || j.Sequence != 2 {
			return fail()
		}
		return nil
	case "draining":
		if j.Drain == nil || j.Drain.Validate(h) != nil {
			return fail()
		}
		drain = j.Drain
		sequence = 3
	case "namespace":
		if j.Namespace == nil || j.Namespace.Validate(h) != nil {
			return fail()
		}
		namespace = j.Namespace
		sequence = 4
	case "retiring":
		if j.Retirement == nil || j.Retirement.Validate(h) != nil {
			return fail()
		}
		retirement = j.Retirement
		sequence = 5
	case "replacing":
		if j.Replacement == nil || j.Replacement.Validate(h) != nil {
			return fail()
		}
		replacement = j.Replacement
		sequence = 6
	case "complete":
		if j.Readiness == nil || j.Readiness.Validate(h) != nil {
			return fail()
		}
		replacement = &j.Readiness.Replacement
		sequence = 7
	default:
		return fail()
	}
	// The selected payload validates its entire embedded ancestry. Walk that
	// same immutable value for outer ID and sequence checks without validating
	// each ancestor (and its complete history) again. No result survives this call.
	if replacement != nil {
		if replacement.OperationID != j.IDs.Replacement || len(replacement.Members) != len(j.IDs.Members) {
			return fail()
		}
		for i, m := range replacement.Members {
			if m.Creation.InstanceID != j.IDs.Members[i].InstanceID || m.Creation.OperationID != j.IDs.Members[i].OperationID {
				return fail()
			}
		}
		retirement = &replacement.Retirement
		sequence += replacement.Sequence
	} else if j.Phase == "replacing" {
		return fail()
	}
	if retirement != nil {
		if retirement.OperationID != j.IDs.Retirement {
			return fail()
		}
		namespace = &retirement.Namespace
		sequence += retirement.Sequence
	} else if j.Phase == "retiring" {
		return fail()
	}
	if namespace != nil {
		if namespace.OperationID != j.IDs.Namespace {
			return fail()
		}
		drain = &namespace.Drain
		sequence += namespace.Sequence
	} else if j.Phase == "namespace" {
		return fail()
	}
	if drain == nil || drain.OperationID != j.IDs.Drain {
		return fail()
	}
	sequence += drain.Sequence
	if sequence != j.Sequence {
		return fail()
	}
	return nil
}

// Advance admits one exact existing-model transition. It cannot skip dispatch,
// manufacture an acknowledgement, rewrite an earlier peer or change reserved IDs.
// The store must commit this returned state before invoking the corresponding
// adapter. Reopened uncertain child states have observation-only adapter paths.
func (j AnchorRecoveryJournal) Advance(h AnchorRecoveryHistory, expected uint64, phase string, update AnchorRecoveryJournal) (AnchorRecoveryJournal, error) {
	fail := func() (AnchorRecoveryJournal, error) {
		return AnchorRecoveryJournal{}, errors.New("stale or invalid recovery journal transition")
	}
	if j.Validate(h) != nil || j.Sequence != expected || expected == ^uint64(0) {
		return fail()
	}
	// Update carries a payload only; it has no authority to change header fields.
	if !equalEvidence(update.IDs, RecoveryJournalIDs{}) || update.Phase != "" || update.Sequence != 0 || update.payloads() > 1 {
		return fail()
	}
	next := AnchorRecoveryJournal{IDs: j.IDs, Phase: phase, Sequence: j.Sequence + 1}
	var err error
	switch {
	case j.Phase == "review-planned" && phase == "reviewing":
		if update.payloads() != 0 {
			return fail()
		}
	case j.Phase == "reviewing" && phase == "reviewed":
		if update.Review == nil {
			return fail()
		}
		next.Review = update.Review
	case j.Phase == "reviewed" && phase == "draining":
		if update.payloads() != 0 {
			return fail()
		}
		v, e := NewAnchorDrain(h, *j.Review, j.IDs.Drain)
		err = e
		next.Drain = &v
	case j.Phase == "draining" && phase == "draining":
		if update.Drain == nil {
			return fail()
		}
		err = exactRecoveryDrainTransition(h, *j.Drain, *update.Drain)
		next.Drain = update.Drain
	case j.Phase == "draining" && phase == "namespace":
		if update.payloads() != 0 {
			return fail()
		}
		v, e := NewAnchorNamespace(h, *j.Drain, j.IDs.Namespace)
		err = e
		next.Namespace = &v
	case j.Phase == "namespace" && phase == "namespace":
		if update.Namespace == nil {
			return fail()
		}
		err = exactRecoveryNamespaceTransition(h, *j.Namespace, *update.Namespace)
		next.Namespace = update.Namespace
	case j.Phase == "namespace" && phase == "retiring":
		if update.payloads() != 0 {
			return fail()
		}
		v, e := NewAnchorRetirement(h, *j.Namespace, j.IDs.Retirement)
		err = e
		next.Retirement = &v
	case j.Phase == "retiring" && phase == "retiring":
		if update.Retirement == nil {
			return fail()
		}
		err = exactRecoveryRetirementTransition(h, *j.Retirement, *update.Retirement)
		next.Retirement = update.Retirement
	case j.Phase == "retiring" && phase == "replacing":
		if update.Replacement == nil {
			return fail()
		}
		v, e := NewAnchorReplacement(h, *j.Retirement, j.IDs.Replacement, update.Replacement.Members)
		err = e
		if !equalEvidence(v, *update.Replacement) {
			return fail()
		}
		next.Replacement = &v
	case j.Phase == "replacing" && phase == "replacing":
		if update.Replacement == nil {
			return fail()
		}
		err = exactRecoveryReplacementTransition(h, *j.Replacement, *update.Replacement)
		next.Replacement = update.Replacement
	case j.Phase == "replacing" && phase == "complete":
		if update.Readiness == nil || !equalEvidence(update.Readiness.Replacement, *j.Replacement) {
			return fail()
		}
		next.Readiness = update.Readiness
	default:
		return fail()
	}
	if err != nil {
		return fail()
	}
	next = next.Clone()
	if next.Validate(h) != nil {
		return fail()
	}
	return next, nil
}
func exactRecoveryDrainTransition(h AnchorRecoveryHistory, old, next AnchorDrain) error {
	bad := errors.New("drain journal transition differs from retained intent")
	i := old.next()
	if i < 0 || len(next.Members) != len(old.Members) {
		return bad
	}
	m := next.Members[i]
	var o *AnchorDrainObservation
	if m.Phase == "stop-dispatched" {
		o = m.Before
	} else if m.Phase == "exited" {
		o = m.After
	}
	want, err := old.Advance(h, i, old.Sequence, m.Phase, o)
	if err != nil || !equalEvidence(want, next) {
		return bad
	}
	return nil
}
func exactRecoveryNamespaceTransition(h AnchorRecoveryHistory, old, next AnchorNamespace) error {
	var o *AnchorDrainObservation
	if old.Phase == "planned" {
		o = next.Before
	} else if next.Phase == "observed" {
		o = next.After
	}
	want, err := old.Advance(h, old.Sequence, next.Phase, o)
	if err != nil || !equalEvidence(want, next) {
		return errors.New("namespace journal transition differs from retained intent")
	}
	return nil
}
func exactRecoveryRetirementTransition(h AnchorRecoveryHistory, old, next AnchorRetirement) error {
	bad := errors.New("retirement journal transition differs from retained intent")
	i := old.next()
	if i < 0 || len(next.Members) != len(old.Members) {
		return bad
	}
	m := next.Members[i]
	var o *AnchorRetirementObservation
	if m.Phase == "remove-dispatched" {
		o = m.Before
	} else if m.Phase == "absent" {
		o = m.After
	}
	want, err := old.Advance(h, i, old.Sequence, m.Phase, o)
	if err != nil || !equalEvidence(want, next) {
		return bad
	}
	return nil
}
func exactRecoveryReplacementTransition(h AnchorRecoveryHistory, old, next AnchorReplacement) error {
	bad := errors.New("replacement journal transition differs from retained intent")
	i := old.next()
	if i < 0 || len(next.Members) != len(old.Members) {
		return bad
	}
	m := next.Members[i]
	result := AnchorReplacementResult{}
	switch m.Phase {
	case "create-dispatched":
		result.Observation = m.Before
	case "created":
		result.ContainerID = m.Container.ID
	case "start-dispatched":
		result.Observation = m.Created
	case "running":
		result.Observation = m.After
	}
	want, err := old.Advance(h, i, old.Sequence, m.Phase, result)
	if err != nil || !equalEvidence(want, next) {
		return bad
	}
	return nil
}
