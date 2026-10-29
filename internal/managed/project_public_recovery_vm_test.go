//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/localrpc"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestProjectPublicRecoveryDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, true, true, true, true)
}

type vmPublicRecoveryCLI struct {
	ctx      context.Context
	runner   string
	registry string
	project  string
	proof    map[string]any
	calls    []map[string]any
	close    func()
}
type vmRecoveryOutput struct{ data bytes.Buffer }

func (b *vmRecoveryOutput) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > 1024*1024 {
		return 0, errors.New("qualification CLI output exceeds limit")
	}
	return b.data.Write(p)
}

func (b *vmRecoveryOutput) Bytes() []byte  { return b.data.Bytes() }
func (b *vmRecoveryOutput) String() string { return b.data.String() }

// Called only after the common no-NIC guest guard. The extra marker refusal
// precedes all listener, registry and subprocess effects.
func vmOpenPublicRecoveryCLI(ctx context.Context, r *Server, p storedProject) (*vmPublicRecoveryCLI, error) {
	token := "offline-project-public-recovery-v1"
	if os.Getenv("OMABIT_TEST_PROJECT_VM") == vmServiceToken {
		token = vmServiceToken
	}
	return vmOpenPublicProjectCLI(ctx, r, p.ID, token, "recovery")
}
func vmOpenPublicProjectCLI(ctx context.Context, r *Server, project, token, role string) (*vmPublicRecoveryCLI, error) {
	marker, err := os.ReadFile("/opt/omabit-lab/guest-marker")
	allowed := (token == "offline-project-public-recovery-v1" || token == vmServiceToken) && role == "recovery" || (token == "offline-project-public-initial-v1" || token == "offline-project-public-members-v1") && (role == "initial" || role == "anchor" || role == "storage") || token == "offline-project-public-members-v1" && role == "members"
	if err != nil || string(marker) != "offline-import-guest-v1\n" || os.Getenv("OMABIT_TEST_PROJECT_VM") != token || !allowed {
		return nil, errors.New("public project CLI requires admitted disposable guest")
	}
	socket := "/home/lab/project-public-" + role + ".sock"
	registry := "/home/lab/project-public-" + role + "-registry"
	listener, err := localrpc.Listen(socket)
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: r, ConnContext: PeerContext, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	c := &vmPublicRecoveryCLI{ctx: ctx, runner: r.s.runnerID, project: project, registry: registry, proof: map[string]any{"ok": false, "actual_collector": true, "registry": registry}, close: func() { server.Close(); <-done }}
	if _, err = c.run("enroll", "lab", "--socket", socket); err != nil {
		c.close()
		return nil, err
	}
	return c, nil
}
func (c *vmPublicRecoveryCLI) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/opt/omabit-lab/urbit-collector", append([]string{"--config-dir", c.registry}, args...)...)
	cmd.Env = []string{"HOME=/home/lab", "XDG_RUNTIME_DIR=/run/user/1000", "PATH=/usr/bin:/bin", "LC_ALL=C"}
	var out, diagnostic vmRecoveryOutput
	cmd.Stdout, cmd.Stderr = &out, &diagnostic
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("public recovery CLI %s: %w: %.512s", args[0], err, diagnostic.String())
	}
	return out.Bytes(), nil
}
func (c *vmPublicRecoveryCLI) pinned(args ...string) ([]byte, error) {
	return c.run(append([]string{"--runner", "lab", "--expected-runner-id", c.runner}, args...)...)
}
func (c *vmPublicRecoveryCLI) recover(kind, parent string, req protocol.ProjectRecoveryActionRequest, replay bool) (protocol.Operation, error) {
	var zero protocol.Operation
	entry := map[string]any{"action": kind, "parent": parent, "request": req, "lookup_only": replay}
	if !replay {
		if kind == "start" {
			raw, err := c.pinned("project-recovery-source", c.project)
			if err != nil {
				return zero, err
			}
			var source protocol.ProjectRecoverySource
			if json.Unmarshal(raw, &source) != nil || source.Project.ID != c.project || source.Project.Revision != req.ExpectedRevision || source.Project.ReviewDigest != req.ReviewDigest || source.HistoryDigest != req.HistoryDigest || source.ActionAuthority != "not-granted" || source.RuntimeObservation != "not-performed" {
				return zero, errors.New("public recovery source differs from reviewed fixture history")
			}
			entry["selection"] = source
		} else {
			raw, err := c.pinned("project-recovery-checkpoint", c.project, parent)
			if err != nil {
				return zero, err
			}
			var checkpoint protocol.ProjectRecoveryCheckpoint
			if json.Unmarshal(raw, &checkpoint) != nil || checkpoint.Operation.ID != parent || checkpoint.Project.ID != c.project || checkpoint.Project.Revision != req.ExpectedRevision || checkpoint.Project.ReviewDigest != req.ReviewDigest || checkpoint.HistoryDigest != req.HistoryDigest || req.ExpectedSequence == nil || checkpoint.Sequence != *req.ExpectedSequence || checkpoint.CheckpointDigest != req.CheckpointDigest || checkpoint.ActionAuthority != "not-granted" || checkpoint.RuntimeObservation != "not-performed" {
				return zero, errors.New("public checkpoint differs from exact retained parent")
			}
			entry["selection"] = checkpoint
		}
	}
	args := []string{"project-recover", c.project, kind, "--revision", strconv.FormatUint(req.ExpectedRevision, 10), "--review-digest", req.ReviewDigest, "--history-digest", req.HistoryDigest, "--request-id", req.RequestID, "--confirm-action"}
	if kind != "start" {
		args = append(args, "--recovery", parent, "--sequence", strconv.FormatUint(*req.ExpectedSequence, 10), "--checkpoint-digest", req.CheckpointDigest)
	}
	if replay {
		args = append(args, "--replay-only")
	}
	raw, err := c.pinned(args...)
	if err != nil {
		return zero, err
	}
	var result protocol.ProjectRecoveryActionResult
	if json.Unmarshal(raw, &result) != nil || result.Operation.ProjectID != c.project || result.Operation.RequestID != req.RequestID {
		return zero, errors.New("public recovery receipt differs from request")
	}
	entry["receipt"] = result
	c.calls = append(c.calls, entry)
	c.proof["calls"] = c.calls
	return result.Operation, nil
}
func vmPublicRecoveryRequest(req projectRecoveryObservationRequest) protocol.ProjectRecoveryActionRequest {
	seq := req.ExpectedSequence
	return protocol.ProjectRecoveryActionRequest{RequestID: req.RequestID, ExpectedRevision: req.ExpectedRevision, ReviewDigest: req.ReviewDigest, HistoryDigest: req.HistoryDigest, ExpectedSequence: &seq, CheckpointDigest: req.CheckpointDigest, ConfirmAction: true}
}

func TestVMRecoveryOutputBound(t *testing.T) {
	var b vmRecoveryOutput
	chunk := make([]byte, 1024*1024)
	if n, e := b.Write(chunk); e != nil || n != len(chunk) {
		t.Fatal(n, e)
	}
	if n, e := b.Write([]byte("x")); e == nil || n != 0 || len(b.Bytes()) != len(chunk) {
		t.Fatal(n, e)
	}
}
