package managed

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func frontendRecordFixture(t *testing.T) deskwork.FrontendRecord {
	t.Helper()
	i := deskIntentFixture()
	i.Action = "upload-frontend"
	i.Baseline = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
	i.SourceDigest = strings.Repeat("f", 64)
	ref := deskwork.FrontendReference{Mode: "glob-ames", Base: "demo", Ship: "~zod", GlobHash: "0v1", DocketHash: "0v2"}
	app := deskwork.AppState{Zest: "live", SourceRevision: 1, ExpectedAgents: []string{}, Agents: []deskwork.AppAgent{}, DocketAvailable: true, DocketHash: ref.DocketHash, FrontendMode: ref.Mode, FrontendState: "glob", FrontendDocketHash: ref.DocketHash}
	review := deskwork.FrontendReview{Version: 1, Ship: "~zod", Observation: i.Baseline, Application: app, Frontend: deskwork.FrontendState{SourceRevision: 1, DocketAvailable: true, Stage: "glob", Declared: ref, Current: ref}, ShipSetting: deskwork.DefaultSetting(i.Target.Ship()), DeskSetting: deskwork.DefaultSetting(i.Target), Continuity: strings.Repeat("c", 64), Stage: deskwork.StageIntent{Version: 1, ID: strings.Repeat("d", 64), Target: i.Target, SourceDigest: strings.Repeat("e", 64), Files: 1, Bytes: 80}, ArtifactDigest: i.SourceDigest, ExpectedGlobHash: "0v3", UnchangedSourceDigest: strings.Repeat("a", 64)}
	g02Check(t, review.Seal())
	checks := []deskwork.FrontendCheck{}
	return deskwork.FrontendRecord{Version: 1, Intent: deskwork.FrontendIntent{Intent: i, StageID: review.Stage.ID, Continuity: review.Continuity, ReviewDigest: review.Digest, ChecksDigest: deskwork.FrontendChecksDigest(checks)}, Review: review, Checks: checks, Actor: "fixture", CreatedAt: time.Now().UTC()}
}
func TestDeskFrontendJournalReopenAndExactReplay(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	record := frontendRecordFixture(t)
	receipt, fresh, err := s.createDeskFrontend(record)
	g02Check(t, err)
	if !fresh {
		t.Fatal("new frontend reused")
	}
	replay, fresh, err := s.createDeskFrontend(record)
	g02Check(t, err)
	if fresh || !reflect.DeepEqual(replay, receipt) {
		t.Fatal("changed replay")
	}
	changed := record
	changed.Actor = "another"
	if _, _, err = s.createDeskFrontend(changed); err == nil {
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
		t.Fatal("unfinished frontend not retained")
	}
	stored, err := loadDeskFrontend(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(stored, record) {
		t.Fatal("frontend review changed")
	}
	before := current
	g02Check(t, s.reconcileDeskOperations())
	current, err = loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(current, before) {
		t.Fatal("recovery repeated work")
	}

	after := record.Review.Observation
	after.Revision++
	after.ClayHash = "0v4"
	app := record.Review.Application
	app.SourceRevision = after.Revision
	app.DocketHash = "0v5"
	app.FrontendDocketHash = "0v5"
	f := record.Review.Frontend
	f.SourceRevision = after.Revision
	f.Declared.GlobHash = record.Review.ExpectedGlobHash
	f.Declared.DocketHash = "0v5"
	f.Current = f.Declared
	current, err = s.appendDeskStep(current, deskwork.Step{Kind: record.Intent.Intent.Action, State: "observed", At: time.Now().UTC(), Observation: &after, Application: &app, Frontend: &f, FrontendSourceDigest: record.Review.UnchangedSourceDigest, VerifiedSourceDigest: record.Review.ArtifactDigest})
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, err := loadDeskOperation(s.db, current.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, current) {
		t.Fatal("observed frontend lost")
	}
}
func TestDeskFrontendRecordAndJournalAreAtomic(t *testing.T) {
	s := testStore(t)
	_, err := s.db.Exec("CREATE TRIGGER fail_frontend BEFORE INSERT ON desk_frontends BEGIN SELECT RAISE(ABORT,'fixture'); END")
	g02Check(t, err)
	if _, _, err = s.createDeskFrontend(frontendRecordFixture(t)); err == nil {
		t.Fatal("failed write accepted")
	}
	ids, err := deskOperationIDs(s.db)
	g02Check(t, err)
	if len(ids) != 0 {
		t.Fatal("orphaned intent persisted")
	}
}
func TestDeskFrontendCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"missing-record", "different-review", "missing-journal", "unexpected-field"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := frontendRecordFixture(t)
			_, _, err = s.createDeskFrontend(record)
			g02Check(t, err)
			switch kind {
			case "missing-record":
				_, err = s.db.Exec("DELETE FROM desk_frontends")
			case "missing-journal":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operation_history")
				g02Check(t, err)
				_, err = s.db.Exec("DELETE FROM desk_operations")
			case "different-review":
				record.Review.Observation.ClayHash = "0v999"
				body, _ := json.Marshal(record)
				_, err = s.db.Exec("UPDATE desk_frontends SET body=?", body)
			case "unexpected-field":
				_, err = s.db.Exec("UPDATE desk_frontends SET body=json_set(body,'$.unknown',true)")
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt frontend history accepted")
			}
		})
	}
}
