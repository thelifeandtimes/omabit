//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

var ErrRecordingDrainPending = errors.New("recording terminal evidence is not saved; preserve the container and inspect its exact job")

// RecordingWorkerStatus is saved evidence, not live worker/process health. It
// returns no raw frames or private exceptions. Prefix status is explicitly not a
// full-session integrity scan. Access control belongs to the authenticated runner.
type RecordingWorkerStatus struct {
	Version        int                      `json:"version"`
	JobID          string                   `json:"job_id"`
	JobSHA256      string                   `json:"job_sha256"`
	State          string                   `json:"state"`
	Worker         *RecordingWorkerIdentity `json:"worker,omitempty"`
	Allocation     *RecordingJobSession     `json:"allocation,omitempty"`
	Ready          *RecordingJobReady       `json:"ready,omitempty"`
	Result         *RecordingJobResult      `json:"result,omitempty"`
	SessionID      string                   `json:"session_id,omitempty"`
	SessionIDBasis string                   `json:"session_id_basis"`
	SessionStorage string                   `json:"session_storage"`
	Metadata       *recording.Metadata      `json:"metadata,omitempty"`
	WriterPresent  bool                     `json:"writer_present"`
	PrefixStatus   string                   `json:"prefix_status"`
	ReadAt         time.Time                `json:"read_at"`
	Basis          string                   `json:"basis"`
}

func ReadManagedRecordingStatus(ctx context.Context, id, digest string) (RecordingWorkerStatus, error) {
	if ctx.Err() != nil || !recordingJobHex(id, 32) || !recordingJobHex(digest, 64) {
		return RecordingWorkerStatus{}, ErrRecordingJobStorage
	}
	opts, err := recordingRootOptions(defaults(), false)
	if err != nil {
		return RecordingWorkerStatus{}, ErrRecordingJobStorage
	}
	return readRecordingWorkerStatus(ctx, opts, id, digest)
}

