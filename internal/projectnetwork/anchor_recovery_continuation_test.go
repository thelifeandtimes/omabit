package projectnetwork

import (
	"reflect"
	"testing"
)

func TestRecoveryContinuationSeparatesNewActionsFromUncertainOutcomes(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		h, ready := journalReadinessFixture(t, stopped)
		for _, ack := range []bool{false, true} {
			trace := replayRecoveryJournal(t, h, ready, ack)
			kinds := map[string]int{}
			for _, j := range trace {
				before := j.Clone()
				target, continuationErr := RecoveryCheckpointContinuationTarget(h, j)
				_, observationErr := RecoveryCheckpointObservationTarget(h, j)
				if continuationErr == nil && observationErr == nil {
					t.Fatal("checkpoint permits both observation and new effects", j.Phase)
				}
				excluded := j.Phase == "review-planned" || j.Phase == "reviewing" || j.Phase == "complete"
				if j.Replacement != nil && !j.Replacement.Running() && j.Replacement.Members[j.Replacement.next()].Phase == "create-dispatched" {
					excluded = true
				}
				if excluded {
					if continuationErr == nil || observationErr == nil {
						t.Fatal("unresolved Create or finished/unreviewed work admitted", j.Phase)
					}
				} else if continuationErr != nil && observationErr != nil {
					t.Fatal("supported checkpoint has no explicit next path", j.Phase, j.Sequence)
				}
				if continuationErr == nil {
					kinds[target.Kind]++
				}
				if !reflect.DeepEqual(j, before) {
					t.Fatal("classification changed retained evidence")
				}
			}
			for _, kind := range []string{"begin-drain", "stop-member", "establish-namespace", "begin-retirement", "retire-member", "plan-replacements", "create-member", "start-created-member"} {
				if kinds[kind] == 0 {
					t.Fatal("missing continuation boundary", kind)
				}
			}
			changed := h.Clone()
			changed.Plan.Members[0].HTTPPort++
			if _, err := RecoveryCheckpointContinuationTarget(changed, trace[2]); err == nil {
				t.Fatal("changed source admitted")
			}
		}
	}
}
