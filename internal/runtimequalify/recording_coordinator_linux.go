//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"errors"
	"os"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

// SubmitRecordingWorker requires the authenticated manager to have committed
// this exact job and launch intent, under its ordinary lifecycle guard. Fence
// checks that original operation and exact current journal state; it is trusted
// local code, never an RPC callback. Saved intent replay cannot call this method.
// The method submits once and checks durable/live/native readiness, but sends no
// container Start/Stop/signal. Normal managed launch does not call it yet.
func SubmitRecordingWorker(ctx context.Context, job ManagedRecordingJob, fence func(context.Context) error) (RecordingJobReady, error) {
	if job.Validate() != nil || fence == nil || ctx.Err() != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	opts, err := recordingServiceOptions(defaults())
	if err != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	return submitRecordingWorker(ctx, opts, job, fence, submitRecordingWorkerCommand, ObserveRecordingSupervisor, func(ctx context.Context, job ManagedRecordingJob) error {
		actual, _, err := PrepareManagedRecording(ctx, job.State, job.Operation, func(recording.Metadata) error { return nil })
		if err != nil || !job.MatchesAdmission(actual) {
			return ErrRecordingWorker
		}
		return nil
	})
}

func recordingJobHasNoFiles(s *RecordingJobs, id string, names ...string) bool {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return false
	}
	defer root.Close()
	for _, name := range names {
		if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	return recordingJobChildBound(s.root, id, root)
}

func recordingReadyMetadata(ctx context.Context, opts RecordingWorkerOptions, job ManagedRecordingJob, ready RecordingJobReady, worker RecordingWorkerIdentity) error {
	s, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
	if err != nil {
		return ErrRecordingWorker
	}
	defer s.Close()
	page, err := s.ReadPage(ctx, ready.SessionID, nil, 1, recording.MaxFrameBytes)
	if err != nil || !page.WriterPresent || page.Tail != "snapshot-eof" || len(page.Frames) != 0 || !job.MatchesAdmission(page.Metadata) || page.Metadata.Identity.HostBootID != worker.HostBootID || page.Metadata.Identity.RecorderNamespaces != worker.Namespaces || page.Metadata.SessionID != ready.SessionID || !page.Metadata.AttachedAt.Equal(ready.AttachedAt) {
		return ErrRecordingWorker
	}
	raw, err := page.Metadata.Bytes()
	if err != nil || recordingJobSHA(raw) != ready.MetadataSHA256 {
		return ErrRecordingWorker
	}
	// A newly created, never-started container has no output yet. Any frame,
	// including a gap/exit, or a closed writer refuses this pre-Start gate.
	return nil
}

func submitRecordingWorker(ctx context.Context, opts RecordingWorkerOptions, job ManagedRecordingJob, fence func(context.Context) error, submit func(context.Context, ManagedRecordingJob) error, observe func(context.Context, ManagedRecordingJob, RecordingWorkerIdentity) (RecordingSupervisorState, error), native func(context.Context, ManagedRecordingJob) error) (RecordingJobReady, error) {
	if ctx.Err() != nil || fence == nil || submit == nil || observe == nil || native == nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	raw, err := job.Bytes()
	if err != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	job, err = ReadManagedRecordingJob(raw) // pin independent nested state
	if err != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	digest := recordingJobSHA(raw)
	jobs, err := OpenRecordingJobs(opts.JobsRoot, opts.MaxJobs, opts.JobReserveFreeBytes)
	if err != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	defer jobs.Close()
	saved, err := jobs.Read(job.ID, digest)
	if err != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	savedBytes, err := saved.Bytes()
	if err != nil || !bytes.Equal(raw, savedBytes) || !recordingJobHasNoFiles(jobs, job.ID, "claim.json", "ready.json", "result.json") || fence(ctx) != nil || ctx.Err() != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	if err := submit(ctx, saved); err != nil {
		if errors.Is(err, ErrRecordingSubmissionUncertain) {
			return RecordingJobReady{}, ErrRecordingSubmissionUncertain
		}
		return RecordingJobReady{}, ErrRecordingWorker
	}
	// systemd's notify/main startup acknowledgement is separate from readable
	// job files. It must be followed by exact live identity and metadata checks.
	ready, err := jobs.ReadReady(job.ID, digest)
	if err != nil || !recordingJobHasNoFiles(jobs, job.ID, "result.json") {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	worker, err := jobs.ReadWorker(job.ID, digest)
	if err != nil || !worker.valid() || worker.InvocationID == "" {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	selected, _ := ReadManagedRecordingJob(raw)
	first, err := observe(ctx, selected, worker)
	if err != nil || CheckRecordingSupervisor(job, first, worker, worker) != nil || recordingReadyMetadata(ctx, opts, job, ready, worker) != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	selected, _ = ReadManagedRecordingJob(raw)
	if native(ctx, selected) != nil || ctx.Err() != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	selected, _ = ReadManagedRecordingJob(raw)
	last, err := observe(ctx, selected, worker)
	if err != nil || last != first || CheckRecordingSupervisor(job, last, worker, worker) != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	finalReady, err := jobs.ReadReady(job.ID, digest)
	finalWorker, workerErr := jobs.ReadWorker(job.ID, digest)
	if err != nil || workerErr != nil || finalReady != ready || finalWorker != worker || !recordingJobHasNoFiles(jobs, job.ID, "result.json") || recordingReadyMetadata(ctx, opts, job, ready, worker) != nil || fence(ctx) != nil || ctx.Err() != nil {
		return RecordingJobReady{}, ErrRecordingWorker
	}
	return ready, nil
}
