//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

const recordingSupervisorProperties = "Id,Type,NotifyAccess,ActiveState,SubState,MainPID,InvocationID,ControlGroup"

// ObserveRecordingSupervisor reads only the selected job's unit and exact main
// process, bracketing the process observation with matching active unit reads.
// It sends no lifecycle command. The caller still owns the authenticated launch
// guard and must repeat this observation immediately before its fenced Start.
func ObserveRecordingSupervisor(ctx context.Context, job ManagedRecordingJob, claimed RecordingWorkerIdentity) (RecordingSupervisorState, error) {
	q := &qualifier{d: defaults()}
	if q.accountBoundary(false) != nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	return observeRecordingSupervisor(ctx, job, claimed, true, readRecordingSupervisor, observeRecordingProcess)
}

func observeRecordingSupervisor(ctx context.Context, job ManagedRecordingJob, claimed RecordingWorkerIdentity, ready bool, unitRead func(context.Context, string) ([]byte, error), processRead func(context.Context, RecordingSupervisorState) (RecordingWorkerIdentity, string, error)) (RecordingSupervisorState, error) {
	want, err := RecordingWorkerUnitName(job)
	if err != nil || !claimed.valid() || claimed.InvocationID == "" || claimed.ExecutableSHA256 != job.ExecutableSHA256 || unitRead == nil || processRead == nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := unitRead(ctx, want)
	if err != nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	first, err := parseRecordingSupervisorState(job, raw, ready)
	if err != nil || first.MainPID != claimed.PID || first.InvocationID != claimed.InvocationID {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	current, group, err := processRead(ctx, first)
	if err != nil || group != first.ControlGroup || CheckRecordingSupervisor(job, first, claimed, current) != nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	raw, err = unitRead(ctx, want)
	if err != nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	last, err := parseRecordingSupervisorState(job, raw, ready)
	if err != nil || last != first || ctx.Err() != nil {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	return last, nil
}

// This fixed command never accepts shell text, environment overrides, extra
// properties or a caller-selected executable. Bound output before allocation.
func readRecordingSupervisor(ctx context.Context, unit string) ([]byte, error) {
	q := &qualifier{d: defaults()}
	if q.accountBoundary(false) != nil || q.ancestors("/usr/bin/systemctl") != nil {
		return nil, ErrRecordingWorker
	}
	n, err := q.d.stat("/usr/bin/systemctl")
	if err != nil || !n.mode.IsRegular() || n.uid != 0 || n.mode.Perm()&0022 != 0 || n.mode.Perm()&0111 == 0 || n.mode&(os.ModeSetuid|os.ModeSetgid) != 0 || !validRecordingUnitName(unit) {
		return nil, ErrRecordingWorker
	}
	runtime := filepath.Join("/run/user", strconv.Itoa(os.Geteuid()))
	cmd := exec.CommandContext(ctx, "/usr/bin/systemctl", "--user", "show", "--property="+recordingSupervisorProperties, "--", unit)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C", "XDG_RUNTIME_DIR=" + runtime, "DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtime + "/bus"}
	cmd.WaitDelay = time.Second
	var out recordingSupervisorOutput
	cmd.Stdout, cmd.Stderr = &out, io.Discard
	if cmd.Run() != nil || ctx.Err() != nil {
		return nil, ErrRecordingWorker
	}
	return out.Bytes(), nil
}

func validRecordingUnitName(unit string) bool {
	parts := strings.Split(strings.TrimSuffix(unit, ".service"), "-")
	return strings.HasSuffix(unit, ".service") && len(parts) == 4 && parts[0] == "omabit" && parts[1] == "recording" && recordingJobHex(parts[2], 16) && recordingJobHex(parts[3], 32)
}

type recordingSupervisorOutput struct{ bytes.Buffer }

func (b *recordingSupervisorOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 8192 {
		return 0, ErrRecordingWorker
	}
	return b.Buffer.Write(p)
}

func recordingProcRead(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, ErrRecordingWorker
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(raw)) > max {
		return nil, ErrRecordingWorker
	}
	return raw, nil
}

