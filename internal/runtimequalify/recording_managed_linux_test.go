//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/rootless"
)

func preparedRecordingFixture(t *testing.T, comet, initialized bool) (*fakeEngine, ManagedState, RecordingOperation, *fakeContainer) {
	t.Helper()
	f, state, persist := managedFixture(t)
	if comet {
		f, state, persist = managedCometFixture(t)
	}
	f.engine.ID = "recording-fixture-daemon"
	f.d.readBootID = func() (string, error) { return "11111111-2222-3333-4444-555555555555", nil }
	var err error
	if initialized {
		state, _, err = applyManaged(t.Context(), "start", state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
		state, _, err = applyManaged(t.Context(), "stop", state, persist, f.d)
		if err != nil {
			t.Fatal(err)
		}
	}
	execute := f.d.execute
	f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
		if len(args) == 7 && args[4] == "container" && args[5] == "start" {
			for _, c := range f.containers {
				if c.id == args[6] && c.kind != "probe" {
					return "", errors.New("fixture stopped before native Start dispatch")
				}
			}
		}
		return execute(ctx, args, env)
	}
	state, _, err = applyManaged(t.Context(), "start", state, persist, f.d)
	if err == nil || len(f.containers) != 1 {
		t.Fatal("prepared-container boundary was not retained", err)
	}
	f.d.execute = execute
	var selected *fakeContainer
	for _, c := range f.containers {
		selected = c
		s := mapAt(c.doc, "State")
		s["Status"], s["Pid"] = "created", 0
		s["StartedAt"], s["FinishedAt"] = "0001-01-01T00:00:00Z", "0001-01-01T00:00:00Z"
	}
	op := RecordingOperation{RunnerID: "runner", Mode: "development-managed", RuntimeID: state.ID, Target: apppolicy.Target{InstanceID: "selected-member"}, ConfigurationRevision: 9007199254740999, OperationID: "original-start", RequestID: "original-request", ContainerID: selected.id}
	return f, state, op, selected
}

type recordingTestStream struct {
	io.Reader
	closes int
}

func (s *recordingTestStream) Close() error { s.closes++; return nil }

func TestManagedRecordingCapturesBeforeBootstrapIdentityAndIndependentExit(t *testing.T) {
	for _, scenario := range []struct {
		name               string
		comet, initialized bool
	}{{"new-fake", false, false}, {"pending-comet", true, false}, {"existing-fake", false, true}} {
		t.Run(scenario.name, func(t *testing.T) {
			f, state, op, selected := preparedRecordingFixture(t, scenario.comet, scenario.initialized)
			before, _ := json.Marshal(state)
			payload := append(bytes.Repeat([]byte("bootstrap\n0wdiagnostic.atom\r\x1b[2K"), 350), 0, 255)
			packet := make([]byte, 8)
			packet[0] = 2
			binary.BigEndian.PutUint32(packet[4:], uint32(len(payload)))
			stream := &recordingTestStream{Reader: bytes.NewReader(append(packet, payload...))}
			attached, ready := 0, 0
			m, source, err := prepareManagedRecording(t.Context(), state, op, func(got recording.Metadata) error {
				ready++
				// Inert fixture evidence only. The factory itself dispatches no
				// Start, Stop, exec, signal, wait or filesystem modification.
				s := mapAt(selected.doc, "State")
				s["Status"], s["StartedAt"], s["FinishedAt"] = "exited", "2026-11-18T00:00:00Z", "2026-11-18T00:00:01Z"
				s["ExitCode"], s["OOMKilled"] = 137, true
				return nil
			}, f.d, func(_ context.Context, socket, id string) (io.ReadCloser, error) {
				attached++
				if socket != filepath.Join(f.d.runtimeDir, rootless.SocketName) || id != op.ContainerID {
					t.Fatal("attachment changed the selected engine/container", socket, id)
				}
				return stream, nil
			})
			if err != nil || attached != 0 || ready != 0 || m.ConfigurationRevision != op.ConfigurationRevision || m.Identity.PierBinding != "prepared-storage" && !scenario.initialized {
				t.Fatal("preparation created an attachment, lost binding or invented storage", m, err)
			}
			encoded, err := m.Bytes()
			if err != nil || bytes.Contains(encoded, []byte(f.d.home)) || bytes.Contains(encoded, []byte("Entrypoint")) || bytes.Contains(encoded, []byte("Env")) {
				t.Fatal("private launch paths/configuration leaked into attachment metadata", err)
			}
			if scenario.comet && (m.Identity.NativeShip != "" || m.Identity.NativeShipStatus != "pending-comet-mining") {
				t.Fatal("mining identity was invented", m.Identity)
			}
			if scenario.initialized && m.Identity.PierBinding != "existing-pier" {
				t.Fatal("existing pier was relabeled as future storage")
			}
			if !scenario.initialized {
				if _, err = os.Lstat(managedPier(f, state)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("recording preparation created a pier", err)
				}
			}
			dir := t.TempDir()
			if err = os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			store, err := recording.OpenStore(dir, recording.Limits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 64 << 10, ReserveFreeBytes: 1 << 20})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			calls := len(f.calls)
			out, err := store.CaptureDocker(t.Context(), m, source)
			if err != nil || attached != 1 || ready != 1 || stream.closes != 1 || !out.ExitSaved || out.GapSaved || !out.ClosedDurably {
				t.Fatal("durable capture/independent terminal evidence failed", out, err)
			}
			page, err := store.ReadPage(t.Context(), m.SessionID, nil, 256, 1<<20)
			if err != nil || len(page.Frames) != 2 || !bytes.Equal(page.Frames[0].Data, payload) {
				t.Fatal("bootstrap originals were transformed", err)
			}
			exit, err := recording.ReadExit(page.Frames[1].Data)
			if err != nil || exit.Code != 137 || !exit.OOMKilled || page.Metadata.Identity.NativeShip != m.Identity.NativeShip {
				t.Fatal("exit or unknown identity changed", exit, err)
			}
			after, _ := json.Marshal(state)
			if !bytes.Equal(before, after) {
				t.Fatal("recording admission changed retained runtime intent")
			}
			for _, call := range f.calls[calls:] {
				if len(call) < 2 || call[1] != "inspect" {
					t.Fatal("capture admission dispatched a runtime operation", call)
				}
			}
		})
	}
}

