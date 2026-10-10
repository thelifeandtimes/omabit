package managed

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func prepareStoredComet(t *testing.T, s *store) {
	t.Helper()
	s.mode = DevelopmentMode
	if _, _, err := s.registerKind("actor", registration("comet", "register-comet", 18087), nil, true); err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "comet")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	v.Runtime.Prepared, v.Runtime.Initialized = true, true
	v.Runtime.Comet = &runtimequalify.ManagedComet{Ship: "~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet", StartedAt: &now, FinishedAt: &now}
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}); err != nil {
		t.Fatal(err)
	}
}

func TestCometMaintenanceAdmissionReplayAndSchema(t *testing.T) {
	for _, kind := range []string{"pack", "meld"} {
		t.Run(kind, func(t *testing.T) {
			s := testStore(t)
			prepareStoredComet(t, s)
			for _, unsupported := range []string{"prune-epochs"} {
				req := protocol.OperationRequest{Kind: unsupported, RequestID: unsupported, ExpectedRevision: 1}
				if unsupported == "prune-epochs" {
					req.EpochCutoff, req.InventoryDigest, req.ConfirmHistoryLoss = "0i0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true
				}
				_, _, err := s.accept("actor", "comet", req)
				var f *fault
				if !errors.As(err, &f) || f.problem.Code != "stale_inventory" {
					t.Fatal("comet pruning bypassed inventory admission", unsupported, err)
				}
			}
			req := protocol.OperationRequest{Kind: kind, RequestID: "maintenance", ExpectedRevision: 1}
			op, hit, err := s.accept("actor", "comet", req)
			if err != nil || hit {
				t.Fatal("initialized stopped comet refused", err)
			}
			var version int
			if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
				t.Fatal("comet maintenance downgrade guard missing")
			}
			if duplicate, hit, err := s.accept("actor", "comet", req); err != nil || !hit || !reflect.DeepEqual(duplicate, op) {
				t.Fatal("queued replay changed")
			}
			actual, v, err := s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			v.Runtime.Maintenance = &runtimequalify.ManagedMaintenance{Kind: kind, StartedAt: now, FinishedAt: &now, BeforePierBytes: 1000, AfterPierBytes: 800}
			if err := s.finish(actual, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
				t.Fatal(err)
			}
			completed, hit, err := s.accept("actor", "comet", req)
			if err != nil || !hit || completed.ID != op.ID || completed.Status != "succeeded" {
				t.Fatal("terminal replay changed", err)
			}
			if err := s.initialize(); err != nil {
				t.Fatal("new schema cannot reopen", err)
			}
			snap, err := s.snapshot()
			if err != nil || snap.Instances[0].Maintenance.Kind != kind || snap.Instances[0].Ship == "" {
				t.Fatal("comet report lost", err)
			}
		})
	}
}

func TestCometRoutineSlotsProvenanceAndMonotonicSchema(t *testing.T) {
	for _, fault := range []string{"none", "identity", "unfinished", "uncertain"} {
		t.Run(fault, func(t *testing.T) {
			s, fakePolicy, _ := routineFixture(t)
			prepareStoredComet(t, s)
			now := time.Now().UTC()
			req := routines.Update{ExpectedInstanceRevision: 1, Config: routines.Config{Enabled: true, Action: "meld", Weekdays: []int{int(now.Weekday())}, HourUTC: now.Hour(), MinuteUTC: now.Minute(), WindowMinutes: 5}}
			p, err := s.putRoutine("actor", "comet", req)
			if err != nil {
				t.Fatal(err)
			}
			p.UpdatedAt = now.Truncate(time.Minute).Add(-time.Second)
			tx, err := s.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			v, err := loadInstance(tx, "comet")
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "identity":
				v.Runtime.Comet.Ship = "~zod"
			case "unfinished":
				v.Runtime.Comet.FinishedAt = nil
			case "uncertain":
				v.Runtime.Comet.Uncertain = true
			}
			if err := saveInstance(tx, v); err != nil {
				t.Fatal(err)
			}
			if err := saveRoutinePolicy(tx, p); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if fault != "none" {
				req.ExpectedRevision = 1
				if _, err := s.putRoutine("actor", "comet", req); err == nil {
					t.Fatal("unsafe comet enabled")
				}
				if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "pack", RequestID: "invalid", ExpectedRevision: 1}); err == nil {
					t.Fatal("unsafe comet compacted")
				}
			}
			op, queued, err := s.claimRoutine("comet", now, func(string) bool { return true })
			if err != nil || queued != (fault == "none") || op.Kind != "routine-meld" || fault != "none" && op.Status != "skipped" {
				t.Fatal("wrong routine admission", op, err)
			}
			if err := s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			if _, again, err := s.claimRoutine("comet", now.Add(time.Second), func(string) bool { return true }); err != nil || again {
				t.Fatal("comet slot repeated after restart")
			}
			fakePolicy.Config.Enabled = false
			if _, err := s.putRoutine("actor", "alpha", routines.Update{ExpectedRevision: fakePolicy.Revision, ExpectedInstanceRevision: 1, Config: fakePolicy.Config}); err != nil {
				t.Fatal(err)
			}
			var version int
			if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
				t.Fatal("fake policy edit lowered comet schema guard")
			}
		})
	}
}
