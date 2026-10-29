package projectnetwork

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
)

// CurrentMemberReadiness is a new observation against the complete retained
// member history. It never changes an earlier readiness, Stop or resume receipt.
// Its durable journal is separate from current mutation admission.
type CurrentMemberReadiness struct {
	InstanceID          string                `json:"instance_id"`
	CreatorOperationID  string                `json:"creator_operation_id"`
	ContainerGeneration uint64                `json:"container_generation"`
	Binding             RuntimeBinding        `json:"binding"`
	Storage             pierstore.Location    `json:"storage"`
	Ship                string                `json:"ship"`
	Endpoint            string                `json:"endpoint"`
	Before              MemberStopObservation `json:"before"`
	After               MemberStopObservation `json:"after"`
	CapturedAt          time.Time             `json:"captured_at"`
}

func cloneMemberContext(c MemberContext) MemberContext {
	c.Binding.Members = slices.Clone(c.Binding.Members)
	return c
}

func currentReadinessEndpoint(plan Plan, instance string) (Member, string, error) {
	for _, m := range plan.Members {
		if m.InstanceID == instance {
			return m, fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort), nil
		}
	}
	return Member{}, "", errors.New("current member is absent from plan")
}

func checkCurrentReadinessObservation(plan Plan, c MemberContext, o MemberStopObservation, now time.Time) error {
	// Reuse the strict running-process/project/pier observation checks used before
	// Stop, without constructing Stop intent or dispatching any effect.
	s := MemberStop{InstanceID: c.Member.InstanceID, Binding: c.Binding, Storage: c.Storage}
	if s.checkObservation(plan, o, false, now) != nil || o.CapturedAt.Before(c.NotBefore) || o.Runtime.ObservedAt.Before(c.NotBefore) || !c.KnownStartedAt.IsZero() && !o.StartedAt.Equal(c.KnownStartedAt) {
		return errors.New("current readiness lost exact process, project or pier")
	}
	return nil
}

func (r CurrentMemberReadiness) Validate(plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness) error {
	c, err := SelectMemberContext(plan, launches, resumes, original)
	if err != nil {
		return err
	}
	return r.validateContext(plan, c)
}
func (r CurrentMemberReadiness) validateContext(plan Plan, c MemberContext) error {
	fail := func() error { return errors.New("invalid current member readiness evidence") }
	if r.CapturedAt.IsZero() || r.InstanceID != c.Member.InstanceID || r.CreatorOperationID != c.Member.OperationID || r.ContainerGeneration != c.Member.Generation || !equalEvidence(r.Binding, c.Binding) || r.Storage != c.Storage {
		return fail()
	}

	m, endpoint, err := currentReadinessEndpoint(plan, r.InstanceID)
	if err != nil || r.Ship != m.Ship || r.Endpoint != endpoint {
		return fail()
	}
	for _, o := range []MemberStopObservation{r.Before, r.After} {
		if checkCurrentReadinessObservation(plan, c, o, r.CapturedAt) != nil {
			return fail()
		}
	}
	if r.After.CapturedAt.Before(r.Before.CapturedAt) || r.After.Runtime.ObservedAt.Before(r.Before.Runtime.ObservedAt) {
		return fail()
	}
	before := cloneMemberStopObservation(r.Before)
	before.CapturedAt, before.Runtime.ObservedAt = r.After.CapturedAt, r.After.Runtime.ObservedAt
	if !equalEvidence(before, r.After) {
		return fail()
	}
	return nil
}

type CurrentReadinessDriver interface {
	VerifyEngine(context.Context, string) error
	Inspect(context.Context, MemberContext) (MemberStopObservation, error)
	ReadHostIdentity(context.Context, string) (string, error)
}

// CollectCurrentMemberReadiness brackets HTTP identity with fresh observations of
// the current causal binding. The caller must hold the project fence and reject
// pending/uncertain operations before collection. For an original container whose
// receipt lacks StartedAt, this proves continuity only across these two reads.
func CollectCurrentMemberReadiness(ctx context.Context, plan Plan, launches []MemberLaunch, resumes []MemberResume, original InitialMemberReadiness, d CurrentReadinessDriver, now func() time.Time) (CurrentMemberReadiness, error) {
	c, err := SelectMemberContext(plan, launches, resumes, original)
	if err != nil {
		return CurrentMemberReadiness{}, err
	}
	return collectCurrentMemberReadinessContext(ctx, plan, c, d, now)
}
func collectCurrentMemberReadinessContext(ctx context.Context, plan Plan, c MemberContext, d CurrentReadinessDriver, now func() time.Time) (CurrentMemberReadiness, error) {
	refuse := func() (CurrentMemberReadiness, error) {
		return CurrentMemberReadiness{}, errors.New("current member readiness could not be established")
	}
	if d == nil || now == nil {
		return refuse()
	}
	m, endpoint, err := currentReadinessEndpoint(plan, c.Member.InstanceID)
	if err != nil {
		return refuse()
	}
	verify := func() bool {
		return ctx.Err() == nil && d.VerifyEngine(ctx, c.Binding.EngineID) == nil && ctx.Err() == nil
	}
	if !verify() {
		return refuse()
	}
	before, err := d.Inspect(ctx, cloneMemberContext(c))
	before = cloneMemberStopObservation(before)
	if err != nil || !verify() || checkCurrentReadinessObservation(plan, c, before, now()) != nil {
		return refuse()
	}
	ship, err := d.ReadHostIdentity(ctx, endpoint)
	if err != nil || ship != m.Ship || !verify() {
		return refuse()
	}
	after, err := d.Inspect(ctx, cloneMemberContext(c))
	after = cloneMemberStopObservation(after)
	if err != nil || !verify() {
		return refuse()
	}
	r := CurrentMemberReadiness{InstanceID: m.InstanceID, CreatorOperationID: c.Member.OperationID, ContainerGeneration: c.Member.Generation, Binding: c.Binding, Storage: c.Storage, Ship: ship, Endpoint: endpoint, Before: before, After: after, CapturedAt: now()}
	if ctx.Err() != nil || r.validateContext(plan, c) != nil {
		return refuse()
	}
	return r, nil
}
