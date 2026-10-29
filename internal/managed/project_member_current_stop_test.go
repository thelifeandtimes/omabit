package managed

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func readyCurrentStopProject(t *testing.T, s *store) (storedProject, projectMemberStopRequest, map[string]string) {
	t.Helper()
	p, req := resumedReadinessProject(t, s)
	instance := p.Members[0].InstanceID
	op, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberResumedReadiness(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.completeProjectMemberResumedReadiness(op.ID, resumedReadinessProof(t, s, p.ID, instance, req.ResumeOperationID)); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, m := range p.Members {
		var id string
		if err = s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE instance_id=?", m.InstanceID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids[m.InstanceID] = id
	}
	return p, projectMemberStopRequest{RequestID: "stop-replacement", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, ReadinessOperationID: op.ID}, ids
}
func currentStopObservations(t *testing.T, s *store, project, instance string) (projectnetwork.MemberStopObservation, projectnetwork.MemberStopObservation) {
	t.Helper()
	var id string
	if err := s.db.QueryRow("SELECT operation_id FROM project_member_resumed_readiness WHERE project_id=? AND instance_id=? AND json_extract(body,'$.phase')='observed' ORDER BY operation_id LIMIT 1", project, instance).Scan(&id); err != nil {
		t.Fatal(err)
	}
	j, err := loadProjectMemberResumedReadiness(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	o := j.Evidence.After
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	before := projectnetwork.MemberStopObservation{Runtime: o.Runtime, Storage: o.Storage, StartedAt: o.StartedAt, CapturedAt: o.CapturedAt}
	data, _ := json.Marshal(before)
	var after projectnetwork.MemberStopObservation
	if err = json.Unmarshal(data, &after); err != nil {
		t.Fatal(err)
	}
	after.CapturedAt = o.CapturedAt.Add(time.Millisecond)
	after.Runtime.ObservedAt = after.CapturedAt
	after.FinishedAt = after.CapturedAt
	for i := range after.Runtime.Members {
		if after.Runtime.Members[i].InstanceID == instance {
			after.Runtime.Members[i].Running = false
			after.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
		}
	}
	return before, after
}
func currentStopHistory(t *testing.T, s *store) map[string]string {
	t.Helper()
	rows, err := s.db.Query("SELECT id,body,runtime FROM operations ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var id string
		var body, runtime []byte
		if err = rows.Scan(&id, &body, &runtime); err != nil {
			t.Fatal(err)
		}
		result[id] = string(body) + "\n" + string(runtime)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestCurrentProjectMemberStopPreservesHistoryAndRefusesStaleReadiness(t *testing.T) {
	s := testStore(t)
	p, req, ids := readyCurrentStopProject(t, s)
	instance := p.Members[0].InstanceID
	before := currentStopHistory(t, s)
	old := req
	old.ReadinessOperationID = ids[instance]
	old.RequestID = "obsolete-stop"
	if _, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, old); err == nil {
		t.Fatal("obsolete readiness authorized Stop")
	}
	op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	bo, ao := currentStopObservations(t, s, p.ID, instance)
	d := &storedStopDriver{t: t, s: s, id: op.ID, before: bo, after: ao}
	if err = s.runProjectMemberStop(context.Background(), op.ID, d); err != nil {
		t.Fatal(err)
	}
	if err = validateProjectMemberResumedReadiness(s.db); err != nil {
		t.Fatal(err)
	}
	after := currentStopHistory(t, s)
	for id, body := range before {
		if after[id] != body {
			t.Fatal("old history changed", id)
		}
	}
	if len(after) != len(before)+1 {
		t.Fatal("unexpected operation")
	}
}

func TestCurrentProjectMemberStopRestartRetainsEveryCommitBoundary(t *testing.T) {
	for _, cut := range []string{"accepted", "begun", "dispatch", "signal", "exit", "finished", "unclean-exit"} {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req, ids := readyCurrentStopProject(t, s)
			instance := p.Members[0].InstanceID
			originalLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			originalReady, _ := loadProjectMemberResumedReadiness(s.db, req.ReadinessOperationID)
			op, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			before, after := currentStopObservations(t, s, p.ID, instance)
			if cut != "accepted" {
				if _, err = s.beginProjectMemberStop(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if cut != "accepted" && cut != "begun" {
				advanceStopFixture(t, s, p, op.ID, "stop-dispatched", &before)
			}
			if cut == "signal" || cut == "exit" || cut == "finished" || cut == "unclean-exit" {
				advanceStopFixture(t, s, p, op.ID, "stop-signalled", nil)
			}
			if cut == "unclean-exit" {
				after.ExitCode = 137
			}
			if cut == "exit" || cut == "finished" || cut == "unclean-exit" {
				advanceStopFixture(t, s, p, op.ID, "stopped", &after)
			}
			if cut == "finished" {
				if err = s.finishProjectMemberStop(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			retained, _ := loadProjectMemberStop(s.db, op.ID)
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			s, err = testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			for i := 0; i < 2; i++ {
				if err = s.reconcileInterrupted(); err != nil {
					t.Fatal(err)
				}
			}
			got, _ := loadProjectMemberStop(s.db, op.ID)
			if !reflect.DeepEqual(retained, got) {
				t.Fatal("restart altered Stop evidence")
			}
			want := "interrupted"
			if cut == "finished" {
				want = "succeeded"
			}
			if cut == "unclean-exit" {
				want = "failed"
			}
			actual, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil || !hit || actual.ID != op.ID || actual.Status != want {
				t.Fatal(actual, hit, err)
			}
			if _, err = s.beginProjectMemberStop(op.ID); err == nil {
				t.Fatal("restarted retained Stop")
			}
			if err = s.finishProjectMemberStop(op.ID); err == nil {
				t.Fatal("re-finalized retained Stop")
			}
			req.RequestID = "another-stop"
			if _, _, err = s.acceptProjectMemberStop("actor", p.ID, instance, req); err == nil {
				t.Fatal("re-signalled old container")
			}
			peer := p.Members[1].InstanceID
			req.ReadinessOperationID = ids[peer]
			_, _, err = s.acceptProjectMemberStop("actor", p.ID, peer, req)
			if err == nil {
				t.Fatal("incorrect peer Stop admission", got.Stop.Phase, err)
			}
			afterLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			afterReady, _ := loadProjectMemberResumedReadiness(s.db, originalReady.OperationID)
			if !reflect.DeepEqual(originalLaunch, afterLaunch) || !reflect.DeepEqual(originalReady, afterReady) {
				t.Fatal("Stop rewrote original receipts")
			}
		})
	}
}

func TestCurrentProjectMemberStopWorkerKeepsExitAndUncertainty(t *testing.T) {
	for _, fault := range []string{"pass", "signal", "cancel", "wait", "late-wait", "unclean", "oom"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req, _ := readyCurrentStopProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			before, after := currentStopObservations(t, s, p.ID, instance)
			if fault == "unclean" {
				after.ExitCode = 137
			}
			if fault == "oom" {
				after.OOMKilled = true
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &storedStopDriver{t: t, s: s, id: op.ID, before: before, after: after, fault: fault, cancel: cancel}
			err = s.runProjectMemberStop(ctx, op.ID, d)
			actual, _ := loadOperation(s.db, op.ID)
			j, _ := loadProjectMemberStop(s.db, op.ID)
			want := "interrupted"
			phase := "stop-signalled"
			if fault == "pass" {
				want = "succeeded"
				phase = "stopped"
			}
			if fault == "signal" {
				phase = "stop-dispatched"
			}
			if fault == "unclean" || fault == "oom" {
				want = "failed"
				phase = "stopped"
			}
			if fault == "late-wait" {
				phase = "stopped"
			}
			if (fault == "pass") != (err == nil) || actual.Status != want || j.Stop.Phase != phase || d.signals != 1 {
				t.Fatal(actual.Status, j.Stop.Phase, want, phase, err)
			}
			if err = validateProjectMemberStops(s.db); err != nil {
				t.Fatal(err)
			}
			if err = s.runProjectMemberStop(context.Background(), op.ID, d); err == nil || d.signals != 1 {
				t.Fatal("replayed retained Stop")
			}
		})
	}
}

func TestCurrentProjectMemberStopRefusesDamagedHistory(t *testing.T) {
	for _, fault := range []string{"replacement-mirror", "retirement-body", "retirement-mirror", "ready-mirror", "stop-creator"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req, _ := readyCurrentStopProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			j, err := loadProjectMemberStop(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "replacement-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", j.Stop.LaunchOperationID)
			case "retirement-body":
				resumed, e := loadProjectMemberResume(s.db, j.Stop.LaunchOperationID)
				if e != nil {
					t.Fatal(e)
				}
				retired, e := loadProjectMemberRetirement(s.db, resumed.RetirementOperationID)
				if e != nil {
					t.Fatal(e)
				}
				retired.Retirement.OperationID = "changed"
				body, _ := json.Marshal(retired)
				_, err = s.db.Exec("UPDATE project_member_retirements SET body=? WHERE operation_id=?", body, retired.OperationID)
			case "retirement-mirror":
				resumed, e := loadProjectMemberResume(s.db, j.Stop.LaunchOperationID)
				if e != nil {
					t.Fatal(e)
				}
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", resumed.RetirementOperationID)
			case "ready-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", req.ReadinessOperationID)
			case "stop-creator":
				j.Stop.LaunchOperationID = "obsolete-creator"
				tx, e := s.db.Begin()
				if e != nil {
					t.Fatal(e)
				}
				err = saveProjectMemberStop(tx, j)
				if err == nil {
					err = tx.Commit()
				} else {
					tx.Rollback()
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = validateProjectMemberStops(s.db); err == nil {
				t.Fatal("damaged history accepted", fault)
			}
			if _, err = s.beginProjectMemberStop(op.ID); err == nil {
				t.Fatal("damaged history dispatched", fault)
			}
		})
	}
}
