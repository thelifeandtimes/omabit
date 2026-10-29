package projectnetwork

import (
	"errors"
	"math"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// RecoveryMemberCreation describes a future immutable container contract after
// all old containers were retired. It is not a dispatch or current-generation
// receipt. The original anchor remains intact; namespace succession is explicit.
// A later fenced queue must bind unique operation IDs across all replacements.
type RecoveryMemberCreation struct {
	InstanceID            string             `json:"instance_id"`
	RuntimeID             string             `json:"runtime_id"`
	OperationID           string             `json:"operation_id"`
	PreviousOperationID   string             `json:"previous_operation_id"`
	PreviousGeneration    uint64             `json:"previous_generation"`
	Generation            uint64             `json:"generation"`
	Storage               pierstore.Location `json:"storage"`
	OriginalAnchor        AnchorStart        `json:"original_anchor"`
	NamespaceOperationID  string             `json:"namespace_operation_id"`
	RetirementOperationID string             `json:"retirement_operation_id"`
	Namespace             NamespaceIdentity  `json:"namespace"`
}

func (c RecoveryMemberCreation) Clone() RecoveryMemberCreation {
	if c.OriginalAnchor.Namespace != nil {
		n := *c.OriginalAnchor.Namespace
		c.OriginalAnchor.Namespace = &n
	}
	return c
}
func SelectRecoveryMemberCreation(h AnchorRecoveryHistory, retired AnchorRetirement, instance, operation string) (RecoveryMemberCreation, error) {
	if retired.Validate(h) != nil || !retired.Complete() {
		return RecoveryMemberCreation{}, errors.New("replacement contract requires complete retained retirement")
	}
	return recoveryMemberCreationFromValidatedRetirement(h, retired, instance, operation)
}

// The caller has validated the entire retirement once. Selecting each member
// from that same immutable value must not recursively revalidate every ancestor.
func recoveryMemberCreationFromValidatedRetirement(h AnchorRecoveryHistory, retired AnchorRetirement, instance, operation string) (RecoveryMemberCreation, error) {
	fail := func() (RecoveryMemberCreation, error) {
		return RecoveryMemberCreation{}, errors.New("replacement contract requires complete retained retirement and new operation")
	}
	if !protocol.ValidID(operation) || recoveryOperationExists(h, operation) || operation == retired.OperationID || operation == retired.Namespace.OperationID || operation == retired.Namespace.Drain.OperationID {
		return fail()
	}
	var previous *AnchorRecoveryMember
	for i, m := range retired.Namespace.Drain.Review.Context.Members {
		if m.Context.Member.InstanceID == instance {
			previous = &retired.Namespace.Drain.Review.Context.Members[i]
		}
	}
	if previous == nil || previous.Context.Member.Generation == math.MaxUint64 {
		return fail()
	}
	old := previous.Context.Member
	var anchor *AnchorStart
	for i, l := range h.Launches {
		if l.Storage.InstanceID == instance {
			anchor = &h.Launches[i].Anchor
		}
	}
	if anchor == nil {
		return fail()
	}
	return (RecoveryMemberCreation{InstanceID: instance, RuntimeID: old.RuntimeID, OperationID: operation, PreviousOperationID: old.OperationID, PreviousGeneration: old.Generation, Generation: old.Generation + 1, Storage: previous.Context.Storage, OriginalAnchor: *anchor, NamespaceOperationID: retired.Namespace.OperationID, RetirementOperationID: retired.OperationID, Namespace: retired.Namespace.After.Runtime.Anchor.Namespace}).Clone(), nil
}
func (c RecoveryMemberCreation) Validate(h AnchorRecoveryHistory, retired AnchorRetirement) error {
	want, err := SelectRecoveryMemberCreation(h, retired, c.InstanceID, c.OperationID)
	if err != nil || !reflect.DeepEqual(c, want) {
		return errors.New("replacement contract differs from retained successor authority")
	}
	return nil
}
