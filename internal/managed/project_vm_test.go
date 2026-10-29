//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// This explicit test-binary entry point is never enabled by the ordinary suite.
// It is packaged only in the no-NIC disposable QEMU appliance, after owner setup.
// It does not start an anchor or ship and retains every resource and database.
func TestProjectPreparationDisposableVM(t *testing.T) {
	exerciseProjectDisposableVM(t, false, false, false, false, false, false, false, false)
}
func exerciseProjectDisposableVM(t *testing.T, startAnchor, prepareMembers, launchMembers, applicationState, livePeer, currentContext, currentLifecycle, anchorReview bool, recoveryMode ...bool) {
	anchorRecovery := len(recoveryMode) >= 1 && recoveryMode[0]
	continuation := len(recoveryMode) >= 2 && recoveryMode[1]
	publicRecovery := len(recoveryMode) >= 3 && recoveryMode[2]
	serviceRestart := len(recoveryMode) >= 6 && recoveryMode[5]
	publicMembers := len(recoveryMode) >= 5 && recoveryMode[4]
	publicInitial := len(recoveryMode) >= 4 && recoveryMode[3] || publicMembers
	token := "offline-project-preparation-v1"
	if startAnchor {
		token = "offline-project-anchor-start-v1"
	}
	if prepareMembers {
		token = "offline-project-member-storage-v1"
	}
	if launchMembers {
		token = "offline-project-member-launch-v1"
	}
	if applicationState {
		token = "offline-project-application-state-v1"
	}
	if livePeer {
		token = "offline-project-live-peer-v1"
	}
	if currentContext {
		token = "offline-project-current-context-v1"
	}
	if currentLifecycle {
		token = "offline-project-context-lifecycle-v1"
	}
	if anchorReview {
		token = "offline-project-anchor-review-v1"
	}
	if anchorRecovery {
		token = "offline-project-anchor-recovery-v1"
	}
	if continuation {
		token = "offline-project-anchor-continuation-v1"
	}
	if publicRecovery {
		token = "offline-project-public-recovery-v1"
	}
	if publicInitial {
		token = "offline-project-public-initial-v1"
	}
	if publicMembers {
		token = "offline-project-public-members-v1"
	}
	if serviceRestart {
		token = vmServiceToken
	}
	if os.Getenv("OMABIT_TEST_PROJECT_VM") != token {
		t.Skip("requires the fresh offline qualification VM")
	}
	for name, want := range map[string]string{
		"/opt/omabit-lab/guest-marker": "offline-import-guest-v1\n",
		"/sys/class/dmi/id/sys_vendor": "QEMU\n",
		"/proc/1/comm":                 "systemd\n",
		"/etc/hostname":                "omabit-disposable-import\n",
	} {
		b, e := os.ReadFile(name)
		if e != nil || string(b) != want {
			t.Fatal("disposable guest admission refused", name)
		}
	}
	cmdline, e := os.ReadFile("/proc/cmdline")
	if e != nil || !strings.Contains(string(cmdline), "omabit.disposable-guest=offline-import-v1") {
		t.Fatal("guest boot marker missing")
	}
	if os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/lab" || os.Getenv("XDG_RUNTIME_DIR") != "/run/user/1000" {
		t.Fatal("wrong disposable owner")
	}
	devices, e := os.ReadDir("/sys/class/net")
	if e != nil || len(devices) != 1 || devices[0].Name() != "lo" {
		t.Fatal("guest has an external network device")
	}
	budget := 180 * time.Second
	if launchMembers {
		budget = 3000 * time.Second
	}
	if anchorRecovery {
		budget = 3600 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	dir := "/home/lab/project-qualification-store"
	if serviceRestart {
		dir = vmServiceDirectory
	} else if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal("qualification store must be fresh", e)
	}
	report := map[string]any{"ok": false, "stage": "store", "ship_operations": false}
	defer func() {
		b, e := json.MarshalIndent(report, "", "  ")
		if e == nil {
			e = os.WriteFile("/home/lab/project-preparation.json", append(b, '\n'), 0600)
		}
		if e != nil {
			t.Error(e)
		}
	}()
	if serviceRestart {
		proof := map[string]any{}
		report["owner_service_setup"] = proof
		if e = vmPrepareOwnerService(ctx, proof); e != nil {
			t.Fatal(e)
		}
	}
	s, e := openStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if s != nil {
			s.close()
		}
	}()
	if serviceRestart {
		if e = vmServiceEmptyStore(s, true); e != nil {
			t.Fatal(e)
		}
		s.mode = DevelopmentMode
		report["owner_service_runner_id"] = s.runnerID
	}
	var cleanups []func()
	registerCleanup := func(f func()) { cleanups = append(cleanups, f) }
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		if t.Failed() {
			report["ok"] = false
		}
	}()
	actor := "qualification"
	var initial *vmInitialPublicSetup
	var reg protocol.Operation
	var hit bool
	if publicInitial {
		actor = "local-uid:1000"
		initial, e = vmOpenInitialPublicSetup(ctx, s, token)
		if e != nil {
			t.Fatal(e)
		}
		defer initial.close()
		report["public_initial"] = initial.cli.proof
		reg, e = initial.cli.registerInitial()
	} else {
		reg, hit, e = s.registerProject(actor, projectRequest(t, "register-vm", "vm", 18381))
	}
	if e != nil || hit {
		t.Fatal(reg, hit, e)
	}
	p, e := loadProject(s.db, reg.ProjectID)
	if e != nil {
		t.Fatal(e)
	}
	digest, e := p.Plan.Digest()
	if e != nil {
		t.Fatal(e)
	}
	req := projectPrepareRequest{RequestID: "prepare-vm", ExpectedRevision: p.Revision, ReviewDigest: digest}
	var op protocol.Operation
	if publicInitial {
		initial.cli.project = p.ID
		op, e = initial.cli.operate("prepare", "", req.RequestID, p.Revision, digest, false)
	} else {
		op, hit, e = s.acceptProjectPreparation(actor, p.ID, req)
	}
	if e != nil || hit {
		t.Fatal(op, hit, e)
	}
	report["project"] = p
	report["operation_id"] = op.ID
	report["runner_id"] = s.runnerID
	selection := runtimequalify.ProjectPreparationSelection{RunnerID: s.runnerID, ProjectID: p.ID, OperationID: op.ID, Revision: p.Revision, Generation: 1, Plan: p.Plan}
	report["stage"] = "open-session"
	var prepared projectnetwork.Preparation
	var wrapped *vmPreparationEngine
	if publicInitial {
		select {
		case <-initial.backend.ready:
		case <-ctx.Done():
			t.Fatal("public preparation setup deadline")
		}
		if initial.backend.err != nil {
			t.Fatal(initial.backend.err)
		}
		prepared, wrapped = initial.backend.prepared, initial.backend.wrapped
	} else {
		var adapter projectnetwork.PreparationEngine
		prepared, adapter, e = runtimequalify.OpenProjectPreparation(ctx, selection)
		if e != nil {
			t.Fatal(e)
		}
		wrapped = &vmPreparationEngine{store: s, id: p.ID, engine: adapter}
	}
	report["prepared"] = prepared
	docker := func(args ...string) json.RawMessage {
		t.Helper()
		budget := 30 * time.Second
		if len(args) > 1 && args[1] == "stop" {
			budget = 330 * time.Second
		}
		commandCtx, commandCancel := context.WithTimeout(context.Background(), budget)
		defer commandCancel()
		c := exec.CommandContext(commandCtx, "/usr/bin/docker", append([]string{"--config", filepath.Join("/home/lab/.local/share/urbit-runner/managed", "project-resources-"+p.ID, "docker-config"), "--host", "unix:///run/user/1000/urbit-docker.sock"}, args...)...)
		c.Env = []string{"HOME=/home/lab", "PATH=/usr/bin:/bin", "XDG_RUNTIME_DIR=/run/user/1000", "LC_ALL=C"}
		b, err := c.Output()
		if err != nil {
			t.Error("private fixture command failed", args[0], err)
			return nil
		}
		return json.RawMessage(b)
	}
	volumesBefore := string(docker("volume", "ls", "--quiet"))
	report["volumes_before"] = strings.Fields(volumesBefore)
	report["stage"] = "execute"
	if publicInitial {
		close(initial.backend.release)
		if e = vmWaitPublicWorker(ctx, initial.server); e != nil {
			t.Fatal(e)
		}
	} else {
		e = s.runProjectPreparation(ctx, p.ID, op.ID, prepared, wrapped)
	}
	j, jerr := loadProjectJob(s.db, p.ID)
	report["journal"] = j
	report["calls"] = wrapped.calls
	receipt, recerr := loadOperation(s.db, op.ID)
	report["operation"] = receipt
	// Preserve exact-ID inspections even if the contract rejects normalization.
	if jerr == nil && j.Preparation != nil {
		if id := j.Preparation.Bridge.ID; id != "" {
			report["bridge_inspection"] = docker("network", "inspect", id)
		}
		if id := j.Preparation.Anchor.ID; id != "" {
			report["anchor_inspection"] = docker("container", "inspect", id)
		}
	}
	report["volumes_after"] = strings.Fields(string(docker("volume", "ls", "--quiet")))
	if e != nil || jerr != nil || recerr != nil {
		t.Fatal(e, jerr, recerr)
	}
	if !reflect.DeepEqual(report["volumes_before"], report["volumes_after"]) {
		t.Fatal("anchor generated an unrecorded persistent volume")
	}
	if j.Preparation == nil || j.Preparation.Sequence != 6 || receipt.Status != "succeeded" || len(wrapped.calls) != 4 {
		t.Fatal("incomplete resource preparation")
	}
	if publicInitial {
		repeated, err := initial.cli.operate("prepare", "", req.RequestID, p.Revision, digest, true)
		if err != nil || !reflect.DeepEqual(repeated, receipt) {
			t.Fatal("public preparation lookup changed receipt", err)
		}
		initial.cli.proof["ok"] = true
	}
	calls := len(wrapped.calls)
	if e = s.runProjectPreparation(ctx, p.ID, op.ID, prepared, wrapped); e == nil || len(wrapped.calls) != calls {
		t.Fatal("repeat worker dispatched")
	}
	if _, _, e = runtimequalify.OpenProjectPreparation(ctx, selection); e == nil {
		t.Fatal("existing workspace reused")
	}
	if startAnchor {
		report["anchor_operations"] = true
		var keepUntilMembersStop func(func())
		if launchMembers {
			keepUntilMembersStop = registerCleanup
		}
		report["anchor_start"] = exerciseVMAnchorStart(t, ctx, s, p.ID, docker, keepUntilMembersStop, publicInitial)
		if !report["anchor_start"].(map[string]any)["ok"].(bool) {
			t.Fatal("anchor Start qualification failed")
		}
	}
	if prepareMembers {
		report["member_storage_operations"] = true
		members := exerciseVMMemberStorage(t, ctx, s, p, publicInitial)
		report["member_storage"] = members
		if len(members) != len(p.Members) {
			t.Fatal("incomplete member storage results")
		}
		for _, member := range members {
			if member["ok"] != true {
				t.Fatal("member storage qualification failed")
			}
		}
	}
	if launchMembers {
		report["ship_operations"] = true
		report["stage"] = "member-launch"
		members := exerciseVMMemberLaunch(t, ctx, s, p, docker, registerCleanup, applicationState, livePeer, currentContext, currentLifecycle, anchorReview, anchorRecovery, continuation, publicRecovery, publicMembers)
		report["member_launch"] = members
		if len(members) != len(p.Members) {
			t.Fatal("incomplete member launch results")
		}
		for _, member := range members {
			if member["ok"] != true {
				t.Fatal("member launch qualification failed")
			}
		}
	}
	report["stage"] = "reopen"
	if e = s.close(); e != nil {
		t.Fatal(e)
	}
	s = nil
	s, e = openStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.reconcileInterrupted(); e != nil {
		t.Fatal(e)
	}
	replay, hit, e := s.acceptProjectPreparation(actor, p.ID, req)
	if e != nil || !hit || !reflect.DeepEqual(replay, receipt) {
		t.Fatal("restart changed receipt", e)
	}
	after, e := loadProjectJob(s.db, p.ID)
	if e != nil || !reflect.DeepEqual(j, after) {
		t.Fatal("restart changed journal", e)
	}
	var count int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM instances").Scan(&count); e != nil || count != 0 {
		t.Fatal("preparation created instances", e)
	}
	if startAnchor {
		start := report["anchor_start"].(map[string]any)
		state, err := loadProjectAnchorStart(s.db, p.ID)
		if err != nil || !reflect.DeepEqual(state, start["journal"]) {
			t.Fatal("restart altered anchor Start journal", err)
		}
		receipt, err := loadOperation(s.db, state.OperationID)
		if err != nil || !reflect.DeepEqual(receipt, start["operation"]) {
			t.Fatal("restart altered anchor Start receipt", err)
		}
		start["reopen_unchanged"] = true
	}
	if prepareMembers {
		for _, member := range report["member_storage"].([]map[string]any) {
			state, err := loadProjectMemberStorage(s.db, member["instance_id"].(string))
			if err != nil || !reflect.DeepEqual(state, member["journal"]) {
				t.Fatal("reopen changed member storage journal", err)
			}
			receipt, err := loadOperation(s.db, state.OperationID)
			if err != nil || !reflect.DeepEqual(receipt, member["operation"]) {
				t.Fatal("reopen changed member storage receipt", err)
			}
			member["reopen_unchanged"] = true
		}
	}
	if launchMembers {
		for _, member := range report["member_launch"].([]map[string]any) {
			state, err := loadProjectMemberLaunch(s.db, member["instance_id"].(string))
			if err != nil || !reflect.DeepEqual(state, member["journal"]) {
				t.Fatal("reopen altered member launch journal", err)
			}
			receipt, err := loadOperation(s.db, state.OperationID)
			if err != nil || !reflect.DeepEqual(receipt, member["operation"]) {
				t.Fatal("reopen altered member launch receipt", err)
			}
			ready, ok := member["readiness"].(map[string]any)
			if !ok || ready["ok"] != true {
				t.Fatal("missing complete member readiness receipt")
			}
			observed, err := loadProjectMemberReadiness(s.db, ready["operation_id"].(string))
			if err != nil || !reflect.DeepEqual(observed, ready["journal"]) {
				t.Fatal("reopen changed readiness journal", err)
			}
			readReceipt, err := loadOperation(s.db, observed.OperationID)
			if err != nil || !reflect.DeepEqual(readReceipt, ready["operation"]) {
				t.Fatal("reopen changed readiness receipt", err)
			}
			ready["reopen_unchanged"] = true
			if livePeer && !currentLifecycle && member["instance_id"] == "vm-nec" {
				if member["fixture_cleanup_only"] != true || len(member["live_peer_observations"].([]json.RawMessage)) != 4 {
					t.Fatal("running peer evidence incomplete")
				}
				var stops int
				if err := s.db.QueryRow("SELECT COUNT(*) FROM project_member_stops WHERE instance_id=?", "vm-nec").Scan(&stops); err != nil || stops != 0 {
					t.Fatal("running peer received a queued Stop", err)
				}
				member["reopen_unchanged"] = true
				continue
			}
			stopped, ok := member["stop"].(map[string]any)
			if !ok || stopped["ok"] != true {
				t.Fatal("missing complete member Stop receipt")
			}
			exit, err := loadProjectMemberStop(s.db, stopped["operation_id"].(string))
			if err != nil || !reflect.DeepEqual(exit, stopped["journal"]) {
				t.Fatal("reopen changed member Stop journal", err)
			}
			exitReceipt, err := loadOperation(s.db, exit.OperationID)
			if err != nil || !reflect.DeepEqual(exitReceipt, stopped["operation"]) {
				t.Fatal("reopen changed member Stop receipt", err)
			}
			if recovered, ok := stopped["exit_observation"].(map[string]any); ok {
				if recovered["ok"] != true {
					t.Fatal("missing complete Stop exit observation")
				}
				observedExit, err := loadProjectMemberStopExit(s.db, recovered["operation_id"].(string))
				if err != nil || !reflect.DeepEqual(observedExit, recovered["journal"]) {
					t.Fatal("reopen changed Stop exit observation", err)
				}
				observedReceipt, err := loadOperation(s.db, observedExit.OperationID)
				if err != nil || !reflect.DeepEqual(observedReceipt, recovered["operation"]) {
					t.Fatal("reopen changed Stop exit receipt", err)
				}
				recovered["reopen_unchanged"] = true
			}
			if retired, ok := member["retirement"].(map[string]any); ok {
				if retired["ok"] != true {
					t.Fatal("missing completed member retirement")
				}
				j, err := loadProjectMemberRetirement(s.db, retired["operation_id"].(string))
				if err != nil || !reflect.DeepEqual(j, retired["journal"]) {
					t.Fatal("reopen changed retirement journal", err)
				}
				receipt, err := loadOperation(s.db, j.OperationID)
				if err != nil || !reflect.DeepEqual(receipt, retired["operation"]) {
					t.Fatal("reopen changed retirement receipt", err)
				}
				retired["reopen_unchanged"] = true
			}
			if resumed, ok := member["resume"].(map[string]any); ok {
				if resumed["ok"] != true {
					t.Fatal("missing complete member resume")
				}
				j, err := loadProjectMemberResume(s.db, resumed["operation_id"].(string))
				if err != nil || !reflect.DeepEqual(j, resumed["journal"]) {
					t.Fatal("reopen changed resume journal", err)
				}
				receipt, err := loadOperation(s.db, j.OperationID)
				if err != nil || !reflect.DeepEqual(receipt, resumed["operation"]) {
					t.Fatal("reopen changed resume receipt", err)
				}
				proof, ok := resumed["readiness"].(map[string]any)
				if !ok || proof["ok"] != true {
					t.Fatal("missing replacement readiness")
				}
				readiness, err := loadProjectMemberResumedReadiness(s.db, proof["operation_id"].(string))
				if err != nil || !reflect.DeepEqual(readiness, proof["journal"]) {
					t.Fatal("reopen changed replacement readiness", err)
				}
				observation, err := loadOperation(s.db, readiness.OperationID)
				if err != nil || !reflect.DeepEqual(observation, proof["operation"]) {
					t.Fatal("reopen changed replacement readiness receipt", err)
				}
				if anchorReview && resumed["anchor_review_cleanup_only"] == true {
					if _, exists := resumed["stop"]; exists {
						t.Fatal("anchor review invented final Stop")
					}
				} else {
					currentStop, ok := resumed["stop"].(map[string]any)
					if !ok || currentStop["ok"] != true {
						t.Fatal("missing completed replacement Stop")
					}
					stopJob, err := loadProjectMemberStop(s.db, currentStop["operation_id"].(string))
					if err != nil || !reflect.DeepEqual(stopJob, currentStop["journal"]) {
						t.Fatal("reopen changed replacement Stop", err)
					}
					stopReceipt, err := loadOperation(s.db, stopJob.OperationID)
					if err != nil || !reflect.DeepEqual(stopReceipt, currentStop["operation"]) {
						t.Fatal("reopen changed replacement Stop receipt", err)
					}
					currentStop["reopen_unchanged"] = true
				}
				proof["reopen_unchanged"] = true
				resumed["reopen_unchanged"] = true
			}
			vmReopenLaterMemberCycle(t, s, member, anchorReview)
			stopped["reopen_unchanged"] = true
			member["reopen_unchanged"] = true
		}
		if after := strings.Fields(string(docker("volume", "ls", "--quiet"))); !reflect.DeepEqual(after, report["volumes_before"]) {
			t.Fatal("member generated an implicit volume")
		}
		report["member_volumes_unchanged"] = true
	}
	if currentContext {
		want := 2
		if currentLifecycle {
			want = 4
		}
		if n := verifyVMCurrentContextsReopen(t, s, report["member_launch"]); n != want {
			t.Fatal("current context reopen requires both members", n)
		}
		report["current_context_reopen_count"] = want
	}
	if anchorRecovery {
		vmReopenAnchorRecovery(t, s, report["member_launch"].([]map[string]any))
	}
	report["reopen_receipt_unchanged"] = true
	report["repeated_dispatch_refused"] = true
	report["instances"] = count
	report["stage"] = "complete"
	report["ok"] = true
}

