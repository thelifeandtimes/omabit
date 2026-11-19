package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func recordingIntentFixture(t *testing.T) (*store, string, recordingIntent) {
	t.Helper()
	dir := privateDir(t)
	s, err := testOpenStore(dir)
	g02Check(t, err)
	t.Cleanup(func() { s.close() })
	_, _, err = s.register("operator", registration("one", "register", 18087))
	g02Check(t, err)
	op, _, err := s.accept("operator", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	g02Check(t, err)
	op, v, err := s.beginOperation(op.ID)
	g02Check(t, err)
	// Synthetic journal boundary only. Native factory/worker admission is a
	// separate required gate, and this fixture never calls an engine or unit.
	v.Runtime.Prepared = true
	v.Runtime.Runtime.Containers = []runtimequalify.ContainerRecord{{ID: strings.Repeat("a", 64), Kind: "http-boot", CreateDispatched: true}}
	g02Check(t, s.runtimeUpdate("one", op.ID, v.Runtime))
	v, err = loadInstance(s.db, "one")
	g02Check(t, err)
	digest, err := recordingStateDigest(v.Runtime)
	g02Check(t, err)
	i := recordingIntent{Version: 1, JobID: strings.Repeat("1", 32), JobSHA256: strings.Repeat("2", 64), RunnerID: s.runnerID, Mode: s.mode, Actor: op.Actor, OperationID: op.ID, RequestID: op.RequestID, InstanceID: op.InstanceID, RuntimeID: v.Runtime.ID, ConfigurationRevision: op.ConfigRevision, RuntimeRevision: v.RuntimeRevision, StateSHA256: digest, ContainerID: strings.Repeat("a", 64), ExecutableSHA256: strings.Repeat("3", 64), AdmissionSHA256: strings.Repeat("4", 64), CreatedAt: time.Unix(1700000000, 123).UTC()}
	return s, dir, i
}

func TestRecordingIntentCommitsExactBindingWithoutChangingOriginalRowsAndReplaysHistory(t *testing.T) {
	s, dir, i := recordingIntentFixture(t)
	before := appAllRows(t, s.db)
	delete(before, "runtime_recording_intents")
	delete(before, "runtime_recording_head")
	got, replay, err := s.admitRecordingIntent(i)
	g02Check(t, err)
	if replay || !reflect.DeepEqual(got, i) {
		t.Fatal("new intent changed binding")
	}
	after := appAllRows(t, s.db)
	delete(after, "runtime_recording_intents")
	delete(after, "runtime_recording_head")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("recording intent changed original state/operation/history")
	}
	all := appAllRows(t, s.db)
	for pass := 0; pass < 2; pass++ {
		got, replay, err = s.admitRecordingIntent(i)
		if err != nil || !replay || !reflect.DeepEqual(got, i) || !reflect.DeepEqual(all, appAllRows(t, s.db)) {
			t.Fatal("saved replay changed original receipt", err)
		}
		g02Check(t, s.close())
		s, err = testOpenStore(dir)
		g02Check(t, err)
	}
	defer s.close()
	g02Check(t, s.reconcileInterrupted())
	after = appAllRows(t, s.db)
	got, replay, err = s.admitRecordingIntent(i)
	if err != nil || !replay || !reflect.DeepEqual(got, i) || !reflect.DeepEqual(after, appAllRows(t, s.db)) {
		t.Fatal("interrupted saved replay became live admission", err)
	}
	changed := i
	changed.JobID = strings.Repeat("5", 32)
	if _, _, err = s.admitRecordingIntent(changed); err == nil {
		t.Fatal("second job admitted for original operation")
	}
}

