//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The actual built Omabit binary speaks stdio MCP and CLI JSON to one real
// managed API/SQLite journal. Only runtime observation is inert.
func TestProjectStopRecoveryMCPSubmitCLILookupAcrossManagerRestart(t *testing.T) {
	binary := os.Getenv("OMABIT_TEST_OMABIT_BINARY")
	if binary == "" {
		t.Skip("requires explicitly built Omabit")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("binary must be absolute")
	}
	r, b, original := interruptedExpandedStopFixtureActor(t, true, fmt.Sprintf("local-uid:%d", os.Geteuid()))
	r.uid = uint32(os.Geteuid())
	var observations, submissions atomic.Int32
	b.onObserve = func() { observations.Add(1) }
	var active atomic.Pointer[Server]
	active.Store(r)
	dir := privateDir(t)
	socket := filepath.Join(dir, "runner.sock")
	listener, err := localrpc.Listen(socket)
	g02Check(t, err)
	server := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method == "POST" && q.URL.Path == protocol.ProjectStopReconcilePath(original.ProjectID) {
			if submissions.Add(1) == 1 {
				accepted := httptest.NewRecorder()
				active.Load().ServeHTTP(accepted, q)
				if accepted.Code != 200 {
					t.Error("fault did not follow committed resolution", accepted.Code, accepted.Body.String())
				}
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
				} else {
					conn.Close()
				}
				return
			}
			if q.Header.Get(protocol.CreationReplayHeader) != "true" {
				t.Error("lookup lost its guard")
			}
		}
		active.Load().ServeHTTP(w, q)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	registryDir := filepath.Join(dir, "registry")
	registry, err := collection.Open(registryDir)
	g02Check(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = registry.Enroll(ctx, "lab", socket)
	g02Check(t, err)
	command := exec.CommandContext(ctx, binary, "--config-dir", registryDir, "mcp", "serve", "--allow-runner", "lab:"+r.s.runnerID, "--enable-mutations")
	command.Env = []string{"HOME=" + dir, "PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "recovery-fixture", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	g02Check(t, err)
	t.Cleanup(func() { session.Close() })
	target := service.Target{Name: "lab", RunnerID: r.s.runnerID}
	input := service.DiagnosisInput{CheckpointInput: service.CheckpointInput{ProjectInput: service.ProjectInput{Target: target, ProjectID: original.ProjectID}, OperationID: original.ID}, InstanceID: original.InstanceID, Probe: true}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.project.stop-diagnosis.v1", Arguments: input})
	g02Check(t, err)
	var envelope struct {
		Data  protocol.ProjectMemberDiagnosis `json:"data"`
		Error *service.Problem                `json:"error"`
	}
	g02Check(t, json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &envelope))
	if result.IsError || envelope.Error != nil || envelope.Data.Review == nil || observations.Load() != 1 {
		t.Fatal("no MCP review", envelope.Error)
	}
	apply := service.StopReconcileInput{ProjectInput: service.ProjectInput{Target: target, ProjectID: original.ProjectID}, Request: protocol.ProjectStopReconcileRequest{RequestID: "cross-interface-original", Review: *envelope.Data.Review, ConfirmAction: true}}
	g02Check(t, apply.Validate())
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.project.stop-reconcile.v1", Arguments: apply})
	g02Check(t, err)
	var lost service.Result
	g02Check(t, json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &lost))
	if !result.IsError || lost.Error == nil || lost.Error.Code != "outcome_uncertain" || submissions.Load() != 1 || observations.Load() != 2 {
		t.Fatal("lost reply retried or changed guidance", lost.Error, submissions.Load(), observations.Load())
	}
	g02Check(t, session.Close())
	// Reopen authoritative state with no stop-observation backend. Exact lookup
	// must survive manager and MCP process loss without observing or reapplying.
	g02Check(t, r.Close())
	saved, err := testOpenStore(r.s.directory)
	g02Check(t, err)
	replacementCtx, replacementCancel := context.WithCancel(context.Background())
	defer replacementCancel()
	replacement := &Server{s: saved, b: &fakeBackend{}, ctx: replacementCtx, cancel: replacementCancel, uid: uint32(os.Geteuid()), workGate: make(chan struct{}, 1)}
	active.Store(replacement)
	t.Cleanup(func() { replacement.Close() })
	apply.LookupOnly = true
	body, err := json.Marshal(apply)
	g02Check(t, err)
	cli := exec.CommandContext(ctx, binary, "--config-dir", registryDir, "call", "project.stop-reconcile.v1", "--input-json", string(body))
	cli.Env = command.Env
	var diagnostic bytes.Buffer
	cli.Stderr = &diagnostic
	output, err := cli.Output()
	if err != nil {
		t.Fatal(err, diagnostic.String(), string(output))
	}
	var lookup struct {
		Data  protocol.ProjectStopReconcileResult `json:"data"`
		Error *service.Problem                    `json:"error"`
	}
	g02Check(t, json.Unmarshal(output, &lookup))
	if lookup.Error != nil || lookup.Data.OriginalOperationID != original.ID || lookup.Data.Operation.RequestID != apply.Request.RequestID || lookup.Data.Operation.Status != "succeeded" || submissions.Load() != 2 || observations.Load() != 2 {
		t.Fatal("CLI lookup changed identity/effects", lookup, submissions.Load(), observations.Load())
	}
	retained, err := loadOperation(saved.db, original.ID)
	g02Check(t, err)
	if !reflect.DeepEqual(retained, original) {
		t.Fatal("original interrupted evidence changed")
	}
	// Both interface failures are bounded and sanitized, never raw private text.
	if bytes.Contains(output, []byte(original.Message)) && original.Message != "" {
		t.Fatal("original private diagnostic leaked")
	}
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
}
