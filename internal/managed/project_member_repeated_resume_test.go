package managed

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

// End this generation with a lost Stop reply, then separately observe its clean
// exit and retire only that container. Prior uncertainty must remain untouched.
func retireCurrentFixture(t *testing.T, s *store, p storedProject, req projectMemberStopRequest, generation int) projectMemberResumeRequest {
	t.Helper()
	return retireStoppedGenerationFixture(t, s, dispatchedCurrentStopFixture(t, s, p, req, generation))
}
func dispatchedCurrentStopFixture(t *testing.T, s *store, p storedProject, req projectMemberStopRequest, generation int) projectMemberStopJob {
	t.Helper()
	instance := p.Members[0].InstanceID
	req.RequestID = fmt.Sprintf("stop-generation-%d", generation)
	stop, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := loadProjectMemberResumedReadiness(s.db, req.ReadinessOperationID)
	if err != nil {
		t.Fatal(err)
	}
	o := ready.Evidence.After
	raw, _ := json.Marshal(o)
	if err = json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	before := projectnetwork.MemberStopObservation{Runtime: o.Runtime, Storage: o.Storage, StartedAt: o.StartedAt, CapturedAt: o.CapturedAt}
	if _, err = s.beginProjectMemberStop(stop.ID); err != nil {
		t.Fatal(err)
	}
	advanceStopFixture(t, s, p, stop.ID, "stop-dispatched", &before)
	if err = s.interruptMemberStop(stop.ID); err != nil {
		t.Fatal(err)
	}
	j, err := loadProjectMemberStop(s.db, stop.ID)
	if err != nil {
		t.Fatal(err)
	}
	return j
}
func retireStoppedGenerationFixture(t *testing.T, s *store, stop projectMemberStopJob) projectMemberResumeRequest {
	t.Helper()
	p, err := loadProject(s.db, stop.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	instance := stop.InstanceID
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	generation, err := memberResumeGeneration(s.db, projectnetwork.MemberRetirement{Exit: projectnetwork.MemberStopExitEvidence{Stop: stop.Stop}}, true)
	if err != nil {
		t.Fatal(err)
	}
	exitReq := projectMemberStopExitRequest{RequestID: fmt.Sprintf("exit-generation-%d", generation), StopOperationID: stop.OperationID, ExpectedRevision: p.Revision, ReviewDigest: digest}
	exit, _, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, exitReq)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberStopExit(exit.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.completeProjectMemberStopExit(exit.ID, memberStopExitProof(t, s, exit.ID)); err != nil {
		t.Fatal(err)
	}
	retiredReq := projectMemberRetirementRequest{RequestID: fmt.Sprintf("retire-generation-%d", generation), ExitOperationID: exit.ID, ExpectedRevision: p.Revision, ReviewDigest: digest}
	retired, _, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, retiredReq)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberRetirement(retired.ID); err != nil {
		t.Fatal(err)
	}
	b, a := retirementStoreObservations(t, s, retired.ID)
	advanceRetirementFixture(t, s, p, retired.ID, "remove-dispatched", &b, nil)
	advanceRetirementFixture(t, s, p, retired.ID, "removed", nil, nil)
	advanceRetirementFixture(t, s, p, retired.ID, "absent", nil, &a)
	if err = s.finishProjectMemberRetirement(retired.ID); err != nil {
		t.Fatal(err)
	}
	return projectMemberResumeRequest{RequestID: fmt.Sprintf("resume-generation-%d", generation+1), RetirementOperationID: retired.ID, ExpectedRevision: p.Revision, ReviewDigest: digest}
}
func resumeThroughFixture(t *testing.T, s *store, id, cut string) {
	t.Helper()
	if cut == "accepted" {
		return
	}
	if _, err := s.beginProjectMemberResume(id, initialResumeFixture(t, s, id)); err != nil {
		t.Fatal(err)
	}
	if cut == "begun" {
		return
	}
	for _, phase := range []string{"create-dispatched", "created", "start-dispatched", "started", "running"} {
		j, err := loadProjectMemberResume(s.db, id)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.advanceProjectMemberResume(id, j.Resume.Phase, nextResumeFixture(t, s, id, phase)); err != nil {
			t.Fatal(err)
		}
		if phase == cut {
			return
		}
	}
	if err := s.finishProjectMemberResume(id); err != nil {
		t.Fatal(err)
	}
}
func observeGenerationFixture(t *testing.T, s *store, p storedProject, resumeID, digest string, generation int) projectMemberStopRequest {
	t.Helper()
	req := projectMemberResumedReadinessRequest{RequestID: fmt.Sprintf("observe-generation-%d", generation), ResumeOperationID: resumeID, ExpectedRevision: p.Revision, ReviewDigest: digest}
	op, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, p.Members[0].InstanceID, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberResumedReadiness(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.completeProjectMemberResumedReadiness(op.ID, resumedReadinessProof(t, s, p.ID, p.Members[0].InstanceID, resumeID)); err != nil {
		t.Fatal(err)
	}
	return projectMemberStopRequest{RequestID: fmt.Sprintf("stop-generation-%d", generation), ReadinessOperationID: op.ID, ExpectedRevision: p.Revision, ReviewDigest: digest}
}
func TestRepeatedMemberResumePreservesCausalHistoryAcrossRestart(t *testing.T) {
	for _, generation := range []int{3, 4} {
		for _, cut := range []string{"accepted", "begun", "create-dispatched", "created", "start-dispatched", "started", "running", "finished"} {
			t.Run(fmt.Sprintf("generation%d-%s", generation, cut), func(t *testing.T) {
				dir := privateDir(t)
				s, err := testOpenStore(dir)
				if err != nil {
					t.Fatal(err)
				}
				p, stopReq, _ := readyCurrentStopProject(t, s)
				if generation == 4 {
					req := retireCurrentFixture(t, s, p, stopReq, 2)
					op, _, e := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
					if e != nil {
						t.Fatal(e)
					}
					resumeThroughFixture(t, s, op.ID, "finished")
					stopReq = observeGenerationFixture(t, s, p, op.ID, req.ReviewDigest, 3)
				}
				req := retireCurrentFixture(t, s, p, stopReq, generation-1)
				history := currentStopHistory(t, s)
				op, hit, err := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
				if err != nil || hit {
					t.Fatal(op, hit, err)
				}
				resumeThroughFixture(t, s, op.ID, cut)
				retained, err := loadProjectMemberResume(s.db, op.ID)
				if err != nil {
					t.Fatal(err)
				}
				if retained.Resume != nil && retained.Resume.PreviousGeneration != uint64(generation-1) {
					t.Fatal("wrong generation")
				}
				if err = s.close(); err != nil {
					t.Fatal(err)
				}
				s, err = testOpenStore(dir)
				if err != nil {
					t.Fatal(err)
				}
				defer s.close()
				for range 2 {
					if err = s.reconcileInterrupted(); err != nil {
						t.Fatal(err)
					}
				}
				got, err := loadProjectMemberResume(s.db, op.ID)
				if err != nil || !reflect.DeepEqual(got, retained) {
					t.Fatal("reopen rewrote history", err)
				}
				want := "interrupted"
				if cut == "finished" {
					want = "succeeded"
				}
				replay, hit, err := s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req)
				if err != nil || !hit || replay.ID != op.ID || replay.Status != want {
					t.Fatal(replay, hit, err)
				}
				after := currentStopHistory(t, s)
				for id, body := range history {
					if after[id] != body {
						t.Fatal("prior history changed", id)
					}
				}
				if _, err = s.beginProjectMemberResume(op.ID, projectnetwork.MemberResume{}); err == nil {
					t.Fatal("replayed creation")
				}
				req.RequestID = "attempt-retirement-reuse"
				if _, _, err = s.acceptProjectMemberResume("actor", p.ID, p.Members[0].InstanceID, req); err == nil {
					t.Fatal("consumed retirement reused")
				}
				if cut == "finished" {
					observeGenerationFixture(t, s, p, op.ID, req.ReviewDigest, generation)
				}
				if err = validateProjectMemberResumedReadiness(s.db); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestRepeatedMemberResumeMigrationRollsBackWithForeignKeysEnabled(t *testing.T) {
	for _, f := range g02Fixtures(t) {
		if f.Schema < 32 || f.Schema > 34 {
			continue
		}
		for _, cut := range []string{"rebuilt", "initialized", "unknown-reference"} {
			t.Run(fmt.Sprintf("schema%d-%s", f.Schema, cut), func(t *testing.T) {
				dir := g02Materialize(t, f)
				db := g02RawDB(t, dir)
				defer db.Close()
				_, err := db.Exec("PRAGMA foreign_keys=ON")
				g02Check(t, err)
				var original string
				g02Check(t, db.QueryRow("SELECT sql FROM sqlite_master WHERE name='project_member_resumes'").Scan(&original))
				if cut == "unknown-reference" {
					_, err = db.Exec("CREATE TABLE unexpected_resume_child (id TEXT REFERENCES project_member_resumes(operation_id))")
					g02Check(t, err)
				}
				for range 2 {
					tx, err := db.Begin()
					g02Check(t, err)
					if cut == "initialized" {
						err = (&store{db: db}).initializeTransaction(tx)
					} else {
						err = migrateRepeatedMemberResumes(tx, f.Schema)
					}
					if cut == "unknown-reference" {
						if err == nil {
							t.Fatal("unknown child was discarded")
						}
					} else {
						g02Check(t, err)
					}
					var enabled int
					g02Check(t, tx.QueryRow("PRAGMA foreign_keys").Scan(&enabled))
					if enabled != 1 {
						t.Fatal("migration disabled foreign keys")
					}
					g02Check(t, tx.Rollback())
					g02AssertRows(t, db, f, nil)
					var after string
					g02Check(t, db.QueryRow("SELECT sql FROM sqlite_master WHERE name='project_member_resumes'").Scan(&after))
					if after != original {
						t.Fatal("rollback changed parent DDL")
					}
					var version, temp int
					g02Check(t, db.QueryRow("PRAGMA user_version").Scan(&version))
					if version != f.Schema {
						t.Fatal("rollback changed schema marker")
					}
					g02Check(t, db.QueryRow("SELECT count(*) FROM sqlite_temp_master WHERE name='retained_resumed_readiness_v35'").Scan(&temp))
					if temp != 0 {
						t.Fatal("migration leaked temporary state")
					}
				}
				if cut == "unknown-reference" {
					_, err = db.Exec("DROP TABLE unexpected_resume_child")
					g02Check(t, err)
				}
				g02Check(t, db.Close())
				s := g02Open(t, dir)
				g02AssertRows(t, s.db, f, nil)
				rows, err := s.db.Query("PRAGMA foreign_key_check")
				g02Check(t, err)
				if rows.Next() {
					t.Fatal("migration damaged references")
				}
				g02Check(t, rows.Err())
				g02Check(t, rows.Close())
			})
		}
	}
}
