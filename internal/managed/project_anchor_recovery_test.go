package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

func recoveryStoreProject(t *testing.T, s *store) (storedProject, projectAnchorRecoveryRequest) {
	t.Helper()
	p, instance, req := contextReadinessProject(t, s, 0)
	op, _, err := s.acceptProjectMemberContextReadiness("actor", p.ID, instance, req)
	g02Check(t, err)
	_, err = s.beginProjectMemberContextReadiness(op.ID)
	g02Check(t, err)
	g02Check(t, s.completeProjectMemberContextReadiness(op.ID, contextReadinessProof(t, s, op.ID)))
	h, _, _, err := projectRecoveryHistory(s.db, p.ID, true)
	g02Check(t, err)
	return p, projectAnchorRecoveryRequest{RequestID: "repair-anchor", ExpectedRevision: p.Revision, ReviewDigest: req.ReviewDigest, HistoryDigest: recoveryHistoryDigest(h)}
}

type recoveryStoreObserver struct {
	o projectnetwork.AnchorRecoveryObservation
}

func (f *recoveryStoreObserver) VerifyEngine(context.Context, string) error { return nil }
func (f *recoveryStoreObserver) InspectAnchorRecovery(context.Context, projectnetwork.AnchorRecoveryContext) (projectnetwork.AnchorRecoveryObservation, error) {
	return f.o, nil
}
func recoveryStoreReview(t *testing.T, j projectAnchorRecoveryJob) projectnetwork.AnchorRecoveryReview {
	t.Helper()
	c, err := projectnetwork.SelectAnchorRecoveryHistoryContext(j.Source)
	g02Check(t, err)
	raw, _ := json.Marshal(j.Source.Originals[0].After)
	var runtime projectnetwork.Observation
	g02Check(t, json.Unmarshal(raw, &runtime))
	now := time.Now().UTC()
	runtime.ObservedAt = now
	runtime.Anchor.Namespace = c.Binding.Namespace
	runtime.Anchor.Namespace.Inode++
	for i := range runtime.Members {
		for _, m := range c.Binding.Members {
			if runtime.Members[i].InstanceID == m.InstanceID {
				runtime.Members[i].Resource = m.Container
				runtime.Members[i].Namespace = c.Binding.Namespace
			}
		}
	}
	o := projectnetwork.AnchorRecoveryObservation{Runtime: runtime, CapturedAt: now}
	for _, m := range c.Members {
		started := m.StartedAt
		if started.IsZero() {
			started = now.Add(-time.Minute)
		}
		o.Processes = append(o.Processes, projectnetwork.AnchorRecoveryProcess{InstanceID: m.Context.Member.InstanceID, Storage: m.Context.Storage, StartedAt: started})
	}
	r, err := projectnetwork.CollectAnchorRecoveryHistoryReview(context.Background(), j.Source, &recoveryStoreObserver{o}, func() time.Time { return now })
	g02Check(t, err)
	return r
}
func recoveryStoreDrainObservation(j projectAnchorRecoveryJob) projectnetwork.AnchorDrainObservation {
	r := j.Journal.Drain.Review.After
	o := projectnetwork.AnchorDrainObservation{Runtime: r.Runtime, CapturedAt: r.CapturedAt, AnchorProcess: projectnetwork.RecoveryProcessState{StartedAt: r.CapturedAt.Add(-time.Minute)}}
	for _, p := range r.Processes {
		o.Processes = append(o.Processes, projectnetwork.AnchorDrainProcess{InstanceID: p.InstanceID, Storage: p.Storage, Process: projectnetwork.RecoveryProcessState{StartedAt: p.StartedAt}})
	}
	return o
}
func advanceRecoveryStoreFixture(t *testing.T, s *store, j projectAnchorRecoveryJob, phase string) projectAnchorRecoveryJob {
	t.Helper()
	var u projectnetwork.AnchorRecoveryJournal
	switch phase {
	case "reviewed":
		r := recoveryStoreReview(t, j)
		u.Review = &r
	case "stop-dispatched":
		o := recoveryStoreDrainObservation(j)
		d, err := j.Journal.Drain.Advance(j.Source, 0, j.Journal.Drain.Sequence, phase, &o)
		g02Check(t, err)
		u.Drain = &d
		phase = "draining"
	}
	next, err := s.advanceProjectAnchorRecovery(j.OperationID, j.Journal.Sequence, phase, u)
	g02Check(t, err)
	return next
}
func TestAnchorRecoveryStoreRestartKeepsIntentAndCreatorReceipts(t *testing.T) {
	for _, cut := range []string{"accepted", "reviewing", "reviewed", "draining", "stop-dispatched"} {
		t.Run(cut, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			p, req := recoveryStoreProject(t, s)
			old := currentStopHistory(t, s)
			op, hit, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
			g02Check(t, err)
			if hit {
				t.Fatal("unexpected replay")
			}
			j, err := loadProjectAnchorRecovery(s.db, op.ID)
			g02Check(t, err)
			for _, phase := range []string{"reviewing", "reviewed", "draining", "stop-dispatched"} {
				if cut == "accepted" {
					break
				}
				j = advanceRecoveryStoreFixture(t, s, j, phase)
				if phase == cut {
					break
				}
			}
			if err = projectRetirementClear(s.db, p.ID, ""); err == nil {
				t.Fatal("active recovery did not fence project")
			}
			assertContextHistory(t, s, old)
			g02Check(t, validateProjectAnchorRecoveries(s.db))
			g02Check(t, s.close())
			s, err = testOpenStore(dir)
			g02Check(t, err)
			defer s.close()
			g02Check(t, s.reconcileInterrupted())
			saved, err := loadProjectAnchorRecovery(s.db, op.ID)
			g02Check(t, err)
			if !reflect.DeepEqual(saved, j) {
				t.Fatal("restart changed retained evidence")
			}
			replay, hit, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
			g02Check(t, err)
			if !hit || replay.ID != op.ID || replay.Status != "interrupted" {
				t.Fatal("actor replay lost exact receipt")
			}
			if _, err = s.advanceProjectAnchorRecovery(op.ID, saved.Journal.Sequence, "reviewing", projectnetwork.AnchorRecoveryJournal{}); err == nil {
				t.Fatal("reopening replayed interrupted intent")
			}
			req.RequestID = "conflicting-recovery"
			if _, _, err = s.acceptProjectAnchorRecovery("actor", p.ID, req); err == nil {
				t.Fatal("new recovery crossed uncertain fence")
			}
			for i, id := range saved.Journal.IDs.Members {
				child, err := loadOperation(s.db, id.OperationID)
				g02Check(t, err)
				if child.Status != "interrupted" || child.Actor != "actor" || child.InstanceID != id.InstanceID {
					t.Fatal("child lost ownership")
				}
				g02Check(t, projectRecoveryMirror(s.db, child.ID, recoveryMemberReceipt(saved, i)))
			}
			g02Check(t, validateProjectAnchorRecoveries(s.db))
			assertContextHistory(t, s, old)
		})
	}
}
func TestAnchorRecoveryStoreReviewFailureAndReplay(t *testing.T) {
	s := testStore(t)
	p, req := recoveryStoreProject(t, s)
	op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err := loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	advanceRecoveryStoreFixture(t, s, j, "reviewing")
	g02Check(t, s.interruptAnchorRecovery(op.ID, true))
	g02Check(t, projectRetirementClear(s.db, p.ID, ""))
	replay, hit, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	if !hit || replay.Status != "failed" {
		t.Fatal("failed review replay changed")
	}
	changed := req
	changed.HistoryDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, _, err = s.acceptProjectAnchorRecovery("actor", p.ID, changed); err == nil {
		t.Fatal("changed request replay accepted")
	}
	req.RequestID = "new-review"
	op, _, err = s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err = loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	j = advanceRecoveryStoreFixture(t, s, j, "reviewing")
	j = advanceRecoveryStoreFixture(t, s, j, "reviewed")
	if err = s.interruptAnchorRecovery(op.ID, true); err == nil {
		t.Fatal("retained fault review discarded")
	}
	g02Check(t, validateProjectAnchorRecoveries(s.db))
}
func TestAnchorRecoveryStoreRejectsChangedEvidence(t *testing.T) {
	for _, fault := range []string{"parent-mirror", "child-mirror", "child-actor", "child-binding", "original-mirror", "current-mirror", "missing-resume", "changed-source", "rewritten-source-mirror", "orphan-child", "duplicate-parent"} {
		t.Run(fault, func(t *testing.T) {
			s := testStore(t)
			p, req := recoveryStoreProject(t, s)
			op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
			g02Check(t, err)
			j, err := loadProjectAnchorRecovery(s.db, op.ID)
			g02Check(t, err)
			switch fault {
			case "parent-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", op.ID)
			case "child-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", j.Journal.IDs.Members[0].OperationID)
			case "child-actor":
				_, err = s.db.Exec("UPDATE operations SET body=json_set(body,'$.actor','foreign') WHERE id=?", j.Journal.IDs.Members[0].OperationID)
			case "child-binding":
				_, err = s.db.Exec("UPDATE project_recovery_members SET instance_id=? WHERE operation_id=?", j.Journal.IDs.Members[1].InstanceID, j.Journal.IDs.Members[0].OperationID)
				if err != nil {
					return
				} // Unique ownership rejects even before validation.
			case "original-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", j.OriginalReadiness[0])
			case "current-mirror":
				_, err = s.db.Exec("UPDATE operations SET runtime='{}' WHERE id=?", j.CurrentReadiness[0])
			case "missing-resume":
				_, err = s.db.Exec("UPDATE operations SET body=json_set(body,'$.status','interrupted') WHERE id=?", j.Source.Resumes[0].OperationID)
			case "changed-source":
				j.Source.Current = nil
				j.CurrentReadiness = nil
				b, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_anchor_recoveries SET body=? WHERE operation_id=?", b, op.ID) // Mirror alone cannot authorize a source rewrite.
			case "rewritten-source-mirror":
				j.Source.Current = nil
				j.CurrentReadiness = nil
				b, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_anchor_recoveries SET body=? WHERE operation_id=?", b, op.ID)
				g02Check(t, err)
				_, err = s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", b, op.ID)
			case "orphan-child":
				_, err = s.db.Exec("DELETE FROM project_recovery_members WHERE operation_id=?", j.Journal.IDs.Members[0].OperationID)
			case "duplicate-parent":
				j.Journal.IDs.Members[0].OperationID = op.ID
				b, _ := json.Marshal(j)
				_, err = s.db.Exec("UPDATE project_anchor_recoveries SET body=? WHERE operation_id=?", b, op.ID)
			}
			g02Check(t, err)
			if err = validateProjectAnchorRecoveries(s.db); err == nil {
				t.Fatal("invalid recovery history accepted", fault)
			}
		})
	}
}
func TestAnchorRecoveryStoreAtomicTransitionAndStaleHistory(t *testing.T) {
	s := testStore(t)
	p, req := recoveryStoreProject(t, s)
	stale := req
	stale.HistoryDigest = fmt.Sprintf("%064d", 0)
	if _, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, stale); err == nil {
		t.Fatal("unreviewed history accepted")
	}
	op, _, err := s.acceptProjectAnchorRecovery("actor", p.ID, req)
	g02Check(t, err)
	j, err := loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	before := currentStopHistory(t, s)
	_, err = s.db.Exec(`CREATE TRIGGER fail_child_recovery BEFORE UPDATE OF runtime ON operations WHEN NEW.id='` + j.Journal.IDs.Members[1].OperationID + `' BEGIN SELECT RAISE(ABORT,'test interrupted commit'); END;`)
	g02Check(t, err)
	if _, err = s.advanceProjectAnchorRecovery(op.ID, 0, "reviewing", projectnetwork.AnchorRecoveryJournal{}); err == nil {
		t.Fatal("child mirror failure committed")
	}
	got, err := loadProjectAnchorRecovery(s.db, op.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, j) {
		t.Fatal("partial journal committed")
	}
	assertContextHistory(t, s, before)
	_, err = s.db.Exec("DROP TRIGGER fail_child_recovery")
	g02Check(t, err)
	next := advanceRecoveryStoreFixture(t, s, j, "reviewing")
	if _, err = s.advanceProjectAnchorRecovery(op.ID, 0, "reviewing", projectnetwork.AnchorRecoveryJournal{}); err == nil {
		t.Fatal("stale sequence accepted")
	}
	if _, err = s.advanceProjectAnchorRecovery(op.ID, next.Journal.Sequence, "namespace", projectnetwork.AnchorRecoveryJournal{}); err == nil {
		t.Fatal("skipped dispatch accepted")
	}
	g02Check(t, validateProjectAnchorRecoveries(s.db))
}
