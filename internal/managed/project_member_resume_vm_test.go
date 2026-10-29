//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

func exerciseVMMemberResume(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage, registerCleanup func(func())) {
	exerciseVMMemberResumeFor(t, ctx, r, b, p, members, docker, registerCleanup, "vm-zod", true)
}
func exerciseVMMemberResumeFor(t *testing.T, ctx context.Context, r *Server, b *vmMemberLaunchBackend, p storedProject, members []map[string]any, docker func(...string) json.RawMessage, registerCleanup func(func()), target string, follow bool) {
	t.Helper()
	var member map[string]any
	var selected projectMember
	for _, m := range members {
		if m["instance_id"] == target {
			member = m
		}
	}
	for _, m := range p.Members {
		if m.InstanceID == target {
			selected = m
		}
	}
	if member == nil || selected.InstanceID == "" {
		t.Error("resume target missing")
		return
	}
	retired, ok := member["retirement"].(map[string]any)
	if !ok || retired["ok"] != true {
		t.Error("resume requires completed retirement")
		member["ok"] = false
		return
	}
	original := retired["journal"].(projectMemberRetirementJob)
	result := map[string]any{"ok": false, "stage": "queue"}
	member["resume"] = result
	registered := map[string]bool{}
	b.resumeRegister = func(s projectnetwork.MemberResume, id string) {
		if registered[id] {
			return
		}
		registered[id] = true
		result["cleanup_registered"] = true
		registerCleanup(func() { vmStopResumedMember(t, r.s.runnerID, s, id, result, docker) })
	}
	digest, err := p.Plan.Digest()
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	b.resume, b.resumeErrors = nil, nil
	req := projectMemberResumeRequest{RequestID: "resume-" + original.OperationID, ExpectedRevision: p.Revision, ReviewDigest: digest, RetirementOperationID: original.OperationID}
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
		t.Error("resume worker deadline")
		member["ok"] = false
		return
	}
	result["setup_errors"] = append([]string(nil), b.resumeErrors...)
	if b.resume != nil {
		result["calls"], result["adapter_errors"], result["stopped_peers"] = b.resume.calls, b.resume.errors, b.resume.peers
	}
	j, err := loadProjectMemberResume(r.s.db, op.ID)
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
	if receipt.Status != "succeeded" || j.Resume == nil || j.Resume.Phase != "running" || b.resume == nil || len(b.resume.errors) != 0 || len(b.resumeErrors) != 0 || !reflect.DeepEqual(b.resume.calls, []string{"inspect-planned", "create-create-dispatched", "inspect-created", "start-start-dispatched", "inspect-started"}) {
		t.Error("member resume incomplete")
		member["ok"] = false
		return
	}
	result["container_inspection"] = docker("container", "inspect", j.Resume.Container.ID)
	warmup, cancel := context.WithTimeout(ctx, 900*time.Second)
	err = vmWaitInitialIdentity(warmup, selected, result)
	cancel()
	if err != nil {
		t.Error(err)
		member["ok"] = false
		return
	}
	current, err := loadProjectMemberRetirement(r.s.db, original.OperationID)
	if err != nil || !reflect.DeepEqual(current, original) {
		t.Error("resume changed original retirement", err)
		member["ok"] = false
		return
	}
	again, hit, err := vmQueueMemberAction(r, b, p.ID, target, req, true)
	if err != nil || !hit || !reflect.DeepEqual(again, receipt) {
		t.Error("resume replay changed receipt", err)
		member["ok"] = false
		return
	}
	req.RequestID = "repeat-resume-" + original.OperationID
	if _, _, err = vmQueueMemberAction(r, b, p.ID, target, req, false); err == nil {
		t.Error("retirement could be consumed twice")
		member["ok"] = false
		return
	}
	result["original_retirement_unchanged"], result["replay_unchanged"], result["repeat_dispatch_refused"], result["stage"], result["ok"] = true, true, true, "complete", true
	exerciseVMMemberResumedReadiness(t, ctx, r, b, p, []map[string]any{member})
	if !follow {
		return
	}
	if b.appState && !t.Failed() {
		switch j.Resume.PreviousGeneration {
		case 1:
			vmApplicationPhase(t, ctx, b, "after-first")
			if !t.Failed() {
				vmApplicationPhase(t, ctx, b, "before-second")
			}
		case 2:
			vmApplicationPhase(t, ctx, b, "after-second")
		default:
			t.Error("unexpected application fixture generation")
		}
	}
	if b.currentContext && j.Resume.PreviousGeneration == 2 && !t.Failed() {
		exerciseVMCurrentContextReadiness(t, ctx, r, b, p, result, j.Resume.OperationID)
	}
	if !t.Failed() && !(b.currentLifecycle && j.Resume.PreviousGeneration == 2) {
		exerciseVMCurrentMemberStop(t, ctx, r, b, p, member)
	}
}
func (b *vmMemberLaunchBackend) OpenProjectMemberResume(ctx context.Context, s runtimequalify.ProjectMemberResumeSelection) (projectnetwork.MemberResume, projectnetwork.MemberResumeDriver, error) {
	initial, d, err := runtimequalify.OpenProjectMemberResume(ctx, s)
	if err != nil {
		b.resumeErrors = append(b.resumeErrors, vmLaunchError(err))
		return initial, nil, err
	}
	peers := s.Stop.StoppedPeers
	if s.ResumedStop != nil {
		peers = s.ResumedStop.StoppedPeers
	}
	if s.CurrentStop != nil {
		peers = s.CurrentStop.StoppedPeers
	}
	b.resume = &vmMemberResumeDriver{store: b.store, driver: d, opID: s.OperationID, peers: peers, register: b.resumeRegister}
	return initial, b.resume, nil
}

