package deskwork

import (
	"context"
	"errors"
	"os"
)

// ReadSourceFiles reads exactly a complete bounded manifest, preserving every
// octet. It neither selects frontend MIME types nor requires a frontend index.
// Confinement, per-file identity/hash and the final whole-tree/parent identity
// check are shared by backend source and frontend artifact readers.
func ReadSourceFiles(ctx context.Context, directory string, expected Manifest) ([]SourceFile, error) {
	if err := expected.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !identity.IsDir() || identity.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("source must be an ordinary directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	files := make([]SourceFile, 0, len(expected.Files))
	for _, f := range expected.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, err := ReadSourceFile(root, f)
		if err != nil {
			return nil, err
		}
		files = append(files, SourceFile{Path: f.Path, Bytes: raw})
	}
	after, err := Snapshot(directory)
	if err != nil {
		return nil, err
	}
	current, err := os.Lstat(directory)
	if err != nil || !os.SameFile(identity, current) || after.Digest != expected.Digest {
		return nil, errors.New("source stage changed")
	}
	return files, ctx.Err()
}
