package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPPinnedFrameworkSourceParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	directory := os.Getenv("OMABIT_TEST_GRUBBERY_SOURCE")
	wantCode := ""
	if directory == "" {
		directory = t.TempDir()
		wantCode = "framework_source_mismatch"
		if err := os.WriteFile(filepath.Join(directory, "sys.kelvin"), []byte("[%zuse 410]\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(t.TempDir(), "must-remain-absent")
	command := coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-source", "framework:"+directory)
	session, err := mcp.NewClient(&mcp.Implementation{Name: "framework-source-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, tc := range []struct{ id, input, code string }{
		{"app.framework.v1", `{"name":"grubbery-7f7701d-v1"}`, ""},
		{"app.verify-framework-source.v1", `{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"."}}`, wantCode},
		{"app.verify-framework-source.v1", `{"name":"grubbery-7f7701d-v1","source":{"root":"missing","directory":"."}}`, "source_not_allowed"},
		{"app.verify-framework-source.v1", `{"name":"grubbery-7f7701d-v1","source":{"root":"framework","directory":"../private"}}`, "invalid_input"},
	} {
		t.Run(tc.id+tc.code, func(t *testing.T) {
			cmd := coreTestCommand(t, "--config-dir", config, "call", tc.id, "--allow-source", "framework:"+directory, "--input-json", tc.input)
			out, err := cmd.Output()
			if (err != nil) != (tc.code != "") {
				t.Fatal("CLI dispatch failed", err)
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: json.RawMessage(tc.input)})
			if err != nil || result.IsError != (tc.code != "") || len(result.Content) != 1 {
				t.Fatal("MCP dispatch failed", result, err)
			}
			if !bytes.Equal(bytes.TrimSpace(out), []byte(result.Content[0].(*mcp.TextContent).Text)) {
				t.Fatal("CLI and stdio MCP framework results differ")
			}
			var envelope service.Result
			if json.Unmarshal(out, &envelope) != nil || tc.code != "" && (envelope.Error == nil || envelope.Error.Code != tc.code) {
				t.Fatal("unexpected error contract", envelope)
			}
		})
	}
	if _, err := os.Stat(config); !os.IsNotExist(err) {
		t.Fatal("offline framework checks created registry", err)
	}
}
