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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingControlActualOriginal64Qualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_CONTROL_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit actual-writer source rehearsal only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-control-upgrade-") {
		t.Fatal("fresh disposable source root required")
	}
	var report struct {
		Schema int                 `json:"schema"`
		Rows   map[string][]string `json:"all_original_rows"`
		Op     protocol.Operation  `json:"pending_operation"`
		Intent recordingIntent     `json:"recording_intent"`
	}
	raw, err := os.ReadFile(filepath.Join(root, "original-report.json"))
	g02Check(t, err)
	g02Check(t, json.Unmarshal(raw, &report))
	if report.Schema != 64 || report.Op.Status != "running" || report.Intent.Version != 1 || report.Intent.OperationID != report.Op.ID {
		t.Fatal("actual64 pending recording ancestry missing")
	}
	delete(report.Rows, "@schema")
	dir := filepath.Join(root, "successor")
	g02Check(t, os.Mkdir(dir, 0700))
	raw, err = os.ReadFile(filepath.Join(root, "original/runner.sqlite"))
	g02Check(t, err)
	g02Check(t, os.WriteFile(filepath.Join(dir, "runner.sqlite"), raw, 0600))
	for pass := 0; pass < 3; pass++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := rowsWithoutEmptyRecordingControl(appAllRows(t, s.db))
		delete(after, "@schema")
		if !reflect.DeepEqual(report.Rows, after) {
			t.Fatal("65 migration changed original64 rows/DDL/intent/uncertainty")
		}
		saved, err := loadRecordingIntent(s.db, report.Intent.JobID)
		g02Check(t, err)
		if !reflect.DeepEqual(saved, report.Intent) {
			t.Fatal("original intent changed")
		}
		g02Check(t, s.close())
	}
	s, err := testOpenStore(dir)
	g02Check(t, err)
	policy := recording.DefaultRetentionPolicy()
	policy.MaxJobs = 32
	in := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "control-configure", Kind: "configure", Policy: &policy}
	configured, replay, err := s.applyRecordingControl(t.Context(), "control-owner", in)
	g02Check(t, err)
	if replay {
		t.Fatal("new configuration replayed")
	}
	pin := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "control-pin", Kind: "pin", ExpectedRevision: configured.State.Revision, ExpectedDigest: configured.State.Digest, JobID: report.Intent.JobID, JobSHA256: report.Intent.JobSHA256, PinReason: "incident"}
	receipt, replay, err := s.applyRecordingControl(t.Context(), "control-owner", pin)
	g02Check(t, err)
	if replay || receipt.State.Revision != 2 || len(receipt.State.Pins) != 1 {
		t.Fatal("linked control receipts missing")
	}
	after := appAllRows(t, s.db)
	delete(after, "@schema")
	delete(after, "runtime_recording_control_events")
	delete(after, "runtime_recording_control_head")
	if !reflect.DeepEqual(report.Rows, after) {
		t.Fatal("policy/pin changed an original64 row")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	before := appAllRows(t, s.db)
	got, replay, err := s.applyRecordingControl(t.Context(), "control-owner", pin)
	if err != nil || !replay || !reflect.DeepEqual(got, receipt) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("reopened control replay changed history", err)
	}
	g02Check(t, s.close())
}

func loadOriginal64ControlFixture(t *testing.T, dir string) {
	t.Helper()
	base := "testdata/recording-control-original64"
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
	if stopEvidenceDigest(raw) != "78a815005b718a26339365230157cf472ce25f977c2b151d55fc8cf126b59129" || m.Schema != 64 || m.Source != "cc55707090061885a75fd144c2e2d6648d8d6292" || m.Helper != "49422d5cf3cba6e1595a6493e0681fe6465081f04c7adb542b6a17e6929c6719" || m.Inputs != "03e2b57dd21f05605a8a48cce79063cf8a2faf1018a40725afa47022ef55d98e" {
		t.Fatal("actual original64 writer provenance changed")
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
		t.Fatal("actual original64 archive changed")
	}
	z, err := gzip.NewReader(bytes.NewReader(packed))
	g02Check(t, err)
	body, err := io.ReadAll(io.LimitReader(z, 4<<20+1))
	g02Check(t, err)
	g02Check(t, z.Close())
	if len(body) > 4<<20 || stopEvidenceDigest(body) != m.SQL {
		t.Fatal("actual original64 SQL changed")
	}
	f, err := os.OpenFile(filepath.Join(dir, "runner.sqlite"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	g02Check(t, err)
	g02Check(t, f.Close())
	db := g02RawDB(t, dir)
	_, err = db.Exec(string(body))
	g02Check(t, err)
	g02Check(t, db.Close())
}

func TestRecordingControlOriginal64MigrationPreservesIntentAndPendingStart(t *testing.T) {
	dir := privateDir(t)
	loadOriginal64ControlFixture(t, dir)
	db := g02RawDB(t, dir)
	before := appAllRows(t, db)
	delete(before, "@schema")
	i, err := loadRecordingIntent(db, strings.Repeat("1", 32))
	g02Check(t, err)
	op, err := loadOperation(db, i.OperationID)
	g02Check(t, err)
	if i.Version != 1 || op.Kind != "start" || op.Status != "running" {
		t.Fatal("original64 uncertainty missing")
	}
	g02Check(t, db.Close())
	for pass := 0; pass < 3; pass++ {
		s, err := testOpenStore(dir)
		g02Check(t, err)
		after := rowsWithoutEmptyRecordingControl(appAllRows(t, s.db))
		delete(after, "@schema")
		if !reflect.DeepEqual(before, after) {
			t.Fatal("65 migration changed actual64 rows/DDL/access/uncertainty")
		}
		saved, err := loadRecordingIntent(s.db, i.JobID)
		g02Check(t, err)
		if saved != i {
			t.Fatal("original recording intent changed")
		}
		g02Check(t, s.close())
	}
}
func TestRecordingControlOriginal64MigrationRollsBackMissingOriginalIntent(t *testing.T) {
	dir := privateDir(t)
	loadOriginal64ControlFixture(t, dir)
	db := g02RawDB(t, dir)
	_, err := db.Exec("DELETE FROM runtime_recording_intents")
	g02Check(t, err)
	before := appAllRows(t, db)
	g02Check(t, db.Close())
	if s, err := testOpenStore(dir); err == nil {
		s.close()
		t.Fatal("broken actual64 recording ancestry admitted")
	}
	db = g02RawDB(t, dir)
	defer db.Close()
	if !reflect.DeepEqual(before, appAllRows(t, db)) {
		t.Fatal("failed65 migration rewrote actual64 evidence")
	}
}
