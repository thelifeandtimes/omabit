package deskwork

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type deploymentReaderFixture struct {
	t                      *testing.T
	observations           []Observation
	reads                  int
	clay                   map[uint64]Manifest
	normalized             map[uint64]Manifest
	failNormalization      bool
	normalizationRevisions []uint64
	normalizationPaths     [][]string
}

func (f *deploymentReaderFixture) Observe(ctx context.Context, desk string) (Observation, error) {
	if desk != "example" {
		f.t.Fatal("wrong desk", desk)
	}
	if err := ctx.Err(); err != nil {
		return Observation{}, err
	}
	i := f.reads
	f.reads++
	if i >= len(f.observations) {
		i = len(f.observations) - 1
	}
	return f.observations[i], nil
}
func (f *deploymentReaderFixture) ClayFiles(_ context.Context, desk string, revision uint64) (Manifest, error) {
	if desk != "example" {
		f.t.Fatal("wrong desk")
	}
	m, ok := f.clay[revision]
	if !ok {
		return Manifest{}, errors.New("unknown revision")
	}
	return m, nil
}
func (f *deploymentReaderFixture) NormalizeFiles(_ context.Context, desk string, revision uint64, files []SourceFile) (Manifest, error) {
	if desk != "example" {
		f.t.Fatal("wrong desk")
	}
	f.normalizationRevisions = append(f.normalizationRevisions, revision)
	selected := []string{}
	out := []File{}
	index := f.normalized[revision].Index()
	for _, file := range files {
		selected = append(selected, file.Path)
		v, ok := index[file.Path]
		if !ok {
			return Manifest{}, errors.New("mark unavailable at revision")
		}
		out = append(out, v)
	}
	f.normalizationPaths = append(f.normalizationPaths, selected)
	if f.failNormalization {
		return Manifest{}, errors.New("normalizer unavailable")
	}
	return NewManifest(out)
}
func deploymentRuntimeFixture(t *testing.T) (string, Manifest, *deploymentReaderFixture) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"new.custom": "new source", "shared.txt": "shared source"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, err := Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := Observation{Exists: true, Revision: 1, ClayHash: "0v123", Publication: "private"}
	f := &deploymentReaderFixture{t: t, observations: []Observation{before}, clay: map[uint64]Manifest{1: deployManifest(t, map[string]string{"shared.txt": "canonical shared", "generated.txt": "generated"}, true)}, normalized: map[uint64]Manifest{1: deployManifest(t, map[string]string{"shared.txt": "canonical shared"}, true)}}
	return dir, source, f
}
func TestDeploymentReviewUsesHistoricalMarksAndRejectsDrift(t *testing.T) {
	for _, kind := range []string{"ok", "source-changed", "desk-changed", "normalizer-failed", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			dir, source, f := deploymentRuntimeFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "source-changed":
				os.WriteFile(filepath.Join(dir, "new.custom"), []byte("edited"), 0600)
			case "desk-changed":
				next := f.observations[0]
				next.Revision++
				f.observations = append(f.observations, next)
			case "normalizer-failed":
				f.failNormalization = true
			case "cancelled":
				cancel()
			}
			got, err := InspectDeployment(ctx, f, "example", dir, source, EmptyOwnership())
			if kind == "ok" {
				if err != nil || got.Plan.Validate() != nil || len(got.Plan.Changes) != 1 || got.Plan.Changes[0].Path != "new.custom" || got.Observation.Revision != 1 || got.ReadAt.IsZero() {
					t.Fatal(got, err)
				}
				if len(f.normalizationPaths) != 1 || len(f.normalizationPaths[0]) != 1 || f.normalizationPaths[0][0] != "shared.txt" || f.normalizationRevisions[0] != 1 {
					t.Fatal("new mark normalized before it exists", f.normalizationPaths)
				}
			} else if err == nil {
				t.Fatal("unstable review accepted")
			}
		})
	}
}
func TestDeploymentResultReadsNewRevisionAndPreservesUncertainty(t *testing.T) {
	for _, kind := range []string{"ok", "no-new-revision", "source-changed", "desk-changed", "normalizer-failed", "generated-changed", "publication-changed"} {
		t.Run(kind, func(t *testing.T) {
			dir, source, f := deploymentRuntimeFixture(t)
			review, err := InspectDeployment(context.Background(), f, "example", dir, source, EmptyOwnership())
			if err != nil {
				t.Fatal(err)
			}
			after := Observation{Exists: true, Revision: 2, ClayHash: "0v456", Publication: "private"}
			f.observations = []Observation{after}
			f.reads = 0
			f.clay[2] = deployManifest(t, map[string]string{"new.custom": "canonical new", "shared.txt": "canonical shared", "generated.txt": "generated"}, true)
			f.normalized[2] = deployManifest(t, map[string]string{"new.custom": "canonical new", "shared.txt": "canonical shared"}, true)
			switch kind {
			case "no-new-revision":
				f.observations = []Observation{review.Observation}
			case "source-changed":
				os.WriteFile(filepath.Join(dir, "new.custom"), []byte("edited"), 0600)
			case "desk-changed":
				next := after
				next.Revision++
				f.observations = append(f.observations, next)
			case "normalizer-failed":
				f.failNormalization = true
			case "generated-changed":
				f.clay[2] = deployManifest(t, map[string]string{"new.custom": "canonical new", "shared.txt": "canonical shared", "generated.txt": "modified"}, true)
			case "publication-changed":
				f.observations[0].Publication = "published"
			}
			observed, owned, err := ObserveDeploymentResult(context.Background(), f, "example", dir, review.Observation, review.Plan)
			if kind == "ok" {
				if err != nil || observed != after || owned.Validate() != nil || len(owned.Raw.Files) != 1 || owned.Raw.Files[0].Path != "new.custom" {
					t.Fatal(observed, owned, err)
				}
			} else if err == nil || observed.Exists || owned.Version != 0 {
				t.Fatal("unverified result accepted", observed, owned, err)
			}
		})
	}
}
func TestCanonicalSubsetRequiresExactSelectedPaths(t *testing.T) {
	dir, _, f := deploymentRuntimeFixture(t)
	normalize := func(ctx context.Context, files []SourceFile) (Manifest, error) {
		return f.NormalizeFiles(ctx, "example", 1, files)
	}
	for _, paths := range [][]string{nil, {"missing.txt"}, {"shared.txt", "shared.txt"}, {"../outside.txt"}} {
		if _, _, err := SnapshotCanonicalPaths(context.Background(), dir, paths, normalize); err == nil {
			t.Fatal("invalid subset accepted", paths)
		}
	}
	raw, canonical, err := SnapshotCanonicalPaths(context.Background(), dir, []string{}, normalize)
	if err != nil || len(raw.Files) != 2 || len(canonical.Files) != 0 {
		t.Fatal(raw, canonical, err)
	}
}
