package managed

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingControlsCASOriginalHistoryAndExactReplay(t *testing.T) {
	s, dir, i := recordingIntentFixture(t)
	_, _, err := s.admitRecordingIntent(i)
	g02Check(t, err)
	state, err := s.recordingControls(t.Context())
	g02Check(t, err)
	if state.Revision != 0 || state.Digest != "" || !reflect.DeepEqual(state.Policy, recording.DefaultRetentionPolicy()) || len(state.Pins) != 0 {
		t.Fatal("new controls invented policy history")
	}
	before := rowsWithoutEmptyRecordingControl(appAllRows(t, s.db))
	delete(before, "@schema")
	policy := state.Policy
	policy.MaxJobs = 32
	policy.Instances = []recording.InstanceRetention{{InstanceID: i.InstanceID, MaxRawBytes: 128 << 20, MaxAgeSeconds: 24 * 60 * 60}}
	configure := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "configure", Kind: "configure", Policy: &policy}
	first, replay, err := s.applyRecordingControl(t.Context(), "owner", configure)
	g02Check(t, err)
	if replay || first.State.Revision != 1 || first.State.Policy.MaxJobs != 32 {
		t.Fatal("policy CAS did not commit once")
	}
	pin := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "pin", Kind: "pin", ExpectedRevision: first.State.Revision, ExpectedDigest: first.State.Digest, JobID: i.JobID, JobSHA256: i.JobSHA256, PinReason: "incident"}
	pinned, replay, err := s.applyRecordingControl(t.Context(), "owner", pin)
	g02Check(t, err)
	if replay || len(pinned.State.Pins) != 1 || pinned.State.Pins[0].Reason != "incident" {
		t.Fatal("exact incident pin missing")
	}
	got, replay, err := s.applyRecordingControl(t.Context(), "owner", configure)
	g02Check(t, err)
	if !replay || !reflect.DeepEqual(got, first) {
		t.Fatal("later policy state rewrote original replay")
	}
	bad := pin
	bad.RequestID = "stale"
	original := appAllRows(t, s.db)
	if _, _, err := s.applyRecordingControl(t.Context(), "owner", bad); err == nil || !reflect.DeepEqual(original, appAllRows(t, s.db)) {
		t.Fatal("stale head changed controls")
	}
	unpin := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "unpin", Kind: "unpin", ExpectedRevision: pinned.State.Revision, ExpectedDigest: pinned.State.Digest, JobID: i.JobID, JobSHA256: i.JobSHA256}
	cleared, _, err := s.applyRecordingControl(t.Context(), "owner", unpin)
	g02Check(t, err)
	if len(cleared.State.Pins) != 0 || len(pinned.State.Pins) != 1 {
		t.Fatal("unpin changed old receipt or failed to clear exact pin")
	}
	after := appAllRows(t, s.db)
	delete(after, "@schema")
	delete(after, "runtime_recording_control_events")
	delete(after, "runtime_recording_control_head")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("controls changed original operation, runtime, recording or app rows")
	}
	g02Check(t, s.close())
	s, err = testOpenStore(dir)
	g02Check(t, err)
	defer s.close()
	got, replay, err = s.applyRecordingControl(t.Context(), "owner", pin)
	if err != nil || !replay || !reflect.DeepEqual(got, pinned) {
		t.Fatal("reopen relabeled old pin", err)
	}
}

func TestRecordingControlRefusesCorruptionOrHeadFailureWithoutRepair(t *testing.T) {
	for _, scenario := range []string{"missing-tail", "changed-body", "missing-head", "oversized", "head-write-refused", "foreign-job"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, i := recordingIntentFixture(t)
			_, _, err := s.admitRecordingIntent(i)
			g02Check(t, err)
			policy := recording.DefaultRetentionPolicy()
			request := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "initial", Kind: "configure", Policy: &policy}
			receipt, _, err := s.applyRecordingControl(t.Context(), "owner", request)
			g02Check(t, err)
			switch scenario {
			case "missing-tail":
				_, err = s.db.Exec("DELETE FROM runtime_recording_control_events")
			case "changed-body":
				_, err = s.db.Exec("UPDATE runtime_recording_control_events SET body=body||' '")
			case "missing-head":
				_, err = s.db.Exec("DELETE FROM runtime_recording_control_head")
			case "oversized":
				_, err = s.db.Exec("UPDATE runtime_recording_control_events SET body=?", strings.Repeat("x", maxRecordingControlBytes+1))
			case "head-write-refused":
				_, err = s.db.Exec("CREATE TEMP TRIGGER refuse_control_head BEFORE UPDATE ON runtime_recording_control_head BEGIN SELECT RAISE(ABORT,'injected control head refusal'); END")
			}
			g02Check(t, err)
			before := appAllRows(t, s.db)
			if scenario == "head-write-refused" || scenario == "foreign-job" {
				next := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "pin-next", Kind: "pin", ExpectedRevision: receipt.State.Revision, ExpectedDigest: receipt.State.Digest, JobID: i.JobID, JobSHA256: i.JobSHA256, PinReason: "owner"}
				if scenario == "foreign-job" {
					next.JobID = strings.Repeat("f", 32)
				}
				if _, _, err := s.applyRecordingControl(t.Context(), "owner", next); err == nil {
					t.Fatal("invalid commit accepted")
				}
			} else if _, err := s.recordingControls(t.Context()); err == nil {
				t.Fatal("corrupt history accepted")
			}
			if !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("refused history was repaired or partially committed")
			}
		})
	}
}

func TestRecordingControlConcurrentHeadAdmitsOneAndCancelDoesNotWrite(t *testing.T) {
	s, _, _ := recordingIntentFixture(t)
	p := recording.DefaultRetentionPolicy()
	r := recording.ControlRequest{RunnerID: s.runnerID, RequestID: "one", Kind: "configure", Policy: &p}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{"one", "two"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			in := r
			in.RequestID = id
			_, _, err := s.applyRecordingControl(t.Context(), "owner", in)
			results <- err
		}(id)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("concurrent stale policy admissions", success)
	}
	before := appAllRows(t, s.db)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := s.applyRecordingControl(ctx, "owner", r); err == nil || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("cancelled request wrote controls")
	}
}
