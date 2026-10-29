package managed

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func preparedMemberLaunchProject(t *testing.T, s *store) (storedProject, projectMemberLaunchRequest) {
	t.Helper()
	anchor, _ := acceptAnchorStartFixture(t, s)
	if _, err := s.beginProjectAnchorStart(anchor.ProjectID, anchor.ID); err != nil {
		t.Fatal(err)
	}
	phases := []string{"planned", "dispatched", "started", "observed"}
	for i := 0; i < 3; i++ {
		var ns *projectnetwork.NamespaceIdentity
		if i == 2 {
			ns = anchorNamespaceFixture()
		}
		if err := s.advanceProjectAnchorStart(anchor.ProjectID, anchor.ID, phases[i], phases[i+1], ns); err != nil {
			t.Fatal(err)
		}
	}
	p, err := loadProject(s.db, anchor.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	for i, m := range p.Members {
		op, _, err := s.acceptProjectMemberStorage("actor", p.ID, m.InstanceID, projectMemberStorageRequest{RequestID: "storage-" + m.InstanceID, ExpectedRevision: 1, ReviewDigest: digest})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.beginProjectMemberStorage(m.InstanceID, op.ID); err != nil {
			t.Fatal(err)
		}
		id := memberMountIdentity
		id.Inode += uint64(i)
		if err = s.advanceProjectMemberStorage(m.InstanceID, op.ID, "planned", "dispatched", pierstore.DirectoryIdentity{}); err != nil {
			t.Fatal(err)
		}
		if err = s.advanceProjectMemberStorage(m.InstanceID, op.ID, "dispatched", "created", id); err != nil {
			t.Fatal(err)
		}
		if err = s.advanceProjectMemberStorage(m.InstanceID, op.ID, "created", "observed", id); err != nil {
			t.Fatal(err)
		}
	}
	return p, projectMemberLaunchRequest{RequestID: "launch-member", ExpectedRevision: 1, ReviewDigest: digest}
}
func memberLaunchSelection(t *testing.T, s *store, op protocol.Operation) projectnetwork.MemberLaunch {
	t.Helper()
	p, a, storage, err := memberLaunchPrerequisites(s.db, op.ProjectID, op.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	peers, err := projectLaunchPeers(s.db, op.ProjectID, op.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	state, err := projectnetwork.NewMemberLaunch(p.Plan, a, storage, op.ID, strings.Repeat("3", 64), peers)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

var memberLaunchPhases = []string{"planned", "workspace-dispatched", "workspace-created", "create-dispatched", "created", "start-dispatched", "started", "running"}

func advanceMemberLaunchFixture(t *testing.T, s *store, op protocol.Operation, n int, id string) {
	t.Helper()
	r := projectnetwork.MemberLaunchResult{}
	switch memberLaunchPhases[n+1] {
	case "workspace-created":
		r.Workspace = pierstore.DirectoryIdentity{Device: 7, Inode: 192, Birth: "1000:2"}
	case "created":
		r.ContainerID = id
	case "running":
		r.Namespace = anchorNamespaceFixture()
	}
	if err := s.advanceProjectMemberLaunch(op.InstanceID, op.ID, memberLaunchPhases[n], memberLaunchPhases[n+1], r); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberLaunchSurvivesEveryCommitBoundary(t *testing.T) {
	for cut := 0; cut <= 8; cut++ {
		t.Run(fmt.Sprint(cut), func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := preparedMemberLaunchProject(t, s)
			instance := p.Members[0].InstanceID
			op, hit, err := s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			initial := memberLaunchSelection(t, s, op)
			if cut > 0 {
				if _, err = s.beginProjectMemberLaunch(instance, op.ID, initial); err != nil {
					t.Fatal(err)
				}
			}
			for n := 0; n < cut-1; n++ {
				advanceMemberLaunchFixture(t, s, op, n, strings.Repeat("c", 64))
			}
			before, err := loadProjectMemberLaunch(s.db, instance)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			after, err := loadProjectMemberLaunch(s.db, instance)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("restart changed launch evidence", err)
			}
			a, err := loadProjectAnchorStart(s.db, p.ID)
			if err != nil || !reflect.DeepEqual(a, initial.Anchor) {
				t.Fatal("anchor receipt changed", err)
			}
			storage, err := loadProjectMemberStorage(s.db, instance)
			if err != nil || storage != initial.Storage {
				t.Fatal("storage receipt changed", err)
			}
			actual, err := loadOperation(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "interrupted"
			if cut == 8 {
				want = "succeeded"
			}
			if actual.Status != want {
				t.Fatal(actual)
			}
			s.validateStorage = func(string) error { t.Error("launch replay accessed filesystem"); return errors.New("unexpected") }
			req.ReplayOnly = true
			again, hit, err := s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil || !hit || !reflect.DeepEqual(actual, again) {
				t.Fatal(again, hit, err)
			}
			if _, err = s.beginProjectMemberLaunch(instance, op.ID, initial); err == nil {
				t.Fatal("terminal worker reclaimed")
			}
			if err = s.advanceProjectMemberLaunch(instance, op.ID, "planned", "workspace-dispatched", projectnetwork.MemberLaunchResult{}); err == nil {
				t.Fatal("late callback advanced receipt")
			}
			req.RequestID = "new-attempt"
			req.ReplayOnly = false
			_, _, err = s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			requireFault(t, err, "project_recovery_required")
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			final, err := loadOperation(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(final, actual) {
				t.Fatal("reconcile changed terminal receipt", err)
			}
			var count int
			if err = s.db.QueryRow("SELECT count(*) FROM instances").Scan(&count); err != nil || count != 0 {
				t.Fatal("launch materialized legacy instance", count, err)
			}
		})
	}
}
func TestProjectMemberLaunchSerialAdmissionAndPeerOwnership(t *testing.T) {
	s := testStore(t)
	p, req := preparedMemberLaunchProject(t, s)
	first, second := p.Members[0].InstanceID, p.Members[1].InstanceID
	op, _, err := s.acceptProjectMemberLaunch("actor", p.ID, first, req)
	if err != nil {
		t.Fatal(err)
	}
	nextReq := req
	nextReq.RequestID = "second-member"
	_, _, err = s.acceptProjectMemberLaunch("actor", p.ID, second, nextReq)
	requireFault(t, err, "project_recovery_required")
	initial := memberLaunchSelection(t, s, op)
	if _, err = s.beginProjectMemberLaunch(first, op.ID, initial); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 7; n++ {
		advanceMemberLaunchFixture(t, s, op, n, strings.Repeat("c", 64))
	}
	next, _, err := s.acceptProjectMemberLaunch("actor", p.ID, second, nextReq)
	if err != nil {
		t.Fatal(err)
	}
	selection := memberLaunchSelection(t, s, next)
	if len(selection.Peers) != 1 || selection.Peers[0].InstanceID != first || selection.Peers[0].Container.ID != strings.Repeat("c", 64) {
		t.Fatal(selection.Peers)
	}
	tampered := selection
	tampered.Peers = nil
	if _, err = s.beginProjectMemberLaunch(second, next.ID, tampered); err == nil {
		t.Fatal("omitted known peer admitted")
	}
	if _, err = s.beginProjectMemberLaunch(second, next.ID, selection); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 7; n++ {
		advanceMemberLaunchFixture(t, s, next, n, strings.Repeat("d", 64))
	}
	if err = validateProjectMemberLaunches(s.db); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberLaunchConcurrentClaimAndAtomicMirror(t *testing.T) {
	s := testStore(t)
	p, req := preparedMemberLaunchProject(t, s)
	instance := p.Members[0].InstanceID
	op, _, err := s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	initial := memberLaunchSelection(t, s, op)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.beginProjectMemberLaunch(instance, op.ID, initial)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		}
	}
	if winners != 1 {
		t.Fatal("multiple workers claimed", winners)
	}
	before, err := loadProjectMemberLaunch(s.db, instance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("CREATE TRIGGER refuse_launch_mirror BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;"); err != nil {
		t.Fatal(err)
	}
	if err = s.advanceProjectMemberLaunch(instance, op.ID, "planned", "workspace-dispatched", projectnetwork.MemberLaunchResult{}); err == nil {
		t.Fatal("mirror failure ignored")
	}
	after, err := loadProjectMemberLaunch(s.db, instance)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("journal update escaped rollback", err)
	}
	if err = validateProjectMemberLaunches(s.db); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberLaunchCorruptionRefusesOpen(t *testing.T) {
	for _, fault := range []string{"runtime", "mirror", "orphan", "status", "namespace"} {
		t.Run(fault, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := preparedMemberLaunchProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberLaunch("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			initial := memberLaunchSelection(t, s, op)
			if _, err = s.beginProjectMemberLaunch(instance, op.ID, initial); err != nil {
				t.Fatal(err)
			}
			j, err := loadProjectMemberLaunch(s.db, instance)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "runtime", "namespace":
				if fault == "runtime" {
					j.Launch.Storage.RuntimeID = strings.Repeat("e", 32)
				} else {
					j.Launch.Anchor.Namespace.Inode++
				}
				body, _ := json.Marshal(j)
				mirror, _ := json.Marshal(j.Launch)
				if _, err = s.db.Exec("UPDATE project_member_launches SET body=? WHERE instance_id=?", body, instance); err != nil {
					t.Fatal(err)
				}
				if _, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", mirror, op.ID); err != nil {
					t.Fatal(err)
				}
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_launches WHERE instance_id=?", instance)
			case "status":
				actual, e := loadOperation(s.db, op.ID)
				if e != nil {
					t.Fatal(e)
				}
				actual.Status = "succeeded"
				b, _ := json.Marshal(actual)
				_, err = s.db.Exec("UPDATE operations SET body=? WHERE id=?", b, op.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			s.close()
			reopened, err := testOpenStore(dir)
			if err == nil {
				reopened.close()
				t.Fatal("corrupt member state admitted")
			}
		})
	}
}
