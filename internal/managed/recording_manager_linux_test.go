//go:build linux && amd64

package managed

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Shape-valid launch inputs from the pinned actual codec fixture, rebound to a
// new synthetic journal operation. Native admission/submission are explicit inert
// seams here; this is not a real engine, executable, service or native gate.
func recordingManagerFixture(t *testing.T) (*store, protocol.Operation, runtimequalify.ManagedState, *recordingManager) {
	t.Helper()
	s := testStore(t)
	s.mode = DevelopmentMode
	_, _, err := s.registerKind("recording-actor", registration("one", "register", 18087), nil, true)
	g02Check(t, err)
	op, _, err := s.accept("recording-actor", "one", protocol.OperationRequest{Kind: "start", RequestID: "start", ExpectedRevision: 1})
	g02Check(t, err)
	op, v, err := s.beginOperation(op.ID)
	g02Check(t, err)
	raw, err := os.ReadFile("../runtimequalify/testdata/original-recording-claim1/files/11111111111111111111111111111111/job.json")
	g02Check(t, err)
	template, err := runtimequalify.ReadManagedRecordingJob(raw)
	g02Check(t, err)
	state := template.State
	oldID := state.ID
	state.ID, state.Runtime.RunID = v.Runtime.ID, v.Runtime.ID
	state.Storage.Path = v.Runtime.Storage.Path
	base := t.TempDir()
	g02Check(t, os.Chmod(base, 0700))
	state.Runtime.Directory = filepath.Join(base, "managed", state.ID)
	state.Runtime.PillPath = filepath.Join(state.Runtime.Directory, filepath.Base(state.Runtime.PillPath))
	for n := range state.Runtime.Containers {
		state.Runtime.Containers[n].Name = strings.ReplaceAll(state.Runtime.Containers[n].Name, oldID, state.ID)
	}
	g02Check(t, s.runtimeUpdate(op.InstanceID, op.ID, state))
	opts := runtimequalify.RecordingWorkerOptions{JobsRoot: filepath.Join(base, "recording-jobs"), RecordingsRoot: filepath.Join(base, "recordings"), MaxJobs: 8, JobReserveFreeBytes: 1 << 20}
	for _, path := range []string{opts.JobsRoot, opts.RecordingsRoot} {
		g02Check(t, os.Mkdir(path, 0700))
	}
	m := &recordingManager{opts: opts, limits: runtimequalify.RecordingJobLimits{MaxSessionBytes: 1 << 20, MaxSegmentBytes: 4096, ReserveFreeBytes: 1 << 20}}
	return s, op, state, m
}

func managerRecordingMetadata(op runtimequalify.RecordingOperation, state runtimequalify.ManagedState) recording.Metadata {
	return recording.Metadata{Version: 4, SessionID: strings.Repeat("7", 32), RunnerID: op.RunnerID, Mode: op.Mode, InstanceID: op.Target.InstanceID, ConfigurationRevision: op.ConfigurationRevision, OperationID: op.OperationID, RequestID: op.RequestID, ContainerID: op.ContainerID, LaunchContractSHA256: strings.Repeat("b", 64), ImageManifestDigest: state.Runtime.ImageID, RuntimeVersion: state.Runtime.VereVersion, Transport: "docker-multiplex", AttachedAt: time.Now().UTC(), Identity: &recording.AttachmentIdentity{Target: op.Target, NativeShipStatus: "pending-comet-mining", PierBinding: "prepared-storage", PierContinuitySHA256: strings.Repeat("c", 64), EngineID: "fixture-daemon", EngineEndpointSHA256: strings.Repeat("d", 64), HostBootID: "11111111-2222-3333-4444-555555555555", RecorderNamespaces: recording.HostNamespaces{User: 1, Mount: 2, PID: 3, Network: 4}}}
}

