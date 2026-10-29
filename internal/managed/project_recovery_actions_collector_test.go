//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Actual development collector, private Unix RPC and inert recovery fixtures.
// No Docker, SSH, systemd, credential store or ship process is used.
func TestPublicRecoveryActionActualCollectorLostReply(t *testing.T) {
	binary := os.Getenv("OMABIT_TEST_COLLECTOR_BINARY")
	if binary == "" {
		t.Skip("requires explicitly built development collector")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("absolute collector required")
	}
	for _, kind := range []string{"start", "observe", "continue"} {
		t.Run(kind, func(t *testing.T) {
			f := publicRecoveryActionFixture(t, kind)
			r := f.r
			r.workGate <- struct{}{}
			var actions atomic.Int32
			socket := filepath.Join(privateDir(t), "runner.sock")
			listener, err := localrpc.Listen(socket)
			g02Check(t, err)
			server := &http.Server{ConnContext: PeerContext, Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
				if _, _, _, ok := protocol.ProjectRecoveryActionIDs(q.URL.Path); ok && q.Method == "POST" {
					if actions.Add(1) == 1 {
						accepted := httptest.NewRecorder()
						r.ServeHTTP(accepted, q)
						if accepted.Code != 202 {
							t.Error("fault must follow acceptance", accepted.Code, accepted.Body.String())
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
			pin := []string{"--runner", "lab", "--expected-runner-id", r.s.runnerID}
			args := append(append([]string{}, pin...), "project-recover", f.project, kind, "--revision", strconv.FormatUint(f.req.ExpectedRevision, 10), "--request-id", f.req.RequestID, "--confirm-action")
			if kind == "start" {
				raw := run(0, append(append([]string{}, pin...), "project-recovery-source", f.project)...)
				var source protocol.ProjectRecoverySource
				g02Check(t, json.Unmarshal(raw, &source))
				if source.HistoryDigest != f.req.HistoryDigest {
					t.Fatal("saved source changed", source)
				}
				args = append(args, "--review-digest", source.Project.ReviewDigest, "--history-digest", source.HistoryDigest)
			} else {
				raw := run(0, append(append([]string{}, pin...), "project-recovery-checkpoint", f.project, f.parent)...)
				var checkpoint protocol.ProjectRecoveryCheckpoint
				g02Check(t, json.Unmarshal(raw, &checkpoint))
				if checkpoint.CheckpointDigest != f.req.CheckpointDigest || checkpoint.Sequence != *f.req.ExpectedSequence {
					t.Fatal("checkpoint changed", checkpoint)
				}
				args = append(args, "--review-digest", checkpoint.Project.ReviewDigest, "--history-digest", checkpoint.HistoryDigest, "--recovery", f.parent, "--sequence", strconv.FormatUint(checkpoint.Sequence, 10), "--checkpoint-digest", checkpoint.CheckpointDigest)
			}
			lost := run(1, args...)
			if !strings.Contains(string(lost), "reconcile_project_recovery") || !strings.Contains(string(lost), "--replay-only") || actions.Load() != 1 {
				t.Fatal("lost reply repeated or lost guidance", string(lost), actions.Load())
			}
			lookup := run(0, append(args, "--replay-only")...)
			var receipt protocol.ProjectRecoveryActionResult
			g02Check(t, json.Unmarshal(lookup, &receipt))
			if receipt.Operation.Status != "queued" || receipt.Operation.RequestID != f.req.RequestID || actions.Load() != 2 || r.updateWorkers != 1 {
				t.Fatal("lookup changed admission", string(lookup), actions.Load())
			}
			f.accepted(receipt.Operation.ID)
			<-r.workGate
			f.release()
			waitRecoveryQueue(t, r)
			calls := append([]string(nil), f.b.calls...)
			raw := run(0, append(args, "--replay-only")...)
			var terminal protocol.ProjectRecoveryActionResult
			g02Check(t, json.Unmarshal(raw, &terminal))
			if terminal.Operation.ID != receipt.Operation.ID || terminal.Operation.Status != "succeeded" || actions.Load() != 3 || r.updateWorkers != 0 || !reflect.DeepEqual(calls, f.b.calls) {
				t.Fatal("terminal lookup changed effects", string(raw))
			}
			status := run(0, append(append([]string{}, pin...), "project-status", f.project)...)
			var saved protocol.ProjectStatus
			g02Check(t, json.Unmarshal(status, &saved))
			found := false
			for _, op := range saved.Operations {
				if op.ID == receipt.Operation.ID {
					found = op.Status == "succeeded"
				}
			}
			if !found || saved.RuntimeObservation != "not-performed" {
				t.Fatal("receipt absent from saved history", string(status))
			}
		})
	}
}
