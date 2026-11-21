//go:build linux && amd64

package runtimequalify

import (
	"context"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
)

func (s *DeskRuntime) ExportSourceFiles(ctx context.Context, desk string, revision uint64, paths []string) ([]deskwork.ExportedFile, error) {
	request, err := deskwork.ExportFiles(s.ship, desk, revision, paths)
	if err != nil {
		return nil, err
	}
	value, err := s.call(ctx, request)
	if err != nil {
		return nil, err
	}
	return deskwork.DecodeExportedFiles(value, paths)
}
