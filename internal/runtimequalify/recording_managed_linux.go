//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	candidate "git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
)

var errRecordingSelection = errors.New("recording selection unavailable or changed; inspect the original launch receipt and retained coverage")

// RecordingOperation is supplied by the authenticated runner's durable launch
// job, never by an RPC accepting arbitrary metadata or filesystem paths. The
// manager must hold its lifecycle guard and bind the public target, revision and
// original request to RuntimeID. This constructor cannot establish that journal
// authority from a name or a digest. Project factories need their own contract.
type RecordingOperation struct {
	RunnerID, Mode, RuntimeID      string
	Target                         apppolicy.Target
	ConfigurationRevision          uint64
	OperationID, RequestID         string
	ContainerID, PreviousSessionID string
}

// PrepareManagedRecording prepares an ordinary, newly created non-TTY Vere
// attachment. It changes no runtime or recording files. The independent worker
// must use its own lifetime, store metadata before Attach, and acknowledge Ready
// before the manager separately admits Start through its normal durable fence.
// It is not wired into installed launch and grants no new authorization.
func PrepareManagedRecording(ctx context.Context, state ManagedState, op RecordingOperation, ready func(recording.Metadata) error) (recording.Metadata, recording.DockerCaptureSource, error) {
	return prepareManagedRecording(ctx, state, op, ready, defaults(), attachDockerRecording)
}

