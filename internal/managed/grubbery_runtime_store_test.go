package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func runtimeReviewFixture(t *testing.T, record grubbery.Record, previous *grubbery.RuntimeRecoveryReceipt) grubbery.RuntimeRecoveryReview {
	t.Helper()
	step, done, err := record.Current()
	g02Check(t, err)
	if done {
		t.Fatal("unfinished record required")
	}
	state, err := record.State(step.ID)
	g02Check(t, err)
	at := time.Now().UTC()
	last := record.HeaderDigest
	if len(record.Events) != 0 {
		last = record.Events[len(record.Events)-1].Digest
	}
	review := grubbery.RuntimeRecoveryReview{Version: 1, OriginalRequestID: record.Plan.RequestID, PlanDigest: record.Plan.Digest, HeaderDigest: record.HeaderDigest, OriginalRevision: record.Revision, OriginalEventDigest: last, StepID: step.ID, State: state, OriginalBinding: record.Plan.Binding, PreviousContainerID: record.Plan.Binding.RuntimeID, Process: grubbery.RuntimeProcess{ContainerID: strings.Repeat("9", 64), PID: 42, StartedAt: at.Add(-time.Minute), HostBootID: "11111111-2222-3333-4444-555555555555", EngineID: "fixture-daemon", ImageManifestDigest: "sha256:" + strings.Repeat("8", 64), RuntimeVersion: "4.6-5a7614f", ContextSHA256: strings.Repeat("7", 64)}, EvidenceBasis: "verified-pending-prefix", NativeEvidenceDigest: strings.Repeat("6", 64), ObservedAt: at}
	if state != "pending" {
		binding, _ := json.Marshal(record.Plan.Binding)
		review.EvidenceBasis = "native-phase-observed"
		review.AppliedPhaseProof = &grubbery.Proof{Version: 1, PlanDigest: record.Plan.Digest, StepID: step.ID, BindingDigest: stopEvidenceDigest(binding), NativeEvidenceDigest: review.NativeEvidenceDigest}
	}
	if previous != nil {
		review.PreviousRecoveryDigest = previous.Digest
		review.PreviousContainerID = previous.Review.Process.ContainerID
		review.Process.ContainerID = stopEvidenceDigest([]byte(previous.Digest + ":replacement"))
		review.Process.PID = previous.Review.Process.PID + 1
	}
	review.Digest = review.ContentDigest()
	g02Check(t, review.ValidatePredecessor(record, previous))
	return review
}
func runtimeStoreFixture(t *testing.T) (*store, grubbery.Record, string) {
	t.Helper()
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	record, _ := grubberyFixture(t, s)
	j := grubberyJournal{s}
	record, err = j.Create(t.Context(), record)
	g02Check(t, err)
	record = appAppendFixture(t, j, record, record.Plan.Steps[0], "dispatched")
	record = appAppendFixture(t, j, record, record.Plan.Steps[0], "acknowledged")
	t.Cleanup(func() { s.close() })
	return s, record, dir
}
func rowsWithoutRuntimeRecovery(all map[string][]string) map[string][]string {
	delete(all, "grubbery_app_runtime_heads")
	delete(all, "grubbery_app_runtime_recoveries")
	return all
}

