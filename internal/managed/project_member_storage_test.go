package managed

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func memberStorageFixture(t *testing.T, s *store) (string, projectMemberStorageRequest) {
	t.Helper()
	req := projectRequest(t, "register-project", "a", 18080)
	op, _, err := s.registerProject("actor", req)
	if err != nil {
		t.Fatal(err)
	}
	return op.ProjectID, projectMemberStorageRequest{RequestID: "prepare-member", ExpectedRevision: 1, ReviewDigest: req.ReviewDigest}
}

var memberMountIdentity = pierstore.DirectoryIdentity{Device: 7, Inode: 42, Birth: "1000000000:1"}

func TestProjectMemberStorageSurvivesEveryCommitBoundary(t *testing.T) {
	for _, cut := range []string{"planned", "dispatched", "created", "observed"} {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			project, req := memberStorageFixture(t, s)
			op, hit, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			if cut != "planned" {
				if _, err = s.beginProjectMemberStorage("a-zod", op.ID); err != nil {
					t.Fatal(err)
				}
				if err = s.advanceProjectMemberStorage("a-zod", op.ID, "planned", "dispatched", pierstore.DirectoryIdentity{}); err != nil {
					t.Fatal(err)
				}
				if cut != "dispatched" {
					if err = s.advanceProjectMemberStorage("a-zod", op.ID, "dispatched", "created", memberMountIdentity); err != nil {
						t.Fatal(err)
					}
				}
				if cut == "observed" {
					if err = s.advanceProjectMemberStorage("a-zod", op.ID, "created", "observed", memberMountIdentity); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, err := loadProjectMemberStorage(s.db, "a-zod")
			if err != nil {
				t.Fatal(err)
			}
			s.close()
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			after, err := loadProjectMemberStorage(s.db, "a-zod")
			if err != nil || !reflect.DeepEqual(before, after) || after.Phase != cut || after.Location.Path != "fakeships/a-zod/zod" {
				t.Fatal(after, err)
			}
			op, err = loadOperation(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "interrupted"
			if cut == "observed" {
				want = "succeeded"
			}
			if op.Status != want || op.FinishedAt == nil {
				t.Fatal(op)
			}
			s.validateStorage = func(string) error { t.Error("replay probed filesystem"); return errors.New("probe") }
			req.ReplayOnly = true
			again, hit, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			if err != nil || !hit || !reflect.DeepEqual(again, op) {
				t.Fatal(again, hit, err)
			}
			if _, err = s.beginProjectMemberStorage("a-zod", op.ID); err == nil {
				t.Fatal("replayed completed/interrupted worker")
			}
			req.RequestID = "new"
			req.ReplayOnly = false
			_, _, err = s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			requireFault(t, err, "project_recovery_required")
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			again, err = loadOperation(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(again, op) {
				t.Fatal("reconciliation rewrote terminal receipt", err)
			}
			var count int
			if err = s.db.QueryRow("SELECT count(*) FROM instances").Scan(&count); err != nil || count != 0 {
				t.Fatal(count, err)
			}
		})
	}
}
func TestProjectMemberStorageClaimBlocksLegacyStorageAndExternalImports(t *testing.T) {
	s := testStore(t)
	project, req := memberStorageFixture(t, s)
	if _, _, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req); err != nil {
		t.Fatal(err)
	}
	for i, path := range []string{"fakeships/a-zod/zod", "fakeships/a-zod/nec", "fakeships/a-zod/nested/nec"} {
		req := registration("other", "other", 19000)
		req.RequestID += string(rune('a' + i))
		req.PierPath = path
		_, _, err := s.register("actor", req)
		requireFault(t, err, "pier_path_conflict")
	}
	if err := reservePierPath(s.db, "different", "fakeships/a-zod-neighbour/zod"); err != nil {
		t.Fatal(err)
	}
	b := &storageActionBackend{home: privateDir(t)}
	if err := pierstore.Init(b.home); err != nil {
		t.Fatal(err)
	}
	r := &Server{s: s}
	for _, path := range []string{filepath.Join(b.home, "piers/fakeships/a-zod"), filepath.Join(b.home, "piers/fakeships/a-zod/zod")} {
		requireFault(t, r.externalPathAvailable(b, path), "source_reserved")
	}
	if err := r.externalPathAvailable(b, filepath.Join(b.home, "piers/fakeships/neighbour")); err != nil {
		t.Fatal(err)
	}
	// Another member's distinct exclusive parent is allowed by the same project.
	req.RequestID = "prepare-nec"
	if _, _, err := s.acceptProjectMemberStorage("actor", project, "a-nec", req); err != nil {
		t.Fatal(err)
	}
	if err := validateProjectMemberStorage(s.db); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberStorageAdmissionRefusesExistingAndPendingLegacyPaths(t *testing.T) {
	for _, move := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing", true: "move-target"}[move], func(t *testing.T) {
			s := testStore(t)
			legacy := registration("legacy", "legacy", 19000)
			if !move {
				legacy.PierPath = "fakeships/a-zod/nec"
			}
			if _, _, err := s.register("actor", legacy); err != nil {
				t.Fatal(err)
			}
			if move {
				v, err := loadInstance(s.db, "legacy")
				if err != nil {
					t.Fatal(err)
				}
				v.Runtime.MoveTarget = "fakeships/a-zod/nec"
				tx, err := s.db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if err = saveInstance(tx, v); err != nil {
					t.Fatal(err)
				}
				if err = tx.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			project, req := memberStorageFixture(t, s)
			_, _, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			requireFault(t, err, "pier_path_conflict")
			var count int
			s.db.QueryRow("SELECT count(*) FROM project_member_storage").Scan(&count)
			if count != 0 {
				t.Fatal("failed admission retained a claim")
			}
		})
	}
}
func TestProjectMemberStorageAtomicMirrorAndSingleWorkerClaim(t *testing.T) {
	s := testStore(t)
	project, req := memberStorageFixture(t, s)
	op, _, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	success := make(chan bool, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.beginProjectMemberStorage("a-zod", op.ID); success <- err == nil }()
	}
	wg.Wait()
	close(success)
	count := 0
	for ok := range success {
		if ok {
			count++
		}
	}
	if count != 1 {
		t.Fatal("duplicate worker claims", count)
	}
	_, err = s.db.Exec(`CREATE TRIGGER fail_member_mirror BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'fixture'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.advanceProjectMemberStorage("a-zod", op.ID, "planned", "dispatched", pierstore.DirectoryIdentity{}); err == nil {
		t.Fatal("failed receipt write accepted")
	}
	state, err := loadProjectMemberStorage(s.db, "a-zod")
	if err != nil || state.Phase != "planned" {
		t.Fatal("partial transaction", state, err)
	}
	if err = validateProjectMemberStorage(s.db); err != nil {
		t.Fatal(err)
	}
}

type memberStorageStoreDriver struct {
	t         *testing.T
	s         *store
	createErr error
	calls     []string
}

func (d *memberStorageStoreDriver) check(v projectnetwork.MemberStorage) {
	d.t.Helper()
	saved, err := loadProjectMemberStorage(d.s.db, v.InstanceID)
	if err != nil || !reflect.DeepEqual(saved, v) {
		d.t.Fatal("filesystem action preceded durable intent", err)
	}
	op, err := loadOperation(d.s.db, v.OperationID)
	if err != nil || op.Status != "running" {
		d.t.Fatal(op, err)
	}
}
func (d *memberStorageStoreDriver) CreateMount(ctx context.Context, v projectnetwork.MemberStorage) (pierstore.DirectoryIdentity, error) {
	d.check(v)
	d.calls = append(d.calls, v.Phase)
	return memberMountIdentity, d.createErr
}
func (d *memberStorageStoreDriver) InspectMount(ctx context.Context, v projectnetwork.MemberStorage) error {
	d.check(v)
	d.calls = append(d.calls, v.Phase)
	return nil
}
func TestProjectMemberStorageWorkerRetainsIdentityOnLateFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "late-sync-failure"}[fail], func(t *testing.T) {
			s := testStore(t)
			project, req := memberStorageFixture(t, s)
			op, _, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			if err != nil {
				t.Fatal(err)
			}
			d := &memberStorageStoreDriver{t: t, s: s}
			if fail {
				d.createErr = errors.New("late filesystem failure")
			}
			err = s.runProjectMemberStorage(context.Background(), "a-zod", op.ID, d)
			if (err != nil) != fail {
				t.Fatal(err)
			}
			state, err := loadProjectMemberStorage(s.db, "a-zod")
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := loadOperation(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			phase, status, calls := "observed", "succeeded", []string{"dispatched", "created"}
			if fail {
				phase, status, calls = "created", "interrupted", []string{"dispatched"}
			}
			if state.Phase != phase || receipt.Status != status || state.Location.MountIdentity != memberMountIdentity || !reflect.DeepEqual(calls, d.calls) {
				t.Fatal(state, receipt, d.calls)
			}
		})
	}
}
func TestProjectMemberStorageCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"runtime", "mirror", "orphan", "status"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			project, req := memberStorageFixture(t, s)
			op, _, err := s.acceptProjectMemberStorage("actor", project, "a-zod", req)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "runtime":
				v, err := loadProjectMemberStorage(s.db, "a-zod")
				if err != nil {
					t.Fatal(err)
				}
				v.RuntimeID = "ffffffffffffffffffffffffffffffff"
				data, _ := json.Marshal(v)
				_, err = s.db.Exec("UPDATE project_member_storage SET body=?", data)
				if err != nil {
					t.Fatal(err)
				}
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_storage")
			case "status":
				_, err = s.db.Exec("UPDATE operations SET body=json_set(body,'$.status','succeeded') WHERE id=?", op.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			s.close()
			opened, err := testOpenStore(dir)
			if err == nil {
				opened.close()
				t.Fatal("corrupt member storage accepted")
			}
		})
	}
}
