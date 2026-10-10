package managed

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/routines"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestManagedCometEpochAdmissionReplayInvalidationAndSchema(t *testing.T) {
	for _, kind := range []string{"inspect-epochs", "roll"} {
		t.Run(kind, func(t *testing.T) {
			s, p, _ := routineFixture(t)
			prepareStoredComet(t, s)
			req := protocol.OperationRequest{Kind: kind, RequestID: "epoch-work", ExpectedRevision: 1}
			op, hit, err := s.accept("actor", "comet", req)
			if err != nil || hit {
				t.Fatal("comet epoch admission", err)
			}
			var version int
			if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
				t.Fatal("epoch schema guard absent")
			}
			actual, v, err := s.beginOperation(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			v.Runtime.Epochs = &epochs.Inventory{Source: epochs.Source, PierID: v.Runtime.ID, ObservedAt: now, Digest: strings.Repeat("d", 64), Entries: []epochs.Entry{{Name: "0i0", Protected: true}}}
			if kind == "roll" {
				v.Runtime.RollNeedsRuntime = true
				v.Runtime.Maintenance = &runtimequalify.ManagedMaintenance{Kind: "roll", StartedAt: now, FinishedAt: &now, CreatedEpoch: "0i100"}
			}
			if err := s.finish(actual, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
				t.Fatal(err)
			}
			completed, hit, err := s.accept("actor", "comet", req)
			if err != nil || !hit || completed.Status != "succeeded" || completed.ID != op.ID {
				t.Fatal("epoch replay", err)
			}
			if err := s.initialize(); err != nil {
				t.Fatal("schema cannot reopen", err)
			}
			stored, err := loadInstance(s.db, "comet")
			if err != nil || !reflect.DeepEqual(stored.Public.Epochs, v.Runtime.Epochs) || stored.Public.RollNeedsRuntime != (kind == "roll") {
				t.Fatal("durable epoch state lost", err)
			}
			if kind == "roll" {
				if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "roll", RequestID: "new-roll-without-resume", ExpectedRevision: 1}); err == nil {
					t.Fatal("consecutive roll admitted")
				}
			}
			_, _, err = s.accept("actor", "comet", protocol.OperationRequest{Kind: "prune-epochs", RequestID: "delete", ExpectedRevision: 1, EpochCutoff: "0i0", InventoryDigest: strings.Repeat("d", 64), ConfirmHistoryLoss: true})
			var f *fault
			if !errors.As(err, &f) || f.problem.Code != "stale_or_unqualified_inventory" {
				t.Fatal("comet deletion accepted an invalid/protected inventory", err)
			}
			cfg := stored.Public.Config
			cfg.HTTPPort++
			if _, err := s.configure("actor", "comet", protocol.ConfigureRequest{ExpectedRevision: 1, Config: cfg}); err != nil {
				t.Fatal(err)
			}
			if replay, hit, err := s.accept("actor", "comet", req); err != nil || !hit || !reflect.DeepEqual(replay, completed) {
				t.Fatal("changed revision broke exact historical replay")
			}
			stored, err = loadInstance(s.db, "comet")
			if err != nil || stored.Public.Epochs != nil || stored.Runtime.Epochs != nil {
				t.Fatal("replay resurrected stale inventory")
			}
			p.Config.Enabled = false
			if _, err := s.putRoutine("actor", "alpha", routines.Update{ExpectedRevision: p.Revision, ExpectedInstanceRevision: 1, Config: p.Config}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.putRoutine("actor", "comet", routines.Update{ExpectedInstanceRevision: 2, Config: p.Config}); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
				t.Fatal("policy update downgraded epoch state")
			}
		})
	}
}

func TestCometInterruptedRollCannotBeAdoptedAfterRestart(t *testing.T) {
	s := testStore(t)
	prepareStoredComet(t, s)
	op, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "roll", RequestID: "roll", ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.MaintenanceUncertain = true
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{Kind: "comet-roll", CreateDispatched: true, CreateUncertain: true}}
	if err := s.runtimeUpdate("comet", op.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	v, err = loadInstance(s.db, "comet")
	if err != nil || !v.Public.RecoveryRequired || !v.Runtime.UncertainMaintenance() {
		t.Fatal("lost roll quarantine", err)
	}
	for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs"} {
		if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: kind, RequestID: kind + "-after-interruption", ExpectedRevision: 1}); err == nil {
			t.Fatal("interrupted roll allowed new work", kind)
		}
	}
	if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "stop", RequestID: "cleanup", ExpectedRevision: 1}); err != nil {
		t.Fatal("cleanup refused", err)
	}
}
