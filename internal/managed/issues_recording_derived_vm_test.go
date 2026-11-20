//go:build linux && amd64

package managed

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
)

// Invoked only by the fixed VM helper after all host/source/fixture guards.
// Saved archives and query/manifest bytes also permit a separate read-only,
// non-product decoder to verify native timing and transformation evidence.
func recordingUsageDerivedExports(t *testing.T, ctx context.Context, read func(string, string, any) json.RawMessage, binary, config, proofRoot string, in service.RecordingExportInput, original recording.ExportManifest, expected map[string][]byte) map[string]any {
	t.Helper()
	archive := func(choice, suffix string, flags ...string) ([]byte, map[string][]byte) {
		t.Helper()
		path := filepath.Join(proofRoot, suffix+"-"+in.JobID+".zip")
		args := []string{"--config-dir", config, "recording-export", "--runner", in.Name, "--expected-runner-id", in.RunnerID, "--instance", in.InstanceID, "--job", in.JobID, "--job-sha256", in.JobSHA256, "--" + choice, "--include-private-data", "--output", path}
		args = append(args, flags...)
		if raw, err := exec.CommandContext(ctx, binary, args...).CombinedOutput(); err != nil {
			t.Fatal("native derived ZIP command failed", suffix, err, string(raw))
		}
		info, err := os.Stat(path)
		g02Check(t, err)
		if info.Mode().Perm() != 0600 || info.Size() > 4<<20 {
			t.Fatal("bounded private derived archive required")
		}
		raw, err := os.ReadFile(path)
		g02Check(t, err)
		reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		g02Check(t, err)
		files := map[string][]byte{}
		for _, file := range reader.File {
			if file.Mode().Perm() != 0600 || file.UncompressedSize64 > 4<<20 || files[file.Name] != nil {
				t.Fatal("invalid native derived ZIP entry")
			}
			r, err := file.Open()
			g02Check(t, err)
			data, err := io.ReadAll(io.LimitReader(r, 4<<20))
			r.Close()
			g02Check(t, err)
			files[file.Name] = data
		}
		return raw, files
	}
	indexedZIP, indexed := archive("original", "indexed", "--timing-index")
	if len(indexed) != len(expected)+3 {
		t.Fatal("indexed archive inventory changed")
	}
	for name, raw := range expected {
		if !bytes.Equal(indexed[name], raw) {
			t.Fatal("indexed export changed original", name)
		}
	}
	lines := bytes.Split(bytes.TrimSuffix(indexed["timing-index.jsonl"], []byte("\n")), []byte("\n"))
	var header recording.TimingIndexRecord
	g02Check(t, json.Unmarshal(lines[0], &header))
	if header.Type != "header" || header.Source == nil || header.Source.Session.JobID != in.JobID || header.Source.Coverage != original.Coverage {
		t.Fatal("index source differs from exact original")
	}
	var entries []recording.TimingEntry
	for _, line := range lines[1 : len(lines)-1] {
		var row recording.TimingIndexRecord
		g02Check(t, json.Unmarshal(line, &row))
		if row.Type != "frame" || row.Entry == nil {
			t.Fatal("missing native indexed frame")
		}
		entries = append(entries, *row.Entry)
	}
	var end recording.TimingIndexRecord
	g02Check(t, json.Unmarshal(lines[len(lines)-1], &end))
	if len(entries) < 18 || len(entries) > 256 || uint64(len(entries)) != original.Coverage.VerifiedFrames || end.Type != "end" || end.End == nil || !end.End.AllSourceFilesVerified || !end.End.VerifiedPrefixComplete || end.End.RuntimeComplete || end.End.Coverage != original.Coverage {
		t.Fatal("native index lacks full verified completion")
	}
	var queries []recording.TimingQueryResult
	for _, query := range []recording.TimingQuery{
		{Limit: 256, ContextBefore: 2},
		{Limit: 17, ContextBefore: 2},
		{Limit: 17, AfterSequence: 17, ExpectedSourceSHA256: header.SourceSHA256, ContextBefore: 2},
		{Limit: 256, Kind: recording.Exit, FromUnixNano: entries[len(entries)-1].UnixNano, ThroughUnixNano: entries[len(entries)-1].UnixNano, ContextBefore: 2},
	} {
		var first recording.TimingQueryResult
		for _, adapter := range []string{"cli", "mcp"} {
			var response recording.TimingQueryResult
			g02Check(t, json.Unmarshal(read(adapter, "runtime.recording-timing.v1", service.RecordingTimingInput{RecordingExportInput: in, Query: query}), &response))
			if response.Index.SourceSHA256 != header.SourceSHA256 || response.Index.Coverage != original.Coverage || !response.Index.AllSourceFilesVerified || !response.Index.VerifiedPrefixComplete || response.Index.RuntimeComplete || !reflect.DeepEqual(response.Source, *header.Source) {
				t.Fatal("native shared query lost exact verified source")
			}
			var want []recording.TimingEntry
			for _, entry := range entries {
				if entry.Sequence > query.AfterSequence && (query.Kind == 0 || entry.Kind == query.Kind) && (query.FromUnixNano == 0 || entry.UnixNano >= query.FromUnixNano) && (query.ThroughUnixNano == 0 || entry.UnixNano <= query.ThroughUnixNano) {
					want = append(want, entry)
				}
			}
			more := len(want) > int(query.Limit)
			want = want[:min(len(want), int(query.Limit))]
			if !reflect.DeepEqual(response.Entries, want) || response.HasMore != more || more && response.NextAfterSequence != want[len(want)-1].Sequence {
				t.Fatal("native timing filter/continuation differs from exported index")
			}
			if len(want) > 0 && want[0].Sequence > 3 {
				prior := entries[want[0].Sequence-4]
				if response.ContextFrom == nil || response.ContextFrom.Sequence != prior.Sequence || response.ContextFrom.Digest != prior.Digest {
					t.Fatal("native timing context lost verified preceding cursor")
				}
			}
			response.ReadAt = first.ReadAt // Only the independent read time may differ.
			if adapter == "cli" {
				first = response
			} else if !reflect.DeepEqual(first, response) {
				t.Fatal("actual CLI/SDK timing results differ")
			}
		}
		queries = append(queries, first)
	}
	redactedInput := service.RecordingRedactedExportInput{Target: in.Target, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Representation: "redacted", IncludePrivateData: true}
	var manifest, other recording.RedactedExportManifest
	g02Check(t, json.Unmarshal(read("cli", "runtime.recording-redacted-export-manifest.v1", redactedInput), &manifest))
	g02Check(t, json.Unmarshal(read("mcp", "runtime.recording-redacted-export-manifest.v1", redactedInput), &other))
	g02Check(t, manifest.Validate())
	if !reflect.DeepEqual(manifest, other) || manifest.SourceSHA256 != header.SourceSHA256 || manifest.Summary.Coverage != original.Coverage {
		t.Fatal("native derived manifest differs across adapters/source")
	}
	var assembled bytes.Buffer
	for index, digest := range manifest.Chunks {
		request := service.RecordingRedactedExportPartInput{RecordingRedactedExportInput: redactedInput, SourceSHA256: manifest.SourceSHA256, ZIPBytes: manifest.ZIPBytes, ZIPSHA256: manifest.ZIPSHA256, ChunkIndex: index, ChunkSHA256: digest}
		var first recording.RedactedExportPart
		for _, adapter := range []string{"cli", "mcp"} {
			var part recording.RedactedExportPart
			g02Check(t, json.Unmarshal(read(adapter, "runtime.recording-redacted-export-part.v1", request), &part))
			g02Check(t, part.Validate())
			if part.Request.SourceSHA256 != manifest.SourceSHA256 || part.Request.ZIPSHA256 != manifest.ZIPSHA256 || part.Request.ZIPBytes != manifest.ZIPBytes || part.Request.ChunkIndex != index || part.Request.ChunkSHA256 != digest {
				t.Fatal("derived native part substituted request")
			}
			if adapter == "cli" {
				first = part
			} else if !reflect.DeepEqual(first, part) {
				t.Fatal("derived native part differs across interfaces")
			}
		}
		assembled.Write(first.Data)
	}
	redactedZIP, redacted := archive("redacted", "redacted")
	redactedHash, indexHash := sha256.Sum256(redactedZIP), sha256.Sum256(indexedZIP)
	if uint64(len(redactedZIP)) != manifest.ZIPBytes || hex.EncodeToString(redactedHash[:]) != manifest.ZIPSHA256 || !bytes.Equal(redactedZIP, assembled.Bytes()) || len(redacted) != 3 || redacted["manifest.json"] == nil || redacted["redacted-frames.jsonl"] == nil || redacted["coverage.json"] == nil {
		t.Fatal("actual redacted CLI archive differs from exact shared parts")
	}
	var technical recording.RedactedBundleManifest
	var coverage recording.RedactedBundleSummary
	g02Check(t, json.Unmarshal(redacted["manifest.json"], &technical))
	g02Check(t, json.Unmarshal(redacted["coverage.json"], &coverage))
	if technical.Representation != "redacted" || !technical.ExactBindingsOmitted || technical.OriginalEncodedFilesIncluded || technical.SourceSHA256 != header.SourceSHA256 || !reflect.DeepEqual(coverage, manifest.Summary) {
		t.Fatal("redacted native ZIP lacks required derivation evidence")
	}
	return map[string]any{"indexed_zip_sha256": hex.EncodeToString(indexHash[:]), "indexed_zip_base64": base64.StdEncoding.EncodeToString(indexedZIP), "timing_queries": queries, "redacted_manifest": manifest, "redacted_zip_base64": base64.StdEncoding.EncodeToString(redactedZIP), "redacted_parts_compared_both_interfaces": len(manifest.Chunks)}
}
