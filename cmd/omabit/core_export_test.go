package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPExportExactPrivateSourceBytes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	hello := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "fixture", AccessRole: "operator", Features: []string{protocol.DeskSourceExportFeature}}
	observation := protocol.DeskObservation{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Selection: protocol.DeskSelection{Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1}, Observation: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, ReadAt: time.Now().UTC()}
	request := protocol.DeskSourceExportRequest{Selection: observation.Selection, Baseline: observation.Observation, Paths: []string{"binary.bin"}}
	raw := []byte{0, 255, 27, 10}
	digest := sha256.Sum256(raw)
	response := protocol.DeskSourceExport{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Request: request, Files: []deskwork.ExportedFile{{Path: request.Paths[0], ContentBase64: base64.StdEncoding.EncodeToString(raw), SHA256: hex.EncodeToString(digest[:])}}, ReadAt: observation.ReadAt}
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if q.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(hello)
			return
		}
		var got protocol.DeskSourceExportRequest
		if q.URL.Path != protocol.DeskSourceExportPath || q.Method != "POST" || json.NewDecoder(q.Body).Decode(&got) != nil || !reflect.DeepEqual(got, request) {
			http.Error(w, "unexpected source read", 400)
			return
		}
		json.NewEncoder(w).Encode(response)
	})}
	go server.Serve(listener)
	defer server.Close()
	if _, err = registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "source-export-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	input, err := json.Marshal(service.DeskExportInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, Observation: observation, Paths: request.Paths})
	if err != nil {
		t.Fatal(err)
	}
	cli, err := coreTestCommand(t, "--config-dir", config, "call", "desk.export-source.v1", "--input-json", string(input)).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(cli))
	}
	tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.desk.export-source.v1", Arguments: json.RawMessage(input)})
	if err != nil || tool.IsError || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
		t.Fatal("source export parity failed", tool, err)
	}
	var envelope struct {
		Data protocol.DeskSourceExport `json:"data"`
	}
	if json.Unmarshal(cli, &envelope) != nil || envelope.Data.Validate() != nil {
		t.Fatal("CLI/MCP changed original source bytes")
	}
	decoded, e := envelope.Data.Files[0].Decode()
	if e != nil || !bytes.Equal(decoded, raw) {
		t.Fatal("binary source changed")
	}
}
