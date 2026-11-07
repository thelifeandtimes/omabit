package deskwork

import (
	"context"
	"errors"
	"os"
	"time"
)

func ValidSourceName(name string) bool { return deskName.MatchString(name) }

type SourceReader interface {
	Observe(context.Context, string) (Observation, error)
	ClayFiles(context.Context, string, uint64) (Manifest, error)
	NormalizeFiles(context.Context, string, uint64, []SourceFile) (Manifest, error)
	AutocommitActive(context.Context) (bool, error)
}

type SourceStatus struct {
	Version       int         `json:"version"`
	Observation   Observation `json:"observation"`
	ReadAt        time.Time   `json:"read_at"`
	Unix          Manifest    `json:"unix"`
	CanonicalUnix Manifest    `json:"canonical_unix"`
	Clay          Manifest    `json:"clay"`
	Comparison    Comparison  `json:"comparison"`
	Differences   []string    `json:"differences"`
	Autocommit    string      `json:"autocommit"`
}

// InspectSource compares one configured Unix source directory against the named
// desk. It does not infer a Clay mount mapping from the directory's name. Without
// a shared baseline it reports 'different', not an invented direction of change.
// No source bytes or filesystem paths are returned in the observation.
func InspectSource(ctx context.Context, reader SourceReader, directory, desk string, baseline *Manifest) (SourceStatus, error) {
	var out SourceStatus
	if reader == nil || !deskName.MatchString(desk) {
		return out, errors.New("invalid source inspection")
	}
	if baseline != nil && baseline.Validate() != nil {
		return out, errors.New("invalid source comparison baseline")
	}
	identity, err := os.Lstat(directory)
	if err != nil {
		return out, errors.New("source directory unavailable")
	}
	before, err := reader.Observe(ctx, desk)
	if err != nil || before.Validate() != nil || !before.Exists {
		return out, errors.New("source desk observation unavailable")
	}
	clay, err := reader.ClayFiles(ctx, desk, before.Revision)
	if err != nil || clay.Validate() != nil {
		return out, errors.New("canonical Clay inventory unavailable")
	}
	raw, canonical, err := SnapshotCanonical(ctx, directory, func(ctx context.Context, files []SourceFile) (Manifest, error) {
		return reader.NormalizeFiles(ctx, desk, before.Revision, files)
	})
	if err != nil {
		return out, errors.New("source comparison unavailable; source changed or a mark could not be converted")
	}
	auto := "unknown"
	if active, e := reader.AutocommitActive(ctx); e == nil {
		auto = "inactive"
		if active {
			auto = "active"
		}
	}
	after, err := reader.Observe(ctx, desk)
	if err != nil || after != before {
		return out, errors.New("Clay changed during source comparison")
	}
	// The final ship read can take time; do not return an already changed Unix view.
	final, err := Snapshot(directory)
	named, statErr := os.Lstat(directory)
	if err != nil || statErr != nil || !os.SameFile(identity, named) || final.Digest != raw.Digest {
		return out, errors.New("Unix changed during source comparison")
	}
	base := clay
	if baseline != nil {
		base = *baseline
	}
	comparison, err := Compare(base, canonical, clay)
	if err != nil {
		return out, err
	}
	delta, err := Compare(clay, canonical, clay)
	if err != nil {
		return out, err
	}
	if baseline == nil && comparison.State != "synchronized" {
		comparison.State = "different"
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	return SourceStatus{Version: 1, Observation: before, ReadAt: time.Now().UTC(), Unix: raw, CanonicalUnix: canonical, Clay: clay, Comparison: comparison, Differences: delta.UnixChanges, Autocommit: auto}, nil
}
