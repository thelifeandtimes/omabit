//go:build linux && amd64

package managed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Runs only inside the admitted disposable guest after the selected target's
// clean exit observation. Other current members may remain running.
func exerciseVMMemberRetirement(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage) {
	exerciseVMMemberRetirementFor(t, ctx, r, b, p, members, docker, "vm-zod")
}
func exerciseVMMemberRetirementFor(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage, target string) {
	t.Helper()
	var member map[string]any
	for _, m := range members {
		if m["instance_id"] == target {
			member = m
		}
	}
	if member == nil {
		t.Error("retirement target missing")
		return
	}
	stop, ok := member["stop"].(map[string]any)
	if !ok || stop["ok"] != true {
		t.Error("retirement requires completed Stop experiment")
		member["ok"] = false
		return
	}
	exit, ok := stop["exit_observation"].(map[string]any)
	if !ok || exit["ok"] != true {
		t.Error("retirement requires clean exit observation")
		member["ok"] = false
		return
	}
	original := exit["journal"].(projectMemberStopExitJob)
	result := map[string]any{"ok": false, "stage": "queue"}
	member["retirement"] = result
	before, err := vmRetirementState(original.Stop.Storage)
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	result["state_before"] = before
	id := ""
	for _, m := range original.Stop.Binding.Members {
		if m.InstanceID == original.InstanceID {
			id = m.Container.ID
		}
	}
	if id == "" {
		t.Error("retirement target identity missing")
		member["ok"] = false
		return
	}
	result["container_before"] = docker("container", "inspect", id)
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	b.retirement, b.retirementErrors = nil, nil
	req := projectMemberRetirementRequest{RequestID: "retire-" + original.OperationID, ExpectedRevision: p.Revision, ReviewDigest: digest, ExitOperationID: original.OperationID}
	op, hit, err := vmQueueMemberAction(r, b, p.ID, target, req, false)
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
		t.Error("retirement worker deadline")
		member["ok"] = false
		return
	}
	result["setup_errors"] = append([]string(nil), b.retirementErrors...)
	if b.retirement != nil {
		result["calls"] = b.retirement.calls
		result["adapter_errors"] = b.retirement.errors
		result["stopped_peers"] = b.retirement.peers
	}
	j, err := loadProjectMemberRetirement(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	receipt, err := loadOperation(r.s.db, op.ID)
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	result["journal"], result["operation"] = j, receipt
	if receipt.Status != "succeeded" || j.Retirement.Phase != "absent" || b.retirement == nil || len(b.retirement.errors) != 0 || len(b.retirementErrors) != 0 || !reflect.DeepEqual(b.retirement.calls, []string{"inspect-planned", "remove-dispatched", "inspect-removed"}) {
		t.Error("retirement incomplete")
		member["ok"] = false
		return
	}
	after, err := vmRetirementState(original.Stop.Storage)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Error("retirement changed stopped state", err)
		member["ok"] = false
		return
	}
	result["state_after"] = after
	retained, err := loadProjectMemberStopExit(r.s.db, original.OperationID)
	if err != nil || !reflect.DeepEqual(retained, original) {
		t.Error("retirement rewrote exit evidence", err)
		member["ok"] = false
		return
	}
	retainedStop, err := loadOperation(r.s.db, original.StopOperationID)
	if err != nil || !reflect.DeepEqual(retainedStop, stop["operation"]) {
		t.Error("retirement rewrote original Stop receipt", err)
		member["ok"] = false
		return
	}
	again, hit, err := vmQueueMemberAction(r, b, p.ID, target, req, true)
	if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
		t.Error("retirement replay changed receipt", err)
		member["ok"] = false
		return
	}
	req.RequestID = "repeat-retire-" + original.OperationID
	if _, _, err = vmQueueMemberAction(r, b, p.ID, target, req, false); err == nil {
		t.Error("retirement could redispatch")
		member["ok"] = false
		return
	}
	result["original_evidence_unchanged"], result["replay_unchanged"], result["repeat_dispatch_refused"], result["stage"], result["ok"] = true, true, true, "complete", true
}

type vmRetirementFile struct {
	Bytes  int64  `json:"bytes"`
	Mode   uint32 `json:"mode"`
	SHA256 string `json:"sha256"`
}

func vmRetirementState(loc pierstore.Location) (map[string]vmRetirementFile, error) {
	root, err := pierstore.Check("/home/lab", loc)
	if err != nil {
		return nil, err
	}
	result := map[string]vmRetirementFile{}
	for _, dir := range []string{".urb/log", ".urb/chk"} {
		err = filepath.WalkDir(filepath.Join(root, dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || len(result) >= 8192 {
				return errors.New("unsupported private fixture state entry")
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			h := sha256.New()
			n, err := io.Copy(h, f)
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
			if n != info.Size() {
				return errors.New("state file changed during hashing")
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			result[rel] = vmRetirementFile{Bytes: n, Mode: uint32(info.Mode().Perm()), SHA256: hex.EncodeToString(h.Sum(nil))}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(result) == 0 {
		return nil, errors.New("retirement fixture has no persisted state")
	}
	return result, nil
}
func (b *vmMemberLaunchBackend) OpenProjectMemberRetirement(ctx context.Context, s runtimequalify.ProjectMemberRetirementSelection) (projectnetwork.MemberRetirement, projectnetwork.MemberRetirementDriver, error) {
	initial, d, err := runtimequalify.OpenProjectMemberRetirement(ctx, s)
	if err != nil {
		b.retirementErrors = append(b.retirementErrors, vmLaunchError(err))
		return initial, nil, err
	}
	b.retirement = &vmMemberRetirementDriver{store: b.store, driver: d, opID: s.OperationID, peers: s.Stop.StoppedPeers}
	return initial, b.retirement, nil
}

type vmMemberRetirementDriver struct {
	store         *store
	driver        projectnetwork.MemberRetirementDriver
	opID          string
	peers         []projectnetwork.MemberStop
	calls, errors []string
}

func (d *vmMemberRetirementDriver) check(s *projectnetwork.MemberRetirement, phase, action string) error {
	j, err := loadProjectMemberRetirement(d.store.db, d.opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if op.Status != "running" || s != nil && (j.Retirement.Phase != phase || !reflect.DeepEqual(j.Retirement, *s)) {
		return errors.New("retirement effect preceded committed journal")
	}
	if err = validateProjectMemberRetirements(d.store.db); err != nil {
		return err
	}
	if action != "" {
		d.calls = append(d.calls, action)
	}
	return nil
}
func (d *vmMemberRetirementDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberRetirementDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(nil, "", ""); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberRetirementDriver) InspectStopped(ctx context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberStopObservation, error) {
	if err := d.check(&s, "planned", "inspect-planned"); err != nil {
		return projectnetwork.MemberStopObservation{}, d.retain(err)
	}
	o, err := d.driver.InspectStopped(ctx, s)
	return o, d.retain(err)
}
func (d *vmMemberRetirementDriver) RemoveStopped(ctx context.Context, s projectnetwork.MemberRetirement) error {
	if err := d.check(&s, "remove-dispatched", "remove-dispatched"); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.RemoveStopped(ctx, s))
}
func (d *vmMemberRetirementDriver) InspectAbsent(ctx context.Context, s projectnetwork.MemberRetirement) (projectnetwork.MemberAbsenceObservation, error) {
	if err := d.check(&s, "removed", "inspect-removed"); err != nil {
		return projectnetwork.MemberAbsenceObservation{}, d.retain(err)
	}
	o, err := d.driver.InspectAbsent(ctx, s)
	return o, d.retain(err)
}
