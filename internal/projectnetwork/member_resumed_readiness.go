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

// ResumedRuntimeBinding replaces only the retired member's resource. Project
// generation and namespace stay unchanged; the member has its own generation.
// The completed receipt conveys historical identity, never live authority.
func ResumedRuntimeBinding(plan Plan, s MemberResume) (RuntimeBinding, error) {
	if s.Validate(plan) != nil || s.Phase != "running" {
		return RuntimeBinding{}, errors.New("completed member resume is required")
	}
	b := s.Retirement.Exit.Stop.Binding
	b.Members = slices.Clone(b.Members)
	for i := range b.Members {
		if b.Members[i].InstanceID == s.Retirement.Exit.Stop.InstanceID {
			b.Members[i].Container = s.Container
		}
	}
	if err := b.Validate(plan); err != nil {
		return RuntimeBinding{}, err
	}
	return b, nil
}

// ResumedMemberReadiness is a separate read-only observation of the exact process
// recorded by a completed resume. Stopped peers may remain stopped. It neither
// completes an uncertain resume nor rewrites original readiness or launch history.
type ResumedMemberReadiness struct {
	InstanceID          string                  `json:"instance_id"`
	ResumeOperationID   string                  `json:"resume_operation_id"`
	ContainerGeneration uint64                  `json:"container_generation"`
	Binding             RuntimeBinding          `json:"binding"`
	Storage             pierstore.Location      `json:"storage"`
	Ship                string                  `json:"ship"`
	Endpoint            string                  `json:"endpoint"`
	Before              MemberResumeObservation `json:"before"`
	After               MemberResumeObservation `json:"after"`
	CapturedAt          time.Time               `json:"captured_at"`
}

func resumedReadinessMember(plan Plan, s MemberResume) (Member, string, error) {
	for _, m := range plan.Members {
		if m.InstanceID == s.Retirement.Exit.Stop.InstanceID {
			return m, fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort), nil
		}
	}
	return Member{}, "", errors.New("resumed member missing from plan")
}
func checkResumedReadinessObservation(plan Plan, s MemberResume, b RuntimeBinding, o MemberResumeObservation, now time.Time) error {
	fail := func() error { return errors.New("resumed readiness lost exact process, project or pier") }
	if s.After == nil || s.checkObservation(o, now, true) != nil || o.CapturedAt.Before(s.After.CapturedAt) || o.Runtime.ObservedAt.Before(s.After.Runtime.ObservedAt) || !o.StartedAt.Equal(s.After.StartedAt) {
		return fail()
	}
	a, err := AssessRuntime(plan, b, o.Runtime, now)
	if err != nil || a.RecoveryRequired || !slices.Contains(a.Running, s.Retirement.Exit.Stop.InstanceID) {
		return fail()
	}
	expected := cloneResumeObservation(*s.After)
	expected.Runtime.ObservedAt, expected.CapturedAt = o.Runtime.ObservedAt, o.CapturedAt
	if !reflect.DeepEqual(expected, o) {
		return fail()
	}
	return nil
}
func (r ResumedMemberReadiness) Validate(plan Plan, s MemberResume) error {
	fail := func() error { return errors.New("invalid resumed member readiness evidence") }
	b, err := ResumedRuntimeBinding(plan, s)
	if err != nil || r.CapturedAt.IsZero() || !reflect.DeepEqual(r.Binding, b) || r.ResumeOperationID != s.OperationID || r.ContainerGeneration != s.PreviousGeneration+1 || r.InstanceID != s.Retirement.Exit.Stop.InstanceID || r.Storage != s.Retirement.Exit.Stop.Storage {
		return fail()
	}
	m, endpoint, err := resumedReadinessMember(plan, s)
	if err != nil || r.Ship != m.Ship || r.Endpoint != endpoint {
		return fail()
	}
	for _, o := range []MemberResumeObservation{r.Before, r.After} {
		if checkResumedReadinessObservation(plan, s, b, o, r.CapturedAt) != nil {
			return fail()
		}
	}
	if r.After.CapturedAt.Before(r.Before.CapturedAt) || r.After.Runtime.ObservedAt.Before(r.Before.Runtime.ObservedAt) {
		return fail()
	}
	return nil
}

type ResumedReadinessDriver interface {
	VerifyEngine(context.Context, string) error
	Inspect(context.Context, MemberResume) (MemberResumeObservation, error)
	ReadHostIdentity(context.Context, string) (string, error)
}

// CollectResumedMemberReadiness brackets credential-free HTTP identity with
// complete exact-process/project/pier observations. It does not retry mutations.
func CollectResumedMemberReadiness(ctx context.Context, plan Plan, s MemberResume, d ResumedReadinessDriver, now func() time.Time) (ResumedMemberReadiness, error) {
	refuse := func() (ResumedMemberReadiness, error) {
		return ResumedMemberReadiness{}, errors.New("resumed member readiness could not be established")
	}
	b, err := ResumedRuntimeBinding(plan, s)
	if err != nil || d == nil || now == nil {
		return refuse()
	}
	s = cloneMemberResume(s)
	m, endpoint, err := resumedReadinessMember(plan, s)
	if err != nil {
		return refuse()
	}
	verify := func() bool { return ctx.Err() == nil && d.VerifyEngine(ctx, b.EngineID) == nil && ctx.Err() == nil }
	if !verify() {
		return refuse()
	}
	before, err := d.Inspect(ctx, cloneMemberResume(s))
	before = cloneResumeObservation(before)
	if err != nil || !verify() || checkResumedReadinessObservation(plan, s, b, before, now()) != nil {
		return refuse()
	}
	ship, err := d.ReadHostIdentity(ctx, endpoint)
	if err != nil || ship != m.Ship || !verify() {
		return refuse()
	}
	after, err := d.Inspect(ctx, cloneMemberResume(s))
	after = cloneResumeObservation(after)
	if err != nil || !verify() {
		return refuse()
	}
	r := ResumedMemberReadiness{InstanceID: m.InstanceID, ResumeOperationID: s.OperationID, ContainerGeneration: s.PreviousGeneration + 1, Binding: b, Storage: s.Retirement.Exit.Stop.Storage, Ship: ship, Endpoint: endpoint, Before: before, After: after, CapturedAt: now()}
	if ctx.Err() != nil || r.Validate(plan, s) != nil {
		return refuse()
	}
	return r, nil
}
