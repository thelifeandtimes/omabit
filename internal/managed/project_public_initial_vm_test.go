//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strconv"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// This gate reserves two fake identities and prepares their empty storage, but
// starts only the private namespace anchor. No ship is booted by this entry.
func TestProjectPublicInitialDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, false, false, false, false, false, false, false, false, false, true)
}

type vmPublicPreparationBackend struct {
	fakeBackend
	store    *store
	ready    chan struct{}
	release  chan struct{}
	prepared projectnetwork.Preparation
	wrapped  *vmPreparationEngine
	err      error
}

func (b *vmPublicPreparationBackend) OpenProjectPreparation(ctx context.Context, s runtimequalify.ProjectPreparationSelection) (projectnetwork.Preparation, projectnetwork.PreparationEngine, error) {
	prepared, adapter, err := runtimequalify.OpenProjectPreparation(ctx, s)
	b.prepared, b.err = prepared, err
	if err == nil {
		b.wrapped = &vmPreparationEngine{store: b.store, id: s.ProjectID, engine: adapter}
	}
	close(b.ready)
	if err != nil {
		return prepared, nil, err
	}
	// Let the fixture record volume inventory after session setup but before any
	// bridge/anchor effects, preserving the existing durability/inventory checks.
	select {
	case <-b.release:
		return prepared, b.wrapped, nil
	case <-ctx.Done():
		return prepared, nil, ctx.Err()
	}
}

type vmInitialPublicSetup struct {
	server  *Server
	backend *vmPublicPreparationBackend
	cli     *vmPublicRecoveryCLI
	close   func()
}

