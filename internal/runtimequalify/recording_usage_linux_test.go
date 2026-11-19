//go:build linux && amd64

package runtimequalify

import (
	"context"
	"crypto/sha256"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func TestRecordingUsageMeasuresSparseAndPartialOwnedStorageWithoutReadingOrChangingIt(t *testing.T) {
	jobs, jobRoot, job, meta, worker := recordingJobsFixture(t, 8)
	digest, err := jobs.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := jobs.Claim(job.ID, digest, worker)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := lease.AllocateSession(meta.SessionID, time.Now()); err != nil {
		t.Fatal(err)
	}
	sessionRoot := t.TempDir()
	if err := os.Chmod(sessionRoot, 0700); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(sessionRoot, meta.SessionID)
	if err := os.Mkdir(session, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(session, "frames.obr")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(4 << 20); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("unverified partial retained bytes"), (4<<20)-4096); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(before)
	// This fixture deliberately has no recording header/valid frame chain.
	// File allocation evidence must not be confused with recording coverage.
	got, err := RecordingCapacity(t.Context(), RecordingWorkerOptions{JobsRoot: jobRoot, RecordingsRoot: sessionRoot}, job.Operation.RunnerID)
	if err != nil || got.Usage == nil || got.Usage.Status != "measured" || got.Usage.Validate(got.RunnerID, got.Instances) != nil {
		t.Fatal("measurement unavailable", got.Usage, err)
	}
	want := recording.StorageTotals{}
	for _, root := range []string{jobRoot, sessionRoot} {
		err := filepath.WalkDir(root, func(p string, e fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := e.Info()
			if err != nil {
				return err
			}
			st := info.Sys().(*syscall.Stat_t)
			want.AllocatedBytes += uint64(st.Blocks) * 512
			if e.IsDir() {
				want.Directories++
			} else {
				want.Files++
				want.LogicalBytes += uint64(info.Size())
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if *got.Usage.Totals != want || want.LogicalBytes <= want.AllocatedBytes || len(got.Usage.Instances) != 1 || got.Usage.Instances[0].InstanceID != job.Operation.Target.InstanceID {
		t.Fatal("physical/logical accounting lost", got.Usage, want)
	}
	if got.CommittedBytes != uint64(job.Limits.MaxSessionBytes) {
		t.Fatal("measurement changed admission budget")
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(after) != hash {
		t.Fatal("measurement changed original bytes", err)
	}
}

func TestRecordingUsageUnknownDoesNotHideCommitmentsOrReleasePartialTotals(t *testing.T) {
	for _, name := range []string{"unexpected-file", "symlink", "hardlink", "permissions", "fifo", "entry-limit", "cancelled", "replaced-root"} {
		t.Run(name, func(t *testing.T) {
			jobs, jobRoot, job, _, _ := recordingJobsFixture(t, 8)
			if _, err := jobs.Create(job); err != nil {
				t.Fatal(err)
			}
			sessionRoot := t.TempDir()
			if err := os.Chmod(sessionRoot, 0700); err != nil {
				t.Fatal(err)
			}
			root, err := openRecordingPrivateRoot(sessionRoot)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			file := filepath.Join(jobRoot, job.ID, "result.json")
			outside := filepath.Join(t.TempDir(), "private-unrelated")
			if err := os.WriteFile(outside, []byte("preserve unrelated bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			limit := recording.MaxUsageEntries
			ctx := t.Context()
			switch name {
			case "unexpected-file":
				if err := os.WriteFile(filepath.Join(jobRoot, job.ID, "unknown"), []byte("retain"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(outside, file); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(outside, file); err != nil {
					t.Fatal(err)
				}
			case "permissions":
				if err := os.WriteFile(file, []byte("partial result"), 0644); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := syscall.Mkfifo(file, 0600); err != nil {
					t.Fatal(err)
				}
			case "entry-limit":
				limit = 2
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "replaced-root":
				moved := filepath.Join(t.TempDir(), "old-session-root")
				if err := os.Rename(sessionRoot, moved); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(sessionRoot, 0700); err != nil {
					t.Fatal(err)
				}
			}
			instances := []recording.InstanceCommitment{{InstanceID: job.Operation.Target.InstanceID, Jobs: 1, CommittedBytes: uint64(job.Limits.MaxSessionBytes)}}
			u := measureRecordingUsage(ctx, jobs.root, root, job.Operation.RunnerID, map[string]string{job.ID: job.Operation.Target.InstanceID}, map[string]string{}, instances, limit)
			if u.Status != "unknown" || u.Reason == "" || u.Totals != nil || u.Shared != nil || len(u.Instances) != 0 || u.Validate(job.Operation.RunnerID, instances) != nil {
				t.Fatal("unsafe/incomplete census became usage", u)
			}
			if name == "entry-limit" && u.Reason != "entry-limit" || name == "cancelled" && u.Reason != "cancelled" || name == "replaced-root" && u.Reason != "changed" {
				t.Fatal("failure reason lost", name, u.Reason)
			}
			data, err := os.ReadFile(outside)
			if err != nil || string(data) != "preserve unrelated bytes" {
				t.Fatal("unrelated bytes changed", err)
			}
		})
	}
}

func TestRecordingUsageInstanceLimitRefusesBeforeOpeningRoots(t *testing.T) {
	instances := make([]recording.InstanceCommitment, recording.MaxUsageInstances+1)
	u := measureRecordingUsage(t.Context(), nil, nil, "runner", nil, nil, instances, recording.MaxUsageEntries)
	if u.Status != "unknown" || u.Reason != "entry-limit" || u.Totals != nil || u.Validate("runner", instances) != nil {
		t.Fatal("large usage census admitted or partially exposed", u)
	}
}
