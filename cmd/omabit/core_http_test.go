//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type coreHTTPTransport func(*http.Request) (*http.Response, error)

func (f coreHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCLIAuthenticatedHTTPStartupDiscoveryAndCleanShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "local-http.json")
	public, err := coreTestCommand(t, "mcp", "create-http-token", "--output", path).CombinedOutput()
	if err != nil {
		t.Fatal("credential CLI", err, string(public))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var credential struct{ Token string }
	if json.Unmarshal(raw, &credential) != nil || len(credential.Token) != 64 || bytes.Contains(public, []byte(credential.Token)) {
		t.Fatal("credential output exposed secret or file invalid")
	}
	cmd := coreTestCommand(t, "--config-dir", filepath.Join(t.TempDir(), "uncreated-registry"), "mcp", "serve", "--transport", "http", "--listen", "127.0.0.1:0", "--token-file", path)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	stopped := false
	defer func() {
		if !stopped {
			cmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				cmd.Process.Kill()
				<-done
			}
		}
	}()
	type ready struct {
		Transport string `json:"transport"`
		URL       string `json:"url"`
	}
	type result struct {
		value ready
		err   error
	}
	metadata := make(chan result, 1)
	go func() { var info ready; err := json.NewDecoder(stderr).Decode(&info); metadata <- result{info, err} }()
	var endpoint string
	select {
	case info := <-metadata:
		if info.err != nil || info.value.Transport != "http" {
			t.Fatal("local endpoint readiness", info.err)
		}
		endpoint = info.value.URL
	case <-ctx.Done():
		t.Fatal("no bounded HTTP readiness")
	}
	client := &http.Client{Transport: coreHTTPTransport(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.Header = r.Header.Clone()
		copy.Header.Set("Authorization", "Bearer "+credential.Token)
		return http.DefaultTransport.RoundTrip(copy)
	})}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "actual-cli-http", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: client}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal("actual CLI HTTP initialize", err)
	}
	tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.catalog", Arguments: map[string]any{}})
	if err != nil || tool.IsError || len(tool.Content) != 1 {
		t.Fatal("actual CLI HTTP catalog", err)
	}
	denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.runner.hello.v1", Arguments: map[string]any{"name": "not-allowed", "runner_id": "runner"}})
	if err != nil || !denied.IsError {
		t.Fatal("actual CLI HTTP empty allowlist bypass", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		stopped = true
		if err != nil {
			t.Fatal("HTTP did not close cleanly", err)
		}
	case <-ctx.Done():
		t.Fatal("HTTP shutdown deadline")
	}
	if stdout.Len() != 0 {
		t.Fatal("HTTP process wrote to protocol stdout")
	}
}
