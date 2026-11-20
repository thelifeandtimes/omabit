//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

var ErrRecordingWorker = errors.New("recording worker did not finish its verified capture; inspect the saved job and retained session")

// RecordingWorkerOptions comes from trusted local runner configuration, never
// an RPC or a model-visible tool's arbitrary filesystem parameters. This source
// worker is supervised independently when ordinary launch explicitly selects the pilot.
type RecordingWorkerOptions struct {
	JobsRoot            string
	RecordingsRoot      string
	MaxJobs             int
	JobReserveFreeBytes uint64
	beforeClaim         func(context.Context, ManagedRecordingJob, RecordingWorkerIdentity) error
}

// RunManagedRecordingWorker uses the worker process's own independent lifetime.
// Acknowledge must be a trusted supervisor handoff after all readiness syncs
// return; a readable file alone cannot establish that acknowledgement. This
// function sends no Start/Stop/signal/stdin and retries neither job nor capture.
func RunManagedRecordingWorker(ctx context.Context, opts RecordingWorkerOptions, id, digest string, acknowledge func(RecordingJobReady) error) (RecordingJobResult, error) {
	d := defaults()
	return runManagedRecordingWorker(ctx, opts, id, digest, acknowledge, d, observeRecordingWorkerIdentity, attachDockerRecording)
}

func observeRecordingWorkerIdentity(q *qualifier) (RecordingWorkerIdentity, error) {
	boot, namespaces, err := recordingHostContext(q)
	if err != nil || os.Getpid() < 2 || uint64(os.Getpid()) > math.MaxUint32 {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	// Kernel-selected self image, not a caller path or a mutable current symlink.
	file, err := os.Open("/proc/self/exe")
	if err != nil {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 128<<20 {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, 128<<20+1))
	if err != nil || count != info.Size() {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	stat, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	ticks, err := recordingWorkerStartTicks(stat)
	if err != nil {
		return RecordingWorkerIdentity{}, err
	}
	worker := RecordingWorkerIdentity{ExecutableSHA256: hex.EncodeToString(hash.Sum(nil)), PID: uint32(os.Getpid()), StartTicks: ticks, HostBootID: boot, Namespaces: namespaces, InvocationID: os.Getenv("INVOCATION_ID")}
	if !worker.valid() {
		return RecordingWorkerIdentity{}, ErrRecordingWorker
	}
	return worker, nil
}

func recordingWorkerStartTicks(raw []byte) (uint64, error) {
	// The parenthesized Linux comm field may itself contain spaces and ')'.
	end := bytes.LastIndexByte(raw, ')')
	if len(raw) > 4096 || end < 0 || end+1 >= len(raw) {
		return 0, ErrRecordingWorker
	}
	fields := strings.Fields(string(raw[end+1:]))
	if len(fields) < 20 {
		return 0, ErrRecordingWorker
	}
	ticks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || ticks == 0 {
		return 0, ErrRecordingWorker
	}
	return ticks, nil
}

func runManagedRecordingWorker(ctx context.Context, opts RecordingWorkerOptions, id, digest string, ack func(RecordingJobReady) error, d dependencies, self func(*qualifier) (RecordingWorkerIdentity, error), attach func(context.Context, string, string) (io.ReadCloser, error)) (out RecordingJobResult, err error) {
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if ack == nil || self == nil || attach == nil {
		return out, ErrRecordingWorker
	}
	q := &qualifier{d: d}
	if q.accountBoundary(false) != nil {
		return out, ErrRecordingWorker
	}
	worker, err := self(q)
	if err != nil {
		return out, ErrRecordingWorker
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, opts.MaxJobs, opts.JobReserveFreeBytes)
	if err != nil {
		return out, err
	}
	defer jobs.Close()
	job, err := jobs.Read(id, digest)
	if err != nil {
		return out, err
	}
	if opts.beforeClaim != nil && opts.beforeClaim(ctx, job, worker) != nil {
		return out, ErrRecordingWorker
	}
	jobs.reserve = max(jobs.reserve, job.Limits.ReserveFreeBytes)
	lease, err := jobs.Claim(id, digest, worker)
	if err != nil {
		return out, err
	}
	defer lease.Close()
	finish := func(code string, capture recording.CaptureResult, cause error) (RecordingJobResult, error) {
		result, saveErr := lease.RecordResult(code, capture, time.Now())
		if cause != nil || saveErr != nil {
			return result, ErrRecordingWorker
		}
		return result, nil
	}
	m, source, err := prepareManagedRecording(ctx, job.State, job.Operation, func(actual recording.Metadata) error {
		if err := lease.Ready(actual); err != nil {
			return ErrRecordingWorker
		}
		ready, err := jobs.ReadReady(id, digest)
		if err != nil {
			return ErrRecordingWorker
		}
		// Only this acknowledgement follows successful fsync returns. The
		// supervisor must additionally bind the exact still-live worker.
		if ack(ready) != nil {
			return ErrRecordingWorker
		}
		return nil
	}, d, attach)
	if err != nil || !job.MatchesAdmission(m) {
		return finish("admission-refused", recording.CaptureResult{}, ErrRecordingWorker)
	}
	if m.Identity.HostBootID != worker.HostBootID || m.Identity.RecorderNamespaces != worker.Namespaces {
		return finish("admission-refused", recording.CaptureResult{}, ErrRecordingWorker)
	}
	if err := lease.AllocateSession(m.SessionID, time.Now()); err != nil {
		return finish("storage-unavailable", recording.CaptureResult{}, err)
	}
	store, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
	if err != nil {
		return finish("storage-unavailable", recording.CaptureResult{}, err)
	}
	capture, captureErr := store.CaptureDocker(ctx, m, source)
	closeErr := store.Close()
	code := "capture-finished"
	if captureErr != nil || closeErr != nil {
		code = "capture-incomplete"
	}
	return finish(code, capture, errors.Join(captureErr, closeErr))
}
