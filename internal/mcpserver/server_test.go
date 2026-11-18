package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/guidance"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProtocolVersionsCatalogSchemasAndCoreParity(t *testing.T) {
	for _, version := range ProtocolVersions {
		t.Run(version, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "not-created")
			core, err := service.New(service.Options{ConfigDir: dir, RequireAllowed: true})
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
			cs, err := mcp.NewClient(&mcp.Implementation{Name: "conformance", Version: "1"}, nil).Connect(ctx, ct, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			if cs.InitializeResult().ProtocolVersion != version {
				t.Fatalf("negotiated %+v", cs.InitializeResult())
			}
			var names []string
			for tool, err := range cs.Tools(ctx, nil) {
				if err != nil {
					t.Fatal(err)
				}
				names = append(names, tool.Name)
				if tool.InputSchema == nil {
					t.Fatal("schema missing")
				}
			}
			if slices.Contains(names, "omabit.project.stop-reconcile.v1") || !slices.Contains(names, "omabit.project.status.v1") {
				t.Fatal("read-only tool filter incorrect")
			}
			input := json.RawMessage(`{"name":"dev","runner_id":"runner","project_id":"project"}`)
			got, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.project.status.v1", Arguments: input})
			if err != nil {
				t.Fatal(err)
			}
			want := core.Call(ctx, "project.status.v1", input)
			encoded, _ := json.Marshal(want)
			if !got.IsError || got.Content[0].(*mcp.TextContent).Text != string(encoded) {
				t.Fatalf("core/MCP errors diverged: %+v", got)
			}
			read, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "omabit://operations/v1"})
			if err != nil || len(read.Contents) != 1 || !json.Valid([]byte(read.Contents[0].Text)) {
				t.Fatalf("catalog resource: %+v %v", read, err)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("offline MCP created state")
			}
		})
	}
}

func TestVerifiedSkillResourceAndOrdinaryToolReadSameBytes(t *testing.T) {
	root := t.TempDir()
	hash := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	names := []string{"omabit-operator", "omabit-plugin-developer"}
	files := map[string]string{}
	for _, name := range names {
		path := "skills/" + name + "/SKILL.md"
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		data := []byte("# " + name + "\nRead-only fixture.\n")
		if err := os.WriteFile(full, data, 0600); err != nil {
			t.Fatal(err)
		}
		files[path] = hash(data)
	}
	build := map[string]any{"version": "fixture-dev", "source_commit": strings.Repeat("a", 40)}
	buildBytes, _ := json.Marshal(build)
	if err := os.WriteFile(filepath.Join(root, "BUILD.json"), buildBytes, 0600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{"format": 1, "skills": names, "files": files, "version": build["version"], "source_commit": build["source_commit"]}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(root, "SKILLS.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	bundle, err := guidance.Open(root, hash(data))
	if err != nil {
		t.Fatal(err)
	}
	core, err := service.New(service.Options{Guidance: bundle, RequireAllowed: true})
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
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "resource-fixture", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if cs.InitializeResult().ProtocolVersion != "2025-11-25" {
		t.Fatal("default SDK did not negotiate qualified legacy version")
	}
	path := "skills/omabit-operator/SKILL.md"
	resource, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "omabit://guidance/" + path})
	if err != nil {
		t.Fatal(err)
	}
	tool, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.guidance.read.v1", Arguments: service.GuidanceInput{Path: path}})
	if err != nil || tool.IsError {
		t.Fatal(tool, err)
	}
	var output struct {
		Data guidance.Document `json:"data"`
	}
	if err = json.Unmarshal([]byte(tool.Content[0].(*mcp.TextContent).Text), &output); err != nil {
		t.Fatal(err)
	}
	if resource.Contents[0].Text != output.Data.Text || hash([]byte(output.Data.Text)) != files[path] || output.Data.ManifestSHA256 != hash(data) {
		t.Fatal("resource/tool bytes or provenance diverged")
	}
	if err = os.WriteFile(filepath.Join(root, path), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "omabit://guidance/" + path}); err == nil {
		t.Fatal("changed resource served")
	}
	tool, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.guidance.read.v1", Arguments: service.GuidanceInput{Path: path}})
	if err != nil || !tool.IsError {
		t.Fatal("changed ordinary read accepted", err)
	}
}
