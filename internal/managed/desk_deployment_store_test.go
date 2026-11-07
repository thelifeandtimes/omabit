package managed

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"reflect"
	"strings"
	"testing"
	"time"
)

func deploymentRecordFixture(t *testing.T) deskwork.DeploymentRecord {
	t.Helper()
	raw, err := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat("a", 64), Bytes: 6}})
	g02Check(t, err)
	empty := deskwork.EmptyOwnership()
	plan, err := deskwork.PlanDeployment(empty, empty.Clay, raw, empty.Clay)
	g02Check(t, err)
	intent := deskIntentFixture()
	intent.Action = "commit"
	intent.SourceDigest = raw.Digest
	intent.Baseline = deskwork.Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
	return deskwork.DeploymentRecord{Version: 1, Intent: deskwork.CommitIntent{Intent: intent, StageID: strings.Repeat("c", 64), SourceName: "example", Continuity: strings.Repeat("d", 64), PlanDigest: plan.Digest}, Actor: "fixture", Plan: plan, CreatedAt: time.Now().UTC()}
}
func completeDeploymentFixture(t *testing.T, s *store, record deskwork.DeploymentRecord, receipt deskwork.Receipt) deskwork.Receipt {
	t.Helper()
	var err error
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "dispatched"))
	g02Check(t, err)
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "acknowledged"))
	g02Check(t, err)
	clay, err := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat("e", 64)}})
	g02Check(t, err)
	owned, err := deskwork.VerifyDeployment(record.Plan, clay, clay)
	g02Check(t, err)
	receipt, err = s.completeDeskDeployment(receipt, deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v456", Publication: "private"}, owned, "")
	g02Check(t, err)
	return receipt
}
func TestDeskDeploymentJournalAndOwnershipSurviveReopen(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	_, _, err = s.register("fixture", registration("alpha", "register", 18080))
	g02Check(t, err)
	record := deploymentRecordFixture(t)
	receipt, fresh, err := s.createDeskDeployment(record)
	g02Check(t, err)
	if !fresh {
		t.Fatal("new deployment not fresh")
	}
	replay, fresh, err := s.createDeskDeployment(record)
	g02Check(t, err)
	if fresh || !reflect.DeepEqual(replay, receipt) {
		t.Fatal("duplicate changed or redispatched")
	}
	altered := record
	altered.Intent.StageID = strings.Repeat("f", 64)
	if _, _, err = s.createDeskDeployment(altered); err == nil {
		t.Fatal("stage rebind accepted")
	}
	receipt = completeDeploymentFixture(t, s, record, receipt)
	owned, err := s.sourceOwnership(record.Intent.Intent.Target, "example", record.Intent.Continuity)
	g02Check(t, err)
	if len(owned.Raw.Files) != 1 {
		t.Fatal("verified ownership missing")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	saved, err := loadDeskDeployment(s.db, record.Intent.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(saved, record) {
		t.Fatal("deployment lost")
	}
	got, err := loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, receipt) {
		t.Fatal("receipt lost")
	}
	after, err := s.sourceOwnership(record.Intent.Intent.Target, "example", record.Intent.Continuity)
	g02Check(t, err)
	if !reflect.DeepEqual(after, owned) {
		t.Fatal("ownership lost")
	}
	for _, selection := range [][2]string{{"other", record.Intent.Continuity}, {"example", strings.Repeat("f", 64)}} {
		empty, err := s.sourceOwnership(record.Intent.Intent.Target, selection[0], selection[1])
		g02Check(t, err)
		if len(empty.Raw.Files) != 0 {
			t.Fatal("ownership crossed source or pier")
		}
	}
}
func TestDeskDeploymentCompletionIsAtomic(t *testing.T) {
	s := testStore(t)
	record := deploymentRecordFixture(t)
	receipt, _, err := s.createDeskDeployment(record)
	g02Check(t, err)
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "dispatched"))
	g02Check(t, err)
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "acknowledged"))
	g02Check(t, err)
	_, err = s.db.Exec("CREATE TRIGGER fail_ownership BEFORE INSERT ON desk_source_ownership BEGIN SELECT RAISE(ABORT,'fixture'); END")
	g02Check(t, err)
	clay, err := deskwork.NewManifest([]deskwork.File{{Path: "hello.txt", SHA256: strings.Repeat("e", 64)}})
	g02Check(t, err)
	ownership, err := deskwork.VerifyDeployment(record.Plan, clay, clay)
	g02Check(t, err)
	_, err = s.completeDeskDeployment(receipt, deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v456", Publication: "private"}, ownership, "")
	if err == nil {
		t.Fatal("failed ownership write accepted")
	}
	after, err := loadDeskOperation(s.db, receipt.Intent.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(after, receipt) {
		t.Fatal("failed completion changed journal")
	}
}
func TestDeskDeploymentCorruptionRefusesReopen(t *testing.T) {
	for _, kind := range []string{"missing-review", "bad-review", "rebound-stage", "bad-owner", "unobserved-owner", "missing-table"} {
		t.Run(kind, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			record := deploymentRecordFixture(t)
			receipt, _, err := s.createDeskDeployment(record)
			g02Check(t, err)
			completeDeploymentFixture(t, s, record, receipt)
			switch kind {
			case "missing-review":
				_, err = s.db.Exec("DELETE FROM desk_deployments")
			case "bad-review":
				_, err = s.db.Exec("UPDATE desk_deployments SET body='{}'")
			case "rebound-stage":
				record.Intent.Intent.SourceDigest = strings.Repeat("f", 64)
				body, _ := json.Marshal(record)
				_, err = s.db.Exec("UPDATE desk_deployments SET body=?", body)
			case "bad-owner":
				_, err = s.db.Exec("UPDATE desk_source_ownership SET body='{}'")
			case "unobserved-owner":
				_, err = s.db.Exec("UPDATE desk_source_ownership SET body=json_set(body,'$.operation_id',?)", strings.Repeat("f", 64))
			case "missing-table":
				_, err = s.db.Exec("DROP TABLE desk_source_ownership")
			}
			g02Check(t, err)
			g02Check(t, s.close())
			if reopened, err := testOpenStore(dir); err == nil {
				reopened.close()
				t.Fatal("corrupt deployment accepted")
			}
		})
	}
}
