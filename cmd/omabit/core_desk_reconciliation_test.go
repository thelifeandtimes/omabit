package main

import (
	"bytes"
	"context"
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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPDeskReconciliationAdmissionParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt := deskwork.Receipt{Version: 1, Actor: "original-creator", Intent: deskwork.Intent{Version: 1, ID: strings.Repeat("e", 64), Target: apppolicy.Target{InstanceID: "alpha", Desk: "example"}, RuntimeID: strings.Repeat("b", 64), ConfigRevision: 1, Action: "create-desk"}}
	for _, state := range []string{"planned", "dispatched", "uncertain"} {
		receipt.Steps = append(receipt.Steps, deskwork.Step{Kind: receipt.Intent.Action, State: state, At: time.Now().UTC()})
	}
	if err = receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	var admission atomic.Value
	admission.Store("busy")
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if req.URL.Path == "/hello" && req.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "current-operator", AccessRole: "operator", Features: []string{protocol.DeskCreateFeature}})
			return
		}
		if req.Method != http.MethodPost || req.URL.Path != protocol.DeskOperationsPath+"/"+receipt.Intent.ID+"/reconcile" {
			t.Error("unexpected effect/target endpoint", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(protocol.DeskOperation{Version: 1, RunnerID: "runner", Mode: "simulation", Receipt: receipt, ReconciliationAdmission: admission.Load().(string)})
	})}
	go server.Serve(listener)
	defer server.Close()
	if _, err = registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(service.DeskOperationInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, OperationID: receipt.Intent.ID})
	session, err := mcp.NewClient(&mcp.Implementation{Name: "desk-reconcile-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, state := range []string{"busy", "queued", "unrecognized"} {
		admission.Store(state)
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", "desk.reconcile.v1", "--input-json", string(input)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.desk.reconcile.v1", Arguments: json.RawMessage(input)})
		if err != nil || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal(state, "CLI/MCP admission mismatch", err, string(cli), tool)
		}
		if state == "unrecognized" {
			if cliErr == nil || !tool.IsError {
				t.Fatal("unknown admission was accepted as evidence")
			}
			continue
		}
		var out struct {
			Data protocol.DeskOperation `json:"data"`
		}
		if cliErr != nil || tool.IsError || json.Unmarshal(cli, &out) != nil || out.Data.ReconciliationAdmission != state || !reflect.DeepEqual(out.Data.Receipt, receipt) {
			t.Fatal("admission or original receipt changed", state, cliErr, string(cli))
		}
	}
}
