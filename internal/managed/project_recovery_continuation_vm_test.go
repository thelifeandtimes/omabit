//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// This helper is reachable only after the common no-NIC VM admission guard.
// Faults lose replies after actual successful actions; none repeats an action.
func vmExerciseRecoveryContinuation(ctx context.Context, r *Server, b *vmAnchorRecoveryBackend, p storedProject, parentID string) (map[string]any, error) {
	result := map[string]any{"ok": false, "stage": "lost-signal", "store_reopens": 0}
	cuts := []map[string]any{}
	result["interruptions"] = cuts
	wait := func() error {
		done := make(chan struct{})
		go func() { r.wg.Wait(); close(done) }()
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	counts := func() map[string]int {
		out := map[string]int{}
		for _, e := range b.events {
			switch e.Action {
			case "drain-SignalDrainMember", "retirement-RemoveRetiredMember", "replacement-CreateReplacement", "replacement-StartReplacement":
				out[e.Action]++
			}
		}
		return out
	}
	reopen := func(stage, receiptID string) (projectAnchorRecoveryJob, error) {
		var zero projectAnchorRecoveryJob
		if err := wait(); err != nil {
			return zero, err
		}
		parent, err := loadProjectAnchorRecovery(r.s.db, parentID)
		if err != nil {
			return zero, err
		}
		receipt, err := loadOperation(r.s.db, receiptID)
		if err != nil {
			return zero, err
		}
		if receipt.Status != "interrupted" {
			return zero, fmt.Errorf("%s did not retain interruption", stage)
		}
		beforeEffects := counts()
		eventCount := len(b.events)
		// All private workers have exited. Keep the shared fixture pointer, replacing
		// only its fully closed/reopened store value (store contains no mutexes).
		old := r.s
		mode := old.mode
		serviceCycle := os.Getenv("OMABIT_TEST_PROJECT_VM") == vmServiceToken
		var checkpoint protocol.ProjectRecoveryCheckpoint
		var savedStatus protocol.ProjectStatus
		if serviceCycle {
			if err = vmServiceEmptyStore(old, false); err != nil {
				return zero, err
			}
			checkpoint, err = old.projectRecoveryCheckpoint(ctx, p.ID, parentID)
			if err != nil {
				return zero, err
			}
			savedStatus, err = old.projectStatus(ctx, p.ID)
			if err != nil {
				return zero, err
			}
		}
		if err = old.close(); err != nil {
			return zero, err
		}
		if serviceCycle {
			proof, cycleErr := vmServiceRecoveryCycle(ctx, old.directory, old.runnerID, p.ID, parentID, stage, checkpoint, savedStatus)
			cycles, _ := result["owner_service_cycles"].([]map[string]any)
			result["owner_service_cycles"] = append(cycles, proof)
			if cycleErr != nil {
				return zero, cycleErr
			}
		}
		opened, err := openStore(old.directory)
		if err != nil {
			return zero, err
		}
		if opened.runnerID != old.runnerID {
			opened.close()
			return zero, errors.New("restart changed runner identity")
		}
		opened.mode = mode
		*old = *opened
		if err = old.reconcileInterrupted(); err != nil {
			return zero, err
		}
		kept, err := loadProjectAnchorRecovery(old.db, parentID)
		if err != nil {
			return zero, err
		}
		actual, err := loadOperation(old.db, receiptID)
		if err != nil {
			return zero, err
		}
		if !reflect.DeepEqual(parent, kept) || !reflect.DeepEqual(receipt, actual) || eventCount != len(b.events) || !reflect.DeepEqual(beforeEffects, counts()) {
			return zero, errors.New("restart changed checkpoint or dispatched effects")
		}
		cuts = append(cuts, map[string]any{"stage": stage, "operation": receipt, "parent_operation_id": parentID, "phase": parent.Journal.Phase, "sequence": parent.Journal.Sequence, "checkpoint_digest": recoveryCheckpointDigest(parent.Journal), "effects": beforeEffects, "store_reopened_unchanged": true})
		result["interruptions"] = cuts
		result["store_reopens"] = len(cuts)
		return kept, nil
	}
	queue := func(kind, actor string, parent projectAnchorRecoveryJob, request string, replay bool) (protocol.Operation, bool, error) {
		req := observationRequest(p, parent, request)
		if b.publicCLI != nil {
			op, err := b.publicCLI.recover(kind, parent.OperationID, vmPublicRecoveryRequest(req), replay)
			return op, replay, err
		}
		if kind == "observe" {
			return r.queueProjectRecoveryObservation(actor, p.ID, req)
		}
		return r.queueProjectRecoveryContinuation(actor, p.ID, req)
	}
	observe := func(parent projectAnchorRecoveryJob, request string) error {
		before := counts()
		op, hit, err := queue("observe", "qualification-observer", parent, request, false)
		if err != nil {
			return err
		}
		if hit {
			return errors.New("new observation unexpectedly replayed")
		}
		if err = wait(); err != nil {
			return err
		}
		receipt, err := loadOperation(r.s.db, op.ID)
		if err != nil {
			return err
		}
		if receipt.Status != "succeeded" || !reflect.DeepEqual(before, counts()) {
			return errors.New("observation failed or dispatched runtime action")
		}
		replay, hit, err := queue("observe", "qualification-observer", parent, request, true)
		if err != nil {
			return err
		}
		if !hit || !reflect.DeepEqual(replay, receipt) {
			return errors.New("observation replay changed")
		}
		result[request] = receipt
		return nil
	}
	continuation := func(parent projectAnchorRecoveryJob, request string) (string, error) {
		op, hit, err := queue("continue", "qualification-continuation", parent, request, false)
		if err != nil {
			return "", err
		}
		if hit {
			return "", errors.New("new continuation unexpectedly replayed")
		}
		if err = wait(); err != nil {
			return "", err
		}
		receipt, err := loadOperation(r.s.db, op.ID)
		if err != nil {
			return "", err
		}
		before := counts()
		replay, hit, err := queue("continue", "qualification-continuation", parent, request, true)
		if err != nil {
			return "", err
		}
		if !hit || !reflect.DeepEqual(replay, receipt) || !reflect.DeepEqual(before, counts()) {
			return "", errors.New("continuation replay changed or dispatched effects")
		}
		result[request] = receipt
		return op.ID, nil
	}
	parent, err := reopen("lost-signal", parentID)
	if err != nil {
		return result, err
	}
	if parent.Journal.Drain == nil || parent.Journal.Drain.Members[0].Phase != "stop-dispatched" {
		return result, errors.New("lost signal checkpoint missing")
	}
	// Wait only for the already-signalled exact member. This read-only wait does
	// not manufacture an acknowledgement or grant the next recovery action.
	d, err := b.OpenProjectAnchorDrain(ctx, runtimequalify.ProjectAnchorDrainSelection{Source: recoveryRuntimeSource(r.s, parent), Drain: parent.Journal.Drain.Clone()})
	if err != nil {
		return result, err
	}
	if err = vmWaitInterruptedDrainExit(ctx, d, *parent.Journal.Drain); err != nil {
		return result, err
	}
	if err = observe(parent, "observe-lost-signal"); err != nil {
		return result, err
	}
	parent, err = loadProjectAnchorRecovery(r.s.db, parentID)
	if err != nil {
		return result, err
	}
	first, err := continuation(parent, "continue-after-signal")
	if err != nil {
		return result, err
	}
	result["stage"] = "created-id"
	parent, err = reopen("created-id", first)
	if err != nil {
		return result, err
	}
	if parent.Journal.Replacement == nil || parent.Journal.Replacement.Members[0].Phase != "created" || parent.Journal.Replacement.Members[0].Container.ID == "" {
		return result, errors.New("known Create reply was not retained")
	}
	second, err := continuation(parent, "continue-created-id")
	if err != nil {
		return result, err
	}
	result["stage"] = "lost-start"
	parent, err = reopen("lost-start", second)
	if err != nil {
		return result, err
	}
	if parent.Journal.Replacement == nil || parent.Journal.Replacement.Members[0].Phase != "start-dispatched" {
		return result, errors.New("lost Start checkpoint missing")
	}
	if err = observe(parent, "observe-lost-start"); err != nil {
		return result, err
	}
	parent, err = loadProjectAnchorRecovery(r.s.db, parentID)
	if err != nil {
		return result, err
	}
	last, err := continuation(parent, "continue-after-start")
	if err != nil {
		return result, err
	}
	done, err := loadOperation(r.s.db, last)
	if err != nil {
		return result, err
	}
	if done.Status != "succeeded" {
		return result, errors.New("final continuation did not complete")
	}
	for _, effect := range []string{"drain-SignalDrainMember", "retirement-RemoveRetiredMember", "replacement-CreateReplacement", "replacement-StartReplacement"} {
		if counts()[effect] != 2 {
			return result, fmt.Errorf("effect repeated or missing: %s", effect)
		}
	}
	if b.loseSignal || b.loseCreate || b.loseStart {
		return result, errors.New("fault injection was not exercised")
	}
	if err = validateProjectRecoveryObservations(r.s.db); err != nil {
		return result, err
	}
	if err = validateProjectRecoveryContinuations(r.s.db); err != nil {
		return result, err
	}
	result["effects"] = counts()
	result["stage"], result["ok"] = "complete", true
	return result, nil
}

func (b *vmAnchorRecoveryBackend) OpenProjectRecoveryObservation(ctx context.Context, s projectRecoveryObservationSelection) (out projectnetwork.AnchorRecoveryObservers, err error) {
	start := time.Now()
	defer func() { b.record("open-observation", start, err) }()
	return (realBackend{}).OpenProjectRecoveryObservation(ctx, s)
}
func (b *vmAnchorRecoveryBackend) OpenProjectCreatedAnchorReplacement(ctx context.Context, s runtimequalify.ProjectAnchorReplacementSelection) (out projectnetwork.AnchorReplacementStartDriver, err error) {
	start := time.Now()
	defer func() { b.record("open-created-replacement", start, err) }()
	d, err := runtimequalify.OpenProjectCreatedAnchorReplacement(ctx, s)
	if err != nil {
		return nil, err
	}
	return &vmRecoveryCreatedStartDriver{backend: b, driver: d}, nil
}

type vmRecoveryCreatedStartDriver struct {
	backend *vmAnchorRecoveryBackend
	driver  projectnetwork.AnchorReplacementStartDriver
}

func (d *vmRecoveryCreatedStartDriver) VerifyEngine(ctx context.Context, id string) error {
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmRecoveryCreatedStartDriver) InspectReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int, stage string) (projectnetwork.AnchorRetirementObservation, error) {
	return d.driver.InspectReplacement(ctx, s, i, stage)
}
func (d *vmRecoveryCreatedStartDriver) StartReplacement(ctx context.Context, s projectnetwork.AnchorReplacement, i int) (err error) {
	start := time.Now()
	defer func() { d.backend.record("replacement-StartReplacement", start, err) }()
	if err = d.backend.check("replacement", s); err != nil {
		return err
	}
	err = d.driver.StartReplacement(ctx, s, i)
	if err == nil && d.backend.loseStart {
		d.backend.loseStart = false
		return errors.New("fixture lost successful Start reply")
	}
	return err
}

// A reopened observer has no in-memory acknowledged dispatch. AwaitDrainExit is
// only for the original driver after a successful signal reply. Poll the
// observation-only method for the retained uncertain intent instead; the durable
// recovery observation below remains responsible for advancing its checkpoint.
func vmWaitInterruptedDrainExit(parent context.Context, d interface {
	InspectDrain(context.Context, projectnetwork.AnchorDrain) (projectnetwork.AnchorDrainObservation, error)
}, saved projectnetwork.AnchorDrain) error {
	ctx, cancel := context.WithTimeout(parent, 300*time.Second)
	defer cancel()
	var last error
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for uncertain drain exit: %w (last observation: %v)", err, last)
		}
		if _, last = d.InspectDrain(ctx, saved.Clone()); last == nil {
			return nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
