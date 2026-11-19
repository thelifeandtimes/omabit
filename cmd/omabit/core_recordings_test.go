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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPPrivateRecordingHistoryAndExactPage(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	socketDir, err := os.MkdirTemp(os.TempDir(), "rec-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(socketDir) })
	socket := filepath.Join(socketDir, "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const revision = uint64(9007199254740993)
	descriptor := recording.SessionDescriptor{RunnerID: "runner", Mode: "simulation", InstanceID: "alpha", RuntimeID: strings.Repeat("b", 32), ConfigurationRevision: revision, ContainerID: strings.Repeat("c", 64), JobID: strings.Repeat("d", 32), JobSHA256: strings.Repeat("e", 64), OperationID: "op-original", RequestID: "start-original", CreatedAt: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)}
	var variant atomic.Value
	variant.Store("ok")
	var reads atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.Method == "GET" && q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.RecordingReadFeature}}
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
		if v == "transport-lost" {
			conn, _, e := w.(http.Hijacker).Hijack()
			if e != nil {
				t.Error(e)
				return
			}
			conn.Close()
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingCatalogPath {
			var in recording.CatalogRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.RunnerID != "runner" || !in.IncludePrivateData {
				t.Error("catalog request changed")
				w.WriteHeader(400)
				return
			}
			d := descriptor
			if v == "wrong-reply" {
				d.RunnerID = "replacement"
			}
			json.NewEncoder(w).Encode(recording.Catalog{Version: 1, RunnerID: "runner", InstanceID: "alpha", Sessions: []recording.SessionDescriptor{d}, ReadAt: descriptor.CreatedAt, Basis: "saved-only inert fixture"})
			return
		}
		if q.Method == "POST" && q.URL.Path == protocol.RecordingPagePath {
			var in recording.PageRequest
			if json.NewDecoder(q.Body).Decode(&in) != nil || in.JobID != descriptor.JobID || in.JobSHA256 != descriptor.JobSHA256 || !in.IncludePrivateData {
				t.Error("page selection changed")
				w.WriteHeader(400)
				return
			}
			d := descriptor
			if v == "wrong-reply" {
				d.JobSHA256 = strings.Repeat("f", 64)
			}
			json.NewEncoder(w).Encode(recording.ReadWindow{Version: 1, Session: d, Status: recording.SavedStatus{State: "job-saved", SessionIDBasis: "unknown", SessionStorage: "unknown"}, Representation: in.Representation, From: in.Cursor, Tail: "not-scanned", ReadAt: descriptor.CreatedAt, Basis: "saved fixture"})
			return
		}
		t.Error("unexpected effect", q.Method, q.URL.Path)
		w.WriteHeader(404)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recordings-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	target := service.Target{Name: "dev", RunnerID: "runner"}
	for _, operation := range []string{"runtime.recordings.v1", "runtime.recording-page.v1"} {
		for _, name := range []string{"ok", "observer", "missing-capability", "no-consent", "wrong-reply", "transport-lost"} {
			variant.Store(name)
			var in any = service.RecordingCatalogInput{Target: target, InstanceID: "alpha", IncludePrivateData: name != "no-consent"}
			if operation == "runtime.recording-page.v1" {
				in = service.RecordingPageInput{Target: target, InstanceID: "alpha", JobID: descriptor.JobID, JobSHA256: descriptor.JobSHA256, Representation: "original", IncludePrivateData: name != "no-consent"}
			}
			raw, _ := json.Marshal(in)
			before := reads.Load()
			cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", operation, "--input-json", string(raw)).CombinedOutput()
			tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + operation, Arguments: json.RawMessage(raw)})
			if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
				t.Fatal("CLI/MCP recording mismatch", operation, name, err, string(cli), tool)
			}
			if name == "ok" {
				if cliErr != nil || tool.IsError || reads.Load() != before+2 || !bytes.Contains(cli, []byte("9007199254740993")) {
					t.Fatal("bounded exact read failed", operation, name, cliErr, string(cli))
				}
			} else {
				if cliErr == nil || !tool.IsError || bytes.Contains(cli, []byte("op-original")) {
					t.Fatal("bad selection exposed private history", operation, name, string(cli))
				}
				expected := int32(2)
				if name == "observer" || name == "missing-capability" || name == "no-consent" {
					expected = 0
				}
				if reads.Load() != before+expected {
					t.Fatal("read was retried or refusal dispatched", operation, name, reads.Load()-before)
				}
				if name == "transport-lost" {
					var envelope struct{ Error *service.Problem }
					if json.Unmarshal(cli, &envelope) != nil || envelope.Error == nil || envelope.Error.Dispatch != "read-failed" {
						t.Fatal("read became uncertain mutation", string(cli))
					}
				}
			}
		}
	}
}
