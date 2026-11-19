//go:build linux && amd64

package runtimequalify

import (
	"context"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

const recordingUsageBasis = "non-atomic stat census of private recording job/session roots only; excludes piers, managed SQLite, executable caches and unrelated storage; inode blocks may include shared extents or filesystem compression; no deletion or free-space reservation"

type recordingUsageScan struct {
	ctx       context.Context
	remaining int
	reason    string
}

func (s *recordingUsageScan) refuse(reason string) bool {
	if s.reason == "" {
		s.reason = reason
	}
	return false
}

func (s *recordingUsageScan) count(info os.FileInfo, directory bool) (recording.StorageTotals, bool) {
	var out recording.StorageTotals
	if s.ctx.Err() != nil {
		return out, s.refuse("cancelled")
	}
	if s.remaining <= 0 {
		return out, s.refuse("entry-limit")
	}
	s.remaining--
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Blocks < 0 || uint64(st.Blocks) > ^uint64(0)/512 || info.Size() < 0 {
		return out, s.refuse("unavailable")
	}
	out.AllocatedBytes = uint64(st.Blocks) * 512
	if directory {
		out.Directories = 1
	} else {
		out.Files = 1
		out.LogicalBytes = uint64(info.Size())
	}
	return out, true
}

func sameRecordingUsageStat(a, b os.FileInfo) bool {
	x, xok := a.Sys().(*syscall.Stat_t)
	y, yok := b.Sys().(*syscall.Stat_t)
	return xok && yok && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) && x.Blocks == y.Blocks && x.Mode == y.Mode && x.Uid == y.Uid && x.Nlink == y.Nlink
}

func recordingUsageRootBound(root *os.Root, info os.FileInfo) bool {
	path, err := os.Lstat(root.Name())
	return err == nil && recordingJobDirectory(path) && os.SameFile(path, info)
}

func (s *recordingUsageScan) add(a *recording.StorageTotals, b recording.StorageTotals) bool {
	value, err := a.Add(b)
	if err != nil {
		return s.refuse("overflow")
	}
	*a = value
	return true
}

func recordingUsageName(name string, session bool) bool {
	if !session {
		switch name {
		case "job.json", "worker.lock", "claim.json", "ready.json", "session.json", "result.json":
			return true
		}
		return false
	}
	if name == "metadata.json" || name == "writer.lock" || name == "frames.obr" {
		return true
	}
	if len(name) != 19 || !strings.HasPrefix(name, "frames-") || !strings.HasSuffix(name, ".obr") {
		return false
	}
	value, err := strconv.ParseUint(name[7:15], 10, 32)
	return err == nil && value >= 2
}

func (s *recordingUsageScan) file(root *os.Root, name string) (recording.StorageTotals, bool) {
	var out recording.StorageTotals
	before, err := root.Lstat(name)
	if err != nil {
		return out, s.refuse("changed")
	}
	if !recordingJobFileInfo(before, int64(^uint64(0)>>1)) {
		return out, s.refuse("unsafe-storage")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return out, s.refuse("unavailable")
	}
	defer f.Close()
	info, e1 := f.Stat()
	after, e2 := root.Lstat(name)
	if e1 != nil || e2 != nil || !recordingJobFileInfo(info, int64(^uint64(0)>>1)) || !sameRecordingUsageStat(before, info) || !sameRecordingUsageStat(info, after) {
		return out, s.refuse("changed")
	}
	return s.count(info, false)
}

func (s *recordingUsageScan) directory(root *os.Root, session bool) (recording.StorageTotals, bool) {
	var out recording.StorageTotals
	dir, err := root.Open(".")
	if err != nil {
		return out, s.refuse("unavailable")
	}
	defer dir.Close()
	before, err := dir.Stat()
	if err != nil || !recordingJobDirectory(before) {
		return out, s.refuse("unsafe-storage")
	}
	out, ok := s.count(before, true)
	if !ok {
		return out, false
	}
	entries, err := dir.ReadDir(s.remaining + 1)
	if err != nil && err != io.EOF {
		return out, s.refuse("unavailable")
	}
	if len(entries) > s.remaining {
		return out, s.refuse("entry-limit")
	}
	for _, e := range entries {
		if !recordingUsageName(e.Name(), session) {
			return out, s.refuse("unsafe-storage")
		}
		total, ok := s.file(root, e.Name())
		if !ok || !s.add(&out, total) {
			return out, false
		}
	}
	after, err := dir.Stat()
	if err != nil || !sameRecordingUsageStat(before, after) {
		return out, s.refuse("changed")
	}
	return out, true
}

