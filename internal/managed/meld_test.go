package managed

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func prepareStopped(t *testing.T, c *client.Client, id string, port uint16) {
	t.Helper()
	ctx := context.Background()
	if _, err := c.CreateFake(ctx, registration(id, "reg-"+id, port)); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"start", "stop"} {
		op, err := c.Operate(ctx, id, protocol.OperationRequest{Kind: kind, RequestID: kind + "-" + id, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		waitOperation(t, c, op.ID, "succeeded")
	}
}

func requireProblem(t *testing.T, err error, code string) {
	t.Helper()
	var remote *client.RemoteError
	if !errors.As(err, &remote) || remote.Problem.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestMeldAdmissionReportAndPersistentReplay(t *testing.T) {
	for _, kind := range []string{"meld", "roll"} {
		t.Run(kind, func(t *testing.T) { testMaintenanceAdmissionReportAndReplay(t, kind) })
	}
}

func testMaintenanceAdmissionReportAndReplay(t *testing.T, kind string) {
	ctx := context.Background()
	dir := privateDir(t)
	b := &fakeBackend{}
	_, c, shutdown := testEndpoint(t, dir, b)
	hello, err := c.Hello(ctx)
	if err != nil || !slices.Contains(hello.Features, "offline-"+kind) || !slices.Contains(hello.Features, "offline-pack") {
		t.Fatalf("missing capability: %+v %v", hello, err)
	}
	if _, err := c.CreateFake(ctx, registration("alpha", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	req := protocol.OperationRequest{Kind: kind, RequestID: kind + "-alpha", ExpectedRevision: 1}
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
	v := snapshot.Instances[0]
	if v.DesiredState != "stopped" || v.ObservedState != "stopped" || v.AppliedRevision != 1 || v.ConfigRevision != 1 || v.Maintenance == nil || v.Maintenance.Kind != kind || v.Maintenance.FinishedAt == nil || v.Maintenance.MemAvailableBytes != 6<<30 || v.Maintenance.AfterPierBytes != 500 {
		t.Fatalf("maintenance report/revision missing: %+v", v)
	}
	if kind == "roll" && (v.Maintenance.CreatedEpoch != "0i100" || len(v.Maintenance.EpochsBefore) != 1 || len(v.Maintenance.EpochsAfter) != 2 || v.Epochs == nil) {
		t.Fatal("roll report or fresh inventory missing")
	}
	if kind == "roll" {
		if !v.RollNeedsRuntime {
			t.Fatal("completed roll omitted runtime-cycle requirement")
		}
		_, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "roll", RequestID: "duplicate-new-roll", ExpectedRevision: 1})
		requireProblem(t, err, "invalid_state")
	}
	original, err := c.Operate(ctx, "alpha", req)
	if err != nil || original.ID != op.ID || original.Status != "succeeded" || original.Actor != hello.Actor {
		t.Fatalf("completed replay/actor: %+v %v", original, err)
	}
	changed := req
	changed.Kind = "pack"
	_, err = c.Operate(ctx, "alpha", changed)
	requireProblem(t, err, "request_conflict")
	resume, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "start", RequestID: "resume", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	waitOperation(t, c, resume.ID, "succeeded")
	shutdown()
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	// Replay still wins over running-state admission after management restart.
	replayed, err := c.Operate(ctx, "alpha", req)
	if err != nil || !reflect.DeepEqual(replayed, original) {
		t.Fatalf("durable meld replay changed: %+v %v", replayed, err)
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 4 {
		t.Fatalf("expected start, stop, meld, resume once each; got %d backend calls", calls)
	}
}

func TestMeldSharesHostGateWithStartsAndPack(t *testing.T) {
	for _, pair := range [][2]string{{"meld", "start"}, {"meld", "pack"}, {"meld", "meld"}, {"start", "meld"}, {"pack", "meld"}, {"roll", "start"}, {"roll", "pack"}, {"roll", "meld"}, {"roll", "roll"}, {"start", "roll"}, {"pack", "roll"}, {"meld", "roll"}} {
		t.Run(pair[0]+"/"+pair[1], func(t *testing.T) {
			b := &fakeBackend{}
			_, c, shutdown := testEndpoint(t, privateDir(t), b)
			defer shutdown()
			prepareStopped(t, c, "alpha", 18080)
			prepareStopped(t, c, "bravo", 18081)
			b.entered, b.release = make(chan struct{}, 2), make(chan struct{})
			ctx := context.Background()
			first, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: pair[0], RequestID: "first", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-b.entered:
			case <-time.After(time.Second):
				t.Fatal("first job did not enter backend")
			}
			req := protocol.OperationRequest{Kind: pair[1], RequestID: "second", ExpectedRevision: 1}
			second, err := c.Operate(ctx, "bravo", req)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := c.Operate(ctx, "bravo", req)
			if err != nil || replay.ID != second.ID || replay.Status != "queued" {
				t.Fatalf("queued replay changed: %+v %v", replay, err)
			}
			_, err = c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "conflicting-stop", ExpectedRevision: 1})
			requireProblem(t, err, "instance_busy")
			select {
			case <-b.entered:
				t.Fatal("expensive jobs entered backend concurrently")
			case <-time.After(50 * time.Millisecond):
			}
			select {
			case b.release <- struct{}{}:
			case <-time.After(time.Second):
				t.Fatal("first job stopped waiting")
			}
			waitOperation(t, c, first.ID, "succeeded")
			select {
			case <-b.entered:
			case <-time.After(time.Second):
				t.Fatal("queued job did not proceed after gate release")
			}
			close(b.release)
			waitOperation(t, c, second.ID, "succeeded")
		})
	}
}

