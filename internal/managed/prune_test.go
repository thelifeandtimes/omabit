package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func genesisInventory() *epochs.Inventory {
	return &epochs.Inventory{Source: epochs.Source, ObservedAt: time.Now().UTC(), Digest: strings.Repeat("d", 64), Entries: []epochs.Entry{
		{Name: "0i0", Digest: strings.Repeat("0", 64), FileCount: 4},
		{Name: "0i100", Digest: strings.Repeat("1", 64), FileCount: 5, SnapshotBytes: 100},
		{Name: "0i200", Digest: strings.Repeat("2", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
		{Name: "0i300", Digest: strings.Repeat("3", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
	}}
}

func TestManagedPruneConfirmationReplayAndResume(t *testing.T) {
	dir := privateDir(t)
	b := &fakeBackend{epochInventory: genesisInventory()}
	_, c, shutdown := testEndpoint(t, dir, b)
	prepareStopped(t, c, "alpha", 18080)
	ctx := context.Background()
	req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "prune", ExpectedRevision: 1, EpochCutoff: "0i0", InventoryDigest: b.epochInventory.Digest, ConfirmHistoryLoss: true}
	_, err := c.Operate(ctx, "alpha", req)
	requireProblem(t, err, "stale_inventory")
	inspect, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "inspect-epochs", RequestID: "inspect", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, inspect.ID, "succeeded")
	bad := req
	bad.ConfirmHistoryLoss = false
	_, err = c.Operate(ctx, "alpha", bad)
	requireProblem(t, err, "invalid_request")
	bad = req
	bad.InventoryDigest = strings.Repeat("a", 64)
	_, err = c.Operate(ctx, "alpha", bad)
	requireProblem(t, err, "stale_or_unqualified_inventory")
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	s := waitOperation(t, c, op.ID, "succeeded")
	v := s.Instances[0]
	if v.ObservedState != "stopped" || v.AppliedRevision != 1 || v.Pruning == nil || v.Pruning.Uncertain || !v.Pruning.EpochRemoved || v.Pruning.RemovedFiles != 4 || len(v.Pruning.RetainedEpochs) != 3 || v.Epochs == nil || len(v.Epochs.Entries) != 3 {
		t.Fatalf("pruning evidence missing: %+v", v)
	}
	original, err := c.Operate(ctx, "alpha", req)
	if err != nil || original.Status != "succeeded" {
		t.Fatal("completed replay failed")
	}
	bad = req
	bad.InventoryDigest = strings.Repeat("a", 64)
	_, err = c.Operate(ctx, "alpha", bad)
	requireProblem(t, err, "request_conflict")
	start, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "start", RequestID: "resume", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, start.ID, "succeeded")
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || !reflect.DeepEqual(replay, original) {
		t.Fatalf("prune replay changed after restart/resume: %+v %v", replay, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 5 {
		t.Fatalf("duplicate prune dispatched: %d backend calls", calls)
	}
}

func TestManagedPruneInterruptionStaysQuarantined(t *testing.T) {
	dir := privateDir(t)
	b := &fakeBackend{epochInventory: genesisInventory()}
	_, c, shutdown := testEndpoint(t, dir, b)
	prepareStopped(t, c, "alpha", 18080)
	ctx := context.Background()
	inspect, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "inspect-epochs", RequestID: "inspect", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, inspect.ID, "succeeded")
	b.entered, b.release = make(chan struct{}, 2), make(chan struct{})
	req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "interrupted-prune", ExpectedRevision: 1, EpochCutoff: "0i0", InventoryDigest: b.epochInventory.Digest, ConfirmHistoryLoss: true}
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("prune did not start")
	}
	shutdown()
	close(b.release)
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "interrupted" {
		t.Fatalf("interrupted replay: %+v %v", replay, err)
	}
	stop, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "cleanup", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	s := waitOperation(t, c, stop.ID, "succeeded")
	if !s.Instances[0].RecoveryRequired || s.Instances[0].Pruning == nil || !s.Instances[0].Pruning.Uncertain {
		t.Fatal("stop hid partial deletion")
	}
	for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs"} {
		_, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: "blocked-" + kind, ExpectedRevision: 1})
		requireProblem(t, err, "recovery_required")
	}
	req.RequestID = "new-prune"
	_, err = c.Operate(ctx, "alpha", req)
	requireProblem(t, err, "recovery_required")
}

