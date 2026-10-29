package projectnetwork

type recoveryCheckpointParts struct {
	review      *AnchorRecoveryReview
	drain       *AnchorDrain
	namespace   *AnchorNamespace
	retirement  *AnchorRetirement
	replacement *AnchorReplacement
}

func recoveryParts(j AnchorRecoveryJournal) recoveryCheckpointParts {
	p := recoveryCheckpointParts{review: j.Review, drain: j.Drain, namespace: j.Namespace, retirement: j.Retirement, replacement: j.Replacement}
	if j.Readiness != nil {
		p.replacement = &j.Readiness.Replacement
	}
	if p.replacement != nil {
		p.retirement = &p.replacement.Retirement
	}
	if p.retirement != nil {
		p.namespace = &p.retirement.Namespace
	}
	if p.namespace != nil {
		p.drain = &p.namespace.Drain
	}
	if p.drain != nil {
		p.review = &p.drain.Review
	}
	return p
}
func recoveryPhaseRank(phase string) int {
	for i, p := range []string{"review-planned", "reviewing", "reviewed", "draining", "namespace", "retiring", "replacing", "complete"} {
		if phase == p {
			return i
		}
	}
	return -1
}
func checkpointStepBefore(old, current string, phases []string) bool {
	a, b := -1, -1
	for i, p := range phases {
		if p == old {
			a = i
		}
		if p == current {
			b = i
		}
	}
	return a >= 0 && b >= a
}

// RecoveryJournalContainsCheckpoint verifies exact retained ancestry. Sequence
// numbers alone cannot authorize a historical observation: every earlier process,
// pier, immutable contract, dispatch and acknowledgement must still be present.
// It never changes the head or infers an acknowledgement omitted by either record.
func RecoveryJournalContainsCheckpoint(h AnchorRecoveryHistory, head, old AnchorRecoveryJournal) bool {
	if head.Validate(h) != nil || old.Validate(h) != nil || !equalEvidence(head.IDs, old.IDs) || old.Sequence > head.Sequence || recoveryPhaseRank(old.Phase) > recoveryPhaseRank(head.Phase) {
		return false
	}
	if old.Sequence == head.Sequence {
		return equalEvidence(head, old)
	}
	p := recoveryParts(head)
	switch old.Phase {
	case "review-planned", "reviewing":
		return true
	case "reviewed":
		return p.review != nil && equalEvidence(*p.review, *old.Review)
	case "draining":
		return p.drain != nil && drainContainsCheckpoint(*p.drain, *old.Drain)
	case "namespace":
		return p.namespace != nil && namespaceContainsCheckpoint(*p.namespace, *old.Namespace)
	case "retiring":
		return p.retirement != nil && retirementContainsCheckpoint(*p.retirement, *old.Retirement)
	case "replacing":
		return p.replacement != nil && replacementContainsCheckpoint(*p.replacement, *old.Replacement)
	default:
		return false // A completed journal cannot acquire later transitions.
	}
}
func drainContainsCheckpoint(head, old AnchorDrain) bool {
	if len(head.Members) != len(old.Members) {
		return false
	}
	head = head.Clone()
	head.Sequence = old.Sequence
	for i, p := range old.Members {
		c := &head.Members[i]
		if !checkpointStepBefore(p.Phase, c.Phase, []string{"planned", "stop-dispatched", "stop-signalled", "exited"}) || p.SignalAcknowledged && !c.SignalAcknowledged {
			return false
		}
		c.Phase = p.Phase
		c.SignalAcknowledged = p.SignalAcknowledged
		if p.Before == nil {
			c.Before = nil
		}
		if p.After == nil {
			c.After = nil
		}
	}
	return equalEvidence(head, old)
}
func namespaceContainsCheckpoint(head, old AnchorNamespace) bool {
	if old.Phase == "observed" {
		return equalEvidence(head, old)
	}
	if head.Phase == "planned" && old.Phase != "planned" || old.StartAcknowledged && !head.StartAcknowledged {
		return false
	}
	head = head.Clone()
	head.Sequence = old.Sequence
	head.Phase = old.Phase
	head.StartAcknowledged = old.StartAcknowledged
	if old.Before == nil {
		head.Before = nil
	}
	if old.After == nil {
		head.After = nil
	}
	return equalEvidence(head, old)
}
func retirementContainsCheckpoint(head, old AnchorRetirement) bool {
	if len(head.Members) != len(old.Members) {
		return false
	}
	head = head.Clone()
	head.Sequence = old.Sequence
	for i, p := range old.Members {
		c := &head.Members[i]
		if !checkpointStepBefore(p.Phase, c.Phase, []string{"planned", "remove-dispatched", "remove-acknowledged", "absent"}) || p.RemoveAcknowledged && !c.RemoveAcknowledged {
			return false
		}
		c.Phase = p.Phase
		c.RemoveAcknowledged = p.RemoveAcknowledged
		if p.Before == nil {
			c.Before = nil
		}
		if p.After == nil {
			c.After = nil
		}
	}
	return equalEvidence(head, old)
}
func replacementContainsCheckpoint(head, old AnchorReplacement) bool {
	if len(head.Members) != len(old.Members) {
		return false
	}
	head = head.Clone()
	head.Sequence = old.Sequence
	for i, p := range old.Members {
		c := &head.Members[i]
		if !checkpointStepBefore(p.Phase, c.Phase, []string{"planned", "create-dispatched", "created", "start-dispatched", "start-acknowledged", "running"}) || p.StartAcknowledged && !c.StartAcknowledged {
			return false
		}
		c.Phase = p.Phase
		c.StartAcknowledged = p.StartAcknowledged
		if p.Container.ID == "" {
			c.Container.ID = ""
		}
		if p.Before == nil {
			c.Before = nil
		}
		if p.Created == nil {
			c.Created = nil
		}
		if p.After == nil {
			c.After = nil
		}
	}
	return equalEvidence(head, old)
}
