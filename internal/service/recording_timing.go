package service

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type RecordingTimingInput struct {
	RecordingExportInput
	Query recording.TimingQuery `json:"query"`
}

func (in RecordingTimingInput) Validate() error {
	if in.RecordingExportInput.Validate() != nil || in.Query.Validate() != nil {
		return invalid()
	}
	return nil
}

func (s *Service) registerRecordingTiming() error {
	return selected(s, description("runtime.recording-timing.v1", "Scan exact operator-private closed original recording parts for payload-free timing metadata. Requires representation=original and include_private_data=true. Filters inclusive wall time/stream and orders by sequence because wall time may reverse. Bounded complete verified-prefix scan; match pages continue only with exact source_sha256. Returns a cursor for separately authorized original/redacted surrounding context. No payload text search, filesystem selector, background job, recording mutation or runtime-health/coverage claim.", nil, []string{"local-metadata-read", "saved-runner-read", "private-runtime-read"}, []string{protocol.RecordingExportFeature}, "operator", "exact runner/instance/job/digest; complete hash-verified files, metadata/codec chain and stable source digest; no paths or caller-written prior authority", "retry this read with the same job/source; changed source refuses continuation; failures withhold all matches; never starts or adopts a recording"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingTimingInput) (recording.TimingQueryResult, error) {
		c := e.Client(true)
		defer c.Close()
		manifest, err := c.RecordingExportManifest(ctx, in.request())
		if err != nil {
			return recording.TimingQueryResult{}, err
		}
		if err = e.Identity().Check(manifest.Session.RunnerID, manifest.Session.Mode); err != nil {
			return recording.TimingQueryResult{}, err
		}
		return recording.QueryTiming(ctx, in.request(), manifest, in.Query, c.RecordingExportPart)
	})
}
