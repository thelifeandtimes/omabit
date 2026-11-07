package managed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func releaseRecordFixture(t *testing.T) deskwork.ReleaseRecord {
	t.Helper()
	i := deskIntentFixture()
	i.Action = "publish-source"
	i.Baseline = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
	desk := deskwork.DefaultSetting(i.Target)
	desk.Desk.Publication = "source"
	review := deskwork.ReleaseReview{Version: 1, Action: i.Action, Observation: i.Baseline, Application: deskwork.AppState{Zest: "dead", SourceRevision: 1, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}, ShipSetting: deskwork.DefaultSetting(i.Target.Ship()), DeskSetting: desk, Continuity: strings.Repeat("c", 64)}
	g02Check(t, review.Seal())
	checks := []deskwork.ReleaseCheck{}
	return deskwork.ReleaseRecord{Version: 1, Intent: deskwork.ReleaseIntent{Intent: i, Continuity: review.Continuity, ReviewDigest: review.Digest, ChecksDigest: deskwork.ReleaseChecksDigest(checks)}, Review: review, Checks: checks, Actor: "fixture", CreatedAt: time.Now().UTC()}
}
func TestDeskReleaseJournalReopenAndExactReplay(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	record := releaseRecordFixture(t)
	receipt, fresh, err := s.createDeskRelease(record)
	g02Check(t, err)
	if !fresh {
		t.Fatal("new release reused")
	}
	replay, fresh, err := s.createDeskRelease(record)
	g02Check(t, err)
	if fresh || !reflect.DeepEqual(replay, receipt) {
		t.Fatal("changed replay")
	}
	changed := record
	changed.Actor = "another"
	if _, _, err = s.createDeskRelease(changed); err == nil {
		t.Fatal("actor rebound")
	}
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "dispatched"))
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	g02Check(t, s.reconcileDeskOperations())
	current, err := loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if current.Steps[len(current.Steps)-1].State != "uncertain" {
		t.Fatal("unfinished release not retained")
	}
	stored, err := loadDeskRelease(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(stored, record) {
		t.Fatal("release review changed")
	}
	before := current
	g02Check(t, s.reconcileDeskOperations())
	current, err = loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(current, before) {
		t.Fatal("recovery repeated work")
	}
	after := record.Review.Observation
	after.Publication = "published"
	app := record.Review.Application
	current, err = s.appendDeskStep(current, deskwork.Step{Kind: record.Intent.Intent.Action, State: "observed", At: time.Now().UTC(), Observation: &after, Application: &app})
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, err := loadDeskOperation(s.db, current.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, current) {
		t.Fatal("observed release lost")
	}
}
func TestDeskReleaseRecordAndJournalAreAtomic(t *testing.T) {
	s := testStore(t)
	_, err := s.db.Exec("CREATE TRIGGER fail_release BEFORE INSERT ON desk_releases BEGIN SELECT RAISE(ABORT,'fixture'); END")
	g02Check(t, err)
	if _, _, err = s.createDeskRelease(releaseRecordFixture(t)); err == nil {
		t.Fatal("failed write accepted")
	}
	ids, err := deskOperationIDs(s.db)
	g02Check(t, err)
	if len(ids) != 0 {
		t.Fatal("orphaned intent persisted")
	}
}
func TestDeskReleaseCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"missing-record", "different-review", "missing-journal", "unexpected-field"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := releaseRecordFixture(t)
			_, _, err = s.createDeskRelease(record)
			g02Check(t, err)
			switch kind {
			case "missing-record":
				_, err = s.db.Exec("DELETE FROM desk_releases")
			case "missing-journal":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operation_history")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operations")
			case "different-review":
				record.Review.Observation.ClayHash = "0v999"
				body, _ := json.Marshal(record)
				_, err = s.db.Exec("UPDATE desk_releases SET body=?", body)
			case "unexpected-field":
				_, err = s.db.Exec("UPDATE desk_releases SET body=json_set(body,'$.unknown',true)")
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt release history accepted")
			}
		})
	}
}
