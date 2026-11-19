package service

import (
	"context"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

type RecordingControlsInput struct {
	Target
	IncludePrivateData bool `json:"include_private_data"`
}

func (i RecordingControlsInput) Validate() error {
	if i.Target.Validate() != nil || !i.IncludePrivateData {
		return invalid()
	}
	return nil
}

type RecordingControlLookupInput struct {
	Target
	RequestID          string `json:"request_id"`
	IncludePrivateData bool   `json:"include_private_data"`
}

func (i RecordingControlLookupInput) Validate() error {
	if i.Target.Validate() != nil || !protocol.ValidID(i.RequestID) || !i.IncludePrivateData {
		return invalid()
	}
	return nil
}

type RecordingControlApplyInput struct {
	Target
	Request recording.ControlRequest `json:"request"`
}

func (i RecordingControlApplyInput) Validate() error {
	if i.Target.Validate() != nil || i.Request.Validate() != nil || i.RunnerID != i.Request.RunnerID {
		return invalid()
	}
	return nil
}
func (s *Service) registerRecordingControls() error {
	caps := []string{protocol.RecordingControlFeature}
	read := []string{"local-metadata-read", "saved-runner-read", "private-runtime-read"}
	return errors.Join(
		selected(s, description("runtime.recording-controls.v1", "Read private saved recording policy, incident pins and conservative admission commitments. Every retained job charges its immutable maximum, including closed/unclaimed jobs. Age/automatic retirement are not implemented; no filesystem deletion or live probe.", nil, read, caps, "operator", "explicit enrolled runner and include_private_data=true", "read only; unknown storage refuses admission"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingControlsInput) (recording.ControlStatus, error) {
			c := e.Client(true)
			defer c.Close()
			return c.RecordingControls(ctx, recording.ControlReadRequest{RunnerID: in.RunnerID, IncludePrivateData: in.IncludePrivateData})
		}),
		selected(s, description("runtime.recording-control-lookup.v1", "Look up an exact original recording-control request for the authenticated actor. This is the recovery path after a lost write response and performs no mutation.", nil, read, caps, "operator", "exact enrolled runner/original request ID and include_private_data=true", "read original receipt; absence is not permission to invent a replacement request"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingControlLookupInput) (recording.ControlLookupResult, error) {
			c := e.Client(true)
			defer c.Close()
			return c.LookupRecordingControl(ctx, recording.ControlLookupRequest{RunnerID: in.RunnerID, RequestID: in.RequestID, IncludePrivateData: in.IncludePrivateData})
		}),
		selected(s, description("runtime.recording-control-apply.v1", "Configure bounded recording admission limits, or pin/unpin an exact saved job. Requires original request ID and exact prior control revision/digest. Changes affect new launches only; automatic retirement is unsupported and no raw data is deleted. This does not change application permissions.", nil, []string{"local-metadata-read", "journal-write", "policy-edit"}, caps, "operator", "exact selected runner, original request, policy or job/digest and prior control head", "exact replay returns original receipt; after lost response use runtime.recording-control-lookup.v1; never mint a replacement request ID"), func(ctx context.Context, _ *collection.Registry, e collection.Connection, in RecordingControlApplyInput) (recording.ControlApplyResult, error) {
			c := e.Client(true)
			defer c.Close()
			return c.ApplyRecordingControl(ctx, in.Request)
		}),
	)
}
