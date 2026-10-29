//go:build linux && amd64

package managed

import (
	"context"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// The disposable guest owns every resource selected here. This follows complete
// replacement readiness and leaves the same pier, stopped peer and anchor intact.
func exerciseVMCurrentMemberStop(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, member map[string]any) {
	t.Helper()
	resumed, ok := member["resume"].(map[string]any)
	if !ok || resumed["ok"] != true {
		t.Error("replacement missing before Stop")
		member["ok"] = false
		return
	}
	ready, ok := resumed["readiness"].(map[string]any)
	if !ok || ready["ok"] != true {
		t.Error("replacement readiness missing before Stop")
		member["ok"] = false
		return
	}
	result := map[string]any{"ok": false, "stage": "queue"}
	resumed["stop"] = result
	fail := func(message string, err error) { t.Helper(); t.Error(message, err); member["ok"] = false }
	digest, err := p.Plan.Digest()
	if err != nil {
		fail("plan", err)
		return
	}
	instance := member["instance_id"].(string)
	req := projectMemberStopRequest{RequestID: "stop-" + ready["operation_id"].(string), ExpectedRevision: p.Revision, ReviewDigest: digest, ReadinessOperationID: ready["operation_id"].(string)}
	b.currentStop, b.currentStopErrors = nil, nil
	op, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, false)
	if err != nil || hit {
		fail("replacement Stop admission", err)
		return
	}
	result["operation_id"] = op.ID
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		fail("replacement Stop deadline", ctx.Err())
		return
	}
	result["setup_errors"] = append([]string(nil), b.currentStopErrors...)
	if b.currentStop != nil {
		result["calls"] = b.currentStop.calls
		result["adapter_errors"] = b.currentStop.errors
		result["stopped_peers"] = b.currentStop.peers
	}
	j, err := loadProjectMemberStop(r.s.db, op.ID)
	if err != nil {
		fail("load Stop", err)
		return
	}
	result["journal"] = j
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		fail("load receipt", err)
		return
	}
	result["operation"] = receipt
	wantStoppedPeers := 1
	if b.livePeer {
		wantStoppedPeers = 0
	}
	if receipt.Status != "succeeded" || !j.Stop.Clean() || b.currentStop == nil || len(b.currentStopErrors) != 0 || len(b.currentStop.errors) != 0 || !reflect.DeepEqual(b.currentStop.calls, []string{"inspect-planned", "signal-stop-dispatched", "await-stop-signalled"}) || len(b.currentStop.peers) != wantStoppedPeers {
		fail("replacement Stop incomplete", nil)
		return
	}
	again, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, true)
	if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
		fail("replacement Stop replay changed", err)
		return
	}
	result["replay_unchanged"] = true
	req.RequestID = "repeat-stop-" + ready["operation_id"].(string)
	if _, _, err = vmQueueMemberAction(r, b, p.ID, instance, req, false); err == nil {
		fail("replacement Stop could replay dispatch", nil)
		return
	}
	result["repeat_dispatch_refused"] = true
	oldStop := member["stop"].(map[string]any)
	original, err := loadProjectMemberStop(r.s.db, oldStop["operation_id"].(string))
	if err != nil || !reflect.DeepEqual(original, oldStop["journal"]) {
		fail("original Stop changed", err)
		return
	}
	resume, err := loadProjectMemberResume(r.s.db, resumed["operation_id"].(string))
	if err != nil || !reflect.DeepEqual(resume, resumed["journal"]) {
		fail("original resume changed", err)
		return
	}
	proof, err := loadProjectMemberResumedReadiness(r.s.db, ready["operation_id"].(string))
	if err != nil || !reflect.DeepEqual(proof, ready["journal"]) {
		fail("original readiness changed", err)
		return
	}
	result["original_history_unchanged"], result["stage"], result["ok"] = true, "complete", true
}
func (b *vmMemberLaunchBackend) OpenResumedProjectMemberStop(ctx context.Context, s runtimequalify.ProjectResumedMemberStopSelection) (projectnetwork.MemberStopDriver, error) {
	d, err := runtimequalify.OpenResumedProjectMemberStop(ctx, s)
	if err != nil {
		b.currentStopErrors = append(b.currentStopErrors, vmLaunchError(err))
		return nil, err
	}
	b.currentStop = &vmMemberStopDriver{store: b.store, driver: d, opID: s.Stop.OperationID, peers: s.StoppedPeers, plan: s.Project.Plan}
	return b.currentStop, nil
}
