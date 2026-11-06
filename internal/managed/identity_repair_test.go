package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func offlineIdentityRepairFixture(t *testing.T) (*OfflineStateLock, IdentityRepairRequest, storedInstance, map[string]string) {
	t.Helper()
	s := testStore(t)
	s.mode = DevelopmentMode
	_, _, err := s.registerShip("operator", protocol.CreateShipRequest{ID: "sampel-palnet", Ship: "~sampel-palnet", RequestID: "register", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	v, err := loadInstance(s.db, "sampel-palnet")
	if err != nil {
		t.Fatal(err)
	}
	v.Runtime.Prepared = true
	v.Runtime.Comet.Uncertain = true
	now := time.Now().UTC().Add(-time.Minute)
	v.Runtime.Comet.StartedAt = &now
	v.Runtime.Storage.MountIdentity = pierstore.DirectoryIdentity{Inode: 10}
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{ID: strings.Repeat("a", 64), Kind: "comet-boot", CreateDispatched: true}}
	v.Public.RecoveryRequired = true
	v.RuntimeRevision = 10
	tx, _ := s.db.Begin()
	if err = saveInstance(tx, v); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Capture exact pre-existing receipts and replay fingerprints.
	history := map[string]string{}
	rows, err := s.db.Query("SELECT id, body||fingerprint FROM operations")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, body string
		if err := rows.Scan(&id, &body); err != nil {
			t.Fatal(err)
		}
		history[id] = body
	}
	rows.Close()
	req := IdentityRepairRequest{RunnerID: s.runnerID, InstanceID: v.Public.ID, Revision: 1, ExpectedShip: v.Public.Ship, Ship: "~sampel-sampel", ContainerID: strings.Repeat("a", 64), RequestID: "correct-once"}
	s.close()
	if err = os.WriteFile(filepath.Join(s.directory, ".lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	lock, err := LockOfflineState(s.directory, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lock.Close() })
	return lock, req, v, history
}

func simulatedIdentityCorrection(_ context.Context, s runtimequalify.ManagedState, expected, ship, id string, saved *runtimequalify.BootIdentityRepair, persist func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error) {
	now := time.Now().UTC()
	if saved != nil {
		s = saved.State
		now = saved.VerifiedAt
	}
	storage := *s.Storage
	s.Storage = &storage
	s.Storage.Identity = pierstore.Identity{Inode: 20, UrbInode: 21}
	s.Runtime.Containers = append([]runtimequalify.ContainerRecord(nil), s.Runtime.Containers...)
	s.Runtime.Containers[0].Removed = true
	j := runtimequalify.BootIdentityRepair{Version: 1, ExpectedShip: expected, Ship: ship, ContainerID: id, VerifiedAt: now, Stage: "removed", State: s}
	if err := persist(j); err != nil {
		return runtimequalify.ManagedState{}, err
	}
	c := *s.Comet
	s.Comet = &c
	s.NativeShip, s.Comet.Ship = ship, ship
	s.Comet.Uncertain, s.Initialized = false, true
	s.Comet.FinishedAt = &now
	return s, nil
}

func TestOfflineIdentityRepairPreservesHistoryAndCommitFailureRecovery(t *testing.T) {
	lock, req, before, history := offlineIdentityRepairFixture(t)
	db, err := lock.openDB(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("CREATE TRIGGER fail_repair_commit BEFORE UPDATE ON instances BEGIN SELECT RAISE(ABORT, 'commit unavailable'); END"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err = lock.repairBootIdentity(context.Background(), req, simulatedIdentityCorrection); err == nil {
		t.Fatal("commit failure not exercised")
	}
	db, _ = lock.openDB(true)
	retained, err := loadInstance(db, req.InstanceID)
	if err != nil || !reflect.DeepEqual(retained, before) {
		t.Fatal("failed commit changed original registration", err)
	}
	if _, err = db.Exec("DROP TRIGGER fail_repair_commit"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	continued := false
	op, err := lock.repairBootIdentity(context.Background(), req, func(ctx context.Context, s runtimequalify.ManagedState, old, ship, id string, j *runtimequalify.BootIdentityRepair, p func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error) {
		if j == nil || j.Stage != "removed" || j.State.NativeShip != req.ExpectedShip {
			t.Fatal("lost pre-commit progress")
		}
		continued = true
		return simulatedIdentityCorrection(ctx, s, old, ship, id, j, p)
	})
	if err != nil || !continued || op.Status != "succeeded" || op.ConfigRevision != 2 {
		t.Fatal(op, err)
	}
	noDispatch := func(context.Context, runtimequalify.ManagedState, string, string, string, *runtimequalify.BootIdentityRepair, func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error) {
		t.Fatal("replay dispatched runtime work")
		return runtimequalify.ManagedState{}, nil
	}
	again, err := lock.repairBootIdentity(context.Background(), req, noDispatch)
	if err != nil || again.ID != op.ID {
		t.Fatal("replay lost", err)
	}
	changed := req
	changed.Ship = "~sampel-marzod"
	if _, err := lock.repairBootIdentity(context.Background(), changed, noDispatch); err == nil {
		t.Fatal("conflicting replay accepted")
	}
	db, _ = lock.openDB(false)
	after, err := loadInstance(db, req.InstanceID)
	if err != nil || after.Public.Ship != req.Ship || after.Public.RecoveryRequired || after.Public.ObservedState != "stopped" || after.Public.DesiredState != "stopped" || after.Public.Config != before.Public.Config || after.Public.ID != before.Public.ID || after.Public.PierPath != before.Public.PierPath || after.Runtime.ID != before.Runtime.ID || after.Runtime.Storage.MountIdentity != before.Runtime.Storage.MountIdentity || after.Runtime.NativeKeyDigest != before.Runtime.NativeKeyDigest || after.RuntimeRevision != before.RuntimeRevision+1 {
		t.Fatal("same-pier/config/identity repair contract lost", err)
	}
	for id, want := range history {
		var got string
		if err := db.QueryRow("SELECT body||fingerprint FROM operations WHERE id=?", id).Scan(&got); err != nil || got != want {
			t.Fatal("original receipt changed", err)
		}
	}
	var schema int
	_ = db.QueryRow("PRAGMA user_version").Scan(&schema)
	if schema != StateSchemaVersion {
		t.Fatal("schema changed")
	}
	db.Close()
	// The unmodified current-schema manager can reopen the correction and retain
	// both its audit and ordinary Start eligibility, with no migration/reset.
	s, err := testOpenStore(lock.directory)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	reopened, err := loadInstance(s.db, req.InstanceID)
	if err != nil || !reflect.DeepEqual(reopened, after) {
		t.Fatal("manager reopen changed correction", err)
	}
	start, _, err := s.accept("operator", req.InstanceID, protocol.OperationRequest{Kind: "start", ExpectedRevision: 2, RequestID: "resume"})
	if err != nil || start.Status != "queued" {
		t.Fatal("normal Start still refused", err)
	}
}

func TestOfflineIdentityRepairBindingsAndDuplicateRefusal(t *testing.T) {
	for _, mode := range []string{"runner", "revision", "identity", "duplicate", "busy", "schema", "intervening-manager", "changed-retry", "changed-storage-record"} {
		t.Run(mode, func(t *testing.T) {
			lock, req, before, _ := offlineIdentityRepairFixture(t)
			if mode == "intervening-manager" || mode == "changed-retry" || mode == "changed-storage-record" {
				_, err := lock.repairBootIdentity(context.Background(), req, func(ctx context.Context, s runtimequalify.ManagedState, old, ship, id string, j *runtimequalify.BootIdentityRepair, p func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error) {
					_, err := simulatedIdentityCorrection(ctx, s, old, ship, id, j, p)
					if err != nil {
						t.Fatal(err)
					}
					return runtimequalify.ManagedState{}, errors.New("interrupted")
				})
				if err == nil {
					t.Fatal("missing interruption")
				}
			}
			db, _ := lock.openDB(true)
			switch mode {
			case "runner":
				req.RunnerID = "runner-" + strings.Repeat("f", 32)
			case "revision":
				req.Revision++
			case "identity":
				req.ExpectedShip = "~sampel-sampel"
			case "changed-retry":
				req.RequestID = "new-attempt"
			case "duplicate":
				other := before
				other.Public.ID, other.Public.Ship = "other", req.Ship
				b, _ := json.Marshal(other)
				if _, err := db.Exec("INSERT INTO instances VALUES(?,?)", "other", b); err != nil {
					t.Fatal(err)
				}
			case "busy":
				_, _ = db.Exec("UPDATE instances SET body=json_set(body,'$.public.active_operation_id','op-active')")
			case "schema":
				_, _ = db.Exec("PRAGMA user_version=46")
			case "intervening-manager":
				_, _ = db.Exec("UPDATE instances SET body=json_set(body,'$.runtime_revision',11)")
			case "changed-storage-record":
				_, _ = db.Exec("UPDATE instances SET body=json_set(body,'$.runtime.storage.path','livenet/planets/sampel-palnet/other')")
			}
			db.Close()
			_, err := lock.repairBootIdentity(context.Background(), req, func(context.Context, runtimequalify.ManagedState, string, string, string, *runtimequalify.BootIdentityRepair, func(runtimequalify.BootIdentityRepair) error) (runtimequalify.ManagedState, error) {
				t.Fatal("refusal dispatched runtime work")
				return runtimequalify.ManagedState{}, nil
			})
			if err == nil {
				t.Fatal("invalid binding accepted")
			}
		})
	}
}
