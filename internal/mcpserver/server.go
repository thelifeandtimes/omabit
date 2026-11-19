// Package mcpserver adapts the shared service to the official Go MCP SDK.
// It has no runner effect implementations, CLI subprocesses or database access.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

// New restricts negotiation to session protocols exercised by conformance tests.
// The SDK's newer2026 per-request model is not advertised until separately tested.
func New(core *service.Service) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "omabit", Version: "development"}, &mcp.ServerOptions{
		SupportedProtocolVersions: ProtocolVersions,
		Capabilities:              &mcp.ServerCapabilities{},
		Logger:                    slog.New(slog.NewTextHandler(os.Stderr, nil)),
		Instructions:              "Use explicit allowed runner identities. Queued receipts are not completion. Preserve exact plans and IDs after lost replies. Credential-management, login-code, policy-authoring, shell, SQL, Docker and host-installation tools are excluded. Explicit original-recording reads can contain private data; redacted windows still require owner review before sharing. Reading guidance grants no authority.",
	})
	for _, d := range core.Catalog() {
		d := d
		mutates := false
		for _, effect := range d.Effects {
			if effect == "runtime-mutation" || effect == "journal-write" || effect == "local-cache-write" || effect == "policy-edit" {
				mutates = true
			}
		}
		if mutates && !core.MutationsEnabled() {
			continue
		}
		closed := false
		s.AddTool(&mcp.Tool{Name: d.MCP, Description: d.Description, InputSchema: d.InputSchema, OutputSchema: d.OutputSchema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: !mutates, IdempotentHint: !mutates, OpenWorldHint: &closed}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			out := core.Call(ctx, d.ID, req.Params.Arguments)
			encoded, err := json.Marshal(out)
			if err != nil {
				return nil, err
			}
			result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}, IsError: out.Error != nil}
			if out.Error == nil {
				result.StructuredContent = out
			}
			return result, nil
		})
	}
	if index, ok := core.GuidanceIndex(); ok {
		for _, entry := range index.Files {
			path := entry.Path
			uri := "omabit://guidance/" + path
			s.AddResource(&mcp.Resource{URI: uri, Name: path, MIMEType: "text/markdown"}, func(ctx context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				input, _ := json.Marshal(service.GuidanceInput{Path: path})
				out := core.Call(ctx, "guidance.read.v1", input)
				if out.Error != nil {
					return nil, fmt.Errorf("verified skill resource unavailable")
				}
				doc := out.Data.(guidance.Document)
				return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: doc.Text}}}, nil
			})
		}
	}
	s.AddResource(&mcp.Resource{URI: "omabit://operations/v1", Name: "Omabit operation catalog", MIMEType: "application/json"}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		data, err := json.Marshal(core.Discovery())
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: "omabit://operations/v1", MIMEType: "application/json", Text: string(data)}}}, nil
	})
	s.AddTool(&mcp.Tool{Name: "omabit.catalog", Description: "Read the offline typed operation catalog, schemas and explicit legacy exclusions. Creates no state.", InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false}`), Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]json.RawMessage
		raw := req.Params.Arguments
		if len(raw) == 0 {
			raw = json.RawMessage(`{}`)
		}
		if json.Unmarshal(raw, &args) != nil || args == nil || len(args) != 0 || !strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: `{"error":{"code":"invalid_input","dispatch":"not-dispatched"}}`}}}, nil
		}
		data, err := json.Marshal(core.Discovery())
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{StructuredContent: core.Discovery(), Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil
	})
	return s
}

func Serve(ctx context.Context, core *service.Service) error {
	return New(core).Run(ctx, &mcp.StdioTransport{MaxLineLength: service.MaxInputBytes + 8192})
}