func TestRecordingIntentRefusesWrongAuthorityOrStaleStateWithoutSaving(t *testing.T) {
	for _, scenario := range []string{"actor", "request", "runner", "mode", "runtime", "revision", "runtime-revision", "state", "container", "quarantine", "queued", "ambiguous-container", "other-container", "uncertain-removed"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, i := recordingIntentFixture(t)
			switch scenario {
			case "actor":
				i.Actor = "other"
			case "request":
				i.RequestID = "other"
			case "runner":
				i.RunnerID = "other"
			case "mode":
				i.Mode = DevelopmentMode
			case "runtime":
				i.RuntimeID = strings.Repeat("f", 32)
			case "revision":
				i.ConfigurationRevision++
			case "runtime-revision":
				i.RuntimeRevision++
			case "state":
				i.StateSHA256 = strings.Repeat("f", 64)
			case "container":
				i.ContainerID = strings.Repeat("f", 64)
			case "queued":
				op, err := loadOperation(s.db, i.OperationID)
				g02Check(t, err)
				op.Status = "queued"
				tx, err := s.db.Begin()
				g02Check(t, err)
				g02Check(t, saveOperation(tx, op))
				g02Check(t, tx.Commit())
			case "quarantine", "ambiguous-container", "other-container", "uncertain-removed":
				v, err := loadInstance(s.db, i.InstanceID)
				g02Check(t, err)
				if scenario == "quarantine" {
					v.Runtime.Storage.Uncertain = true
				} else {
					extra := v.Runtime.Runtime.Containers[0]
					if scenario == "other-container" || scenario == "uncertain-removed" {
						extra.ID = strings.Repeat("b", 64)
					}
					if scenario == "uncertain-removed" {
						extra.Removed, extra.CreateUncertain = true, true
					}
					v.Runtime.Runtime.Containers = append(v.Runtime.Runtime.Containers, extra)
				}
				g02Check(t, s.runtimeUpdate(i.InstanceID, i.OperationID, v.Runtime))
				v, err = loadInstance(s.db, i.InstanceID)
				g02Check(t, err)
				i.RuntimeRevision, i.StateSHA256 = v.RuntimeRevision, mustRecordingStateDigest(t, v.Runtime)
			}
			before := appAllRows(t, s.db)
			if _, _, err := s.admitRecordingIntent(i); err == nil || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("stale/unowned intent admitted or refusal changed state", err)
			}
		})
	}
}

func TestRecordingIntentFreshFenceNeverTurnsHistoricalReplayIntoAuthority(t *testing.T) {
	for _, scenario := range []string{"interrupted", "runtime-revision", "different-intent", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, i := recordingIntentFixture(t)
			before := appAllRows(t, s.db)
			if s.checkRecordingIntent(t.Context(), i) == nil || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("unsaved intent granted authority or changed state")
			}
			_, _, err := s.admitRecordingIntent(i)
			g02Check(t, err)
			before = appAllRows(t, s.db)
			g02Check(t, s.checkRecordingIntent(t.Context(), i))
			if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("fresh fence changed history")
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			selected := i
			switch scenario {
			case "interrupted":
				g02Check(t, s.reconcileInterrupted())
			case "runtime-revision":
				v, err := loadInstance(s.db, i.InstanceID)
				g02Check(t, err)
				// Even unchanged runtime bytes have a different saved generation.
				g02Check(t, s.runtimeUpdate(i.InstanceID, i.OperationID, v.Runtime))
			case "different-intent":
				selected.JobSHA256 = strings.Repeat("f", 64)
			case "canceled":
				cancel()
			}
			before = appAllRows(t, s.db)
			if err := s.checkRecordingIntent(ctx, selected); err != errRecordingHistory || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("stale/refused fence became current authority or changed history", err)
			}
			got, replay, err := s.admitRecordingIntent(i)
			if err != nil || !replay || !reflect.DeepEqual(got, i) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("historical replay was lost or rewrote its original", err)
			}
		})
	}
}

func mustRecordingStateDigest(t *testing.T, v any) string {
	t.Helper()
	h, err := recordingStateDigest(v)
	g02Check(t, err)
	return h
}

