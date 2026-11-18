package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
)

func grubberyFixture(t *testing.T, s *store) (grubbery.Record, []deskwork.SourceFile) {
	t.Helper()
	framework, err := appdev.ReadFramework(appdev.GrubberyFramework)
	g02Check(t, err)
	empty := deskwork.EmptyOwnership()
	plan, err := deskwork.PlanDeployment(empty, empty.Clay, framework.Source, empty.Clay)
	g02Check(t, err)
	deployment := deploymentRecordFixture(t)
	deployment.Plan = plan
	deployment.Intent.Intent.ID = stopEvidenceDigest([]byte("schema61-synthetic-framework-receipt"))
	deployment.Intent.SourceName = "grubbery"
	deployment.Intent.PlanDigest = plan.Digest
	deployment.Intent.Intent.SourceDigest = framework.Source.Digest
	deployment.Intent.Intent.Target.Desk = "grubbery"
	receipt, _, err := s.createDeskDeployment(deployment)
	g02Check(t, err)
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "dispatched"))
	g02Check(t, err)
	receipt, err = s.appendDeskStep(receipt, deskStep(receipt, "acknowledged"))
	g02Check(t, err)
	// Synthetic canonical Clay evidence: hashes/paths only. This is not a native
	// compilation, source-conversion, install or activation qualification.
	entries := append([]deskwork.File{}, framework.Source.Files...)
	for i := range entries {
		entries[i].Bytes = 0
		entries[i].SHA256 = strings.Repeat("f", 64)
	}
	clay, err := deskwork.NewManifest(entries)
	g02Check(t, err)
	owned, err := deskwork.VerifyDeployment(plan, clay, clay)
	g02Check(t, err)
	observation := deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v456", Publication: "private"}
	_, err = s.completeDeskDeployment(receipt, observation, owned, "")
	g02Check(t, err)
	binding := grubbery.Binding{RunnerID: s.runnerID, Mode: s.mode, Actor: "fixture", Target: deployment.Intent.Intent.Target, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1, PolicyRevision: 0, Continuity: deployment.Intent.Continuity, FrameworkReceiptID: receipt.Intent.ID, FrameworkBaseline: observation, AppID: "state"}
	files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: []byte(":: metadata-only fixture\n")}}
	source := []deskwork.File{}
	for _, f := range files {
		source = append(source, deskwork.File{Path: f.Path, Bytes: int64(len(f.Bytes)), SHA256: stopEvidenceDigest(f.Bytes)})
	}
	manifest, err := deskwork.NewManifest(source)
	g02Check(t, err)
	appPlan, err := grubbery.NewPlan(binding, strings.Repeat("c", 64), strings.Repeat("1", 64), manifest, files, grubbery.Before{SnapshotDigest: strings.Repeat("e", 64), SourceState: "absent", DeskState: "absent"})
	g02Check(t, err)
	r, err := grubbery.NewRecord(appPlan, time.Now().UTC())
	g02Check(t, err)
	return r, files
}
func appFixtureEvent(t *testing.T, r grubbery.Record, step grubbery.Step, state string) grubbery.Event {
	t.Helper()
	raw, _ := json.Marshal(r.Plan.Binding)
	binding := stopEvidenceDigest(raw)
	e := grubbery.Event{StepID: step.ID, State: state, At: time.Now().UTC()}
	switch state {
	case "dispatched":
		e.Review = &grubbery.Review{Version: 1, PlanDigest: r.Plan.Digest, StepID: step.ID, BindingDigest: binding, PolicyRevision: r.Plan.Binding.PolicyRevision, NativeEvidenceDigest: strings.Repeat("a", 64)}
	case "acknowledged":
		e.ResponseDigest = strings.Repeat("a", 64)
	case "uncertain":
		e.Reason = "transport-outcome-unknown"
	case "observed":
		e.Proof = &grubbery.Proof{Version: 1, PlanDigest: r.Plan.Digest, StepID: step.ID, BindingDigest: binding, NativeEvidenceDigest: strings.Repeat("b", 64)}
	}
	next, err := r.WithEvent(e)
	g02Check(t, err)
	return next.Events[len(next.Events)-1]
}
func appAppendFixture(t *testing.T, j grubberyJournal, r grubbery.Record, step grubbery.Step, state string) grubbery.Record {
	t.Helper()
	next, err := j.Append(context.Background(), r.Plan.RequestID, r.Revision, appFixtureEvent(t, r, step, state))
	g02Check(t, err)
	return next
}
func appNewFixture(t *testing.T, previous grubbery.Record, files []deskwork.SourceFile, before grubbery.Before, id string) grubbery.Record {
	t.Helper()
	p, err := grubbery.NewPlan(previous.Plan.Binding, previous.Plan.StageID, id, previous.Plan.Source.Manifest, files, before)
	g02Check(t, err)
	r, err := grubbery.NewRecord(p, time.Now().UTC())
	g02Check(t, err)
	return r
}
func TestGrubberyJournalPreservesOriginalEventsAndLeaseAcrossReopen(t *testing.T) {
	ctx := context.Background()
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	r, files := grubberyFixture(t, s)
	j := grubberyJournal{s}
	r, err = j.Create(ctx, r)
	g02Check(t, err)
	originalHeader, _ := json.Marshal(r)
	step := r.Plan.Steps[0]
	r = appAppendFixture(t, j, r, step, "dispatched")
	r = appAppendFixture(t, j, r, step, "uncertain")
	before := r
	competitor := appNewFixture(t, r, files, r.Plan.Before, strings.Repeat("2", 64))
	if _, err = j.Create(ctx, competitor); !errors.Is(err, grubbery.ErrNamespaceBusy) {
		t.Fatal("unfinished lease not preserved", err)
	}
	for n := 0; n < 3; n++ {
		g02Check(t, s.close())
		s, err = testOpenStore(dir)
		g02Check(t, err)
		j = grubberyJournal{s}
		r, err = j.Load(ctx, r.Plan.RequestID)
		g02Check(t, err)
		if !reflect.DeepEqual(before, r) {
			t.Fatal("opening interrupted/rewrote app phase history")
		}
	}
	defer s.close()
	var saved []byte
	g02Check(t, s.db.QueryRow("SELECT body FROM grubbery_app_operations WHERE id=?", r.Plan.RequestID).Scan(&saved))
	if string(saved) != string(originalHeader) {
		t.Fatal("immutable operation header replaced")
	}
	r = appAppendFixture(t, j, r, step, "observed")
	for _, step := range r.Plan.Steps[1:] {
		r = appAppendFixture(t, j, r, step, "dispatched")
		r = appAppendFixture(t, j, r, step, "observed")
	}
	var leases int
	g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_leases").Scan(&leases))
	if leases != 0 {
		t.Fatal("completed lease not released")
	}
	own, err := grubbery.DeriveOwnership([]grubbery.Record{r})
	g02Check(t, err)
	b := grubbery.Before{SnapshotDigest: strings.Repeat("e", 64), SourceState: "owned", SourceOwnerSHA256: own.OwnerSHA256, DeskState: "owned", DeskOwnerSHA256: own.OwnerSHA256, Epoch: own.Epoch, ReleaseRoot: own.ReleaseRoot}
	next := appNewFixture(t, r, files, b, strings.Repeat("3", 64))
	_, err = j.Create(ctx, next)
	g02Check(t, err)
	if _, err = j.Append(ctx, r.Plan.RequestID, r.Revision, grubbery.Event{}); !errors.Is(err, grubbery.ErrPhase) {
		t.Fatal("terminal original appended/reclaimed successor lease", err)
	}
}
func TestGrubberyJournalAtomicConcurrentPhaseCAS(t *testing.T) {
	s := testStore(t)
	r, _ := grubberyFixture(t, s)
	j := grubberyJournal{s}
	ctx := context.Background()
	_, err := j.Create(ctx, r)
	g02Check(t, err)
	event := appFixtureEvent(t, r, r.Plan.Steps[0], "dispatched")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := j.Append(ctx, r.Plan.RequestID, 0, event); results <- err }()
	}
	wg.Wait()
	close(results)
	wins, loses := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, grubbery.ErrCAS) {
			loses++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || loses != 1 {
		t.Fatal("CAS did not yield one writer", wins, loses)
	}
	saved, err := j.Load(ctx, r.Plan.RequestID)
	g02Check(t, err)
	if saved.Revision != 1 {
		t.Fatal("concurrent duplicate event")
	}
	// A constraint failure while appending cannot leave an updated revision or
	// release the namespace. The index exists only in this synthetic store.
	_, err = s.db.Exec("CREATE UNIQUE INDEX fixture_app_failure ON grubbery_app_events(id)")
	g02Check(t, err)
	_, err = j.Append(ctx, r.Plan.RequestID, 1, appFixtureEvent(t, saved, r.Plan.Steps[0], "acknowledged"))
	if err == nil {
		t.Fatal("injected append failure ignored")
	}
	after, err := j.Load(ctx, r.Plan.RequestID)
	g02Check(t, err)
	if !reflect.DeepEqual(saved, after) {
		t.Fatal("failed transaction rewrote phase")
	}
}
func TestGrubberyJournalNoEffectAbandonmentIsDurableAndReleasesExactLease(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	r, files := grubberyFixture(t, s)
	j := grubberyJournal{s}
	ctx := context.Background()
	_, err = j.Create(ctx, r)
	g02Check(t, err)
	call := grubbery.Call{RunnerID: r.Plan.Binding.RunnerID, Mode: r.Plan.Binding.Mode, Actor: r.Plan.Binding.Actor, Target: r.Plan.Binding.Target, AppID: r.Plan.Binding.AppID, RequestID: r.Plan.RequestID, PlanDigest: r.Plan.Digest, StepID: r.Plan.Steps[0].ID}
	e := grubbery.Execution{Store: j}
	abandoned, err := e.Abandon(ctx, call)
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	j = grubberyJournal{s}
	e.Store = j
	got, err := e.Abandon(ctx, call)
	g02Check(t, err)
	if !reflect.DeepEqual(got, abandoned) {
		t.Fatal("abandonment replay changed")
	}
	next := appNewFixture(t, r, files, r.Plan.Before, strings.Repeat("4", 64))
	_, err = j.Create(ctx, next)
	g02Check(t, err)
}
func TestGrubberyJournalRefusesChangedIdentityOrFrameworkAncestry(t *testing.T) {
	for _, change := range []string{"runner", "mode", "framework", "continuity", "baseline", "before"} {
		t.Run(change, func(t *testing.T) {
			s := testStore(t)
			r, files := grubberyFixture(t, s)
			binding := r.Plan.Binding
			before := r.Plan.Before
			switch change {
			case "runner":
				binding.RunnerID = "other"
			case "mode":
				binding.Mode = "other"
			case "framework":
				binding.FrameworkReceiptID = strings.Repeat("f", 64)
			case "continuity":
				binding.Continuity = strings.Repeat("f", 64)
			case "baseline":
				binding.FrameworkBaseline.Revision++
			case "before":
				before.SourceState = "owned"
				marker, _ := binding.OwnerMarker()
				before.SourceOwnerSHA256 = stopEvidenceDigest(marker)
			}
			p, err := grubbery.NewPlan(binding, r.Plan.StageID, r.Plan.RequestID, r.Plan.Source.Manifest, files, before)
			g02Check(t, err)
			r, err = grubbery.NewRecord(p, r.CreatedAt)
			g02Check(t, err)
			if _, err = (grubberyJournal{s}).Create(context.Background(), r); err == nil {
				t.Fatal("changed ancestry admitted")
			}
			var count int
			g02Check(t, s.db.QueryRow("SELECT count(*) FROM grubbery_app_operations").Scan(&count))
			if count != 0 {
				t.Fatal("refusal left partial operation")
			}
		})
	}
}
func TestGrubberyJournalCorruptionRefusesWithoutRepair(t *testing.T) {
	changes := map[string]string{
		"missing-event": "DELETE FROM grubbery_app_events", "bad-event": "UPDATE grubbery_app_events SET body='{}'", "event-rebound": "UPDATE grubbery_app_events SET digest='bad'", "missing-lease": "DELETE FROM grubbery_app_leases", "bad-lease": "UPDATE grubbery_app_leases SET namespace='other'", "header": "UPDATE grubbery_app_operations SET body='{}'", "ordinal": "UPDATE grubbery_app_operations SET ordinal=2", "missing-all": "DELETE FROM grubbery_app_events;DELETE FROM grubbery_app_leases;DELETE FROM grubbery_app_operations", "clock": "UPDATE grubbery_app_clock SET ordinal=2", "table": "DROP TABLE grubbery_app_leases", "revision": "UPDATE grubbery_app_operations SET revision=0", "framework": "UPDATE grubbery_app_operations SET framework_id='missing'", "trigger": "CREATE TRIGGER fixture_extra AFTER INSERT ON grubbery_app_events BEGIN SELECT 1; END",
	}
	for name, query := range changes {
		t.Run(name, func(t *testing.T) {
			dir := privateDir(t)
			s, err := testOpenStore(dir)
			g02Check(t, err)
			r, _ := grubberyFixture(t, s)
			j := grubberyJournal{s}
			_, err = j.Create(context.Background(), r)
			g02Check(t, err)
			appAppendFixture(t, j, r, r.Plan.Steps[0], "dispatched")
			g02Check(t, s.close())
			db := g02RawDB(t, dir)
			_, err = db.Exec(query)
			g02Check(t, err)
			before := appAllRows(t, db)
			g02Check(t, db.Close())
			if s, err := testOpenStore(dir); err == nil {
				s.close()
				t.Fatal("corruption accepted")
			}
			db = g02RawDB(t, dir)
			defer db.Close()
			if !reflect.DeepEqual(before, appAllRows(t, db)) {
				t.Fatal("opening repaired/rewrote corrupt state")
			}
		})
	}
}

