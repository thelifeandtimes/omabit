//go:build linux && amd64

package managed

import (
	"context"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectStatusContainsPersistedRecoveryAndObservation(t *testing.T) {
	r, b, p, parent := prepareRecoveryContinuation(t, "signal")
	before := len(b.calls)
	status, err := r.s.projectStatus(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]protocol.ProjectOperationSummary{}
	for _, op := range status.Operations {
		seen[op.ID] = op
	}
	if seen[parent.OperationID].Status != "interrupted" {
		t.Fatal("lost recovery interruption", status.Operations)
	}
	observer := false
	for _, op := range status.Operations {
		if op.Kind == "observe-project-recovery" && op.Status == "succeeded" {
			observer = true
		}
	}
	if !observer || before != len(b.calls) {
		t.Fatal("lost observation or dispatched effects")
	}
}