func vmOpenInitialPublicSetup(ctx context.Context, s *store, token string) (*vmInitialPublicSetup, error) {
	child, cancel := context.WithCancel(ctx)
	b := &vmPublicPreparationBackend{store: s, ready: make(chan struct{}), release: make(chan struct{})}
	r := &Server{s: s, b: b, ctx: child, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
	cli, err := vmOpenPublicProjectCLI(ctx, r, "", token, "initial")
	if err != nil {
		cancel()
		return nil, err
	}
	return &vmInitialPublicSetup{server: r, backend: b, cli: cli, close: func() { cancel(); cli.close(); r.wg.Wait() }}, nil
}
func vmWaitPublicWorker(ctx context.Context, r *Server) error {
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
		if r.failed.Load() {
			return errors.New("public project worker failed runner")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *vmPublicRecoveryCLI) registerInitial() (protocol.Operation, error) {
	var zero protocol.Operation
	raw, err := c.run("project-plan", "--label", "Development", "--member", "vm-zod:~zod:18381", "--member", "vm-nec:~nec:18382", "--shared-administration")
	if err != nil {
		return zero, err
	}
	var review struct {
		Plan              projectnetwork.Plan `json:"plan"`
		ReviewDigest      string              `json:"review_digest"`
		RuntimeAuthorized bool                `json:"runtime_authorized"`
		ReservationCheck  string              `json:"reservation_check"`
	}
	if json.Unmarshal(raw, &review) != nil {
		return zero, errors.New("invalid actual CLI plan")
	}
	digest, err := review.Plan.Digest()
	if err != nil || digest != review.ReviewDigest || review.RuntimeAuthorized || review.ReservationCheck != "not-performed" {
		return zero, errors.New("CLI plan review changed")
	}
	const path = "/home/lab/project-public-initial-plan.json"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return zero, err
	}
	_, writeErr := f.Write(raw)
	if err = errors.Join(writeErr, f.Sync(), f.Close()); err != nil {
		return zero, err
	}
	args := []string{"project-register", "--plan", path, "--review-digest", digest, "--request-id", "register-vm", "--confirm-reservation"}
	var original protocol.ProjectRegistrationResult
	for _, lookup := range []bool{false, true} {
		request := append([]string(nil), args...)
		if lookup {
			request = append(request, "--replay-only")
		}
		raw, err = c.pinned(request...)
		if err != nil {
			return zero, err
		}
		var result protocol.ProjectRegistrationResult
		if json.Unmarshal(raw, &result) != nil || !result.ReservationOnly || result.ReviewDigest != digest || result.Operation.RequestID != "register-vm" || result.Operation.Actor != "local-uid:1000" || result.Operation.ProjectID == "" {
			return zero, errors.New("public reservation receipt changed")
		}
		if lookup && !reflect.DeepEqual(original, result) {
			return zero, errors.New("public reservation lookup changed receipt")
		}
		original = result
		c.calls = append(c.calls, map[string]any{"action": "register", "lookup_only": lookup, "selection": review, "receipt": result})
	}
	c.proof["calls"] = c.calls
	c.project = original.Operation.ProjectID
	return original.Operation, nil
}

func (c *vmPublicRecoveryCLI) operate(kind, member, requestID string, revision uint64, digest string, replay bool) (protocol.Operation, error) {
	req := protocol.ProjectActionRequest{Kind: kind, InstanceID: member, RequestID: requestID, ExpectedRevision: revision, ReviewDigest: digest, ConfirmAction: true}
	return c.operateRequest(req, replay)
}

func (c *vmPublicRecoveryCLI) operateRequest(req protocol.ProjectActionRequest, replay bool) (protocol.Operation, error) {
	var zero protocol.Operation
	kind, member, requestID, revision, digest := req.Kind, req.InstanceID, req.RequestID, req.ExpectedRevision, req.ReviewDigest
	if !protocol.ValidProjectActionRequest(req) {
		return zero, errors.New("invalid reviewed fixture action")
	}
	args := []string{"project-operate", c.project, kind, "--revision", strconv.FormatUint(revision, 10), "--review-digest", digest, "--request-id", requestID, "--confirm-action"}
	if member != "" {
		args = append(args, "--member", member)
	}
	if req.AfterOperationID != "" {
		args = append(args, "--after", req.AfterOperationID)
	}
	if req.OriginalReadinessOperationID != "" {
		args = append(args, "--original-readiness", req.OriginalReadinessOperationID)
	}
	if replay {
		args = append(args, "--replay-only")
	}
	entry := map[string]any{"action": kind, "request": req, "lookup_only": replay}
	raw, err := c.pinned(args...)
	if err != nil {
		entry["refused"] = true
		entry["diagnostic"] = err.Error()
		c.calls = append(c.calls, entry)
		c.proof["calls"] = c.calls
		return zero, err
	}
	var result protocol.ProjectActionResult
	_, operation, _ := protocol.ProjectAction(kind)
	if json.Unmarshal(raw, &result) != nil || result.Operation.ProjectID != c.project || result.Operation.RequestID != requestID || result.Operation.InstanceID != member || result.Operation.Actor != "local-uid:1000" || result.Operation.Kind != operation || result.ReviewDigest != digest {
		return zero, errors.New("public initial action receipt changed")
	}
	entry["receipt"] = result
	c.calls = append(c.calls, entry)
	c.proof["calls"] = c.calls
	return result.Operation, nil
}

func TestVMPublicProjectCLIRequiresGuestBeforeListener(t *testing.T) {
	for _, v := range []struct{ token, role string }{
		{"offline-project-public-initial-v1", "initial"},
		{"offline-project-public-initial-v1", "anchor"},
		{"offline-project-public-initial-v1", "storage"},
		{"offline-project-public-recovery-v1", "recovery"},
		{"offline-project-public-initial-v1", "unexpected"},
		{"offline-project-public-members-v1", "members"},
		{"offline-project-public-members-v1", "initial"},
		{"offline-project-public-members-v1", "recovery"},
	} {
		t.Setenv("OMABIT_TEST_PROJECT_VM", v.token)
		// The ordinary host has no guest marker. A nil server must never be used.
		cli, err := vmOpenPublicProjectCLI(context.Background(), nil, "", v.token, v.role)
		if err == nil || cli != nil {
			t.Fatal("host public CLI admission unexpectedly succeeded")
		}
	}
}
