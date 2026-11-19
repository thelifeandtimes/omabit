package service

import (
	"context"
	"errors"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type RecordingCatalogInput struct {
	Target
	InstanceID         string `json:"instance_id"`
	AfterJobID         string `json:"after_job_id,omitempty"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (in RecordingCatalogInput) request() recording.CatalogRequest {
	return recording.CatalogRequest{RunnerID: in.RunnerID, InstanceID: in.InstanceID, AfterJobID: in.AfterJobID, IncludePrivateData: in.IncludePrivateData}
}
func (in RecordingCatalogInput) Validate() error {
	if in.Target.Validate() != nil || in.request().Validate() != nil {
		return invalid()
	}
	return nil
}

type RecordingPageInput struct {
	Target
	InstanceID         string            `json:"instance_id"`
	JobID              string            `json:"job_id"`
	JobSHA256          string            `json:"job_sha256"`
	Cursor             *recording.Cursor `json:"cursor,omitempty"`
	Representation     string            `json:"representation"`
	IncludePrivateData bool              `json:"include_private_data"`
}

func (in RecordingPageInput) request() recording.PageRequest {
	return recording.PageRequest{RunnerID: in.RunnerID, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Cursor: in.Cursor, Representation: in.Representation, IncludePrivateData: in.IncludePrivateData}
}
func (in RecordingPageInput) Validate() error {
	if in.Target.Validate() != nil || in.request().Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerRecordings() error {
	effects := []string{"local-metadata-read", "saved-runner-read", "private-runtime-read"}
	caps := []string{protocol.RecordingReadFeature}
	return errors.Join(
		selected(s, description("runtime.recordings.v1", "Read bounded operator-private saved recording launch history with include_private_data=true. No live probe, worker action, retention change or health claim.", nil, effects, caps, "operator", "explicit enrolled runner/instance and exact immutable job/digest descriptors", "read next catalog window with its exact after_job_id; never starts or adopts capture"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingCatalogInput) (recording.Catalog, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RecordingCatalog(ctx, in.request())
		}),
		selected(s, description("runtime.recording-page.v1", "Read an exact operator-private recording window. Explicit representation=redacted derives escaped plain text; original returns unchanged bytes as base64. Redaction needs owner review before sharing. EOF is not exit; this is not a complete export.", nil, effects, caps, "operator", "exact runner/instance/job/digest/cursor and explicit private representation; raw originals remain private", "poll the exact returned cursor for following; gaps/partial tails stay explicit and no capture is retried"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingPageInput) (recording.ReadWindow, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RecordingPage(ctx, in.request())
		}),
	)
}
