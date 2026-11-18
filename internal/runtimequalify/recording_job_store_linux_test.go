//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func recordingJobsFixture(t *testing.T, maximum int) (*RecordingJobs, string, ManagedRecordingJob, recording.Metadata, RecordingWorkerIdentity) {
	t.Helper()
	job, _, _, m := recordingJobFixture(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := OpenRecordingJobs(dir, maximum, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	worker := RecordingWorkerIdentity{ExecutableSHA256: job.ExecutableSHA256, PID: uint32(os.Getpid()), HostBootID: m.Identity.HostBootID, Namespaces: m.Identity.RecorderNamespaces}
	return store, dir, job, m, worker
}

func TestRecordingJobsDurablePrivateHandoffSurvivesParentHandleClosure(t *testing.T) {
	parent, dir, job, m, identity := recordingJobsFixture(t, 8)
	want, _ := job.Bytes()
	digest, err := parent.Create(job)
	if err != nil || digest != recordingJobSHA(want) {
		t.Fatal("job acknowledgement was not exact", err)
	}
	for _, name := range []string{"job.json", "worker.lock"} {
		info, err := os.Lstat(filepath.Join(dir, job.ID, name))
		if err != nil || !recordingJobFileInfo(info, MaxRecordingJobBytes) {
			t.Fatal("job file was exposed/linked", name, err)
		}
	}
	if _, err = parent.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobNotReady) {
		t.Fatal("job creation became readiness", err)
	}
	worker, err := OpenRecordingJobs(dir, 8, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	lease, err := worker.Claim(job.ID, digest, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err = parent.Close(); err != nil {
		t.Fatal(err)
	}
	// Separate handles model separate manager/worker ownership; this is not
	// actual systemd/process-lifetime qualification.
	if err = lease.Ready(m); err != nil {
		t.Fatal("parent handle closure canceled independent writer", err)
	}
	ready, err := worker.ReadReady(job.ID, digest)
	metadataHash, _ := m.Digest()
	if err != nil || ready.SessionID != m.SessionID || ready.MetadataSHA256 != recordingJobSHA(mustRecordingMetadata(t, m)) || ready.JobSHA256 != digest || metadataHash == [32]byte{} {
		t.Fatal("durable readiness rebound metadata", ready, err)
	}
	before, err := os.ReadFile(filepath.Join(dir, job.ID, "ready.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = lease.Ready(m); err == nil {
		t.Fatal("immutable readiness was overwritten")
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := worker.Read(job.ID, digest)
	if err != nil || !reflect.DeepEqual(got, job) {
		t.Fatal("read-only lookup changed job", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, job.ID, "ready.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("repeated readiness/close/lookup changed original", err)
	}
	if _, err = worker.Claim(job.ID, digest, identity); !errors.Is(err, ErrRecordingJobClaimed) {
		t.Fatal("closed worker claim was replayed", err)
	}
	// Saved readiness is still a historical receipt after the lock closes.
	if _, err = worker.ReadReady(job.ID, digest); err != nil {
		t.Fatal("saved readiness became dependent on live worker", err)
	}
}

func mustRecordingMetadata(t *testing.T, m recording.Metadata) []byte {
	t.Helper()
	b, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRecordingJobsClaimIsExclusiveAndExecutableBound(t *testing.T) {
	s, dir, job, _, worker := recordingJobsFixture(t, 8)
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	wrong := worker
	wrong.ExecutableSHA256 = strings.Repeat("f", 64)
	if _, err = s.Claim(job.ID, digest, wrong); !errors.Is(err, ErrRecordingJob) {
		t.Fatal("wrong executable claimed job", err)
	}
	if _, err = os.Lstat(filepath.Join(dir, job.ID, "claim.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused executable created claim", err)
	}
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJobActive) {
		t.Fatal("two workers claimed same job", err)
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJobClaimed) {
		t.Fatal("released claim was silently retried", err)
	}
	if _, err = s.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobNotReady) {
		t.Fatal("claim without attachment became readiness", err)
	}
}

func TestRecordingJobsRefuseChangedFilesBeforeReadiness(t *testing.T) {
	for _, scenario := range []string{"job-bytes", "job-link", "lock-replaced", "claim-bytes", "directory-replaced", "directory-exposed", "worker-boot", "worker-namespaces", "metadata-binding", "ready-partial"} {
		t.Run(scenario, func(t *testing.T) {
			s, dir, job, m, worker := recordingJobsFixture(t, 8)
			digest, err := s.Create(job)
			if err != nil {
				t.Fatal(err)
			}
			lease, err := s.Claim(job.ID, digest, worker)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			root := filepath.Join(dir, job.ID)
			switch scenario {
			case "job-bytes":
				if err = os.WriteFile(filepath.Join(root, "job.json"), []byte("partial original"), 0600); err != nil {
					t.Fatal(err)
				}
			case "job-link":
				if err = os.Link(filepath.Join(root, "job.json"), filepath.Join(dir, "retained-link")); err != nil {
					t.Fatal(err)
				}
			case "lock-replaced":
				if err = os.Rename(filepath.Join(root, "worker.lock"), filepath.Join(root, "retained.lock")); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, "worker.lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "claim-bytes":
				if err = os.WriteFile(filepath.Join(root, "claim.json"), []byte("partial original"), 0600); err != nil {
					t.Fatal(err)
				}
			case "directory-replaced":
				if err = os.Rename(root, root+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err = os.Mkdir(root, 0700); err != nil {
					t.Fatal(err)
				}
			case "directory-exposed":
				if err = os.Chmod(root, 0755); err != nil {
					t.Fatal(err)
				}
			case "worker-boot":
				m.Identity.HostBootID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			case "worker-namespaces":
				m.Identity.RecorderNamespaces.PID++
			case "metadata-binding":
				m.ContainerID = strings.Repeat("f", 64)
			case "ready-partial":
				if err = os.WriteFile(filepath.Join(root, "ready.json"), []byte("partial original"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err = lease.Ready(m); err == nil {
				t.Fatal("changed private handoff became readiness")
			}
			if scenario == "ready-partial" {
				data, _ := os.ReadFile(filepath.Join(root, "ready.json"))
				if string(data) != "partial original" {
					t.Fatal("partial readiness was repaired/overwritten")
				}
			} else if _, err = os.Lstat(filepath.Join(root, "ready.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("refusal created readiness", err)
			}
		})
	}
}

func TestRecordingJobsSpaceAndCountRefusalPreserveOriginals(t *testing.T) {
	s, dir, job, m, worker := recordingJobsFixture(t, 1)
	s.free = func(*os.Root, uint64) error { return ErrRecordingJobLimit }
	if _, err := s.Create(job); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("space refusal created control files", err)
	}
	s.free = recordingJobSpace
	digest, err := s.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	second := job
	second.ID = strings.Repeat("3", 32)
	if _, err = s.Create(second); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal("quota admitted another job", err)
	}
	s.free = func(*os.Root, uint64) error { return ErrRecordingJobLimit }
	if _, err = s.Claim(job.ID, digest, worker); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal(err)
	}
	if _, err = os.Lstat(filepath.Join(dir, job.ID, "claim.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pressure created a claim", err)
	}
	s.free = recordingJobSpace
	lease, err := s.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	s.free = func(*os.Root, uint64) error { return ErrRecordingJobLimit }
	if err = lease.Ready(m); !errors.Is(err, ErrRecordingJobLimit) {
		t.Fatal("pressure acknowledged attachment", err)
	}
	if _, err = s.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobNotReady) {
		t.Fatal(err)
	}
	if _, err = s.Read(job.ID, digest); err != nil {
		t.Fatal("pressure changed original job", err)
	}
}

func TestRecordingJobsConcurrentCreatorsCannotExceedQuota(t *testing.T) {
	s, dir, job, _, _ := recordingJobsFixture(t, 1)
	other, err := OpenRecordingJobs(dir, 1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	second := job
	second.ID = strings.Repeat("3", 32)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, store := range []*RecordingJobs{s, other} {
		wg.Add(1)
		go func(index int, store *RecordingJobs) {
			defer wg.Done()
			j := job
			if index == 1 {
				j = second
			}
			_, err := store.Create(j)
			results <- err
		}(i, store)
	}
	wg.Wait()
	close(results)
	created := 0
	for err := range results {
		if err == nil {
			created++
		} else if !errors.Is(err, ErrRecordingJobActive) && !errors.Is(err, ErrRecordingJobLimit) {
			t.Fatal(err)
		}
	}
	if created != 1 {
		t.Fatal("quota failed under independent admission locks", created)
	}
}

func TestRecordingJobsSavedReadinessRequiresOriginalClaimAndCanonicalBytes(t *testing.T) {
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
	root := filepath.Join(dir, job.ID)
	claim, err := os.ReadFile(filepath.Join(root, "claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(filepath.Join(root, "claim.json"), filepath.Join(root, "retained-claim.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobStorage) {
		t.Fatal("readiness without original claim was accepted", err)
	}
	if err = os.WriteFile(filepath.Join(root, "claim.json"), claim, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "ready.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields["unexpected"] = json.RawMessage("true")
	changed, _ := json.Marshal(fields)
	if err = os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadReady(job.ID, digest); !errors.Is(err, ErrRecordingJobStorage) {
		t.Fatal("noncanonical/unknown readiness promoted", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, changed) {
		t.Fatal("lookup repaired a malformed original")
	}
}