type vmMemberResumeDriver struct {
	store         *store
	driver        projectnetwork.MemberResumeDriver
	opID          string
	peers         []projectnetwork.MemberStop
	calls, errors []string
	register      func(projectnetwork.MemberResume, string)
}

func (d *vmMemberResumeDriver) check(s *projectnetwork.MemberResume, phase, action string) error {
	j, err := loadProjectMemberResume(d.store.db, d.opID)
	if err != nil {
		return err
	}
	op, err := loadOperation(d.store.db, d.opID)
	if err != nil {
		return err
	}
	if op.Status != "running" || j.Resume == nil || s != nil && (j.Resume.Phase != phase || !reflect.DeepEqual(*j.Resume, *s)) {
		return errors.New("resume effect preceded committed journal")
	}
	if err = validateProjectMemberResumes(d.store.db); err != nil {
		return err
	}
	if action != "" {
		d.calls = append(d.calls, action)
	}
	return nil
}
func (d *vmMemberResumeDriver) retain(err error) error {
	if err != nil {
		d.errors = append(d.errors, vmLaunchError(err))
	}
	return err
}
func (d *vmMemberResumeDriver) VerifyEngine(ctx context.Context, id string) error {
	if err := d.check(nil, "", ""); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.VerifyEngine(ctx, id))
}
func (d *vmMemberResumeDriver) InspectAbsent(ctx context.Context, s projectnetwork.MemberResume) (projectnetwork.MemberAbsenceObservation, error) {
	if err := d.check(&s, "planned", "inspect-planned"); err != nil {
		return projectnetwork.MemberAbsenceObservation{}, d.retain(err)
	}
	o, err := d.driver.InspectAbsent(ctx, s)
	return o, d.retain(err)
}
func (d *vmMemberResumeDriver) Create(ctx context.Context, s projectnetwork.MemberResume) (string, error) {
	if err := d.check(&s, "create-dispatched", "create-create-dispatched"); err != nil {
		return "", d.retain(err)
	}
	id, err := d.driver.Create(ctx, s)
	if id != "" && d.register != nil {
		d.register(s, id)
	}
	return id, d.retain(err)
}
func (d *vmMemberResumeDriver) Inspect(ctx context.Context, s projectnetwork.MemberResume) (projectnetwork.MemberResumeObservation, error) {
	if err := d.check(&s, s.Phase, "inspect-"+s.Phase); err != nil {
		return projectnetwork.MemberResumeObservation{}, d.retain(err)
	}
	o, err := d.driver.Inspect(ctx, s)
	return o, d.retain(err)
}
func (d *vmMemberResumeDriver) Start(ctx context.Context, s projectnetwork.MemberResume) error {
	if err := d.check(&s, "start-dispatched", "start-start-dispatched"); err != nil {
		return d.retain(err)
	}
	return d.retain(d.driver.Start(ctx, s))
}

