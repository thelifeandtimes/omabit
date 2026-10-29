//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/pierstore"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Called only by the guarded disposable-VM member-launch test after its pair is
// responsive. The adapter cannot launch, stop, adopt or repair any resource.
func exerciseVMMemberReadiness(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, results []map[string]any) {
	t.Helper()
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return
	}
	for _, member := range results {
		instance := member["instance_id"].(string)
		result := map[string]any{"ok": false, "stage": "queue"}
		member["readiness"] = result
		b.readiness = nil
		b.readinessErrors = nil
		req := projectMemberReadinessRequest{RequestID: "observe-" + instance, ExpectedRevision: p.Revision, ReviewDigest: digest}
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
			t.Error("readiness worker deadline")
			member["ok"] = false
			return
		}
		result["setup_errors"] = append([]string(nil), b.readinessErrors...)
		if b.readiness != nil {
			result["calls"] = b.readiness.calls
			result["adapter_errors"] = b.readiness.errors
		}
		j, err := loadProjectMemberReadiness(r.s.db, op.ID)
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
		if receipt.Status != "succeeded" || j.Phase != "observed" || j.Evidence == nil || b.readiness == nil || len(b.readiness.errors) != 0 {
			t.Error("readiness did not complete", receipt.Status)
			member["ok"] = false
			return
		}
		if !reflect.DeepEqual(b.readiness.calls, []string{"project", "pier", "identity", "pier", "project"}) {
			t.Error("unexpected readiness observation sequence")
			member["ok"] = false
			return
		}
		again, hit, err := vmQueueMemberAction(r, b, p.ID, instance, req, true)
		if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
			t.Error("readiness replay changed receipt", err)
			member["ok"] = false
			return
		}
		result["replay_unchanged"] = true
		result["stage"] = "complete"
		result["ok"] = true
	}
}
func (b *vmMemberLaunchBackend) OpenInitialProjectReadiness(ctx context.Context, selection runtimequalify.ProjectInitialReadinessSelection) (projectnetwork.InitialReadinessDriver, error) {
	d, err := runtimequalify.OpenInitialProjectReadiness(ctx, selection)
	if err != nil {
		b.readinessErrors = append(b.readinessErrors, vmLaunchError(err))
		return nil, err
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_readiness j JOIN operations o ON j.operation_id=o.id WHERE j.project_id=? AND json_extract(o.body,'$.status')='queued'", selection.Project.ProjectID).Scan(&id); err != nil {
		return nil, err
	}
	b.readiness = &vmMemberReadinessDriver{store: b.store, driver: d, opID: id}
	return b.readiness, nil
}

type vmMemberReadinessDriver struct {
	store         *store
	driver        projectnetwork.InitialReadinessDriver
	opID          string
	calls, errors []string
}

func (d *vmMemberReadinessDriver) check(action string) error {
	j, err := loadProjectMemberReadiness(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || j.Evidence != nil {
		return errors.New("readiness read preceded committed observation journal")
	}
	if err = validateProjectMemberReadiness(d.store.db); err != nil {
		return err
	}
	if action != "" {
		d.calls = append(d.calls, action)
	}
	return nil
}
func (d *vmMemberReadinessDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberReadinessDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(""); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberReadinessDriver) InspectProject(ctx context.Context) (projectnetwork.Observation, error) {
	if err := d.check("project"); err != nil {
		return projectnetwork.Observation{}, d.retain(err)
	}
	o, err := d.driver.InspectProject(ctx)
	return o, d.retain(err)
}
func (d *vmMemberReadinessDriver) InspectPier(ctx context.Context, s projectnetwork.MemberLaunch) (pierstore.Location, error) {
	if err := d.check("pier"); err != nil {
		return pierstore.Location{}, d.retain(err)
	}
	v, err := d.driver.InspectPier(ctx, s)
	return v, d.retain(err)
}
func (d *vmMemberReadinessDriver) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if err := d.check("identity"); err != nil {
		return "", d.retain(err)
	}
	v, err := d.driver.ReadHostIdentity(ctx, endpoint)
	return v, d.retain(err)
}
