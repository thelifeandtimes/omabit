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

func interruptedStopProject(t *testing.T, s *store) (storedProject, projectMemberStopExitRequest) {
	t.Helper()
	p, req, _ := readyStopProject(t, s)
	instance := p.Members[0].InstanceID
	op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberStop(op.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := storeStopObservations(t, s, p.ID, instance)
	advanceStopFixture(t, s, p, op.ID, "stop-dispatched", &before)
	if err = s.interruptMemberStop(op.ID); err != nil {
		t.Fatal(err)
	}
	return p, projectMemberStopExitRequest{RequestID: "observe-retained-stop", StopOperationID: op.ID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
}
func memberStopExitProof(t *testing.T, s *store, id string) projectnetwork.MemberStopExitEvidence {
	t.Helper()
	j, err := loadProjectMemberStopExit(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(j.Stop.Before)
	var o projectnetwork.MemberStopObservation
	if err = json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	o.CapturedAt = time.Now().UTC()
	o.Runtime.ObservedAt = o.CapturedAt
	o.FinishedAt = o.CapturedAt
	for i := range o.Runtime.Members {
		if o.Runtime.Members[i].InstanceID == j.InstanceID {
			o.Runtime.Members[i].Running = false
			o.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
		}
	}
	if j.Stop.After != nil {
		raw, _ := json.Marshal(j.Stop.After)
		if err = json.Unmarshal(raw, &o); err != nil {
			t.Fatal(err)
		}
		o.CapturedAt = time.Now().UTC()
		o.Runtime.ObservedAt = o.CapturedAt
	}
	return projectnetwork.MemberStopExitEvidence{OperationID: id, Stop: j.Stop, Observation: o}
}
func TestProjectMemberStopExitRestartAndReplayPreserveOriginal(t *testing.T) {
	for _, phase := range []string{"planned", "observing", "observed"} {
		t.Run(phase, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := interruptedStopProject(t, s)
			instance := p.Members[0].InstanceID
			original, _ := loadProjectMemberStop(s.db, req.StopOperationID)
			originalOp, _ := loadOperation(s.db, req.StopOperationID)
			op, hit, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			if phase != "planned" {
				if _, err = s.beginProjectMemberStopExit(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "observed" {
				if err = s.completeProjectMemberStopExit(op.ID, memberStopExitProof(t, s, op.ID)); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := loadProjectMemberStopExit(s.db, op.ID)
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
			after, _ := loadProjectMemberStopExit(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("restart changed observation body")
			}
			retained, _ := loadProjectMemberStop(s.db, req.StopOperationID)
			retainedOp, _ := loadOperation(s.db, req.StopOperationID)
			if !reflect.DeepEqual(retained, original) || !reflect.DeepEqual(retainedOp, originalOp) {
				t.Fatal("observation acknowledged or rewrote original Stop")
			}
			replay, hit, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, req)
			want := "interrupted"
			if phase == "observed" {
				want = "succeeded"
			}
			if err != nil || !hit || replay.ID != op.ID || replay.Status != want {
				t.Fatal(replay, hit, err)
			}
			if _, err = s.beginProjectMemberStopExit(op.ID); err == nil {
				t.Fatal("old observation restarted")
			}
			req.RequestID = "observe-again"
			next, hit, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, req)
			if err != nil || hit || next.ID == op.ID {
				t.Fatal(next, hit, err)
			}
			jobs, err := projectMemberStopExitJobs(s.db)
			if err != nil || len(jobs) != 2 {
				t.Fatal("attempt history lost", err)
			}
		})
	}
}
func TestProjectMemberStopExitRejectsInvalidCompletion(t *testing.T) {
	for _, fault := range []string{"not-begun", "operation", "original-operation", "original-phase", "storage", "process", "peer", "incomplete", "stale", "future"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := interruptedStopProject(t, s)
			op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			if fault != "not-begun" {
				if _, err = s.beginProjectMemberStopExit(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			proof := memberStopExitProof(t, s, op.ID)
			switch fault {
			case "operation":
				proof.OperationID = "other-observation"
			case "original-operation":
				proof.Stop.OperationID = "other-stop"
			case "original-phase":
				proof.Stop.Phase = "stop-signalled"
			case "storage":
				proof.Observation.Storage.Identity.Inode++
			case "process":
				proof.Observation.StartedAt = proof.Observation.StartedAt.Add(time.Second)
			case "peer":
				for i := range proof.Observation.Runtime.Members {
					if proof.Observation.Runtime.Members[i].InstanceID != op.InstanceID {
						proof.Observation.Runtime.Members[i].Running = false
						proof.Observation.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
					}
				}
			case "incomplete":
				proof.Observation.Runtime.Complete = false
			case "stale":
				proof.Observation.CapturedAt = time.Now().Add(-31 * time.Second)
				proof.Observation.Runtime.ObservedAt = proof.Observation.CapturedAt
			case "future":
				proof.Observation.CapturedAt = time.Now().Add(6 * time.Second)
				proof.Observation.Runtime.ObservedAt = proof.Observation.CapturedAt
			}
			before, _ := loadProjectMemberStopExit(s.db, op.ID)
			if err = s.completeProjectMemberStopExit(op.ID, proof); err == nil {
				t.Fatal("invalid completion accepted")
			}
			after, _ := loadProjectMemberStopExit(s.db, op.ID)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("partial evidence committed")
			}
		})
	}
}
func TestProjectMemberStopExitAdmissionAndConcurrency(t *testing.T) {
	s := testStore(t)
	p, req := interruptedStopProject(t, s)
	instance := p.Members[0].InstanceID
	var wg sync.WaitGroup
	accepted := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.RequestID = fmt.Sprintf("observe-%d", i)
			op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, r)
			if err == nil {
				accepted <- op.ID
			}
		}(i)
	}
	wg.Wait()
	close(accepted)
	if len(accepted) != 1 {
		t.Fatal("overlapping observations", len(accepted))
	}
	id := <-accepted
	if err := s.interruptMemberStopExit(id); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"actor", "revision", "review", "instance", "stop", "replay-only"} {
		r := req
		r.RequestID = "invalid-" + fault
		actor, target := "actor", instance
		switch fault {
		case "actor":
			actor = ""
		case "revision":
			r.ExpectedRevision++
		case "review":
			r.ReviewDigest = "wrong"
		case "instance":
			target = p.Members[1].InstanceID
		case "stop":
			r.StopOperationID = "missing-stop"
		case "replay-only":
			r.ReplayOnly = true
		}
		if _, _, err := s.acceptProjectMemberStopExit(actor, p.ID, target, r); err == nil {
			t.Fatal("invalid admission", fault)
		}
	}
}
func TestProjectMemberStopExitRefusesUnstartedOrActiveStop(t *testing.T) {
	for _, mode := range []string{"queued", "running-planned", "interrupted-planned", "running-dispatched"} {
		t.Run(mode, func(t *testing.T) {
			s := testStore(t)
			p, req, _ := readyStopProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "queued" {
				if _, err = s.beginProjectMemberStop(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "running-dispatched" {
				before, _ := storeStopObservations(t, s, p.ID, instance)
				advanceStopFixture(t, s, p, op.ID, "stop-dispatched", &before)
			}
			if mode == "interrupted-planned" {
				if err = s.interruptMemberStop(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			_, _, err = s.acceptProjectMemberStopExit("actor", p.ID, instance, projectMemberStopExitRequest{RequestID: "observe", StopOperationID: op.ID, ExpectedRevision: 1, ReviewDigest: req.ReviewDigest})
			requireFault(t, err, "project_stop_not_observable")
		})
	}
}
func TestProjectMemberStopExitCorruptionRefusesReopen(t *testing.T) {
	for _, fault := range []string{"mirror", "orphan", "phase", "stop-binding"} {
		t.Run(fault, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := interruptedStopProject(t, s)
			op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			j, _ := loadProjectMemberStopExit(s.db, op.ID)
			switch fault {
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_stop_exits WHERE operation_id=?", op.ID)
			default:
				if fault == "phase" {
					j.Phase = "observed"
				} else {
					j.Stop.OperationID = "different-stop"
				}
				raw, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_member_stop_exits SET body=? WHERE operation_id=?", raw, op.ID)
				if err == nil {
					_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			next, err := testOpenStore(dir)
			if err == nil {
				next.close()
				t.Fatal("corrupt observation accepted")
			}
		})
	}
}

type storedStopExitDriver struct {
	t      *testing.T
	s      *store
	id     string
	proof  projectnetwork.MemberStopExitEvidence
	reads  int
	fault  string
	cancel context.CancelFunc
}

func (d *storedStopExitDriver) check() {
	d.t.Helper()
	j, err := loadProjectMemberStopExit(d.s.db, d.id)
	if err != nil || j.Phase != "observing" {
		d.t.Fatal("read before observation commit", err)
	}
	if err = validateProjectMemberStopExit(d.s.db); err != nil {
		d.t.Fatal(err)
	}
}
func (d *storedStopExitDriver) VerifyEngine(context.Context, string) error { d.check(); return nil }
func (d *storedStopExitDriver) InspectExit(_ context.Context, retained projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	d.check()
	d.reads++
	if !reflect.DeepEqual(retained, d.proof.Stop) {
		d.t.Fatal("original Stop changed")
	}
	if d.fault == "cancel" {
		d.cancel()
	}
	if d.fault == "read" {
		return d.proof.Observation, errors.New("synthetic read failure")
	}
	return d.proof.Observation, nil
}
func TestProjectMemberStopExitWorkerPreservesUncertainty(t *testing.T) {
	for _, fault := range []string{"pass", "unclean", "read", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := interruptedStopProject(t, s)
			original, _ := loadOperation(s.db, req.StopOperationID)
			op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &storedStopExitDriver{t: t, s: s, id: op.ID, proof: memberStopExitProof(t, s, op.ID), fault: fault, cancel: cancel}
			if fault == "unclean" {
				d.proof.Observation.ExitCode = 137
				d.proof.Observation.OOMKilled = true
			}
			err = s.runProjectMemberStopExit(ctx, op.ID, d)
			actual, _ := loadOperation(s.db, op.ID)
			j, _ := loadProjectMemberStopExit(s.db, op.ID)
			pass := fault == "pass" || fault == "unclean"
			if (err == nil) != pass || pass && (actual.Status != "succeeded" || j.Evidence == nil) || !pass && (actual.Status != "interrupted" || j.Evidence != nil) || d.reads != 1 {
				t.Fatal(actual, j, err)
			}
			if pass && j.Evidence.Clean() != (fault == "pass") {
				t.Fatal("unclean exit lost")
			}
			retained, _ := loadOperation(s.db, req.StopOperationID)
			if !reflect.DeepEqual(original, retained) || retained.Status != "interrupted" {
				t.Fatal("read observation resolved original acknowledgement")
			}
		})
	}
}

func TestProjectMemberStopExitForeignKeyPreservesBinding(t *testing.T) {
	s := testStore(t)
	p, req := interruptedStopProject(t, s)
	op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, p.Members[0].InstanceID, req)
	if err != nil {
		t.Fatal(err)
	}
	before, err := loadProjectMemberStopExit(s.db, op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE project_member_stop_exits SET stop_operation_id=operation_id WHERE operation_id=?", op.ID); err == nil {
		t.Fatal("dangling original Stop accepted")
	}
	after, err := loadProjectMemberStopExit(s.db, op.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("failed foreign-key update changed binding", err)
	}
	if err = validateProjectMemberStopExit(s.db); err != nil {
		t.Fatal(err)
	}
}
