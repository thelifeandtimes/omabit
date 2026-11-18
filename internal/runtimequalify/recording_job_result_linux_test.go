//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingJobResultIsFiniteImmutableAndLookupOnly(t *testing.T) {
	s, dir, job, _, worker := recordingJobsFixture(t, 8)
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lease.RecordResult("capture-finished", recording.CaptureResult{}, time.Now()); !errors.Is(err, ErrRecordingJobStorage) {
		t.Fatal("empty capture became finished", err)
	}
	if _, err = lease.RecordResult("PRIVATE-DIAGNOSTIC", recording.CaptureResult{}, time.Now()); !errors.Is(err, ErrRecordingJobStorage) {
		t.Fatal("arbitrary diagnostic entered result", err)
	}
	out, err := lease.RecordResult("admission-refused", recording.CaptureResult{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, job.ID, "result.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lease.RecordResult("capture-incomplete", recording.CaptureResult{}, time.Now()); err == nil {
		t.Fatal("original outcome was overwritten")
	}
	lease.Close()
	for range 3 {
		saved, e := s.ReadResult(job.ID, digest)
		if e != nil || saved.Code != out.Code || saved.JobSHA256 != out.JobSHA256 {
			t.Fatal("saved status required live worker or changed binding", e)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("lookup/repeated result changed original", err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(before, &fields); err != nil {
		t.Fatal(err)
	}
	fields["private_error"] = json.RawMessage(`"PRIVATE-DIAGNOSTIC"`)
	bad, _ := json.Marshal(fields)
	if err = os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadResult(job.ID, digest); !errors.Is(err, ErrRecordingJobStorage) {
		t.Fatal("unknown result fields were accepted", err)
	}
	if data, _ := os.ReadFile(path); !bytes.Equal(data, bad) {
		t.Fatal("malformed result repaired by lookup")
	}
}

func TestRecordingJobLegacyClaimLeavesProcessStartAndInvocationUnknown(t *testing.T) {
	// Synthetic old-format shape only, not an actual-original-writer gate.
	s, dir, job, m, worker := recordingJobsFixture(t, 8)
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	if err = lease.Ready(m); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	worker.StartTicks = 0
	worker.InvocationID = ""
	old, _ := json.Marshal(recordingJobClaim{Version: 1, JobID: job.ID, JobSHA256: digest, Worker: worker})
	path := filepath.Join(dir, job.ID, "claim.json")
	if err = os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadReady(job.ID, digest); err != nil {
		t.Fatal("old claim was relabeled or discarded", err)
	}
	if worker.valid() {
		t.Fatal("old unknown start time became live-worker authority")
	}
	if _, err = s.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJob) {
		t.Fatal("missing start identity could claim anew", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(old, after) || bytes.Contains(after, []byte("start_ticks")) || bytes.Contains(after, []byte("invocation_id")) {
		t.Fatal("legacy lookup invented worker identity")
	}
	worker.StartTicks = 1
	worker.InvocationID = strings.Repeat("4", 32)
	if _, err = s.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJobClaimed) {
		t.Fatal("legacy claim was replayed under new identity", err)
	}
}