type vmPreparationEngine struct {
	store  *store
	id     string
	engine projectnetwork.PreparationEngine
	calls  []string
}

func (e *vmPreparationEngine) VerifyEngine(ctx context.Context, id string) error {
	return e.engine.VerifyEngine(ctx, id)
}
func (e *vmPreparationEngine) check(kind string, c projectnetwork.Creation, phase string) error {
	j, err := loadProjectJob(e.store.db, e.id)
	if err != nil {
		return err
	}
	if j.Preparation == nil {
		return errors.New("no durable preparation")
	}
	expected := j.Preparation.Bridge
	if kind == "anchor" {
		expected = j.Preparation.Anchor
	}
	if expected != c || c.Phase != phase {
		return errors.New("resource effect preceded its journal commit")
	}
	e.calls = append(e.calls, phase+"-"+kind)
	return nil
}
func (e *vmPreparationEngine) Create(ctx context.Context, kind string, c projectnetwork.Creation, bridge string) (string, error) {
	if err := e.check(kind, c, "dispatched"); err != nil {
		return "", err
	}
	return e.engine.Create(ctx, kind, c, bridge)
}
func (e *vmPreparationEngine) Inspect(ctx context.Context, kind string, c projectnetwork.Creation, bridge string) (projectnetwork.ResourceIdentity, error) {
	if err := e.check(kind, c, "created"); err != nil {
		return projectnetwork.ResourceIdentity{}, err
	}
	return e.engine.Inspect(ctx, kind, c, bridge)
}
