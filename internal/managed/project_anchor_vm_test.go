//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func TestProjectAnchorStartDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, false, false, false, false, false, false, false)
}

type vmAnchorBackend struct {
	fakeBackend
	store   *store
	id      string
	wrapped *vmAnchorEngine
}

func (b *vmAnchorBackend) OpenProjectAnchorStart(ctx context.Context, selection runtimequalify.ProjectPreparationSelection, p projectnetwork.Preparation) (projectnetwork.AnchorStartEngine, error) {
	engine, err := runtimequalify.OpenProjectAnchorStart(ctx, selection, p)
	if err != nil {
		return nil, err
	}
	b.wrapped = &vmAnchorEngine{store: b.store, id: b.id, engine: engine}
	return b.wrapped, nil
}

type vmAnchorEngine struct {
	store            *store
	id               string
	engine           projectnetwork.AnchorStartEngine
	calls            []string
	observations     []projectnetwork.Observation
	inspectionErrors []string
}

func (e *vmAnchorEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.engine.VerifyEngine(ctx, id)
}
func (e *vmAnchorEngine) StartAnchor(ctx context.Context, p projectnetwork.Preparation) error {
	state, err := loadProjectAnchorStart(e.store.db, e.id)
	if err != nil {
		return err
	}
	if state.Phase != "dispatched" || !reflect.DeepEqual(state.Prepared, p) {
		return errors.New("Start preceded committed dispatch")
	}
	e.calls = append(e.calls, "start-dispatched")
	return e.engine.StartAnchor(ctx, p)
}
func (e *vmAnchorEngine) InspectAnchor(ctx context.Context, p projectnetwork.Preparation) (projectnetwork.Observation, error) {
	state, err := loadProjectAnchorStart(e.store.db, e.id)
	if err != nil {
		return projectnetwork.Observation{}, err
	}
	expected := "planned"
	if len(e.observations) > 0 {
		expected = "started"
	}
	if state.Phase != expected || !reflect.DeepEqual(state.Prepared, p) {
		return projectnetwork.Observation{}, errors.New("inspection preceded prior committed phase")
	}
	e.calls = append(e.calls, "inspect-"+expected)
	observed, err := e.engine.InspectAnchor(ctx, p)
	if err != nil {
		// This guarded fixture invokes fixed private-anchor commands only. Docker
		// stdout/stderr are never in the adapter's bounded diagnostic errors.
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		e.inspectionErrors = append(e.inspectionErrors, message)
	}

	if err == nil {
		e.observations = append(e.observations, observed)
	}
	return observed, err
}

func exerciseVMAnchorStart(t *testing.T, ctx context.Context, s *store, id string, docker func(...string) json.RawMessage, registerCleanup func(func()), publicMode ...bool) map[string]any {
	t.Helper()
	result := map[string]any{"ok": false, "stage": "queue"}
	p, err := loadProject(s.db, id)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := p.Plan.Digest()
	j, err := loadProjectJob(s.db, id)
	if err != nil || j.Preparation == nil {
		t.Fatal(err)
	}
	// Always retain the exact selected anchor's final state, including a failed
	// or uncertain Start. Cleanup stops only this fresh test's exact anchor ID.
	cleanup := func() {
		raw := docker("container", "inspect", j.Preparation.Anchor.ID)
		result["final_inspection_before_stop"] = raw
		var values []struct{ State struct{ Running bool } }
		if json.Unmarshal(raw, &values) != nil || len(values) != 1 {
			t.Error("cannot classify fresh anchor for cleanup")
			return
		}
		if values[0].State.Running {
			docker("container", "stop", "--timeout", "10", j.Preparation.Anchor.ID)
		}
		result["cleanup_inspection"] = docker("container", "inspect", j.Preparation.Anchor.ID)
		var stopped []struct {
			State struct {
				Running, OOMKilled bool
				ExitCode           int
			}
		}
		rawStop := result["cleanup_inspection"].(json.RawMessage)
		if json.Unmarshal(rawStop, &stopped) != nil || len(stopped) != 1 || stopped[0].State.Running || stopped[0].State.OOMKilled || values[0].State.Running && stopped[0].State.ExitCode != 143 {
			result["ok"] = false
			result["stage"] = "cleanup-failed"
			t.Error("fresh anchor did not stop cleanly")
			return
		}
		result["clean_anchor_stop"] = true

	}
	if registerCleanup == nil {
		defer cleanup()
	} else {
		registerCleanup(cleanup)
	}
	b := &vmAnchorBackend{store: s, id: id}
	child, cancel := context.WithCancel(ctx)
	r := &Server{s: s, b: b, ctx: child, cancel: cancel, workGate: make(chan struct{}, 1), uid: uint32(os.Geteuid())}
	defer func() { cancel(); r.wg.Wait() }()
	var cli *vmPublicRecoveryCLI
	if len(publicMode) > 0 && publicMode[0] {
		var openErr error
		cli, openErr = vmOpenPublicProjectCLI(ctx, r, p.ID, os.Getenv("OMABIT_TEST_PROJECT_VM"), "anchor")
		if openErr != nil {
			t.Fatal(openErr)
		}
		defer cli.close()
	}

	req := projectAnchorStartRequest{RequestID: "start-anchor-vm", ExpectedRevision: p.Revision, ReviewDigest: digest}

	queue := func(q projectAnchorStartRequest, replay bool) (protocol.Operation, bool, error) {
		if cli != nil {
			op, e := cli.operate("start-anchor", "", q.RequestID, q.ExpectedRevision, q.ReviewDigest, replay)
			return op, replay, e
		}
		return r.queueProjectAnchorStart("qualification", id, q)
	}
	if cli != nil {
		result["public_actions"] = cli.proof
	}
	op, hit, err := queue(req, false)
	if err != nil || hit {
		t.Error(op, hit, err)
		return result
	}
	result["operation_id"] = op.ID
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("anchor worker deadline")
		return result
	}
	receipt, err := loadOperation(s.db, op.ID)
	if err != nil {
		t.Error(err)
		return result
	}
	result["operation"] = receipt
	state, err := loadProjectAnchorStart(s.db, id)
	if err != nil {
		t.Error(err)
		return result
	}
	result["journal"] = state
	if b.wrapped != nil {
		result["calls"] = b.wrapped.calls
		result["observations"] = b.wrapped.observations
		result["inspection_errors"] = b.wrapped.inspectionErrors
	}
	result["anchor_inspection"] = docker("container", "inspect", j.Preparation.Anchor.ID)
	result["bridge_inspection"] = docker("network", "inspect", j.Preparation.Bridge.ID)
	result["volumes"] = strings.Fields(string(docker("volume", "ls", "--quiet")))
	if receipt.Status != "succeeded" || state.Phase != "observed" || state.Namespace == nil || b.wrapped == nil || len(b.wrapped.observations) != 2 || !reflect.DeepEqual(b.wrapped.calls, []string{"inspect-planned", "start-dispatched", "inspect-started"}) {
		t.Error("anchor Start did not complete verified journal")
		return result
	}
	replay, hit, err := queue(req, true)
	if err != nil || !hit || !reflect.DeepEqual(replay, receipt) {
		t.Error("Start replay changed receipt", err)
		return result
	}
	req.RequestID = "another-start-vm"
	_, _, err = queue(req, false)
	if err == nil {
		t.Error("new request bypassed retained Start")
		return result
	}
	if r.failed.Load() {
		t.Error("runner state publication failed")
		return result
	}
	result["repeat_dispatch_refused"] = true
	result["stage"] = "complete"
	result["ok"] = true
	if cli != nil {
		cli.proof["ok"] = true
	}
	return result
}
