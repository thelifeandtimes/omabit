package managed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// This optional regression runs the unchanged published RC.11 coordinator, not
// today's MoveStopped implementation. The binary is supplied locally and verified
// before execution; all endpoints, storage, actors and backends are fixtures.
func TestHandoffPublishedRC11CollectorContinuation(t *testing.T) {
	binary := os.Getenv("OMABIT_RC11_COLLECTOR")
	if binary == "" {
		t.Skip("set OMABIT_RC11_COLLECTOR to the published RC.11 linux-amd64 collector")
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	const publishedSHA256 = "c8825ef9d5041a0b1cf4ac1f86fed84a135ebcc496a9e207c96302c486be5c78"
	if fmt.Sprintf("%x", sha256.Sum256(data)) != publishedSHA256 {
		t.Fatal("collector does not match the verified published RC.11 binary")
	}
	executable := filepath.Join(t.TempDir(), "rc11-collector")
	if err := os.WriteFile(executable, data, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	a, b := privateDir(t), privateDir(t)
	sourceSocket, destinationSocket := filepath.Join(a, "rpc.sock"), filepath.Join(b, "rpc.sock")
	var interruptCommit atomic.Bool
	interruptCommit.Store(true)
	sourceFilter := func(w http.ResponseWriter, req *http.Request) bool {
		if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/commit") && interruptCommit.Swap(false) {
			w.Header().Set(protocol.Header, "1")
			respond(w, 503, protocol.ErrorResponse{Error: protocol.APIError{Code: "fixture_before_commit", Message: "interrupted after destination receipt"}})
			return true
		}
		return false
	}
	var uploads atomic.Int32
	destinationFilter := func(w http.ResponseWriter, req *http.Request) bool {
		if req.Method == "PUT" && protocol.HandoffDataPath(req.URL.Path) {
			uploads.Add(1)
		}
		return false
	}
	src, source, _ := handoffCompatibilityEndpoint(t, a, sourceSocket, &handoffBackend{dir: a}, sourceFilter)
	destinationBackend := &handoffBackend{dir: b}
	dst, _, closeDestination := handoffCompatibilityEndpoint(t, b, destinationSocket, destinationBackend, destinationFilter)
	if _, err := source.CreateFake(ctx, registration("ship", "register", 18080)); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"start", "stop"} {
		op, err := source.Operate(ctx, "ship", protocol.OperationRequest{Kind: kind, RequestID: kind, ExpectedRevision: 1})
		if err != nil {
			t.Fatal(err)
		}
		waitOperation(t, source, op.ID, "succeeded")
	}
	config := filepath.Join(t.TempDir(), "collector")
	invoke := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, executable, append([]string{"--config-dir", config}, args...)...).CombinedOutput()
	}
	for name, socket := range map[string]string{"source": sourceSocket, "destination": destinationSocket} {
		if out, err := invoke("enroll", name, "--socket", socket); err != nil {
			t.Fatalf("old collector enrollment: %v %s", err, out)
		}
	}
	args := []string{"--runner", "source", "--expected-runner-id", src.s.runnerID, "move-host", "ship", "--to-runner", "destination", "--to-runner-id", dst.s.runnerID, "--to-instance", "arrived", "--revision", "1", "--request-id", "original", "--http-port", "18081"}
	if out, err := invoke(args...); err == nil || !bytes.Contains(out, []byte("fixture_before_commit")) {
		t.Fatalf("old collector did not reach injected interruption: %v %s", err, out)
	}
	if receiptLoad(t, src, "ship").Runtime.Handoff.State != "ready" || receiptLoad(t, dst, "arrived").Runtime.Handoff.State != "received" || uploads.Load() != 1 {
		t.Fatal("fixture did not stop at source ready / destination received")
	}
	actor := fmt.Sprintf("local-uid:%d", os.Geteuid())
	audit := handoffAuditRow(t, dst, actor, "original-receive")
	var original handoffReceiptOperation
	if err := json.Unmarshal([]byte(strings.SplitN(audit, "\x00", 2)[0]), &original); err != nil || original.Receipt == nil || original.Receipt.Status != 201 || original.Receipt.Handoff.State != "reserved" {
		t.Fatal("original acceptance audit missing", err)
	}
	closeDestination()
	dst, _, _ = handoffCompatibilityEndpoint(t, b, destinationSocket, destinationBackend, destinationFilter)
	for attempt := 0; attempt < 2; attempt++ {
		out, err := invoke(args...) // identical old binary, selection and request ID
		if err != nil {
			t.Fatalf("old collector same-request continuation: %v %s", err, out)
		}
		var completed protocol.Handoff
		if err := json.Unmarshal(out, &completed); err != nil || completed.State != "complete" || !completed.ValidCommitment(completed.Commitment) {
			t.Fatalf("old collector completion: %v %s", err, out)
		}
		if uploads.Load() != 1 {
			t.Fatal("old collector retransmitted an already received archive", uploads.Load())
		}
		if got := handoffAuditRow(t, dst, actor, "original-receive"); got != audit {
			t.Fatal("old collector continuation rewrote the original reservation receipt")
		}
	}
	from, to := receiptLoad(t, src, "ship"), receiptLoad(t, dst, "arrived")
	if from.Runtime.Handoff.State != "moved" || !from.Runtime.Handoff.Fenced() || to.Runtime.Handoff.State != "complete" || to.Runtime.Handoff.Fenced() || to.Public.DesiredState != "stopped" {
		t.Fatal("old collector completion lost source fence or stopped destination intent")
	}
	t.Log("verified published RC.11 collector resumed after destination restart; one upload, unchanged original reservation audit, signed completion")
}

func handoffCompatibilityEndpoint(t *testing.T, dir, socket string, b backend, intercept func(http.ResponseWriter, *http.Request) bool) (*Server, *client.Client, func()) {
	t.Helper()
	r, err := newServer(dir, b, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := localrpc.Listen(socket)
	if err != nil {
		r.Close()
		t.Fatal(err)
	}
	srv := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !intercept(w, req) {
			r.ServeHTTP(w, req)
		}
	})}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	c := client.New(socket)
	var once sync.Once
	close := func() {
		once.Do(func() {
			c.Close()
			srv.Close()
			<-done
			if err := r.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(close)
	return r, c, close
}
