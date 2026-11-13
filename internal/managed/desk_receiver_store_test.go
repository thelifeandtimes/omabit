package managed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func receiverRecordFixture(t *testing.T) deskwork.ReceiverRecord {
	t.Helper()
	i := deskIntentFixture()
	i.Action = "install-remote"
	publisher := deskwork.Publisher{Ship: "~nec", Desk: "demo", ExpectedHash: "0v123"}
	i.SourceDigest = publisher.Digest()
	review := deskwork.ReceiverReview{Version: 1, Publisher: publisher, ShipSetting: deskwork.DefaultSetting(i.Target.Ship()), DeskSetting: deskwork.DefaultSetting(i.Target), Continuity: strings.Repeat("c", 64)}
	g02Check(t, review.Seal())
	return deskwork.ReceiverRecord{Version: 1, Intent: deskwork.ReceiverIntent{Intent: i, Publisher: publisher, Continuity: review.Continuity, ReviewDigest: review.Digest}, Review: review, Actor: "fixture", CreatedAt: time.Now().UTC()}

}
func TestDeskReceiverJournalReopenAndExactReplay(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	record := receiverRecordFixture(t)
	receipt, fresh, err := s.createDeskReceiver(record)
	g02Check(t, err)
	if !fresh {
		t.Fatal("new receiver reused")
	}
	replay, fresh, err := s.createDeskReceiver(record)
	g02Check(t, err)
	if fresh || !reflect.DeepEqual(replay, receipt) {
		t.Fatal("changed replay")
	}
	changed := record
	changed.Actor = "another"
	if _, _, err = s.createDeskReceiver(changed); err == nil {
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
		t.Fatal("unfinished receiver not retained")
	}
	stored, err := loadDeskReceiver(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(stored, record) {
		t.Fatal("receiver review changed")
	}
	before := current
	g02Check(t, s.reconcileDeskOperations())
	current, err = loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(current, before) {
		t.Fatal("recovery repeated work")
	}
	after := deskwork.Observation{Exists: true, Revision: 1, ClayHash: record.Intent.Publisher.ExpectedHash, Publication: "private"}
	app := deskwork.AppState{Zest: "live", SourceRevision: 1, SourceShip: record.Intent.Publisher.Ship, SourceDesk: record.Intent.Publisher.Desk, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, FrontendMode: "none", FrontendState: "unavailable"}
	current, err = s.appendDeskStep(current, deskwork.Step{Kind: record.Intent.Intent.Action, State: "observed", At: time.Now().UTC(), Observation: &after, Application: &app})
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, err := loadDeskOperation(s.db, current.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, current) {
		t.Fatal("observed receiver lost")
	}
}
func TestDeskReceiverRecordAndJournalAreAtomic(t *testing.T) {
	s := testStore(t)
	_, err := s.db.Exec("CREATE TRIGGER fail_receiver BEFORE INSERT ON desk_receivers BEGIN SELECT RAISE(ABORT,'fixture'); END")
	g02Check(t, err)
	if _, _, err = s.createDeskReceiver(receiverRecordFixture(t)); err == nil {
		t.Fatal("failed write accepted")
	}
	ids, err := deskOperationIDs(s.db)
	g02Check(t, err)
	if len(ids) != 0 {
		t.Fatal("orphaned intent persisted")
	}
}
func TestDeskReceiverCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"missing-record", "different-review", "missing-journal", "unexpected-field"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := receiverRecordFixture(t)
			_, _, err = s.createDeskReceiver(record)
			g02Check(t, err)
			switch kind {
			case "missing-record":
				_, err = s.db.Exec("DELETE FROM desk_receivers")
			case "missing-journal":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operation_history")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operations")
			case "different-review":
				record.Review.Observation.ClayHash = "0v999"
				body, _ := json.Marshal(record)
				_, err = s.db.Exec("UPDATE desk_receivers SET body=?", body)
			case "unexpected-field":
				_, err = s.db.Exec("UPDATE desk_receivers SET body=json_set(body,'$.unknown',true)")
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt receiver history accepted")
			}
		})
	}
}
