package managed

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

type compatibilityObservationBackend struct {
	fakeBackend
	problem error
}

func (b *compatibilityObservationBackend) Observe(_ context.Context, s runtimequalify.ManagedState) (runtimequalify.ManagedState, runtimequalify.ManagedObservation, error) {
	return s, runtimequalify.ManagedObservation{}, b.problem
}

func TestHostCompatibilityObservationDoesNotInventPierRecovery(t *testing.T) {
	for _, compatibility := range []bool{false, true} {
		t.Run(fmt.Sprint(compatibility), func(t *testing.T) {
			problem := errors.New("runtime observation genuinely failed")
			if compatibility {
				problem = errors.Join(runtimequalify.ErrHostCompatibility, context.DeadlineExceeded)
			}
			r, c, _ := testEndpoint(t, privateDir(t), &compatibilityObservationBackend{problem: problem})
			if _, err := c.CreateFake(context.Background(), registration("alpha", "create", 18080)); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(3 * time.Second)
			for {
				s, err := c.Snapshot(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				i := s.Instances[0]
				if i.ObservedAt != nil && i.ObservedState == "unknown" {
					if i.RecoveryRequired == compatibility || i.Health != "unknown" || r.failed.Load() {
						t.Fatalf("host evidence failure confused with pier/store failure: %+v", i)
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("observation was not recorded")
				}
				time.Sleep(time.Millisecond)
			}
			op, err := c.Operate(context.Background(), "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1})
			if err != nil {
				t.Fatal("control became unavailable after host report failure", err)
			}
			waitOperation(t, c, op.ID, "succeeded")
		})
	}
}

func TestHostCompatibilityOperationReceiptPreservesRealFencesAndRetry(t *testing.T) {
	for _, fence := range []string{"none", "public", "maintenance", "dispatched", "ordinary-error"} {
		t.Run(fence, func(t *testing.T) {
			s := testStore(t)
			if _, _, err := s.register("actor", registration("alpha", "create", 18080)); err != nil {
				t.Fatal(err)
			}
			req := protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1}
			op, _, err := s.accept("actor", "alpha", req)
			if err != nil {
				t.Fatal(err)
			}
			op, v, err := s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			state := v.Runtime
			problem := fmt.Errorf("%w: package evidence unavailable", runtimequalify.ErrHostCompatibility)
			switch fence {
			case "public":
				v.Public.RecoveryRequired = true
				tx, err := s.db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if err := saveInstance(tx, v); err != nil {
					tx.Rollback()
					t.Fatal(err)
				}
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
			case "maintenance":
				state.MaintenanceUncertain = true
			case "dispatched":
				state.Runtime.Containers = []runtimequalify.ContainerRecord{{CreateUncertain: true}}
			case "ordinary-error":
				problem = errors.New("runtime action genuinely failed")
			}
			if err := s.finish(op, state, runtimequalify.ManagedObservation{State: "unknown", Health: "unknown"}, problem, false); err != nil {
				t.Fatal(err)
			}
			// Preserve the finished receipt through an actual journal reopen.
			s = rc11ReopenStore(t, s)
			after, err := loadInstance(s.db, "alpha")
			if err != nil || after.Public.RecoveryRequired != (fence != "none") || after.Public.ActiveOperationID != "" {
				t.Fatalf("wrong durable recovery state: %+v %v", after, err)
			}
			replayed, found, err := s.accept("actor", "alpha", req)
			if err != nil || !found || replayed.ID != op.ID || replayed.Status != "failed" {
				t.Fatalf("failed admission receipt lost: %+v %v", replayed, err)
			}
			if fence == "none" {
				if err := s.observe(after, after.Runtime, runtimequalify.ManagedObservation{State: "uninitialized", Health: "unknown"}); err != nil {
					t.Fatal(err)
				}
				req.RequestID = "explicit-retry"
				if _, replayed, err := s.accept("actor", "alpha", req); err != nil || replayed {
					t.Fatal("host repair did not permit ordinary explicit retry", err)
				}
			}
		})
	}
}
