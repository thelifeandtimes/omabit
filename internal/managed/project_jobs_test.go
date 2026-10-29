package managed

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func projectPreparationFixture(t *testing.T, s *store, id, op string) projectnetwork.Preparation {
	t.Helper()
	p, err := loadProject(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	return projectnetwork.Preparation{ProjectID: id, ProjectRevision: p.Revision, Generation: 1, OperationID: op, PlanDigest: digest, EngineID: "synthetic-engine", Bridge: projectnetwork.Creation{Name: id + "-bridge", ContractDigest: strings.Repeat("1", 64), Phase: "planned"}, Anchor: projectnetwork.Creation{Name: id + "-anchor", ContractDigest: strings.Repeat("2", 64), Phase: "planned"}}
}

func acceptProjectFixture(t *testing.T, s *store) (protocol.Operation, projectPrepareRequest) {
	t.Helper()
	reg, _, err := s.registerProject("actor", projectRequest(t, "register-project", "a", 28080))
	if err != nil {
		t.Fatal(err)
	}
	p, err := loadProject(s.db, reg.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	req := projectPrepareRequest{RequestID: "prepare", ExpectedRevision: 1, ReviewDigest: digest}
	op, hit, err := s.acceptProjectPreparation("actor", p.ID, req)
	if err != nil || hit {
		t.Fatal(op, hit, err)
	}
	return op, req
}

func TestProjectPreparationSurvivesEveryWriteAheadBoundary(t *testing.T) {
	for boundary := 0; boundary <= 7; boundary++ {
		t.Run(string(rune('0'+boundary)), func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			op, req := acceptProjectFixture(t, s)
			if boundary >= 1 {
				if err = s.beginProjectPreparation(op.ProjectID, op.ID, projectPreparationFixture(t, s, op.ProjectID, op.ID)); err != nil {
					t.Fatal(err)
				}
			}
			transitions := []struct{ kind, phase, id string }{{"bridge", "dispatched", ""}, {"bridge", "created", strings.Repeat("a", 64)}, {"bridge", "observed", strings.Repeat("a", 64)}, {"anchor", "dispatched", ""}, {"anchor", "created", strings.Repeat("b", 64)}, {"anchor", "observed", strings.Repeat("b", 64)}}
			for n := 0; n < boundary-1; n++ {
				v := transitions[n]
				if err = s.advanceProjectPreparation(op.ProjectID, op.ID, uint64(n), v.kind, v.phase, v.id); err != nil {
					t.Fatal(err)
				}
			}
			before, err := loadProjectJob(s.db, op.ProjectID)
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
			after, err := loadProjectJob(s.db, op.ProjectID)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("restart altered retained intent", before, after, err)
			}
			replay, hit, err := s.acceptProjectPreparation("actor", op.ProjectID, req)
			if err != nil || !hit || replay.ID != op.ID {
				t.Fatal(replay, hit, err)
			}
			want := "interrupted"
			if boundary == 7 {
				want = "succeeded"
			}
			if replay.Status != want {
				t.Fatal(replay)
			}
			req.RequestID = "different-attempt"
			_, _, err = s.acceptProjectPreparation("actor", op.ProjectID, req)
			requireFault(t, err, "project_recovery_required")
			if err = s.advanceProjectPreparation(op.ProjectID, op.ID, 0, "bridge", "dispatched", ""); err == nil {
				t.Fatal("late callback advanced interrupted or completed work")
			}
			if err = s.reconcileInterrupted(); err != nil {
				t.Fatal(err)
			}
			final, err := loadOperation(s.db, op.ID)
			if err != nil || !reflect.DeepEqual(replay, final) {
				t.Fatal("second restart changed terminal receipt", err)
			}
			r, err := projectReservations(s.db)
			if err != nil || len(r.Instances) != 2 || len(r.HTTPPorts) != 2 {
				t.Fatal(r, err)
			}
		})
	}
}

