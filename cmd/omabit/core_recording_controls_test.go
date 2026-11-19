package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/recording"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPRecordingControlsAndLostWriteLookup(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("/tmp", "rec-control-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	policy := recording.DefaultRetentionPolicy()
	policy.MaxJobs = 32
	request := recording.ControlRequest{RunnerID: "runner", RequestID: "configure-original", Kind: "configure", Policy: &policy}
	event := recording.ControlEvent{Version: 1, ID: "control-original", RunnerID: "runner", Actor: "owner", Request: request, CreatedAt: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)}
	raw, _ := json.Marshal(struct {
		Domain   string                 `json:"domain"`
		Ordinal  uint64                 `json:"ordinal"`
		Previous string                 `json:"previous_digest"`
		Event    recording.ControlEvent `json:"event"`
	}{"omabit-recording-control-v1", 1, "", event})
	sum := sha256.Sum256(raw)
	receipt := recording.ControlReceipt{Event: event, State: recording.ControlState{Version: 1, RunnerID: "runner", Revision: 1, Digest: hex.EncodeToString(sum[:]), Policy: policy, Pins: []recording.IncidentPin{}, Basis: "inert fixture"}}
	var variant atomic.Value
	variant.Store("ok")
	var calls atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		v := variant.Load().(string)
		if q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Features: []string{protocol.RecordingControlFeature}}
			if v == "observer" {
				h.AccessRole = "observer"
			}
			if v == "missing-capability" {
				h.Features = nil
			}
			json.NewEncoder(w).Encode(h)
			return
		}
		calls.Add(1)
		if v == "transport-lost" {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		r := receipt
		if v == "wrong-reply" {
			r.State.RunnerID = "foreign"
		}
		switch q.URL.Path {
		case protocol.RecordingControlStatusPath:
			json.NewEncoder(w).Encode(recording.ControlStatus{Version: 1, CapacityStatus: "pilot-disabled", State: r.State, ReadAt: event.CreatedAt, Basis: "inert fixture"})
		case protocol.RecordingControlLookupPath:
			json.NewEncoder(w).Encode(recording.ControlLookupResult{Version: 1, RunnerID: "runner", RequestID: request.RequestID, Found: true, Receipt: &r})
		case protocol.RecordingControlApplyPath:
			json.NewEncoder(w).Encode(recording.ControlApplyResult{Receipt: r})
		default:
			t.Error("unexpected path", q.URL.Path)
			w.WriteHeader(404)
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recording-controls-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	target := service.Target{Name: "dev", RunnerID: "runner"}
	for _, op := range []string{"runtime.recording-controls.v1", "runtime.recording-control-lookup.v1", "runtime.recording-control-apply.v1"} {
		for _, name := range []string{"ok", "observer", "missing-capability", "bad-input", "wrong-reply", "transport-lost"} {
			variant.Store(name)
			var in any = service.RecordingControlsInput{Target: target, IncludePrivateData: name != "bad-input"}
			if op == "runtime.recording-control-lookup.v1" {
				in = service.RecordingControlLookupInput{Target: target, RequestID: request.RequestID, IncludePrivateData: name != "bad-input"}
			}
			if op == "runtime.recording-control-apply.v1" {
				r := request
				if name == "bad-input" {
					r.RunnerID = "wrong"
				}
				in = service.RecordingControlApplyInput{Target: target, Request: r}
			}
			raw, _ := json.Marshal(in)
			before := calls.Load()
			cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", op, "--input-json", string(raw)).CombinedOutput()
			tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + op, Arguments: json.RawMessage(raw)})
			if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
				t.Fatal("CLI/MCP controls differ", op, name, err, string(cli), tool)
			}
			if name == "ok" {
				if cliErr != nil || tool.IsError || calls.Load() != before+2 {
					t.Fatal("control failed", op, string(cli))
				}
				continue
			}
			if cliErr == nil || !tool.IsError || bytes.Contains(cli, []byte("control-original")) {
				t.Fatal("refused control leaked result", op, name, string(cli))
			}
			want := int32(2)
			if name == "observer" || name == "missing-capability" || name == "bad-input" {
				want = 0
			}
			if calls.Load() != before+want {
				t.Fatal("refusal dispatched or request retried", op, name, calls.Load()-before)
			}
			if name == "transport-lost" {
				var out struct{ Error *service.Problem }
				if json.Unmarshal(cli, &out) != nil || out.Error == nil {
					t.Fatal(string(cli))
				}
				want := "read-failed"
				if op == "runtime.recording-control-apply.v1" {
					want = "outcome-uncertain"
				}
				if out.Error.Dispatch != want {
					t.Fatal("lost write/read classification changed", op, string(cli))
				}
			}
		}
	}
	variant.Store("ok")
	lookup, _ := json.Marshal(service.RecordingControlLookupInput{Target: target, RequestID: request.RequestID, IncludePrivateData: true})
	before := calls.Load()
	out, err := coreTestCommand(t, "--config-dir", config, "call", "runtime.recording-control-lookup.v1", "--input-json", string(lookup)).CombinedOutput()
	if err != nil || calls.Load() != before+1 || !bytes.Contains(out, []byte("control-original")) {
		t.Fatal("original lost-write receipt cannot be read", err, string(out))
	}
}