func prepareManagedRecording(ctx context.Context, state ManagedState, op RecordingOperation, ready func(recording.Metadata) error, d dependencies, attach func(context.Context, string, string) (io.ReadCloser, error)) (recording.Metadata, recording.DockerCaptureSource, error) {
	refuse := func() (recording.Metadata, recording.DockerCaptureSource, error) {
		return recording.Metadata{}, recording.DockerCaptureSource{}, errRecordingSelection
	}
	if ready == nil || attach == nil || !state.Prepared || op.RuntimeID != state.ID || op.Target.Validate() != nil || op.Target.InstanceID == "" || op.Target.ProjectID != "" || op.Target.Desk != "" || !containerIDPattern.MatchString(op.ContainerID) || state.Storage == nil || state.Storage.Uncertain || state.Fake != nil && state.Fake.Uncertain || state.ImportRestore != nil || state.Deletion != nil || state.Handoff.Fenced() || state.MoveTarget != "" {
		return refuse()
	}
	// Neither a caller's later pointer edits nor inspect's bookkeeping may change
	// the retained launch selection. No private state is returned or written.
	raw, err := json.Marshal(state)
	var pinned ManagedState
	if err != nil || json.Unmarshal(raw, &pinned) != nil {
		return refuse()
	}
	state = pinned
	q, err := managedSession(&state, nil, d)
	if err != nil || q.accountBoundary(false) != nil || q.verifyEngine(ctx) != nil {
		return refuse()
	}
	engine := q.engineEvidence.ID
	index := -1
	for i, c := range q.r.Containers {
		if c.Removed {
			if c.CreateUncertain {
				return refuse()
			}
			continue
		}
		if index >= 0 || c.ID != op.ContainerID || !c.CreateDispatched || c.CreateUncertain || c.Kind == "probe" {
			return refuse()
		}
		index = i
	}
	if index < 0 {
		return refuse()
	}
	kind := q.r.Containers[index].Kind
	boot, namespaces, err := recordingHostContext(q)
	if err != nil {
		return refuse()
	}
	pierBasis := "existing-pier"
	storageDigest := ""
	checkStorage := func(beforeStart bool) error {
		mount, err := pierstore.CheckMount(d.home, *state.Storage)
		if err != nil || mount != q.dataMount().Source {
			return errRecordingSelection
		}
		if state.Initialized {
			if _, err = pierstore.Check(d.home, *state.Storage); err != nil {
				return errRecordingSelection
			}
			return nil
		}
		if state.Storage.Identity != (pierstore.Identity{}) {
			return errRecordingSelection
		}
		if beforeStart {
			if _, err = os.Lstat(q.pierPath()); !errors.Is(err, os.ErrNotExist) {
				return errRecordingSelection
			}
		}
		return nil
	}
	if checkStorage(true) != nil {
		return refuse()
	}
	if state.Initialized {
		storageDigest = pierstore.Fingerprint(state.Storage.Identity)
	} else {
		pierBasis = "prepared-storage"
		storageDigest, err = recordingDigest("omabit-recording-prepared-storage-v1", struct {
			Path  string
			Mount pierstore.DirectoryIdentity
		}{state.Storage.Path, state.Storage.MountIdentity})
		if err != nil {
			return refuse()
		}
	}
	read := func(ctx context.Context, beforeStart bool) (container, error) {
		if q.verifyEngineAccess(ctx, !beforeStart) != nil || q.engineEvidence.ID != engine || checkStorage(beforeStart) != nil {
			return container{}, errRecordingSelection
		}
		if beforeStart && (q.usesPillFile(kind) && q.verifyLocalPill(ctx) != nil || kind == "comet-boot" && state.NativeShip != "" && q.verifyBootKey() != nil) {
			return container{}, errRecordingSelection
		}
		currentBoot, currentNamespaces, err := recordingHostContext(q)
		if err != nil || currentBoot != boot || currentNamespaces != namespaces {
			return container{}, errRecordingSelection
		}
		c, err := q.inspect(ctx, index)
		if err != nil || c.ID != op.ContainerID || q.containment(c, kind) != nil {
			return container{}, errRecordingSelection
		}
		if isNetworkedKind(kind) {
			if _, err := q.inspectNetwork(ctx); err != nil {
				return container{}, errRecordingSelection
			}
		}
		if beforeStart && (c.State.Status != "created" || active(c) || c.State.Pid != 0 || c.State.Dead || *c.State.OOMKilled || c.State.Error != "" || *c.State.ExitCode != 0 || !recordingZeroTime(c.State.StartedAt) || !recordingZeroTime(c.State.FinishedAt)) {
			return container{}, errRecordingSelection
		}
		afterBoot, afterNamespaces, err := recordingHostContext(q)
		if err != nil || afterBoot != boot || afterNamespaces != namespaces || checkStorage(beforeStart) != nil {
			return container{}, errRecordingSelection
		}
		return c, nil
	}
	selected, err := read(ctx, true)
	if err != nil {
		return refuse()
	}
	// The complete accepted inspect result is compared while still created.
	// Runtime-dependent process/port observations may change only after Start.
	checkPrepared := func(ctx context.Context) error {
		current, err := read(ctx, true)
		if err != nil || !reflect.DeepEqual(current, selected) || q.verifyEngine(ctx) != nil || q.engineEvidence.ID != engine || checkStorage(true) != nil {
			return errRecordingSelection
		}
		return nil
	}
	if checkPrepared(ctx) != nil || q.r.RepoDigest != candidate.Image && q.r.RepoDigest != strings.TrimPrefix(candidate.Image, "docker.io/") || q.r.VereVersion != candidate.VereVersion {
		return refuse()
	}
	contractDigest, err := recordingDigest("omabit-recording-managed-launch-v1", struct {
		Container  container
		PillSHA256 string
		PillBytes  int64
		KeySHA256  string
	}{selected, q.r.PillSHA256, q.r.PillBytes, state.NativeKeyDigest})
	if err != nil {
		return refuse()
	}
	ship, status := managedDeskShip(state), "registered"
	if state.Comet != nil && state.NativeShip == "" && state.Comet.Ship == "" {
		ship, status = "", "pending-comet-mining"
	}
	session, err := recording.NewSessionID()
	if err != nil {
		return refuse()
	}
	endpoint := sha256.Sum256([]byte(q.socket))
	m := recording.Metadata{Version: recording.CurrentMetadataVersion, SessionID: session, PreviousSessionID: op.PreviousSessionID, RunnerID: op.RunnerID, Mode: op.Mode, InstanceID: op.Target.InstanceID, ConfigurationRevision: op.ConfigurationRevision, OperationID: op.OperationID, RequestID: op.RequestID, ContainerID: op.ContainerID, LaunchContractSHA256: contractDigest, ImageManifestDigest: candidate.ManifestDigest, RuntimeVersion: q.r.VereVersion, Transport: "docker-multiplex", AttachedAt: time.Now().UTC(), Identity: &recording.AttachmentIdentity{Target: op.Target, NativeShip: ship, NativeShipStatus: status, PierBinding: pierBasis, PierContinuitySHA256: storageDigest, EngineID: engine, EngineEndpointSHA256: hex.EncodeToString(endpoint[:]), HostBootID: boot, RecorderNamespaces: namespaces}}
	if m.Validate() != nil {
		return refuse()
	}
	// Pin a separate canonical copy for callbacks; returned nested metadata is
	// deliberately not retained by reference.
	canonical, _ := m.Bytes()
	bound, _ := recording.ReadMetadata(canonical)
	source := recording.DockerCaptureSource{
		Attach: func(ctx context.Context) (io.ReadCloser, error) {
			if checkPrepared(ctx) != nil {
				return nil, errRecordingSelection
			}
			stream, err := attach(ctx, q.socket, op.ContainerID)
			if err != nil || stream == nil || checkPrepared(ctx) != nil {
				if stream != nil {
					stream.Close()
				}
				return nil, errRecordingSelection
			}
			return stream, nil
		},
		Ready: func(got recording.Metadata) error {
			expected := bound
			expected.AttachedAt = got.AttachedAt // CaptureDocker owns its epoch.
			want, _ := expected.Bytes()
			actual, err := got.Bytes()
			if err != nil || !bytes.Equal(want, actual) || checkPrepared(ctx) != nil {
				return errRecordingSelection
			}
			copy, _ := recording.ReadMetadata(actual)
			return ready(copy)
		},
		ObserveExit: func(ctx context.Context) (*recording.ExitEvent, error) {
			first, err := read(ctx, false)
			if err != nil || first.State.Status != "exited" || active(first) || first.State.Pid != 0 || !recordingSameLaunch(first, selected) {
				return nil, errRecordingSelection
			}
			last, err := read(ctx, false)
			if err != nil || !reflect.DeepEqual(first, last) || q.verifyEngineAccess(ctx, true) != nil || q.engineEvidence.ID != engine {
				return nil, errRecordingSelection
			}
			started, e1 := recordingProcessTime(first.State.StartedAt)
			finished, e2 := recordingProcessTime(first.State.FinishedAt)
			event := &recording.ExitEvent{Version: 1, ContainerID: op.ContainerID, StartedUnixNano: started, FinishedUnixNano: finished, Code: *first.State.ExitCode, OOMKilled: *first.State.OOMKilled, Dead: first.State.Dead, DockerErrorPresent: first.State.Error != ""}
			if e1 != nil || e2 != nil || event.Validate() != nil {
				return nil, errRecordingSelection
			}
			return event, nil
		},
	}
	return m, source, nil
}

