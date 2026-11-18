//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Actual built CLI/official-SDK stdio to a real managed API/SQLite. Only native
// process/marker observation is inert. No engine, runtime or user store is opened.
func TestGrubberyRuntimeMCPApplyLostReplyCLILookupAcrossManagerRestart(t *testing.T) {
	binary := os.Getenv("OMABIT_TEST_OMABIT_BINARY")
	if binary == "" {
		t.Skip("explicitly built/pinned Omabit source qualification only")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("absolute binary required")
	}
	raw, err := os.ReadFile(binary)
	g02Check(t, err)
	pin := os.Getenv("OMABIT_TEST_OMABIT_BINARY_SHA256")
	if len(pin) != 64 || stopEvidenceDigest(raw) != pin {
		t.Fatal("exact explicitly supplied client binary pin required")
	}
	r, f, original, selection := managedRuntimeRecoveryFixture(t)
	g02Check(t, os.RemoveAll(filepath.Join(r.s.directory, "desk-stages-v1")))
	var active atomic.Pointer[Server]
	active.Store(r)
	var submissions atomic.Int32
	dir := privateDir(t)
	socket := filepath.Join(dir, "runner.sock")
	listener, err := localrpc.Listen(socket)
	g02Check(t, err)
	server := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" && req.URL.Path == protocol.GrubberyRuntimeRecoveriesPath && submissions.Add(1) == 1 {
			committed := httptest.NewRecorder()
			active.Load().ServeHTTP(committed, req)
			if committed.Code != 200 {
				t.Error("loss did not follow committed audit", committed.Code, committed.Body.String())
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
			} else {
				conn.Close()
			}
			return
		}
		active.Load().ServeHTTP(w, req)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	config := filepath.Join(dir, "registry")
	registry, err := collection.Open(config)
	g02Check(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	_, err = registry.Enroll(ctx, "lab", socket)
	g02Check(t, err)
	command := exec.CommandContext(ctx, binary, "--config-dir", config, "mcp", "serve", "--allow-runner", "lab:"+r.s.runnerID, "--enable-mutations")
	command.Env = []string{"HOME=" + dir, "PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "app-runtime-audit-lost-reply", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	g02Check(t, err)
	t.Cleanup(func() { session.Close() })
	target := service.Target{Name: "lab", RunnerID: r.s.runnerID}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.app.review-runtime-recovery.v1", Arguments: service.AppRuntimeReviewInput{Target: target, Request: protocol.GrubberyRuntimeReviewRequest{OriginalRequestID: original.Plan.RequestID, Selection: selection}}})
	g02Check(t, err)
	var review struct {
		Data  protocol.GrubberyRuntimeReview `json:"data"`
		Error *service.Problem               `json:"error"`
	}
	g02Check(t, json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &review))
	if result.IsError || review.Error != nil || review.Data.Validate() != nil {
		t.Fatal("no actual stdio fresh review", review.Error)
	}
	apply := service.AppRuntimeApplyInput{Target: target, Request: protocol.GrubberyRuntimeApplyRequest{OriginalRequestID: original.Plan.RequestID, RequestID: strings.Repeat("5", 64), Review: review.Data.Review, Confirm: true}}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.app.apply-runtime-recovery.v1", Arguments: apply})
	g02Check(t, err)
	var lost service.Result
	g02Check(t, json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &lost))
	if !result.IsError || lost.Error == nil || lost.Error.Code != "outcome_uncertain" || submissions.Load() != 1 {
		t.Fatal("lost apply reply retried or became safe refusal", lost.Error, submissions.Load())
	}
	g02Check(t, session.Close())
	r.mu.Lock()
	reads, writes := f.reads, f.writes
	r.mu.Unlock()
	if writes != 1 {
		t.Fatal("audit repeated original effect")
	}
	g02Check(t, r.Close())
	saved, err := testOpenStore(r.s.directory)
	g02Check(t, err)
	replacementCtx, replacementCancel := context.WithCancel(context.Background())
	defer replacementCancel()
	replacement := &Server{s: saved, ctx: replacementCtx, cancel: replacementCancel, uid: uint32(os.Geteuid())}
	replacement.updateFence.Held = true
	active.Store(replacement)
	t.Cleanup(func() { replacement.Close() })
	input, err := json.Marshal(service.AppRuntimeStatusInput{Target: target, OriginalRequestID: original.Plan.RequestID, AuditRequestID: apply.Request.RequestID})
	g02Check(t, err)
	cli := exec.CommandContext(ctx, binary, "--config-dir", config, "call", "app.runtime-recovery.v1", "--input-json", string(input))
	cli.Env = command.Env
	var stderr bytes.Buffer
	cli.Stderr = &stderr
	output, err := cli.Output()
	if err != nil {
		t.Fatal("actual CLI saved lookup failed", err, stderr.String(), string(output))
	}
	var lookup struct {
		Data  protocol.GrubberyRuntimeRecoveryStatus `json:"data"`
		Error *service.Problem                       `json:"error"`
	}
	g02Check(t, json.Unmarshal(output, &lookup))
	if lookup.Error != nil || lookup.Data.Validate() != nil || lookup.Data.Receipt == nil || lookup.Data.Receipt.RequestID != apply.Request.RequestID || lookup.Data.Receipt.Review.Digest != apply.Request.Review.Digest || !reflect.DeepEqual(lookup.Data.Original, original) || submissions.Load() != 1 || f.reads != reads || f.writes != writes {
		t.Fatal("manager/stdio loss changed audit identity, uncertainty or effects", lookup.Error, submissions.Load())
	}
	t.Log("actual client binary sha256", pin, "committed audit retained across lost reply and manager replacement; native observation inert")
}
