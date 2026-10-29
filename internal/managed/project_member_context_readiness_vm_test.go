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

func TestProjectCurrentContextDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, true, true, true, true, true, true, false, false)
}

// Both current members must run. Record new observations while preserving every
// original readiness receipt; this helper performs no lifecycle mutation.
func exerciseVMCurrentContextReadiness(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, owner map[string]any, creator string) {
	t.Helper()
	var results []map[string]any
	defer func() { owner["current_context_readiness"] = results }()
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return
	}
	for _, member := range p.Members {
		result := map[string]any{"instance_id": member.InstanceID, "ok": false, "stage": "queue"}
		results = append(results, result)
		var originalID string
		if err = r.s.db.QueryRow("SELECT operation_id FROM project_member_readiness WHERE project_id=? AND instance_id=?", p.ID, member.InstanceID).Scan(&originalID); err != nil {
			t.Error(err)
			return
		}
		original, err := loadProjectMemberReadiness(r.s.db, originalID)
		if err != nil {
			t.Error(err)
			return
		}
		src, err := projectContextReadinessSource(r.s.db, p.ID, member.InstanceID, originalID, nil)
		if err != nil {
			t.Error(err)
			return
		}
		b.contextReadiness = nil
		b.contextReadinessErrors = nil
		req := projectMemberContextReadinessRequest{OriginalReadinessOperationID: originalID, CreatorOperationID: src.Context.Member.OperationID, RequestID: "context-" + member.InstanceID + "-" + creator, ExpectedRevision: p.Revision, ReviewDigest: digest}
		op, hit, err := vmQueueMemberAction(r, b, p.ID, member.InstanceID, req, false)
		if err != nil || hit {
			t.Error(op, hit, err)
			return
		}
		result["operation_id"] = op.ID
		done := make(chan struct{})
		go func() { r.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-ctx.Done():
			t.Error("current readiness worker deadline")
			return
		}
		result["setup_errors"] = append([]string(nil), b.contextReadinessErrors...)
		if b.contextReadiness != nil {
			result["calls"] = b.contextReadiness.calls
			result["adapter_errors"] = b.contextReadiness.errors
		}
		j, err := loadProjectMemberContextReadiness(r.s.db, op.ID)
		if err != nil {
			t.Error(err)
			return
		}
		result["journal"] = j
		receipt, err := loadOperation(r.s.db, op.ID)
		if err != nil {
			t.Error(err)
			return
		}
		result["operation"] = receipt
		if receipt.Status != "succeeded" || j.Phase != "observed" || j.Evidence == nil || b.contextReadiness == nil || len(b.contextReadiness.errors) != 0 || !reflect.DeepEqual(j.Context, src.Context) || !reflect.DeepEqual(b.contextReadiness.calls, []string{"project", "identity", "project"}) {
			t.Error("current readiness failed", receipt.Status)
			return
		}
		replay, hit, err := vmQueueMemberAction(r, b, p.ID, member.InstanceID, req, true)
		if err != nil || !hit || !reflect.DeepEqual(replay, receipt) {
			t.Error("current readiness replay changed receipt", err)
			return
		}
		after, err := loadProjectMemberReadiness(r.s.db, originalID)
		if err != nil || !reflect.DeepEqual(original, after) {
			t.Error("current readiness rewrote original evidence", err)
			return
		}
		result["original_readiness"] = original
		result["original_readiness_unchanged"] = true
		result["replay_unchanged"] = true
		result["stage"] = "complete"
		result["ok"] = true
	}
}
func (b *vmMemberLaunchBackend) OpenCurrentProjectReadiness(ctx context.Context, s runtimequalify.ProjectCurrentReadinessSelection) (projectnetwork.CurrentReadinessDriver, error) {
	driver, err := runtimequalify.OpenCurrentProjectReadiness(ctx, s)
	if err != nil {
		b.contextReadinessErrors = append(b.contextReadinessErrors, vmLaunchError(err))
		return nil, err
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_context_readiness j JOIN operations o ON j.operation_id=o.id WHERE j.project_id=? AND j.instance_id=? AND json_extract(o.body,'$.status')='queued'", s.Project.ProjectID, s.Original.InstanceID).Scan(&id); err != nil {
		return nil, err
	}
	b.contextReadiness = &vmCurrentReadinessDriver{store: b.store, driver: driver, opID: id}
	return b.contextReadiness, nil
}

type vmCurrentReadinessDriver struct {
	store         *store
	driver        projectnetwork.CurrentReadinessDriver
	opID          string
	calls, errors []string
}

func (d *vmCurrentReadinessDriver) check(c *projectnetwork.MemberContext) error {
	j, err := loadProjectMemberContextReadiness(d.store.db, d.opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || op.Status != "running" || c != nil && !reflect.DeepEqual(j.Context, *c) {
		return errors.New("current readiness preceded committed observation")
	}
	return nil
}
func (d *vmCurrentReadinessDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(nil); err != nil {
		return err
	}
	return d.driver.VerifyEngine(ctx, id)
}
func (d *vmCurrentReadinessDriver) Inspect(ctx context.Context, c projectnetwork.MemberContext) (projectnetwork.MemberStopObservation, error) {
	if err := d.check(&c); err != nil {
		return projectnetwork.MemberStopObservation{}, err
	}
	d.calls = append(d.calls, "project")
	o, err := d.driver.Inspect(ctx, c)
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return o, err
}
func (d *vmCurrentReadinessDriver) ReadHostIdentity(ctx context.Context, endpoint string) (string, error) {
	if err := d.check(nil); err != nil {
		return "", err
	}
	d.calls = append(d.calls, "identity")
	ship, err := d.driver.ReadHostIdentity(ctx, endpoint)
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return ship, err
}

func verifyVMCurrentContextsReopen(t *testing.T, s *store, value any) int {
	t.Helper()
	count := 0
	switch v := value.(type) {
	case []map[string]any:
		for _, item := range v {
			count += verifyVMCurrentContextsReopen(t, s, item)
		}
	case map[string]any:
		for key, child := range v {
			if key != "current_context_readiness" {
				count += verifyVMCurrentContextsReopen(t, s, child)
				continue
			}
			rows, ok := child.([]map[string]any)
			if !ok {
				t.Error("current readiness results missing")
				continue
			}
			for _, row := range rows {
				saved, ok := row["journal"].(projectMemberContextReadinessJob)
				if !ok || row["ok"] != true {
					t.Error("current readiness result incomplete")
					continue
				}
				actual, err := loadProjectMemberContextReadiness(s.db, saved.OperationID)
				if err != nil || !reflect.DeepEqual(saved, actual) {
					t.Error("reopen changed current readiness", err)
					continue
				}
				old, err := loadProjectMemberReadiness(s.db, saved.OriginalReadinessOperationID)
				if err != nil || !reflect.DeepEqual(old, row["original_readiness"]) {
					t.Error("reopen changed original readiness", err)
					continue
				}
				if _, _, err = contextReadinessJobSource(s.db, actual); err != nil {
					t.Error(err)
					continue
				}
				row["reopen_unchanged"] = true
				count++
			}
		}
	}
	return count
}
