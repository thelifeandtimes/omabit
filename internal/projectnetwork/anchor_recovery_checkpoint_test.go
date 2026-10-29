package projectnetwork

import (
	"reflect"
	"testing"
	"time"
)

func TestRecoveryJournalCheckpointRetainsEveryEarlierReceipt(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		h, ready := journalReadinessFixture(t, stopped)
		for _, ack := range []bool{false, true} {
			trace := replayRecoveryJournal(t, h, ready, ack)
			head := trace[len(trace)-1]
			before := head.Clone()
			for i, old := range trace {
				if !RecoveryJournalContainsCheckpoint(h, head, old) {
					t.Fatal("lost historical checkpoint", stopped, ack, i, old.Phase)
				}
				if i+1 < len(trace) && RecoveryJournalContainsCheckpoint(h, old, trace[i+1]) {
					t.Fatal("future checkpoint accepted")
				}
			}
			if !reflect.DeepEqual(head, before) {
				t.Fatal("historical comparison rewrote head")
			}
		}
	}
}
func TestRecoveryJournalCheckpointRejectsForkedEvidenceAndAcknowledgements(t *testing.T) {
	h, ready := journalReadinessFixture(t, false)
	ack := replayRecoveryJournal(t, h, ready, true)
	lost := replayRecoveryJournal(t, h, ready, false)
	head := lost[len(lost)-1]
	for _, old := range ack {
		p := recoveryParts(old)
		claimed := false
		if p.drain != nil {
			for _, m := range p.drain.Members {
				claimed = claimed || m.SignalAcknowledged
			}
		}
		if claimed && RecoveryJournalContainsCheckpoint(h, head, old) {
			t.Fatal("invented omitted acknowledgement", old.Phase)
		}
	}
	var checkpoint AnchorRecoveryJournal
	for _, v := range lost {
		if v.Phase == "draining" && v.Drain.Members[0].Phase == "stop-dispatched" {
			checkpoint = v.Clone()
			break
		}
	}
	checkpoint.Drain.Members[0].Before.CapturedAt = checkpoint.Drain.Members[0].Before.CapturedAt.Add(time.Nanosecond)
	if err := checkpoint.Validate(h); err != nil {
		t.Fatal("fork fixture must remain individually valid", err)
	}
	if RecoveryJournalContainsCheckpoint(h, head, checkpoint) {
		t.Fatal("rewritten historical dispatch accepted")
	}
	foreign := h.Clone()
	foreign.Plan.Members[0].HTTPPort++
	if RecoveryJournalContainsCheckpoint(foreign, head, lost[0]) {
		t.Fatal("foreign source accepted")
	}
}