func TestAppRuntimeStorePreservesOriginalRowsLeaseAndExpiredReplayAcrossReopen(t *testing.T) {
	s, original, dir := runtimeStoreFixture(t)
	before := rowsWithoutRuntimeRecovery(appAllRows(t, s.db))
	review := runtimeReviewFixture(t, original, nil)
	id := strings.Repeat("5", 64)
	j := grubberyRuntimeJournal{s}
	receipt, err := j.Apply(t.Context(), original.Plan.RequestID, id, review, review.ObservedAt.Add(time.Second))
	g02Check(t, err)
	if err = receipt.ValidateAncestry(original); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, rowsWithoutRuntimeRecovery(appAllRows(t, s.db))) {
		t.Fatal("metadata binding changed original row, event, history or lease")
	}
	for i := 0; i < 3; i++ {
		got, err := j.Apply(t.Context(), original.Plan.RequestID, id, review, review.ObservedAt.Add(24*time.Hour))
		g02Check(t, err)
		if !reflect.DeepEqual(got, receipt) {
			t.Fatal("expired exact replay changed saved audit")
		}
		latest, err := j.Latest(t.Context(), original.Plan.RequestID)
		g02Check(t, err)
		if latest == nil || !reflect.DeepEqual(*latest, receipt) {
			t.Fatal("wrong effective audit head")
		}
		g02Check(t, s.close())
		s, err = testOpenStore(dir)
		g02Check(t, err)
		j = grubberyRuntimeJournal{s}
		lookup, err := j.Load(t.Context(), original.Plan.RequestID, id)
		g02Check(t, err)
		if !reflect.DeepEqual(lookup, receipt) {
			t.Fatal("saved lookup changed across restart")
		}
		gotOriginal, err := (grubberyJournal{s}).Load(t.Context(), original.Plan.RequestID)
		g02Check(t, err)
		if !reflect.DeepEqual(gotOriginal, original) {
			t.Fatal("runtime binding resolved or rewrote original uncertainty")
		}
	}
	defer s.close()
	altered := review
	altered.Process.PID++ // Same old digest cannot hide a changed request.
	if _, err = j.Apply(t.Context(), original.Plan.RequestID, id, altered, time.Now().UTC()); !errors.Is(err, grubbery.ErrConflict) {
		t.Fatal("changed replay body accepted", err)
	}
	if _, err = j.Apply(t.Context(), original.Plan.RequestID, strings.Repeat("4", 64), review, review.ObservedAt.Add(time.Second)); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("same review appended over a changed recovery head", err)
	}
}
func TestAppRuntimeStoreRefusesStalePrefixAndChainsOnlyExactNewHead(t *testing.T) {
	s, record, _ := runtimeStoreFixture(t)
	j := grubberyRuntimeJournal{s}
	first := runtimeReviewFixture(t, record, nil)
	saved, err := j.Apply(t.Context(), record.Plan.RequestID, strings.Repeat("5", 64), first, first.ObservedAt.Add(time.Second))
	g02Check(t, err)
	next := runtimeReviewFixture(t, record, &saved)
	record = appAppendFixture(t, grubberyJournal{s}, record, record.Plan.Steps[0], "observed")
	before := appAllRows(t, s.db)
	if _, err = j.Apply(t.Context(), record.Plan.RequestID, strings.Repeat("4", 64), next, next.ObservedAt.Add(time.Second)); !errors.Is(err, grubbery.ErrCAS) {
		t.Fatal("later app progress refreshed an old review", err)
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("stale apply changed state")
	}
	fresh := runtimeReviewFixture(t, record, &saved)
	successor, err := j.Apply(t.Context(), record.Plan.RequestID, strings.Repeat("3", 64), fresh, fresh.ObservedAt.Add(time.Second))
	g02Check(t, err)
	if successor.Review.PreviousRecoveryDigest != saved.Digest || successor.Review.PreviousContainerID != saved.Review.Process.ContainerID {
		t.Fatal("new receipt lost exact predecessor")
	}
	latest, err := j.Latest(t.Context(), record.Plan.RequestID)
	g02Check(t, err)
	if latest == nil || latest.Digest != successor.Digest {
		t.Fatal("new head not visible")
	}
	if err = validateGrubberyJournal(s.db); err != nil {
		t.Fatal("historical receipt invalidated after legitimate progress", err)
	}
}
func TestAppRuntimeStoreConcurrentApplyAdmitsOneAuditAndNoOriginalEvent(t *testing.T) {
	s, record, _ := runtimeStoreFixture(t)
	review := runtimeReviewFixture(t, record, nil)
	j := grubberyRuntimeJournal{s}
	before := record.Revision
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := j.Apply(context.Background(), record.Plan.RequestID, stopEvidenceDigest([]byte{byte(i)}), review, review.ObservedAt.Add(time.Second))
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if success != 1 {
		t.Fatal("concurrent old-head reviews both admitted", success)
	}
	var count int
	g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_runtime_recoveries").Scan(&count))
	if count != 1 {
		t.Fatal("duplicate metadata binding")
	}
	got, err := (grubberyJournal{s}).Load(t.Context(), record.Plan.RequestID)
	g02Check(t, err)
	if got.Revision != before || !reflect.DeepEqual(got, record) {
		t.Fatal("binding dispatched or observed a native phase")
	}
}
func TestAppRuntimeStoreCorruptionRefusesWithoutRepair(t *testing.T) {
	for _, scenario := range []string{"missing-tail", "missing-head", "body", "digest", "head", "trigger"} {
		t.Run(scenario, func(t *testing.T) {
			s, record, dir := runtimeStoreFixture(t)
			review := runtimeReviewFixture(t, record, nil)
			_, err := (grubberyRuntimeJournal{s}).Apply(t.Context(), record.Plan.RequestID, strings.Repeat("5", 64), review, review.ObservedAt.Add(time.Second))
			g02Check(t, err)
			statements := map[string]string{"missing-tail": "DELETE FROM grubbery_app_runtime_recoveries", "missing-head": "DELETE FROM grubbery_app_runtime_heads", "body": "UPDATE grubbery_app_runtime_recoveries SET body='{}'", "digest": "UPDATE grubbery_app_runtime_recoveries SET digest='invalid'", "head": "UPDATE grubbery_app_runtime_heads SET digest='invalid'", "trigger": "CREATE TRIGGER injected_runtime AFTER INSERT ON grubbery_app_runtime_recoveries BEGIN SELECT 1; END"}
			_, err = s.db.Exec(statements[scenario])
			g02Check(t, err)
			before := appAllRows(t, s.db)
			g02Check(t, s.close())
			if opened, err := testOpenStore(dir); err == nil {
				opened.close()
				t.Fatal("corrupt recovery graph opened")
			}
			db := g02RawDB(t, dir)
			defer db.Close()
			if !reflect.DeepEqual(before, appAllRows(t, db)) {
				t.Fatal("open repaired changed audit ancestry")
			}
		})
	}
}
func TestAppRuntimeMigrationRefusesUnownedSuccessorTablesAndRollsBack(t *testing.T) {
	for _, table := range []string{"grubbery_app_runtime_recoveries", "grubbery_app_runtime_heads"} {
		t.Run(table, func(t *testing.T) {
			dir := privateDir(t)
			loadOriginal61AppFixture(t, dir)
			db := g02RawDB(t, dir)
			_, err := db.Exec("CREATE TABLE " + table + "(unowned TEXT)")
			g02Check(t, err)
			before := appAllRows(t, db)
			g02Check(t, db.Close())
			if s, err := testOpenStore(dir); err == nil {
				s.close()
				t.Fatal("unowned successor table accepted")
			}
			db = g02RawDB(t, dir)
			defer db.Close()
			if !reflect.DeepEqual(before, appAllRows(t, db)) {
				t.Fatal("failed migration changed old rows/schema")
			}
		})
	}
}

