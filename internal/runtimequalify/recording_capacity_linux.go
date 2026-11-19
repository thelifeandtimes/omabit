//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"io"
	"sort"
	"syscall"
	"time"
)

// RecordingCapacity reads only saved private job/allocation files. It contacts no
// engine or unit. Unknown/foreign/partial jobs and unattributed session entries
// refuse admission rather than being silently omitted or repaired.
func RecordingCapacity(ctx context.Context, opts RecordingWorkerOptions, runner string) (recording.Capacity, error) {
	out := recording.Capacity{Version: 1, RunnerID: runner, Instances: []recording.InstanceCommitment{}, ReadAt: time.Now().UTC(), Basis: "full immutable maximum of every retained job, including closed/unclaimed jobs; not measured usage, free-space reservation, or raw availability"}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, 4096, 1)
	if err != nil {
		return out, err
	}
	defer jobs.Close()
	root, err := jobs.root.Open(".")
	if err != nil {
		return out, ErrRecordingJobStorage
	}
	var fs syscall.Statfs_t
	if syscall.Fstatfs(int(root.Fd()), &fs) != nil || fs.Bsize <= 0 || fs.Bavail > ^uint64(0)/uint64(fs.Bsize) {
		root.Close()
		return out, ErrRecordingJobStorage
	}
	out.AvailableBytes = fs.Bavail * uint64(fs.Bsize)
	entries, err := root.ReadDir(4098)
	root.Close()
	if err != nil && err != io.EOF || len(entries) > 4097 {
		return out, ErrRecordingJobStorage
	}
	instances := map[string]recording.InstanceCommitment{}
	sessions := map[string]bool{}
	jobInstances := map[string]string{}
	sessionInstances := map[string]string{}
	for _, e := range entries {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		if e.Name() == "admission.lock" {
			b, err := recordingJobRead(jobs.root, e.Name(), 0)
			if err != nil || len(b) != 0 {
				return out, ErrRecordingJobStorage
			}
			continue
		}
		child, err := recordingJobChild(jobs.root, e.Name())
		if err != nil {
			return out, ErrRecordingJobStorage
		}
		raw, err := recordingJobRead(child, "job.json", MaxRecordingJobBytes)
		child.Close()
		if err != nil {
			return out, ErrRecordingJobStorage
		}
		job, err := jobs.Read(e.Name(), recordingJobSHA(raw))
		if err != nil || job.Operation.RunnerID != runner || job.Operation.Target.InstanceID == "" {
			return out, ErrRecordingJobStorage
		}
		if a, err := jobs.ReadSession(job.ID, recordingJobSHA(raw)); err == nil {
			if sessions[a.SessionID] {
				return out, ErrRecordingJobStorage
			}
			sessions[a.SessionID] = true
			sessionInstances[a.SessionID] = job.Operation.Target.InstanceID
		} else if !errors.Is(err, ErrRecordingSessionUnknown) {
			return out, ErrRecordingJobStorage
		}
		i := instances[job.Operation.Target.InstanceID]
		i.InstanceID = job.Operation.Target.InstanceID
		i.Jobs++
		i.CommittedBytes += uint64(job.Limits.MaxSessionBytes)
		instances[i.InstanceID] = i
		jobInstances[job.ID] = i.InstanceID
		out.Jobs++
		out.CommittedBytes += uint64(job.Limits.MaxSessionBytes)
	}
	// A partial session may lack a header; its allocation still charges the job's
	// full maximum. A directory with no saved allocation has unknown ownership.
	recordings, err := openRecordingPrivateRoot(opts.RecordingsRoot)
	if err != nil {
		return out, err
	}
	defer recordings.Close()
	dir, err := recordings.Open(".")
	if err != nil {
		return out, err
	}
	var recordingFS syscall.Statfs_t
	if syscall.Fstatfs(int(dir.Fd()), &recordingFS) != nil || recordingFS.Bsize <= 0 || recordingFS.Bavail > ^uint64(0)/uint64(recordingFS.Bsize) {
		dir.Close()
		return out, ErrRecordingJobStorage
	}
	out.AvailableBytes = min(out.AvailableBytes, recordingFS.Bavail*uint64(recordingFS.Bsize))
	saved, err := dir.ReadDir(4097)
	dir.Close()
	if err != nil && err != io.EOF || len(saved) > 4096 {
		return out, ErrRecordingJobStorage
	}
	for _, e := range saved {
		if !sessions[e.Name()] {
			return out, ErrRecordingJobStorage
		}
		child, err := recordingJobChild(recordings, e.Name())
		if err != nil {
			return out, err
		}
		child.Close()
	}
	for _, i := range instances {
		out.Instances = append(out.Instances, i)
	}
	sort.Slice(out.Instances, func(i, j int) bool { return out.Instances[i].InstanceID < out.Instances[j].InstanceID })
	usage := measureRecordingUsage(ctx, jobs.root, recordings, runner, jobInstances, sessionInstances, out.Instances, recording.MaxUsageEntries)
	out.Usage = &usage
	if out.Validate() != nil || ctx.Err() != nil {
		return out, ErrRecordingJobStorage
	}
	return out, nil
}
