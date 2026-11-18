//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// RecordingWorkerUnitName isolates exact jobs across runner identities, modes
// and installation roots even when they share one user's systemd manager.
func RecordingWorkerUnitName(job ManagedRecordingJob) (string, error) {
	if job.Validate() != nil {
		return "", ErrRecordingWorker
	}
	namespace, err := recordingDigest("omabit-recording-unit-v1", struct{ RunnerID, Mode, Directory string }{job.Operation.RunnerID, job.Operation.Mode, job.State.Runtime.Directory})
	if err != nil {
		return "", ErrRecordingWorker
	}
	return "omabit-recording-" + namespace[:16] + "-" + job.ID + ".service", nil
}

type RecordingSupervisorState struct {
	Unit         string
	MainPID      uint32
	InvocationID string
	ControlGroup string
}

// ParseRecordingSupervisorState accepts only the exact selected unit's bounded
// systemctl property response. It grants no authority and performs no probe.
func ParseRecordingSupervisorState(job ManagedRecordingJob, raw []byte) (RecordingSupervisorState, error) {
	return parseRecordingSupervisorState(job, raw, true)
}

func parseRecordingSupervisorState(job ManagedRecordingJob, raw []byte, ready bool) (RecordingSupervisorState, error) {
	want, err := RecordingWorkerUnitName(job)
	if err != nil || len(raw) > 8192 {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	allowed := map[string]bool{"Id": true, "Type": true, "NotifyAccess": true, "ActiveState": true, "SubState": true, "MainPID": true, "InvocationID": true, "ControlGroup": true}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || !allowed[key] || strings.ContainsAny(value, "\x00\r\n") {
			return RecordingSupervisorState{}, ErrRecordingWorker
		}
		if _, duplicate := values[key]; duplicate {
			return RecordingSupervisorState{}, ErrRecordingWorker
		}
		values[key] = value
	}
	lifecycle := values["ActiveState"] == "active" && values["SubState"] == "running"
	if !ready {
		lifecycle = values["ActiveState"] == "activating" && values["SubState"] == "start"
	}
	if len(values) != len(allowed) || values["Id"] != want || values["Type"] != "notify" || values["NotifyAccess"] != "main" || !lifecycle || !recordingJobHex(values["InvocationID"], 32) {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	pid, err := strconv.ParseUint(values["MainPID"], 10, 32)
	group := values["ControlGroup"]
	if err != nil || pid < 2 || strconv.FormatUint(pid, 10) != values["MainPID"] || !recordingJobPath(group) || !strings.HasSuffix(group, "/"+want) {
		return RecordingSupervisorState{}, ErrRecordingWorker
	}
	return RecordingSupervisorState{Unit: want, MainPID: uint32(pid), InvocationID: values["InvocationID"], ControlGroup: group}, nil
}

func (s *RecordingJobs) ReadWorker(id, digest string) (RecordingWorkerIdentity, error) {
	root, err := recordingJobChild(s.root, id)
	if err != nil {
		return RecordingWorkerIdentity{}, err
	}
	defer root.Close()
	job, err := s.read(root, id, digest)
	if err != nil {
		return RecordingWorkerIdentity{}, err
	}
	data, err := recordingJobRead(root, "claim.json", 4096)
	var claim recordingJobClaim
	if err != nil || json.Unmarshal(data, &claim) != nil || !claim.valid() || claim.JobID != id || claim.JobSHA256 != digest || claim.Worker.ExecutableSHA256 != job.ExecutableSHA256 {
		return RecordingWorkerIdentity{}, ErrRecordingJobStorage
	}
	canonical, _ := json.Marshal(claim)
	if !bytes.Equal(data, canonical) {
		return RecordingWorkerIdentity{}, ErrRecordingJobStorage
	}
	return claim.Worker, nil // legacy omissions remain zero/unknown
}

// CheckRecordingSupervisor binds independently obtained current facts, not
// cached or caller-supplied RPC assertions. Old claim1 cannot pass. Callers must
// obtain fresh unit/proc observations and revalidate before Start under their
// lifecycle guard; this comparison starts, signals and authorizes nothing.
func CheckRecordingSupervisor(job ManagedRecordingJob, unit RecordingSupervisorState, claimed, current RecordingWorkerIdentity) error {
	want, err := RecordingWorkerUnitName(job)
	if err != nil || !claimed.valid() || !current.valid() || claimed.InvocationID == "" || claimed.ExecutableSHA256 != job.ExecutableSHA256 || unit.Unit != want || unit.MainPID != claimed.PID || unit.InvocationID != claimed.InvocationID || current != claimed || !recordingJobPath(unit.ControlGroup) || !strings.HasSuffix(unit.ControlGroup, "/"+want) {
		return ErrRecordingWorker
	}
	return nil
}
