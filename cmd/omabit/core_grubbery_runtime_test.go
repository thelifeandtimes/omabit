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
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIAndMCPExactAppRuntimeRecoveryParityAndMutationBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	original := coreAppReceipt(t).Record
	b := original.Plan.Binding
	at := time.Now().UTC()
	facts := grubbery.RuntimeRecoveryReview{Version: 1, OriginalRequestID: original.Plan.RequestID, PlanDigest: original.Plan.Digest, HeaderDigest: original.HeaderDigest, OriginalEventDigest: original.HeaderDigest, StepID: original.Plan.Steps[0].ID, State: "pending", OriginalBinding: b, PreviousContainerID: b.RuntimeID, Process: grubbery.RuntimeProcess{ContainerID: strings.Repeat("9", 64), PID: 4242, StartedAt: at.Add(-time.Minute), HostBootID: "11111111-2222-3333-4444-555555555555", EngineID: "fixture-runtime-daemon", ImageManifestDigest: "sha256:" + strings.Repeat("8", 64), RuntimeVersion: "4.6-5a7614f", ContextSHA256: strings.Repeat("7", 64)}, EvidenceBasis: "verified-pending-prefix", NativeEvidenceDigest: strings.Repeat("6", 64), ObservedAt: at}
	facts.Digest = facts.ContentDigest()
	review := protocol.GrubberyRuntimeReview{Version: 1, RunnerID: b.RunnerID, Mode: b.Mode, Original: original, Review: facts}
	if err = review.Validate(); err != nil {
		t.Fatal(err)
	}
	audit := grubbery.RuntimeRecoveryReceipt{Version: 1, RequestID: strings.Repeat("5", 64), Review: facts, AppliedAt: at.Add(time.Second)}
	audit.Digest = audit.ContentDigest()
	status := protocol.GrubberyRuntimeRecoveryStatus{Version: 1, RunnerID: b.RunnerID, Mode: b.Mode, Original: original, Receipt: &audit}
	if err = status.Validate(); err != nil {
		t.Fatal(err)
	}
	q := protocol.GrubberyRuntimeReviewRequest{OriginalRequestID: original.Plan.RequestID, Selection: protocol.DeskSelection{Target: b.Target, ConfigRevision: b.ConfigRevision, RuntimeID: facts.Process.ContainerID}}
	apply := protocol.GrubberyRuntimeApplyRequest{OriginalRequestID: q.OriginalRequestID, RequestID: audit.RequestID, Review: facts, Confirm: true}
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: b.RunnerID, Mode: b.Mode, Actor: b.Actor, AccessRole: "operator", Features: []string{protocol.GrubberyRuntimeReceiptsFeature, protocol.GrubberyRuntimeRecoveryFeature}})
			return
		}
		requests.Add(1)
		switch r.URL.Path {
		case protocol.GrubberyRuntimeReviewPath:
			var got protocol.GrubberyRuntimeReviewRequest
			if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&got) != nil || got != q {
				t.Error("review selection changed")
			}
			json.NewEncoder(w).Encode(review)
		case protocol.GrubberyRuntimeRecoveriesPath:
			var got protocol.GrubberyRuntimeApplyRequest
			if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&got) != nil || got.Validate() != nil || got.Review.Digest != facts.Digest || got.RequestID != audit.RequestID {
				t.Error("confirmed exact audit changed")
			}
			json.NewEncoder(w).Encode(status)
		default:
			id, auditID, ok := protocol.GrubberyRuntimeReceiptPath(r.URL.Path)
			if r.Method != "GET" || !ok || id != original.Plan.RequestID || auditID != audit.RequestID {
				t.Error("lookup became an unexpected effect or selection")
			}
			json.NewEncoder(w).Encode(status)
		}
	})}
	go srv.Serve(listener)
	defer srv.Close()
	if _, err = registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	target := service.Target{Name: "dev", RunnerID: b.RunnerID}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "app-runtime-parity", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:"+b.RunnerID, "--enable-mutations")}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, tc := range []struct {
		id    string
		input any
	}{
		{"app.runtime-recovery.v1", service.AppRuntimeStatusInput{Target: target, OriginalRequestID: q.OriginalRequestID, AuditRequestID: audit.RequestID}},
		{"app.review-runtime-recovery.v1", service.AppRuntimeReviewInput{Target: target, Request: q}},
		{"app.apply-runtime-recovery.v1", service.AppRuntimeApplyInput{Target: target, Request: apply}},
	} {
		input, _ := json.Marshal(tc.input)
		cli, cliErr := coreTestCommand(t, "--config-dir", config, "call", tc.id, "--input-json", string(input)).CombinedOutput()
		tool, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: json.RawMessage(input)})
		if cliErr != nil || err != nil || tool.IsError || len(tool.Content) != 1 || !bytes.Equal(bytes.TrimSpace(cli), []byte(tool.Content[0].(*mcp.TextContent).Text)) {
			t.Fatal("CLI/stdio recovery mismatch", tc.id, cliErr, err, string(cli), tool)
		}
	}
	if requests.Load() != 6 {
		t.Fatal("unexpected RPC retry", requests.Load())
	}
	readOnly, err := service.New(service.Options{ConfigDir: config, RequireAllowed: true, Allowed: []service.Target{target}})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(service.AppRuntimeApplyInput{Target: target, Request: apply})
	result := readOnly.Call(ctx, "app.apply-runtime-recovery.v1", input)
	if result.Error == nil || result.Error.Code != "mutation_disabled" || requests.Load() != 6 {
		t.Fatal("audit mutation bypassed startup policy", result)
	}
}