func readRecordingWorkerStatus(ctx context.Context, opts RecordingWorkerOptions, id, digest string) (RecordingWorkerStatus, error) {
	out := RecordingWorkerStatus{Version: 1, JobID: id, JobSHA256: digest, State: "job-saved", SessionIDBasis: "unknown", SessionStorage: "unknown", PrefixStatus: "not-scanned", ReadAt: time.Now().UTC(), Basis: "saved job/receipt and metadata prefix at read time; live health and full recording integrity are not certified"}
	if ctx.Err() != nil {
		return out, ErrRecordingJobStorage
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, opts.MaxJobs, opts.JobReserveFreeBytes)
	if err != nil {
		return out, ErrRecordingJobStorage
	}
	defer jobs.Close()
	job, err := jobs.Read(id, digest)
	if err != nil {
		return out, ErrRecordingJobStorage
	}
	if !recordingJobHasNoFiles(jobs, id, "claim.json") {
		worker, err := jobs.ReadWorker(id, digest)
		if err != nil {
			return out, ErrRecordingJobStorage
		}
		out.Worker, out.State = &worker, "worker-claimed"
	}
	if allocation, err := jobs.ReadSession(id, digest); err == nil {
		out.Allocation, out.State = &allocation, "session-allocated"
		out.SessionID, out.SessionIDBasis = allocation.SessionID, "allocation"
	} else if !errors.Is(err, ErrRecordingSessionUnknown) {
		return out, ErrRecordingJobStorage
	}
	if ready, err := jobs.ReadReady(id, digest); err == nil {
		out.Ready, out.State = &ready, "attachment-ready"
		if out.SessionID == "" {
			out.SessionID, out.SessionIDBasis = ready.SessionID, "saved-readiness"
		} else if out.SessionID != ready.SessionID {
			return out, ErrRecordingJobStorage
		}
	} else if !errors.Is(err, ErrRecordingJobNotReady) {
		return out, ErrRecordingJobStorage
	}
	if result, err := jobs.ReadResult(id, digest); err == nil {
		out.Result, out.State = &result, result.Code
		if result.Capture.SessionID != "" {
			if out.SessionID == "" {
				out.SessionID, out.SessionIDBasis = result.Capture.SessionID, "saved-result"
			} else if out.SessionID != result.Capture.SessionID {
				return out, ErrRecordingJobStorage
			}
		}
	} else if !errors.Is(err, ErrRecordingJobNotReady) {
		return out, ErrRecordingJobStorage
	}
	if out.Worker == nil && (out.Allocation != nil || out.Ready != nil || out.Result != nil) {
		return out, ErrRecordingJobStorage
	}
	if out.SessionID == "" {
		return out, nil
	}
	out.SessionStorage = "unavailable"
	root, err := openRecordingPrivateRoot(opts.RecordingsRoot)
	if err != nil {
		return out, nil
	}
	_, childErr := root.Lstat(out.SessionID)
	root.Close()
	if errors.Is(childErr, os.ErrNotExist) {
		out.SessionStorage = "header-not-found"
		return out, nil
	}
	if childErr != nil {
		return out, nil
	}
	s, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
	if err != nil {
		return out, nil // saved storage failure remains inspectable
	}
	defer s.Close()
	page, err := s.ReadPage(ctx, out.SessionID, nil, 1, recording.MaxFrameBytes)
	if errors.Is(err, os.ErrNotExist) {
		out.SessionStorage = "header-not-found"
		return out, nil
	}
	if err != nil {
		return out, nil
	}
	if !job.MatchesAdmission(page.Metadata) || out.Worker == nil || page.Metadata.Identity.HostBootID != out.Worker.HostBootID || page.Metadata.Identity.RecorderNamespaces != out.Worker.Namespaces {
		out.SessionStorage = "binding-changed"
		return out, nil
	}
	if out.Ready != nil {
		raw, err := page.Metadata.Bytes()
		if err != nil || recordingJobSHA(raw) != out.Ready.MetadataSHA256 || !page.Metadata.AttachedAt.Equal(out.Ready.AttachedAt) {
			out.SessionStorage = "binding-changed"
			return out, nil
		}
	}
	out.Metadata, out.WriterPresent, out.PrefixStatus = &page.Metadata, page.WriterPresent, page.Tail
	out.SessionStorage = "header-available"
	if page.Tail == "corrupt" || page.Tail == "incomplete" {
		out.SessionStorage = page.Tail
	}
	if ctx.Err() != nil {
		return out, ErrRecordingJobStorage
	}
	return out, nil
}

// AwaitManagedRecordingDrain reads saved outcome only. Success is a closed
// worker's saved successful capture/exit receipt, not permission to remove a
// container or proof of full raw integrity. The caller still verifies its native
// terminal identity and journal fence, and chooses explicit incomplete handling.
func AwaitManagedRecordingDrain(ctx context.Context, id, digest string) (RecordingJobResult, error) {
	if ctx.Err() != nil {
		return RecordingJobResult{}, ErrRecordingDrainPending
	}
	if !recordingJobHex(id, 32) || !recordingJobHex(digest, 64) {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	opts, err := recordingRootOptions(defaults(), false)
	if err != nil {
		return RecordingJobResult{}, ErrRecordingJobStorage
	}
	return awaitRecordingDrain(ctx, opts, id, digest, 100*time.Millisecond)
}

func awaitRecordingDrain(ctx context.Context, opts RecordingWorkerOptions, id, digest string, interval time.Duration) (RecordingJobResult, error) {
	if interval <= 0 || ctx.Err() != nil {
		return RecordingJobResult{}, ErrRecordingDrainPending
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if ctx.Err() != nil {
			return RecordingJobResult{}, ErrRecordingDrainPending
		}
		status, err := readRecordingWorkerStatus(ctx, opts, id, digest)
		if err != nil {
			if ctx.Err() != nil {
				return RecordingJobResult{}, ErrRecordingDrainPending
			}
			return RecordingJobResult{}, err
		}
		if status.Result != nil {
			result := *status.Result
			if result.Code == "capture-finished" && result.Capture.ClosedDurably && result.Capture.ExitSaved && status.SessionStorage == "header-available" && !status.WriterPresent {
				return result, nil
			}
			return result, recording.ErrCaptureIncomplete
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return RecordingJobResult{}, ErrRecordingDrainPending
		case <-timer.C:
		}
	}
}
