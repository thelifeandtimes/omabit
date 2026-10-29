//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Reached only after the common disposable-guest guard, both launches and both
// readiness receipts. Stop the peer first, then zod, retaining every resource.
func exerciseVMMemberStop(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, results []map[string]any) {
	t.Helper()
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return
	}
	for index := len(results) - 1; index >= 0; index-- {
		member := results[index]
		if b.livePeer && member["instance_id"] == "vm-nec" {
			continue
		}
		ready, ok := member["readiness"].(map[string]any)
		if !ok || ready["ok"] != true {
			t.Error("Stop requires complete readiness")
			member["ok"] = false
			return
		}
		instance := member["instance_id"].(string)
		result := map[string]any{"ok": false, "stage": "queue"}
		member["stop"] = result
		b.stop, b.stopErrors = nil, nil
		b.loseStopReply = index == 0 // Lose only the target reply in either peer mode.
		req := projectMemberStopRequest{RequestID: "stop-" + instance, ExpectedRevision: p.Revision, ReviewDigest: digest, ReadinessOperationID: ready["operation_id"].(string)}
		op, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, false)
		if err != nil || hit {
			t.Error(op, hit, err)
			member["ok"] = false
			return
		}
		result["operation_id"] = op.ID
		done := make(chan struct{})
		go func() { r.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			t.Error("Stop worker deadline")
			member["ok"] = false
			return
		}
		result["setup_errors"] = append([]string(nil), b.stopErrors...)
		if b.stop != nil {
			result["calls"] = b.stop.calls
			result["adapter_errors"] = b.stop.errors
			result["stopped_peers"] = b.stop.peers
			result["reply_discarded"] = b.stop.lostReplyObserved
			if b.stop.hiddenExit != nil {
				result["fixture_hidden_exit"] = *b.stop.hiddenExit
			}
		}
		j, err := loadProjectMemberStop(r.s.db, op.ID)
		if err != nil {
			t.Error(err)
			member["ok"] = false
			return
		}
		result["journal"] = j
		receipt, err := loadOperation(r.s.db, op.ID)
		if err != nil {
			t.Error(err)
			member["ok"] = false
			return
		}
		result["operation"] = receipt
		if b.stop == nil || len(b.stop.errors) != 0 || len(b.stopErrors) != 0 {
			t.Error("Stop adapter failed")
			member["ok"] = false
			return
		}
		expectedCalls := []string{"inspect-planned", "signal-stop-dispatched", "await-stop-signalled"}
		if b.loseStopReply {
			expectedCalls = expectedCalls[:2]
			if receipt.Status != "interrupted" || j.Stop.Phase != "stop-dispatched" || j.Stop.Before == nil || j.Stop.After != nil || !b.stop.lostReplyObserved || b.stop.hiddenExit == nil {
				t.Error("lost reply did not retain dispatch uncertainty")
				member["ok"] = false
				return
			}
			if !exerciseVMMemberStopExit(t, ctx, r, b, p, instance, result) {
				member["ok"] = false
				return
			}
		} else if receipt.Status != "succeeded" || !j.Stop.Clean() {
			t.Error("Stop did not complete", receipt.Status)
			member["ok"] = false
			return
		}
		if !reflect.DeepEqual(b.stop.calls, expectedCalls) {
			t.Error("unexpected Stop callback sequence")
			member["ok"] = false
			return
		}
		wantStoppedPeers := len(results) - index - 1
		if b.livePeer {
			wantStoppedPeers = 0
		}
		if len(b.stop.peers) != wantStoppedPeers {
			t.Error("Stop lost earlier peer exit")
			member["ok"] = false
			return
		}
		again, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, true)
		if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
			t.Error("Stop replay changed receipt", err)
			member["ok"] = false
			return
		}
		result["replay_unchanged"] = true
		req.RequestID = "another-stop-" + instance
		if _, _, err = vmQueueMemberAction(r, b, p.ID, instance, req, false); err == nil {
			t.Error("new request could re-signal same container")
			member["ok"] = false
			return
		}
		result["repeat_dispatch_refused"], result["stage"], result["ok"] = true, "complete", true
	}
}
func (b *vmMemberLaunchBackend) OpenProjectMemberStop(ctx context.Context, selection runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	d, err := runtimequalify.OpenProjectMemberStop(ctx, selection)
	if err != nil {
		b.stopErrors = append(b.stopErrors, vmLaunchError(err))
		return nil, err
	}
	b.stop = &vmMemberStopDriver{store: b.store, driver: d, opID: selection.Stop.OperationID, peers: selection.StoppedPeers, plan: selection.Project.Plan, loseReply: b.loseStopReply}
	return b.stop, nil
}

type vmMemberStopDriver struct {
	store                        *store
	driver                       projectnetwork.MemberStopDriver
	opID                         string
	peers                        []projectnetwork.MemberStop
	calls, errors                []string
	plan                         projectnetwork.Plan
	loseReply, lostReplyObserved bool
	hiddenExit                   *projectnetwork.MemberStopObservation
}

func (d *vmMemberStopDriver) check(s *projectnetwork.MemberStop, phase, action string) error {
	j, err := loadProjectMemberStop(d.store.db, d.opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if op.Status != "running" || s != nil && (j.Stop.Phase != phase || !reflect.DeepEqual(j.Stop, *s)) {
		return errors.New("Stop effect preceded its committed journal")
	}
	if err = validateProjectMemberStops(d.store.db); err != nil {
		return err
	}
	if action != "" {
		d.calls = append(d.calls, action)
	}
	return nil
}
func (d *vmMemberStopDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberStopDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(nil, "", ""); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberStopDriver) Inspect(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	if err := d.check(&s, "planned", "inspect-planned"); err != nil {
		return projectnetwork.MemberStopObservation{}, d.retain(err)
	}
	o, err := d.driver.Inspect(ctx, s)
	return o, d.retain(err)
}
func (d *vmMemberStopDriver) SignalStop(ctx context.Context, s projectnetwork.MemberStop) error {
	if err := d.check(&s, "stop-dispatched", "signal-stop-dispatched"); err != nil {
		return d.retain(err)
	}
	if err := d.driver.SignalStop(ctx, s); err != nil {
		return d.retain(err)
	}
	if !d.loseReply {
		return nil
	}
	// Fixture-only read: wait for the actual exit, then deliberately discard the
	// successful signal reply. This pure in-memory phase is never committed.
	acknowledged, err := s.Advance(d.plan, "stop-dispatched", "stop-signalled", nil)
	if err != nil {
		return d.retain(err)
	}
	observed, err := d.driver.AwaitStopped(ctx, acknowledged)
	if err != nil {
		return d.retain(err)
	}
	d.hiddenExit, d.lostReplyObserved = &observed, true
	return errors.New("fixture deliberately discarded successful Stop reply")
}
func (d *vmMemberStopDriver) AwaitStopped(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	if err := d.check(&s, "stop-signalled", "await-stop-signalled"); err != nil {
		return projectnetwork.MemberStopObservation{}, d.retain(err)
	}
	o, err := d.driver.AwaitStopped(ctx, s)
	return o, d.retain(err)
}
