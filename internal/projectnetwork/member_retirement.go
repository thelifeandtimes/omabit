package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberRetirement removes one exact exited container, retaining the initialized
// pier, immutable Stop/exit receipts, peers and project resources. It is a separate
// explicit intent; observing a clean exit never automatically schedules removal.
// This source-only model is not persisted or exposed as a runner capability.
type MemberRetirement struct {
	OperationID string                    `json:"operation_id"`
	Phase       string                    `json:"phase"`
	Exit        MemberStopExitEvidence    `json:"exit"`
	Before      *MemberStopObservation    `json:"before,omitempty"`
	After       *MemberAbsenceObservation `json:"after,omitempty"`
}
type MemberAbsenceObservation struct {
	Runtime    Observation        `json:"runtime"`
	Storage    pierstore.Location `json:"storage"`
	CapturedAt time.Time          `json:"captured_at"`
}

func cloneMemberRetirement(s MemberRetirement) MemberRetirement {
	s.Exit.Stop = cloneMemberStop(s.Exit.Stop)
	s.Exit.Observation = cloneMemberStopObservation(s.Exit.Observation)
	if s.Before != nil {
		v := cloneMemberStopObservation(*s.Before)
		s.Before = &v
	}
	if s.After != nil {
		v := *s.After
		v.Runtime = cloneReadinessObservation(v.Runtime)
		s.After = &v
	}
	return s
}
func NewMemberRetirement(plan Plan, exit MemberStopExitEvidence, operation string) (MemberRetirement, error) {
	s := cloneMemberRetirement(MemberRetirement{OperationID: operation, Phase: "planned", Exit: exit})
	return s, s.Validate(plan)
}
func (s MemberRetirement) Validate(plan Plan) error {
	fail := func() error { return errors.New("invalid stopped member retirement") }
	if !protocol.ValidID(s.OperationID) || s.OperationID == s.Exit.OperationID || s.OperationID == s.Exit.Stop.OperationID || s.OperationID == s.Exit.Stop.LaunchOperationID || s.OperationID == s.Exit.Stop.ReadinessOperationID || s.Exit.Validate(plan) != nil || !s.Exit.Clean() {
		return fail()
	}
	switch s.Phase {
	case "planned":
		if s.Before != nil || s.After != nil {
			return fail()
		}
	case "remove-dispatched", "removed":
		if s.Before == nil || s.After != nil || s.checkBefore(plan, *s.Before, s.Before.CapturedAt) != nil {
			return fail()
		}
	case "absent":
		if s.Before == nil || s.After == nil || s.checkBefore(plan, *s.Before, s.Before.CapturedAt) != nil || s.checkAbsent(*s.After, s.After.CapturedAt) != nil {
			return fail()
		}
	default:
		return fail()
	}
	return nil
}
func (s MemberRetirement) checkBefore(plan Plan, o MemberStopObservation, now time.Time) error {
	fail := func() error { return errors.New("retirement lost exact clean exit or project ownership") }
	if s.Exit.Stop.checkObservation(plan, o, true, now) != nil || o.CapturedAt.Before(s.Exit.Observation.CapturedAt) || o.Runtime.ObservedAt.Before(s.Exit.Observation.Runtime.ObservedAt) {
		return fail()
	}
	// This gate deliberately requires the same peer lifecycle state. A changed
	// peer or namespace belongs to coordinated recovery, not an implicit adoption.
	expected := cloneMemberStopObservation(s.Exit.Observation)
	expected.CapturedAt = o.CapturedAt
	expected.Runtime.ObservedAt = o.Runtime.ObservedAt
	if !reflect.DeepEqual(expected, o) {
		return fail()
	}
	return nil
}
func (s MemberRetirement) checkAbsent(o MemberAbsenceObservation, now time.Time) error {
	fail := func() error { return errors.New("retirement has no complete unchanged absence and pier proof") }
	if s.Before == nil || now.IsZero() || o.CapturedAt.IsZero() || now.Sub(o.CapturedAt) > 30*time.Second || o.CapturedAt.Sub(now) > 5*time.Second || o.CapturedAt.Before(s.Before.CapturedAt) || o.Storage != s.Exit.Stop.Storage || o.Runtime.ObservedAt.Before(s.Before.Runtime.ObservedAt) || o.CapturedAt.Sub(o.Runtime.ObservedAt) > 30*time.Second || o.Runtime.ObservedAt.Sub(o.CapturedAt) > 5*time.Second || now.Sub(o.Runtime.ObservedAt) > 30*time.Second || o.Runtime.ObservedAt.Sub(now) > 5*time.Second {
		return fail()
	}
	expected := cloneReadinessObservation(s.Before.Runtime)
	expected.ObservedAt = o.Runtime.ObservedAt
	expected.Members = nil
	for _, m := range s.Before.Runtime.Members {
		if m.InstanceID != s.Exit.Stop.InstanceID {
			expected.Members = append(expected.Members, m)
		}
	}
	if len(o.Runtime.Members) == 0 {
		o.Runtime.Members = nil
	}
	if !reflect.DeepEqual(expected, o.Runtime) {
		return fail()
	}
	return nil
}
func (s MemberRetirement) Advance(plan Plan, expected, next string, before *MemberStopObservation, after *MemberAbsenceObservation) (MemberRetirement, error) {
	fail := func() (MemberRetirement, error) {
		return MemberRetirement{}, errors.New("stale or invalid member retirement transition")
	}
	steps := map[string]string{"planned": "remove-dispatched", "remove-dispatched": "removed", "removed": "absent"}
	if s.Validate(plan) != nil || s.Phase != expected || steps[expected] != next {
		return fail()
	}
	s = cloneMemberRetirement(s)
	switch next {
	case "remove-dispatched":
		if before == nil || after != nil {
			return fail()
		}
		v := cloneMemberStopObservation(*before)
		s.Before = &v
	case "removed":
		if before != nil || after != nil {
			return fail()
		}
	case "absent":
		if before != nil || after == nil {
			return fail()
		}
		v := *after
		v.Runtime = cloneReadinessObservation(v.Runtime)
		s.After = &v
	}
	s.Phase = next
	if s.Validate(plan) != nil {
		return fail()
	}
	return s, nil
}

