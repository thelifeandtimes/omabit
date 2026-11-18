//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// Actual public collector + HTTP routing + SQLite admission, with an inert
// runtime observer. This is interface qualification, not live ship qualification.
func TestProjectStopRecoveryActualCollectorLostReply(t *testing.T) {
	binary := os.Getenv("OMABIT_TEST_COLLECTOR_BINARY")
	if binary == "" {
		t.Skip("requires explicitly built collector")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("collector binary must be absolute")
	}
	r, b, original := interruptedExpandedStopFixtureActor(t, true, fmt.Sprintf("local-uid:%d", os.Geteuid()))
	r.uid = uint32(os.Geteuid())
	dir, err := os.MkdirTemp("/tmp", "omabit-stop-cli-")
	g02Check(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "runner.sock")
	listener, err := localrpc.Listen(socket)
	g02Check(t, err)
	var submissions atomic.Int32
	server := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.Method == "POST" && q.URL.Path == protocol.ProjectStopReconcilePath(original.ProjectID) {
			if submissions.Add(1) == 1 {
				accepted := httptest.NewRecorder()
				r.ServeHTTP(accepted, q)
				if accepted.Code != 200 {
					t.Error("fault must follow completed reconciliation", accepted.Code, accepted.Body.String())
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
				t.Error("follow-up lost lookup-only guard")
			}
		}
		r.ServeHTTP(w, q)
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Close()
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	})
	registry := filepath.Join(dir, "registry")
	run := func(want int, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, append([]string{"--config-dir", registry}, args...)...)
		command.Env = []string{"HOME=" + dir, "XDG_CONFIG_HOME=" + dir, "PATH=/usr/bin:/bin", "LC_ALL=C"}
		var out, diagnostic bytes.Buffer
		command.Stdout = &out
		command.Stderr = &diagnostic
		err := command.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatal(err)
			}
			code = exit.ExitCode()
		}
		if code != want {
			t.Fatal(args, code, diagnostic.String())
		}
		if want != 0 {
			return diagnostic.Bytes()
		}
		return out.Bytes()
	}
	run(0, "enroll", "lab", "--socket", socket)
	pin := []string{"--runner", "lab", "--expected-runner-id", r.s.runnerID}
	args := append(append([]string{}, pin...), "project-diagnose", original.ProjectID, "--member", original.InstanceID, "--operation", original.ID, "--probe")
	review := run(0, args...)
	var diagnosis protocol.ProjectMemberDiagnosis
	g02Check(t, json.Unmarshal(review, &diagnosis))
	if diagnosis.Review == nil || b.observations != 1 {
		t.Fatal("CLI review unavailable", string(review))
	}
	path := filepath.Join(dir, "review.json")
	g02Check(t, os.WriteFile(path, review, 0600))
	apply := append(append([]string{}, pin...), "project-reconcile-stop", original.ProjectID, "--review", path, "--request-id", "same-reconciliation", "--confirm-action")
	lost := run(1, apply...)
	if !strings.Contains(string(lost), "lookup_stop_reconciliation") || !strings.Contains(string(lost), "--replay-only") || submissions.Load() != 1 || b.observations != 2 {
		t.Fatal("lost reply retried or lost guidance", string(lost), submissions.Load(), b.observations)
	}
	lookup := run(0, append(apply, "--replay-only")...)
	var result protocol.ProjectStopReconcileResult
	g02Check(t, json.Unmarshal(lookup, &result))
	if result.OriginalOperationID != original.ID || result.Operation.Status != "succeeded" || submissions.Load() != 2 || b.observations != 2 {
		t.Fatal("lookup changed effects", string(lookup))
	}
	retained, err := loadOperation(r.s.db, original.ID)
	g02Check(t, err)
	if retained.Status != "interrupted" || retained.Message != original.Message {
		t.Fatal("original evidence lost")
	}
	state, err := loadProjectExpansion(r.s.db, original.ProjectID)
	g02Check(t, err)
	for _, m := range state.Members {
		if m.Member.InstanceID == original.InstanceID && m.Phase != "stopped" {
			t.Fatal("member fence did not resolve")
		}
	}
}
