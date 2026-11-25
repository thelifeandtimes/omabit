//go:build linux && amd64

package runtimequalify

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimeprofile"
	"os"
	"path/filepath"
)

// RunRecordingService is the internal independently supervised worker entry.
// It accepts only an exact existing job and digest, never a filesystem root,
// command, native input or configuration from a client. Normal managed launch
// does not create or submit jobs yet; its journal/downgrade guard is still needed.
func RunRecordingService(ctx context.Context, id, digest string) error {
	d := defaults()
	q := &qualifier{d: d}
	if q.accountBoundary(false) != nil {
		return ErrRecordingWorker
	}
	runtimeprofile.ConfigureStore(filepath.Join(d.home, ".local/share/urbit-runner/runtime-profiles"))
	return runRecordingService(ctx, id, digest, d, os.Getenv, RunManagedRecordingWorker)
}

func runRecordingService(ctx context.Context, id, digest string, d dependencies, getenv func(string) string, worker func(context.Context, RecordingWorkerOptions, string, string, func(RecordingJobReady) error) (RecordingJobResult, error)) error {
	if !recordingJobHex(id, 32) || !recordingJobHex(digest, 64) || ctx.Err() != nil || getenv == nil || worker == nil || !validRecordingNotifyEnvironment(getenv("NOTIFY_SOCKET"), getenv("INVOCATION_ID")) {
		return ErrRecordingWorker
	}
	opts, err := recordingServiceOptions(d)
	if err != nil {
		return ErrRecordingWorker
	}
	opts.beforeClaim = func(ctx context.Context, job ManagedRecordingJob, current RecordingWorkerIdentity) error {
		// A caller-supplied service environment cannot establish supervision.
		// Before consuming the once-only claim, verify that systemd actually
		// tracks this exact self image/process as the starting notify/main unit.
		_, err := observeRecordingSupervisor(ctx, job, current, false, readRecordingSupervisor, observeRecordingProcess)
		return err
	}
	_, err = worker(ctx, opts, id, digest, func(RecordingJobReady) error {
		return notifyRecordingSupervisor(ctx, getenv("NOTIFY_SOCKET"), getenv("INVOCATION_ID"))
	})
	if err != nil {
		return ErrRecordingWorker
	}
	return nil
}

func recordingServiceOptions(d dependencies) (RecordingWorkerOptions, error) {
	return recordingRootOptions(d, true)
}

func recordingRootOptions(d dependencies, requireRecordings bool) (RecordingWorkerOptions, error) {
	q := &qualifier{d: d}
	if !recordingJobPath(d.home) || d.home == "/" || q.accountBoundary(false) != nil {
		return RecordingWorkerOptions{}, ErrRecordingWorker
	}
	base := filepath.Join(d.home, ".local/share/urbit-runner")
	opts := RecordingWorkerOptions{JobsRoot: filepath.Join(base, "recording-jobs"), RecordingsRoot: filepath.Join(base, "recordings"), MaxJobs: 1024, JobReserveFreeBytes: 64 << 20}
	paths := []string{opts.JobsRoot}
	if requireRecordings {
		paths = append(paths, opts.RecordingsRoot)
	}
	for _, path := range paths {
		if q.ancestors(path, true) != nil || q.owned(path, os.ModeDir, true) != nil {
			return RecordingWorkerOptions{}, ErrRecordingWorker
		}
	}
	return opts, nil
}
