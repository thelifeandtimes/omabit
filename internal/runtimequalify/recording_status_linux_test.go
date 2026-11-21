//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingStatusRetainsSavedOutcomeWithoutRawOutputOrLiveHealthClaims(t *testing.T) {
	f, opts, job, digest, worker, selected := recordingWorkerFixture(t)
	status, err := readRecordingWorkerStatus(t.Context(), opts, job.ID, digest)
	if err != nil || status.State != "job-saved" || status.Worker != nil || status.SessionIDBasis != "unknown" {
		t.Fatal("new job invented worker/capture state", err)
	}
	payload := bytes.Repeat([]byte("PRIVATE-RAW authorization 0wdiagnostic.atom\n"), 1000)
	stream := &recordingTestStream{Reader: bytes.NewReader(recordingWorkerPacket(payload))}
	out, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, func(RecordingJobReady) error {
		s := mapAt(selected.doc, "State")
		s["Status"], s["Running"], s["Pid"] = "exited", false, 0
		s["StartedAt"], s["FinishedAt"] = "2026-11-18T00:00:00Z", "2026-11-18T00:00:01Z"
		s["ExitCode"], s["OOMKilled"] = 0, false
		return nil
	}, f.d, func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return stream, nil })
	if err != nil || out.Code != "capture-finished" {
		t.Fatal("source capture failed", err)
	}
	status, err = readRecordingWorkerStatus(t.Context(), opts, job.ID, digest)
	if err != nil || status.State != "capture-finished" || status.Result == nil || !reflect.DeepEqual(*status.Result, out) || status.Allocation == nil || status.SessionIDBasis != "allocation" || status.SessionStorage != "header-available" || status.WriterPresent || status.Metadata == nil || !strings.Contains(status.Basis, "full recording integrity are not certified") {
		t.Fatal("saved status lost provenance or invented live/full health", status, err)
	}
	raw, _ := json.Marshal(status)
	if bytes.Contains(raw, []byte("PRIVATE-RAW")) || bytes.Contains(raw, []byte("0wdiagnostic.atom")) {
		t.Fatal("private raw output entered status")
	}
	saved, err := awaitRecordingDrain(t.Context(), opts, job.ID, digest, time.Millisecond)
	if err != nil || !reflect.DeepEqual(saved, out) {
		t.Fatal("closed saved capture/exit not observable", err)
	}
	// A missing path can be moved data. Retain it and report only the observed
	// absence; never infer disposal or successful integrity from its receipt.
	path := filepath.Join(opts.RecordingsRoot, out.Capture.SessionID)
	if err := os.Rename(path, path+"-retained"); err != nil {
		t.Fatal(err)
	}
	status, err = readRecordingWorkerStatus(t.Context(), opts, job.ID, digest)
	if err != nil || status.Result == nil || status.SessionStorage != "header-not-found" {
		t.Fatal("missing header erased receipt or invented non-creation", err)
	}
	if _, err := awaitRecordingDrain(t.Context(), opts, job.ID, digest, time.Millisecond); !errors.Is(err, recording.ErrCaptureIncomplete) {
		t.Fatal("saved receipt certified missing raw storage", err)
	}
	if _, err := os.Stat(filepath.Join(path+"-retained", "frames.obr")); err != nil {
		t.Fatal("lookup removed retained original", err)
	}
}

func TestRecordingStatusOldReadinessIsHistoricalAndDrainTimeoutPerformsNoEffects(t *testing.T) {
	s, root, job, m, worker := recordingJobsFixture(t, 8)
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Ready(m); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	recordings := t.TempDir()
	os.Chmod(recordings, 0700)
	opts := RecordingWorkerOptions{JobsRoot: root, RecordingsRoot: recordings, MaxJobs: 8, JobReserveFreeBytes: 1 << 20}
	status, err := readRecordingWorkerStatus(t.Context(), opts, job.ID, digest)
	if err != nil || status.State != "attachment-ready" || status.Allocation != nil || status.SessionIDBasis != "saved-readiness" || status.SessionID != m.SessionID || status.Result != nil || status.SessionStorage != "header-not-found" {
		t.Fatal("old saved readiness gained allocation/capture facts", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if _, err := awaitRecordingDrain(ctx, opts, job.ID, digest, time.Millisecond); err != ErrRecordingDrainPending {
		t.Fatal("pending receipt became terminal success", err)
	}
	if _, err := s.ReadSession(job.ID, digest); !errors.Is(err, ErrRecordingSessionUnknown) {
		t.Fatal("status/drain backfilled history", err)
	}
	if _, err := s.ReadResult(job.ID, digest); !errors.Is(err, ErrRecordingJobNotReady) {
		t.Fatal("status/drain invented a result", err)
	}
}

func TestRecordingStatusIncompleteCaptureIsNotSuccessfulDrain(t *testing.T) {
	f, opts, job, digest, worker, _ := recordingWorkerFixture(t)
	stream := &recordingTestStream{Reader: bytes.NewReader(recordingWorkerPacket([]byte("raw final bytes\n")))}
	out, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, func(RecordingJobReady) error { return nil }, f.d,
		func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) { return stream, nil })
	if err == nil || out.Code != "capture-incomplete" || out.Capture.ExitSaved {
		t.Fatal("EOF fixture became a native exit", err)
	}
	status, err := readRecordingWorkerStatus(t.Context(), opts, job.ID, digest)
	if err != nil || status.Result == nil || status.State != "capture-incomplete" || status.WriterPresent {
		t.Fatal("incomplete saved evidence lost", err)
	}
	got, err := awaitRecordingDrain(t.Context(), opts, job.ID, digest, time.Millisecond)
	if !errors.Is(err, recording.ErrCaptureIncomplete) || !reflect.DeepEqual(got, out) {
		t.Fatal("incomplete result certified successful drain", err)
	}
}
