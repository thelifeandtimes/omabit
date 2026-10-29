//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
	"reflect"
	"testing"
)

func TestProjectContextLifecycleDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, true, false)
}

// Both members belong to this fresh, guarded guest. Replace the original sender
// only after its peer has reached generation3; preserve that receiver throughout.
func exerciseVMContextPeerCycle(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage, registerCleanup func(func())) {
	t.Helper()
	var sender, receiver map[string]any
	for _, m := range members {
		if m["instance_id"] == "vm-nec" {
			sender = m
		}
		if m["instance_id"] == "vm-zod" {
			receiver = m
		}
	}
	if sender == nil || receiver == nil {
		t.Error("context lifecycle members missing")
		return
	}
	cycles, ok := receiver["later_generations"].([]map[string]any)
	if !ok || len(cycles) != 1 {
		t.Error("context lifecycle receiver history missing")
		return
	}
	latest := cycles[0]["resume"].(map[string]any)
	rows := latest["current_context_readiness"].([]map[string]any)
	var ready map[string]any
	for _, row := range rows {
		if row["instance_id"] == "vm-nec" {
			ready = row
		}
	}
	if ready == nil || ready["ok"] != true {
		t.Error("sender current context missing")
		return
	}
	vmApplicationPhase(t, ctx, b, "before-peer")
	if t.Failed() {
		return
	}
	receiverID := latest["journal"].(projectMemberResumeJob).Resume.Container.ID
	receiverBefore := docker("container", "inspect", receiverID)
	sender["receiver_before_peer_cycle"] = receiverBefore
	checkReceiver := func() error {
		raw := docker("container", "inspect", receiverID)
		sender["receiver_after_peer_cycle"] = raw
		return vmCheckUnchangedLivePeer(receiverBefore, raw)
	}
	stopped := exerciseVMContextStop(t, ctx, r, b, p, "vm-nec", ready, 0)
	sender["stop"] = stopped
	sender["fixture_cleanup_only"] = false
	if t.Failed() {
		return
	}
	if !exerciseVMMemberStopExit(t, ctx, r, b, p, "vm-nec", stopped) {
		return
	}
	exerciseVMMemberRetirementFor(t, ctx, r, b, p, []map[string]any{sender}, docker, "vm-nec")
	if t.Failed() {
		return
	}
	exerciseVMMemberResumeFor(t, ctx, r, b, p, []map[string]any{sender}, docker, registerCleanup, "vm-nec", false)
	if t.Failed() {
		return
	}
	if err := checkReceiver(); err != nil {
		t.Error(err)
		return
	}
	b.checkLivePeer = checkReceiver
	vmApplicationPhase(t, ctx, b, "after-peer")
	if t.Failed() {
		return
	}
	resumed := sender["resume"].(map[string]any)
	exerciseVMCurrentContextReadiness(t, ctx, r, b, p, resumed, resumed["operation_id"].(string))
	if t.Failed() {
		return
	}
	rows = resumed["current_context_readiness"].([]map[string]any)
	ready = nil
	for _, row := range rows {
		if row["instance_id"] == "vm-zod" {
			ready = row
		}
	}
	if ready == nil {
		t.Error("receiver refreshed context missing")
		return
	}
	if b.anchorReview {
		// This describes the selected cleanup path even when fault review fails.
		// Reopen must never assume that an unrequested final Stop was queued.
		latest["anchor_review_cleanup_only"] = true
		resumed["anchor_review_cleanup_only"] = true
		if b.anchorRecovery {
			vmApplicationPhase(t, ctx, b, "before-anchor")
			if t.Failed() {
				return
			}
		}
		exerciseVMAnchorRecoveryReview(t, ctx, r, p, members, resumed, sender, docker)
		if b.anchorRecovery && !t.Failed() {
			exerciseVMAnchorRecovery(t, ctx, r, b, p, latest, resumed, sender, docker, registerCleanup)
		}
		return
	}
	latest["stop"] = exerciseVMContextStop(t, ctx, r, b, p, "vm-zod", ready, 0)
	if t.Failed() {
		return
	}
	// The sender's replacement readiness has the same latest membership. Its
	// final Stop records the already stopped current receiver as an exact peer.
	resumed["stop"] = exerciseVMContextStop(t, ctx, r, b, p, "vm-nec", resumed["readiness"].(map[string]any), 1)
	if t.Failed() {
		return
	}
	sender["current_context_lifecycle_complete"] = true
}

