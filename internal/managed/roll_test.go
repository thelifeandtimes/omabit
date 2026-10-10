package managed

import (
	"context"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRollRuntimeCycleRequirementSurvivesRestartAndOtherMaintenance(t *testing.T) {
	dir := privateDir(t)
	b := &fakeBackend{}
	_, c, shutdown := testEndpoint(t, dir, b)
	prepareStopped(t, c, "alpha", 18080)
	ctx := context.Background()
	req := protocol.OperationRequest{Kind: "roll", RequestID: "roll", ExpectedRevision: 1}
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "succeeded" {
		t.Fatal("runtime-cycle guard blocked exact replay")
	}
	for _, kind := range []string{"stop", "pack", "meld"} {
		op, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: "after-roll-" + kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		s := waitOperation(t, c, op.ID, "succeeded")
		if !s.Instances[0].RollNeedsRuntime {
			t.Fatalf("%s erased the runtime-cycle requirement", kind)
		}
		_, err = c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "roll", RequestID: "blocked-after-" + kind, ExpectedRevision: 1})
		requireProblem(t, err, "invalid_state")
	}
	for _, kind := range []string{"start", "stop"} {
		op, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: "cycle-" + kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		s := waitOperation(t, c, op.ID, "succeeded")
		if s.Instances[0].RollNeedsRuntime {
			t.Fatal("successful runtime cycle did not release roll admission")
		}
	}
	op, err = c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "roll", RequestID: "next-roll", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
}
