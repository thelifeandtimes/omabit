//go:build linux && amd64

package managed

import (
	"context"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func (realBackend) ReadRecording(ctx context.Context, in recording.PageRequest) (recording.ReadWindow, error) {
	return runtimequalify.ReadManagedRecordingWindow(ctx, in)
}

func (realBackend) RecordingExportManifest(ctx context.Context, in recording.ExportRequest) (recording.ExportManifestResult, error) {
	return runtimequalify.ReadManagedRecordingExportManifest(ctx, in)
}

func (realBackend) RecordingExportPart(ctx context.Context, in recording.ExportPartRequest) (recording.ExportPart, error) {
	return runtimequalify.ReadManagedRecordingExportPart(ctx, in)
}