// The populated genuine62 writer fixture is produced externally, never invented
// by relabeling a current database. This hook is invoked only in source rehearsal.
func TestAppRuntimeOriginal62Qualification(t *testing.T) {
	root := os.Getenv("OMABIT_APP_RUNTIME_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit original-writer rehearsal only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-app-runtime-upgrade-") {
		t.Fatal("private disposable fixture required")
	}
	raw, err := os.ReadFile(filepath.Join(root, "original-report.json"))
	g02Check(t, err)
	var report struct {
		Schema int                 `json:"schema"`
		Record grubbery.Record     `json:"original_record"`
		Rows   map[string][]string `json:"all_original_rows"`
	}
	g02Check(t, json.Unmarshal(raw, &report))
	if report.Schema != 62 || report.Record.Validate() != nil {
		t.Fatal("wrong actual original writer report")
	}
	dir := filepath.Join(root, "successor")
	if _, err = os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatal("fresh successor required")
	}
	g02Check(t, os.Mkdir(dir, 0700))
	source, err := os.ReadFile(filepath.Join(root, "original/runner.sqlite"))
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), source, 0600))
	delete(report.Rows, "@schema")
	var s *store
	for i := 0; i < 3; i++ {
		s, err = testOpenStore(dir)
		g02Check(t, err)
		after := appAllRows(t, s.db)
		delete(after, "@schema")
		after = rowsWithoutRuntimeRecovery(after)
		if !reflect.DeepEqual(report.Rows, after) {
			t.Fatal("migration rewrote genuine62 rows/DDL/history")
		}
		got, err := (grubberyJournal{s}).Load(t.Context(), report.Record.Plan.RequestID)
		g02Check(t, err)
		if !reflect.DeepEqual(got, report.Record) {
			t.Fatal("migration lost original acknowledged uncertainty")
		}
		g02Check(t, s.close())
	}
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	review := runtimeReviewFixture(t, report.Record, nil)
	saved, err := (grubberyRuntimeJournal{s}).Apply(t.Context(), report.Record.Plan.RequestID, strings.Repeat("5", 64), review, review.ObservedAt.Add(time.Second))
	g02Check(t, err)
	if saved.ValidateAncestry(report.Record) != nil {
		t.Fatal("populated63 audit invalid")
	}
	before := rowsWithoutRuntimeRecovery(appAllRows(t, s.db))
	delete(before, "@schema")
	if !reflect.DeepEqual(report.Rows, before) {
		t.Fatal("audit changed original62 rows")
	}
}

