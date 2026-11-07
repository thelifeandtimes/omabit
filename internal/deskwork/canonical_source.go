package deskwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// ReadSourceFile reads one exact manifest entry without following replacement
// symlinks or accepting bytes different from the reviewed snapshot.
func ReadSourceFile(root *os.Root, item File) ([]byte, error) {
	if !ValidPath(item.Path) || !digestPattern.MatchString(item.SHA256) || item.Bytes < 0 || item.Bytes > MaxFileBytes {
		return nil, errors.New("invalid source entry")
	}
	before, err := root.Lstat(item.Path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() != item.Bytes {
		return nil, errors.New("source file changed")
	}
	f, err := root.Open(item.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, errors.New("source file replaced")
	}
	raw, err := io.ReadAll(io.LimitReader(f, item.Bytes+1))
	if err != nil {
		return nil, err
	}
	named, err := root.Lstat(item.Path)
	if err != nil || !named.Mode().IsRegular() || !os.SameFile(opened, named) {
		return nil, errors.New("source file replaced")
	}
	digest := sha256.Sum256(raw)
	if int64(len(raw)) != item.Bytes || hex.EncodeToString(digest[:]) != item.SHA256 {
		return nil, errors.New("source file changed")
	}
	return raw, nil
}

// SnapshotCanonical bounds each normalization batch and re-observes the Unix
// directory afterward. Its two manifests describe the same captured source;
// errors return no usable observation, rather than a false synchronized state.
func SnapshotCanonical(ctx context.Context, directory string, normalize func(context.Context, []SourceFile) (Manifest, error)) (raw, canonical Manifest, err error) {
	if normalize == nil {
		return raw, canonical, errors.New("mark normalizer unavailable")
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		return raw, canonical, err
	}
	raw, err = Snapshot(directory)
	if err != nil {
		return Manifest{}, Manifest{}, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return Manifest{}, Manifest{}, err
	}
	defer root.Close()
	all := []File{}
	batch := []SourceFile{}
	var bytes int64
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		normalized, e := normalize(ctx, batch)
		if e != nil {
			return e
		}
		if normalized.Validate() != nil || len(normalized.Files) != len(batch) {
			return errors.New("normalizer changed file selection")
		}
		for i, f := range normalized.Files {
			if f.Path != batch[i].Path || f.Bytes != 0 {
				return errors.New("normalizer changed file selection")
			}
		}
		all = append(all, normalized.Files...)
		batch = []SourceFile{}
		bytes = 0
		return nil
	}
	for _, f := range raw.Files {
		if err = ctx.Err(); err != nil {
			return Manifest{}, Manifest{}, err
		}
		if len(batch) == 128 || bytes+f.Bytes > MaxFileBytes {
			if err = flush(); err != nil {
				return Manifest{}, Manifest{}, err
			}
		}
		data, e := ReadSourceFile(root, f)
		if e != nil {
			return Manifest{}, Manifest{}, e
		}
		batch = append(batch, SourceFile{Path: f.Path, Bytes: data})
		bytes += f.Bytes
	}
	if err = flush(); err != nil {
		return Manifest{}, Manifest{}, err
	}
	after, err := Snapshot(directory)
	if err != nil {
		return Manifest{}, Manifest{}, err
	}
	named, err := os.Lstat(directory)
	if err != nil || !os.SameFile(identity, named) || raw.Digest != after.Digest {
		return Manifest{}, Manifest{}, errors.New("source directory changed during comparison")
	}
	if err = ctx.Err(); err != nil {
		return Manifest{}, Manifest{}, err
	}
	canonical, err = NewManifest(all)
	if err != nil {
		return Manifest{}, Manifest{}, err
	}
	return raw, canonical, nil
}
