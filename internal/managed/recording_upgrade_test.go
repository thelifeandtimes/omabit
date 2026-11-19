package managed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRecordingStoreActualOriginal63Qualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_STORE_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit actual source-writer rehearsal only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-store-upgrade-") {
		t.Fatal("fresh disposable source qualification required")
	}
	var report struct {
		Schema int                 `json:"schema"`
		Rows   map[string][]string `json:"all_original_rows"`
		Op     protocol.Operation  `json:"pending_operation"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "original-report.json"))
	g02Check(t, err)
	g02Check(t, json.Unmarshal(raw, &report))
	if report.Schema != 63 || report.Op.Status != "running" || report.Op.Kind != "start" {
		t.Fatal("actual63 pending runtime fixture required")
	}
	delete(report.Rows, "@schema")
	dir := filepath.Join(root, "successor")
	g02Check(t, os.Mkdir(dir, 0700))
	raw, err = os.ReadFile(filepath.Join(root, "original", "runner.sqlite"))
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), raw, 0600))
	for pass := 0; pass < 3; pass++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := rowsWithoutEmptyRecordingSuccessor(appAllRows(t, s.db))
		delete(after, "@schema")
		if !reflect.DeepEqual(report.Rows, after) {
			t.Fatal("64 migration rewrote actual63 rows/DDL/history or dispatched work")
		}
		g02Check(t, s.close())
	}
	s, err := testOpenStore(dir)
	g02Check(t, err)
	v, err := loadInstance(s.db, report.Op.InstanceID)
	g02Check(t, err)
	i := recordingIntent{Version: 1, JobID: strings.Repeat("1", 32), JobSHA256: strings.Repeat("2", 64), RunnerID: s.runnerID, Mode: s.mode, Actor: report.Op.Actor, OperationID: report.Op.ID, RequestID: report.Op.RequestID, InstanceID: report.Op.InstanceID, RuntimeID: v.Runtime.ID, ConfigurationRevision: report.Op.ConfigRevision, RuntimeRevision: v.RuntimeRevision, StateSHA256: mustRecordingStateDigest(t, v.Runtime), ContainerID: strings.Repeat("a", 64), ExecutableSHA256: strings.Repeat("3", 64), AdmissionSHA256: strings.Repeat("4", 64), CreatedAt: time.Unix(1700000000, 123).UTC()}
	got, replay, err := s.admitRecordingIntent(i)
	g02Check(t, err)
	if replay || !reflect.DeepEqual(got, i) {
		t.Fatal("new intent rebound original63 operation")
	}
	after := appAllRows(t, s.db)
	delete(after, "@schema")
	delete(after, "runtime_recording_intents")
	delete(after, "runtime_recording_head")
	if !reflect.DeepEqual(report.Rows, after) {
		t.Fatal("recording intent changed an original63 row")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	before := appAllRows(t, s.db)
	got, replay, err = s.admitRecordingIntent(i)
	if err != nil || !replay || !reflect.DeepEqual(got, i) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("reopened receipt was replaced or replayed", err)
	}
	head, _, err := recordingHead(s.db)
	g02Check(t, err)
	if head != 1 {
		t.Fatal("wrong recording intent head")
	}
}
