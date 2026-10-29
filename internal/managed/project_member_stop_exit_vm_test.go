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

// The disposable-guest fixture deliberately withholds one successful signal
// reply. Only the separate read-only operation may establish its current exit;
// the original Stop remains interrupted at its committed dispatch boundary.
func exerciseVMMemberStopExit(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, instance string, stopped map[string]any) bool {
	t.Helper()
	original := stopped["journal"].(projectMemberStopJob)
	originalReceipt, err := loadOperation(r.s.db, original.OperationID)
	if err != nil {
		t.Error(err)
		return false
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return false
	}
	result := map[string]any{"ok": false, "stage": "queue"}
	stopped["exit_observation"] = result
	b.stopExit, b.stopExitErrors = nil, nil
	req := projectMemberStopExitRequest{RequestID: "exit-" + original.OperationID, StopOperationID: original.OperationID, ExpectedRevision: p.Revision, ReviewDigest: digest}
	op, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, false)
	if err != nil || hit {
		t.Error(op, hit, err)
		return false
	}
	result["operation_id"] = op.ID
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("Stop exit observation deadline")
		return false
	}
	result["setup_errors"] = append([]string(nil), b.stopExitErrors...)
	if b.stopExit != nil {
		result["calls"] = b.stopExit.calls
		result["adapter_errors"] = b.stopExit.errors
		result["stopped_peers"] = b.stopExit.peers
	}
	j, err := loadProjectMemberStopExit(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return false
	}
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		return false
	}
	result["journal"], result["operation"] = j, receipt
	if receipt.Status != "succeeded" || j.Evidence == nil || !j.Evidence.Clean() || b.stopExit == nil || len(b.stopExit.errors) != 0 || len(b.stopExitErrors) != 0 || !reflect.DeepEqual(b.stopExit.calls, []string{"inspect-observing"}) {
		t.Error("retained Stop exit observation did not complete")
		return false
	}
	retained, err := loadProjectMemberStop(r.s.db, original.OperationID)
	if err != nil || !reflect.DeepEqual(retained, original) {
		t.Error("exit observation rewrote original Stop", err)
		return false
	}
	retainedReceipt, err := loadOperation(r.s.db, original.OperationID)
	if err != nil || !reflect.DeepEqual(retainedReceipt, originalReceipt) || retainedReceipt.Status != originalReceipt.Status {
		t.Error("exit observation acknowledged original Stop", err)
		return false
	}
	result["original_stop_unchanged"] = true
	again, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, true)
	if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
		t.Error("exit observation replay changed receipt", err)
		return false
	}
	result["replay_unchanged"], result["stage"], result["ok"] = true, "complete", true
	return true
}

func (b *vmMemberLaunchBackend) OpenProjectMemberStopExit(ctx context.Context, selection runtimequalify.ProjectMemberStopSelection) (projectnetwork.MemberStopExitDriver, error) {
	d, err := runtimequalify.OpenProjectMemberStopExit(ctx, selection)
	if err != nil {
		b.stopExitErrors = append(b.stopExitErrors, vmLaunchError(err))
		return nil, err
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_stop_exits j JOIN operations o ON j.operation_id=o.id WHERE j.stop_operation_id=? AND json_extract(o.body,'$.status')='queued'", selection.Stop.OperationID).Scan(&id); err != nil {
		return nil, err
	}
	b.stopExit = &vmMemberStopExitDriver{store: b.store, driver: d, opID: id, original: selection.Stop, peers: selection.StoppedPeers}
	return b.stopExit, nil
}

type vmMemberStopExitDriver struct {
	store         *store
	driver        projectnetwork.MemberStopExitDriver
	opID          string
	original      projectnetwork.MemberStop
	peers         []projectnetwork.MemberStop
	calls, errors []string
}

func (d *vmMemberStopExitDriver) check(s *projectnetwork.MemberStop) error {
	j, err := loadProjectMemberStopExit(d.store.db, d.opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Phase != "observing" || !reflect.DeepEqual(j.Stop, d.original) || s != nil && !reflect.DeepEqual(*s, d.original) {
		return errors.New("Stop exit read preceded committed observation")
	}
	return validateProjectMemberStopExit(d.store.db)
}
func (d *vmMemberStopExitDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberStopExitDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(nil); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberStopExitDriver) InspectExit(ctx context.Context, s projectnetwork.MemberStop) (projectnetwork.MemberStopObservation, error) {
	if err := d.check(&s); err != nil {
		return projectnetwork.MemberStopObservation{}, d.retain(err)
	}
	d.calls = append(d.calls, "inspect-observing")
	o, err := d.driver.InspectExit(ctx, s)
	return o, d.retain(err)
}
