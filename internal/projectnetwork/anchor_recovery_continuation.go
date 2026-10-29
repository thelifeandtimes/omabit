package projectnetwork

import "errors"

// RecoveryContinuationTarget describes the first new step after an explicitly
// reviewed interrupted checkpoint. It does not resume a worker or grant runtime
// authority. A separate actor/request-bound store must admit continuation.
type RecoveryContinuationTarget struct {
	Kind       string `json:"kind"`
	InstanceID string `json:"instance_id,omitempty"`
}

// RecoveryCheckpointContinuationTarget refuses every dispatched-but-unobserved
// effect. Those require the observation path first. Unknown Create never becomes
// adoption or a second Create; a retained returned ID may only reach a separately
// implemented fresh inspection/Start path.
func RecoveryCheckpointContinuationTarget(h AnchorRecoveryHistory, j AnchorRecoveryJournal) (RecoveryContinuationTarget, error) {
	fail := func() (RecoveryContinuationTarget, error) {
		return RecoveryContinuationTarget{}, errors.New("checkpoint needs an observation or separate review before new recovery actions")
	}
	if j.Validate(h) != nil {
		return fail()
	}
	target := RecoveryContinuationTarget{}
	switch j.Phase {
	case "reviewed":
		target.Kind = "begin-drain"
	case "draining":
		if j.Drain.Clean() {
			target.Kind = "establish-namespace"
			break
		}
		for _, m := range j.Drain.Members {
			if m.Phase == "exited" {
				for _, p := range m.After.Processes {
					if p.InstanceID == m.InstanceID && (p.Process.ExitCode != 0 || p.Process.OOMKilled) {
						return fail()
					}
				}
				continue
			}
			if m.Phase == "planned" {
				target = RecoveryContinuationTarget{"stop-member", m.InstanceID}
			}
			break
		}
	case "namespace":
		if j.Namespace.Phase == "planned" {
			target.Kind = "establish-namespace"
		}
		if j.Namespace.Phase == "observed" {
			target.Kind = "begin-retirement"
		}
	case "retiring":
		if j.Retirement.Complete() {
			target.Kind = "plan-replacements"
			break
		}
		m := j.Retirement.Members[j.Retirement.next()]
		if m.Phase == "planned" {
			target = RecoveryContinuationTarget{"retire-member", m.InstanceID}
		}
	case "replacing":
		if j.Replacement.Running() {
			break
		}
		m := j.Replacement.Members[j.Replacement.next()]
		switch m.Phase {
		case "planned":
			target = RecoveryContinuationTarget{"create-member", m.Creation.InstanceID}
		case "created":
			target = RecoveryContinuationTarget{"start-created-member", m.Creation.InstanceID}
		}
	}
	if target.Kind == "" {
		return fail()
	}
	return target, nil
}