func measureRecordingUsage(ctx context.Context, jobs, sessions *os.Root, runner string, jobInstances, sessionInstances map[string]string, instances []recording.InstanceCommitment, limit int) recording.StorageUsage {
	out := recording.StorageUsage{Version: 1, RunnerID: runner, Status: "unknown", StartedAt: time.Now().UTC(), Basis: recordingUsageBasis}
	if len(instances) > recording.MaxUsageInstances {
		out.ReadAt = time.Now().UTC()
		out.Reason = "entry-limit"
		return out
	}
	scan := recordingUsageScan{ctx: ctx, remaining: limit}
	shared := recording.StorageTotals{}
	values := map[string]recording.StorageTotals{}
	for _, i := range instances {
		values[i.InstanceID] = recording.StorageTotals{}
	}
	for _, selection := range []struct {
		root     *os.Root
		bindings map[string]string
		session  bool
	}{{jobs, jobInstances, false}, {sessions, sessionInstances, true}} {
		if scan.reason != "" {
			break
		}
		dir, err := selection.root.Open(".")
		if err != nil {
			scan.refuse("unavailable")
			break
		}
		before, err := dir.Stat()
		if err != nil || !recordingJobDirectory(before) {
			dir.Close()
			scan.refuse("unsafe-storage")
			break
		}
		if !recordingUsageRootBound(selection.root, before) {
			dir.Close()
			scan.refuse("changed")
			break
		}
		total, ok := scan.count(before, true)
		if !ok || !scan.add(&shared, total) {
			dir.Close()
			break
		}
		entries, err := dir.ReadDir(scan.remaining + 1)
		if err != nil && err != io.EOF {
			dir.Close()
			scan.refuse("unavailable")
			break
		}
		if len(entries) > scan.remaining {
			dir.Close()
			scan.refuse("entry-limit")
			break
		}
		seen := map[string]bool{}
		for _, e := range entries {
			if !selection.session && e.Name() == "admission.lock" {
				value, ok := scan.file(selection.root, e.Name())
				if !ok || !scan.add(&shared, value) {
					break
				}
				continue
			}
			instance, ok := selection.bindings[e.Name()]
			if !ok {
				scan.refuse("changed")
				break
			}
			child, err := recordingJobChild(selection.root, e.Name())
			if err != nil {
				scan.refuse("unsafe-storage")
				break
			}
			value, ok := scan.directory(child, selection.session)
			bound := recordingJobChildBound(selection.root, e.Name(), child)
			child.Close()
			if !bound {
				scan.refuse("changed")
				break
			}
			if !ok {
				break
			}
			current, ok := values[instance]
			if !ok {
				scan.refuse("changed")
				break
			}
			if !scan.add(&current, value) {
				break
			}
			values[instance] = current
			seen[e.Name()] = true
		}
		after, err := dir.Stat()
		dir.Close()
		if err != nil || !sameRecordingUsageStat(before, after) || !recordingUsageRootBound(selection.root, after) || !selection.session && len(seen) != len(selection.bindings) {
			scan.refuse("changed")
		}
		// An allocated session may not have created its directory yet. Its job
		// still charges its full immutable maximum in admission commitments.
	}
	out.ReadAt = time.Now().UTC()
	if scan.reason != "" {
		out.Reason = scan.reason
		return out
	}
	total := shared
	measured := make([]recording.InstanceStorageUsage, 0, len(values))
	for id, value := range values {
		if !scan.add(&total, value) {
			out.Reason = scan.reason
			return out
		}
		measured = append(measured, recording.InstanceStorageUsage{InstanceID: id, Totals: value})
	}
	out.Instances = measured
	sort.Slice(out.Instances, func(i, j int) bool { return out.Instances[i].InstanceID < out.Instances[j].InstanceID })
	out.Status = "measured"
	out.Totals = &total
	out.Shared = &shared
	if out.Validate(runner, instances) != nil {
		out.Status = "unknown"
		out.Reason = "overflow"
		out.Totals = nil
		out.Shared = nil
		out.Instances = nil
	}
	return out
}
