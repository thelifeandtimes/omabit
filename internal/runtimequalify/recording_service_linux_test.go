//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingServiceUsesOnlySelectedOwnerRootAndRefusesMissingOrUnsafeDirectories(t *testing.T) {
	f, _, _ := managedFixture(t)
	base := filepath.Join(f.d.home, ".local/share/urbit-runner")
	jobs, sessions := filepath.Join(base, "recording-jobs"), filepath.Join(base, "recordings")
	if _, err := recordingServiceOptions(f.d); err == nil {
		t.Fatal("missing roots were created or adopted")
	}
	for _, path := range []string{jobs, sessions} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("missing root changed", err)
		}
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", "/unrelated-home")
	t.Setenv("XDG_DATA_HOME", "/unrelated-data")
	t.Setenv("DOCKER_HOST", "unix:///unrelated.sock")
	opts, err := recordingServiceOptions(f.d)
	if err != nil || opts.JobsRoot != jobs || opts.RecordingsRoot != sessions || opts.MaxJobs != 1024 || opts.JobReserveFreeBytes != 64<<20 {
		t.Fatal("recording roots were not fixed to owner selection", err)
	}
	if err := os.Chmod(sessions, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := recordingServiceOptions(f.d); err == nil {
		t.Fatal("exposed session root accepted")
	}
	if err := os.Rename(sessions, sessions+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sessions+"-original", sessions); err != nil {
		t.Fatal(err)
	}
	if _, err := recordingServiceOptions(f.d); err == nil {
		t.Fatal("symlinked root accepted")
	}
}

func TestRecordingServiceRejectsMalformedOrUnsupervisedEntryBeforeAccountOrStorage(t *testing.T) {
	id, digest := strings.Repeat("a", 32), strings.Repeat("b", 64)
	for _, scenario := range []string{"missing-invocation", "missing-socket", "invalid-id", "invalid-digest", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			jobID, sum := id, digest
			if scenario == "invalid-id" {
				jobID = "../other"
			}
			if scenario == "invalid-digest" {
				sum = "PRIVATE-INPUT"
			}
			if scenario == "canceled" {
				cancel()
			}
			getenv := func(key string) string {
				if key == "INVOCATION_ID" && scenario != "missing-invocation" {
					return strings.Repeat("c", 32)
				}
				if key == "NOTIFY_SOCKET" && scenario != "missing-socket" {
					return "/run/user/1000/notify"
				}
				return ""
			}
			err := runRecordingService(ctx, jobID, sum, dependencies{}, getenv, func(context.Context, RecordingWorkerOptions, string, string, func(RecordingJobReady) error) (RecordingJobResult, error) {
				t.Fatal("malformed entry reached worker")
				return RecordingJobResult{}, nil
			})
			if !errors.Is(err, ErrRecordingWorker) {
				t.Fatal("entry did not refuse finitely", err)
			}
		})
	}
}

func TestRecordingServiceWorkerFailureIsFiniteAndNotRetried(t *testing.T) {
	f, _, _ := managedFixture(t)
	for _, leaf := range []string{"recording-jobs", "recordings"} {
		if err := os.Mkdir(filepath.Join(f.d.home, ".local/share/urbit-runner", leaf), 0700); err != nil {
			t.Fatal(err)
		}
	}
	getenv := func(key string) string {
		if key == "INVOCATION_ID" {
			return strings.Repeat("c", 32)
		}
		return "/run/user/1000/notify"
	}
	calls := 0
	err := runRecordingService(t.Context(), strings.Repeat("a", 32), strings.Repeat("b", 64), f.d, getenv, func(_ context.Context, opts RecordingWorkerOptions, _, _ string, ack func(RecordingJobReady) error) (RecordingJobResult, error) {
		calls++
		if opts.beforeClaim == nil || ack == nil {
			t.Fatal("service lost supervisor or post-sync notification gate")
		}
		return RecordingJobResult{}, errors.New("PRIVATE-FAILURE")
	})
	if err != ErrRecordingWorker || calls != 1 {
		t.Fatal("worker retried or private failure leaked", calls, err)
	}
}

func TestRecordingWorkerSupervisorRefusalPreservesUnclaimedJob(t *testing.T) {
	f, opts, job, digest, worker, _ := recordingWorkerFixture(t)
	opts.beforeClaim = func(_ context.Context, selected ManagedRecordingJob, self RecordingWorkerIdentity) error {
		if selected.ID != job.ID || self != worker {
			t.Fatal("supervisor gate rebound worker")
		}
		return ErrRecordingWorker
	}
	before := len(f.calls)
	_, err := runManagedRecordingWorker(t.Context(), opts, job.ID, digest, func(RecordingJobReady) error { t.Fatal("notified without claim"); return nil }, f.d,
		func(*qualifier) (RecordingWorkerIdentity, error) { return worker, nil }, func(context.Context, string, string) (io.ReadCloser, error) {
			t.Fatal("attached without supervised claim")
			return nil, ErrRecordingWorker
		})
	if err != ErrRecordingWorker || len(f.calls) != before {
		t.Fatal("refused supervisor reached engine", err)
	}
	for _, name := range []string{"claim.json", "ready.json", "result.json"} {
		if _, err := os.Lstat(filepath.Join(opts.JobsRoot, job.ID, name)); !os.IsNotExist(err) {
			t.Fatal("refusal consumed or changed original job", name, err)
		}
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer jobs.Close()
	lease, err := jobs.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal("refused supervisor consumed the only claim", err)
	}
	lease.Close()
}
