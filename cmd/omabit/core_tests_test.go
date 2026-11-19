//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNamedPlansAndSelectedReportsActualCLIAndStdioParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	dir, dirErr := filepath.EvalSymlinks(t.TempDir())
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	config := filepath.Join(dir, "uncreated-registry")
	file := filepath.Join(dir, "report.json")
	raw := []byte(`{"profile":"isolated-lifecycle","run_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","checkpoint_passed":true,"runtime_qualified":false,"cleanup_complete":true,"phases":[{"name":"private-phase-name","passed":true,"detail":"private-secret-value"}],"containers":[],"directory":"/private/path"}`)
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	flag := "saved:" + digest + ":" + file
	command := coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-test-report", flag)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "named-checks-conformance", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, tc := range []struct {
		id, input string
		failure   bool
	}{
		{"tests.plans.v1", `{}`, false},
		{"tests.plan.v1", `{"id":"runtime-http-v1"}`, false},
		{"tests.plan.v1", `{"id":"run_shell"}`, true},
		{"tests.reports.v1", `{"include_private_data":true}`, false},
		{"tests.reports.v1", `{}`, true},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true}`, false},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-http-v1","sha256":"` + digest + `","include_private_data":true}`, true},
		{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true,"path":"/unselected"}`, true},
	} {
		cli := coreTestCommand(t, "--config-dir", config, "call", tc.id, "--allow-test-report", flag, "--input-json", tc.input)
		out, cliErr := cli.Output()
		if (cliErr != nil) != tc.failure {
			t.Fatal("CLI status", tc.id, cliErr)
		}
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: json.RawMessage(tc.input)})
		if err != nil || tool.IsError != tc.failure || len(tool.Content) != 1 {
			t.Fatal("SDK status", tc.id, err)
		}
		text, ok := tool.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("missing JSON text fallback")
		}
		decode := func(raw []byte) map[string]any {
			t.Helper()
			var value map[string]any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if decoder.Decode(&value) != nil {
				t.Fatal("invalid JSON")
			}
			return value
		}
		a, b := decode(out), decode([]byte(text.Text))
		if tc.id == "tests.report.v1" && !tc.failure {
			for _, value := range []map[string]any{a, b} {
				data := value["data"].(map[string]any)
				stamp, ok := data["read_at"].(string)
				if !ok {
					t.Fatal("missing snapshot time")
				}
				if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil {
					t.Fatal(err)
				}
				delete(data, "read_at")
			}
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatal("actual CLI/SDK parity differs", tc.id, a, b)
		}
		for _, private := range []string{"private-secret-value", "private-phase-name", "/private/path"} {
			if strings.Contains(string(out), private) || strings.Contains(text.Text, private) {
				t.Fatal("private raw field escaped summary")
			}
		}
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	// MCP reserves stdout for protocol traffic; diagnostics belong on stderr.
	// SDK session lifecycle messages are expected, but selected report content
	// and its private path must never be reflected into those diagnostics.
	for _, private := range []string{"private-secret-value", "private-phase-name", "/private/path", file} {
		if strings.Contains(stderr.String(), private) {
			t.Fatal("private report data escaped diagnostics")
		}
	}
	if _, err := os.Lstat(config); !os.IsNotExist(err) {
		t.Fatal("offline interfaces created registry")
	}
	for _, args := range [][]string{{"call", "tests.plan.v1", "--allow-test-report", "not-a-selection"}, {"mcp", "serve", "--allow-test-report", "not-a-selection"}} {
		if err := coreTestCommand(t, args...).Run(); err == nil {
			t.Fatal("invalid startup report flag accepted")
		}
	}
	help := coreTestCommand(t, "mcp", "serve", "--allow-test-report", "saved:"+strings.Repeat("a", 64)+":/missing/report.json", "--help")
	if err := help.Run(); err != nil {
		t.Fatal("help read a selected report", err)
	}
}
