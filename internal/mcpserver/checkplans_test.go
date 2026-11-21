//go:build linux || darwin

package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/checkplan"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNamedPlansAndSelectedReportAcrossQualifiedProtocols(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"profile":"isolated-lifecycle","run_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","checkpoint_passed":false,"runtime_qualified":false,"cleanup_complete":false,"phases":[],"containers":[],"failure":"private original diagnostic"}`)
	file := filepath.Join(root, "report.json")
	if err = os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	report, err := checkplan.OpenReport("saved", file, digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range ProtocolVersions {
		t.Run(version, func(t *testing.T) {
			core, err := service.New(service.Options{RequireAllowed: true, TestReports: []*checkplan.Report{report}})
			if err != nil {
				t.Fatal(err)
			}
			server := New(core)
			st, ct := mcp.NewInMemoryTransports()
			ctx := context.Background()
			ss, err := server.Connect(ctx, st, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			cs, err := mcp.NewClient(&mcp.Implementation{Name: "named-checks", Version: "1"}, nil).Connect(ctx, ct, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			if cs.InitializeResult().ProtocolVersion != version {
				t.Fatal("protocol changed")
			}
			for _, tc := range []struct{ id, input string }{
				{"tests.plans.v1", `{}`},
				{"tests.plan.v1", `{"id":"runtime-http-v1"}`},
				{"tests.reports.v1", `{"include_private_data":true}`},
				{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true}`},
				{"tests.report.v1", `{"name":"saved","plan_id":"runtime-lifecycle-v1","sha256":"` + digest + `","include_private_data":true,"path":"/unselected"}`},
			} {
				input := json.RawMessage(tc.input)
				want := core.Call(ctx, tc.id, input)
				encoded, _ := json.Marshal(want)
				tool, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: input})
				if err != nil || tool.IsError != (want.Error != nil) || len(tool.Content) != 1 {
					t.Fatal("SDK result differs", tc.id, err)
				}
				text, ok := tool.Content[0].(*mcp.TextContent)
				if !ok || text.Text != string(encoded) {
					t.Fatal("JSON parity differs", tc.id)
				}
			}
		})
	}
}
