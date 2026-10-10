package managed

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const testBootID = "00000000-0000-0000-0000-000000000001"

func saveStartupFixture(t *testing.T, s *store, v storedInstance, operations ...protocol.Operation) {
	t.Helper()
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, op := range operations {
		if err := insertOperation(tx, op, "fixture:"+op.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := saveInstance(tx, v); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func seedStartup(t *testing.T, s *store, id string, port uint16) storedInstance {
	t.Helper()
	if _, _, err := s.register("operator", registration(id, "register-"+id, port)); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	v.Public.DesiredState, v.Public.ObservedState, v.Public.Health = "running", "running", "healthy"
	v.Public.AppliedRevision, v.RuntimeConfigRevision = 1, 1
	v.Runtime.Prepared, v.Runtime.Initialized = true, true
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "http-restart", ID: strings.Repeat("c", 64), CreateDispatched: true, HostHTTPVerified: true}}
	v.Runtime.Runtime.Network = &runtimequalify.NetworkRecord{ID: strings.Repeat("d", 64), CreateDispatched: true}
	saveStartupFixture(t, s, v)
	return v
}

type startupBackend struct {
	mu                sync.Mutex
	running           map[string]bool
	calls             int
	active, maxActive int
	fail              bool
	unavailable       bool
	entered           chan struct{}
	release           chan struct{}
}

func (b *startupBackend) Observe(ctx context.Context, state runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	b.mu.Lock()
	running, unavailable := b.running[state.ID], b.unavailable
	b.mu.Unlock()
	if unavailable {
		return state, runtimequalify.ManagedObservation{}, errors.New("engine unavailable")
	}
	state.Runtime.Containers = append([]runtimequalify.ContainerRecord(nil), state.Runtime.Containers...)
	for i := range state.Runtime.Containers {
		state.Runtime.Containers[i].LastState = &runtimequalify.ContainerState{ObservedAt: time.Now().UTC(), Running: running}
	}
	if running {
		return state, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, nil
	}
	return state, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown", RecoveryRequired: true}, nil
}

func (b *startupBackend) Apply(ctx context.Context, action string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if action != "startup-resume" {
		return state, runtimequalify.ManagedObservation{}, errors.New("unexpected action")
	}
	b.mu.Lock()
	b.calls++
	b.active++
	if b.active > b.maxActive {
		b.maxActive = b.active
	}
	fail := b.fail
	b.mu.Unlock()
	defer func() { b.mu.Lock(); b.active--; b.mu.Unlock() }()
	if b.entered != nil {
		b.entered <- struct{}{}
	}
	if b.release != nil {
		select {
		case <-ctx.Done():
			return state, runtimequalify.ManagedObservation{}, ctx.Err()
		case <-b.release:
		}
	}
	if fail {
		return state, runtimequalify.ManagedObservation{}, errors.New("resume failed")
	}
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	b.mu.Lock()
	if b.running == nil {
		b.running = map[string]bool{}
	}
	b.running[state.ID] = true
	b.mu.Unlock()
	return state, runtimequalify.ManagedObservation{State: "running", Health: "healthy"}, nil
}

func startupOperations(t *testing.T, r *Server, count int, status string) []protocol.Operation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := r.s.snapshot()
		if err != nil {
			t.Fatal(err)
		}
		var matches []protocol.Operation
		for _, op := range s.Operations {
			if op.Kind == "startup-resume" && op.Status == status {
				matches = append(matches, op)
			}
		}
		if len(matches) == count {
			return matches
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("startup operations did not finish", count, status)
	return nil
}

func TestStartupResumeOncePerBootAndExplicitStops(t *testing.T) {
	for _, newBoot := range []bool{false, true} {
		dir := privateDir(t)
		s, err := testOpenStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		v := seedStartup(t, s, "alpha", 18080)
		stopped := seedStartup(t, s, "stopped", 18081)
		stopped.Public.DesiredState = "stopped"
		saveStartupFixture(t, s, stopped)
		id := s.runnerID
		if err := s.close(); err != nil {
			t.Fatal(err)
		}
		b := &startupBackend{}
		r, err := newServerWithStartup(dir, b, 5*time.Millisecond, true, testBootID)
		if err != nil {
			t.Fatal(err)
		}
		op := startupOperations(t, r, 1, "succeeded")[0]
		if op.InstanceID != "alpha" || op.Actor != startupActor || op.ConfigRevision != 1 || r.s.runnerID != id {
			t.Fatal("startup attribution/identity", op)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		b.mu.Lock()
		b.running[v.Runtime.ID] = false
		b.mu.Unlock()
		boot := testBootID
		if newBoot {
			boot = "00000000-0000-0000-0000-000000000002"
		}
		r, err = newServerWithStartup(dir, b, 5*time.Millisecond, true, boot)
		if err != nil {
			t.Fatal(err)
		}
		if newBoot {
			startupOperations(t, r, 2, "succeeded")
		} else {
			time.Sleep(60 * time.Millisecond)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		b.mu.Lock()
		calls := b.calls
		b.mu.Unlock()
		want := 1
		if newBoot {
			want = 2
		}
		if calls != want {
			t.Fatal("startup repeated or started an intentionally stopped ship", calls, want)
		}
	}
}

func TestStartupSkipsInterruptedUncertainAndUnappliedInstances(t *testing.T) {
	s := testStore(t)
	for i, fault := range []string{"recovery", "maintenance", "uninitialized", "unapplied", "interrupted"} {
		v := seedStartup(t, s, fault, uint16(18080+i))
		var operations []protocol.Operation
		switch fault {
		case "recovery":
			v.Public.RecoveryRequired = true
		case "maintenance":
			v.Runtime.MaintenanceUncertain = true
		case "uninitialized":
			v.Runtime.Initialized = false
		case "unapplied":
			v.Public.ConfigRevision++
		case "interrupted":
			op := protocol.Operation{ID: "op-interrupted", InstanceID: v.Public.ID, Actor: "operator", RequestID: "interrupted", Kind: "start", Status: "running", ConfigRevision: 1, CreatedAt: time.Now().UTC()}
			operations = append(operations, op)
			v.Public.ActiveOperationID = op.ID
		}
		saveStartupFixture(t, s, v, operations...)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	if candidates, err := s.startupCandidates(); err != nil || len(candidates) != 0 {
		t.Fatal("unsafe startup candidates", candidates, err)
	}
}

func TestStartupClaimRejectsStaleObservationAndDoesNotRetryFailure(t *testing.T) {
	s := testStore(t)
	v := seedStartup(t, s, "alpha", 18080)
	b := &startupBackend{}
	state, observation, err := b.Observe(context.Background(), v.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.observe(v, state, observation); err != nil {
		t.Fatal(err)
	}
	if _, queued, err := s.claimStartup(v, state, observation, testBootID); err != nil || queued {
		t.Fatal("stale observation admitted", err)
	}
	v, err = loadInstance(s.db, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	op, queued, err := s.claimStartup(v, state, observation, testBootID)
	if err != nil || !queued {
		t.Fatal("startup claim", err)
	}
	claimed, err := loadInstance(s.db, "alpha")
	if err != nil || claimed.Public.RecoveryRequired || claimed.Public.ActiveOperationID != op.ID {
		t.Fatal("verified startup still presented as unresolved recovery", err)
	}
	op, _, err = s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.finish(op, state, observation, errors.New("failed"), false); err != nil {
		t.Fatal(err)
	}
	replayed, queued, err := s.claimStartup(v, state, observation, testBootID)
	if err != nil || queued || replayed.ID != op.ID || replayed.Status != "failed" {
		t.Fatal("startup failure retried", replayed, err)
	}
}

func TestStartupWaitsForEngineAndSerializesWork(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedStartup(t, s, "alpha", 18080)
	seedStartup(t, s, "bravo", 18081)
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	b := &startupBackend{unavailable: true, entered: make(chan struct{}, 2), release: make(chan struct{})}
	r, err := newServerWithStartup(dir, b, 5*time.Millisecond, true, testBootID)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	time.Sleep(30 * time.Millisecond)
	b.mu.Lock()
	before := b.calls
	b.unavailable = false
	b.mu.Unlock()
	if before != 0 {
		t.Fatal("work dispatched while engine unavailable")
	}
	select {
	case <-b.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("startup did not recover engine availability")
	}
	close(b.release)
	startupOperations(t, r, 2, "succeeded")
	b.mu.Lock()
	max := b.maxActive
	b.mu.Unlock()
	if max != 1 {
		t.Fatal("startup bypassed host work exclusion", max)
	}
}

func TestStartupPolicyRequiresOptInAndIsNotAnRPCOperation(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	seedStartup(t, s, "alpha", 18080)
	if _, _, err := s.accept("operator", "alpha", protocol.OperationRequest{Kind: "startup-resume", RequestID: "manual", ExpectedRevision: 1}); err == nil {
		t.Fatal("internal startup operation exposed to RPC")
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	b := &startupBackend{}
	r, err := newServerMode(dir, b, 5*time.Millisecond, true)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	calls := b.calls
	b.mu.Unlock()
	if calls != 0 {
		t.Fatal("startup policy enabled implicitly")
	}
}

func TestStartupClaimHonorsConcurrentOperatorStop(t *testing.T) {
	s := testStore(t)
	v := seedStartup(t, s, "alpha", 18080)
	b := &startupBackend{}
	state, observation, err := b.Observe(context.Background(), v.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.accept("operator", "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop-during-startup", ExpectedRevision: 1}); err != nil {
		t.Fatal(err)
	}
	if _, queued, err := s.claimStartup(v, state, observation, testBootID); err != nil || queued {
		t.Fatal("startup raced the operator stop", err)
	}
}
