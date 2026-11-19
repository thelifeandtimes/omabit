package service

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type RecordingExportInput struct {
	Target
	InstanceID         string `json:"instance_id"`
	JobID              string `json:"job_id"`
	JobSHA256          string `json:"job_sha256"`
	Representation     string `json:"representation"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (in RecordingExportInput) request() recording.ExportRequest {
	return recording.ExportRequest{RunnerID: in.RunnerID, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Representation: in.Representation, IncludePrivateData: in.IncludePrivateData}
}
func (in RecordingExportInput) Validate() error {
	if in.Target.Validate() != nil || in.request().Validate() != nil {
		return invalid()
	}
	return nil
}

type RecordingExportPartInput struct {
	RecordingExportInput
	SessionID      string `json:"session_id"`
	MetadataSHA256 string `json:"metadata_sha256"`
	FileName       string `json:"file_name"`
	FileBytes      int64  `json:"file_bytes"`
	ChunkIndex     int    `json:"chunk_index"`
	ChunkSHA256    string `json:"chunk_sha256"`
}

func (in RecordingExportPartInput) part() recording.ExportPartRequest {
	return recording.ExportPartRequest{ExportRequest: in.RecordingExportInput.request(), SessionID: in.SessionID, MetadataSHA256: in.MetadataSHA256, FileName: in.FileName, FileBytes: in.FileBytes, ChunkIndex: in.ChunkIndex, ChunkSHA256: in.ChunkSHA256}
}
func (in RecordingExportPartInput) Validate() error {
	if in.RecordingExportInput.Validate() != nil || in.part().Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerRecordingExports() error {
	effects := []string{"local-metadata-read", "saved-runner-read", "private-runtime-read"}
	caps := []string{protocol.RecordingExportFeature}
	return errors.Join(
		selected(s, description("runtime.recording-export-manifest.v1", "Prepare an owner-private original recording export manifest for a closed saved job. Contains complete encoded-file/chunk hashes and verified-prefix timing/gap coverage; retained completeness never proves runtime completeness. No originals are redacted or deleted.", nil, effects, caps, "operator", "exact runner/instance/job/digest; include_private_data=true and representation=original", "save the unchanged manifest; fetch all its exact parts and verify complete file hashes; never substitutes another session"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingExportInput) (recording.ExportManifestResult, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RecordingExportManifest(ctx, in.request())
		}),
		selected(s, description("runtime.recording-export-part.v1", "Read one at most256KiB original encoded-file chunk as base64 using the exact saved manifest hash for that chunk. Never render terminal bytes, publish, write a local path or claim a complete export from one part. Redacted requests refuse.", nil, effects, caps, "operator", "exact runner/job/session/metadata/file size/chunk index and SHA256; explicit private original choice", "repeat the exact read after disconnect; changed/missing chunks refuse without rewind, substitution or repair"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingExportPartInput) (recording.ExportPart, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RecordingExportPart(ctx, in.part())
		}),
	)
}
