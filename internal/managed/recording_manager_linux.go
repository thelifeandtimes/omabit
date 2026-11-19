//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// This bounded pilot is selected only by explicit development service opt-in.
// Configurable retention, running/project/utility factories and native/installed
// qualification must precede advertising comprehensive recording support.
type recordingManager struct {
	opts    runtimequalify.RecordingWorkerOptions
	limits  runtimequalify.RecordingJobLimits
	prepare func(context.Context, runtimequalify.ManagedState, runtimequalify.RecordingOperation) (recording.Metadata, error)
	freeze  func(context.Context) (runtimequalify.RecordingExecutable, error)
	submit  func(context.Context, runtimequalify.ManagedRecordingJob, func(context.Context) error) (runtimequalify.RecordingJobReady, error)
	status  func(context.Context, string, string) (runtimequalify.RecordingWorkerStatus, error)
	observe func(context.Context, runtimequalify.ManagedRecordingJob, runtimequalify.RecordingWorkerIdentity) (runtimequalify.RecordingSupervisorState, error)
	drain   func(context.Context, string, string) (runtimequalify.RecordingJobResult, error)
}

func configureRecordingServer(r *Server) error {
	if _, ok := r.b.(recordingBackend); !ok || r.s.mode != DevelopmentMode {
		return errors.New("recording pilot requires development management with a recording backend")
	}
	m, err := newRecordingManager(r.s.directory)
	if err != nil {
		return err
	}
	r.recording = func(op protocol.Operation) runtimequalify.ManagedRecordingLifecycle { return m.lifecycle(r.s, op) }
	return nil
}

func newRecordingManager(dir string) (*recordingManager, error) {
	opts, err := runtimequalify.PrepareManagedRecordingHome(dir)
	if err != nil {
		return nil, err
	}
	// Sixteen64MiB sessions bound pilot raw storage to1GiB plus metadata;
	// there is no automatic pruning or silent eviction. Leave2GiB free.
	opts.MaxJobs, opts.JobReserveFreeBytes = 16, 2<<30
	return &recordingManager{opts: opts, limits: runtimequalify.RecordingJobLimits{MaxSessionBytes: 64 << 20, MaxSegmentBytes: 8 << 20, ReserveFreeBytes: 2 << 30},
		prepare: func(ctx context.Context, state runtimequalify.ManagedState, op runtimequalify.RecordingOperation) (recording.Metadata, error) {
			m, _, err := runtimequalify.PrepareManagedRecording(ctx, state, op, func(recording.Metadata) error { return nil })
			return m, err
		}, freeze: runtimequalify.FreezeRecordingExecutable, submit: runtimequalify.SubmitRecordingWorker,
		status: runtimequalify.ReadManagedRecordingStatus, observe: runtimequalify.ObserveRecordingSupervisor, drain: runtimequalify.AwaitManagedRecordingDrain}, nil
}

