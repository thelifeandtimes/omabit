package managed

import (
	"context"
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

type recordingReadFixtureBackend struct {
	fakeBackend
	selected recording.SessionDescriptor
	reads    int
	wrong    bool
}

func (b *recordingReadFixtureBackend) ReadRecording(_ context.Context, in recording.PageRequest) (recording.ReadWindow, error) {
	b.reads++
	descriptor := b.selected
	if b.wrong {
		descriptor.InstanceID = "other"
	}
	return recording.ReadWindow{Version: 1, Session: descriptor, Status: recording.SavedStatus{State: "job-saved", SessionIDBasis: "unknown", SessionStorage: "unknown"}, Representation: in.Representation, From: in.Cursor, Tail: "not-scanned", ReadAt: time.Now().UTC(), Basis: "inert saved fixture"}, nil
}

func TestRecordingReadsRemainHistoricalPrivateAndBoundToOriginalJournal(t *testing.T) {
	s, _, intent := recordingIntentFixture(t)
	_, _, err := s.admitRecordingIntent(intent)
	g02Check(t, err)
	g02Check(t, s.reconcileInterrupted())
	s.mode = DevelopmentMode
	b := &recordingReadFixtureBackend{selected: recordingDescriptor(intent)}
	r := &Server{s: s, b: b, uid: uint32(os.Geteuid())}
	catalog := recording.CatalogRequest{RunnerID: s.runnerID, InstanceID: intent.InstanceID, IncludePrivateData: true}
	page := recording.PageRequest{RunnerID: s.runnerID, InstanceID: intent.InstanceID, JobID: intent.JobID, JobSHA256: intent.JobSHA256, IncludePrivateData: true, Representation: "redacted"}
	before := appAllRows(t, s.db)
	call := func(path string, body any, role string) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		g02Check(t, err)
		req := httptest.NewRequest("POST", path, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(protocol.Header, "1")
		w := httptest.NewRecorder()
		r.serveRecordings(w, req, role)
		return w
	}
	w := call(protocol.RecordingCatalogPath, catalog, "operator")
	var out recording.Catalog
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	if w.Code != 200 || len(out.Sessions) != 1 || !reflect.DeepEqual(out.Sessions[0], b.selected) || b.reads != 0 {
		t.Fatal("historical catalog changed or probed native/files", w.Code, w.Body)
	}
	catalog.AfterJobID = intent.JobID
	w = call(protocol.RecordingCatalogPath, catalog, "operator")
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &out))
	if w.Code != 200 || len(out.Sessions) != 0 {
		t.Fatal("catalog cursor replay duplicated history", w.Body)
	}
	w = call(protocol.RecordingPagePath, page, "operator")
	if w.Code != 200 || b.reads != 1 {
		t.Fatal("historical selection refused", w.Code, w.Body)
	}
	for _, scenario := range []string{"observer", "no-consent", "foreign-digest", "foreign-instance", "wrong-reply"} {
		t.Run(scenario, func(t *testing.T) {
			selected := page
			role := "operator"
			b.wrong = false
			switch scenario {
			case "observer":
				role = "observer"
			case "no-consent":
				selected.IncludePrivateData = false
			case "foreign-digest":
				selected.JobSHA256 = strings.Repeat("f", 64)
			case "foreign-instance":
				selected.InstanceID = "other"
			case "wrong-reply":
				b.wrong = true
			}
			reads := b.reads
			w := call(protocol.RecordingPagePath, selected, role)
			if w.Code == 200 || scenario != "wrong-reply" && b.reads != reads {
				t.Fatal("unauthorized or rebound read reached/exposed backend", w.Code, w.Body)
			}
		})
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("history read changed original receipt/journal")
	}
}
