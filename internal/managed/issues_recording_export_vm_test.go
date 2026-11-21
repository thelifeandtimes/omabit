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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Called only after the fixed usage helper has verified its VM/source/uid/home/
// predecessor guard. No independent host entry or arbitrary path is exposed.
func recordingUsageOriginalExports(t *testing.T, ctx context.Context, session *mcp.ClientSession, binary, config, proofRoot, base string, target service.Target, original map[string]string) []map[string]any {
	t.Helper()
	read := func(adapter, operation string, in any) json.RawMessage {
		t.Helper()
		encoded, err := json.Marshal(in)
		g02Check(t, err)
		var raw []byte
		if adapter == "cli" {
			raw, err = exec.CommandContext(ctx, binary, "--config-dir", config, "call", operation, "--input-json", string(encoded)).Output()
			g02Check(t, err)
		} else {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + operation, Arguments: json.RawMessage(encoded)})
			g02Check(t, err)
			if result.IsError || len(result.Content) != 1 {
				t.Fatal("native export SDK read failed", operation)
			}
			text, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("native export result is not JSON text")
			}
			raw = []byte(text.Text)
		}
		var envelope struct {
			Data  json.RawMessage  `json:"data"`
			Error *service.Problem `json:"error"`
		}
		g02Check(t, json.Unmarshal(raw, &envelope))
		if envelope.Error != nil || envelope.Data == nil {
			t.Fatal("native export refused", operation, envelope.Error)
		}
		return envelope.Data
	}
	catalogInput := service.RecordingCatalogInput{Target: target, InstanceID: "recording-zod", IncludePrivateData: true}
	var cliCatalog, mcpCatalog recording.Catalog
	g02Check(t, json.Unmarshal(read("cli", "runtime.recordings.v1", catalogInput), &cliCatalog))
	g02Check(t, json.Unmarshal(read("mcp", "runtime.recordings.v1", catalogInput), &mcpCatalog))
	if len(cliCatalog.Sessions) != 2 || cliCatalog.NextJobID != "" || !reflect.DeepEqual(cliCatalog.Sessions, mcpCatalog.Sessions) {
		t.Fatal("exact two original closed jobs required")
	}
	var summaries []map[string]any
	for _, descriptor := range cliCatalog.Sessions {
		in := service.RecordingExportInput{Target: target, InstanceID: descriptor.InstanceID, JobID: descriptor.JobID, JobSHA256: descriptor.JobSHA256, Representation: "original", IncludePrivateData: true}
		selection := recording.ExportRequest{RunnerID: target.RunnerID, InstanceID: in.InstanceID, JobID: in.JobID, JobSHA256: in.JobSHA256, Representation: "original", IncludePrivateData: true}
		var cliManifest, mcpManifest recording.ExportManifestResult
		g02Check(t, json.Unmarshal(read("cli", "runtime.recording-export-manifest.v1", in), &cliManifest))
		g02Check(t, json.Unmarshal(read("mcp", "runtime.recording-export-manifest.v1", in), &mcpManifest))
		g02Check(t, cliManifest.Validate(selection))
		g02Check(t, mcpManifest.Validate(selection))
		if !reflect.DeepEqual(cliManifest.Session, descriptor) || !reflect.DeepEqual(cliManifest.Manifest.Files, mcpManifest.Manifest.Files) || !reflect.DeepEqual(cliManifest.Manifest.Metadata, mcpManifest.Manifest.Metadata) || cliManifest.Manifest.Coverage != mcpManifest.Manifest.Coverage || cliManifest.Manifest.RuntimeComplete || !cliManifest.Manifest.RetainedFilesComplete {
			t.Fatal("native CLI/MCP manifest changed its binding or coverage")
		}
		manifest := cliManifest.Manifest
		expected := map[string][]byte{}
		parts := 0
		for _, file := range manifest.Files {
			relative := "recordings/" + manifest.Metadata.SessionID + "/" + file.Name
			data, err := os.ReadFile(filepath.Join(base, relative))
			g02Check(t, err)
			hash := sha256.Sum256(data)
			if hex.EncodeToString(hash[:]) != original[relative] || file.SHA256 != original[relative] || file.Bytes != int64(len(data)) {
				t.Fatal("native manifest differs from independent original bytes", relative)
			}
			expected[file.Name] = data
			for index, digest := range file.Chunks {
				request := service.RecordingExportPartInput{RecordingExportInput: in, SessionID: manifest.Metadata.SessionID, MetadataSHA256: manifest.MetadataSHA256, FileName: file.Name, FileBytes: file.Bytes, ChunkIndex: index, ChunkSHA256: digest}
				for _, adapter := range []string{"cli", "mcp"} {
					var part recording.ExportPart
					g02Check(t, json.Unmarshal(read(adapter, "runtime.recording-export-part.v1", request), &part))
					g02Check(t, part.Validate())
					offset := index * recording.ExportChunkBytes
					if !bytes.Equal(part.Data, data[offset:min(offset+recording.ExportChunkBytes, len(data))]) {
						t.Fatal("native exact part changed original octets")
					}
				}
				parts++
			}
		}
		path := filepath.Join(proofRoot, "original-"+descriptor.JobID+".zip")
		command := exec.CommandContext(ctx, binary, "--config-dir", config, "recording-export", "--runner", target.Name, "--expected-runner-id", target.RunnerID, "--instance", descriptor.InstanceID, "--job", descriptor.JobID, "--job-sha256", descriptor.JobSHA256, "--original", "--include-private-data", "--output", path)
		if raw, err := command.CombinedOutput(); err != nil {
			t.Fatal("native original ZIP command failed", err, string(raw))
		}
		info, err := os.Stat(path)
		g02Check(t, err)
		if info.Mode().Perm() != 0600 || info.Size() > 2<<20 {
			t.Fatal("bounded private native archive required")
		}
		archive, err := zip.OpenReader(path)
		g02Check(t, err)
		if len(archive.File) != len(expected)+2 {
			archive.Close()
			t.Fatal("native ZIP file inventory changed")
		}
		for _, file := range archive.File {
			if file.Mode().Perm() != 0600 {
				archive.Close()
				t.Fatal("native ZIP entry is not private")
			}
			reader, err := file.Open()
			g02Check(t, err)
			data, err := io.ReadAll(io.LimitReader(reader, 2<<20))
			reader.Close()
			g02Check(t, err)
			if want, ok := expected[file.Name]; ok && !bytes.Equal(data, want) {
				archive.Close()
				t.Fatal("native ZIP did not retain exact encoded bytes", file.Name)
			}
		}
		g02Check(t, archive.Close())
		archiveBytes, err := os.ReadFile(path)
		g02Check(t, err)
		hash := sha256.Sum256(archiveBytes)
		derived := recordingUsageDerivedExports(t, ctx, read, binary, config, proofRoot, in, manifest, expected)
		summaries = append(summaries, map[string]any{"manifest": cliManifest, "parts_compared_both_interfaces": parts, "zip_sha256": hex.EncodeToString(hash[:]), "zip_bytes": len(archiveBytes), "zip_base64": base64.StdEncoding.EncodeToString(archiveBytes), "zip_matches_independent_original_files": true, "derived": derived})
	}
	return summaries
}
