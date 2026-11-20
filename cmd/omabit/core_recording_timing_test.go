package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPTimingQueryAndExplicitIndexedExport(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(os.TempDir(), "rt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	metadataRaw, err := os.ReadFile("../../internal/recording/testdata/original-metadata1/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := recording.ReadMetadata(metadataRaw)
	if err != nil {
		t.Fatal(err)
	}
	const revision = uint64(9007199254740993)
	meta.RunnerID, meta.Mode, meta.InstanceID, meta.ConfigurationRevision = "runner", "simulation", "alpha", revision
	meta.ContainerID, meta.OperationID, meta.RequestID = strings.Repeat("c", 64), "op-timing", "start-timing"
	metadataRaw, err = meta.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	metaHash := sha256.Sum256(metadataRaw)
	header := recording.Header{MetadataSHA256: metaHash, Segment: 1, FirstSequence: 1, StartedUnixNano: meta.AttachedAt.UnixNano()}
	id, _ := hex.DecodeString(meta.SessionID)
	copy(header.SessionID[:], id)
	var encoded bytes.Buffer
	encoder, err := recording.NewEncoder(&encoded, header)
	if err != nil {
		t.Fatal(err)
	}
	const wall = int64(1795046400123456789)
	for i := 0; i < 8; i++ {
		kind := recording.Stdout
		if i%2 != 0 {
			kind = recording.Stderr
		}
		if _, err = encoder.Append(kind, []byte("private-payload-marker\x00\x1b[2K\xff\n"), wall-int64(i), uint64(i)); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string][]byte{"metadata.json": metadataRaw, "frames.obr": encoded.Bytes()}
	digest := func(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
	in := service.RecordingTimingInput{RecordingExportInput: service.RecordingExportInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, InstanceID: "alpha", JobID: strings.Repeat("d", 32), JobSHA256: strings.Repeat("e", 64), Representation: "original", IncludePrivateData: true}, Query: recording.TimingQuery{Kind: recording.Stdout, Limit: 2, ContextBefore: 1}}
	manifest := recording.ExportManifest{Version: 1, Metadata: meta, MetadataSHA256: digest(metadataRaw), Representation: "original", ChunkBytes: recording.ExportChunkBytes, Coverage: recording.ExportCoverage{VerifiedFrames: 8, OutputBytes: uint64(8 * len("private-payload-marker\x00\x1b[2K\xff\n")), FirstUnixNano: wall, LastUnixNano: wall - 7, Tail: "snapshot-eof"}, RetainedFilesComplete: true, ReadAt: time.Now().UTC(), Basis: "real-codec private CLI/official-SDK fixture; no live runtime"}
	for _, name := range []string{"metadata.json", "frames.obr"} {
		data := files[name]
		manifest.Files = append(manifest.Files, recording.ExportFile{Name: name, Bytes: int64(len(data)), SHA256: digest(data), Chunks: []string{digest(data)}})
	}
	manifestHash, err := manifest.Digest()
	if err != nil {
		t.Fatal(err)
	}
	result := recording.ExportManifestResult{Session: recording.SessionDescriptor{JobID: in.JobID, JobSHA256: in.JobSHA256, RunnerID: meta.RunnerID, Mode: meta.Mode, InstanceID: meta.InstanceID, ConfigurationRevision: revision, ContainerID: meta.ContainerID, OperationID: meta.OperationID, RequestID: meta.RequestID, CreatedAt: meta.AttachedAt}, Status: recording.SavedStatus{State: "capture-incomplete", ResultCode: "capture-incomplete", SessionID: meta.SessionID, SessionStorage: "header-available"}, Manifest: manifest, ManifestSHA256: manifestHash}
	var variant atomic.Value
	variant.Store("ok")
	var reads atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.Method == "GET" && q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.RecordingExportFeature}}
			if v == "observer" {
				h.AccessRole = "observer"
			}
			if v == "missing-capability" {
				h.Features = nil
			}
			json.NewEncoder(w).Encode(h)
			return
		}
		reads.Add(1)
		if v == "transport-lost" {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingExportManifestPath {
			var request recording.ExportRequest
			if json.NewDecoder(q.Body).Decode(&request) != nil || request.Validate() != nil || request.JobID != in.JobID {
				w.WriteHeader(400)
				return
			}
			out := result
			if v == "wrong-reply" {
				out.Session.InstanceID = "other"
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingExportPartPath {
			var request recording.ExportPartRequest
			if json.NewDecoder(q.Body).Decode(&request) != nil || request.Validate() != nil || request.JobID != in.JobID {
				w.WriteHeader(400)
				return
			}
			part := recording.ExportPart{Version: 1, Request: request, Data: append([]byte(nil), files[request.FileName]...), ReadAt: manifest.ReadAt}
			if v == "bad-part" && request.FileName == "frames.obr" {
				part.Data[0] ^= 1
			}
			json.NewEncoder(w).Encode(part)
			return
		}
		t.Error("unexpected operation", q.Method, q.URL.Path)
		w.WriteHeader(404)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recording-timing-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, name := range []string{"ok", "observer", "missing-capability", "no-consent", "redacted", "wrong-reply", "bad-part", "stale-source", "transport-lost"} {
		variant.Store(name)
		request := in
		if name == "no-consent" {
			request.IncludePrivateData = false
		}
		if name == "redacted" {
			request.Representation = "redacted"
		}
		if name == "stale-source" {
			request.Query.ExpectedSourceSHA256 = strings.Repeat("f", 64)
		}
		raw, _ := json.Marshal(request)
		before := reads.Load()
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", "runtime.recording-timing.v1", "--input-json", string(raw)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.runtime.recording-timing.v1", Arguments: json.RawMessage(raw)})
		if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal("timing CLI/MCP mismatch", name, err, string(cli), tool)
		}
		if name == "ok" {
			var out struct {
				Data recording.TimingQueryResult `json:"data"`
			}
			if cliErr != nil || tool.IsError || json.Unmarshal(cli, &out) != nil || len(out.Data.Entries) != 2 || out.Data.Entries[0].Sequence != 1 || out.Data.Entries[1].Sequence != 3 || out.Data.Entries[0].UnixNano != wall || out.Data.Source.Session.ConfigurationRevision != revision || out.Data.Source.Session.OperationID != "op-timing" || !out.Data.HasMore || out.Data.NextAfterSequence != 3 || reads.Load() != before+6 || bytes.Contains(cli, []byte("private-payload-marker")) {
				t.Fatal("timing metadata/authority lost", cliErr, string(cli))
			}
		} else {
			expected := int32(2)
			if name == "observer" || name == "missing-capability" || name == "no-consent" || name == "redacted" {
				expected = 0
			}
			if name == "bad-part" {
				expected = 6
			}
			if cliErr == nil || !tool.IsError || reads.Load() != before+expected || bytes.Contains(cli, []byte("op-timing")) || bytes.Contains(cli, []byte("private-payload-marker")) {
				t.Fatal("timing refusal leaked or retried", name, cliErr, string(cli), reads.Load()-before)
			}
			if name == "transport-lost" {
				var out struct{ Error *service.Problem }
				if json.Unmarshal(cli, &out) != nil || out.Error == nil || out.Error.Dispatch != "read-failed" {
					t.Fatal("timing read became uncertain mutation")
				}
			}
		}
	}
	variant.Store("ok")
	outputDir := t.TempDir()
	if err := os.Chmod(outputDir, 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(outputDir, "indexed.zip")
	raw, err := coreTestCommand(t, "--config-dir", config, "recording-export", "--runner", "dev", "--expected-runner-id", "runner", "--instance", "alpha", "--job", in.JobID, "--job-sha256", in.JobSHA256, "--original", "--include-private-data", "--timing-index", "--output", output).CombinedOutput()
	var exported struct {
		Status string
		Index  recording.TimingIndexSummary `json:"timing_index"`
	}
	if err != nil || json.Unmarshal(raw, &exported) != nil || exported.Status != "exported" || exported.Index.Frames != 8 || !exported.Index.VerifiedPrefixComplete || exported.Index.RuntimeComplete {
		t.Fatal("actual indexed export failed", err, string(raw))
	}
	z, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	found := false
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil || f.Mode().Perm() != 0600 {
			t.Fatal("index ZIP entry invalid", err)
		}
		if original, ok := files[f.Name]; ok && !bytes.Equal(original, data) {
			t.Fatal("original ZIP bytes changed")
		}
		if f.Name == "timing-index.jsonl" {
			found = true
			if digest(data) != exported.Index.IndexSHA256 || bytes.Contains(data, []byte("private-payload-marker")) {
				t.Fatal("actual index mismatched")
			}
		}
	}
	if !found {
		t.Fatal("indexed ZIP omitted index")
	}
}