func TestManagedRecordingCannotTurnUnverifiedExitIntoTerminalEvidence(t *testing.T) {
	for _, scenario := range []string{"valid-nonzero-oom", "reverse-wallclock", "created", "running", "no-process", "out-of-range-time", "changed-environment", "exit-during-read", "engine-during-read", "boot-during-read"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, op, selected := preparedRecordingFixture(t, false, false)
			boot := "11111111-2222-3333-4444-555555555555"
			f.d.readBootID = func() (string, error) { return boot, nil }
			_, source, err := prepareManagedRecording(t.Context(), state, op, func(recording.Metadata) error { return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
				t.Fatal("an exit read attached to the output stream")
				return nil, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			s := mapAt(selected.doc, "State")
			s["Status"], s["StartedAt"], s["FinishedAt"] = "exited", "2026-11-18T00:00:01Z", "2026-11-18T00:00:02Z"
			s["ExitCode"], s["OOMKilled"], s["Error"] = 137, true, "private daemon diagnostic"
			switch scenario {
			case "reverse-wallclock":
				s["FinishedAt"] = "2026-11-18T00:00:00Z"
			case "created":
				s["Status"] = "created"
			case "running":
				s["Status"], s["Running"], s["Pid"] = "running", true, 4242
			case "no-process":
				s["StartedAt"] = "0001-01-01T00:00:00Z"
			case "out-of-range-time":
				s["StartedAt"] = "1000-01-01T00:00:00Z"
			case "changed-environment":
				mapAt(selected.doc, "Config")["Env"] = []string{"UNEXPECTED=private"}
			}
			reads := 0
			f.editInspect = func(_ string, doc map[string]any) {
				reads++
				if scenario == "exit-during-read" && reads == 2 {
					mapAt(doc, "State")["ExitCode"] = 0
				}
				if scenario == "engine-during-read" && reads == 1 {
					f.engine.ID = "replacement-daemon"
				}
				if scenario == "boot-during-read" && reads == 1 {
					boot = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
				}
			}
			report := filepath.Join(state.Runtime.Directory, "report.json")
			before, err := os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			event, err := source.ObserveExit(t.Context())
			if scenario == "valid-nonzero-oom" || scenario == "reverse-wallclock" {
				if err != nil || event == nil || event.Code != 137 || !event.OOMKilled || !event.DockerErrorPresent || event.ContainerID != op.ContainerID {
					t.Fatal("independent abnormal exit evidence was erased", event, err)
				}
				if scenario == "reverse-wallclock" && event.FinishedUnixNano >= event.StartedUnixNano {
					t.Fatal("wall-clock reversal was silently normalized")
				}
				encoded, _ := event.Bytes()
				if bytes.Contains(encoded, []byte("private daemon diagnostic")) {
					t.Fatal("private daemon error leaked from typed evidence")
				}
			} else if !errors.Is(err, errRecordingSelection) || event != nil {
				t.Fatal("missing or changed terminal proof was accepted", event, err)
			}
			after, err := os.ReadFile(report)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("read-only terminal observation changed retained intent", err)
			}
		})
	}
}