func TestRecordingManagerBindsActualOperationBeforeOneSubmissionAndFreshGuard(t *testing.T) {
	for _, scenario := range []string{"pass", "wrong-actor", "wrong-snapshot", "changed-before-admit", "lost-ack", "stale-final-guard"} {
		t.Run(scenario, func(t *testing.T) {
			s, op, state, m := recordingManagerFixture(t)
			prepares, submissions, observations := 0, 0, 0
			var metadata recording.Metadata
			var status runtimequalify.RecordingWorkerStatus
			m.prepare = func(_ context.Context, snapshot runtimequalify.ManagedState, selected runtimequalify.RecordingOperation) (recording.Metadata, error) {
				prepares++
				if selected.OperationID != op.ID || selected.RequestID != op.RequestID || selected.Target.InstanceID != op.InstanceID || selected.RuntimeID != state.ID || selected.RunnerID != s.runnerID || selected.Mode != s.mode {
					t.Fatal("manager invented operation/target authority")
				}
				if scenario == "changed-before-admit" {
					g02Check(t, s.runtimeUpdate(op.InstanceID, op.ID, state))
				}
				metadata = managerRecordingMetadata(selected, snapshot)
				return metadata, nil
			}
			m.freeze = func(context.Context) (runtimequalify.RecordingExecutable, error) {
				return runtimequalify.RecordingExecutable{Path: "/inert/not-executed", SHA256: strings.Repeat("e", 64)}, nil
			}
			m.submit = func(ctx context.Context, job runtimequalify.ManagedRecordingJob, fence func(context.Context) error) (runtimequalify.RecordingJobReady, error) {
				submissions++
				g02Check(t, fence(ctx))
				i, err := loadRecordingIntent(s.db, job.ID)
				g02Check(t, err)
				if i.Actor != op.Actor || i.OperationID != op.ID || i.RequestID != op.RequestID || job.Actor != op.Actor {
					t.Fatal("submission preceded original actor/operation journal")
				}
				jobs, err := runtimequalify.OpenRecordingJobs(m.opts.JobsRoot, 8, 1<<20)
				g02Check(t, err)
				_, err = jobs.Read(job.ID, i.JobSHA256)
				g02Check(t, err)
				g02Check(t, jobs.Close())
				if scenario == "lost-ack" {
					return runtimequalify.RecordingJobReady{}, runtimequalify.ErrRecordingSubmissionUncertain
				}
				worker := runtimequalify.RecordingWorkerIdentity{ExecutableSHA256: job.ExecutableSHA256, PID: 42, StartTicks: 123, InvocationID: strings.Repeat("a", 32), HostBootID: metadata.Identity.HostBootID, Namespaces: metadata.Identity.RecorderNamespaces}
				raw, _ := metadata.Bytes()
				ready := runtimequalify.RecordingJobReady{Version: 1, JobID: job.ID, JobSHA256: i.JobSHA256, SessionID: metadata.SessionID, MetadataSHA256: stopEvidenceDigest(raw), AttachedAt: metadata.AttachedAt}
				status = runtimequalify.RecordingWorkerStatus{JobID: job.ID, JobSHA256: i.JobSHA256, Worker: &worker, Ready: &ready, Allocation: &runtimequalify.RecordingJobSession{JobID: job.ID, JobSHA256: i.JobSHA256, SessionID: metadata.SessionID}, SessionID: metadata.SessionID, SessionStorage: "header-available", Metadata: &metadata, WriterPresent: true, PrefixStatus: "snapshot-eof"}
				return ready, nil
			}
			m.status = func(context.Context, string, string) (runtimequalify.RecordingWorkerStatus, error) {
				return status, nil
			}
			m.observe = func(_ context.Context, job runtimequalify.ManagedRecordingJob, worker runtimequalify.RecordingWorkerIdentity) (runtimequalify.RecordingSupervisorState, error) {
				observations++
				unit, err := runtimequalify.RecordingWorkerUnitName(job)
				g02Check(t, err)
				return runtimequalify.RecordingSupervisorState{Unit: unit, MainPID: worker.PID, InvocationID: worker.InvocationID, ControlGroup: "/user.slice/" + unit}, nil
			}
			selectedOp, selectedState := op, state
			if scenario == "wrong-actor" {
				selectedOp.Actor = "other"
			}
			if scenario == "wrong-snapshot" {
				selectedState.Port++
			}
			container := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			guard, err := m.lifecycle(s, selectedOp).BeforeStart(t.Context(), selectedState, container)
			if scenario == "pass" || scenario == "stale-final-guard" {
				g02Check(t, err)
				if scenario == "stale-final-guard" {
					g02Check(t, s.runtimeUpdate(op.InstanceID, op.ID, state))
					if guard(t.Context()) == nil || observations != 0 {
						t.Fatal("stale final journal state reached live probe/Start")
					}
				} else {
					g02Check(t, guard(t.Context()))
					if observations != 1 {
						t.Fatal("final live observation missing")
					}
				}
				if submissions != 1 {
					t.Fatal("submission count changed", submissions)
				}
			} else {
				if err == nil || guard != nil {
					t.Fatal("invalid handoff granted guard", scenario)
				}
				if scenario == "lost-ack" && err != runtimequalify.ErrRecordingSubmissionUncertain {
					t.Fatal("lost reply was relabeled", err)
				}
				if scenario != "lost-ack" && submissions != 0 {
					t.Fatal("refused state reached submission")
				}
				if (scenario == "wrong-actor" || scenario == "wrong-snapshot") && prepares != 0 {
					t.Fatal("unowned snapshot reached native preparation")
				}
			}
		})
	}
}

