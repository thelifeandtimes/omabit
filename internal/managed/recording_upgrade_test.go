package managed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
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
	after := rowsWithoutEmptyRecordingControl(appAllRows(t, s.db))
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

func loadOriginal63RecordingFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/recording-original63"
	raw, err := os.ReadFile(filepath.Join(base, "manifest.json"))
	g02Check(t, err)
	var m struct {
		Schema int    `json:"schema"`
		Source string `json:"original_source"`
		Packed string `json:"gzip_sha256"`
		SQL    string `json:"raw_sql_sha256"`
		Helper string `json:"declared_helper_sha256"`
		Inputs string `json:"original_source_manifest_sha256"`
	}
	g02Check(t, json.Unmarshal(raw, &m))
	if stopEvidenceDigest(raw) != "9fe62f73c1cffbc4f8d3bfa021636cea559ebf1ce62e972b003cdcc7d58a28cf" || m.Schema != 63 || m.Source != "1589158b4484c1a7c0f96c188a223e0faa83e141" || m.Helper != "25defad826b12b20b2430adc859b0b6966b57ee65edd1bfd758dd89ac22e5c67" || m.Inputs != "8edc0d810c2dd1cd7adfbbc0ceb3939258ff3c80868c802a14511956f08720af" {
		t.Fatal("actual original63 writer provenance changed")
	}
	for name, pin := range map[string]string{"writer-helper.go.txt": m.Helper, "original-source.json": m.Inputs} {
		data, err := os.ReadFile(filepath.Join(base, name))
		g02Check(t, err)
		if stopEvidenceDigest(data) != pin {
			t.Fatal("frozen source/helper changed", name)
		}
	}
	packed, err := os.ReadFile(filepath.Join(base, "state.sql.gz"))
	g02Check(t, err)
	if stopEvidenceDigest(packed) != m.Packed {
		t.Fatal("actual original63 archive changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	body, err := io.ReadAll(io.LimitReader(z, 4<<20+1))
	g02Check(t, err)
	g02Check(t, z.Close())
	if len(body) > 4<<20 || stopEvidenceDigest(body) != m.SQL {
		t.Fatal("actual original63 SQL changed")
	}
	f, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, f.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(body))
	g02Check(t, err)
	g02Check(t, db.Close())
}

func TestRecordingOriginal63MigrationPreservesAllRowsAuditAndPendingStart(t *testing.T) {
	dir := privateDir(t)
	loadOriginal63RecordingFixture(t, dir)
	db := g02RawDB(t, dir)
	before := appAllRows(t, db)
	delete(before, "@schema")
	var opID string
	g02Check(t, db.QueryRow("SELECT id FROM operations WHERE actor='recording-actor' AND request_id='recording-start'").Scan(&opID))
	op, err := loadOperation(db, opID)
	g02Check(t, err)
	if op.Status != "running" || op.Kind != "start" {
		t.Fatal("original63 pending Start missing")
	}
	var audits int
	g02Check(t, db.QueryRow("SELECT count(*) FROM grubbery_app_runtime_recoveries").Scan(&audits))
	if audits != 1 {
		t.Fatal("original63 linked audit missing")
	}
	g02Check(t, db.Close())
	for pass := 0; pass < 3; pass++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := rowsWithoutEmptyRecordingSuccessor(appAllRows(t, s.db))
		delete(after, "@schema")
		if !reflect.DeepEqual(before, after) {
			t.Fatal("64 migration changed original rows/DDL/access or uncertainty")
		}
		got, err := loadOperation(s.db, opID)
		g02Check(t, err)
		if !reflect.DeepEqual(op, got) {
			t.Fatal("migration replayed or changed original Start")
		}
		g02Check(t, s.close())
	}
}

func TestRecordingOriginal63MigrationRollsBackMissingOriginalAudit(t *testing.T) {
	dir := privateDir(t)
	loadOriginal63RecordingFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec("DELETE FROM grubbery_app_runtime_recoveries")
	g02Check(t, err)
	before := appAllRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("missing actual63 ancestry admitted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	if !reflect.DeepEqual(before, appAllRows(t, db)) {
		t.Fatal("failed64 migration repaired63 or committed recording tables")
	}
}
