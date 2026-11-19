//go:build linux && amd64

package runtimequalify

import (
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRecordingCapacityChargesUnclaimedJobsAndRefusesUnattributedStorage(t *testing.T) {
	jobs, dir, job, _, _ := recordingJobsFixture(t, 8)
	recordings := t.TempDir()
	if err := os.Chmod(recordings, 0700); err != nil {
		t.Fatal(err)
	}
	opts := RecordingWorkerOptions{JobsRoot: dir, RecordingsRoot: recordings}
	digest, err := jobs.Create(job)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, job.ID, "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := RecordingCapacity(t.Context(), opts, job.Operation.RunnerID)
	if err != nil || got.Jobs != 1 || got.CommittedBytes != uint64(job.Limits.MaxSessionBytes) || len(got.Instances) != 1 || got.Instances[0].InstanceID != job.Operation.Target.InstanceID {
		t.Fatal("unclaimed job disappeared from budget", got, err)
	}
	policy := recording.DefaultRetentionPolicy()
	policy.MaxSessionBytes = job.Limits.MaxSessionBytes
	policy.MaxSegmentBytes = job.Limits.MaxSegmentBytes
	policy.InstanceMaxRawBytes = uint64(job.Limits.MaxSessionBytes)
	if got.Admit(policy, job.Operation.Target.InstanceID) == nil {
		t.Fatal("ship quota did not include unclaimed job")
	}
	if got.Admit(policy, "new-instance") != nil {
		t.Fatal("different instance lost remaining host capacity")
	}
	if _, err := RecordingCapacity(t.Context(), opts, "foreign-runner"); err == nil {
		t.Fatal("foreign jobs admitted")
	}
	orphan := filepath.Join(recordings, strings.Repeat("e", 32))
	if err := os.Mkdir(orphan, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordingCapacity(t.Context(), opts, job.Operation.RunnerID); err == nil {
		t.Fatal("unattributed session omitted")
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Fatal("unknown session was removed")
	}
	after, err := os.ReadFile(filepath.Join(dir, job.ID, "job.json"))
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("capacity read changed original job", err)
	}
	if _, err := jobs.Read(job.ID, digest); err != nil {
		t.Fatal(err)
	}
}
