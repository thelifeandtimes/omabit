package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type recordingExportFixtureBackend struct {
	fakeBackend
	manifest recording.ExportManifestResult
	reads    int
	wrong    bool
}

type recordingExportDeadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (w *recordingExportDeadlineWriter) SetWriteDeadline(at time.Time) error {
	w.deadline = at
	return nil
}

func (b *recordingExportFixtureBackend) RecordingExportManifest(context.Context, recording.ExportRequest) (recording.ExportManifestResult, error) {
	b.reads++
	out := b.manifest
	if b.wrong {
		out.Session.InstanceID = "other"
	}
	return out, nil
}
func (b *recordingExportFixtureBackend) RecordingExportPart(_ context.Context, in recording.ExportPartRequest) (recording.ExportPart, error) {
	b.reads++
	out := recording.ExportPart{Version: 1, Request: in, Data: []byte("private"), ReadAt: time.Now().UTC()}
	if b.wrong {
		out.Request.InstanceID = "other"
	}
	return out, nil
}

func TestRecordingExportsRequireExactPrivateJournalAndPreserveAllRows(t *testing.T) {
	s, _, intent := recordingIntentFixture(t)
	_, _, err := s.admitRecordingIntent(intent)
	g02Check(t, err)
	g02Check(t, s.reconcileInterrupted())
	raw, err := os.ReadFile("../recording/testdata/original-metadata1/metadata.json")
	g02Check(t, err)
	m, err := recording.ReadMetadata(raw)
	g02Check(t, err)
	d := recordingDescriptor(intent)
	m.RunnerID, m.InstanceID, m.Mode, m.ConfigurationRevision = d.RunnerID, d.InstanceID, d.Mode, d.ConfigurationRevision
	m.ContainerID, m.OperationID, m.RequestID = d.ContainerID, d.OperationID, d.RequestID
	raw, err = m.Bytes()
	g02Check(t, err)
	hash := func(p []byte) string { h := sha256.Sum256(p); return hex.EncodeToString(h[:]) }
	manifest := recording.ExportManifest{Version: 1, Metadata: m, MetadataSHA256: hash(raw), Representation: "original", ChunkBytes: recording.ExportChunkBytes, Files: []recording.ExportFile{{Name: "metadata.json", Bytes: int64(len(raw)), SHA256: hash(raw), Chunks: []string{hash(raw)}}, {Name: "frames.obr", Bytes: 7, SHA256: hash([]byte("private")), Chunks: []string{hash([]byte("private"))}}}, Coverage: recording.ExportCoverage{Tail: "corrupt"}, RetainedFilesComplete: true, ReadAt: time.Now().UTC(), Basis: "synthetic encoded-file descriptor only; native bytes are not qualified by this API guard fixture"}
	digest, err := manifest.Digest()
	g02Check(t, err)
	b := &recordingExportFixtureBackend{manifest: recording.ExportManifestResult{Session: d, Status: recording.SavedStatus{State: "capture-incomplete", ResultCode: "capture-incomplete", SessionStorage: "header-available", SessionID: m.SessionID}, Manifest: manifest, ManifestSHA256: digest}}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	in := recording.ExportRequest{RunnerID: d.RunnerID, InstanceID: d.InstanceID, JobID: d.JobID, JobSHA256: d.JobSHA256, Representation: "original", IncludePrivateData: true}
	part := recording.ExportPartRequest{ExportRequest: in, SessionID: m.SessionID, MetadataSHA256: manifest.MetadataSHA256, FileName: "frames.obr", FileBytes: 7, ChunkSHA256: hash([]byte("private"))}
	before := appAllRows(t, s.db)
	for _, path := range []string{protocol.RecordingExportManifestPath, protocol.RecordingExportPartPath} {
		for _, variant := range []string{"ok", "observer", "no-consent", "foreign-instance", "foreign-job", "foreign-runner", "redacted", "wrong-reply"} {
			request := part
			role := "operator"
			b.wrong = false
			switch variant {
			case "observer":
				role = "observer"
			case "no-consent":
				request.IncludePrivateData = false
			case "foreign-instance":
				request.InstanceID = "other"
			case "foreign-job":
				request.JobSHA256 = strings.Repeat("e", 64)
			case "foreign-runner":
				request.RunnerID = "other"
			case "redacted":
				request.Representation = "redacted"
			case "wrong-reply":
				b.wrong = true
			}
			var value any = request
			if path == protocol.RecordingExportManifestPath {
				value = request.ExportRequest
			}
			raw, err := json.Marshal(value)
			g02Check(t, err)
			q := httptest.NewRequest("POST", path, strings.NewReader(string(raw)))
			q.Header.Set("Content-Type", "application/json")
			q.Header.Set(protocol.Header, "1")
			w := &recordingExportDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
			old := b.reads
			r.serveRecordingExport(w, q, role)
			if variant == "ok" {
				if w.Code != 200 || b.reads != old+1 {
					t.Fatal(path, variant, w.Code, w.Body)
				}
				if time.Until(w.deadline) < protocol.RecordingExportTimeout {
					t.Fatal("export inherited ordinary five-second server write timeout")
				}
			} else if w.Code == 200 || variant != "wrong-reply" && b.reads != old {
				t.Fatal("unauthorized/rebound export", path, variant, w.Code, w.Body)
			}
		}
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("export read changed original history")
	}
}
