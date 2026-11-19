package managed

import (
	"encoding/json"
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
