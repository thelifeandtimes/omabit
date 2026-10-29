package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// MemberStopExitEvidence is a separate read-only observation of a retained Stop.
// It does not rewrite that Stop, claim a lost signal acknowledgement, or authorize
// a retry/resume. OperationID belongs to the new observation, not the old effect.
type MemberStopExitEvidence struct {
	OperationID string                `json:"operation_id"`
	Stop        MemberStop            `json:"stop"`
	Observation MemberStopObservation `json:"observation"`
}

func (e MemberStopExitEvidence) Validate(plan Plan) error {
	fail := func() error { return errors.New("invalid retained Stop exit observation") }
	s := e.Stop
	if !protocol.ValidID(e.OperationID) || e.OperationID == s.OperationID || e.OperationID == s.LaunchOperationID || e.OperationID == s.ReadinessOperationID || s.Validate(plan) != nil || s.Before == nil {
		return fail()
	}
	if s.Phase != "stop-dispatched" && s.Phase != "stop-signalled" && s.Phase != "stopped" {
		return fail()
	}
	o := e.Observation
	if s.checkObservation(plan, o, true, o.CapturedAt) != nil {
		return fail()
	}
	if s.After != nil {
		if o.CapturedAt.Before(s.After.CapturedAt) || o.Runtime.ObservedAt.Before(s.After.Runtime.ObservedAt) {
			return fail()
		}
		// A retained known exit cannot change when observed again. Runtime freshness
		// timestamps may advance, but process outcome and filesystem identity may not.
		old := cloneMemberStopObservation(*s.After)
		old.CapturedAt = o.CapturedAt
		old.Runtime.ObservedAt = o.Runtime.ObservedAt
		if !reflect.DeepEqual(old, o) {
			return fail()
		}
	}
	return nil
}

// Clean classifies exit metadata only. Validate first; collect a fresh ownership
// proof before any later lifecycle action. A clean observation does not change
// the status of the original Stop operation.
func (e MemberStopExitEvidence) Clean() bool {
	return e.Observation.ExitCode == 0 && !e.Observation.OOMKilled && !e.Observation.FinishedAt.IsZero()
}

// InspectExit must perform a complete bracketed read of the bound project,
// original initialized storage and exact recorded process. It has no mutation
// method. A running/missing/replaced target is not a completed exit observation.
type MemberStopExitDriver interface {
	VerifyEngine(context.Context, string) error
	InspectExit(context.Context, MemberStop) (MemberStopObservation, error)
}

func CollectMemberStopExit(ctx context.Context, plan Plan, s MemberStop, operation string, d MemberStopExitDriver, now func() time.Time) (MemberStopExitEvidence, error) {
	fail := func(message string) (MemberStopExitEvidence, error) {
		return MemberStopExitEvidence{}, errors.Join(errors.New(message), ctx.Err())
	}
	if d == nil || now == nil || s.Validate(plan) != nil || s.Before == nil || !protocol.ValidID(operation) || operation == s.OperationID || operation == s.LaunchOperationID || operation == s.ReadinessOperationID {
		return fail("exit observation requires its own intent and a retained process binding")
	}
	s = cloneMemberStop(s)
	verify := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.VerifyEngine(ctx, s.Binding.EngineID); err != nil {
			return errors.New("exit observation engine verification failed")
		}
		return ctx.Err()
	}
	if err := verify(); err != nil {
		return fail("exit observation engine is unavailable")
	}
	o, err := d.InspectExit(ctx, cloneMemberStop(s))
	o = cloneMemberStopObservation(o)
	if err != nil {
		return fail("retained Stop has no complete current exit observation")
	}
	if err = verify(); err != nil {
		return fail("exit observation engine changed")
	}
	result := MemberStopExitEvidence{OperationID: operation, Stop: s, Observation: o}
	if result.Validate(plan) != nil || s.checkObservation(plan, o, true, now()) != nil {
		return fail("retained Stop exit observation is stale or changed")
	}
	if err = ctx.Err(); err != nil {
		return fail("exit observation was cancelled")
	}
	return result, nil
}