func TestProjectPreparationRejectsReorderingAndStaleCallbacks(t *testing.T) {
	s := testStore(t)
	op, _ := acceptProjectFixture(t, s)
	initial := projectPreparationFixture(t, s, op.ProjectID, op.ID)
	if err := s.beginProjectPreparation(op.ProjectID, op.ID, initial); err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		seq             uint64
		kind, phase, id string
	}{
		{0, "anchor", "dispatched", ""}, {0, "bridge", "observed", strings.Repeat("a", 64)},
		{1, "bridge", "dispatched", ""}, {0, "bridge", "dispatched", strings.Repeat("a", 64)},
		{0, "bridge", "unknown", ""},
	} {
		if err := s.advanceProjectPreparation(op.ProjectID, op.ID, v.seq, v.kind, v.phase, v.id); err == nil {
			t.Fatal("invalid transition committed", v)
		}
	}
	if err := s.advanceProjectPreparation(op.ProjectID, op.ID, 0, "bridge", "dispatched", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.advanceProjectPreparation(op.ProjectID, op.ID, 0, "bridge", "dispatched", ""); err == nil {
		t.Fatal("repeated dispatch accepted")
	}
	if err := s.advanceProjectPreparation(op.ProjectID, op.ID, 1, "bridge", "observed", ""); err == nil {
		t.Fatal("lost reply adopted resource without exact ID")
	}
	if err := s.advanceProjectPreparation(op.ProjectID, op.ID, 1, "bridge", "created", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := s.advanceProjectPreparation(op.ProjectID, op.ID, 2, "bridge", "observed", strings.Repeat("b", 64)); err == nil {
		t.Fatal("inspection changed the returned resource ID")
	}
	if err := s.beginProjectPreparation(op.ProjectID, op.ID, initial); err == nil {
		t.Fatal("second begin reset dispatch")
	}
	j, err := loadProjectJob(s.db, op.ProjectID)
	if err != nil || j.Preparation.Sequence != 2 || j.Preparation.Bridge.Phase != "created" || j.Preparation.Bridge.ID != strings.Repeat("a", 64) {
		t.Fatal(j, err)
	}
}

func TestProjectPreparationAtomicMirrorAndCorruptionRefusal(t *testing.T) {
	for _, query := range []string{
		"DELETE FROM project_jobs",
		"UPDATE operations SET runtime='{}' WHERE json_extract(body,'$.kind')='prepare-project'",
		"UPDATE project_jobs SET body='{}'",
	} {
		t.Run(query, func(t *testing.T) {
			s := testStore(t)
			op, _ := acceptProjectFixture(t, s)
			if err := s.beginProjectPreparation(op.ProjectID, op.ID, projectPreparationFixture(t, s, op.ProjectID, op.ID)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if err := validateProjectJobs(s.db); err == nil {
				t.Fatal("corrupt journal accepted")
			}
			dir := s.directory
			s.close()
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt project reopened")
			}
		})
	}
	s := testStore(t)
	op, _ := acceptProjectFixture(t, s)
	if err := s.beginProjectPreparation(op.ProjectID, op.ID, projectPreparationFixture(t, s, op.ProjectID, op.ID)); err != nil {
		t.Fatal(err)
	}
	before, err := loadProjectJob(s.db, op.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER fail_project_mirror BEFORE UPDATE OF runtime ON operations BEGIN SELECT RAISE(ABORT,'synthetic mirror failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err = s.advanceProjectPreparation(op.ProjectID, op.ID, 0, "bridge", "dispatched", ""); err == nil {
		t.Fatal("failed mirror published intent")
	}
	after, err := loadProjectJob(s.db, op.ProjectID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("partial journal committed", after, err)
	}
}

func TestProjectPreparationConcurrentWritersAndActorReplay(t *testing.T) {
	dir := privateDir(t)
	a, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()
	op, req := acceptProjectFixture(t, a)
	b, err := testOpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	if err = a.beginProjectPreparation(op.ProjectID, op.ID, projectPreparationFixture(t, a, op.ProjectID, op.ID)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, s := range []*store{a, b} {
		wg.Add(1)
		go func(s *store) {
			defer wg.Done()
			results <- s.advanceProjectPreparation(op.ProjectID, op.ID, 0, "bridge", "dispatched", "")
		}(s)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, projectnetwork.ErrRuntimeObservation) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatal("duplicate dispatch accepted", success)
	}
	req.ReplayOnly = true
	replay, hit, err := b.acceptProjectPreparation("actor", op.ProjectID, req)
	if err != nil || !hit || replay.ID != op.ID {
		t.Fatal(replay, hit, err)
	}
	_, _, err = b.acceptProjectPreparation("different-actor", op.ProjectID, req)
	requireFault(t, err, "replay_not_found")
	req.ReplayOnly = false
	req.ExpectedRevision++
	_, _, err = b.acceptProjectPreparation("actor", op.ProjectID, req)
	requireFault(t, err, "request_conflict")
	var instances int
	if err = b.db.QueryRow("SELECT count(*) FROM instances").Scan(&instances); err != nil || instances != 0 {
		t.Fatal("preparation created member instances", instances, err)
	}
}

func TestProjectPreparationValidationRejectsChangedBinding(t *testing.T) {
	for _, mutate := range []func(*projectnetwork.Preparation){
		func(p *projectnetwork.Preparation) { p.ProjectID = "another-project" }, func(p *projectnetwork.Preparation) { p.ProjectRevision++ },
		func(p *projectnetwork.Preparation) { p.Generation++ }, func(p *projectnetwork.Preparation) { p.OperationID = "op-other" },
		func(p *projectnetwork.Preparation) { p.PlanDigest = strings.Repeat("e", 64) }, func(p *projectnetwork.Preparation) { p.EngineID = "" },
		func(p *projectnetwork.Preparation) { p.Bridge.Name = p.Anchor.Name }, func(p *projectnetwork.Preparation) {
			p.Anchor.Phase = "observed"
			p.Anchor.ID = strings.Repeat("b", 64)
		},
	} {
		s := testStore(t)
		op, _ := acceptProjectFixture(t, s)
		p := projectPreparationFixture(t, s, op.ProjectID, op.ID)
		mutate(&p)
		if err := s.beginProjectPreparation(op.ProjectID, op.ID, p); err == nil {
			data, _ := json.Marshal(p)
			t.Fatal("changed binding accepted", string(data))
		}
	}
}