func TestPruneWaitsForHostGateAndPreservesIntent(t *testing.T) {
	b := &fakeBackend{epochInventory: genesisInventory()}
	r, c, shutdown := testEndpoint(t, privateDir(t), b)
	defer shutdown()
	prepareStopped(t, c, "alpha", 18080)
	ctx := context.Background()
	inspect, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "inspect-epochs", RequestID: "inspect", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, inspect.ID, "succeeded")
	r.workGate <- struct{}{}
	req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "queued-prune", ExpectedRevision: 1, EpochCutoff: "0i0", InventoryDigest: b.epochInventory.Digest, ConfirmHistoryLoss: true}
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "queued" {
		t.Fatal("prune bypassed host gate")
	}
	v, err := loadInstance(r.s.db, "alpha")
	if err != nil || v.Runtime.PruneIntent == nil || !v.Runtime.PruneIntent.Confirmed || v.Runtime.PruneIntent.InventoryDigest != req.InventoryDigest {
		t.Fatal("queued prune lost its bound intent")
	}
	_, err = c.Configure(ctx, "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: protocol.Configuration{HTTPPort: 18081, AmesPort: 34343, Loom: 31}})
	requireProblem(t, err, "instance_busy")
	<-r.workGate
	waitOperation(t, c, op.ID, "succeeded")
}

func TestRepeatedManagedPrefixReplayAndHistoricalEvidence(t *testing.T) {
	inv := genesisInventory()
	inv.Entries[2].Protected = false
	inv.Entries = append(inv.Entries, epochs.Entry{Name: "0i400", Digest: strings.Repeat("4", 64), FileCount: 5, SnapshotBytes: 100, Protected: true})
	dir := privateDir(t)
	b := &fakeBackend{epochInventory: inv}
	r, c, shutdown := testEndpoint(t, dir, b)
	prepareStopped(t, c, "alpha", 18080)
	ctx := context.Background()
	var requests []protocol.OperationRequest
	var originals []protocol.Operation
	for i, cutoff := range []string{"0i100", "0i200"} {
		inspect, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "inspect-epochs", RequestID: "inspect-" + cutoff, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		s := waitOperation(t, c, inspect.ID, "succeeded")
		req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "prune-" + cutoff, ExpectedRevision: 1, EpochCutoff: cutoff, InventoryDigest: s.Instances[0].Epochs.Digest, ConfirmHistoryLoss: true}
		op, err := c.Operate(ctx, "alpha", req)
		if err != nil {
			t.Fatal(err)
		}
		s = waitOperation(t, c, op.ID, "succeeded")
		p := s.Instances[0].Pruning
		selected, remaining := 2, 3
		if i == 1 {
			selected, remaining = 1, 2
		}
		if p == nil || p.Uncertain || len(p.SelectedEpochs) != selected || len(p.RemovedEpochs) != selected || len(p.RetainedEpochs) != remaining || !p.EpochRemoved {
			t.Fatalf("prefix report: %+v", p)
		}
		original, err := c.Operate(ctx, "alpha", req)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, req)
		originals = append(originals, original)
		for _, kind := range []string{"start", "stop"} {
			op, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: kind + "-" + cutoff, ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			waitOperation(t, c, op.ID, "succeeded")
		}
	}
	var archived []byte
	if err := r.s.db.QueryRow("SELECT runtime FROM operations WHERE id=?", originals[0].ID).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	var state runtimequalify.ManagedState
	if err := json.Unmarshal(archived, &state); err != nil || state.PrefixPrune == nil || state.PrefixPrune.Cutoff != "0i100" || state.PrefixPrune.Uncertain || len(state.PrefixPrune.After.Entries) != 3 {
		t.Fatal("later prune overwrote historical runtime evidence")
	}
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	for i, req := range requests {
		op, err := c.Operate(ctx, "alpha", req)
		if err != nil || !reflect.DeepEqual(op, originals[i]) {
			t.Fatal("historical prefix replay changed after later pruning/restart")
		}
	}
	s, err := c.Snapshot(ctx)
	if err != nil || s.Instances[0].Pruning.Cutoff != "0i200" || len(s.Instances[0].Pruning.RetainedEpochs) != 2 {
		t.Fatal("latest pruning report was not preserved")
	}
}
