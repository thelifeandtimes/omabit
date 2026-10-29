package projectnetwork

import "testing"

// Each outer phase must still reject corrupt evidence at every embedded depth.
func TestAnchorRecoveryJournalRejectsCorruptEmbeddedAncestry(t *testing.T) {
	h, ready := journalReadinessFixture(t, true)
	trace := replayRecoveryJournal(t, h, ready, true)
	seen := map[string]bool{}
	for _, valid := range trace {
		if seen[valid.Phase] {
			continue
		}
		seen[valid.Phase] = true
		for _, mutation := range []struct {
			name  string
			apply func(recoveryCheckpointParts) bool
		}{
			{"review-process", func(p recoveryCheckpointParts) bool {
				if p.review == nil {
					return false
				}
				p.review.After.Processes[0].StartedAt = p.review.After.Processes[0].StartedAt.Add(1)
				return true
			}},
			{"drain-sequence", func(p recoveryCheckpointParts) bool {
				if p.drain == nil {
					return false
				}
				p.drain.Sequence++
				return true
			}},
			{"namespace-sequence", func(p recoveryCheckpointParts) bool {
				if p.namespace == nil {
					return false
				}
				p.namespace.Sequence++
				return true
			}},
			{"retirement-sequence", func(p recoveryCheckpointParts) bool {
				if p.retirement == nil {
					return false
				}
				p.retirement.Sequence++
				return true
			}},
			{"replacement-sequence", func(p recoveryCheckpointParts) bool {
				if p.replacement == nil {
					return false
				}
				p.replacement.Sequence++
				return true
			}},
			{"drain-members", func(p recoveryCheckpointParts) bool {
				if p.drain == nil {
					return false
				}
				p.drain.Members = nil
				return true
			}},
			{"namespace-before", func(p recoveryCheckpointParts) bool {
				if p.namespace == nil || p.namespace.Before == nil {
					return false
				}
				p.namespace.Before = nil
				return true
			}},
			{"retirement-members", func(p recoveryCheckpointParts) bool {
				if p.retirement == nil {
					return false
				}
				p.retirement.Members = nil
				return true
			}},
			{"replacement-members", func(p recoveryCheckpointParts) bool {
				if p.replacement == nil {
					return false
				}
				p.replacement.Members = nil
				return true
			}},
		} {
			candidate := valid.Clone()
			if !mutation.apply(recoveryParts(candidate)) {
				continue
			}
			t.Run(valid.Phase+"/"+mutation.name, func(t *testing.T) {
				if candidate.Validate(h) == nil {
					t.Fatal("corrupt embedded receipt accepted")
				}
			})
		}
	}
}
