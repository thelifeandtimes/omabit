package projectnetwork

import (
	"context"
	"errors"
	"slices"
	"time"
)

// AnchorRecoveryReadiness brackets all successor identities with complete fresh
// observations. It preserves the replacement history; it is not a persistent
// completion record or permission to release the runner's recovery fence.
type AnchorRecoveryReadiness struct {
	Replacement AnchorReplacement           `json:"replacement"`
	Binding     RuntimeBinding              `json:"binding"`
	Members     []RecoveredHostIdentity     `json:"members"`
	Before      AnchorRetirementObservation `json:"before"`
	After       AnchorRetirementObservation `json:"after"`
	CapturedAt  time.Time                   `json:"captured_at"`
}
type RecoveredHostIdentity struct {
	InstanceID          string    `json:"instance_id"`
	CreatorOperationID  string    `json:"creator_operation_id"`
	ContainerGeneration uint64    `json:"container_generation"`
	Ship                string    `json:"ship"`
	Endpoint            string    `json:"endpoint"`
	CapturedAt          time.Time `json:"captured_at"`
}

func (r AnchorRecoveryReadiness) Clone() AnchorRecoveryReadiness {
	r.Replacement = r.Replacement.Clone()
	r.Binding.Members = slices.Clone(r.Binding.Members)
	r.Members = slices.Clone(r.Members)
	r.Before = cloneAnchorRetirementObservation(r.Before)
	r.After = cloneAnchorRetirementObservation(r.After)
	return r
}

// RecoveredRuntimeBinding derives successor membership without rewriting the
// original anchor Start, member creators, or project resource generation.
func RecoveredRuntimeBinding(h AnchorRecoveryHistory, s AnchorReplacement) (RuntimeBinding, error) {
	if s.Validate(h) != nil || !s.Running() {
		return RuntimeBinding{}, errors.New("recovered binding requires every replacement running")
	}
	b := s.Retirement.Namespace.Drain.Review.Context.Binding
	b.Namespace = s.Retirement.Namespace.After.Runtime.Anchor.Namespace
	b.Members = nil
	for _, m := range s.Members {
		b.Members = append(b.Members, RuntimeMember{InstanceID: m.Creation.InstanceID, RuntimeID: m.Creation.RuntimeID, Container: m.Container})
	}
	return b, b.Validate(h.Plan)
}
func checkRecoveryReadinessObservation(s AnchorReplacement, o AnchorRetirementObservation, now time.Time) error {
	fail := func() error {
		return errors.New("recovery readiness lost exact replacement processes, piers or successor")
	}
	if !s.Running() {
		return fail()
	}
	base := s.Members[len(s.Members)-1].After
	if base == nil || !freshNamespaceObservation(AnchorDrainObservation{Runtime: o.Runtime, CapturedAt: o.CapturedAt}, now, base.CapturedAt) {
		return fail()
	}
	expected := cloneAnchorRetirementObservation(*base)
	expected.CapturedAt = o.CapturedAt
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	// Compare every process, including the last replacement. Creation's running
	// transition deliberately permits that member's first start; readiness must not.
	if !equalEvidence(sortedRetirementObservation(expected), sortedRetirementObservation(o)) {
		return fail()
	}
	return nil
}
func ValidateRecoveryReadinessObservation(h AnchorRecoveryHistory, s AnchorReplacement, o AnchorRetirementObservation, now time.Time) error {
	if _, err := RecoveredRuntimeBinding(h, s); err != nil {
		return err
	}
	return checkRecoveryReadinessObservation(s, o, now)
}
func (r AnchorRecoveryReadiness) Validate(h AnchorRecoveryHistory) error {
	fail := func() error { return errors.New("invalid complete recovery readiness evidence") }
	b, err := RecoveredRuntimeBinding(h, r.Replacement)
	if err != nil || !equalEvidence(b, r.Binding) || len(r.Members) != len(r.Replacement.Members) || r.CapturedAt.IsZero() {
		return fail()
	}
	for _, o := range []AnchorRetirementObservation{r.Before, r.After} {
		if checkRecoveryReadinessObservation(r.Replacement, o, r.CapturedAt) != nil {
			return fail()
		}
	}
	if r.After.CapturedAt.Before(r.Before.CapturedAt) || r.After.Runtime.ObservedAt.Before(r.Before.Runtime.ObservedAt) {
		return fail()
	}
	previous := r.Before.CapturedAt
	for i, v := range r.Members {
		m := r.Replacement.Members[i]
		plan, endpoint, err := currentReadinessEndpoint(h.Plan, m.Creation.InstanceID)
		if err != nil || v.InstanceID != m.Creation.InstanceID || v.CreatorOperationID != m.Creation.OperationID || v.ContainerGeneration != m.Creation.Generation || v.Ship != plan.Ship || v.Endpoint != endpoint || v.CapturedAt.IsZero() || v.CapturedAt.Before(previous) || v.CapturedAt.After(r.After.Runtime.ObservedAt) || v.CapturedAt.After(r.CapturedAt) {
			return fail()
		}
		previous = v.CapturedAt
	}
	return nil
}

type AnchorRecoveryReadinessDriver interface {
	VerifyEngine(context.Context, string) error
	InspectRecoveryReadiness(context.Context, AnchorReplacement) (AnchorRetirementObservation, error)
	ReadHostIdentity(context.Context, string) (string, error)
}

// CollectAnchorRecoveryReadiness is read-only. The caller holds the project fence
// throughout and excludes uncertain work before supplying the retained history.
func CollectAnchorRecoveryReadiness(ctx context.Context, h AnchorRecoveryHistory, s AnchorReplacement, d AnchorRecoveryReadinessDriver, now func() time.Time) (AnchorRecoveryReadiness, error) {
	refuse := func() (AnchorRecoveryReadiness, error) {
		return AnchorRecoveryReadiness{}, errors.New("complete recovery readiness could not be established")
	}
	h = h.Clone()
	s = s.Clone()
	b, err := RecoveredRuntimeBinding(h, s)
	if err != nil || d == nil || now == nil {
		return refuse()
	}
	verify := func() bool { return ctx.Err() == nil && d.VerifyEngine(ctx, b.EngineID) == nil && ctx.Err() == nil }
	if !verify() {
		return refuse()
	}
	before, err := d.InspectRecoveryReadiness(ctx, s.Clone())
	before = cloneAnchorRetirementObservation(before)
	if err != nil || !verify() || checkRecoveryReadinessObservation(s, before, now()) != nil {
		return refuse()
	}
	r := AnchorRecoveryReadiness{Replacement: s, Binding: b, Before: before}
	for _, m := range s.Members {
		member, endpoint, err := currentReadinessEndpoint(h.Plan, m.Creation.InstanceID)
		if err != nil || ctx.Err() != nil || checkRecoveryReadinessObservation(s, before, now()) != nil {
			return refuse()
		}
		ship, err := d.ReadHostIdentity(ctx, endpoint)
		if err != nil || ship != member.Ship || !verify() {
			return refuse()
		}
		r.Members = append(r.Members, RecoveredHostIdentity{InstanceID: m.Creation.InstanceID, CreatorOperationID: m.Creation.OperationID, ContainerGeneration: m.Creation.Generation, Ship: ship, Endpoint: endpoint, CapturedAt: now()})
	}
	after, err := d.InspectRecoveryReadiness(ctx, s.Clone())
	after = cloneAnchorRetirementObservation(after)
	if err != nil || !verify() {
		return refuse()
	}
	r.After = after
	r.CapturedAt = now()
	if ctx.Err() != nil || r.Validate(h) != nil {
		return refuse()
	}
	return r.Clone(), nil
}