func TestAppRuntimeStoreRetainsBoundedAuditHistoryAndRefusesOverflow(t *testing.T) {
	s, record, _ := runtimeStoreFixture(t)
	j := grubberyRuntimeJournal{s}
	var previous *grubbery.RuntimeRecoveryReceipt
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	for i := 0; i < grubbery.MaxRuntimeRecoveries; i++ {
		review := runtimeReviewFixture(t, record, previous)
		receipt, err := j.Apply(ctx, record.Plan.RequestID, stopEvidenceDigest([]byte("bounded-runtime-recovery-"+strconv.Itoa(i))), review, review.ObservedAt.Add(time.Second))
		g02Check(t, err)
		previous = &receipt
	}
	before := appAllRows(t, s.db)
	review := runtimeReviewFixture(t, record, previous)
	if _, err := j.Apply(ctx, record.Plan.RequestID, strings.Repeat("2", 64), review, review.ObservedAt.Add(time.Second)); !errors.Is(err, grubbery.ErrAdmission) {
		t.Fatal("unbounded audit history admitted", err)
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("count refusal wrote original/audit state")
	}
	latest, err := j.Latest(ctx, record.Plan.RequestID)
	g02Check(t, err)
	if latest == nil || !reflect.DeepEqual(*latest, *previous) {
		t.Fatal("overflow changed effective binding")
	}
}

// Exit after the FULL-synchronous audit commit without closing SQLite or sending
// a response. This verifies process death/lost reply, not host power-loss safety.
func TestAppRuntimeStoreCrashHelper(t *testing.T) {
	dir := os.Getenv("OMABIT_TEST_APP_RUNTIME_CRASH_DIRECTORY")
	if dir == "" {
		t.Skip("private child only")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "review.json"))
	g02Check(t, err)
	var review grubbery.RuntimeRecoveryReview
	g02Check(t, json.Unmarshal(raw, &review))
	s, err := testOpenStore(dir)
	g02Check(t, err)
	receipt, err := (grubberyRuntimeJournal{s}).Apply(t.Context(), review.OriginalRequestID, strings.Repeat("5", 64), review, review.ObservedAt.Add(time.Second))
	g02Check(t, err)
	raw, err = json.Marshal(receipt)
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(dir, "committed.json"), raw, 0600))
	os.Exit(31)
}

func TestAppRuntimeStoreProcessDeathReplaysCommittedAuditAndPreservesActual62(t *testing.T) {
	dir := privateDir(t)
	loadOriginal62AppFixture(t, dir)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	var original string
	g02Check(t, s.db.QueryRow("SELECT id FROM grubbery_app_operations").Scan(&original))
	record, err := (grubberyJournal{s}).Load(t.Context(), original)
	g02Check(t, err)
	before := rowsWithoutRuntimeRecovery(appAllRows(t, s.db))
	review := runtimeReviewFixture(t, record, nil)
	raw, err := json.Marshal(review)
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(dir, "review.json"), raw, 0600))
	g02Check(t, s.close())
	executable, err := os.Executable()
	g02Check(t, err)
	child := exec.CommandContext(t.Context(), executable, "-test.run=^TestAppRuntimeStoreCrashHelper$", "-test.timeout=30s")
	child.Env = append(os.Environ(), "OMABIT_TEST_APP_RUNTIME_CRASH_DIRECTORY="+dir)
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 31 {
		t.Fatalf("child did not reach committed audit exit: %v %s", err, output)
	}
	raw, err = os.ReadFile(filepath.Join(dir, "committed.json"))
	g02Check(t, err)
	var committed grubbery.RuntimeRecoveryReceipt
	g02Check(t, json.Unmarshal(raw, &committed))
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	if !reflect.DeepEqual(before, rowsWithoutRuntimeRecovery(appAllRows(t, s.db))) {
		t.Fatal("process death changed original62 rows, events or lease")
	}
	j := grubberyRuntimeJournal{s}
	got, err := j.Load(t.Context(), original, committed.RequestID)
	g02Check(t, err)
	if !reflect.DeepEqual(got, committed) {
		t.Fatal("lost-reply lookup changed committed audit")
	}
	after := appAllRows(t, s.db)
	replay, err := j.Apply(t.Context(), original, committed.RequestID, review, review.ObservedAt.Add(24*time.Hour))
	g02Check(t, err)
	if !reflect.DeepEqual(replay, committed) || !reflect.DeepEqual(after, appAllRows(t, s.db)) {
		t.Fatal("expired exact replay appended an audit or changed original uncertainty")
	}
}
