package managed

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func rc11ReopenStore(t *testing.T, s *store) *store {
	t.Helper()
	dir, mode := s.directory, s.mode
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	s, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.mode = mode
	t.Cleanup(func() { s.close() })
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	return s
}

func rc11RegisterInitialShip(t *testing.T, s *store, owned bool) string {
	t.Helper()
	s.mode = DevelopmentMode
	if owned {
		_, _, err := s.registerShip("actor", protocol.CreateShipRequest{Ship: "~sampel-palnet", RequestID: "register", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true})
		if err != nil {
			t.Fatal(err)
		}
		return "sampel-palnet"
	}
	if _, _, err := s.registerKind("actor", registration("comet", "register", 18080), nil, true); err != nil {
		t.Fatal(err)
	}
	return "comet"
}

func TestRC11QueuedInitialStartReopensWithInterruptedReceiptAndNewRequest(t *testing.T) {
	for _, owned := range []bool{false, true} {
		name := "comet"
		if owned {
			name = "owned"
		}
		t.Run(name, func(t *testing.T) {
			s := testStore(t)
			id := rc11RegisterInitialShip(t, s, owned)
			req := protocol.OperationRequest{Kind: "start", RequestID: "first-start", ExpectedRevision: 1}
			op, _, err := s.accept("actor", id, req)
			if err != nil || op.Status != "queued" {
				t.Fatal(op, err)
			}
			before, err := loadInstance(s.db, id)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				s = rc11ReopenStore(t, s)
				v, err := loadInstance(s.db, id)
				if err != nil || v.Runtime.UncertainProvisioning() || v.Public.RecoveryRequired || v.Public.ActiveOperationID != "" || v.Public.ObservedState != "uninitialized" || !reflect.DeepEqual(v.Runtime, before.Runtime) {
					t.Fatalf("queued Start invented runtime work or quarantine: %+v %v", v, err)
				}
				old, replayed, err := s.accept("actor", id, req)
				if err != nil || !replayed || old.ID != op.ID || old.Status != "interrupted" || old.FinishedAt == nil || old.Actor != "actor" {
					t.Fatalf("lost interrupted receipt: %+v %v", old, err)
				}
			}
			next, replayed, err := s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "new-start", ExpectedRevision: 1})
			if err != nil || replayed || next.ID == op.ID || next.Status != "queued" {
				t.Fatalf("ordinary retry refused: %+v %v", next, err)
			}
			if actual, _, err := s.beginOperation(next.ID); err != nil || actual.Status != "running" {
				t.Fatal("retry could not enter runtime work", err)
			}
		})
	}
}

func TestRC11InitialStartReconciliationPreservesExistingFencesAndRunningUncertainty(t *testing.T) {
	for _, owned := range []bool{false, true} {
		name := "comet"
		if owned {
			name = "owned"
		}
		for _, phase := range []string{"running", "queued-runtime-callback", "queued-provisioning-fence", "queued-maintenance-fence", "queued-public-fence"} {
			t.Run(name+"/"+phase, func(t *testing.T) {
				s := testStore(t)
				id := rc11RegisterInitialShip(t, s, owned)
				op, _, err := s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
				if err != nil {
					t.Fatal(err)
				}
				v, err := loadInstance(s.db, id)
				if err != nil {
					t.Fatal(err)
				}
				switch phase {
				case "running":
					if _, _, err := s.beginOperation(op.ID); err != nil {
						t.Fatal(err)
					}
				case "queued-runtime-callback":
					// Legacy/inconsistent queued receipts with runtime evidence are
					// not proof of non-dispatch merely because their status is queued.
					if err := s.runtimeUpdate(id, op.ID, v.Runtime); err != nil {
						t.Fatal(err)
					}
				case "queued-provisioning-fence":
					v.Runtime.Comet.Uncertain, v.Runtime.Storage.Uncertain = true, true
					saveStartupFixture(t, s, v)
				case "queued-maintenance-fence":
					v.Runtime.MaintenanceUncertain = true
					saveStartupFixture(t, s, v)
				case "queued-public-fence":
					v.Public.RecoveryRequired = true
					saveStartupFixture(t, s, v)
				}
				s = rc11ReopenStore(t, s)
				v, err = loadInstance(s.db, id)
				if err != nil {
					t.Fatal(err)
				}
				if phase == "queued-maintenance-fence" && !v.Runtime.MaintenanceUncertain || phase == "queued-public-fence" && !v.Public.RecoveryRequired || phase != "queued-maintenance-fence" && phase != "queued-public-fence" && !v.Runtime.Comet.Uncertain || phase == "queued-provisioning-fence" && !v.Runtime.Storage.Uncertain {
					t.Fatal("existing or running-work fence was lost")
				}
				_, _, err = s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "new-start", ExpectedRevision: 1})
				code := "recovery_required"
				if phase == "queued-public-fence" {
					code = "invalid_state"
				}
				requireFault(t, err, code)
			})
		}
	}
}

