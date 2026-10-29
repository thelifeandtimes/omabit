package managed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func resumedReadinessProject(t *testing.T, s *store) (storedProject, projectMemberResumedReadinessRequest) {
	t.Helper()
	p, req := readyResumeProject(t, s)
	instance := p.Members[0].InstanceID
	op, hit, err := s.acceptProjectMemberResume("actor", p.ID, instance, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	if _, err = s.beginProjectMemberResume(op.ID, initialResumeFixture(t, s, op.ID)); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"create-dispatched", "created", "start-dispatched", "started", "running"} {
		j, err := loadProjectMemberResume(s.db, op.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.advanceProjectMemberResume(op.ID, j.Resume.Phase, nextResumeFixture(t, s, op.ID, phase)); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.finishProjectMemberResume(op.ID); err != nil {
		t.Fatal(err)
	}
	return p, projectMemberResumedReadinessRequest{RequestID: "observe-resumed", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, ResumeOperationID: op.ID}
}
func resumedReadinessProof(t *testing.T, s *store, project, instance, resumeID string) projectnetwork.ResumedMemberReadiness {
	t.Helper()
	p, resumed, b, err := projectResumedReadinessPrerequisites(s.db, project, instance, resumeID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(resumed.After)
	var o projectnetwork.MemberResumeObservation
	if err = json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	var ship, endpoint string
	for _, m := range p.Plan.Members {
		if m.InstanceID == instance {
			ship = m.Ship
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/~/host", m.HTTPPort)
		}
	}
	r := projectnetwork.ResumedMemberReadiness{InstanceID: instance, ResumeOperationID: resumeID, ContainerGeneration: resumed.PreviousGeneration + 1, Binding: b, Storage: resumed.After.Storage, Ship: ship, Endpoint: endpoint, Before: o, After: o, CapturedAt: o.CapturedAt}
	if err = r.Validate(p.Plan, resumed); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestProjectMemberResumedReadinessRestartReplay(t *testing.T) {
	for _, phase := range []string{"planned", "observing", "observed"} {
		t.Run(phase, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := resumedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			old, _ := loadProjectMemberResume(s.db, req.ResumeOperationID)
			op, hit, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			if phase != "planned" {
				if _, err = s.beginProjectMemberResumedReadiness(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "observed" {
				if err = s.completeProjectMemberResumedReadiness(op.ID, resumedReadinessProof(t, s, p.ID, instance, req.ResumeOperationID)); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := loadProjectMemberResumedReadiness(s.db, op.ID)
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
			after, _ := loadProjectMemberResumedReadiness(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("reopen rewrote observation")
			}
			replay, hit, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			status := "interrupted"
			if phase == "observed" {
				status = "succeeded"
			}
			if err != nil || !hit || replay.ID != op.ID || replay.Status != status {
				t.Fatal(replay, hit, err)
			}
			if _, err = s.beginProjectMemberResumedReadiness(op.ID); err == nil {
				t.Fatal("restarted old observation")
			}
			req.RequestID = "new-observation"
			next, hit, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			if err != nil || hit || next.ID == op.ID {
				t.Fatal(next, hit, err)
			}
			now, _ := loadProjectMemberResume(s.db, req.ResumeOperationID)
			if !reflect.DeepEqual(old, now) {
				t.Fatal("rewrote replacement history")
			}
		})
	}
}
func TestProjectMemberResumedReadinessRefusesChangedProof(t *testing.T) {
	for _, fault := range []string{"not-begun", "resume", "generation", "old-container", "process", "stale", "future"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := resumedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			if fault != "not-begun" {
				if _, err = s.beginProjectMemberResumedReadiness(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			proof := resumedReadinessProof(t, s, p.ID, instance, req.ResumeOperationID)
			switch fault {
			case "resume":
				proof.ResumeOperationID = "wrong-resume"
			case "generation":
				proof.ContainerGeneration++
			case "old-container":
				j, _ := loadProjectMemberResume(s.db, req.ResumeOperationID)
				proof.Binding = j.Resume.Retirement.Exit.Stop.Binding
			case "process":
				proof.After.StartedAt = proof.After.StartedAt.Add(time.Second)
			case "stale":
				proof.CapturedAt = proof.CapturedAt.Add(-time.Minute)
			case "future":
				proof.CapturedAt = proof.CapturedAt.Add(time.Minute)
			}
			if err = s.completeProjectMemberResumedReadiness(op.ID, proof); err == nil {
				t.Fatal("changed evidence accepted")
			}
			j, _ := loadProjectMemberResumedReadiness(s.db, op.ID)
			if j.Evidence != nil {
				t.Fatal("partial evidence persisted")
			}
		})
	}
}

type resumedReadinessStoreDriver struct {
	proof  projectnetwork.ResumedMemberReadiness
	fault  string
	reads  int
	cancel context.CancelFunc
}

func (d *resumedReadinessStoreDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *resumedReadinessStoreDriver) Inspect(context.Context, projectnetwork.MemberResume) (projectnetwork.MemberResumeObservation, error) {
	d.reads++
	if d.fault == "lost-observation" {
		return projectnetwork.MemberResumeObservation{}, errors.New("fixture read failure")
	}
	return d.proof.Before, nil
}
func (d *resumedReadinessStoreDriver) ReadHostIdentity(context.Context, string) (string, error) {
	if d.fault == "cancel" {
		d.cancel()
	}
	return d.proof.Ship, nil
}
func TestProjectMemberResumedReadinessWorker(t *testing.T) {
	for _, fault := range []string{"pass", "lost-observation", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := resumedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &resumedReadinessStoreDriver{proof: resumedReadinessProof(t, s, p.ID, instance, req.ResumeOperationID), fault: fault, cancel: cancel}
			err = s.runProjectMemberResumedReadiness(ctx, op.ID, d)
			actual, _ := loadOperation(s.db, op.ID)
			if fault == "pass" {
				if err != nil || actual.Status != "succeeded" {
					t.Fatal(actual, err)
				}
			} else if err == nil || actual.Status != "interrupted" {
				t.Fatal(actual, err)
			}
		})
	}
}
func TestProjectMemberResumedReadinessConcurrentAdmission(t *testing.T) {
	s := testStore(t)
	p, req := resumedReadinessProject(t, s)
	instance := p.Members[0].InstanceID
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.RequestID = fmt.Sprintf("reader-%d", i)
			_, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, r)
			if err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if accepted != 1 {
		t.Fatal("overlapping admissions", accepted)
	}
}
func TestProjectMemberResumedReadinessRejectsDamagedJournal(t *testing.T) {
	for _, fault := range []string{"body", "mirror", "orphan"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := resumedReadinessProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberResumedReadiness("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "body":
				_, err = s.db.Exec("UPDATE project_member_resumed_readiness SET body=json_set(body,'$.resume_operation_id','changed') WHERE operation_id=?", op.ID)
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_resumed_readiness WHERE operation_id=?", op.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			if validateProjectMemberResumedReadiness(s.db) == nil {
				t.Fatal("damaged receipt accepted")
			}
		})
	}
}