func TestRecordingIntentCorruptionRefusesWithoutRepair(t *testing.T) {
	for _, scenario := range []string{"changed-body", "missing-operation", "noncanonical", "unknown-field", "trigger", "missing-table", "missing-tail", "missing-head", "changed-head", "oversize"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir, i := recordingIntentFixture(t)
			_, _, err := s.admitRecordingIntent(i)
			g02Check(t, err)
			switch scenario {
			case "changed-body":
				_, err = s.db.Exec("UPDATE runtime_recording_intents SET body=?", []byte("partial"))
			case "missing-operation":
				_, err = s.db.Exec("PRAGMA foreign_keys=OFF; DELETE FROM operations WHERE id=?", i.OperationID)
			case "noncanonical":
				raw, _ := i.bytes()
				_, err = s.db.Exec("UPDATE runtime_recording_intents SET body=?", append(raw, '\n'))
			case "unknown-field":
				raw, _ := i.bytes()
				var fields map[string]json.RawMessage
				g02Check(t, json.Unmarshal(raw, &fields))
				fields["unexpected"] = json.RawMessage("true")
				raw, err = json.Marshal(fields)
				g02Check(t, err)
				_, err = s.db.Exec("UPDATE runtime_recording_intents SET body=?", raw)
			case "trigger":
				_, err = s.db.Exec("CREATE TRIGGER unexpected_recording AFTER INSERT ON runtime_recording_intents BEGIN SELECT 1; END")
			case "missing-table":
				_, err = s.db.Exec("DROP TABLE runtime_recording_intents")
			case "missing-tail":
				_, err = s.db.Exec("DELETE FROM runtime_recording_intents")
			case "missing-head":
				_, err = s.db.Exec("DELETE FROM runtime_recording_head")
			case "changed-head":
				_, err = s.db.Exec("UPDATE runtime_recording_head SET digest=?", strings.Repeat("f", 64))
			case "oversize":
				_, err = s.db.Exec("UPDATE runtime_recording_intents SET body=?", bytes.Repeat([]byte("x"), 4097))
			}
			g02Check(t, err)
			before := appAllRows(t, s.db)
			g02Check(t, s.close())
			if opened, err := testOpenStore(dir); err == nil {
				opened.close()
				t.Fatal("corrupt history opened")
			}
			db := g02RawDB(t, dir)
			defer db.Close()
			if !reflect.DeepEqual(before, appAllRows(t, db)) {
				t.Fatal("opening repaired or relabeled corrupt history")
			}
		})
	}
}

func TestRecordingIntentProcessExitAfterCommitPreservesExactReplay(t *testing.T) {
	if root := os.Getenv("OMABIT_RECORDING_STORE_EXIT_ROOT"); root != "" {
		if !strings.HasPrefix(root, os.TempDir()+"/TestRecordingIntentProcessExitAfterCommitPreservesExactReplay") {
			t.Fatal("fresh exact test child directory required")
		}
		s, err := testOpenStore(root)
		g02Check(t, err)
		raw := os.Getenv("OMABIT_RECORDING_STORE_EXIT_INPUT")
		var i recordingIntent
		g02Check(t, json.Unmarshal([]byte(raw), &i))
		_, _, err = s.admitRecordingIntent(i)
		g02Check(t, err)
		os.Exit(0) // no response or orderly store close
	}
	s, dir, i := recordingIntentFixture(t)
	g02Check(t, s.close())
	raw, err := i.bytes()
	g02Check(t, err)
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecordingIntentProcessExitAfterCommitPreservesExactReplay$")
	cmd.Env = append(os.Environ(), "OMABIT_RECORDING_STORE_EXIT_ROOT="+dir, "OMABIT_RECORDING_STORE_EXIT_INPUT="+string(raw))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal("private child failed", err, string(out))
	}
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	before := appAllRows(t, s.db)
	got, replay, err := s.admitRecordingIntent(i)
	if err != nil || !replay || !reflect.DeepEqual(got, i) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("committed/no-response intent was recreated", err)
	}
}