func TestRC11ResolvedMaintenanceIntentReadmitsOtherShipAfterReopen(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll", "comet-pack", "comet-meld", "comet-roll"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			for i, id := range []string{"alpha", "bravo"} {
				if _, _, err := s.register("actor", registration(id, "register-"+id, uint16(18080+i))); err != nil {
					t.Fatal(err)
				}
			}
			v, err := loadInstance(s.db, "alpha")
			if err != nil {
				t.Fatal(err)
			}
			v.Runtime.Initialized, v.Public.ObservedState = true, "stopped"
			saveStartupFixture(t, s, v)
			req := protocol.OperationRequest{Kind: strings.TrimPrefix(kind, "comet-"), RequestID: "maintenance", ExpectedRevision: 1}
			maintenance, _, err := s.accept("actor", "alpha", req)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := s.beginOperation(maintenance.ID); err != nil {
				t.Fatal(err)
			}
			v.Runtime.MaintenanceUncertain = true
			v.Runtime.Maintenance = &runtimequalify.ManagedMaintenance{Kind: req.Kind, StartedAt: time.Now().UTC()}
			v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: kind}}
			if err := s.runtimeUpdate("alpha", maintenance.ID, v.Runtime); err != nil {
				t.Fatal(err)
			}
			s = rc11ReopenStore(t, s)
			other, _, err := s.accept("actor", "bravo", protocol.OperationRequest{Kind: "start", RequestID: "blocked-start", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = s.beginOperation(other.ID)
			requireFault(t, err, "host_recovery_required")
			stop, _, err := s.accept("actor", "alpha", protocol.OperationRequest{Kind: "stop", RequestID: "stop", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			stop, v, err = s.beginOperation(stop.ID)
			if err != nil {
				t.Fatal(err)
			}
			// Driver resolution is covered by the runtime fixture suite. Exercise
			// its callback/receipt contract through real SQLite and host admission.
			v.Runtime.Runtime.Containers[0].Removed = true
			v.Runtime.MaintenanceUncertain = false
			if err := s.runtimeUpdate("alpha", stop.ID, v.Runtime); err != nil {
				t.Fatal(err)
			}
			if err := s.finish(stop, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
				t.Fatal(err)
			}
			s = rc11ReopenStore(t, s)
			old, replayed, err := s.accept("actor", "alpha", req)
			if err != nil || !replayed || old.ID != maintenance.ID || old.Status != "interrupted" {
				t.Fatal("cleanup rewrote maintenance outcome", old, err)
			}
			other, _, err = s.accept("actor", "bravo", protocol.OperationRequest{Kind: "start", RequestID: "retry-start", ExpectedRevision: 1})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := s.beginOperation(other.ID); err != nil {
				t.Fatal("resolved maintenance still blocked another ship", err)
			}
		})
	}
}
