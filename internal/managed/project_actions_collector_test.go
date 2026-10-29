//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// Opt-in actual collector binary, temporary Unix RPC and the existing inert
// engine/store fixture. No Docker, SSH, systemd, keyring or ship process is used.
func TestProjectActionActualCollectorLostReply(t *testing.T) {
	binary := os.Getenv("OMABIT_TEST_COLLECTOR_BINARY")
	if binary == "" {
		t.Skip("requires the explicitly built development collector")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("collector path must be absolute")
	}
	r, backend, _, _ := projectQueueFixture(t)
	r.uid = uint32(os.Geteuid())
	r.workGate <- struct{}{}
	var actions atomic.Int32
	socket := filepath.Join(privateDir(t), "runner.sock")
	listener, err := localrpc.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if _, ok := protocol.ProjectActionID(q.URL.Path); ok && q.Method == "POST" {
			if actions.Add(1) == 1 {
				accepted := httptest.NewRecorder()
				r.ServeHTTP(accepted, q)
				if accepted.Code != 202 {
					t.Error("fault must follow accepted work", accepted.Code, accepted.Body.String())
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
				t.Error("follow-up must be lookup-only")
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
	private := privateDir(t)
	registry := filepath.Join(private, "registry")
	run := func(want int, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, append([]string{"--config-dir", registry}, args...)...)
		command.Env = []string{"HOME=" + private, "XDG_CONFIG_HOME=" + private, "PATH=/usr/bin:/bin", "LC_ALL=C"}
		var out, diagnostic bytes.Buffer
		command.Stdout, command.Stderr = &out, &diagnostic
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
	plan := run(0, "project-plan", "--label", "CLI integration", "--member", "cli-zod:~zod:19381", "--member", "cli-nec:~nec:19382", "--shared-administration")
	var review struct {
		ReviewDigest string `json:"review_digest"`
	}
	if json.Unmarshal(plan, &review) != nil || len(review.ReviewDigest) != 64 {
		t.Fatal("invalid planner output")
	}
	planPath := filepath.Join(private, "plan.json")
	if err := os.WriteFile(planPath, plan, 0600); err != nil {
		t.Fatal(err)
	}
	pin := []string{"--runner", "lab", "--expected-runner-id", r.s.runnerID}
	registered := run(0, append(append([]string{}, pin...), "project-register", "--plan", planPath, "--review-digest", review.ReviewDigest, "--request-id", "cli-register", "--confirm-reservation")...)
	var registration protocol.ProjectRegistrationResult
	if json.Unmarshal(registered, &registration) != nil || !registration.ReservationOnly {
		t.Fatal(string(registered))
	}
	id := registration.Operation.ProjectID
	args := append(append([]string{}, pin...), "project-operate", id, "prepare", "--revision", "1", "--review-digest", review.ReviewDigest, "--request-id", "cli-prepare", "--confirm-action")
	lost := run(1, args...)
	if !strings.Contains(string(lost), "reconcile_project_action") || !strings.Contains(string(lost), "--replay-only") || actions.Load() != 1 {
		t.Fatal("lost reply repeated or lost guidance", actions.Load(), string(lost))
	}
	lookup := run(0, append(args, "--replay-only")...)
	var receipt protocol.ProjectActionResult
	if json.Unmarshal(lookup, &receipt) != nil || receipt.Operation.Status != "queued" || receipt.Operation.RequestID != "cli-prepare" || actions.Load() != 2 || r.updateWorkers != 1 {
		t.Fatal("lookup changed admission", string(lookup), actions.Load(), r.updateWorkers)
	}
	<-r.workGate
	close(backend.release)
	waitProjectWorker(t, r)
	status := run(0, append(append([]string{}, pin...), "project-status", id)...)
	var saved protocol.ProjectStatus
	if json.Unmarshal(status, &saved) != nil || len(saved.Operations) != 2 || saved.Operations[0].ID != receipt.Operation.ID || saved.Operations[0].Status != "succeeded" || saved.RuntimeObservation != "not-performed" {
		t.Fatal(string(status))
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.apply != 0 {
		t.Fatal("ordinary ship runtime was invoked")
	}
}
