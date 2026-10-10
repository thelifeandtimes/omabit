package managed

import (
	"context"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"os"
	"testing"
	"time"
)

func routineFixture(t *testing.T) (*store, routines.Policy, time.Time) {
	t.Helper()
	s := testStore(t)
	if _, _, err := s.register("actor", registration("alpha", "reg", 18081)); err != nil {
		t.Fatal(err)
	}
	v, _ := loadInstance(s.db, "alpha")
	v.Runtime.Initialized = true
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	req := routines.Update{ExpectedInstanceRevision: 1, Config: routines.Config{Enabled: true, Action: "pack", Weekdays: []int{int(now.Weekday())}, HourUTC: now.Hour(), MinuteUTC: now.Minute(), WindowMinutes: 5}}
	p, err := s.putRoutine("actor", "alpha", req)
	if err != nil {
		t.Fatal(err)
	}
	p.UpdatedAt = now.Truncate(time.Minute).Add(-time.Second)
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := saveRoutinePolicy(tx, p); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return s, p, now
}

type routineBackendFixture struct{ fakeBackend }

func (b *routineBackendFixture) Apply(ctx context.Context, kind string, state runtimequalify.ManagedState, persist func(runtimequalify.ManagedState) error) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	if kind != "routine-pack" {
		return b.fakeBackend.Apply(ctx, kind, state, persist)
	}
	now := time.Now().UTC()
	state.Routine.StartedAt = &now
	state.Routine.FinishedAt = &now
	state.Routine.Stage = "complete"
	state.Routine.Succeeded = true
	if err := persist(state); err != nil {
		return state, runtimequalify.ManagedObservation{}, err
	}
	return state, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil
}

func TestDispatcherExecutesClaimedRoutineExactlyOnce(t *testing.T) {
	s, p, now := routineFixture(t)
	p.Owner = fmt.Sprintf("local-uid:%d", os.Geteuid())
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := saveRoutinePolicy(tx, p); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Server{s: s, b: &routineBackendFixture{}, ctx: ctx, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
	defer r.Close()
	r.dispatchRoutines(now)
	deadline := time.Now().Add(2 * time.Second)
	for {
		v, err := loadInstance(s.db, "alpha")
		if err != nil {
			t.Fatal(err)
		}
		if v.Public.ActiveOperationID == "" && v.Public.Routine != nil && v.Public.Routine.Succeeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduled worker did not complete")
		}
		time.Sleep(time.Millisecond)
	}
	r.dispatchRoutines(now.Add(time.Second))
	snapshot, err := s.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, op := range snapshot.Operations {
		if op.Kind == "routine-pack" {
			count++
			if op.Status != "succeeded" || op.Actor != p.Owner {
				t.Fatal("wrong routine outcome")
			}
		}
	}
	if count != 1 || snapshot.Instances[0].Schedule.LastOperationID == "" {
		t.Fatal("routine duplicated or metadata missing")
	}
}

func TestRoutineSlotClaimIsDurableAndNotRepeated(t *testing.T) {
	s, p, now := routineFixture(t)
	op, queued, err := s.claimRoutine("alpha", now, func(string) bool { return true })
	if err != nil || !queued || op.Kind != "routine-pack" {
		t.Fatal("slot admission", err)
	}
	if _, again, err := s.claimRoutine("alpha", now, func(string) bool { return true }); err != nil || again {
		t.Fatal("slot replayed")
	}
	v, _ := loadInstance(s.db, "alpha")
	if v.Public.ActiveOperationID != op.ID || v.Runtime.Routine.PolicyID != p.ID {
		t.Fatal("routine not bound to operation")
	}
	if _, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "start", RequestID: "race", ExpectedRevision: 1}); err == nil {
		t.Fatal("manual operation bypassed routine exclusion")
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	if _, again, err := s.claimRoutine("alpha", now.Add(time.Second), func(string) bool { return true }); err != nil || again {
		t.Fatal("restart re-executed interrupted routine")
	}
	v, _ = loadInstance(s.db, "alpha")
	if v.Public.Routine.Stage != "interrupted" || v.Public.Routine.Succeeded {
		t.Fatal("interrupted routine lost")
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal("policy downgrade guard missing")
	}
}

func TestRoutineSkipsRevocationDriftAndMissedWindow(t *testing.T) {
	for _, reason := range []string{"revoked", "drift", "missed"} {
		t.Run(reason, func(t *testing.T) {
			s, _, now := routineFixture(t)
			authorized := reason != "revoked"
			if reason == "drift" {
				v, _ := loadInstance(s.db, "alpha")
				cfg := v.Public.Config
				cfg.HTTPPort++
				if _, err := s.configure("actor", "alpha", protocol.ConfigureRequest{ExpectedRevision: 1, Config: cfg}); err != nil {
					t.Fatal(err)
				}
			}
			if reason == "missed" {
				now = now.Add(10 * time.Minute)
			}
			op, queued, err := s.claimRoutine("alpha", now, func(string) bool { return authorized })
			if err != nil || queued || op.Status != "skipped" || op.Message == "" {
				t.Fatal("unsafe policy dispatched", err)
			}
			if _, again, err := s.claimRoutine("alpha", now, func(string) bool { return true }); err != nil || again {
				t.Fatal("skipped slot retried")
			}
		})
	}
}

func TestPolicyCompareAndSwapAndActorAttribution(t *testing.T) {
	s, p, _ := routineFixture(t)
	req := routines.Update{ExpectedRevision: 0, ExpectedInstanceRevision: 1, Config: p.Config}
	if _, err := s.putRoutine("other", "alpha", req); err == nil {
		t.Fatal("stale policy write accepted")
	}
	req.ExpectedRevision = p.Revision
	req.Config.Enabled = false
	updated, err := s.putRoutine("other", "alpha", req)
	if err != nil || updated.Owner != "other" || updated.Revision != p.Revision+1 || updated.NextSlot != nil {
		t.Fatal("policy update failed", err)
	}
}