func TestManagedRecordingRefusesUnboundOrAmbiguousInitialSelection(t *testing.T) {
	for _, scenario := range []string{"unprepared", "runtime", "project", "desk", "container", "uncertain-create", "undispatched-create", "second-container", "uncertain-removed", "storage-uncertain", "unregistered-storage", "missing-runner", "missing-revision"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, op, _ := preparedRecordingFixture(t, false, false)
			index := len(state.Runtime.Containers) - 1
			switch scenario {
			case "unprepared":
				state.Prepared = false
			case "runtime":
				op.RuntimeID = "different"
			case "project":
				op.Target.ProjectID, op.Target.Generation, op.Target.Creator = "project", 1, "creator"
			case "desk":
				op.Target.Desk = "sample"
			case "container":
				op.ContainerID = strings.Repeat("e", 64)
			case "uncertain-create":
				state.Runtime.Containers[index].CreateUncertain = true
			case "undispatched-create":
				state.Runtime.Containers[index].CreateDispatched = false
			case "second-container":
				state.Runtime.Containers = append(state.Runtime.Containers, state.Runtime.Containers[index])
			case "uncertain-removed":
				state.Runtime.Containers[0].CreateUncertain = true
			case "storage-uncertain":
				state.Storage.Uncertain = true
			case "unregistered-storage":
				state.Storage.MountIdentity.Inode = 0
			case "missing-runner":
				op.RunnerID = ""
			case "missing-revision":
				op.ConfigurationRevision = 0
			}
			_, _, err := prepareManagedRecording(t.Context(), state, op, func(recording.Metadata) error { t.Fatal("refused selection reached readiness"); return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
				t.Fatal("refused selection attached")
				return nil, nil
			})
			if !errors.Is(err, errRecordingSelection) {
				t.Fatal("unbound or ambiguous prepared selection was accepted", err)
			}
		})
	}
}

func TestManagedRecordingRefusesDriftBeforeOrDuringAttachment(t *testing.T) {
	for _, scenario := range []string{"engine", "boot", "tty", "launch", "label", "container", "network", "pill", "storage", "unexpected-pier", "started", "missing-start", "during-attach", "metadata", "state-pointer"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, op, selected := preparedRecordingFixture(t, false, false)
			boot := "11111111-2222-3333-4444-555555555555"
			f.d.readBootID = func() (string, error) { return boot, nil }
			attached, ready := 0, 0
			stream := &recordingTestStream{Reader: bytes.NewReader(nil)}
			m, source, err := prepareManagedRecording(t.Context(), state, op, func(recording.Metadata) error { ready++; return nil }, f.d, func(context.Context, string, string) (io.ReadCloser, error) {
				attached++
				if scenario == "during-attach" {
					mapAt(selected.doc, "Config")["Tty"] = true
				}
				return stream, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "engine":
				f.engine.ID = "replacement-daemon"
			case "boot":
				boot = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			case "tty":
				mapAt(selected.doc, "Config")["Tty"] = true
			case "launch":
				mapAt(selected.doc, "HostConfig")["Memory"] = 1
			case "label":
				mapAt(mapAt(selected.doc, "Config"), "Labels")[f.runLabelName()] = "unrelated"
			case "container":
				selected.doc["Id"] = strings.Repeat("f", 64)
			case "network":
				mapAt(selected.doc, "HostConfig")["NetworkMode"] = "none"
			case "pill":
				if err = os.WriteFile(state.Runtime.PillPath, []byte("changed fixture bootstrap bytes"), 0600); err != nil {
					t.Fatal(err)
				}
			case "storage":
				mount := filepath.Dir(managedPier(f, state))
				if err = os.Rename(mount, mount+"-retained"); err != nil {
					t.Fatal(err)
				}
				if err = os.Mkdir(mount, 0700); err != nil {
					t.Fatal(err)
				}
			case "unexpected-pier":
				if err = os.Mkdir(managedPier(f, state), 0700); err != nil {
					t.Fatal(err)
				}
			case "started":
				mapAt(selected.doc, "State")["StartedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
			case "missing-start":
				delete(mapAt(selected.doc, "State"), "StartedAt")
			case "metadata":
				m.Identity.NativeShip = "~nec"
				if source.Ready(m) == nil || ready != 0 {
					t.Fatal("changed caller metadata reached durable readiness")
				}
				return
			case "state-pointer":
				state.Storage.Path = "fakeships/unrelated/pier"
			}
			connection, err := source.Attach(t.Context())
			if scenario == "state-pointer" {
				if err != nil || connection == nil || attached != 1 {
					t.Fatal("caller pointer changed the pinned selection", err)
				}
				connection.Close()
				return
			}
			if !errors.Is(err, errRecordingSelection) || connection != nil || ready != 0 {
				t.Fatal("changed exact launch admitted capture", err)
			}
			if scenario == "during-attach" {
				if attached != 1 || stream.closes != 1 {
					t.Fatal("changed attachment was not detached", attached, stream.closes)
				}
			} else if attached != 0 {
				t.Fatal("refused selection contacted the output transport")
			}
		})
	}
}
