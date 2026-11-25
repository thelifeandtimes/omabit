//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

const MaxRecordingJobBytes = 64 << 10

var ErrRecordingJob = errors.New("private recording job is unsupported, incomplete or rebound; retain the original job and launch receipt")

// ManagedRecordingJob is a private, versioned control handoff, not an RPC input,
// public diagnostic or authorization grant. The authenticated manager supplies
// the actual original operation and its own native pre-Start admission. A worker
// must verify its executable and independently repeat native admission before
// attachment. No job storage, supervisor or installed launch is enabled here.
type ManagedRecordingJob struct {
	Version          int                `json:"version"`
	ID               string             `json:"id"`
	CreatedAt        time.Time          `json:"created_at"`
	ExecutableSHA256 string             `json:"executable_sha256"`
	Actor            string             `json:"actor"`
	Operation        RecordingOperation `json:"operation"`
	AdmissionSHA256  string             `json:"admission_sha256"`
	State            ManagedState       `json:"state"`
	Limits           RecordingJobLimits `json:"limits"`
}

type RecordingJobLimits struct {
	MaxSessionBytes  int64  `json:"max_session_bytes"`
	MaxSegmentBytes  int64  `json:"max_segment_bytes"`
	ReserveFreeBytes uint64 `json:"reserve_free_bytes"`
}

