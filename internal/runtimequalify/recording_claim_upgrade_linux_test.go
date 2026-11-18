//go:build linux && amd64

package runtimequalify

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingActualOriginalClaimWriterQualification(t *testing.T) {
	root := os.Getenv("OMABIT_RECORDING_CLAIM_UPGRADE_ROOT")
	if root == "" {
		t.Skip("explicit source-writer rehearsal only")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !strings.HasPrefix(root, "/tmp/omabit-recording-claim-upgrade-") {
		t.Fatal("explicit disposable source qualification root required")
	}
	raw, err := os.ReadFile(filepath.Join(root, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := recording.ReadMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	job, digest, legacy := checkOriginalRecordingClaim(t, filepath.Join(root, "original"))
	if legacy.StartTicks != 0 || legacy.InvocationID != "" || !job.MatchesAdmission(m) {
		t.Fatal("old omissions gained authority or metadata changed")
	}
	if err := os.Mkdir(filepath.Join(root, "successor"), 0700); err != nil {
		t.Fatal("fresh successor root required", err)
	}
	s, err := OpenRecordingJobs(filepath.Join(root, "successor"), 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Create(job)
	if err != nil || got != digest {
		t.Fatal("successor changed original job bytes", err)
	}
	worker := legacy
	worker.StartTicks, worker.InvocationID = 123, strings.Repeat("a", 32)
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	if err = lease.Ready(m); err != nil {
		t.Fatal(err)
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := s.ReadWorker(job.ID, digest); err != nil || got != worker {
		t.Fatal("new claim lost exact identity", err)
	}
}

func checkOriginalRecordingClaim(t *testing.T, root string) (ManagedRecordingJob, string, RecordingWorkerIdentity) {
	t.Helper()
	id := strings.Repeat("1", 32)
	raw, err := os.ReadFile(filepath.Join(root, id, "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	job, err := ReadManagedRecordingJob(raw)
	if err != nil {
		t.Fatal(err)
	}
	digest := recordingJobSHA(raw)
	for pass := 0; pass < 3; pass++ {
		s, err := OpenRecordingJobs(root, 8, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		worker, err := s.ReadWorker(job.ID, digest)
		if err != nil || worker.StartTicks != 0 || worker.InvocationID != "" || worker.valid() {
			t.Fatal("legacy claim gained current process authority", err)
		}
		ready, err := s.ReadReady(job.ID, digest)
		if err != nil || ready.SessionID != strings.Repeat("2", 32) || ready.JobSHA256 != digest {
			t.Fatal("original saved readiness changed", err)
		}
		current := worker
		current.StartTicks, current.InvocationID = 123, strings.Repeat("a", 32)
		if _, err = s.Claim(job.ID, digest, current); !errors.Is(err, ErrRecordingJobClaimed) {
			t.Fatal("legacy claim was replayed", err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		if pass == 2 {
			return job, digest, worker
		}
	}
	panic("unreachable")
}
