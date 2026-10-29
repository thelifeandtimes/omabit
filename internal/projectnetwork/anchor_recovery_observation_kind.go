package projectnetwork

import (
	"errors"
)

type RecoveryObservationTarget struct {
	Kind       string `json:"kind"`
	InstanceID string `json:"instance_id,omitempty"`
}

// RecoveryCheckpointObservationTarget describes only a retained outcome that can
// be established without effects. It grants no authority to adopt an unknown ID.
func RecoveryCheckpointObservationTarget(h AnchorRecoveryHistory, j AnchorRecoveryJournal) (RecoveryObservationTarget, error) {
	if j.Validate(h) != nil {
		return RecoveryObservationTarget{}, errors.New("invalid recovery checkpoint")
	}
	target := RecoveryObservationTarget{}
	switch j.Phase {
	case "draining":
		if i := j.Drain.next(); i >= 0 {
			m := j.Drain.Members[i]
			if m.Phase == "stop-dispatched" || m.Phase == "stop-signalled" {
				target = RecoveryObservationTarget{"member-exit", m.InstanceID}
			}
		}
	case "namespace":
		p := j.Namespace.Phase
		if p == "observing" || p == "start-dispatched" || p == "start-acknowledged" {
			target.Kind = "namespace"
		}
	case "retiring":
		if i := j.Retirement.next(); i >= 0 {
			m := j.Retirement.Members[i]
			if m.Phase == "remove-dispatched" || m.Phase == "remove-acknowledged" {
				target = RecoveryObservationTarget{"member-absence", m.InstanceID}
			}
		}
	case "replacing":
		if j.Replacement.Running() {
			target.Kind = "readiness"
		} else {
			i := j.Replacement.next()
			m := j.Replacement.Members[i]
			if m.Phase == "start-dispatched" || m.Phase == "start-acknowledged" {
				target = RecoveryObservationTarget{"member-running", m.Creation.InstanceID}
			}
		}
	}
	if target.Kind == "" {
		return target, errors.New("checkpoint needs separate review or new continuation authority; no uncertain effect may be repeated")
	}
	return target, nil
}

// RecoveryJournalObservedTransition rejects even otherwise valid mutation or
// acknowledgement transitions. Only the one requested observation may commit.
func RecoveryJournalObservedTransition(h AnchorRecoveryHistory, before, after AnchorRecoveryJournal) error {
	fail := func() error { return errors.New("recovery result is not one exact observation transition") }
	target, err := RecoveryCheckpointObservationTarget(h, before)
	if err != nil {
		return err
	}
	if after.Validate(h) != nil {
		return fail()
	}
	update := AnchorRecoveryJournal{}
	switch target.Kind {
	case "member-exit":
		i := before.Drain.next()
		if after.Phase != "draining" || after.Drain == nil || after.Drain.Members[i].Phase != "exited" {
			return fail()
		}
		update.Drain = after.Drain
	case "namespace":
		if after.Phase != "namespace" || after.Namespace == nil || after.Namespace.Phase != "observed" {
			return fail()
		}
		update.Namespace = after.Namespace
	case "member-absence":
		i := before.Retirement.next()
		if after.Phase != "retiring" || after.Retirement == nil || after.Retirement.Members[i].Phase != "absent" {
			return fail()
		}
		update.Retirement = after.Retirement
	case "member-running":
		i := before.Replacement.next()
		if after.Phase != "replacing" || after.Replacement == nil || after.Replacement.Members[i].Phase != "running" {
			return fail()
		}
		update.Replacement = after.Replacement
	case "readiness":
		if after.Phase != "complete" || after.Readiness == nil {
			return fail()
		}
		update.Readiness = after.Readiness
	default:
		return fail()
	}
	expected, err := before.Advance(h, before.Sequence, after.Phase, update)
	if err != nil || !equalEvidence(expected, after) {
		return fail()
	}
	return nil
}
