package protocol

import (
	"strings"
	"testing"
)

func TestProjectActionRequiredPrerequisiteFields(t *testing.T) {
	for _, kind := range []string{"prepare", "start-anchor", "prepare-storage", "launch", "observe", "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current"} {
		t.Run(kind, func(t *testing.T) {
			_, _, member := ProjectAction(kind)
			q := ProjectActionRequest{Kind: kind, RequestID: "request", ExpectedRevision: 1, ReviewDigest: strings.Repeat("a", 64), ConfirmAction: true}
			if member {
				q.InstanceID = "member"
			}
			after := false
			switch kind {
			case "stop", "observe-exit", "retire", "resume", "observe-resumed", "observe-current":
				after = true
				q.AfterOperationID = "prerequisite"
			}
			if kind == "observe-current" {
				q.OriginalReadinessOperationID = "original"
			}
			if !ValidProjectActionRequest(q) {
				t.Fatal(q)
			}
			changed := q
			if after {
				changed.AfterOperationID = ""
			} else {
				changed.AfterOperationID = "irrelevant"
			}
			if ValidProjectActionRequest(changed) {
				t.Fatal("incorrect prerequisite accepted")
			}
			changed = q
			if kind == "observe-current" {
				changed.OriginalReadinessOperationID = ""
			} else {
				changed.OriginalReadinessOperationID = "irrelevant"
			}
			if ValidProjectActionRequest(changed) {
				t.Fatal("incorrect original readiness accepted")
			}
			changed = q
			changed.ConfirmAction = false
			if ValidProjectActionRequest(changed) {
				t.Fatal("unconfirmed action accepted")
			}
		})
	}
}