func exerciseVMContextStop(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, instance string, ready map[string]any, stoppedPeers int) map[string]any {
	t.Helper()
	result := map[string]any{"ok": false, "stage": "queue"}
	history := currentStopHistory(t, r.s)
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return result
	}
	id := ready["operation_id"].(string)
	req := projectMemberStopRequest{RequestID: "context-stop-" + id, ExpectedRevision: p.Revision, ReviewDigest: digest, ReadinessOperationID: id}
	b.currentStop, b.currentStopErrors = nil, nil
	op, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, false)
	if err != nil || hit {
		t.Error("context Stop admission", err)
		return result
	}
	result["operation_id"] = op.ID
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("context Stop deadline")
		return result
	}
	result["setup_errors"] = append([]string(nil), b.currentStopErrors...)
	if b.currentStop != nil {
		result["calls"] = b.currentStop.calls
		result["adapter_errors"] = b.currentStop.errors
		result["stopped_peers"] = b.currentStop.peers
	}
	j, err := loadProjectMemberStop(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return result
	}
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return result
	}
	result["journal"], result["operation"] = j, receipt
	if receipt.Status != "succeeded" || !j.Stop.Clean() || b.currentStop == nil || len(b.currentStopErrors) != 0 || len(b.currentStop.errors) != 0 || len(b.currentStop.peers) != stoppedPeers || !reflect.DeepEqual(b.currentStop.calls, []string{"inspect-planned", "signal-stop-dispatched", "await-stop-signalled"}) {
		t.Error("context Stop incomplete")
		return result
	}
	again, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, true)
	if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
		t.Error("context Stop replay changed", err)
		return result
	}
	req.RequestID = "repeat-" + req.RequestID
	if _, _, err = vmQueueMemberAction(r, b, p.ID, instance, req, false); err == nil {
		t.Error("context Stop repeated dispatch")
		return result
	}
	assertContextHistory(t, r.s, history)
	result["replay_unchanged"], result["repeat_dispatch_refused"], result["original_history_unchanged"], result["stage"], result["ok"] = true, true, true, "complete", true
	return result
}

func (b *vmMemberLaunchBackend) OpenCurrentProjectMemberStop(ctx context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	d, err := runtimequalify.OpenCurrentProjectMemberStop(ctx, s)
	if err != nil {
		b.currentStopErrors = append(b.currentStopErrors, vmLaunchError(err))
		return nil, err
	}
	b.currentStop = &vmMemberStopDriver{store: b.store, driver: d, opID: s.Stop.OperationID, peers: s.StoppedPeers, plan: s.ReadinessSource.Project.Plan}
	return b.currentStop, nil
}
func (b *vmMemberLaunchBackend) OpenCurrentProjectMemberStopExit(ctx context.Context, s runtimequalify.ProjectCurrentMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	d, err := runtimequalify.OpenCurrentProjectMemberStopExit(ctx, s)
	if err != nil {
		b.stopExitErrors = append(b.stopExitErrors, vmLaunchError(err))
		return nil, err
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_stop_exits j JOIN operations o ON j.operation_id=o.id WHERE j.stop_operation_id=? AND json_extract(o.body,'$.status')='queued'", s.Stop.OperationID).Scan(&id); err != nil {
		return nil, err
	}
	b.stopExit = &vmMemberStopExitDriver{store: b.store, driver: d, opID: id, original: s.Stop, peers: s.StoppedPeers}
	return b.stopExit, nil
}
func (b *vmMemberLaunchBackend) OpenCurrentProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectCurrentMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	initial, d, err := runtimequalify.OpenCurrentProjectMemberRetirement(ctx, s)
	if err != nil {
		b.retirementErrors = append(b.retirementErrors, vmLaunchError(err))
		return initial, nil, err
	}
	b.retirement = &vmMemberRetirementDriver{store: b.store, driver: d, opID: s.OperationID, peers: s.Stop.StoppedPeers}
	return initial, b.retirement, nil
}