func TestRecordingContainerLookupKeepsHistoricalModeAndRejectsChangedTarget(t *testing.T) {
	s, _, i := recordingIntentFixture(t)
	_, _, err := s.admitRecordingIntent(i)
	g02Check(t, err)
	s.mode = DevelopmentMode // same runner/home promoted from fake-only management
	before := appAllRows(t, s.db)
	got, err := s.recordingIntentForContainer(t.Context(), i.InstanceID, i.RuntimeID, i.ContainerID)
	g02Check(t, err)
	if got == nil || !reflect.DeepEqual(*got, i) || !reflect.DeepEqual(before, appAllRows(t, s.db)) {
		t.Fatal("mode transition lost original capture or rewrote it")
	}
	got, err = s.recordingIntentForContainer(t.Context(), "other", i.RuntimeID, i.ContainerID)
	if err != nil || got != nil {
		t.Fatal("different target adopted old capture", err)
	}
}

func TestRecordingManagerRemovalKeepsIncompleteEvidenceWithoutDeadlockingClosedCapture(t *testing.T) {
	for _, scenario := range []string{"finished", "incomplete-closed", "incomplete-open", "pending", "wrong-result", "wrong-header", "changed-during-drain", "unrecorded"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, i := recordingIntentFixture(t)
			if scenario != "unrecorded" {
				_, _, err := s.admitRecordingIntent(i)
				g02Check(t, err)
			}
			op, err := loadOperation(s.db, i.OperationID)
			g02Check(t, err)
			v, err := loadInstance(s.db, i.InstanceID)
			g02Check(t, err)
			result := runtimequalify.RecordingJobResult{Version: 1, JobID: i.JobID, JobSHA256: i.JobSHA256, Code: "capture-finished", FinishedAt: time.Now().UTC(), Capture: recording.CaptureResult{SessionID: strings.Repeat("7", 32), AttachmentAcknowledged: true, ReadyAcknowledged: true, ClosedDurably: true, ExitSaved: true}}
			result.Capture.Stats.AtFrameBoundary = true
			if strings.HasPrefix(scenario, "incomplete") {
				result.Code, result.Capture.ExitSaved, result.Capture.GapReason = "capture-incomplete", false, "transport-interrupted"
				if scenario == "incomplete-open" {
					result.Capture.ClosedDurably = false
				}
			}
			if scenario == "wrong-result" {
				result.JobID = strings.Repeat("f", 32)
			}
			metaState := v.Runtime
			metaState.Runtime.ImageID, metaState.Runtime.VereVersion = "sha256:"+strings.Repeat("a", 64), "fixture-native"
			metadata := managerRecordingMetadata(runtimequalify.RecordingOperation{RunnerID: i.RunnerID, Mode: i.Mode, RuntimeID: i.RuntimeID, Target: apppolicy.Target{InstanceID: i.InstanceID}, ConfigurationRevision: i.ConfigurationRevision, OperationID: i.OperationID, RequestID: i.RequestID, ContainerID: i.ContainerID}, metaState)
			if scenario == "wrong-header" {
				metadata.InstanceID, metadata.Identity.Target.InstanceID = "other", "other"
			}
			drains := 0
			m := &recordingManager{
				drain: func(context.Context, string, string) (runtimequalify.RecordingJobResult, error) {
					drains++
					if scenario == "pending" {
						return runtimequalify.RecordingJobResult{}, runtimequalify.ErrRecordingDrainPending
					}
					if scenario == "changed-during-drain" {
						changed := v.Runtime
						changed.Port++
						g02Check(t, s.runtimeUpdate(i.InstanceID, i.OperationID, changed))
					}
					if result.Code == "capture-incomplete" {
						return result, recording.ErrCaptureIncomplete
					}
					return result, nil
				},
				status: func(context.Context, string, string) (runtimequalify.RecordingWorkerStatus, error) {
					return runtimequalify.RecordingWorkerStatus{JobID: i.JobID, JobSHA256: i.JobSHA256, Result: &result, SessionID: result.Capture.SessionID, SessionStorage: "header-available", Metadata: &metadata}, nil
				},
			}
			before := appAllRows(t, s.db)
			err = m.lifecycle(s, op).BeforeRemove(t.Context(), v.Runtime, i.ContainerID)
			allowed := scenario == "finished" || scenario == "incomplete-closed" || scenario == "unrecorded"
			if allowed && err != nil || !allowed && err == nil {
				t.Fatal("incorrect closed/pending/remapped removal boundary", scenario, err)
			}
			if scenario == "unrecorded" && drains != 0 {
				t.Fatal("old unrecorded runtime was adopted into capture")
			}
			if scenario != "changed-during-drain" && !reflect.DeepEqual(before, appAllRows(t, s.db)) {
				t.Fatal("removal callback changed original operation/capture history")
			}
		})
	}
}
