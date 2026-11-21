//go:build linux

package recording

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func timingIndexFixture(t *testing.T, variant string) (*Store, string, ExportRequest, ExportManifestResult) {
	t.Helper()
	s, dir := segmentStore(t, 4<<20, 350<<10)
	m := captureMetadata()
	w, err := s.Create(m)
	if err != nil {
		t.Fatal(err)
	}
	if variant != "empty" && variant != "exit-only" {
		for i := 0; i < 12; i++ {
			payload := append([]byte("private-payload-marker cookie=never-in-index\x00\x1b[2K\xff\n"), bytes.Repeat([]byte{'x'}, 60<<10)...)
			kind := Stdout
			if i%2 != 0 {
				kind = Stderr
			}
			if _, err = w.Append(kind, payload, m.AttachedAt.UnixNano()-int64(i), uint64(i)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if strings.HasPrefix(variant, "exit") {
		data, err := (ExitEvent{Version: 1, ContainerID: m.ContainerID, StartedUnixNano: m.AttachedAt.UnixNano(), FinishedUnixNano: m.AttachedAt.UnixNano() + 1, Code: 0}).Bytes()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Append(Exit, data, m.AttachedAt.UnixNano()+1, 13); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, m.SessionID)
	if variant == "exit-extra" {
		paths, err := filepath.Glob(filepath.Join(root, "frames*.obr"))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, segmentName(uint64(len(paths)+1))), []byte("invalid retained file after exit"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if variant == "incomplete" || variant == "partial-restart" || variant == "corrupt" {
		paths, err := filepath.Glob(filepath.Join(root, "frames*.obr"))
		if err != nil {
			t.Fatal(err)
		}
		last := paths[len(paths)-1]
		if filepath.Base(last) == "frames.obr" && len(paths) > 1 {
			last = paths[len(paths)-2]
		}
		f, err := os.OpenFile(last, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			t.Fatal(err)
		}
		tail := []byte{1, 2, 3}
		if variant == "corrupt" {
			tail = make([]byte, frameHeaderBytes)
		}
		_, err = f.Write(tail)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	if variant == "restart" || variant == "partial-restart" {
		at := m.AttachedAt.Add(time.Hour)
		w, err = s.Continue(t.Context(), m, at)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Append(Stdout, []byte("private-payload-marker after restart\n"), at.UnixNano(), 1); err != nil {
			t.Fatal(err)
		}
		if err = w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := s.ExportManifest(t.Context(), m.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	in := exportSelection(m)
	return s, root, in, bundleResult(manifest, in)
}

func TestTimingIndexMatchesEncodedOffsetsEpochsAndCompleteVerifiedPrefix(t *testing.T) {
	for _, variant := range []string{"pass", "empty", "incomplete", "corrupt", "restart", "partial-restart", "exit", "exit-only", "exit-extra"} {
		t.Run(variant, func(t *testing.T) {
			s, root, in, result := timingIndexFixture(t, variant)
			before := fileSnapshot(t, root)
			var output bytes.Buffer
			summary, err := WriteTimingIndex(t.Context(), in, result, s.ExportPart, &output)
			if err != nil || summary.Coverage != result.Manifest.Coverage || summary.Frames != result.Manifest.Coverage.VerifiedFrames || !summary.AllSourceFilesVerified || !summary.VerifiedPrefixComplete || summary.RuntimeComplete || summary.Bytes != uint64(output.Len()) {
				t.Fatal("index lost coverage", variant, err, summary)
			}
			h := sha256.Sum256(output.Bytes())
			if summary.IndexSHA256 != hex.EncodeToString(h[:]) || bytes.Contains(output.Bytes(), []byte("private-payload-marker")) || bytes.Contains(output.Bytes(), []byte("never-in-index")) || bytes.Contains(output.Bytes(), []byte{0x1b}) {
				t.Fatal("index leaked data or hash mismatched")
			}
			decode := json.NewDecoder(bytes.NewReader(output.Bytes()))
			var count uint64
			var last TimingIndexRecord
			seenReverse := false
			var wall int64
			for {
				var row TimingIndexRecord
				err := decode.Decode(&row)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				switch row.Type {
				case "header":
					if row.Source == nil || row.SourceSHA256 != summary.SourceSHA256 || row.Source.MetadataSHA256 != result.Manifest.MetadataSHA256 {
						t.Fatal("wrong source header")
					}
				case "frame":
					count++
					e := row.Entry
					if e == nil || e.Sequence != count || e.EncodedBytes != int64(frameHeaderBytes)+int64(e.PayloadBytes)+32 {
						t.Fatal("invalid row offset", e)
					}
					data, ok := before[e.File]
					if !ok || e.Offset < int64(headerBytes) || e.Offset+e.EncodedBytes > int64(len(data)) {
						t.Fatal("index outside source file")
					}
					encoded := data[e.Offset : e.Offset+e.EncodedBytes]
					if binary.BigEndian.Uint64(encoded) != e.Sequence || int64(binary.BigEndian.Uint64(encoded[8:])) != e.UnixNano || binary.BigEndian.Uint64(encoded[16:]) != e.ElapsedNano || Kind(encoded[24]) != e.Kind || binary.BigEndian.Uint32(encoded[25:]) != e.PayloadBytes || hex.EncodeToString(encoded[len(encoded)-32:]) != e.Digest {
						t.Fatal("index differs from encoded original")
					}
					d, err := NewDecoder(bytes.NewReader(data))
					if err != nil || d.Header.Segment != e.Segment || d.Header.StartedUnixNano != e.EpochUnixNano {
						t.Fatal("receive epoch lost", err)
					}
					if wall > e.UnixNano {
						seenReverse = true
					}
					wall = e.UnixNano
				case "end":
					if row.End == nil || row.End.Coverage != summary.Coverage || !row.End.AllSourceFilesVerified || !row.End.VerifiedPrefixComplete || row.End.RuntimeComplete {
						t.Fatal("invalid end record")
					}
				default:
					t.Fatal("unknown record type")
				}
				last = row
			}
			if count != summary.Frames || last.Type != "end" || variant != "empty" && variant != "exit-only" && !seenReverse {
				t.Fatal("prefix/order incomplete")
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
				t.Fatal("index changed original evidence")
			}
			var again bytes.Buffer
			result.Manifest.ReadAt = result.Manifest.ReadAt.Add(time.Second)
			result.ManifestSHA256, err = result.Manifest.Digest()
			if err != nil {
				t.Fatal(err)
			}
			other, err := WriteTimingIndex(t.Context(), in, result, s.ExportPart, &again)
			if err != nil || other.SourceSHA256 != summary.SourceSHA256 || !bytes.Equal(output.Bytes(), again.Bytes()) {
				t.Fatal("volatile read timestamp changed content index", err)
			}
		})
	}
}

func TestTimingOutputCoverageSeparatesRetainedOutputOrderWallClockAndLifecycle(t *testing.T) {
	for _, variant := range []string{"pass", "exit", "exit-only", "empty", "corrupt"} {
		t.Run(variant, func(t *testing.T) {
			s, root, in, result := timingIndexFixture(t, variant)
			before := fileSnapshot(t, root)
			// Limit/filter affect matches only. Earliest output is computed from
			// the complete verified source, including matches not in this page.
			query, err := QueryTiming(t.Context(), in, result, TimingQuery{Kind: Exit, Limit: 1}, s.ExportPart)
			if err != nil || query.Index.OutputCoverage == nil {
				t.Fatal("missing retained output coverage", err)
			}
			c := query.Index.OutputCoverage
			if c.Version != 1 || c.Basis == "" || query.Index.RuntimeComplete {
				t.Fatal("coverage lost scope")
			}
			if variant == "empty" || variant == "exit-only" {
				if c.Status != "no-output-in-verified-prefix" || c.Frames != 0 || c.Bytes != 0 || c.FirstSequence != 0 || c.LastSequence != 0 || c.FirstUnixNano != 0 || c.LastUnixNano != 0 || c.EarliestUnixNano != 0 || c.LatestUnixNano != 0 {
					t.Fatal("lifecycle-only/empty prefix invented output", c)
				}
			} else {
				first := result.Manifest.Metadata.AttachedAt.UnixNano()
				if c.Status != "verified-output-prefix" || c.Frames != 12 || c.Bytes != result.Manifest.Coverage.OutputBytes || c.FirstSequence != 1 || c.LastSequence != 12 || c.FirstUnixNano != first || c.LastUnixNano != first-11 || c.EarliestUnixNano != first-11 || c.LatestUnixNano != first {
					t.Fatal("output chronology conflated sequence/wall/lifecycle time", c)
				}
			}
			if variant == "exit-only" && (!query.Index.Coverage.ExitSaved || query.Index.Frames != 1) || variant == "exit" && (query.Index.Coverage.LastUnixNano <= c.LatestUnixNano || query.Index.Frames <= c.Frames) || variant == "corrupt" && query.Index.Coverage.Tail != "corrupt" {
				t.Fatal("output summary erased lifecycle or damaged-tail evidence")
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
				t.Fatal("coverage read changed retained originals")
			}
		})
	}
}

func TestTimingIndexWithholdsCompletionForChangedSourceOrFailedOutput(t *testing.T) {
	s, root, in, result := timingIndexFixture(t, "pass")
	before := fileSnapshot(t, root)
	for _, variant := range []string{"part", "whole", "coverage", "fetch", "cancel", "writer"} {
		t.Run(variant, func(t *testing.T) {
			copy := result
			copy.Manifest.Files = append([]ExportFile(nil), result.Manifest.Files...)
			if variant == "whole" {
				copy.Manifest.Files[1].SHA256 = strings.Repeat("f", 64)
			}
			if variant == "coverage" {
				copy.Manifest.Coverage.VerifiedFrames++
			}
			var err error
			copy.ManifestSHA256, err = copy.Manifest.Digest()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if variant == "cancel" {
				cancel()
			}
			calls := 0
			fetch := func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
				calls++
				if variant == "fetch" && calls == 2 {
					return ExportPart{}, errors.New("injected disconnect")
				}
				out, err := s.ExportPart(ctx, in)
				if variant == "part" && calls == 2 && err == nil {
					out.Data[0] ^= 1
				}
				return out, err
			}
			var output bytes.Buffer
			var writer io.Writer = &output
			if variant == "writer" {
				writer = timingShortWriter{}
			}
			summary, err := WriteTimingIndex(ctx, in, copy, fetch, writer)
			if err == nil || summary != (TimingIndexSummary{}) || bytes.Contains(output.Bytes(), []byte(`"type":"end"`)) {
				t.Fatal("failed index completed", variant, err)
			}
			if variant == "fetch" && calls != 2 {
				t.Fatal("source fetch retried")
			}
			if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
				t.Fatal("failure changed original")
			}
		})
	}
}

type timingShortWriter struct{}

func (timingShortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestTimingIndexOptionalBundlePreservesDefaultAndWithholdsPartialArchive(t *testing.T) {
	s, root, in, result := timingIndexFixture(t, "partial-restart")
	before := fileSnapshot(t, root)
	var output bytes.Buffer
	summary, err := WriteOriginalBundleWithTimingIndex(t.Context(), in, result, s.ExportPart, &output)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != len(result.Manifest.Files)+3 {
		t.Fatal("indexed bundle entries missing")
	}
	found := false
	for _, file := range z.File {
		if file.Mode().Perm() != 0600 {
			t.Fatal("non-private index entry")
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if raw, ok := before[file.Name]; ok && !bytes.Equal(raw, data) {
			t.Fatal("indexed original changed", file.Name)
		}
		if file.Name == "timing-index.jsonl" {
			found = true
			h := sha256.Sum256(data)
			if hex.EncodeToString(h[:]) != summary.IndexSHA256 || bytes.Contains(data, []byte("private-payload-marker")) {
				t.Fatal("bad derived index")
			}
		}
	}
	if !found {
		t.Fatal("separate index missing")
	}
	parts := 0
	for _, f := range result.Manifest.Files {
		parts += len(f.Chunks)
	}
	output.Reset()
	calls := 0
	fetch := func(ctx context.Context, in ExportPartRequest) (ExportPart, error) {
		calls++
		if calls == parts+2 {
			return ExportPart{}, errors.New("injected second-pass source loss")
		}
		return s.ExportPart(ctx, in)
	}
	got, err := WriteOriginalBundleWithTimingIndex(t.Context(), in, result, fetch, &output)
	if err == nil || got != (TimingIndexSummary{}) {
		t.Fatal("partial index returned completed bundle")
	}
	if _, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len())); err == nil {
		t.Fatal("partial archive gained completion directory")
	}
	if !reflect.DeepEqual(before, fileSnapshot(t, root)) {
		t.Fatal("indexed export changed original files")
	}
}
