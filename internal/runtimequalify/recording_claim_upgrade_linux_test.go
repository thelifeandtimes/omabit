//go:build linux && amd64

package runtimequalify

import (
	"encoding/json"
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
		if _, err := s.ReadSession(job.ID, digest); !errors.Is(err, ErrRecordingSessionUnknown) {
			t.Fatal("legacy omission was backfilled or became session authority", err)
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

func TestRecordingActualOriginalClaimFixture(t *testing.T) {
	base := "testdata/original-recording-claim1"
	raw, err := os.ReadFile(filepath.Join(base, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Source string            `json:"original_source"`
		Helper string            `json:"declared_helper_sha256"`
		Inputs string            `json:"original_source_manifest_sha256"`
		Files  map[string]string `json:"files"`
	}
	if recordingJobSHA(raw) != "9d45af2e61b3d60e36593e5e508423b6ffa1dc672a4806eec9c338b1ec27d3e8" || json.Unmarshal(raw, &m) != nil || m.Source != "85ac7c6892dd4934b64e5dacc98846001ab3b01d" || m.Helper != "14a10b65054d9840010c0f6fcfc138db0439ecf2960840e840ba0d56dff6bf9c" || m.Inputs != "f85918611d6aab2c41d41dc57fbe1531070ef89d6f78421da4d03e42a2f7c4c9" || len(m.Files) != 5 {
		t.Fatal("actual original writer provenance changed")
	}
	for name, pin := range map[string]string{"writer-helper.go.txt": m.Helper, "original-source.json": m.Inputs} {
		data, err := os.ReadFile(filepath.Join(base, name))
		if err != nil || recordingJobSHA(data) != pin {
			t.Fatal("original source/helper changed", name, err)
		}
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	for name, pin := range m.Files {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name {
			t.Fatal("fixture escaped root")
		}
		data, err := os.ReadFile(filepath.Join(base, "files", name))
		if err != nil || recordingJobSHA(data) != pin {
			t.Fatal("original writer bytes changed", name, err)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	job, _, _ := checkOriginalRecordingClaim(t, root)
	if job.Operation.ConfigurationRevision != 9007199254740999 {
		t.Fatal("original wide revision was truncated")
	}
	for name, pin := range m.Files {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || recordingJobSHA(data) != pin {
			t.Fatal("read/reopen/refused claim changed original bytes", name, err)
		}
	}
}