func (m *recordingManager) lifecycle(s *store, op protocol.Operation) runtimequalify.ManagedRecordingLifecycle {
	return runtimequalify.ManagedRecordingLifecycle{
		BeforeStart: func(ctx context.Context, snapshot runtimequalify.ManagedState, container string) (func(context.Context) error, error) {
			v, err := s.recordingSnapshotAuthority(ctx, op, snapshot)
			if err != nil || op.Kind != "start" && op.Kind != "startup-resume" {
				return nil, errRecordingHistory
			}
			selected := runtimequalify.RecordingOperation{RunnerID: s.runnerID, Mode: s.mode, RuntimeID: snapshot.ID, Target: apppolicy.Target{InstanceID: op.InstanceID}, ConfigurationRevision: op.ConfigRevision, OperationID: op.ID, RequestID: op.RequestID, ContainerID: container}
			metadata, err := m.prepare(ctx, snapshot, selected)
			if err != nil {
				return nil, err
			}
			image, err := m.freeze(ctx)
			if err != nil {
				return nil, err
			}
			job, err := runtimequalify.NewManagedRecordingJob(snapshot, selected, metadata, image.SHA256, op.Actor, time.Now(), m.limits)
			if err != nil {
				return nil, err
			}
			stateDigest, err := recordingStateDigest(snapshot)
			if err != nil {
				return nil, err
			}
			jobs, err := runtimequalify.OpenRecordingJobs(m.opts.JobsRoot, m.opts.MaxJobs, m.opts.JobReserveFreeBytes)
			if err != nil {
				return nil, err
			}
			defer jobs.Close()
			digest, err := jobs.Create(job)
			if err != nil {
				return nil, err
			}
			i := recordingIntent{Version: 1, JobID: job.ID, JobSHA256: digest, RunnerID: s.runnerID, Mode: s.mode, Actor: op.Actor, OperationID: op.ID, RequestID: op.RequestID, InstanceID: op.InstanceID, RuntimeID: snapshot.ID, ConfigurationRevision: op.ConfigRevision, RuntimeRevision: v.RuntimeRevision, StateSHA256: stateDigest, ContainerID: container, ExecutableSHA256: image.SHA256, AdmissionSHA256: job.AdmissionSHA256, CreatedAt: job.CreatedAt}
			_, replay, err := s.admitRecordingIntent(i)
			if err != nil || replay {
				return nil, errRecordingHistory // orphan files stay retained, never submitted/replayed
			}
			ready, err := m.submit(ctx, job, func(ctx context.Context) error { return s.checkRecordingIntent(ctx, i) })
			if err != nil {
				return nil, err
			}
			status, err := m.status(ctx, job.ID, digest)
			if err != nil || !recordingStatusPrepared(status, ready) {
				return nil, runtimequalify.ErrRecordingWorker
			}
			worker := *status.Worker
			return func(ctx context.Context) error {
				if s.checkRecordingIntent(ctx, i) != nil {
					return errRecordingHistory
				}
				status, err := m.status(ctx, job.ID, digest)
				if err != nil || !recordingStatusPrepared(status, ready) || *status.Worker != worker {
					return runtimequalify.ErrRecordingWorker
				}
				unit, err := m.observe(ctx, job, worker)
				if err != nil || runtimequalify.CheckRecordingSupervisor(job, unit, worker, worker) != nil {
					return runtimequalify.ErrRecordingWorker
				}
				return s.checkRecordingIntent(ctx, i)
			}, nil
		},
		BeforeRemove: func(ctx context.Context, snapshot runtimequalify.ManagedState, container string) error {
			if _, err := s.recordingSnapshotAuthority(ctx, op, snapshot); err != nil {
				return err
			}
			i, err := s.recordingIntentForContainer(ctx, op.InstanceID, snapshot.ID, container)
			if err != nil || i == nil {
				return err // an older unrecorded container remains ordinary native cleanup
			}
			if m.prepare != nil {
				neverStarted, err := m.verifiedNeverStarted(ctx, snapshot, *i)
				if err != nil {
					return err
				}
				if neverStarted {
					// Strict native preparation proves this exact original launch
					// is still created with zero process/times. There is no process
					// exit to drain. Retain its job/partial session without inventing
					// a successful capture or holding Stop cleanup indefinitely.
					_, err := s.recordingSnapshotAuthority(ctx, op, snapshot)
					return err
				}
			}
			result, err := m.drain(ctx, i.JobID, i.JobSHA256)
			finished := err == nil && result.Code == "capture-finished" && result.Capture.ExitSaved
			incomplete := errors.Is(err, recording.ErrCaptureIncomplete) && result.Code == "capture-incomplete"
			if (finished || incomplete) && result.JobID == i.JobID && result.JobSHA256 == i.JobSHA256 && result.Capture.ClosedDurably {
				status, readErr := m.status(ctx, i.JobID, i.JobSHA256)
				if readErr == nil && status.JobID == i.JobID && status.JobSHA256 == i.JobSHA256 && status.Result != nil && reflect.DeepEqual(*status.Result, result) && status.SessionID == result.Capture.SessionID && !status.WriterPresent && status.SessionStorage == "header-available" && recordingStatusMatchesIntent(status, *i) {
					// The saved gap/failure remains visible. Native Stop cleanup is
					// not blocked by a durably closed, explicitly incomplete log.
					_, err := s.recordingSnapshotAuthority(ctx, op, snapshot)
					return err
				}
			}
			if err != nil {
				return err
			}
			return runtimequalify.ErrRecordingDrainPending
		},
	}
}

func (m *recordingManager) verifiedNeverStarted(ctx context.Context, snapshot runtimequalify.ManagedState, i recordingIntent) (bool, error) {
	jobs, err := runtimequalify.OpenRecordingJobs(m.opts.JobsRoot, m.opts.MaxJobs, m.opts.JobReserveFreeBytes)
	if err != nil {
		return false, err
	}
	defer jobs.Close()
	job, err := jobs.Read(i.JobID, i.JobSHA256)
	if err != nil || job.Actor != i.Actor || job.Operation.RunnerID != i.RunnerID || job.Operation.Mode != i.Mode || job.Operation.RuntimeID != i.RuntimeID || job.Operation.Target.InstanceID != i.InstanceID || job.Operation.OperationID != i.OperationID || job.Operation.RequestID != i.RequestID || job.Operation.ConfigurationRevision != i.ConfigurationRevision || job.Operation.ContainerID != i.ContainerID {
		return false, errRecordingHistory
	}
	actual, err := m.prepare(ctx, snapshot, job.Operation)
	if err != nil {
		return false, nil // no never-started proof; normal terminal drain is required
	}
	return job.MatchesAdmission(actual), nil
}

func recordingStatusMatchesIntent(status runtimequalify.RecordingWorkerStatus, i recordingIntent) bool {
	m := status.Metadata
	return m != nil && m.Validate() == nil && m.RunnerID == i.RunnerID && m.Mode == i.Mode && m.InstanceID == i.InstanceID && m.ProjectID == "" && m.ContainerID == i.ContainerID && m.OperationID == i.OperationID && m.RequestID == i.RequestID && m.ConfigurationRevision == i.ConfigurationRevision && m.SessionID == status.SessionID
}

func recordingStatusPrepared(status runtimequalify.RecordingWorkerStatus, ready runtimequalify.RecordingJobReady) bool {
	return status.JobID == ready.JobID && status.JobSHA256 == ready.JobSHA256 && status.Ready != nil && *status.Ready == ready && status.Worker != nil && status.Allocation != nil && status.Allocation.JobID == ready.JobID && status.Allocation.JobSHA256 == ready.JobSHA256 && status.Allocation.SessionID == ready.SessionID && status.SessionID == ready.SessionID && status.SessionStorage == "header-available" && status.Metadata != nil && status.Metadata.SessionID == ready.SessionID && status.WriterPresent && status.PrefixStatus == "snapshot-eof" && status.Result == nil
}
