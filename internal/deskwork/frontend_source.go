package deskwork

import (
	"context"
	"errors"
	"os"
)

// ReadFrontendFiles reads a complete verified stage. It performs no Clay mark
// conversion: built assets must retain their exact original octets.
func ReadFrontendFiles(ctx context.Context, directory string, expected Manifest) (FrontendManifest, []SourceFile, error) {
	m, err := NewFrontendManifest(expected)
	if err != nil {
		return m, nil, err
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		return FrontendManifest{}, nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return FrontendManifest{}, nil, err
	}
	defer root.Close()
	files := make([]SourceFile, 0, len(m.Files))
	for _, f := range m.Files {
		if err = ctx.Err(); err != nil {
			return FrontendManifest{}, nil, err
		}
		raw, e := ReadSourceFile(root, f.File)
		if e != nil {
			return FrontendManifest{}, nil, e
		}
		files = append(files, SourceFile{Path: f.File.Path, Bytes: raw})
	}
	after, err := Snapshot(directory)
	if err != nil {
		return FrontendManifest{}, nil, err
	}
	current, err := os.Lstat(directory)
	if err != nil || !os.SameFile(identity, current) || after.Digest != expected.Digest {
		return FrontendManifest{}, nil, errors.New("frontend stage changed")
	}
	if err = validateFrontendFiles(m, files); err != nil {
		return FrontendManifest{}, nil, err
	}
	return m, files, ctx.Err()
}
