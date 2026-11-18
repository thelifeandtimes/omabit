package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func coreAppReceipt(t *testing.T) protocol.GrubberyAppOperation {
	t.Helper()
	files := []deskwork.SourceFile{{Path: "bill.json", Bytes: []byte(`{"state":"/state"}`)}, {Path: "nex/state.hoon", Bytes: []byte(":: inert fixture only\n")}}
	entries := []deskwork.File{}
	for _, f := range files {
		d := sha256.Sum256(f.Bytes)
		entries = append(entries, deskwork.File{Path: f.Path, Bytes: int64(len(f.Bytes)), SHA256: hex.EncodeToString(d[:])})
	}
	manifest, err := deskwork.NewManifest(entries)
	if err != nil {
		t.Fatal(err)
	}
	b := grubbery.Binding{RunnerID: "runner", Mode: "simulation", Actor: "original-creator", Target: apppolicy.Target{InstanceID: "alpha", Desk: "grubbery"}, RuntimeID: strings.Repeat("b", 64), Continuity: strings.Repeat("c", 64), FrameworkReceiptID: strings.Repeat("d", 64), ConfigRevision: 1, FrameworkBaseline: deskwork.Observation{Exists: true, Revision: 2, ClayHash: "0v123", Publication: "private"}, AppID: "state"}
	p, err := grubbery.NewPlan(b, strings.Repeat("e", 64), strings.Repeat("1", 64), manifest, files, grubbery.Before{SnapshotDigest: strings.Repeat("f", 64), SourceState: "absent", DeskState: "absent"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := grubbery.NewRecord(p, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return protocol.GrubberyAppOperation{Version: 1, RunnerID: "runner", Mode: "simulation", Record: r}
}

func TestCLIAndMCPReadOriginalAppReceiptWithoutMutationPermission(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt := coreAppReceipt(t)
	var observer, old, wrong, corrupt atomic.Bool
	var reads atomic.Int32
	socket := filepath.Join(t.TempDir(), "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if q.Method != "GET" {
			t.Error("receipt lookup attempted mutation")
			w.WriteHeader(405)
			return
		}
		if q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Actor: "current-reader", Features: []string{protocol.GrubberyAppReceiptsFeature}}
			if observer.Load() {
				h.AccessRole = "observer"
			}
			if old.Load() {
				h.Features = nil
			}
			json.NewEncoder(w).Encode(h)
			return
		}
		if q.URL.Path != protocol.GrubberyAppOperationsPath+"/"+receipt.Record.Plan.RequestID {
			t.Error("unexpected live probe", q.URL.Path)
			w.WriteHeader(404)
			return
		}
		reads.Add(1)
		out := receipt
		if wrong.Load() {
			out.RunnerID = "another"
		}
		if corrupt.Load() {
			out.Record.HeaderDigest = strings.Repeat("0", 64)
		}
		json.NewEncoder(w).Encode(out)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err = r.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(service.DeskOperationInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, OperationID: receipt.Record.Plan.RequestID})
	session, err := mcp.NewClient(&mcp.Implementation{Name: "app-receipt-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, name := range []string{"ok", "observer", "old", "wrong", "corrupt"} {
		observer.Store(name == "observer")
		old.Store(name == "old")
		wrong.Store(name == "wrong")
		corrupt.Store(name == "corrupt")
		before := reads.Load()
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", "app.source-operation.v1", "--input-json", string(input)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.app.source-operation.v1", Arguments: json.RawMessage(input)})
		if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal(name, "CLI/MCP mismatch", err, string(cli), tool)
		}
		if name == "ok" {
			var out struct {
				Data protocol.GrubberyAppOperation `json:"data"`
			}
			if cliErr != nil || tool.IsError || json.Unmarshal(cli, &out) != nil || !reflect.DeepEqual(out.Data, receipt) {
				t.Fatal("original receipt changed", string(cli), cliErr)
			}
		} else if cliErr == nil || !tool.IsError {
			t.Fatal(name, "accepted invalid authority/receipt")
		}
		if (name == "observer" || name == "old") && before != reads.Load() {
			t.Fatal("refusal accessed receipt")
		}
	}
	// Startup allowlists are enforced independently of selected enrollment.
	denied, err := service.New(service.Options{ConfigDir: config, RequireAllowed: true})
	if err != nil {
		t.Fatal(err)
	}
	result := denied.Call(ctx, "app.source-operation.v1", input)
	if result.Error == nil || result.Error.Code != "target_not_allowed" {
		t.Fatal("empty allowlist admitted runner", result)
	}
}