func recordingProcessStat(raw []byte, pid uint32) (uint64, error) {
	prefix := strconv.FormatUint(uint64(pid), 10) + " ("
	end := bytes.LastIndexByte(raw, ')')
	if !bytes.HasPrefix(raw, []byte(prefix)) || end < len(prefix) || end+1 >= len(raw) {
		return 0, ErrRecordingWorker
	}
	fields := strings.Fields(string(raw[end+1:]))
	if len(fields) < 20 || len(fields[0]) != 1 || strings.ContainsAny(fields[0], "ZXTtx") {
		return 0, ErrRecordingWorker
	}
	return recordingWorkerStartTicks(raw)
}

func recordingProcessCgroup(raw []byte) (string, error) {
	// Unified cgroup2 only. Multiple controllers or namespace-relative roots
	// cannot prove membership of the selected user manager's exact service.
	value := strings.TrimSuffix(string(raw), "\n")
	group, ok := strings.CutPrefix(value, "0::")
	if !ok || !recordingJobPath(group) || group == "/" || strings.Contains(group, "\n") {
		return "", ErrRecordingWorker
	}
	return group, nil
}

func observeRecordingProcess(ctx context.Context, unit RecordingSupervisorState) (RecordingWorkerIdentity, string, error) {
	if unit.MainPID < 2 || !recordingJobHex(unit.InvocationID, 32) || !recordingJobPath(unit.ControlGroup) || ctx.Err() != nil {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	base := filepath.Join("/proc", strconv.FormatUint(uint64(unit.MainPID), 10))
	before, err := os.Stat(base)
	if err != nil || !recordingProcessOwner(before) {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	raw, err := recordingProcRead(filepath.Join(base, "stat"), 4096)
	ticks, statErr := recordingProcessStat(raw, unit.MainPID)
	if err != nil || statErr != nil {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	f, err := os.Open(filepath.Join(base, "exe"))
	if err != nil {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	defer f.Close()
	image, err := f.Stat()
	if err != nil || !image.Mode().IsRegular() || image.Size() <= 0 || image.Size() > 128<<20 {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	h := sha256.New()
	n, err := io.Copy(h, &recordingProcessReader{ctx: ctx, r: io.LimitReader(f, 128<<20+1)})
	if err != nil || n != image.Size() {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	boot, err := recordingProcRead("/proc/sys/kernel/random/boot_id", 64)
	if err != nil {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	var ns recording.HostNamespaces
	for name, into := range map[string]*uint64{"user": &ns.User, "mnt": &ns.Mount, "pid": &ns.PID, "net": &ns.Network} {
		info, err := os.Stat(filepath.Join(base, "ns", name))
		if err != nil {
			return RecordingWorkerIdentity{}, "", ErrRecordingWorker
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Ino == 0 {
			return RecordingWorkerIdentity{}, "", ErrRecordingWorker
		}
		*into = st.Ino
	}
	rawGroup, err := recordingProcRead(filepath.Join(base, "cgroup"), 4096)
	group, groupErr := recordingProcessCgroup(rawGroup)
	if err != nil || groupErr != nil || group != unit.ControlGroup {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	raw, err = recordingProcRead(filepath.Join(base, "stat"), 4096)
	lastTicks, lastErr := recordingProcessStat(raw, unit.MainPID)
	after, dirErr := os.Stat(base)
	lastImage, imageErr := os.Stat(filepath.Join(base, "exe"))
	if err != nil || lastErr != nil || lastTicks != ticks || dirErr != nil || !recordingProcessOwner(after) || !os.SameFile(before, after) || imageErr != nil || !os.SameFile(image, lastImage) || ctx.Err() != nil {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	worker := RecordingWorkerIdentity{ExecutableSHA256: hex.EncodeToString(h.Sum(nil)), PID: unit.MainPID, StartTicks: ticks, HostBootID: strings.TrimSuffix(string(boot), "\n"), Namespaces: ns, InvocationID: unit.InvocationID}
	if !worker.valid() {
		return RecordingWorkerIdentity{}, "", ErrRecordingWorker
	}
	return worker, group, nil
}

func recordingProcessOwner(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.IsDir() && s.Uid == uint32(os.Geteuid()) && os.Geteuid() != 0
}

type recordingProcessReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *recordingProcessReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
