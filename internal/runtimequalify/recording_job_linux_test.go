//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func recordingJobFixture(t *testing.T) (ManagedRecordingJob, ManagedState, RecordingOperation, recording.Metadata) {
	t.Helper()
	f, state, op, _ := preparedRecordingFixture(t, true, false)
	m, _, err := prepareManagedRecording(t.Context(), state, op, func(recording.Metadata) error { t.Fatal("model created readiness"); return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
		t.Fatal("model attached output")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	j, err := NewManagedRecordingJob(state, op, m, strings.Repeat("e", 64), "local-uid:1000", time.Unix(1700000000, 123).UTC(), RecordingJobLimits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return j, state, op, m
}

func TestRecordingJobPreservesBindingWithoutCopyingPrivateBodies(t *testing.T) {
	j, state, op, m := recordingJobFixture(t)
	signing, originalLog, endpoint := "PRIVATE-BOOTSTRAP-SIGNING-KEY", "PRIVATE-ORIGINAL-LOG-BODY", "PRIVATE-ENDPOINT-DIAGNOSTIC"
	state.HandoffSigningKey = []byte(signing)
	state.Runtime.RetainedLogs = &protocol.RuntimeLogs{Lines: []string{originalLog}}
	state.Runtime.Failure = originalLog
	for i := range state.Runtime.Containers {
		state.Runtime.Containers[i].HostEndpoint = endpoint
		state.Runtime.Containers[i].LastState = &ContainerState{LaunchError: originalLog}
	}
	before, _ := json.Marshal(state)
	got, err := NewManagedRecordingJob(state, op, m, j.ExecutableSHA256, j.Actor, j.CreatedAt, j.Limits)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := got.Bytes()
	if err != nil || bytes.Contains(raw, []byte(signing)) || bytes.Contains(raw, []byte(originalLog)) || bytes.Contains(raw, []byte(endpoint)) || bytes.Contains(raw, []byte("handoff_signing_key")) || bytes.Contains(raw, []byte("retained_logs")) {
		t.Fatal("private key/output/diagnostic body entered worker handoff", err)
	}
	after, _ := json.Marshal(state)
	if !bytes.Equal(before, after) || got.Operation.ConfigurationRevision != op.ConfigurationRevision || !got.MatchesAdmission(m) || got.State.Comet == nil || got.State.Comet.Ship != "" {
		t.Fatal("job changed original state, lost uint64 binding or invented mining identity")
	}
	state.Storage.Path = "livenet/comets/unrelated/pier"
	state.Comet.Ship = "~zod"
	state.Runtime.Containers[0].ID = strings.Repeat("f", 64)
	if got.State.Storage.Path == state.Storage.Path || got.State.Comet.Ship != "" || got.State.Runtime.Containers[0].ID == state.Runtime.Containers[0].ID {
		t.Fatal("job retained mutable caller pointers")
	}
	decoded, err := ReadManagedRecordingJob(raw)
	if err != nil || !reflect.DeepEqual(decoded, got) {
		t.Fatal("private job canonical roundtrip changed binding", err)
	}
}

func TestRecordingJobWorkerMustReobserveSameAdmission(t *testing.T) {
	j, _, _, m := recordingJobFixture(t)
	raw, _ := m.Bytes()
	for _, scenario := range []string{"new-worker-context", "engine", "endpoint", "boot", "launch", "image", "pier", "native-status", "target", "operation", "revision", "previous-session"} {
		t.Run(scenario, func(t *testing.T) {
			actual, err := recording.ReadMetadata(raw)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "new-worker-context":
				actual.SessionID, actual.AttachedAt = strings.Repeat("2", 32), m.AttachedAt.Add(time.Second)
				actual.Identity.RecorderNamespaces.User++
				actual.Identity.RecorderNamespaces.Mount++
				actual.Identity.RecorderNamespaces.PID++
				actual.Identity.RecorderNamespaces.Network++
			case "engine":
				actual.Identity.EngineID = "replacement-daemon"
			case "endpoint":
				actual.Identity.EngineEndpointSHA256 = strings.Repeat("f", 64)
			case "boot":
				actual.Identity.HostBootID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			case "launch":
				actual.LaunchContractSHA256 = strings.Repeat("f", 64)
			case "image":
				actual.ImageManifestDigest = "sha256:" + strings.Repeat("f", 64)
			case "pier":
				actual.Identity.PierContinuitySHA256 = strings.Repeat("f", 64)
			case "native-status":
				actual.Identity.NativeShip, actual.Identity.NativeShipStatus = "~zod", "registered"
			case "target":
				actual.InstanceID, actual.Identity.Target.InstanceID = "other", "other"
			case "operation":
				actual.OperationID = "other-operation"
			case "revision":
				actual.ConfigurationRevision++
			case "previous-session":
				actual.PreviousSessionID = strings.Repeat("3", 32)
			}
			if actual.Validate() != nil {
				t.Fatal("test changed shape instead of a valid admission binding")
			}
			if j.MatchesAdmission(actual) != (scenario == "new-worker-context") {
				t.Fatal("worker substituted a binding or copied parent receive context")
			}
		})
	}
	after, _ := m.Bytes()
	if !bytes.Equal(after, raw) {
		t.Fatal("admission comparison changed original metadata")
	}
}

func TestRecordingJobRejectsUnsupportedNoncanonicalAndSensitiveControlPayloads(t *testing.T) {
	j, _, _, _ := recordingJobFixture(t)
	raw, _ := j.Bytes()
	for _, scenario := range []string{"version", "binary", "admission", "runtime", "runner", "mode", "target", "revision", "request", "container", "actor", "limits", "signing-key", "retained-log", "source-url", "relative-path", "unknown-field", "duplicate-field", "whitespace", "trailing", "oversize"} {
		t.Run(scenario, func(t *testing.T) {
			changed, err := ReadManagedRecordingJob(raw)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "version":
				changed.Version++
			case "binary":
				changed.ExecutableSHA256 = strings.Repeat("0", 64)
			case "admission":
				changed.AdmissionSHA256 = "invalid"
			case "runtime":
				changed.Operation.RuntimeID = strings.Repeat("d", 32)
			case "runner":
				changed.Operation.RunnerID = ""
			case "mode":
				changed.Operation.Mode = "unknown"
			case "target":
				changed.Operation.Target.ProjectID = "project"
			case "revision":
				changed.Operation.ConfigurationRevision = 0
			case "request":
				changed.Operation.RequestID = ""
			case "container":
				changed.Operation.ContainerID = "partial"
			case "actor":
				changed.Actor = "actor\nprivate"
			case "limits":
				changed.Limits.ReserveFreeBytes = 0
			case "signing-key":
				changed.State.HandoffSigningKey = []byte("private")
			case "retained-log":
				changed.State.Runtime.RetainedLogs = &protocol.RuntimeLogs{Lines: []string{"private"}}
			case "source-url":
				changed.State.Runtime.PillURL = "https://private.invalid/?credential=private"
			case "relative-path":
				changed.State.Runtime.Directory = "relative"
			}
			data, err := json.Marshal(changed) // deliberately bypass encoder validation
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "unknown-field":
				data = append([]byte(`{"unexpected":true,`), data[1:]...)
			case "duplicate-field":
				data = append([]byte(`{"version":1,`), data[1:]...)
			case "whitespace":
				data = append(data, '\n')
			case "trailing":
				data = append(data, []byte(`{}`)...)
			case "oversize":
				data = bytes.Repeat([]byte{'x'}, MaxRecordingJobBytes+1)
			}
			if _, err = ReadManagedRecordingJob(data); !errors.Is(err, ErrRecordingJob) || strings.Contains(err.Error(), "credential=") {
				t.Fatal("unsupported or sensitive control payload was accepted/reflected", err)
			}
		})
	}
}

func TestRecordingJobCannotStripSourceQuarantineToCreateLaunchAuthority(t *testing.T) {
	j, state, op, m := recordingJobFixture(t)
	state.Storage.Uncertain = true
	if _, err := NewManagedRecordingJob(state, op, m, j.ExecutableSHA256, j.Actor, j.CreatedAt, j.Limits); !errors.Is(err, ErrRecordingJob) {
		t.Fatal("source quarantine was sanitized into a clean job", err)
	}
}
