package protocol

import (
	"strings"
	"testing"
)

func TestPublicRecoveryActionExactPathsAndFields(t *testing.T) {
	for _, kind := range []string{"start", "observe", "continue"} {
		parent := ""
		r := ProjectRecoveryActionRequest{RequestID: "request", ExpectedRevision: 1, ReviewDigest: strings.Repeat("a", 64), HistoryDigest: strings.Repeat("b", 64), ConfirmAction: true}
		if kind != "start" {
			parent = "parent"
			seq := uint64(0)
			r.ExpectedSequence = &seq
			r.CheckpointDigest = strings.Repeat("c", 64)
		}
		path := ProjectRecoveryActionPath("project", parent, kind)
		p, o, k, ok := ProjectRecoveryActionIDs(path)
		if !ok || p != "project" || o != parent || k != kind || !ValidProjectRecoveryAction(kind, parent, r) {
			t.Fatal(path, r)
		}
		r.ConfirmAction = false
		if ValidProjectRecoveryAction(kind, parent, r) {
			t.Fatal("unconfirmed")
		}
		r.ConfirmAction = true
		if kind == "start" {
			seq := uint64(0)
			r.ExpectedSequence = &seq
		} else {
			r.ExpectedSequence = nil
		}
		if ValidProjectRecoveryAction(kind, parent, r) {
			t.Fatal("missing/irrelevant sequence")
		}
	}
	for _, path := range []string{"/v1/projects//recovery", "/v1/projects/p/recovery/o", "/v1/projects/p/recovery/o/observe", "/v1/projects/p/recovery/o/observations/extra", "/v1/projects/p/q/recovery"} {
		if _, _, _, ok := ProjectRecoveryActionIDs(path); ok {
			t.Fatal(path)
		}
	}
}
