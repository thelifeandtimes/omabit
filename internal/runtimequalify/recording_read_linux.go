//go:build linux && amd64

package runtimequalify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

// ReadManagedRecordingWindow observes saved originals only. It never opens an
// engine transport, launches/claims a worker, repairs a file or changes retention.
func ReadManagedRecordingWindow(ctx context.Context, in recording.PageRequest) (recording.ReadWindow, error) {
	var out recording.ReadWindow
	if in.Validate() != nil || ctx.Err() != nil {
		return out, recording.ErrSelection
	}
	opts, err := recordingRootOptions(defaults(), false)
	if err != nil {
		return out, err
	}
	return readManagedRecordingWindow(ctx, opts, in)
}

func readManagedRecordingWindow(ctx context.Context, opts RecordingWorkerOptions, in recording.PageRequest) (recording.ReadWindow, error) {
	var out recording.ReadWindow
	if in.Validate() != nil || ctx.Err() != nil {
		return out, recording.ErrSelection
	}
	jobs, err := OpenRecordingJobs(opts.JobsRoot, opts.MaxJobs, opts.JobReserveFreeBytes)
	if err != nil {
		return out, err
	}
	defer jobs.Close()
	job, err := jobs.Read(in.JobID, in.JobSHA256)
	if err != nil || job.Operation.RunnerID != in.RunnerID || job.Operation.Target.InstanceID != in.InstanceID {
		return out, recording.ErrSelection
	}
	status, err := readRecordingWorkerStatus(ctx, opts, in.JobID, in.JobSHA256)
	if err != nil {
		return out, err
	}
	op := job.Operation
	out = recording.ReadWindow{Version: 1, Session: recording.SessionDescriptor{JobID: job.ID, JobSHA256: in.JobSHA256, RunnerID: op.RunnerID, Mode: op.Mode, InstanceID: op.Target.InstanceID, RuntimeID: op.RuntimeID, ConfigurationRevision: op.ConfigurationRevision, ContainerID: op.ContainerID, OperationID: op.OperationID, RequestID: op.RequestID, CreatedAt: job.CreatedAt}, Status: recording.SavedStatus{State: status.State, SessionID: status.SessionID, SessionIDBasis: status.SessionIDBasis, SessionStorage: status.SessionStorage, WriterPresent: status.WriterPresent}, Representation: in.Representation, From: in.Cursor, Tail: "not-scanned", ReadAt: time.Now().UTC(), Basis: "bounded verified saved prefix; EOF is a read-time snapshot, not process exit or a complete export; redaction is derived and requires owner review before sharing"}
	if status.Result != nil {
		out.Status.ResultCode = status.Result.Code
		out.Status.ClosedDurably = status.Result.Capture.ClosedDurably
		out.Status.ExitSaved = status.Result.Capture.ExitSaved
		out.Status.GapReason = status.Result.Capture.GapReason
	}
	if status.SessionStorage != "header-available" {
		return out, nil
	}
	if status.Metadata == nil {
		return recording.ReadWindow{}, recording.ErrSelection
	}
	s, err := recording.OpenStore(opts.RecordingsRoot, recording.Limits{MaxSessionBytes: job.Limits.MaxSessionBytes, MaxSegmentBytes: job.Limits.MaxSegmentBytes, ReserveFreeBytes: job.Limits.ReserveFreeBytes})
	if err != nil {
		return recording.ReadWindow{}, err
	}
	defer s.Close()
	page, err := s.ReadPage(ctx, status.SessionID, in.Cursor, recording.MaxReadFrames, recording.MaxReadBytes)
	if err != nil || !job.MatchesAdmission(page.Metadata) || status.Worker == nil || page.Metadata.SessionID != status.SessionID || page.Metadata.Identity.HostBootID != status.Worker.HostBootID || page.Metadata.Identity.RecorderNamespaces != status.Worker.Namespaces {
		return recording.ReadWindow{}, recording.ErrSelection
	}
	raw, err := page.Metadata.Bytes()
	if err != nil {
		return recording.ReadWindow{}, err
	}
	savedMetadata, err := status.Metadata.Bytes()
	if err != nil || !bytes.Equal(raw, savedMetadata) {
		return recording.ReadWindow{}, recording.ErrSelection
	}
	digest := sha256.Sum256(raw)
	out.Metadata = &page.Metadata
	out.MetadataSHA256 = hex.EncodeToString(digest[:])
	out.Tail = page.Tail
	out.ReadAt = page.ReadAt
	out.FrameCount = len(page.Frames)
	out.Status.WriterPresent = page.WriterPresent
	if page.Next.Sequence != 0 {
		next := page.Next
		out.Next = &next
	}
	if in.Representation == "original" {
		for _, f := range page.Frames {
			export, err := recording.ExportOriginalFrame(f)
			if err != nil {
				return recording.ReadWindow{}, err
			}
			out.Original = append(out.Original, export)
		}
	} else if len(page.Frames) > 0 {
		options := recording.RedactionOptions{}
		if in.Cursor == nil && page.Frames[0].Sequence == 1 && op.PreviousSessionID == "" {
			options.VerifiedLineStarts = []recording.Kind{recording.Stdout, recording.Stderr}
		}
		view, err := recording.RenderRedactedFrames(page.Frames, options)
		if err != nil {
			return recording.ReadWindow{}, err
		}
		out.Redacted = &view
	}
	// Existing local/SSH RPC readers cap replies at1MiB. Refuse a derived
	// window that cannot fit; never silently truncate a frame/span or switch
	// to original bytes after an explicit redacted choice.
	encoded, err := json.Marshal(out)
	if err != nil || len(encoded) > 900<<10 {
		return recording.ReadWindow{}, recording.ErrLimit
	}
	return out, nil
}