func rowsWithoutEmptyRecordingSuccessor(all map[string][]string) map[string][]string {
	all = rowsWithoutEmptyRecordingControl(all)
	rows, head := all["runtime_recording_intents"], all["runtime_recording_head"]
	if len(rows) == 1 && rows[0] == recordingIntentDDL && len(head) == 2 && head[0] == recordingHeadDDL && head[1] == `[1,0,""]` {
		delete(all, "runtime_recording_intents")
		delete(all, "runtime_recording_head")
	}
	return all
}

func rowsWithoutEmptyRecordingControl(all map[string][]string) map[string][]string {
	rows, head := all["runtime_recording_control_events"], all["runtime_recording_control_head"]
	if len(rows) == 1 && rows[0] == recordingControlDDL && len(head) == 2 && head[0] == recordingControlHeadDDL && head[1] == `[1,0,""]` {
		delete(all, "runtime_recording_control_events")
		delete(all, "runtime_recording_control_head")
	}
	return all
}

func TestRecordingIntentMigrationRejectsMixedOldSchemaAndRollsBack(t *testing.T) {
	dir := privateDir(t)
	loadOriginal62AppFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec(recordingIntentDDL)
	g02Check(t, err)
	before := appAllRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("unowned successor table adopted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	if !reflect.DeepEqual(before, appAllRows(t, db)) {
		t.Fatal("mixed schema was repaired or promoted")
	}
}

func TestRecordingIntentCanonicalCodecKeepsWideRevision(t *testing.T) {
	_, _, i := recordingIntentFixture(t)
	i.ConfigurationRevision = 9007199254740999
	raw, err := i.bytes()
	g02Check(t, err)
	var decoded recordingIntent
	g02Check(t, json.Unmarshal(raw, &decoded))
	if decoded.ConfigurationRevision != i.ConfigurationRevision || !bytes.Contains(raw, []byte("9007199254740999")) {
		t.Fatal("wide revision truncated")
	}
}

func TestRecordingIntentConcurrentAdmissionsKeepOneOriginalOperationAndExactHead(t *testing.T) {
	s, dir, i := recordingIntentFixture(t)
	other, err := testOpenStore(dir)
	g02Check(t, err)
	defer other.close()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for n, source := range []*store{s, other} {
		wg.Add(1)
		go func(index int, source *store) {
			defer wg.Done()
			selected := i
			if index == 1 {
				selected.JobID, selected.JobSHA256 = strings.Repeat("5", 32), strings.Repeat("6", 64)
			}
			_, replay, err := source.admitRecordingIntent(selected)
			if err == nil && replay {
				err = errRecordingHistory // distinct jobs cannot be exact replay
			}
			results <- err
		}(n, source)
	}
	wg.Wait()
	close(results)
	passed, refused := 0, 0
	for err := range results {
		if err == nil {
			passed++
		} else {
			refused++
		}
	}
	if passed != 1 || refused != 1 {
		t.Fatal("competing jobs acquired the same operation", passed, refused)
	}
	g02Check(t, validateRecordingHistory(s.db))
	head, _, err := recordingHead(s.db)
	g02Check(t, err)
	if head != 1 {
		t.Fatal("head advanced without exactly one intent")
	}
}

func TestRecordingIntentHeadCommitFailureRollsBackIntentAndOriginals(t *testing.T) {
	s, _, i := recordingIntentFixture(t)
	_, err := s.db.Exec("CREATE TEMP TRIGGER fail_recording_head BEFORE UPDATE ON runtime_recording_head BEGIN SELECT RAISE(ABORT,'injected head commit refusal'); END")
	g02Check(t, err)
	before := appAllRows(t, s.db)
	if _, _, err := s.admitRecordingIntent(i); err == nil {
		t.Fatal("head refusal committed an intent")
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("failed head commit changed originals or left an orphan intent")
	}
	g02Check(t, validateRecordingHistory(s.db))
}
