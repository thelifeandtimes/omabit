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

func TestCLIAndMCPPrivateRecordingHistoryAndExactPage(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp(os.TempDir(), "rec-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(socketDir) })
	socket := filepath.Join(socketDir, "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const revision = uint64(9007199254740993)
	descriptor := recording.SessionDescriptor{RunnerID: "runner", Mode: "development-managed", InstanceID: "alpha", RuntimeID: strings.Repeat("b", 32), ConfigurationRevision: revision, ContainerID: strings.Repeat("c", 64), JobID: strings.Repeat("d", 32), JobSHA256: strings.Repeat("e", 64), OperationID: "op-original", RequestID: "start-original", CreatedAt: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)}
	metadataBytes, err := os.ReadFile("../../internal/recording/testdata/original-metadata1/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := recording.ReadMetadata(metadataBytes)
	if err != nil {
		t.Fatal(err)
	}
	metadata.InstanceID, metadata.ConfigurationRevision, metadata.ContainerID = descriptor.InstanceID, descriptor.ConfigurationRevision, descriptor.ContainerID
	metadata.OperationID, metadata.RequestID = descriptor.OperationID, descriptor.RequestID
	metadataBytes, err = metadata.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	metadataHash := sha256.Sum256(metadataBytes)
	metadataDigest := hex.EncodeToString(metadataHash[:])
	payload := []byte("original private octets\x00\x1b[2K\xff\n")
	frame := recording.Frame{Segment: 1, Sequence: 1, UnixNano: 1795046400123456789, Kind: recording.Stdout, Digest: sha256.Sum256(payload), Data: payload}
	original, err := recording.ExportOriginalFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	redacted, err := recording.RenderRedactedFrames([]recording.Frame{frame}, recording.RedactionOptions{})
	if err != nil {
		t.Fatal(err)
	}
	hashBytes := func(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
	// An inert encoded-file fixture checks transport/selection parity; the real
	// storage codec and worker fixtures independently verify actual OBR files.
	exportFiles := map[string][]byte{"metadata.json": metadataBytes, "frames.obr": payload}
	exportManifest := recording.ExportManifest{Version: 1, Metadata: metadata, MetadataSHA256: metadataDigest, Representation: "original", ChunkBytes: recording.ExportChunkBytes, Coverage: recording.ExportCoverage{VerifiedFrames: 1, OutputBytes: uint64(len(payload)), FirstUnixNano: frame.UnixNano, LastUnixNano: frame.UnixNano, Tail: "corrupt"}, RetainedFilesComplete: true, ReadAt: descriptor.CreatedAt, Basis: "inert CLI/SDK original export transport fixture"}
	for _, name := range []string{"metadata.json", "frames.obr"} {
		data := exportFiles[name]
		exportManifest.Files = append(exportManifest.Files, recording.ExportFile{Name: name, Bytes: int64(len(data)), SHA256: hashBytes(data), Chunks: []string{hashBytes(data)}})
	}
	exportDigest, err := exportManifest.Digest()
	if err != nil {
		t.Fatal(err)
	}
	var variant atomic.Value
	variant.Store("ok")
	var reads atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.Method == "GET" && q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.RecordingReadFeature, protocol.RecordingExportFeature}}
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
			conn, _, e := w.(http.Hijacker).Hijack()
			if e != nil {
				t.Error(e)
				return
			}
			conn.Close()
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingExportManifestPath {
			var in recording.ExportRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.Validate() != nil || in.JobID != descriptor.JobID {
				w.WriteHeader(400)
				return
			}
			out := recording.ExportManifestResult{Session: descriptor, Status: recording.SavedStatus{State: "capture-incomplete", ResultCode: "capture-incomplete", SessionID: metadata.SessionID, SessionStorage: "header-available"}, Manifest: exportManifest, ManifestSHA256: exportDigest}
			if v == "wrong-reply" {
				out.Session.InstanceID = "other"
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingExportPartPath {
			var in recording.ExportPartRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.Validate() != nil || in.JobID != descriptor.JobID {
				w.WriteHeader(400)
				return
			}
			out := recording.ExportPart{Version: 1, Request: in, Data: exportFiles[in.FileName], ReadAt: descriptor.CreatedAt}
			if v == "wrong-reply" {
				out.Request.InstanceID = "other"
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingCatalogPath {
			var in recording.CatalogRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.RunnerID != "runner" || !in.IncludePrivateData {
				t.Error("catalog request changed")
				w.WriteHeader(400)
				return
			}
			d := descriptor
			if v == "wrong-reply" {
				d.RunnerID = "replacement"
			}
			json.NewEncoder(w).Encode(recording.Catalog{Version: 1, RunnerID: "runner", InstanceID: "alpha", Sessions: []recording.SessionDescriptor{d}, ReadAt: descriptor.CreatedAt, Basis: "saved-only inert fixture"})
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingPagePath {
			var in recording.PageRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.JobID != descriptor.JobID || in.JobSHA256 != descriptor.JobSHA256 || !in.IncludePrivateData {
				t.Error("page selection changed")
				w.WriteHeader(400)
				return
			}
			d := descriptor
			if v == "wrong-reply" {
				d.JobSHA256 = strings.Repeat("f", 64)
			}
			page := recording.ReadWindow{Version: 1, Session: d, Status: recording.SavedStatus{State: "job-saved", SessionID: metadata.SessionID, SessionIDBasis: "immutable-metadata", SessionStorage: "unknown"}, Metadata: &metadata, MetadataSHA256: metadataDigest, Representation: in.Representation, From: in.Cursor, Next: &recording.Cursor{Version: 1, SessionID: metadata.SessionID, MetadataSHA256: metadataDigest, Sequence: frame.Sequence, Digest: hex.EncodeToString(frame.Digest[:])}, FrameCount: 1, Tail: "not-scanned", ReadAt: descriptor.CreatedAt, Basis: "saved fixture"}
			if in.Representation == "original" {
				page.Original = []recording.OriginalFrameExport{original}
			} else {
				page.Redacted = &redacted
			}
			json.NewEncoder(w).Encode(page)
			return
		}
		t.Error("unexpected effect", q.Method, q.URL.Path)
		w.WriteHeader(404)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recordings-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	target := service.Target{Name: "dev", RunnerID: "runner"}
	for _, selection := range []struct{ operation, representation string }{{"runtime.recordings.v1", ""}, {"runtime.recording-page.v1", "original"}, {"runtime.recording-page.v1", "redacted"}, {"runtime.recording-export-manifest.v1", "original"}, {"runtime.recording-export-part.v1", "original"}} {
		operation := selection.operation
		for _, name := range []string{"ok", "observer", "missing-capability", "no-consent", "wrong-reply", "transport-lost"} {
			variant.Store(name)
			var in any = service.RecordingCatalogInput{Target: target, InstanceID: "alpha", IncludePrivateData: name != "no-consent"}
			if operation == "runtime.recording-page.v1" {
				in = service.RecordingPageInput{Target: target, InstanceID: "alpha", JobID: descriptor.JobID, JobSHA256: descriptor.JobSHA256, Representation: selection.representation, IncludePrivateData: name != "no-consent"}
			}
			exportInput := service.RecordingExportInput{Target: target, InstanceID: "alpha", JobID: descriptor.JobID, JobSHA256: descriptor.JobSHA256, Representation: "original", IncludePrivateData: name != "no-consent"}
			if operation == "runtime.recording-export-manifest.v1" {
				in = exportInput
			}
			if operation == "runtime.recording-export-part.v1" {
				in = service.RecordingExportPartInput{RecordingExportInput: exportInput, SessionID: metadata.SessionID, MetadataSHA256: metadataDigest, FileName: "frames.obr", FileBytes: int64(len(payload)), ChunkSHA256: hashBytes(payload)}
			}
			raw, _ := json.Marshal(in)
			before := reads.Load()
			cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", operation, "--input-json", string(raw)).CombinedOutput()
			tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + operation, Arguments: json.RawMessage(raw)})
			if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
				t.Fatal("CLI/MCP recording mismatch", operation, name, err, string(cli), tool)
			}
			if name == "ok" {
				if cliErr != nil || tool.IsError || reads.Load() != before+2 || operation != "runtime.recording-export-part.v1" && !bytes.Contains(cli, []byte("9007199254740993")) {
					t.Fatal("bounded exact read failed", operation, name, cliErr, string(cli))
				}
				if operation == "runtime.recording-export-part.v1" {
					var decoded struct {
						Data recording.ExportPart `json:"data"`
					}
					if json.Unmarshal(cli, &decoded) != nil || decoded.Data.Validate() != nil || !bytes.Equal(decoded.Data.Data, payload) {
						t.Fatal("exact exported chunk changed")
					}
				}
				if operation == "runtime.recording-page.v1" {
					var decoded struct {
						Data recording.ReadWindow `json:"data"`
					}
					if err := json.Unmarshal(cli, &decoded); err != nil {
						t.Fatal(err)
					}
					if decoded.Data.FrameCount != 1 || !bytes.Contains(cli, []byte("1795046400123456789")) || decoded.Data.MetadataSHA256 != metadataDigest || decoded.Data.Next == nil || decoded.Data.Next.Digest != hex.EncodeToString(frame.Digest[:]) {
						t.Fatal("nonempty frame binding changed", selection)
					}
					if selection.representation == "original" && (len(decoded.Data.Original) != 1 || !bytes.Equal(decoded.Data.Original[0].OriginalBytes, payload) || decoded.Data.Original[0].Frame.Digest != frame.Digest) {
						t.Fatal("original octets changed")
					}
					if selection.representation == "redacted" && (decoded.Data.Redacted == nil || len(decoded.Data.Redacted.Frames) != 1 || len(decoded.Data.Original) != 0) {
						t.Fatal("derived representation changed")
					}
				}
			} else {
				if cliErr == nil || !tool.IsError || bytes.Contains(cli, []byte("op-original")) {
					t.Fatal("bad selection exposed private history", operation, name, string(cli))
				}
				expected := int32(2)
				if name == "observer" || name == "missing-capability" || name == "no-consent" {
					expected = 0
				}
				if reads.Load() != before+expected {
					t.Fatal("read was retried or refusal dispatched", operation, name, reads.Load()-before)
				}
				if name == "transport-lost" {
					var envelope struct{ Error *service.Problem }
					if json.Unmarshal(cli, &envelope) != nil || envelope.Error == nil || envelope.Error.Dispatch != "read-failed" {
						t.Fatal("read became uncertain mutation", string(cli))
					}
				}
			}
		}
	}
	variant.Store("ok")
	outputDir := t.TempDir()
	if err := os.Chmod(outputDir, 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(outputDir, "original.zip")
	args := []string{"--config-dir", config, "recording-export", "--runner", "dev", "--expected-runner-id", "runner", "--instance", "alpha", "--job", descriptor.JobID, "--job-sha256", descriptor.JobSHA256, "--original", "--include-private-data", "--output", output}
	if raw, err := coreTestCommand(t, args...).CombinedOutput(); err != nil || !bytes.Contains(raw, []byte(`"status":"exported"`)) || !bytes.Contains(raw, []byte(`"runtime_complete":false`)) {
		t.Fatal("CLI archive failed", err, string(raw))
	}
	file, err := os.Stat(output)
	if err != nil || file.Mode().Perm() != 0600 {
		t.Fatal("archive is not private", err)
	}
	archive, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if expected, ok := exportFiles[file.Name]; ok {
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(r)
			r.Close()
			if err != nil || !bytes.Equal(got, expected) {
				t.Fatal("CLI archive changed original", file.Name, err)
			}
		}
	}
	before := reads.Load()
	if raw, err := coreTestCommand(t, args...).CombinedOutput(); err == nil || reads.Load() != before {
		t.Fatal("existing output was overwritten or contacted runner", err, string(raw))
	}
}
