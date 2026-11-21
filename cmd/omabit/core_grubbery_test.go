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

func TestCLIAndMCPExactAppSourceOperationParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt := coreAppReceipt(t)
	p := receipt.Record.Plan
	q := protocol.GrubberyAppReviewRequest{Selection: protocol.DeskSelection{Target: p.Binding.Target, RuntimeID: p.Binding.RuntimeID, ConfigRevision: p.Binding.ConfigRevision}, FrameworkReceiptID: p.Binding.FrameworkReceiptID, AppID: p.Binding.AppID, StageID: p.StageID, RequestID: p.RequestID}
	review := protocol.GrubberyAppReview{Version: 1, RunnerID: receipt.RunnerID, Mode: receipt.Mode, Request: q, Plan: p, ReadAt: time.Now().UTC()}
	socket := filepath.Join(t.TempDir(), "runner.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var posts atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if req.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Actor: p.Binding.Actor, Features: []string{protocol.GrubberyAppSourceFeature}})
			return
		}
		if req.Method != "POST" || !protocol.GrubberyAppSourceEndpoint(req.Method, req.URL.Path) {
			t.Error("unexpected native/generic request", req.Method, req.URL.Path)
			w.WriteHeader(400)
			return
		}
		posts.Add(1)
		if req.URL.Path == protocol.GrubberyAppReviewPath {
			var got protocol.GrubberyAppReviewRequest
			if json.NewDecoder(req.Body).Decode(&got) != nil || got != q {
				t.Error("review selection changed")
			}
			json.NewEncoder(w).Encode(review)
			return
		}
		if req.URL.Path == protocol.GrubberyAppOperationsPath {
			var got protocol.GrubberyAppBeginRequest
			if json.NewDecoder(req.Body).Decode(&got) != nil || !reflect.DeepEqual(got.Plan, p) {
				t.Error("begin plan changed")
			}
		} else {
			id, _, ok := protocol.GrubberyAppPhasePath(req.URL.Path)
			var got protocol.GrubberyAppPhaseRequest
			if !ok || id != p.RequestID || json.NewDecoder(req.Body).Decode(&got) != nil || got.PlanDigest != p.Digest || got.StepID != p.Steps[0].ID {
				t.Error("exact original phase changed")
			}
		}
		json.NewEncoder(w).Encode(receipt)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if _, err = r.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "app-source-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	target := service.Target{Name: "dev", RunnerID: "runner"}
	for _, tc := range []struct {
		id    string
		input any
	}{
		{"app.review-source.v1", service.AppSourceReviewInput{Target: target, Request: q}},
		{"app.begin-source.v1", service.AppSourceBeginInput{Target: target, Request: protocol.GrubberyAppBeginRequest{Plan: p}}},
		{"app.advance-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: p.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}}},
		{"app.reconcile-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: p.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}}},
		{"app.abandon-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: p.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}}},
	} {
		input, _ := json.Marshal(tc.input)
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", tc.id, "--input-json", string(input)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: json.RawMessage(input)})
		if cliErr != nil || err != nil || tool.IsError || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal(tc.id, "CLI/MCP parity", cliErr, err, string(cli), tool)
		}
	}
	if posts.Load() != 10 {
		t.Fatal("unexpected retries", posts.Load())
	}
	readonly, err := service.New(service.Options{ConfigDir: config, RequireAllowed: true, Allowed: []service.Target{target}})
	if err != nil {
		t.Fatal(err)
	}
	in, _ := json.Marshal(service.AppSourcePhaseInput{Target: target, OperationID: p.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: p.Steps[0].ID}})
	for _, action := range []string{"advance", "reconcile", "abandon"} {
		out := readonly.Call(ctx, "app."+action+"-source.v1", in)
		if out.Error == nil || out.Error.Code != "mutation_disabled" || posts.Load() != 10 {
			t.Fatal("journal/runtime mutation bypassed startup policy", action, out)
		}
	}
}
