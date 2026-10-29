package projectnetwork

import (
	"context"
	"errors"
	"reflect"
	"time"
)

// SelectRecoveryMemberContext preserves every completed boundary and the current
// ordinary suffix. It never substitutes a successor namespace into AnchorStart.
func SelectRecoveryMemberContext(h AnchorRecoveryHistory, instance string) (MemberContext, error) {
	if len(h.Epochs) == 0 {
		return MemberContext{}, errors.New("member history has no completed recovery")
	}
	c, err := SelectAnchorRecoveryHistoryContext(h)
	if err != nil {
		return MemberContext{}, err
	}
	for _, m := range c.Members {
		if m.Context.Member.InstanceID == instance {
			v := cloneMemberContext(m.Context)
			v.KnownStartedAt = m.StartedAt
			v.NotBefore = m.NotBefore
			return v, nil
		}
	}
	return MemberContext{}, errors.New("member absent from recovered history")
}
func (r CurrentMemberReadiness) ValidateRecovered(h AnchorRecoveryHistory) error {
	c, err := SelectRecoveryMemberContext(h, r.InstanceID)
	if err != nil {
		return err
	}
	return r.validateContext(h.Plan, c)
}
func (r CurrentMemberReadiness) ValidateRecoveredHistorical(h AnchorRecoveryHistory) error {
	prefix, err := AnchorRecoveryHistoryAtBinding(h, r.Binding)
	if err != nil {
		return err
	}
	return r.ValidateRecovered(prefix)
}
func CollectRecoveredCurrentMemberReadiness(ctx context.Context, h AnchorRecoveryHistory, instance string, d CurrentReadinessDriver, now func() time.Time) (CurrentMemberReadiness, error) {
	h = h.Clone()
	c, err := SelectRecoveryMemberContext(h, instance)
	if err != nil {
		return CurrentMemberReadiness{}, err
	}
	return collectCurrentMemberReadinessContext(ctx, h.Plan, c, d, now)
}
func NewRecoveredCurrentMemberStop(h AnchorRecoveryHistory, ready CurrentMemberReadiness, readinessOperation, operation string) (MemberStop, error) {
	if ready.ValidateRecovered(h) != nil {
		return MemberStop{}, errors.New("Stop requires recovered current readiness")
	}
	return newCurrentMemberStop(h.Plan, ready, readinessOperation, operation)
}
func ValidateRecoveredCurrentMemberStop(h AnchorRecoveryHistory, ready CurrentMemberReadiness, retained MemberStop) error {
	fail := func() error { return errors.New("recovered Stop lost historical readiness or process") }
	if retained.Validate(h.Plan) != nil || ready.ValidateRecoveredHistorical(h) != nil {
		return fail()
	}
	prefix, err := AnchorRecoveryHistoryAtBinding(h, ready.Binding)
	if err != nil {
		return fail()
	}
	expected, err := NewRecoveredCurrentMemberStop(prefix, ready, retained.ReadinessOperationID, retained.OperationID)
	baseline := cloneMemberStop(retained)
	baseline.Phase = "planned"
	baseline.Before = nil
	baseline.After = nil
	if err != nil || !reflect.DeepEqual(expected, baseline) {
		return fail()
	}
	if retained.Before != nil && (!retained.Before.StartedAt.Equal(ready.After.StartedAt) || retained.Before.CapturedAt.Before(ready.After.CapturedAt) || retained.Before.Runtime.ObservedAt.Before(ready.After.Runtime.ObservedAt)) {
		return fail()
	}
	return nil
}