func TestMeldInterruptionPersistsAcrossManagementRestartAndStop(t *testing.T) {
	for _, kind := range []string{"meld", "roll"} {
		t.Run(kind, func(t *testing.T) { testMaintenanceInterruptionAcrossRestart(t, kind) })
	}
}

func testMaintenanceInterruptionAcrossRestart(t *testing.T, kind string) {
	dir := privateDir(t)
	b := &fakeBackend{}
	_, c, shutdown := testEndpoint(t, dir, b)
	prepareStopped(t, c, "alpha", 18080)
	b.entered, b.release = make(chan struct{}, 2), make(chan struct{})
	ctx := context.Background()
	req := protocol.OperationRequest{Kind: kind, RequestID: "interrupted-" + kind, ExpectedRevision: 1}
	op, err := c.Operate(ctx, "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("meld did not begin")
	}
	shutdown()
	close(b.release)
	_, c, shutdown = testEndpoint(t, dir, b)
	defer shutdown()
	replay, err := c.Operate(ctx, "alpha", req)
	if err != nil || replay.ID != op.ID || replay.Status != "interrupted" {
		t.Fatalf("interrupted meld was not replayed: %+v %v", replay, err)
	}
	stop, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "cleanup", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := waitOperation(t, c, stop.ID, "succeeded")
	if !snapshot.Instances[0].RecoveryRequired || snapshot.Instances[0].Maintenance.FinishedAt != nil {
		t.Fatal("stop cleared the interrupted maintenance requirement")
	}
	for _, kind := range []string{"start", "pack", "meld", "roll"} {
		_, err := c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: kind, RequestID: "blocked-" + kind, ExpectedRevision: 1})
		requireProblem(t, err, "recovery_required")
	}
	b.mu.Lock()
	calls := b.apply
	b.mu.Unlock()
	if calls != 4 {
		t.Fatalf("unexpected backend calls after interruption: %d", calls)
	}
}

func TestUnresolvedMaintenanceBlocksOtherRuntimeWorkUntilCleanup(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll", "comet-pack", "comet-meld", "comet-roll"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			b := &fakeBackend{}
			r, c, shutdown := testEndpoint(t, dir, b)
			prepareStopped(t, c, "alpha", 18080)
			prepareStopped(t, c, "bravo", 18081)
			// Model a daemon-side utility whose worker lost its result. Public
			// state alone does not establish whether the utility has stopped.
			tx, err := r.s.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			v, err := loadInstance(tx, "alpha")
			if err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
			v.Runtime.MaintenanceUncertain = true
			v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: kind, CreateDispatched: true, CreateUncertain: true}}
			v.RuntimeRevision++
			if err := saveInstance(tx, v); err != nil {
				tx.Rollback()
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			shutdown()
			r, c, shutdown = testEndpoint(t, dir, b)
			defer shutdown()
			ctx := context.Background()
			req := protocol.OperationRequest{Kind: "start", RequestID: "blocked-start", ExpectedRevision: 1}
			op, err := c.Operate(ctx, "bravo", req)
			if err != nil {
				t.Fatal(err)
			}
			waitOperation(t, c, op.ID, "failed")
			replay, err := c.Operate(ctx, "bravo", req)
			if err != nil || replay.ID != op.ID || replay.Status != "failed" || !strings.Contains(replay.Message, "retained maintenance resources on alpha") || r.failed.Load() {
				t.Fatalf("host admission outcome: %+v %v", replay, err)
			}
			b.mu.Lock()
			calls := b.apply
			b.mu.Unlock()
			if calls != 4 {
				t.Fatal("backend dispatched work alongside an unresolved utility")
			}
			// Stop is permitted; resources can be resolved while alpha's own pier
			// remains quarantined. Bravo can then run with a new request ID.
			for _, id := range []string{"alpha", "bravo"} {
				op, err := c.Operate(ctx, id, protocol.OperationRequest{Kind: "stop", RequestID: "cleanup-" + id, ExpectedRevision: 1})
				if err != nil {
					t.Fatal(err)
				}
				waitOperation(t, c, op.ID, "succeeded")
			}
			req.RequestID = "allowed-start"
			op, err = c.Operate(ctx, "bravo", req)
			if err != nil {
				t.Fatal(err)
			}
			waitOperation(t, c, op.ID, "succeeded")
			_, err = c.Operate(ctx, "alpha", protocol.OperationRequest{Kind: "meld", RequestID: "still-blocked", ExpectedRevision: 1})
			requireProblem(t, err, "recovery_required")
		})
	}
}