func recordingZeroTime(raw string) bool {
	if raw == "" {
		return false // Missing evidence is not a Docker zero timestamp.
	}
	at, err := time.Parse(time.RFC3339Nano, raw)
	return err == nil && at.IsZero()
}

func recordingProcessTime(raw string) (int64, error) {
	at, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || at.Before(time.Unix(0, 1)) || at.After(time.Unix(0, math.MaxInt64)) {
		return 0, errRecordingSelection
	}
	return at.UnixNano(), nil
}

func recordingSameLaunch(a, b container) bool {
	return a.ID == b.ID && a.Name == b.Name && a.Image == b.Image && reflect.DeepEqual(a.Config, b.Config) && reflect.DeepEqual(a.HostConfig, b.HostConfig) && reflect.DeepEqual(a.Mounts, b.Mounts)
}

func recordingDigest(domain string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(domain+"\x00"), raw...))
	return hex.EncodeToString(sum[:]), nil
}

func recordingHostContext(q *qualifier) (string, recording.HostNamespaces, error) {
	boot, err := deskProcessBootReader(q)()
	if err != nil {
		return "", recording.HostNamespaces{}, errRecordingSelection
	}
	var ns recording.HostNamespaces
	for name, out := range map[string]*uint64{"user": &ns.User, "mnt": &ns.Mount, "pid": &ns.PID, "net": &ns.Network} {
		info, err := os.Stat(filepath.Join("/proc/self/ns", name))
		if err != nil {
			return "", recording.HostNamespaces{}, errRecordingSelection
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Ino == 0 {
			return "", recording.HostNamespaces{}, errRecordingSelection
		}
		*out = stat.Ino
	}
	return boot, ns, nil
}