// Only a container ID returned by this fixture's Create can register this cleanup.
// Preserve the resumed pier; cleanup sends Stop only after exact label validation.
func vmStopResumedMember(t *testing.T, runnerID string, s projectnetwork.MemberResume, id string, result map[string]any, docker func(...string) json.RawMessage) {
	t.Helper()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(id) {
		t.Error("invalid returned resume cleanup ID")
		return
	}
	if retired, ok := result["retirement"].(map[string]any); ok && retired["ok"] == true {
		j, valid := retired["journal"].(projectMemberRetirementJob)
		if !valid || j.ContainerID != id || j.InstanceID != s.Retirement.Exit.Stop.InstanceID || j.Retirement.Phase != "absent" || j.Retirement.After == nil || !j.Retirement.Exit.Clean() || j.Retirement.Exit.Stop.LaunchOperationID != s.OperationID || j.Retirement.Exit.Stop.Storage != s.Retirement.Exit.Stop.Storage {
			result["ok"] = false
			t.Error("retired replacement lacks exact retained evidence")
			return
		}
		raw := docker("container", "ls", "--all", "--quiet", "--no-trunc")
		if raw == nil {
			result["ok"] = false
			return
		}
		ids := strings.Fields(string(raw))
		for _, found := range ids {
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(found) || found == id {
				result["ok"] = false
				t.Error("retired replacement still present or incomplete inventory")
				return
			}
		}
		result["retirement_cleanup_inventory"], result["retired_container_absent"], result["clean_member_stop"] = ids, true, true
		return
	}
	if vmRecoveryRetirementCleanup(t, id, s.OperationID, s.Retirement.Exit.Stop.Storage, result, docker) {
		return
	}
	old := s.Retirement.Exit.Stop
	runtime := ""
	for _, m := range old.Binding.Members {
		if m.InstanceID == old.InstanceID {
			runtime = m.RuntimeID
		}
	}
	generation := strconv.FormatUint(s.PreviousGeneration+1, 10)
	label := "org.urbit-runner.project"
	want := map[string]string{label: "true", label + ".runner": runnerID, label + ".project": old.Binding.ProjectID, label + ".generation": strconv.FormatUint(old.Binding.Generation, 10), label + ".instance": old.InstanceID, label + ".runtime": runtime, label + ".operation": s.OperationID, label + ".container-generation": generation}
	var values []struct {
		ID     string `json:"Id"`
		Name   string
		Config struct{ Labels map[string]string }
		State  struct {
			Running, OOMKilled bool
			ExitCode           int
			Status, StartedAt  string
		}
	}
	raw := docker("container", "inspect", id)
	result["final_inspection_before_stop"] = raw
	if json.Unmarshal(raw, &values) != nil || len(values) != 1 || values[0].ID != id || values[0].Name != "/urbit-project-member-"+runtime+"-g"+generation || !reflect.DeepEqual(values[0].Config.Labels, want) {
		result["ok"] = false
		t.Error("resumed member cleanup ownership mismatch")
		return
	}
	if values[0].State.Running {
		docker("container", "stop", "--timeout", "300", id)
	}
	raw = docker("container", "inspect", id)
	result["cleanup_inspection"] = raw
	if json.Unmarshal(raw, &values) != nil || len(values) != 1 || values[0].ID != id || values[0].State.Running || values[0].State.OOMKilled || values[0].State.ExitCode != 0 || values[0].State.Status != "exited" || values[0].State.StartedAt == "0001-01-01T00:00:00Z" {
		result["ok"] = false
		t.Error("resumed member did not stop cleanly")
		return
	}
	state, err := vmRetirementState(s.Retirement.Exit.Stop.Storage)
	if err != nil {
		result["ok"] = false
		t.Error("resumed state could not be retained", err)
		return
	}
	result["state_after_stop"] = state
	result["clean_member_stop"] = true
}
