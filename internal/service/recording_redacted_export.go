package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type RecordingRedactedExportInput struct {
	Target
	InstanceID         string `json:"instance_id"`
	JobID              string `json:"job_id"`
	JobSHA256          string `json:"job_sha256"`
	Representation     string `json:"representation"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (in RecordingRedactedExportInput) request() recording.RedactedExportRequest {
	return recording.RedactedExportRequest{RunnerID: in.RunnerID, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Representation: in.Representation, IncludePrivateData: in.IncludePrivateData}
}
func (in RecordingRedactedExportInput) originalRequest() recording.ExportRequest {
	return recording.ExportRequest{RunnerID: in.RunnerID, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Representation: "original", IncludePrivateData: in.IncludePrivateData}
}
func (in RecordingRedactedExportInput) Validate() error {
	if in.Target.Validate() != nil || in.request().Validate() != nil {
		return invalid()
	}
	return nil
}

type RecordingRedactedExportPartInput struct {
	RecordingRedactedExportInput
	SourceSHA256 string `json:"source_sha256"`
	ZIPBytes     uint64 `json:"zip_bytes"`
	ZIPSHA256    string `json:"zip_sha256"`
	ChunkIndex   int    `json:"chunk_index"`
	ChunkSHA256  string `json:"chunk_sha256"`
}

func (in RecordingRedactedExportPartInput) requestPart() recording.RedactedExportPartRequest {
	return recording.RedactedExportPartRequest{RedactedExportRequest: in.RecordingRedactedExportInput.request(), SourceSHA256: in.SourceSHA256, ZIPBytes: in.ZIPBytes, ZIPSHA256: in.ZIPSHA256, ChunkIndex: in.ChunkIndex, ChunkSHA256: in.ChunkSHA256}
}
func (in RecordingRedactedExportPartInput) Validate() error {
	if in.RecordingRedactedExportInput.Validate() != nil || in.requestPart().Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerRecordingRedactedExports() error {
	effects := []string{"local-metadata-read", "saved-runner-read", "private-runtime-read"}
	caps := []string{protocol.RecordingExportFeature}
	return errors.Join(
		selected(s, description("runtime.recording-redacted-export-manifest.v1", "Prepare exact operator-private complete derived redacted ZIP hashes for a closed recording job. Requires representation=redacted and include_private_data=true. Finite credential rules, escaped text, omitted-span/timing/gap evidence and technical metadata; exact names/IDs/paths and original encoded files are omitted. Still requires owner review before sharing. No local path, background job, raw-original fallback or runtime/retention action. Full source is verified before returning the manifest.", nil, effects, caps, "operator", "exact runner/instance/job/digest; explicit redacted choice; complete source/ZIP hashes and fixed transformation policy", "save this immutable source/ZIP manifest; parts rederive the full source and refuse changed bytes/policy rather than substitute originals"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingRedactedExportInput) (recording.RedactedExportManifest, error) {
			c := e.Client(true)
			defer c.Close()
			source, err := c.RecordingExportManifest(ctx, in.originalRequest())
			if err != nil {
				return recording.RedactedExportManifest{}, err
			}
			if err = e.Identity().Check(source.Session.RunnerID, source.Session.Mode); err != nil {
				return recording.RedactedExportManifest{}, err
			}
			return recording.PlanRedactedExport(ctx, in.request(), source, c.RecordingExportPart)
		}),
		selected(s, description("runtime.recording-redacted-export-part.v1", "Read one at most256KiB chunk of the exact saved derived redacted ZIP. Every request re-verifies and re-derives the complete closed original source before returning any part; larger exports can be expensive. This avoids persistent derived cache/retention authority. Bytes are base64 archive data, never terminal rendering. No source change, original representation or incomplete result is accepted.", nil, effects, caps, "operator", "exact job/source/ZIP/size/chunk index/hash and explicit private redacted choice", "retry the identical read after loss; changed derivation/source refuses without raw fallback or silent manifest refresh"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingRedactedExportPartInput) (recording.RedactedExportPart, error) {
			c := e.Client(true)
			defer c.Close()
			source, err := c.RecordingExportManifest(ctx, in.originalRequest())
			if err != nil {
				return recording.RedactedExportPart{}, err
			}
			if err = e.Identity().Check(source.Session.RunnerID, source.Session.Mode); err != nil {
				return recording.RedactedExportPart{}, err
			}
			return recording.ReadRedactedExportPart(ctx, in.requestPart(), source, c.RecordingExportPart)
		}),
	)
}

// Local output orchestration uses exactly the same typed manifest/part handlers
// as MCP; it adds no arbitrary filesystem access to those handlers.
func (s *Service) WriteRedactedRecordingBundle(ctx context.Context, in RecordingRedactedExportInput, output io.Writer) (recording.RedactedBundleSummary, error) {
	var zero recording.RedactedBundleSummary
	if in.Validate() != nil {
		return zero, invalid()
	}
	raw, _ := json.Marshal(in)
	result := s.Call(ctx, "runtime.recording-redacted-export-manifest.v1", raw)
	if result.Error != nil {
		return zero, errors.New(result.Error.Code + ": " + result.Error.Message)
	}
	manifest, ok := result.Data.(recording.RedactedExportManifest)
	if !ok {
		return zero, recording.ErrSelection
	}
	err := recording.WriteRedactedExportParts(ctx, in.request(), manifest, func(ctx context.Context, part recording.RedactedExportPartRequest) (recording.RedactedExportPart, error) {
		request := RecordingRedactedExportPartInput{RecordingRedactedExportInput: in, SourceSHA256: part.SourceSHA256, ZIPBytes: part.ZIPBytes, ZIPSHA256: part.ZIPSHA256, ChunkIndex: part.ChunkIndex, ChunkSHA256: part.ChunkSHA256}
		raw, _ := json.Marshal(request)
		result := s.Call(ctx, "runtime.recording-redacted-export-part.v1", raw)
		if result.Error != nil {
			return recording.RedactedExportPart{}, errors.New(result.Error.Code + ": " + result.Error.Message)
		}
		out, ok := result.Data.(recording.RedactedExportPart)
		if !ok {
			return recording.RedactedExportPart{}, recording.ErrSelection
		}
		return out, nil
	}, output)
	if err != nil {
		return zero, err
	}
	return manifest.Summary, nil
}
