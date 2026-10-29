//go:build linux && amd64

package managed

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// Called only by the guarded disposable-VM member-launch test after its pair is
// responsive. The adapter cannot launch, stop, adopt or repair any resource.
func exerciseVMMemberResumedReadiness(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, results []map[string]any) {
	t.Helper()
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		return
	}
	for _, member := range results {
		instance := member["instance_id"].(string)
		resumed, ok := member["resume"].(map[string]any)
		if !ok {
			continue
		}
		if resumed["ok"] != true {
			t.Error("readiness requires complete replacement")
			member["ok"] = false
			return
		}
		original := resumed["journal"].(projectMemberResumeJob)
		result := map[string]any{"ok": false, "stage": "queue"}
		resumed["readiness"] = result
		b.resumedReadiness = nil
		b.resumedReadinessErrors = nil
		req := projectMemberResumedReadinessRequest{RequestID: "observe-" + original.OperationID, ExpectedRevision: p.Revision, ReviewDigest: digest, ResumeOperationID: original.OperationID}
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
		result["setup_errors"] = append([]string(nil), b.resumedReadinessErrors...)
		if b.resumedReadiness != nil {
			result["calls"] = b.resumedReadiness.calls
			result["adapter_errors"] = b.resumedReadiness.errors
			result["observations"] = b.resumedReadiness.observations
			result["events"] = b.resumedReadiness.events
			result["identity_ship"] = b.resumedReadiness.identityShip
		}
		j, err := loadProjectMemberResumedReadiness(r.s.db, op.ID)
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
		// The expensive whole-history audit brackets the read-only experiment,
		// outside its production freshness interval. Every individual read still
		// proves the exact committed observation phase below.
		if err = validateProjectMemberResumedReadiness(r.s.db); err != nil {
			result["post_audit_error"] = vmLaunchError(err)
			t.Error(err)
			member["ok"] = false
			return
		}
		result["post_audit_passed"] = true
		if receipt.Status != "succeeded" || j.Phase != "observed" || j.Evidence == nil || b.resumedReadiness == nil || len(b.resumedReadiness.errors) != 0 {
			t.Error("readiness did not complete", receipt.Status)
			member["ok"] = false
			return
		}
		if !reflect.DeepEqual(b.resumedReadiness.calls, []string{"project", "identity", "project"}) {
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
		current, err := loadProjectMemberResume(r.s.db, original.OperationID)
		if err != nil || !reflect.DeepEqual(current, original) {
			t.Error("readiness changed original replacement", err)
			member["ok"] = false
			return
		}
		result["original_resume_unchanged"] = true
		result["replay_unchanged"] = true
		result["stage"] = "complete"
		result["ok"] = true
	}
}
func (b *vmMemberLaunchBackend) OpenResumedProjectReadiness(ctx context.Context, selection runtimequalify.ProjectResumedReadinessSelection) (projectnetwork.ResumedReadinessDriver, error) {
	// This fixture-only audit precedes collection; it must not add repeated
	// full-database work between the two fresh runtime observations.
	if err := validateProjectMemberResumedReadiness(b.store.db); err != nil {
		b.resumedReadinessErrors = append(b.resumedReadinessErrors, vmLaunchError(err))
		return nil, err
	}
	d, err := runtimequalify.OpenResumedProjectReadiness(ctx, selection)
	if err != nil {
		b.resumedReadinessErrors = append(b.resumedReadinessErrors, vmLaunchError(err))
		return nil, err
	}
	project := selection.Resume.Stop.Project.ProjectID
	if selection.Resume.ResumedStop != nil {
		project = selection.Resume.ResumedStop.Project.ProjectID
	}
	if selection.Resume.CurrentStop != nil {
		project = selection.Resume.CurrentStop.ReadinessSource.Project.ProjectID
	}
	var id string
	if err = b.store.db.QueryRow("SELECT j.operation_id FROM project_member_resumed_readiness j JOIN operations o ON j.operation_id=o.id WHERE j.project_id=? AND json_extract(o.body,'$.status')='queued'", project).Scan(&id); err != nil {
		return nil, err
	}
	b.resumedReadiness = &vmMemberResumedReadinessDriver{store: b.store, driver: d, opID: id}
	return b.resumedReadiness, nil
}

