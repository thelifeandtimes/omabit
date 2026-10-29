package projectnetwork

import (
	"errors"
	"math"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// RecoveredResumeCreation describes an ordinary replacement after a coordinated
// recovery. It keeps the original AnchorStart and explicitly names the recovery
// and namespace history. It is not a dispatch intent or a rewritten recovery
// replacement: RetirementOperationID refers to this ordinary member retirement.
type RecoveredResumeCreation struct {
	InstanceID            string             `json:"instance_id"`
	RuntimeID             string             `json:"runtime_id"`
	OperationID           string             `json:"operation_id"`
	PreviousOperationID   string             `json:"previous_operation_id"`
	PreviousGeneration    uint64             `json:"previous_generation"`
	Generation            uint64             `json:"generation"`
	Storage               pierstore.Location `json:"storage"`
	OriginalAnchor        AnchorStart        `json:"original_anchor"`
	NamespaceOperationID  string             `json:"namespace_operation_id"`
	RecoveryOperationID   string             `json:"recovery_operation_id"`
	RetirementOperationID string             `json:"retirement_operation_id"`
	Namespace             NamespaceIdentity  `json:"namespace"`
}

func (c RecoveredResumeCreation) Clone() RecoveredResumeCreation {
	if c.OriginalAnchor.Namespace != nil {
		n := *c.OriginalAnchor.Namespace
		c.OriginalAnchor.Namespace = &n
	}
	return c
}
func SelectRecoveredResumeCreation(h AnchorRecoveryHistory, retired MemberRetirement, operation string) (RecoveredResumeCreation, error) {
	fail := func() (RecoveredResumeCreation, error) {
		return RecoveredResumeCreation{}, errors.New("post-recovery resume requires exact retained retirement and new operation")
	}
	if len(h.Epochs) == 0 || retired.Validate(h.Plan) != nil || retired.Phase != "absent" || !protocol.ValidID(operation) || recoveryOperationExists(h, operation) {
		return fail()
	}
	old := retired.Exit.Stop
	for _, id := range []string{retired.OperationID, retired.Exit.OperationID, old.OperationID, old.ReadinessOperationID, old.LaunchOperationID} {
		if operation == id {
			return fail()
		}
	}
	i := len(h.Epochs) - 1
	epoch := h.Epochs[i]
	source := recoveryEpochSource(h, i)
	c, err := SelectRecoveredMemberContext(source, epoch.Readiness, h.Resumes, old.InstanceID)
	if err != nil || c.Member.Generation == math.MaxUint64 || !reflect.DeepEqual(old.Binding, c.Binding) || old.LaunchOperationID != c.Member.OperationID || old.Storage != c.Storage || old.Before == nil || !old.Before.StartedAt.Equal(c.KnownStartedAt) || old.Before.CapturedAt.Before(c.NotBefore) || old.Before.Runtime.ObservedAt.Before(c.NotBefore) {
		return fail()
	}
	var anchor *AnchorStart
	for k, l := range h.Launches {
		if l.Storage.InstanceID == old.InstanceID {
			anchor = &h.Launches[k].Anchor
		}
	}
	if anchor == nil {
		return fail()
	}
	return (RecoveredResumeCreation{InstanceID: old.InstanceID, RuntimeID: c.Member.RuntimeID, OperationID: operation, PreviousOperationID: c.Member.OperationID, PreviousGeneration: c.Member.Generation, Generation: c.Member.Generation + 1, Storage: c.Storage, OriginalAnchor: *anchor, NamespaceOperationID: epoch.Readiness.Replacement.Retirement.Namespace.OperationID, RecoveryOperationID: epoch.Readiness.Replacement.OperationID, RetirementOperationID: retired.OperationID, Namespace: c.Binding.Namespace}).Clone(), nil
}
func (c RecoveredResumeCreation) Validate(h AnchorRecoveryHistory, r MemberRetirement) error {
	want, err := SelectRecoveredResumeCreation(h, r, c.OperationID)
	if err != nil || !reflect.DeepEqual(c, want) {
		return errors.New("post-recovery creation differs from retained authority")
	}
	return nil
}
