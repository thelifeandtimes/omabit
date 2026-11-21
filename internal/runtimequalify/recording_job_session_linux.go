//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"time"
)

var ErrRecordingSessionUnknown = errors.New("recording job has no saved session allocation; coverage remains unknown")

// RecordingJobSession records allocation only, before capture storage/attachment.
// It is not a recording header, successful capture, readiness or process start.
// Old jobs without it remain unknown; lookups never fabricate or backfill one.
type RecordingJobSession struct {
	Version     int       `json:"version"`
	JobID       string    `json:"job_id"`
	JobSHA256   string    `json:"job_sha256"`
	ClaimSHA256 string    `json:"claim_sha256"`
	SessionID   string    `json:"session_id"`
	AllocatedAt time.Time `json:"allocated_at"`
}

func (s RecordingJobSession) valid() bool {
	return s.Version == 1 && recordingJobHex(s.JobID, 32) && recordingJobHex(s.JobSHA256, 64) && recordingJobHex(s.ClaimSHA256, 64) && recordingJobHex(s.SessionID, 32) && !s.AllocatedAt.Before(time.Unix(0, 1)) && !s.AllocatedAt.After(time.Unix(0, math.MaxInt64))
}

func (l *RecordingJobLease) AllocateSession(id string, at time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	allocation := RecordingJobSession{Version: 1, JobID: l.job.ID, JobSHA256: l.digest, ClaimSHA256: l.claimSHA, SessionID: id, AllocatedAt: at.UTC()}
	if l.closed || !allocation.valid() || !recordingJobBound(l.root, "worker.lock", l.lock, 0) {
		return ErrRecordingJobStorage
	}
	if _, err := l.store.read(l.root, l.job.ID, l.digest); err != nil {
		return err
	}
	claim, err := recordingJobRead(l.root, "claim.json", 4096)
	if err != nil || recordingJobSHA(claim) != l.claimSHA {
		return ErrRecordingJobStorage
	}
	if err := l.store.admit(4096); err != nil {
		return err
	}
	raw, _ := json.Marshal(allocation)
	return recordingJobWrite(l.root, "session.json", raw)
}

func (s *RecordingJobs) ReadSession(id, digest string) (RecordingJobSession, error) {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return RecordingJobSession{}, err
	}
	defer root.Close()
	if _, err := s.read(root, id, digest); err != nil {
		return RecordingJobSession{}, err
	}
	raw, err := recordingJobRead(root, "session.json", 4096)
	if errors.Is(err, os.ErrNotExist) {
		return RecordingJobSession{}, ErrRecordingSessionUnknown
	}
	var allocation RecordingJobSession
	if err != nil || json.Unmarshal(raw, &allocation) != nil || !allocation.valid() || allocation.JobID != id || allocation.JobSHA256 != digest {
		return RecordingJobSession{}, ErrRecordingJobStorage
	}
	canonical, _ := json.Marshal(allocation)
	if !bytes.Equal(raw, canonical) {
		return RecordingJobSession{}, ErrRecordingJobStorage
	}
	if _, err := s.ReadWorker(id, digest); err != nil {
		return RecordingJobSession{}, ErrRecordingJobStorage
	}
	claim, err := recordingJobRead(root, "claim.json", 4096)
	if err != nil || recordingJobSHA(claim) != allocation.ClaimSHA256 {
		return RecordingJobSession{}, ErrRecordingJobStorage
	}
	return allocation, nil
}

func (s *RecordingJobs) checkSession(id, digest, session string) error {
	allocation, err := s.ReadSession(id, digest)
	if errors.Is(err, ErrRecordingSessionUnknown) {
		return nil // historical omission is not current readiness/authority
	}
	if err != nil || session != "" && allocation.SessionID != session {
		return ErrRecordingJobStorage
	}
	return nil
}
