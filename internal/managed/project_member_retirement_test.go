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

func readyRetirementProject(t *testing.T, s *store) (storedProject, projectMemberRetirementRequest) {
	t.Helper()
	p, req := interruptedStopProject(t, s)
	op, _, err := s.acceptProjectMemberStopExit("actor", p.ID, p.Members[0].InstanceID, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.beginProjectMemberStopExit(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.completeProjectMemberStopExit(op.ID, memberStopExitProof(t, s, op.ID)); err != nil {
		t.Fatal(err)
	}
	return p, projectMemberRetirementRequest{RequestID: "retire-member", ExitOperationID: op.ID, ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest}
}
func retirementStoreObservations(t *testing.T, s *store, id string) (projectnetwork.MemberStopObservation, projectnetwork.MemberAbsenceObservation) {
	t.Helper()
	j, err := loadProjectMemberRetirement(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(j.Retirement.Exit.Observation)
	var before projectnetwork.MemberStopObservation
	if err = json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	before.CapturedAt = time.Now().UTC()
	before.Runtime.ObservedAt = before.CapturedAt
	raw, _ = json.Marshal(before.Runtime)
	var runtime projectnetwork.Observation
	if err = json.Unmarshal(raw, &runtime); err != nil {
		t.Fatal(err)
	}
	after := projectnetwork.MemberAbsenceObservation{Runtime: runtime, Storage: before.Storage, CapturedAt: time.Now().UTC()}
	after.Runtime.ObservedAt = after.CapturedAt
	after.Runtime.Members = nil
	for _, m := range before.Runtime.Members {
		if m.InstanceID != j.InstanceID {
			after.Runtime.Members = append(after.Runtime.Members, m)
		}
	}
	return before, after
}
func advanceRetirementFixture(t *testing.T, s *store, p storedProject, id, next string, b *projectnetwork.MemberStopObservation, a *projectnetwork.MemberAbsenceObservation) {
	t.Helper()
	j, err := loadProjectMemberRetirement(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	wanted, err := j.Retirement.Advance(p.Plan, j.Retirement.Phase, next, b, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.advanceProjectMemberRetirement(id, j.Retirement.Phase, wanted); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberRetirementRestartPreservesEveryBoundary(t *testing.T) {
	for _, cut := range []string{"accepted", "begun", "dispatch", "removed", "absent", "finished"} {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := readyRetirementProject(t, s)
			instance := p.Members[0].InstanceID
			originalExit, _ := loadProjectMemberStopExit(s.db, req.ExitOperationID)
			originalStop, _ := loadProjectMemberStop(s.db, originalExit.StopOperationID)
			originalStopOp, _ := loadOperation(s.db, originalExit.StopOperationID)
			op, hit, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			b, a := retirementStoreObservations(t, s, op.ID)
			if cut != "accepted" {
				if _, err = s.beginProjectMemberRetirement(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			if cut != "accepted" && cut != "begun" {
				advanceRetirementFixture(t, s, p, op.ID, "remove-dispatched", &b, nil)
			}
			if cut == "removed" || cut == "absent" || cut == "finished" {
				advanceRetirementFixture(t, s, p, op.ID, "removed", nil, nil)
			}
			if cut == "absent" || cut == "finished" {
				advanceRetirementFixture(t, s, p, op.ID, "absent", nil, &a)
			}
			if cut == "finished" {
				if err = s.finishProjectMemberRetirement(op.ID); err != nil {
					t.Fatal(err)
				}
			}
			retained, _ := loadProjectMemberRetirement(s.db, op.ID)
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
			got, err := loadProjectMemberRetirement(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(got, retained) {
				t.Fatal("retirement journal changed", err)
			}
			stop, _ := loadProjectMemberStop(s.db, originalExit.StopOperationID)
			exit, _ := loadProjectMemberStopExit(s.db, req.ExitOperationID)
			stopOp, _ := loadOperation(s.db, originalExit.StopOperationID)
			if !reflect.DeepEqual(stop, originalStop) || !reflect.DeepEqual(exit, originalExit) || !reflect.DeepEqual(stopOp, originalStopOp) {
				t.Fatal("retirement rewrote original evidence")
			}
			again, hit, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
			want := "interrupted"
			if cut == "finished" {
				want = "succeeded"
			}
			if err != nil || !hit || again.ID != op.ID || again.Status != want {
				t.Fatal(again, hit, err)
			}
			if _, err = s.beginProjectMemberRetirement(op.ID); err == nil {
				t.Fatal("retirement replayed")
			}
			req.RequestID = "another-retirement"
			if _, _, err = s.acceptProjectMemberRetirement("actor", p.ID, instance, req); err == nil {
				t.Fatal("new request removed same container twice")
			}
		})
	}
}
func TestProjectMemberRetirementRejectsChangedTransitions(t *testing.T) {
	for _, fault := range []string{"operation", "exit", "phase", "stale", "future", "storage", "peer", "unacknowledged-absence"} {
		t.Run(fault, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, req := readyRetirementProject(t, s)
			op, _, err := s.acceptProjectMemberRetirement("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			initial, err := s.beginProjectMemberRetirement(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			b, a := retirementStoreObservations(t, s, op.ID)
			next, err := initial.Advance(p.Plan, "planned", "remove-dispatched", &b, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "operation":
				next.OperationID = "different"
			case "exit":
				next.Exit.OperationID = "different"
			case "phase":
				next.Phase = "removed"
			case "stale":
				next.Before.CapturedAt = time.Now().Add(-31 * time.Second)
				next.Before.Runtime.ObservedAt = next.Before.CapturedAt
			case "future":
				next.Before.CapturedAt = time.Now().Add(6 * time.Second)
				next.Before.Runtime.ObservedAt = next.Before.CapturedAt
			case "storage":
				next.Before.Storage.Identity.Inode++
			case "peer":
				next.Before.Runtime.Members[0].Resource.ID = "different"
			case "unacknowledged-absence":
				next.Phase = "absent"
				next.After = &a
			}
			if err = s.advanceProjectMemberRetirement(op.ID, "planned", next); err == nil {
				t.Fatal("invalid transition committed")
			}
			got, _ := loadProjectMemberRetirement(s.db, op.ID)
			if !reflect.DeepEqual(got.Retirement, initial) {
				t.Fatal("failed transition changed journal")
			}
		})
	}
}
func TestProjectMemberRetirementAdmissionAndConcurrency(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, req := readyRetirementProject(t, s)
	instance := p.Members[0].InstanceID
	for _, fault := range []string{"revision", "digest", "exit", "replay-only"} {
		q := req
		switch fault {
		case "revision":
			q.ExpectedRevision++
		case "digest":
			q.ReviewDigest = "wrong"
		case "exit":
			q.ExitOperationID = "missing"
		case "replay-only":
			q.ReplayOnly = true
		}
		if _, _, err = s.acceptProjectMemberRetirement("actor", p.ID, instance, q); err == nil {
			t.Fatal("invalid admission", fault)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q := req
			q.RequestID = fmt.Sprintf("retirement-%d", i)
			_, _, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, q)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatal("overlapping retirement", accepted)
	}
	if err = validateProjectMemberRetirements(s.db); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberRetirementCorruptionRefusesReopen(t *testing.T) {
	for _, fault := range []string{"body", "mirror", "orphan"} {
		t.Run(fault, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req := readyRetirementProject(t, s)
			op, _, err := s.acceptProjectMemberRetirement("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			sql := map[string]string{"body": "UPDATE project_member_retirements SET body='{}' WHERE operation_id=?", "mirror": "UPDATE operations SET runtime='{}' WHERE id=?", "orphan": "DELETE FROM project_member_retirements WHERE operation_id=?"}[fault]
			if _, err = s.db.Exec(sql, op.ID); err != nil {
				t.Fatal(err)
			}
			if err = s.close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := testOpenStore(dir)
			if err == nil {
				reopened.close()
				t.Fatal("corrupt retirement accepted")
			}
		})
	}
}

type storedRetirementDriver struct {
	t         *testing.T
	s         *store
	id, fault string
	before    projectnetwork.MemberStopObservation
	after     projectnetwork.MemberAbsenceObservation
	removes   int
}

func (d *storedRetirementDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *storedRetirementDriver) check(state projectnetwork.MemberRetirement, phase string) {
	d.t.Helper()
	j, err := loadProjectMemberRetirement(d.s.db, d.id)
	op, e := loadOperation(d.s.db, d.id)
	if err != nil || e != nil || op.Status != "running" || state.Phase != phase || !reflect.DeepEqual(state, j.Retirement) {
		d.t.Fatal("effect before durable intent", err, e)
	}
}
func (d *storedRetirementDriver) InspectStopped(_ context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberStopObservation, error) {
	d.check(s, "planned")
	return d.before, nil
}
func (d *storedRetirementDriver) RemoveStopped(_ context.Context, s projectnetwork.MemberRetirement) error {
	d.check(s, "remove-dispatched")
	d.removes++
	if d.fault == "remove" {
		return errors.New("lost reply")
	}
	return nil
}
func (d *storedRetirementDriver) InspectAbsent(_ context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberAbsenceObservation, error) {
	d.check(s, "removed")
	if d.fault == "absence" {
		return d.after, errors.New("read failed")
	}
	return d.after, nil
}
func TestProjectMemberRetirementWorkerRetainsOriginalReceipts(t *testing.T) {
	for _, fault := range []string{"", "remove", "absence", "cancel"} {
		t.Run(fault, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, req := readyRetirementProject(t, s)
			op, _, err := s.acceptProjectMemberRetirement("actor", p.ID, p.Members[0].InstanceID, req)
			if err != nil {
				t.Fatal(err)
			}
			b, a := retirementStoreObservations(t, s, op.ID)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancel" {
				cancel()
			}
			d := &storedRetirementDriver{t: t, s: s, id: op.ID, fault: fault, before: b, after: a}
			err = s.runProjectMemberRetirement(ctx, op.ID, d)
			if (err == nil) != (fault == "") {
				t.Fatal(err)
			}
			receipt, _ := loadOperation(s.db, op.ID)
			j, _ := loadProjectMemberRetirement(s.db, op.ID)
			want := "interrupted"
			if fault == "" {
				want = "succeeded"
			}
			if receipt.Status != want {
				t.Fatal(receipt)
			}
			original, _ := loadOperation(s.db, j.Retirement.Exit.Stop.OperationID)
			if original.Status != "interrupted" {
				t.Fatal("original Stop outcome rewritten")
			}
			calls := d.removes
			if err = s.runProjectMemberRetirement(context.Background(), op.ID, d); err == nil || d.removes != calls {
				t.Fatal("worker replayed removal")
			}
		})
	}
}

func TestProjectMemberRetirementRequiresCompleteCleanBoundExit(t *testing.T) {
	for _, mode := range []string{"unobserved", "unclean", "foreign-member", "foreign-key"} {
		t.Run(mode, func(t *testing.T) {
			s, err := testOpenStore(privateDir(t))
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			p, exitReq := interruptedStopProject(t, s)
			instance := p.Members[0].InstanceID
			observed, _, err := s.acceptProjectMemberStopExit("actor", p.ID, instance, exitReq)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "unobserved" {
				if _, err = s.beginProjectMemberStopExit(observed.ID); err != nil {
					t.Fatal(err)
				}
				proof := memberStopExitProof(t, s, observed.ID)
				if mode == "unclean" {
					proof.Observation.ExitCode = 137
				}
				if err = s.completeProjectMemberStopExit(observed.ID, proof); err != nil {
					t.Fatal(err)
				}
			}
			req := projectMemberRetirementRequest{RequestID: "retire-member", ExpectedRevision: p.Revision, ReviewDigest: exitReq.ReviewDigest, ExitOperationID: observed.ID}
			if mode == "foreign-member" {
				instance = p.Members[1].InstanceID
			}
			op, _, err := s.acceptProjectMemberRetirement("actor", p.ID, instance, req)
			if mode != "foreign-key" {
				if err == nil {
					t.Fatal("unqualified exit accepted", mode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			before, err := loadProjectMemberRetirement(s.db, op.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.db.Exec("UPDATE project_member_retirements SET exit_operation_id='missing' WHERE operation_id=?", op.ID); err == nil {
				t.Fatal("dangling exit reference accepted")
			}
			after, err := loadProjectMemberRetirement(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("rejected foreign key changed journal", err)
			}
		})
	}
}