// The child exits after FULL-synchronous dispatch commit without closing SQLite.
// Reopening must retain its original fence rather than redispatch or invent an
// interruption event. This checks process death, not host power-loss durability.
func TestGrubberyJournalCrashHelper(t *testing.T) {
	dir := os.Getenv("OMABIT_TEST_APP_CRASH_DIRECTORY")
	if dir == "" {
		t.Skip("private child only")
	}
	s, err := testOpenStore(dir)
	g02Check(t, err)
	j := grubberyJournal{s}
	r, err := j.Load(t.Context(), os.Getenv("OMABIT_TEST_APP_CRASH_REQUEST"))
	g02Check(t, err)
	appAppendFixture(t, j, r, r.Plan.Steps[0], "dispatched")
	os.Exit(29)
}
func TestGrubberyJournalProcessDeathPreservesCommittedDispatch(t *testing.T) {
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	r, _ := grubberyFixture(t, s)
	_, err = (grubberyJournal{s}).Create(t.Context(), r)
	g02Check(t, err)
	g02Check(t, s.close())
	executable, err := os.Executable()
	g02Check(t, err)
	child := exec.CommandContext(t.Context(), executable, "-test.run=^TestGrubberyJournalCrashHelper$", "-test.timeout=30s")
	child.Env = append(os.Environ(), "OMABIT_TEST_APP_CRASH_DIRECTORY="+dir, "OMABIT_TEST_APP_CRASH_REQUEST="+r.Plan.RequestID)
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 29 {
		t.Fatalf("child did not reach committed exit: %v %s", err, output)
	}
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	saved, err := (grubberyJournal{s}).Load(t.Context(), r.Plan.RequestID)
	g02Check(t, err)
	if state, _ := saved.State(r.Plan.Steps[0].ID); state != "dispatched" || saved.Revision != 1 {
		t.Fatal("process death rewrote dispatch")
	}
	call := grubbery.Call{RunnerID: r.Plan.Binding.RunnerID, Mode: r.Plan.Binding.Mode, Actor: r.Plan.Binding.Actor, Target: r.Plan.Binding.Target, AppID: r.Plan.Binding.AppID, RequestID: r.Plan.RequestID, PlanDigest: r.Plan.Digest, StepID: r.Plan.Steps[0].ID}
	replay, err := (grubbery.Execution{Store: grubberyJournal{s}}).Advance(t.Context(), call)
	g02Check(t, err)
	if !reflect.DeepEqual(saved, replay) {
		t.Fatal("crash replay altered saved fence")
	}
}
