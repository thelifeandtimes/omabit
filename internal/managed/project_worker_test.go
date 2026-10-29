package managed

import (
	"context"
	"errors"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
)

type storedPreparationEngine struct {
	t              *testing.T
	s              *store
	id             string
	calls          []string
	failInspection bool
}

func (e *storedPreparationEngine) VerifyEngine(context.Context, string) error { return nil }
func (e *storedPreparationEngine) Create(_ context.Context, kind string, c projectnetwork.Creation, bridge string) (string, error) {
	j, err := loadProjectJob(e.s.db, e.id)
	if err != nil {
		e.t.Fatal(err)
	}
	record := j.Preparation.Bridge
	if kind == "anchor" {
		record = j.Preparation.Anchor
		if bridge != j.Preparation.Bridge.ID {
			e.t.Fatal("anchor lost exact bridge binding")
		}
	}
	if record != c || c.Phase != "dispatched" {
		e.t.Fatal("create preceded SQLite dispatch commit")
	}
	e.calls = append(e.calls, "create-"+kind)
	id := strings.Repeat("a", 64)
	if kind == "anchor" {
		id = strings.Repeat("b", 64)
	}
	return id, nil
}
func (e *storedPreparationEngine) Inspect(_ context.Context, kind string, c projectnetwork.Creation, _ string) (projectnetwork.ResourceIdentity, error) {
	j, err := loadProjectJob(e.s.db, e.id)
	if err != nil {
		e.t.Fatal(err)
	}
	record := j.Preparation.Bridge
	if kind == "anchor" {
		record = j.Preparation.Anchor
	}
	if record != c || c.Phase != "created" {
		e.t.Fatal("inspection preceded SQLite returned-ID commit")
	}
	e.calls = append(e.calls, "inspect-"+kind)
	if e.failInspection {
		return projectnetwork.ResourceIdentity{}, errors.New("synthetic inspection failure")
	}
	return projectnetwork.ResourceIdentity{ID: c.ID, ContractDigest: c.ContractDigest}, nil
}
func TestProjectPreparationWorkerUsesDurableJournalAndNeverReplays(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "inspection-failed"}[failed], func(t *testing.T) {
			s := testStore(t)
			op, req := acceptProjectFixture(t, s)
			prepared := projectPreparationFixture(t, s, op.ProjectID, op.ID)
			engine := &storedPreparationEngine{t: t, s: s, id: op.ProjectID, failInspection: failed}
			err := s.runProjectPreparation(context.Background(), op.ProjectID, op.ID, prepared, engine)
			if (err != nil) != failed {
				t.Fatal(err)
			}
			replay, hit, err := s.acceptProjectPreparation("actor", op.ProjectID, req)
			if err != nil || !hit {
				t.Fatal(replay, hit, err)
			}
			want := "succeeded"
			calls := 4
			if failed {
				want = "interrupted"
				calls = 2
			}
			if replay.Status != want || len(engine.calls) != calls {
				t.Fatal(replay, engine.calls)
			}
			if err = s.runProjectPreparation(context.Background(), op.ProjectID, op.ID, prepared, engine); err == nil || len(engine.calls) != calls {
				t.Fatal("same worker replayed engine mutation", err, engine.calls)
			}
			j, err := loadProjectJob(s.db, op.ProjectID)
			if err != nil {
				t.Fatal(err)
			}
			if failed && (j.Preparation.Bridge.Phase != "created" || j.Preparation.Bridge.ID != strings.Repeat("a", 64) || j.Preparation.Anchor.Phase != "planned") {
				t.Fatal("failure lost exact returned ID", j)
			}
			if err = validateProjectJobs(s.db); err != nil {
				t.Fatal(err)
			}
		})
	}
}
