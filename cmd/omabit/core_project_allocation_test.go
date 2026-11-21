package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectusage"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPProjectAllocationPrivateBindingsAndUnknowns(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(os.TempDir(), "pa-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const revision = uint64(9007199254740993)
	r := projectusage.Root{ID: "mount-0", Kind: "member-mount", InstanceID: "alpha", RuntimeID: strings.Repeat("b", 32), Path: "/private/fixture/mount", Expected: pierstore.DirectoryIdentity{Device: 1, Inode: 2, Birth: "1:1"}}
	totals := projectusage.Totals{Directories: 1, RegularFiles: 1, UniqueFileInodes: 1, LogicalBytes: revision, AllocatedBytes: 4096}
	base := projectusage.Observation{Version: 1, Binding: projectusage.Binding{RunnerID: "runner", Mode: "simulation", ProjectID: "project", Revision: revision, Generation: 1, PlanDigest: strings.Repeat("c", 64), LedgerKind: projectusage.ExpandedLedger, LedgerSHA256: strings.Repeat("d", 64)}, Scope: projectusage.Scope, Status: "measured", DeclaredRoots: []projectusage.Root{r}, Roots: []projectusage.RootUsage{{Root: r, Observed: r.Expected, Totals: totals}}, Totals: &totals, ExcludedComponents: projectusage.Exclusions(), SharedAllowanceStatus: "not-configured", StartedAt: time.Now().UTC(), ReadAt: time.Now().UTC(), Basis: "inert CLI/SDK source interface fixture; no native filesystem scan"}
	g := projectusage.Compare(projectusage.Observation{}, base, 0)
	base.Growth = &g
	var variant atomic.Value
	variant.Store("ok")
	var reads atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.Method == "GET" && q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.ProjectAllocationFeature}}
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
		if q.Method != "POST" || q.URL.Path != protocol.ProjectAllocationPath {
			t.Error("unexpected operation", q.Method, q.URL.Path)
			w.WriteHeader(404)
			return
		}
		var in projectusage.Request
		if json.NewDecoder(q.Body).Decode(&in) != nil || in.Validate() != nil || in.ExpectedRevision != revision || in.RunnerID != "runner" || in.ProjectID != "project" || in.ReviewDigest != base.Binding.PlanDigest {
			t.Error("exact input changed")
			w.WriteHeader(400)
			return
		}
		if v == "transport-lost" {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		out := base.Clone()
		switch v {
		case "original-history":
			out.Binding.LedgerKind = projectusage.OriginalHistory
		case "unknown":
			out.Status, out.Reason, out.UnknownRootID, out.Totals, out.Roots = "unknown", "changed", r.ID, nil, nil
			growth := projectusage.Compare(projectusage.Observation{}, out, 0)
			out.Growth = &growth
		case "wrong-reply":
			out.Binding.LedgerSHA256 = "bad"
		case "wrong-mode":
			out.Binding.Mode = "development-managed"
		case "partial":
			out.Status, out.Reason = "unknown", "changed"
		}
		json.NewEncoder(w).Encode(out)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "project-allocation-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	const operation = "project.allocation.v1"
	for _, name := range []string{"ok", "original-history", "unknown", "observer", "missing-capability", "no-consent", "wrong-reply", "wrong-mode", "partial", "transport-lost"} {
		variant.Store(name)
		in := service.ProjectAllocationInput{ProjectInput: service.ProjectInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, ProjectID: "project"}, ExpectedRevision: revision, ReviewDigest: base.Binding.PlanDigest, IncludePrivateData: name != "no-consent"}
		raw, _ := json.Marshal(in)
		before := reads.Load()
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", operation, "--input-json", string(raw)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + operation, Arguments: json.RawMessage(raw)})
		if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal("CLI/MCP allocation mismatch", name, err, string(cli), tool)
		}
		if name == "ok" || name == "original-history" || name == "unknown" {
			var envelope struct {
				Data projectusage.Observation `json:"data"`
			}
			if cliErr != nil || tool.IsError || reads.Load() != before+2 || json.Unmarshal(cli, &envelope) != nil || envelope.Data.Validate() != nil || !bytes.Contains(cli, []byte("9007199254740993")) {
				t.Fatal("exact observation lost", name, cliErr, string(cli))
			}
			if name == "unknown" && (envelope.Data.Totals != nil || len(envelope.Data.Roots) != 0 || envelope.Data.Growth.Status != "unknown") {
				t.Fatal("unknown leaked partial numbers")
			}
		} else {
			expected := int32(2)
			if name == "observer" || name == "missing-capability" || name == "no-consent" {
				expected = 0
			}
			if cliErr == nil || !tool.IsError || reads.Load() != before+expected || bytes.Contains(cli, []byte("/private/fixture")) {
				t.Fatal("refusal leaked/retried", name, cliErr, string(cli), reads.Load()-before)
			}
			if name == "transport-lost" {
				var out struct{ Error *service.Problem }
				if json.Unmarshal(cli, &out) != nil || out.Error == nil || out.Error.Dispatch != "read-failed" {
					t.Fatal("read treated as mutation", string(cli))
				}
			}
		}
	}
}
