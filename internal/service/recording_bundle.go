package service

import (
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"io"
)

// WriteRecordingBundle composes the same public operations used by CLI/MCP. It
// grants no extra authority and writes only to its caller-supplied stream.
func (s *Service) WriteRecordingBundle(ctx context.Context, in RecordingExportInput, output io.Writer) error {
	return s.writeRecordingBundle(ctx, in, output, nil)
}

func (s *Service) WriteRecordingBundleWithTimingIndex(ctx context.Context, in RecordingExportInput, output io.Writer) (recording.TimingIndexSummary, error) {
	var summary recording.TimingIndexSummary
	err := s.writeRecordingBundle(ctx, in, output, &summary)
	return summary, err
}

func (s *Service) writeRecordingBundle(ctx context.Context, in RecordingExportInput, output io.Writer, timing *recording.TimingIndexSummary) error {
	if in.Validate() != nil {
		return invalid()
	}
	raw, _ := json.Marshal(in)
	result := s.Call(ctx, "runtime.recording-export-manifest.v1", raw)
	if result.Error != nil {
		return errors.New(result.Error.Code + ": " + result.Error.Message)
	}
	manifest, ok := result.Data.(recording.ExportManifestResult)
	if !ok {
		return recording.ErrSelection
	}
	fetch := func(ctx context.Context, part recording.ExportPartRequest) (recording.ExportPart, error) {
		request := RecordingExportPartInput{RecordingExportInput: in, SessionID: part.SessionID, MetadataSHA256: part.MetadataSHA256, FileName: part.FileName, FileBytes: part.FileBytes, ChunkIndex: part.ChunkIndex, ChunkSHA256: part.ChunkSHA256}
		raw, _ := json.Marshal(request)
		result := s.Call(ctx, "runtime.recording-export-part.v1", raw)
		if result.Error != nil {
			return recording.ExportPart{}, errors.New(result.Error.Code + ": " + result.Error.Message)
		}
		value, ok := result.Data.(recording.ExportPart)
		if !ok {
			return recording.ExportPart{}, recording.ErrSelection
		}
		return value, nil
	}
	if timing != nil {
		summary, err := recording.WriteOriginalBundleWithTimingIndex(ctx, in.request(), manifest, fetch, output)
		if err == nil {
			*timing = summary
		}
		return err
	}
	return recording.WriteOriginalBundle(ctx, in.request(), manifest, fetch, output)
}