// NewManagedRecordingJob copies only the native launch inputs needed by the
// ordinary factory. Signing keys, retained logs, runtime diagnostics, user
// endpoints, import history and unrelated maintenance state are not copied.
// A matching digest associates evidence; only the manager's journal/guard can
// authorize Start. This function performs no filesystem or engine operation.
func NewManagedRecordingJob(state ManagedState, op RecordingOperation, admitted recording.Metadata, executable, actor string, at time.Time, limits RecordingJobLimits) (ManagedRecordingJob, error) {
	if state.ImportRestore != nil || state.Deletion != nil || state.Handoff.Fenced() || state.MoveTarget != "" || state.Storage == nil || state.Storage.Uncertain || state.Fake != nil && state.Fake.Uncertain {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	digest, err := recordingAdmissionDigest(admitted)
	if err != nil || !recordingOperationMatches(op, admitted) {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	id, err := recording.NewSessionID()
	if err != nil {
		return ManagedRecordingJob{}, err
	}
	job := ManagedRecordingJob{Version: 1, ID: id, CreatedAt: at.UTC(), ExecutableSHA256: executable, Actor: actor, Operation: op, AdmissionSHA256: digest, State: recordingLaunchState(state), Limits: limits}
	raw, err := job.Bytes()
	if err != nil {
		return ManagedRecordingJob{}, err
	}
	return ReadManagedRecordingJob(raw) // independent nested state pointers
}

func (j ManagedRecordingJob) Validate() error {
	op, state := j.Operation, j.State
	if j.Version != 1 || !recordingJobHex(j.ID, 32) || !recordingJobHex(j.ExecutableSHA256, 64) || !recordingJobHex(j.AdmissionSHA256, 64) || j.CreatedAt.Before(time.Unix(0, 1)) || j.CreatedAt.After(time.Unix(0, math.MaxInt64)) || j.Actor == "" || len(j.Actor) > 128 || strings.IndexFunc(j.Actor, func(r rune) bool { return r < 33 || r > 126 }) >= 0 {
		return ErrRecordingJob
	}
	if !protocol.ValidID(op.RunnerID) || op.Mode != "development-managed" && op.Mode != "development-managed-fake" || op.Target.Validate() != nil || op.Target.InstanceID == "" || op.Target.ProjectID != "" || op.Target.Desk != "" || op.ConfigurationRevision == 0 || !protocol.ValidID(op.OperationID) || !protocol.ValidID(op.RequestID) || !recordingJobHex(op.ContainerID, 64) || op.PreviousSessionID != "" && !recordingJobHex(op.PreviousSessionID, 32) {
		return ErrRecordingJob
	}
	if !state.Prepared || state.ID != op.RuntimeID || !runIDPattern.MatchString(state.ID) || state.Storage == nil || !pierstore.ValidPath(state.Storage.Path) || state.Storage.Uncertain || !reflect.DeepEqual(state, recordingLaunchState(state)) || !recordingJobPath(state.Runtime.Directory) || state.Runtime.RunID != state.ID || state.Runtime.PillPath != filepath.Join(state.Runtime.Directory, candidate.PillName) || len(state.Runtime.Containers) == 0 || len(state.Runtime.Containers) > 3 {
		return ErrRecordingJob
	}
	// These values have a finite, credential-free source contract. A copied
	// saved URL or arbitrary diagnostic must not become a new control payload.
	if _, err := candidate.Resolve(state.RuntimeProfile); err != nil || state.Runtime.RuntimeProfile != state.RuntimeProfile {
		return ErrRecordingJob
	}
	q := &qualifier{d: dependencies{runtimeProfile: state.RuntimeProfile, boot: state.Boot, bootstrap: state.Bootstrap}}
	for _, c := range state.Runtime.Containers {
		if c.RuntimeProfile != state.RuntimeProfile {
			return ErrRecordingJob
		}
	}
	if state.Runtime.Image != q.runtimeProfile().Image || state.Runtime.RepoDigest != q.runtimeProfile().Image && state.Runtime.RepoDigest != strings.TrimPrefix(q.runtimeProfile().Image, "docker.io/") || !q.runtimeProfile().AcceptsImageID(state.Runtime.ImageID) || state.Runtime.VereVersion != q.runtimeProfile().VereVersion || state.Runtime.PillURL != q.pillSource() {
		return ErrRecordingJob
	}
	if j.Limits.MaxSessionBytes < 1<<20 || j.Limits.MaxSessionBytes > 1<<30 || j.Limits.MaxSegmentBytes < 4096 || j.Limits.MaxSegmentBytes > j.Limits.MaxSessionBytes || j.Limits.ReserveFreeBytes == 0 {
		return ErrRecordingJob
	}
	return nil
}

func recordingJobPath(path string) bool {
	return len(path) <= 4096 && filepath.IsAbs(path) && filepath.Clean(path) == path && strings.IndexFunc(path, func(r rune) bool { return r < 32 || r == 127 }) < 0
}

func (j ManagedRecordingJob) Bytes() ([]byte, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}
	j.CreatedAt = j.CreatedAt.UTC()
	raw, err := json.Marshal(j)
	if err != nil || len(raw) > MaxRecordingJobBytes {
		return nil, ErrRecordingJob
	}
	return raw, nil
}

func ReadManagedRecordingJob(raw []byte) (ManagedRecordingJob, error) {
	var j ManagedRecordingJob
	if len(raw) > MaxRecordingJobBytes || json.Unmarshal(raw, &j) != nil {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	canonical, err := j.Bytes()
	if err != nil || !bytes.Equal(raw, canonical) {
		return ManagedRecordingJob{}, ErrRecordingJob
	}
	return j, nil
}

// MatchesAdmission ignores only the new session/receive-clock origin and the
// independent worker's own namespaces. Those fields must be obtained by that
// worker, never copied from the parent or invented before the process exists.
// Every other metadata field remains bound, including host boot and engine,
// endpoint, native-name/storage basis, launch, artifacts and original operation.
// A match is not authentication, a fresh native probe, or permission to Start.
func (j ManagedRecordingJob) MatchesAdmission(actual recording.Metadata) bool {
	if j.Validate() != nil || !recordingOperationMatches(j.Operation, actual) {
		return false
	}
	digest, err := recordingAdmissionDigest(actual)
	return err == nil && digest == j.AdmissionSHA256
}

func recordingOperationMatches(op RecordingOperation, m recording.Metadata) bool {
	return m.Validate() == nil && m.Version == 4 && m.Identity != nil && m.ProjectID == "" && op.RuntimeID != "" && op.RunnerID == m.RunnerID && op.Mode == m.Mode && op.Target == m.Identity.Target && op.ConfigurationRevision == m.ConfigurationRevision && op.OperationID == m.OperationID && op.RequestID == m.RequestID && op.ContainerID == m.ContainerID && op.PreviousSessionID == m.PreviousSessionID
}

func recordingAdmissionDigest(m recording.Metadata) (string, error) {
	if m.Validate() != nil || m.Version != 4 || m.Identity == nil {
		return "", ErrRecordingJob
	}
	// This hash descriptor is not attachment metadata. Deliberately enumerate
	// its fields; future metadata versions need a separately reviewed contract.
	i := *m.Identity
	i.RecorderNamespaces = recording.HostNamespaces{} // excluded parent context
	return recordingDigest("omabit-recording-worker-admission-v1", struct {
		Version               int
		PreviousSessionID     string
		RunnerID              string
		Mode                  string
		InstanceID            string
		ConfigurationRevision uint64
		OperationID           string
		RequestID             string
		ContainerID           string
		LaunchContractSHA256  string
		ImageManifestDigest   string
		RuntimeVersion        string
		Transport             string
		Identity              recording.AttachmentIdentity
	}{4, m.PreviousSessionID, m.RunnerID, m.Mode, m.InstanceID, m.ConfigurationRevision, m.OperationID, m.RequestID, m.ContainerID, m.LaunchContractSHA256, m.ImageManifestDigest, m.RuntimeVersion, m.Transport, i})
}

func recordingJobHex(raw string, size int) bool {
	if len(raw) != size || raw == strings.Repeat("0", size) {
		return false
	}
	for _, r := range raw {
		if r < '0' || r > '9' && r < 'a' || r > 'f' {
			return false
		}
	}
	return true
}

func recordingLaunchState(s ManagedState) ManagedState {
	r := s.Runtime
	selected := Result{RuntimeProfile: r.RuntimeProfile, Profile: r.Profile, RunID: r.RunID, Directory: r.Directory, Image: r.Image, ImageID: r.ImageID, RepoDigest: r.RepoDigest, Platform: r.Platform, EngineVersion: r.EngineVersion, VereVersion: r.VereVersion, PillURL: r.PillURL, PillSHA256: r.PillSHA256, PillBytes: r.PillBytes, PillPath: r.PillPath}
	for _, c := range r.Containers {
		selected.Containers = append(selected.Containers, ContainerRecord{RuntimeProfile: c.RuntimeProfile, MemoryGiB: c.MemoryGiB, CPUCores: c.CPUCores, EngineVersion: c.EngineVersion, Loom: c.Loom, NativeControl: c.NativeControl, Name: c.Name, ID: c.ID, Kind: c.Kind, SnapshotMinutes: c.SnapshotMinutes, Removed: c.Removed, CreateDispatched: c.CreateDispatched, CreateUncertain: c.CreateUncertain})
	}
	if r.Network != nil {
		n := *r.Network
		selected.Network = &n
	}
	return ManagedState{RuntimeProfile: s.RuntimeProfile, MemoryGiB: s.MemoryGiB, CPUCores: s.CPUCores, Loom: s.Loom, NativeControl: s.NativeControl, Bootstrap: s.Bootstrap, ID: s.ID, Port: s.Port, Prepared: s.Prepared, Initialized: s.Initialized, Runtime: selected, Comet: s.Comet, Fake: s.Fake, Storage: s.Storage, NativeShip: s.NativeShip, NativeKeyDigest: s.NativeKeyDigest, Boot: s.Boot, AmesPort: s.AmesPort}
}
