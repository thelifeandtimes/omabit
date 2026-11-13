package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"reflect"
	"strings"
	"testing"
)

func adoptionRecordFixture(t *testing.T) deskwork.AdoptionRecord {
	t.Helper()
	d := deploymentRecordFixture(t)
	canonical, err := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat("e", 64)}})
	g02Check(t, err)
	plan, err := deskwork.PlanAdoption(deskwork.EmptyOwnership(), canonical, d.Plan.Source, canonical)
	g02Check(t, err)
	i := deskwork.AdoptionIntent(d.Intent)
	i.Intent.Action = "adopt-source"
	i.PlanDigest = plan.Digest
	return deskwork.AdoptionRecord{Version: 1, Intent: i, Actor: d.Actor, Plan: plan, CreatedAt: d.CreatedAt}
}
func allowAdoptionFixture(t *testing.T, s *store, record *deskwork.AdoptionRecord) {
	t.Helper()
	_, _, err := s.register("fixture", registration("alpha", "adoption-register", 18080))
	g02Check(t, err)
	p, err := s.putAppPolicy("setup", apppolicy.Update{Scope: "desk", Target: record.Intent.Intent.Target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Allow}})
	g02Check(t, err)
	record.Intent.Intent.PolicyRevision = p.Revision
}
func TestDeskAdoptionAtomicJournalOwnershipReplayAndReopen(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "allowed"}[allowed], func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := adoptionRecordFixture(t)
			if allowed {
				allowAdoptionFixture(t, s, &record)
			}
			receipt, err := s.adoptDeskSource(record)
			g02Check(t, err)
			want := "blocked"
			if allowed {
				want = "observed"
			}
			if len(receipt.Steps) != 2 || receipt.Steps[1].State != want {
				t.Fatal(receipt)
			}
			before, err := s.sourceOwnership(record.Intent.Intent.Target, "example", record.Intent.Continuity)
			g02Check(t, err)
			if (len(before.Raw.Files) == 1) != allowed {
				t.Fatal("ownership escaped gate", before)
			}
			g02Check(t, s.close())
			s, err = testOpenStore(dir)
			g02Check(t, err)
			defer s.close()
			replay, err := s.adoptDeskSource(record)
			g02Check(t, err)
			if !reflect.DeepEqual(replay, receipt) {
				t.Fatal("adoption replay changed receipt")
			}
			after, err := s.sourceOwnership(record.Intent.Intent.Target, "example", record.Intent.Continuity)
			g02Check(t, err)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("ownership changed on reopen")
			}
			rebound := record
			rebound.Intent.StageID = strings.Repeat("f", 64)
			if _, err = s.adoptDeskSource(rebound); err == nil {
				t.Fatal("rebound adoption accepted")
			}
		})
	}
}
func TestDeskAdoptionFailureRollsBackEverything(t *testing.T) {
	s := testStore(t)
	record := adoptionRecordFixture(t)
	allowAdoptionFixture(t, s, &record)
	_, err := s.db.Exec("CREATE TRIGGER fail_adoption BEFORE INSERT ON desk_source_ownership BEGIN SELECT RAISE(ABORT,'fixture'); END")
	g02Check(t, err)
	if _, err = s.adoptDeskSource(record); err == nil {
		t.Fatal("failed transaction accepted")
	}
	for _, table := range []string{"desk_operations", "desk_operation_history", "desk_adoptions", "desk_source_ownership"} {
		var n int
		g02Check(t, s.db.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
		if n != 0 {
			t.Fatal("partial adoption", table, n)
		}
	}
}
func TestDeskAdoptionCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"missing-record", "missing-operation", "bad-record", "changed-plan", "changed-owner", "orphan", "missing-table"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := adoptionRecordFixture(t)
			allowAdoptionFixture(t, s, &record)
			_, err = s.adoptDeskSource(record)
			g02Check(t, err)
			switch kind {
			case "missing-record":
				_, err = s.db.Exec("DELETE FROM desk_adoptions")
			case "missing-operation":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF; DELETE FROM desk_operation_history; DELETE FROM desk_operations")
			case "bad-record":
				_, err = s.db.Exec("UPDATE desk_adoptions SET body='{}'")
			case "changed-plan":
				record.Plan.Next = deskwork.EmptyOwnership()
				b, _ := json.Marshal(record)
				_, err = s.db.Exec("UPDATE desk_adoptions SET body=?", b)
			case "changed-owner":
				_, err = s.db.Exec("UPDATE desk_source_ownership SET body=json_set(body,'$.source_name','other')")
			case "orphan":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("INSERT INTO desk_adoptions(id,body) VALUES('orphan','{}')")
			case "missing-table":
				_, err = s.db.Exec("DROP TABLE desk_adoptions")
			}
			g02Check(t, err)
			if kind == "missing-record" || kind == "missing-operation" {
				if _, err = s.sourceOwnership(record.Intent.Intent.Target, "example", record.Intent.Continuity); err == nil {
					t.Fatal("missing evidence treated as unowned source")
				}
			}
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt adoption accepted")
			}
		})
	}
}
