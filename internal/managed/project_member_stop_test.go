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

func readyStopProject(t *testing.T, s *store) (storedProject, projectMemberStopRequest, map[string]string) {
	t.Helper()
	p, req := launchedReadinessProject(t, s)
	ids := map[string]string{}
	for _, m := range p.Members {
		req.RequestID = "observe-" + m.InstanceID
		op, _, err := s.acceptProjectMemberReadiness("actor", p.ID, m.InstanceID, req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.beginProjectMemberReadiness(op.ID); err != nil {
			t.Fatal(err)
		}
		if err = s.completeProjectMemberReadiness(op.ID, memberReadinessProof(t, s, p.ID, m.InstanceID)); err != nil {
			t.Fatal(err)
		}
		ids[m.InstanceID] = op.ID
	}
	return p, projectMemberStopRequest{RequestID: "stop-member", ExpectedRevision: 1, ReviewDigest: req.ReviewDigest, ReadinessOperationID: ids[p.Members[0].InstanceID]}, ids
}
func storeStopObservations(t *testing.T, s *store, project, instance string) (projectnetwork.MemberStopObservation, projectnetwork.MemberStopObservation) {
	t.Helper()
	proof := memberReadinessProof(t, s, project, instance)
	now := time.Now().UTC()
	before := projectnetwork.MemberStopObservation{Runtime: proof.Before, Storage: proof.Storage, StartedAt: now.Add(-time.Minute), CapturedAt: now}
	raw, _ := json.Marshal(before)
	var after projectnetwork.MemberStopObservation
	if err := json.Unmarshal(raw, &after); err != nil {
		t.Fatal(err)
	}
	after.CapturedAt = now.Add(time.Millisecond)
	after.FinishedAt = after.CapturedAt
	after.Runtime.ObservedAt = after.CapturedAt
	for i := range after.Runtime.Members {
		if after.Runtime.Members[i].InstanceID == instance {
			after.Runtime.Members[i].Running = false
			after.Runtime.Members[i].Namespace = projectnetwork.NamespaceIdentity{}
		}
	}
	return before, after
}
func advanceStopFixture(t *testing.T, s *store, p storedProject, id, phase string, o *projectnetwork.MemberStopObservation) {
	t.Helper()
	j, err := loadProjectMemberStop(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	next, err := j.Stop.Advance(p.Plan, j.Stop.Phase, phase, o)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.advanceProjectMemberStop(id, j.Stop.Phase, next); err != nil {
		t.Fatal(err)
	}
}
func TestProjectMemberStopRestartRetainsEveryCommitBoundary(t *testing.T) {
	for _, cut := range []string{"accepted", "begun", "dispatch", "signal", "exit", "finished", "unclean-exit"} {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req, ids := readyStopProject(t, s)
			instance := p.Members[0].InstanceID
			originalLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			originalReady, _ := loadProjectMemberReadiness(s.db, req.ReadinessOperationID)
			op, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil || hit {
				t.Fatal(op, hit, err)
			}
			before, after := storeStopObservations(t, s, p.ID, instance)
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
			if (got.Stop.Phase == "stopped") != (err == nil) {
				t.Fatal("incorrect peer Stop admission", got.Stop.Phase, err)
			}
			afterLaunch, _ := loadProjectMemberLaunch(s.db, instance)
			afterReady, _ := loadProjectMemberReadiness(s.db, originalReady.OperationID)
			if !reflect.DeepEqual(originalLaunch, afterLaunch) || !reflect.DeepEqual(originalReady, afterReady) {
				t.Fatal("Stop rewrote original receipts")
			}
		})
	}
}
func TestProjectMemberStopAdmissionRejectsUnboundSelection(t *testing.T) {
	s := testStore(t)
	p, req, ids := readyStopProject(t, s)
	instance := p.Members[0].InstanceID
	for _, fault := range []string{"actor", "revision", "review", "readiness", "wrong-member", "replay-only"} {
		r := req
		a := "actor"
		switch fault {
		case "actor":
			a = ""
		case "revision":
			r.ExpectedRevision++
		case "review":
			r.ReviewDigest = "wrong"
		case "readiness":
			r.ReadinessOperationID = "missing"
		case "wrong-member":
			r.ReadinessOperationID = ids[p.Members[1].InstanceID]
		case "replay-only":
			r.ReplayOnly = true
		}
		if _, _, err := s.acceptProjectMemberStop(a, p.ID, instance, r); err == nil {
			t.Fatal(fault)
		}
	}
	op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
	if err != nil {
		t.Fatal(err)
	}
	changed := req
	changed.ReadinessOperationID = ids[p.Members[1].InstanceID]
	if _, _, err = s.acceptProjectMemberStop("actor", p.ID, instance, changed); err == nil {
		t.Fatal("changed replay accepted")
	}
	req.ReplayOnly = true
	again, hit, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
	if err != nil || !hit || again.ID != op.ID {
		t.Fatal(again, hit, err)
	}
}
func TestProjectMemberStopConcurrentAdmissionIsSingleIntent(t *testing.T) {
	s := testStore(t)
	p, req, _ := readyStopProject(t, s)
	var wg sync.WaitGroup
	success := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := req
			r.RequestID = fmt.Sprintf("stop-%d", i)
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, p.Members[0].InstanceID, r)
			if err == nil {
				success <- op.ID
			}
		}(i)
	}
	wg.Wait()
	close(success)
	if len(success) != 1 {
		t.Fatal("duplicate Stop admitted", len(success))
	}
}
func TestProjectMemberStopRefusesTamperedTransitionAndMirror(t *testing.T) {
	for _, fault := range []string{"binding", "old-proof", "future-proof", "mirror", "orphan", "ownership", "success-without-exit"} {
		t.Run(fault, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			p, req, _ := readyStopProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			state, err := s.beginProjectMemberStop(op.ID)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := storeStopObservations(t, s, p.ID, instance)
			if fault == "binding" || fault == "old-proof" || fault == "future-proof" {
				if fault == "old-proof" {
					before.CapturedAt = time.Now().UTC().Add(-31 * time.Second)
					before.Runtime.ObservedAt = before.CapturedAt
				}
				if fault == "future-proof" {
					before.CapturedAt = time.Now().UTC().Add(6 * time.Second)
					before.Runtime.ObservedAt = before.CapturedAt
				}
				next, err := state.Advance(p.Plan, "planned", "stop-dispatched", &before)
				if err != nil {
					t.Fatal(err)
				}
				if fault == "binding" {
					next.ReadinessOperationID = "other"
				}
				if err = s.advanceProjectMemberStop(op.ID, "planned", next); err == nil {
					t.Fatal("invalid transition accepted")
				}
				j, _ := loadProjectMemberStop(s.db, op.ID)
				if !reflect.DeepEqual(j.Stop, state) {
					t.Fatal("invalid transition partially committed")
				}
				s.close()
				return
			}
			switch fault {
			case "mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "orphan":
				_, err = s.db.Exec("DELETE FROM project_member_stops WHERE operation_id=?", op.ID)
			case "ownership":
				j, _ := loadProjectMemberStop(s.db, op.ID)
				j.Stop.ReadinessOperationID = "other"
				raw, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_member_stops SET body=? WHERE operation_id=?", raw, op.ID)
				if err == nil {
					_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", raw, op.ID)
				}
			case "success-without-exit":
				actual, _ := loadOperation(s.db, op.ID)
				actual.Status = "succeeded"
				tx, e := s.db.Begin()
				if e != nil {
					t.Fatal(e)
				}
				err = saveOperation(tx, actual)
				if err == nil {
					err = tx.Commit()
				} else {
					tx.Rollback()
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			s.close()
			opened, err := testOpenStore(dir)
			if err == nil {
				opened.close()
				t.Fatal("corrupt Stop accepted")
			}
		})
	}
}

type storedStopDriver struct {
	t             *testing.T
	s             *store
	id            string
	before, after projectnetwork.MemberStopObservation
	fault         string
	signals       int
	cancel        context.CancelFunc
}

func (d *storedStopDriver) check(state projectnetwork.MemberStop, phase string) {
	d.t.Helper()
	j, err := loadProjectMemberStop(d.s.db, d.id)
	if err != nil || j.Stop.Phase != phase || !reflect.DeepEqual(j.Stop, state) {
		d.t.Fatal("Stop effect before committed journal", err)
	}
	if err = validateProjectMemberStops(d.s.db); err != nil {
		d.t.Fatal(err)
	}
}
func (d *storedStopDriver) VerifyEngine(context.Context, string) error { return nil }
func (d *storedStopDriver) Inspect(_ context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	d.check(s, "planned")
	return d.before, nil
}
func (d *storedStopDriver) SignalStop(_ context.Context, s projectnetwork.MemberStop) error {
	d.check(s, "stop-dispatched")
	d.signals++
	if d.fault == "signal" {
		return errors.New("synthetic signal uncertainty")
	}
	if d.fault == "cancel" {
		d.cancel()
	}
	return nil
}
func (d *storedStopDriver) AwaitStopped(_ context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	d.check(s, "stop-signalled")
	if d.fault == "wait" {
		return projectnetwork.MemberStopObservation{}, errors.New("synthetic wait failure")
	}
	if d.fault == "late-wait" {
		return d.after, errors.New("synthetic late wait error")
	}
	return d.after, nil
}
func TestProjectMemberStopWorkerKeepsExitAndUncertainty(t *testing.T) {
	for _, fault := range []string{"pass", "signal", "cancel", "wait", "late-wait", "unclean", "oom"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req, _ := readyStopProject(t, s)
			instance := p.Members[0].InstanceID
			op, _, err := s.acceptProjectMemberStop("actor", p.ID, instance, req)
			if err != nil {
				t.Fatal(err)
			}
			before, after := storeStopObservations(t, s, p.ID, instance)
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
