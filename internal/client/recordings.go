package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
)

func (c *Client) RecordingCatalog(ctx context.Context, in recording.CatalogRequest) (recording.Catalog, error) {
	var out recording.Catalog
	if err := in.Validate(); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", protocol.RecordingCatalogPath, in, &out, true); err != nil {
		return out, err
	}
	if out.Version != 1 || out.RunnerID != in.RunnerID || out.InstanceID != in.InstanceID || out.ReadAt.IsZero() || out.Sessions == nil || len(out.Sessions) > recording.MaxCatalogSessions {
		return recording.Catalog{}, errors.New("invalid recording catalog selection")
	}
	seen := map[string]bool{}
	for _, s := range out.Sessions {
		if s.RunnerID != in.RunnerID || s.InstanceID != in.InstanceID || !recording.APIHex(s.JobID, 32) || !recording.APIHex(s.JobSHA256, 64) || s.ConfigurationRevision == 0 || seen[s.JobID] || s.JobID == in.AfterJobID {
			return recording.Catalog{}, errors.New("changed recording catalog binding")
		}
		seen[s.JobID] = true
	}
	if out.NextJobID != "" && (len(out.Sessions) != recording.MaxCatalogSessions || out.NextJobID != out.Sessions[len(out.Sessions)-1].JobID) {
		return recording.Catalog{}, errors.New("invalid recording catalog cursor")
	}
	return out, nil
}

func (c *Client) RecordingPage(ctx context.Context, in recording.PageRequest) (recording.ReadWindow, error) {
	var out recording.ReadWindow
	if err := in.Validate(); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", protocol.RecordingPagePath, in, &out, true); err != nil {
		return out, err
	}
	if out.Version != 1 || out.Session.JobID != in.JobID || out.Session.JobSHA256 != in.JobSHA256 || out.Session.RunnerID != in.RunnerID || out.Session.InstanceID != in.InstanceID || out.Representation != in.Representation || !reflect.DeepEqual(out.From, in.Cursor) || out.ReadAt.IsZero() || out.CompleteExport || out.FrameCount < 0 || out.FrameCount > recording.MaxReadFrames {
		return recording.ReadWindow{}, errors.New("invalid recording page selection")
	}
	if in.Representation == "redacted" && len(out.Original) != 0 || in.Representation == "original" && out.Redacted != nil {
		return recording.ReadWindow{}, errors.New("recording representation changed")
	}
	if out.Metadata != nil {
		m := out.Metadata
		s := out.Session
		raw, err := m.Bytes()
		digest := sha256.Sum256(raw)
		if err != nil || hex.EncodeToString(digest[:]) != out.MetadataSHA256 || m.RunnerID != s.RunnerID || m.InstanceID != s.InstanceID || m.Mode != s.Mode || m.ContainerID != s.ContainerID || m.ConfigurationRevision != s.ConfigurationRevision || m.OperationID != s.OperationID || m.RequestID != s.RequestID || m.SessionID != out.Status.SessionID {
			return recording.ReadWindow{}, errors.New("recording metadata changed")
		}
	}
	if out.FrameCount > 0 && out.Metadata == nil || in.Representation == "original" && len(out.Original) != out.FrameCount || in.Representation == "redacted" && out.FrameCount > 0 && (out.Redacted == nil || len(out.Redacted.Frames) != out.FrameCount || out.Redacted.Representation != "escaped-literal-plain-text") {
		return recording.ReadWindow{}, errors.New("recording frame coverage changed")
	}
	if out.Next != nil && (out.Metadata == nil || out.Next.Version != 1 || out.Next.SessionID != out.Metadata.SessionID || out.Next.MetadataSHA256 != out.MetadataSHA256 || !recording.APIHex(out.Next.Digest, 64) || out.Next.Sequence == 0) {
		return recording.ReadWindow{}, errors.New("recording cursor changed")
	}
	return out, nil
}

func (c *Client) RecordingControls(ctx context.Context, in recording.ControlReadRequest) (recording.ControlStatus, error) {
	var out recording.ControlStatus
	if err := in.Validate(); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", protocol.RecordingControlStatusPath, in, &out, true); err != nil {
		return out, err
	}
	if out.Version != 1 || out.State.Validate() != nil || out.State.RunnerID != in.RunnerID || out.ReadAt.IsZero() || out.Capacity != nil && (out.Capacity.Validate() != nil || out.Capacity.RunnerID != in.RunnerID) || !out.PilotEnabled && out.CapacityStatus != "pilot-disabled" || out.PilotEnabled && out.CapacityStatus != "available" && out.CapacityStatus != "unknown" || (out.CapacityStatus == "available") != (out.Capacity != nil) {
		return recording.ControlStatus{}, errors.New("invalid recording control status binding")
	}
	return out, nil
}
func (c *Client) ApplyRecordingControl(ctx context.Context, in recording.ControlRequest) (recording.ControlApplyResult, error) {
	var out recording.ControlApplyResult
	if err := in.Validate(); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", protocol.RecordingControlApplyPath, in, &out, true); err != nil {
		return out, err
	}
	if out.Receipt.Validate() != nil || !reflect.DeepEqual(out.Receipt.Event.Request, in) {
		return recording.ControlApplyResult{}, errors.New("invalid recording control receipt binding; lookup original request")
	}
	return out, nil
}
func (c *Client) LookupRecordingControl(ctx context.Context, in recording.ControlLookupRequest) (recording.ControlLookupResult, error) {
	var out recording.ControlLookupResult
	if err := in.Validate(); err != nil {
		return out, err
	}
	if err := c.do(ctx, "POST", protocol.RecordingControlLookupPath, in, &out, true); err != nil {
		return out, err
	}
	if out.Version != 1 || out.RunnerID != in.RunnerID || out.RequestID != in.RequestID || out.Found != (out.Receipt != nil) || out.Receipt != nil && (out.Receipt.Validate() != nil || out.Receipt.Event.Request.RunnerID != in.RunnerID || out.Receipt.Event.Request.RequestID != in.RequestID) {
		return recording.ControlLookupResult{}, errors.New("invalid recording control lookup binding")
	}
	return out, nil
}