type vmMemberResumedReadinessDriver struct {
	store         *store
	driver        projectnetwork.ResumedReadinessDriver
	opID          string
	calls, errors []string
	observations  []projectnetwork.MemberResumeObservation
	identityShip  string
	events        []vmResumedReadinessEvent
}
type vmResumedReadinessEvent struct {
	Action     string    `json:"action"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Error      string    `json:"error,omitempty"`
}

func (d *vmMemberResumedReadinessDriver) event(action string, start time.Time, err error) {
	e := vmResumedReadinessEvent{Action: action, StartedAt: start, FinishedAt: time.Now().UTC()}
	if err != nil {
		e.Error = vmLaunchError(err)
	}
	d.events = append(d.events, e)
}

func (d *vmMemberResumedReadinessDriver) check(action string) error {
	j, err := loadProjectMemberResumedReadiness(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if j.Phase != "observing" || j.Evidence != nil {
		return errors.New("readiness read preceded committed observation journal")
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if op.Status != "running" || op.Kind != "observe-resumed-project-member" || op.ProjectID != j.ProjectID || op.InstanceID != j.InstanceID {
		return errors.New("readiness read lost its running operation")
	}
	if action != "" {
		d.calls = append(d.calls, action)
	}
	return nil
}
func (d *vmMemberResumedReadinessDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberResumedReadinessDriver) VerifyEngine(ctx context.Context, id string) (err error) {
	start := time.Now().UTC()
	defer func() { d.event("engine", start, err) }()
	if err = d.check(""); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberResumedReadinessDriver) Inspect(ctx context.Context, s projectnetwork.MemberResume) (o projectnetwork.MemberResumeObservation, err error) {
	start := time.Now().UTC()
	defer func() { d.event("project", start, err) }()
	if err = d.check("project"); err != nil {
		return o, d.retain(err)
	}
	o, err = d.driver.Inspect(ctx, s)
	if err == nil {
		d.observations = append(d.observations, o)
	}
	return o, d.retain(err)
}
func (d *vmMemberResumedReadinessDriver) ReadHostIdentity(ctx context.Context, endpoint string) (ship string, err error) {
	start := time.Now().UTC()
	defer func() { d.event("identity", start, err) }()
	if err = d.check("identity"); err != nil {
		return "", d.retain(err)
	}
	ship, err = d.driver.ReadHostIdentity(ctx, endpoint)
	if err == nil {
		d.identityShip = ship
	}
	return ship, d.retain(err)
}

// This source-only test never invokes Docker or a host adapter. It proves that
// moving fixture audits does not permit a read before dispatch or after interrupt.
func TestVMResumedReadinessChecksCommittedRunningOperation(t *testing.T) {
	s, err := testOpenStore(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	p, req := resumedReadinessProject(t, s)
	op, hit, err := s.acceptProjectMemberResumedReadiness("qualification", p.ID, p.Members[0].InstanceID, req)
	if err != nil || hit {
		t.Fatal(err, hit)
	}
	d := &vmMemberResumedReadinessDriver{store: s, opID: op.ID}
	if err = d.check("project"); err == nil || len(d.calls) != 0 {
		t.Fatal("read before observing intent")
	}
	if _, err = s.beginProjectMemberResumedReadiness(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = validateProjectMemberResumedReadiness(s.db); err != nil {
		t.Fatal(err)
	}
	if err = d.check("project"); err != nil || !reflect.DeepEqual(d.calls, []string{"project"}) {
		t.Fatal("committed read refused", err)
	}
	if err = s.interruptMemberResumedReadiness(op.ID); err != nil {
		t.Fatal(err)
	}
	if err = d.check("identity"); err == nil || !reflect.DeepEqual(d.calls, []string{"project"}) {
		t.Fatal("interrupted read admitted")
	}
	if err = validateProjectMemberResumedReadiness(s.db); err != nil {
		t.Fatal(err)
	}
}
