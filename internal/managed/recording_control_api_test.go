package managed

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestRecordingControlAPIExactActorLookupCASAndReplay(t *testing.T) {
	s, _, intent := recordingIntentFixture(t)
	_, _, err := s.admitRecordingIntent(intent)
	g02Check(t, err)
	r := &Server{s: s}
	call := func(path, actor, role string, in any) *httptest.ResponseRecorder {
		body, err := json.Marshal(in)
		g02Check(t, err)
		req := httptest.NewRequest("POST", path, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.serveRecordingControls(w, req, actor, role)
		return w
	}
	read := recording.ControlReadRequest{RunnerID: s.runnerID, IncludePrivateData: true}
	if w := call(protocol.RecordingControlStatusPath, "owner", "observer", read); w.Code != 403 {
		t.Fatal("observer read controls", w.Code)
	}
	read.IncludePrivateData = false
	if w := call(protocol.RecordingControlStatusPath, "owner", "operator", read); w.Code == 200 {
		t.Fatal("private consent omitted")
	}
	policy := recording.DefaultRetentionPolicy()
	policy.MaxJobs = 32
	in := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "configure-original", Kind: "configure", Policy: &policy}
	r.updateWorkers = 1
	if w := call(protocol.RecordingControlApplyPath, "owner", "operator", in); w.Code != 409 {
		t.Fatal("new control raced active launch", w.Code, w.Body)
	}
	r.updateWorkers = 0
	r.updateFence.Held = true
	if w := call(protocol.RecordingControlApplyPath, "owner", "operator", in); w.Code != 409 || !strings.Contains(w.Body.String(), "recording_control_quiesced") {
		t.Fatal("control bypassed activation fence", w.Code, w.Body)
	}
	r.updateFence.Held = false
	w := call(protocol.RecordingControlApplyPath, "owner", "operator", in)
	var first recording.ControlApplyResult
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &first))
	if w.Code != 200 || first.Replayed || first.Receipt.Validate() != nil {
		t.Fatal("invalid saved control", w.Code, w.Body)
	}
	before := appAllRows(t, s.db)
	r.recording = func(protocol.Operation) runtimequalify.ManagedRecordingLifecycle {
		return runtimequalify.ManagedRecordingLifecycle{}
	}
	r.recordingCapacity = func(context.Context, string) (recording.Capacity, error) {
		return recording.Capacity{}, errors.New("unattributed fixture storage")
	}
	read.IncludePrivateData = true
	statusReply := call(protocol.RecordingControlStatusPath, "owner", "operator", read)
	var status recording.ControlStatus
	g02Check(t, json.Unmarshal(statusReply.Body.Bytes(), &status))
	if statusReply.Code != 200 || status.CapacityStatus != "unknown" || status.Capacity != nil || status.State.Revision != 1 {
		t.Fatal("unknown capacity erased saved controls", statusReply.Code, statusReply.Body)
	}
	r.updateWorkers = 1
	w = call(protocol.RecordingControlApplyPath, "owner", "operator", in)
	var again recording.ControlApplyResult
	g02Check(t, json.Unmarshal(w.Body.Bytes(), &again))
	if w.Code != 200 || !again.Replayed || !reflect.DeepEqual(first.Receipt, again.Receipt) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("busy exact replay changed history", w.Code, w.Body)
	}
	lookup := recording.ControlLookupRequest{RunnerID: s.runnerID, RequestID: in.RequestID, IncludePrivateData: true}
	for _, actor := range []string{"owner", "other"} {
		w = call(protocol.RecordingControlLookupPath, actor, "operator", lookup)
		var saved recording.ControlLookupResult
		g02Check(t, json.Unmarshal(w.Body.Bytes(), &saved))
		if w.Code != 200 || saved.Found != (actor == "owner") || saved.Receipt != nil && !reflect.DeepEqual(*saved.Receipt, first.Receipt) {
			t.Fatal("lookup crossed original actor/request", w.Code, w.Body)
		}
	}
	r.updateWorkers = 0
	changed := in
	changed.RequestID = "new-stale-request"
	if w := call(protocol.RecordingControlApplyPath, "owner", "operator", changed); w.Code != 409 {
		t.Fatal("stale revision applied", w.Code, w.Body)
	}
	policy.AutoRetire = true
	changed.ExpectedRevision = first.Receipt.State.Revision
	changed.ExpectedDigest = first.Receipt.State.Digest
	if w := call(protocol.RecordingControlApplyPath, "owner", "operator", changed); w.Code != 501 {
		t.Fatal("unimplemented automatic removal accepted", w.Code, w.Body)
	}
	if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("lookup/refusal changed control history")
	}
}

func TestRecordingIntent2BindsControlHeadWithoutRewritingVersion1(t *testing.T) {
	s, dir, intent := recordingIntentFixture(t)
	intent.Version = 2
	_, _, err := s.admitRecordingIntent(intent)
	g02Check(t, err)
	g02Check(t, s.checkRecordingIntent(t.Context(), intent))
	policy := recording.DefaultRetentionPolicy()
	policy.MaxJobs = 32
	_, _, err = s.applyRecordingControl(t.Context(), "owner", recording.ControlRequest{RunnerID: s.runnerID, RequestID: "change", Kind: "configure", Policy: &policy})
	g02Check(t, err)
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	before := appAllRows(t, s.db)
	if s.checkRecordingIntent(t.Context(), intent) == nil {
		t.Fatal("old control head retained fresh Start authority")
	}
	saved, replay, err := s.admitRecordingIntent(intent)
	if err != nil || !replay || saved != intent || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("historical intent2 replay rewritten", err)
	}
}
