package managed

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestEpochInspectionAdmissionReplayAndCacheInvalidation(t *testing.T) {
	ctx := context.Background()
	dir := privateDir(t)
	b := &fakeBackend{}
	_, c, shutdown := testEndpoint(t, dir, b)
	hello, err := c.Hello(ctx)
	if err != nil || !slices.Contains(hello.Features, "epoch-inventory") {
		t.Fatalf("capability: %+v %v", hello, err)
	}
	if _, err := c.CreateFake(ctx, registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	req := protocol.OperationRequest{Kind: "inspect-epochs", RequestID: "inspect", ExpectedRevision: 1}
	_, err = c.Operate(ctx, "alpha", req)
	requireProblem(t, err, "invalid_state")
	for _, kind := range []string{"start", "stop"} {
		op, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		waitOperation(t, c, op.ID, "succeeded")
		if kind == "start" {
			_, err = c.Operate(ctx, "alpha", req)
			requireProblem(t, err, "invalid_state")
		}
	}
	stale := req
	stale.ExpectedRevision = 2
	_, err = c.Operate(ctx, "alpha", stale)
	requireProblem(t, err, "revision_conflict")
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := waitOperation(t, c, op.ID, "succeeded")
	inv := snapshot.Instances[0].Epochs
	if inv == nil || snapshot.Instances[0].AppliedRevision != 1 || snapshot.Instances[0].ObservedState != "stopped" {
		t.Fatalf("inspection result: %+v", snapshot.Instances[0])
	}
	if _, err := epochs.Plan(inv, "0i0", inv.Digest, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "succeeded" {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	snapshot, err = c.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(inv, snapshot.Instances[0].Epochs) {
		t.Fatal("inventory lost or regenerated across restart")
	}
	config := protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18080, AmesPort: 34343, Loom: 31}}
	unchanged, err := c.Configure(ctx, "alpha", config)
	if err != nil || unchanged.Epochs == nil {
		t.Fatal("no-op configuration discarded inventory")
	}
	config.Config.HTTPPort = 18081
	changed, err := c.Configure(ctx, "alpha", config)
	if err != nil || changed.ConfigRevision != 2 || changed.Epochs != nil {
		t.Fatal("configuration change retained preview")
	}
	if replay, err := c.Operate(ctx, "alpha", req); err != nil || replay.ID != op.ID {
		t.Fatal("old revision exact replay failed")
	}
	snapshot, err = c.Snapshot(ctx)
	if err != nil || snapshot.Instances[0].Epochs != nil {
		t.Fatal("replay resurrected stale inventory")
	}
	req.RequestID, req.ExpectedRevision = "inspect-again", 2
	op, err = c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, op.ID, "succeeded")
	start, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "start", RequestID: "resume", ExpectedRevision: 2})
	if err != nil {
		t.Fatal(err)
	}
	snapshot = waitOperation(t, c, start.ID, "succeeded")
	if snapshot.Instances[0].Epochs != nil {
		t.Fatal("resume retained epoch inventory")
	}
	_, err = c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "prune-epochs", RequestID: "prune", ExpectedRevision: 2})
	requireProblem(t, err, "invalid_request")
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 5 {
		t.Fatalf("inspection replay executed again: %d calls", calls)
	}
}

func TestLateEpochObservationCannotUndoConfigurationInvalidation(t *testing.T) {
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.Epochs = &epochs.Inventory{Source: epochs.Source, ObservedAt: time.Now().UTC()}
	if _, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}}); err != nil {
		t.Fatal(err)
	}
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}); err != nil {
		t.Fatal(err)
	}
	actual, err := loadInstance(s.db, "alpha")
	if err != nil || actual.Runtime.Epochs != nil || actual.Public.Epochs != nil {
		t.Fatal("late poll restored invalidated inventory")
	}
}
