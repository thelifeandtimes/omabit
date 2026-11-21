//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

// RecordingJobResult contains finite worker status, never raw output or private
// exception text. Finished means this capture's successful stream/exit/fsync
// path, not proof of bytes before attachment or full native bootstrap coverage.
type RecordingJobResult struct {
	Version    int                     `json:"version"`
	JobID      string                  `json:"job_id"`
	JobSHA256  string                  `json:"job_sha256"`
	Code       string                  `json:"code"`
	FinishedAt time.Time               `json:"finished_at"`
	Capture    recording.CaptureResult `json:"capture"`
}

func (r RecordingJobResult) valid() bool {
	if r.Version != 1 || !recordingJobHex(r.JobID, 32) || !recordingJobHex(r.JobSHA256, 64) || r.FinishedAt.Before(time.Unix(0, 1)) || r.FinishedAt.After(time.Unix(0, math.MaxInt64)) {
		return false
	}
	c := r.Capture
	if c.SessionID != "" && !recordingJobHex(c.SessionID, 32) || c.SessionID == "" && (c.AttachmentAcknowledged || c.ReadyAcknowledged || c.ExitSaved || c.GapSaved || c.ClosedDurably || c.Stats.OriginalBytes != 0) || c.ReadyAcknowledged && !c.AttachmentAcknowledged || c.GapSaved && c.GapReason == "" {
		return false
	}
	if c.GapReason != "" && (recording.GapEvent{Version: 1, Reason: c.GapReason}).Validate() != nil {
		return false
	}
	switch r.Code {
	case "capture-finished":
		return c.AttachmentAcknowledged && c.ReadyAcknowledged && c.ExitSaved && c.ClosedDurably && c.GapReason == "" && !c.GapSaved && c.Stats.AtFrameBoundary && c.Stats.KnownUnrecordedBytes == 0
	case "admission-refused", "storage-unavailable":
		return !c.AttachmentAcknowledged && !c.ReadyAcknowledged && !c.ExitSaved
	case "capture-incomplete":
		return true
	}
	return false
}

func (l *RecordingJobLease) RecordResult(code string, capture recording.CaptureResult, at time.Time) (RecordingJobResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := RecordingJobResult{Version: 1, JobID: l.job.ID, JobSHA256: l.digest, Code: code, FinishedAt: at.UTC(), Capture: capture}
	if l.closed || !out.valid() || !recordingJobBound(l.root, "worker.lock", l.lock, 0) {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	if _, err := l.store.read(l.root, l.job.ID, l.digest); err != nil {
		return RecordingJobResult{}, err
	}
	claim, err := recordingJobRead(l.root, "claim.json", 4096)
	if err != nil || recordingJobSHA(claim) != l.claimSHA {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	if capture.ReadyAcknowledged {
		ready, err := l.store.ReadReady(l.job.ID, l.digest)
		if err != nil || ready.SessionID != capture.SessionID {
			return RecordingJobResult{}, ErrRecordingJobStorage
		}
	}
	if l.store.checkSession(l.job.ID, l.digest, capture.SessionID) != nil {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	if err = l.store.admit(8192); err != nil {
		return RecordingJobResult{}, err
	}
	data, _ := json.Marshal(out)
	if err = recordingJobWrite(l.root, "result.json", data); err != nil {
		return RecordingJobResult{}, err
	}
	return out, nil
}

// ReadResult is saved lookup only. Missing/partial status is not a successful
// capture, an exited process, or permission to replay an interrupted job.
func (s *RecordingJobs) ReadResult(id, digest string) (RecordingJobResult, error) {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return RecordingJobResult{}, err
	}
	defer root.Close()
	if _, err = s.read(root, id, digest); err != nil {
		return RecordingJobResult{}, err
	}
	data, err := recordingJobRead(root, "result.json", 4096)
	if errors.Is(err, os.ErrNotExist) {
		return RecordingJobResult{}, ErrRecordingJobNotReady
	}
	if err != nil {
		return RecordingJobResult{}, err
	}
	var result RecordingJobResult
	if json.Unmarshal(data, &result) != nil || !result.valid() || result.JobID != id || result.JobSHA256 != digest {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	canonical, _ := json.Marshal(result)
	if !bytes.Equal(canonical, data) {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	if result.Capture.ReadyAcknowledged {
		ready, err := s.ReadReady(id, digest)
		if err != nil || ready.SessionID != result.Capture.SessionID {
			return RecordingJobResult{}, ErrRecordingJobStorage
		}
	}
	if s.checkSession(id, digest, result.Capture.SessionID) != nil {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	return result, nil
}
