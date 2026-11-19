//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/hex"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

// Export reads use the original immutable job and saved result, never the engine
// or live process observation. A missing result cannot be promoted to closure.
func recordingExportStore(ctx context.Context, opts RecordingWorkerOptions, in recording.ExportRequest) (*recording.Store, ManagedRecordingJob, RecordingWorkerStatus, error) {
	if in.Validate() != nil || ctx.Err() != nil {
		return nil, ManagedRecordingJob{}, RecordingWorkerStatus{}, recording.ErrSelection
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, opts.MaxJobs, opts.JobReserveFreeBytes)
	if err != nil {
		return nil, ManagedRecordingJob{}, RecordingWorkerStatus{}, err
	}
	defer jobs.Close()
	job, err := jobs.Read(in.JobID, in.JobSHA256)
	if err != nil || job.Operation.RunnerID != in.RunnerID || job.Operation.Target.InstanceID != in.InstanceID {
		return nil, job, RecordingWorkerStatus{}, recording.ErrSelection
	}
	status, err := readRecordingWorkerStatus(ctx, opts, in.JobID, in.JobSHA256)
	storage := status.SessionStorage
	if err != nil || status.Result == nil || status.Worker == nil || status.Metadata == nil || status.WriterPresent || storage != "header-available" && storage != "incomplete" && storage != "corrupt" || !job.MatchesAdmission(*status.Metadata) || status.Metadata.Identity == nil || status.Metadata.Identity.HostBootID != status.Worker.HostBootID || status.Metadata.Identity.RecorderNamespaces != status.Worker.Namespaces {
		return nil, job, status, recording.ErrSelection
	}
	s, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
	return s, job, status, err
}

func ReadManagedRecordingExportManifest(ctx context.Context, in recording.ExportRequest) (recording.ExportManifestResult, error) {
	opts, err := recordingRootOptions(defaults(), false)
	if err != nil {
		return recording.ExportManifestResult{}, err
	}
	return readManagedRecordingExportManifest(ctx, opts, in)
}

func readManagedRecordingExportManifest(ctx context.Context, opts RecordingWorkerOptions, in recording.ExportRequest) (recording.ExportManifestResult, error) {
	s, job, status, err := recordingExportStore(ctx, opts, in)
	if err != nil {
		return recording.ExportManifestResult{}, err
	}
	defer s.Close()
	manifest, err := s.ExportManifest(ctx, status.SessionID)
	if err != nil || !job.MatchesAdmission(manifest.Metadata) {
		return recording.ExportManifestResult{}, recording.ErrSelection
	}
	want, _ := status.Metadata.Digest()
	got, _ := manifest.Metadata.Digest()
	if want != got {
		return recording.ExportManifestResult{}, recording.ErrSelection
	}
	op := job.Operation
	result := status.Result
	out := recording.ExportManifestResult{Session: recording.SessionDescriptor{JobID: job.ID, JobSHA256: in.JobSHA256, RunnerID: op.RunnerID, Mode: op.Mode, InstanceID: op.Target.InstanceID, RuntimeID: op.RuntimeID, ConfigurationRevision: op.ConfigurationRevision, ContainerID: op.ContainerID, OperationID: op.OperationID, RequestID: op.RequestID, CreatedAt: job.CreatedAt}, Status: recording.SavedStatus{State: status.State, SessionID: status.SessionID, SessionIDBasis: status.SessionIDBasis, SessionStorage: status.SessionStorage, ResultCode: result.Code, ClosedDurably: result.Capture.ClosedDurably, ExitSaved: result.Capture.ExitSaved, GapReason: result.Capture.GapReason}, Manifest: manifest}
	out.ManifestSHA256, err = manifest.Digest()
	if err != nil || out.Validate(in) != nil {
		return recording.ExportManifestResult{}, recording.ErrSelection
	}
	return out, nil
}

func ReadManagedRecordingExportPart(ctx context.Context, in recording.ExportPartRequest) (recording.ExportPart, error) {
	opts, err := recordingRootOptions(defaults(), false)
	if err != nil {
		return recording.ExportPart{}, err
	}
	return readManagedRecordingExportPart(ctx, opts, in)
}

func readManagedRecordingExportPart(ctx context.Context, opts RecordingWorkerOptions, in recording.ExportPartRequest) (recording.ExportPart, error) {
	if in.Validate() != nil {
		return recording.ExportPart{}, recording.ErrSelection
	}
	s, _, status, err := recordingExportStore(ctx, opts, in.ExportRequest)
	if err != nil {
		return recording.ExportPart{}, err
	}
	defer s.Close()
	digest, err := status.Metadata.Digest()
	if err != nil || status.SessionID != in.SessionID || hex.EncodeToString(digest[:]) != in.MetadataSHA256 {
		return recording.ExportPart{}, recording.ErrSelection
	}
	return s.ExportPart(ctx, in)
}
