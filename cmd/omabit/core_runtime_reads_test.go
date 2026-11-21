package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPExactFleetSnapshotAndPrivateLogTail(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(t.TempDir(), "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const revision = uint64(9007199254740993)
	cid := strings.Repeat("a", 64)
	var variant atomic.Value
	variant.Store("ok")
	var reads atomic.Int32
	snapshot := protocol.Snapshot{RunnerID: "runner", Mode: "simulation", Instances: []protocol.Instance{}, Operations: []protocol.Operation{}, Incidents: []protocol.Incident{}}
	tail := protocol.RuntimeLogs{RunnerID: "runner", InstanceID: "alpha", ConfigRevision: revision, ContainerID: cid, Kind: "http-restart", Source: "container", ObservedAt: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC), Lines: []string{"private application diagnostic 0wdiagnostic.atom"}, Truncated: true}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.Method == "GET" && q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{"snapshot", "runtime-logs"}}
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
		if q.Method == "GET" && q.URL.Path == "/v1/snapshot" {
			out := snapshot
			if v == "wrong-runner" {
				out.RunnerID = "replacement"
			}
			if v == "wrong-mode" {
				out.Mode = "development"
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		if q.Method == "POST" && q.URL.Path == "/v1/instances/alpha/logs" {
			if v == "transport-lost" {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				conn.Close()
				return
			}
			var in protocol.LogsRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.ExpectedRevision != revision {
				t.Error("exact log revision changed")
				w.WriteHeader(400)
				return
			}
			out := tail
			if v == "wrong-container" {
				out.ContainerID = strings.Repeat("b", 64)
			}
			if v == "wrong-revision" {
				out.ConfigRevision++
			}
			if v == "wrong-runner" {
				out.RunnerID = "replacement"
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		t.Error("unexpected read or mutation", q.Method, q.URL.Path)
		w.WriteHeader(404)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err = r.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "runtime-reads-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	target := service.Target{Name: "dev", RunnerID: "runner"}
	for _, operation := range []string{"runner.snapshot.v1", "runtime.log-tail.v1"} {
		cases := []string{"ok", "observer", "missing-capability", "wrong-runner", "wrong-mode"}
		if operation == "runtime.log-tail.v1" {
			cases = []string{"ok", "observer", "missing-capability", "wrong-runner", "wrong-container", "wrong-revision", "private-not-selected", "transport-lost"}
		}
		for _, name := range cases {
			variant.Store(name)
			var in any = target
			if operation == "runtime.log-tail.v1" {
				in = service.LogTailInput{Target: target, InstanceID: "alpha", ConfigurationRevision: revision, ContainerID: cid, IncludePrivateData: name != "private-not-selected"}
			}
			input, _ := json.Marshal(in)
			before := reads.Load()
			cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", operation, "--input-json", string(input)).CombinedOutput()
			tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + operation, Arguments: json.RawMessage(input)})
			if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
				t.Fatal(operation, name, "CLI/MCP mismatch", err, string(cli), tool)
			}
			ok := name == "ok" || operation == "runner.snapshot.v1" && name == "observer"
			if ok {
				if cliErr != nil || tool.IsError || reads.Load() != before+2 {
					t.Fatal(operation, name, "bounded read failed or retried", cliErr, string(cli))
				}
				var envelope struct{ Data json.RawMessage }
				if json.Unmarshal(cli, &envelope) != nil {
					t.Fatal("invalid typed envelope")
				}
				var want any = snapshot
				if operation == "runtime.log-tail.v1" {
					want = tail
				}
				encoded, _ := json.Marshal(want)
				if !bytes.Equal(envelope.Data, encoded) {
					t.Fatal("selected response changed", string(cli))
				}
			} else {
				if cliErr == nil || !tool.IsError || bytes.Contains(cli, []byte("private application diagnostic")) {
					t.Fatal(operation, name, "invalid selection or private response accepted", string(cli))
				}
				preflight := name == "observer" || name == "missing-capability" || name == "private-not-selected"
				want := int32(2)
				if preflight {
					want = 0
				}
				if reads.Load() != before+want {
					t.Fatal("refusal submitted an unexpected read/retry", operation, name, reads.Load()-before)
				}
				if name == "transport-lost" {
					var envelope struct{ Error *service.Problem }
					if json.Unmarshal(cli, &envelope) != nil || envelope.Error == nil || envelope.Error.Dispatch != "read-failed" || envelope.Error.Code != "operation_failed" {
						t.Fatal("lost read became an uncertain mutation", string(cli))
					}
				}
			}
		}
	}
}
