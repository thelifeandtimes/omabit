package deskwork

import (
	"context"
)

// ReadFrontendFiles reads a complete verified stage. It performs no Clay mark
// conversion: built assets must retain their exact original octets.
func ReadFrontendFiles(ctx context.Context, directory string, expected Manifest) (FrontendManifest, []SourceFile, error) {
	m, err := NewFrontendManifest(expected)
	if err != nil {
		return m, nil, err
	}
	files, err := ReadSourceFiles(ctx, directory, expected)
	if err != nil {
		return FrontendManifest{}, nil, err
	}
	if err = validateFrontendFiles(m, files); err != nil {
		return FrontendManifest{}, nil, err
	}
	return m, files, ctx.Err()
}
