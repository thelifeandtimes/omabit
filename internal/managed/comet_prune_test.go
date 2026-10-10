package managed

import (
	"encoding/json"
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

func storedCometPrefix(t *testing.T) (*store, *epochs.Inventory) {
	t.Helper()
	s := testStore(t)
	prepareStoredComet(t, s)
	v, err := loadInstance(s.db, "comet")
	if err != nil {
		t.Fatal(err)
	}
	inv := genesisInventory()
	inv.PierID = v.Runtime.ID
	inv.Entries[2].Protected = false
	inv.Entries = append(inv.Entries, epochs.Entry{Name: "0i400", Digest: strings.Repeat("4", 64), FileCount: 5, SnapshotBytes: 100, Protected: true})
	v.Runtime.Epochs = inv
	if err := s.observe(v, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}); err != nil {
		t.Fatal(err)
	}
	return s, inv
}

func TestCometPruneConfirmationRepeatedReplayAndSchema(t *testing.T) {
	s, inv := storedCometPrefix(t)
	var firstID string
	for round, cutoff := range []string{"0i100", "0i200"} {
		req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "prune-" + cutoff, ExpectedRevision: 1, EpochCutoff: cutoff, InventoryDigest: inv.Digest, ConfirmHistoryLoss: true}
		for _, reason := range []string{"consent", "digest", "protected", "revision"} {
			bad := req
			want := "stale_or_unqualified_inventory"
			switch reason {
			case "consent":
				bad.ConfirmHistoryLoss = false
				want = "invalid_request"
			case "digest":
				bad.InventoryDigest = strings.Repeat("f", 64)
			case "protected":
				bad.EpochCutoff = inv.Entries[len(inv.Entries)-1].Name
			case "revision":
				bad.ExpectedRevision = 2
				want = "revision_conflict"
			}
			_, _, err := s.accept("actor", "comet", bad)
			var problem *fault
			if !errors.As(err, &problem) || problem.problem.Code != want {
				t.Fatal("invalid comet prune admitted", reason, err)
			}
		}
		op, hit, err := s.accept("actor", "comet", req)
		if err != nil || hit {
			t.Fatal("confirmed prune refused", err)
		}
		if round == 0 {
			firstID = op.ID
		}
		var version int
		if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
			t.Fatal("comet prune compatibility guard missing")
		}
		actual, v, err := s.beginOperation(op.ID)
		if err != nil || v.Runtime.PruneIntent == nil || !v.Runtime.PruneIntent.Confirmed || v.Runtime.PruneIntent.InventoryDigest != req.InventoryDigest || v.Runtime.PruneIntent.Cutoff != cutoff {
			t.Fatal("accepted intent lost", err)
		}
		selected := 2
		if round == 1 {
			selected = 1
		}
		after := *inv
		after.Entries = append([]epochs.Entry(nil), inv.Entries[selected:]...)
		after.Digest = strings.Repeat([]string{"e", "a"}[round], 64)
		after.ObservedAt = time.Now().UTC()
		now := time.Now().UTC()
		p := &runtimequalify.PrefixPruneRecord{StartedAt: now, FinishedAt: &now, Before: inv, After: &after, Cutoff: cutoff}
		for _, entry := range inv.Entries[:selected] {
			p.Targets = append(p.Targets, runtimequalify.EpochDeletion{Name: entry.Name, Removed: true, Files: make([]runtimequalify.PruneFile, entry.FileCount), RemovedFiles: make([]string, entry.FileCount)})
		}
		v.Runtime.PrefixPrune = p
		v.Runtime.Epochs = &after
		v.Runtime.PruneIntent = nil
		if err := s.runtimeUpdate("comet", op.ID, v.Runtime); err != nil {
			t.Fatal(err)
		}
		if err := s.finish(actual, v.Runtime, runtimequalify.ManagedObservation{State: "stopped", Health: "unknown"}, nil, false); err != nil {
			t.Fatal(err)
		}
		original, hit, err := s.accept("actor", "comet", req)
		if err != nil || !hit || original.Status != "succeeded" {
			t.Fatal("terminal prune replay", err)
		}
		if err := s.initialize(); err != nil {
			t.Fatal(err)
		}
		if replay, hit, err := s.accept("actor", "comet", req); err != nil || !hit || !reflect.DeepEqual(replay, original) {
			t.Fatal("persistent replay changed")
		}
		bad := req
		bad.InventoryDigest = strings.Repeat("b", 64)
		if _, _, err := s.accept("actor", "comet", bad); err == nil {
			t.Fatal("changed deletion evidence reused request ID")
		}
		inv = &after
	}
	var archived []byte
	if err := s.db.QueryRow("SELECT runtime FROM operations WHERE id=?", firstID).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	var state runtimequalify.ManagedState
	if json.Unmarshal(archived, &state) != nil || state.PrefixPrune.Cutoff != "0i100" || len(state.PrefixPrune.After.Entries) != 3 {
		t.Fatal("later prune overwrote first evidence")
	}
	v, err := loadInstance(s.db, "comet")
	if err != nil || v.Public.Pruning.Cutoff != "0i200" || len(v.Public.Pruning.RetainedEpochs) != 2 {
		t.Fatal("latest report lost", err)
	}
	config := routines.Config{Enabled: false, Action: "pack", Weekdays: []int{0}, WindowMinutes: 5}
	if _, err := s.putRoutine("actor", "comet", routines.Update{ExpectedInstanceRevision: 1, Config: config}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.register("actor", registration("fake", "reg-fake", 18081)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.putRoutine("actor", "fake", routines.Update{ExpectedInstanceRevision: 1, Config: config}); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != currentSchema {
		t.Fatal("older-feature policy edit lowered schema")
	}
}

func TestInterruptedCometPruneRetainsIndependentMarker(t *testing.T) {
	s, inv := storedCometPrefix(t)
	req := protocol.OperationRequest{Kind: "prune-epochs", RequestID: "partial-prune", ExpectedRevision: 1, EpochCutoff: "0i100", InventoryDigest: inv.Digest, ConfirmHistoryLoss: true}
	op, _, err := s.accept("actor", "comet", req)
	if err != nil {
		t.Fatal(err)
	}
	_, v, err := s.beginOperation(op.ID)
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.PrefixPrune = &runtimequalify.PrefixPruneRecord{Cutoff: "0i100", Before: inv, Uncertain: true, StartedAt: time.Now().UTC()}
	v.Runtime.MaintenanceUncertain = false // the per-prefix marker is authoritative
	if err := s.runtimeUpdate("comet", op.ID, v.Runtime); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileInterrupted(); err != nil {
		t.Fatal(err)
	}
	if replay, hit, err := s.accept("actor", "comet", req); err != nil || !hit || replay.Status != "interrupted" {
		t.Fatal("interrupted receipt changed")
	}
	for _, kind := range []string{"start", "pack", "meld", "roll", "inspect-epochs", "prune-epochs"} {
		attempt := req
		attempt.Kind = kind
		attempt.RequestID = "new-" + kind
		if kind != "prune-epochs" {
			attempt.EpochCutoff = ""
			attempt.InventoryDigest = ""
			attempt.ConfirmHistoryLoss = false
		}
		_, _, err := s.accept("actor", "comet", attempt)
		var problem *fault
		if !errors.As(err, &problem) || problem.problem.Code != "recovery_required" {
			t.Fatal("partial prune allowed work", kind, err)
		}
	}
	if _, _, err := s.accept("actor", "comet", protocol.OperationRequest{Kind: "stop", RequestID: "cleanup", ExpectedRevision: 1}); err != nil {
		t.Fatal("explicit cleanup refused", err)
	}
}