// The concrete driver must re-observe exact stopped process/configuration,
// initialized pier and lock state at dispatch. RemoveStopped may remove only the
// bound full container ID, without force, volumes, recursive cleanup or signal.
// InspectAbsent must bracket the complete inventory, unchanged project/peers and
// identical initialized storage. No method creates a replacement or starts Vere.
type MemberRetirementDriver interface {
	VerifyEngine(context.Context, string) error
	InspectStopped(context.Context, MemberRetirement) (MemberStopObservation, error)
	RemoveStopped(context.Context, MemberRetirement) error
	InspectAbsent(context.Context, MemberRetirement) (MemberAbsenceObservation, error)
}
type MemberRetirementCommit func(expected string, next MemberRetirement) error

func ExecuteMemberRetirement(ctx context.Context, plan Plan, s MemberRetirement, d MemberRetirementDriver, commit MemberRetirementCommit, now func() time.Time) (MemberRetirement, error) {
	if s.Validate(plan) != nil || s.Phase != "planned" || d == nil || commit == nil || now == nil {
		return s, errors.New("retirement requires new explicit intent and durable dispatch")
	}
	s = cloneMemberRetirement(s)
	fail := func(message string) error { return errors.Join(errors.New(message), ctx.Err()) }
	verify := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.VerifyEngine(ctx, s.Exit.Stop.Binding.EngineID) != nil {
			return fail("retirement engine verification failed")
		}
		return ctx.Err()
	}
	advance := func(phase string, b *MemberStopObservation, a *MemberAbsenceObservation) error {
		next, err := s.Advance(plan, s.Phase, phase, b, a)
		if err != nil {
			return err
		}
		if err = commit(s.Phase, cloneMemberRetirement(next)); err != nil {
			return err
		}
		s = next
		return nil
	}
	if err := verify(); err != nil {
		return s, err
	}
	before, err := d.InspectStopped(ctx, cloneMemberRetirement(s))
	before = cloneMemberStopObservation(before)
	if err != nil {
		return s, fail("retirement stopped-process observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkBefore(plan, before, now()); err != nil {
		return s, err
	}
	if err = advance("remove-dispatched", &before, nil); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkBefore(plan, *s.Before, now()); err != nil {
		return s, err
	}
	if err = d.RemoveStopped(ctx, cloneMemberRetirement(s)); err != nil {
		return s, fail("retirement removal outcome is unresolved")
	}
	// Retain an acknowledged effect even if cancellation arrived with its reply.
	if err = advance("removed", nil, nil); err != nil {
		return s, err
	}
	if err = verify(); err != nil {
		return s, err
	}
	after, err := d.InspectAbsent(ctx, cloneMemberRetirement(s))
	after.Runtime = cloneReadinessObservation(after.Runtime)
	if err != nil {
		return s, fail("retirement absence observation failed")
	}
	if err = verify(); err != nil {
		return s, err
	}
	if err = s.checkAbsent(after, now()); err != nil {
		return s, err
	}
	if err = advance("absent", nil, &after); err != nil {
		return s, err
	}
	return s, nil
}
