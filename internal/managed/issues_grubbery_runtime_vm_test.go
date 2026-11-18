//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const runtimeVMCase = "cow-recover-pending-zod-v1"
const runtimeVMPin = "a23f10ddc89657489d8e55eaf5e4362603848366a447dbf9f51ce192837a08f9"
const runtimeVMBinaries = "/run/omabit-runtime-candidate"

// Exact task-only read-only-backed fixture, never a host/manual-review runner.
// Uses supported lifecycle and public typed source/recovery APIs throughout.
func TestIssuesGrubberyRuntimeRecoveryDisposableVM(t *testing.T) {
	if os.Getenv("OMABIT_APP_RUNTIME_VM") == "" {
		t.Skip("explicit disposable pending-app VM only")
	}
	vendor, e1 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e2 := os.ReadFile("/proc/cmdline")
	raw, e3 := os.ReadFile("/opt/omabit-release-input/scenario.json")
	var scenario struct {
		Kind        string `json:"kind"`
		Case        string `json:"case"`
		Source      string `json:"source_commit"`
		Predecessor string `json:"predecessor_root_sha256"`
		Original    int    `json:"original_schema"`
		Target      int    `json:"target_schema"`
	}
	if e1 != nil || e2 != nil || e3 != nil || json.Unmarshal(raw, &scenario) != nil || scenario.Kind != "issues18-product-grubbery-runtime-v1" || scenario.Case != runtimeVMCase || scenario.Predecessor != runtimeVMPin || scenario.Original != 62 || scenario.Target != StateSchemaVersion || scenario.Source != issuesVMSourceRevision || len(issuesVMSourceRevision) != 40 || strings.TrimSpace(string(vendor)) != "QEMU" || !strings.Contains(string(cmdline), "omabit.disposable-guest=release-upgrade-v1") || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/review" || os.Getenv("OMABIT_APP_RUNTIME_VM") != runtimeVMCase {
		t.Fatal("refused outside fixed disposable pending-app copy")
	}
	fixture := appVMFixture{Root: "/home/review/omabit-app-runtime-recovery", Instance: "grubbery-zod", Ship: "~zod", Environment: runtimeVMCase, Port: 19380}
	if _, err := os.Lstat(fixture.Root); !os.IsNotExist(err) {
		t.Fatal("fresh recovery proof directory required")
	}
	g02Check(t, os.Mkdir(fixture.Root, 0700))
	// The single-CPU native fixture includes the original recovery, both full
	// source workflows and restart/state checks. Each actual RPC still keeps
	// its independent four-minute bound; this is only the whole-case budget.
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Minute)
	defer cancel()
	proof := map[string]any{"scope": "copied fixed pending app pier; actual source runner/Docker/curl, supported lifecycle and audited recovery, CLI/stdio; no installed/DEV/production or full workflow qualification"}
	save := func(key string, value any) {
		t.Helper()
		proof[key] = value
		body, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile(filepath.Join(fixture.Root, "proof.json"), body, 0600))
		t.Log(key)
	}
	config := filepath.Join(fixture.Root, "registry")
	registry, err := collection.Open(config)
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "fixture", "/run/user/1000/urbit-runner/runner.sock")
	g02Check(t, err)
	c := entry.Client(true)
	defer c.Close()
	original, err := c.GrubberyAppOperation(ctx, entry.Identity(), stopEvidenceDigest([]byte("product-source-request-v1")))
	g02Check(t, err)
	b := original.Record.Plan.Binding
	if original.Record.Version != 2 || len(original.Record.Events) != 17 || b.Target.InstanceID != fixture.Instance || b.RuntimeID != "5ff5f1b73b3baeef0203297e8747202573ca0577918e48bd79b0332201e7b6fa" || b.Target.Desk != grubbery.FrameworkDesk {
		t.Fatal("wrong exact original pending receipt")
	}
	save("original-uncertain-before-recovery", original)
	lookupInstance := func() protocol.Instance {
		t.Helper()
		snapshot, e := c.Snapshot(ctx)
		g02Check(t, e)
		if len(snapshot.Instances) != 1 || snapshot.Instances[0].ID != fixture.Instance || snapshot.Instances[0].Ship != fixture.Ship || snapshot.Instances[0].ConfigRevision != b.ConfigRevision {
			t.Fatal("fixed registered pier/config inventory changed")
		}
		return snapshot.Instances[0]
	}
	lifecycle := func(action, request string) {
		t.Helper()
		row := lookupInstance()
		op, e := c.Operate(ctx, fixture.Instance, protocol.OperationRequest{Kind: action, RequestID: request, ExpectedRevision: row.ConfigRevision})
		g02Check(t, e)
		for deadline := time.Now().Add(5 * time.Minute); ; {
			snapshot, e := c.Snapshot(ctx)
			g02Check(t, e)
			done := false
			for _, current := range snapshot.Operations {
				if current.ID == op.ID {
					if current.Status == "succeeded" {
						done = true
					} else if current.Status != "queued" && current.Status != "running" && current.Status != "pending" {
						t.Fatal("supported fixture lifecycle failed", action, current.Status)
					}
				}
			}
			if done {
				return
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("supported lifecycle unresolved", action)
			}
			time.Sleep(time.Second)
		}
	}
	waitRunning := func() protocol.Instance {
		t.Helper()
		for deadline := time.Now().Add(time.Minute); ; {
			snapshot, e := c.Snapshot(ctx)
			g02Check(t, e)
			if row, ok := appVMRunningInstanceFor(snapshot, fixture.Instance); ok {
				return row
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("current process selection unresolved")
			}
			time.Sleep(time.Second)
		}
	}
	// The inherited process is already stopped. Supported Stop retires only its
	// recorded container; Start resumes its initialized pier, never bootstrap.
	lifecycle("stop", "runtime-recovery-retire-stopped")
	lifecycle("start", "runtime-recovery-resume-existing")
	current := waitRunning()
	if current.Resources.ContainerID == b.RuntimeID {
		t.Fatal("supported replacement did not create a new exact container")
	}
	save("supported-first-replacement", current)
	for deadline := time.Now().Add(time.Minute); ; {
		ready, readErr := registry.InspectDesk(ctx, "fixture", entry.RunnerID, fixture.Instance, b.Target.Desk)
		if readErr == nil {
			if ready.Observation.Selection.RuntimeID != current.Resources.ContainerID || ready.Observation.Observation != b.FrameworkBaseline {
				t.Fatal("resumed framework/config baseline changed")
			}
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Fatal("resumed framework read readiness unavailable")
		}
		time.Sleep(time.Second)
	}
	stderr, err := os.OpenFile(filepath.Join(fixture.Root, "mcp-stderr.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	g02Check(t, err)
	defer stderr.Close()
	command := exec.CommandContext(ctx, runtimeVMBinaries+"/candidate/omabit", "--config-dir", config, "mcp", "serve", "--allow-runner", "fixture:"+entry.RunnerID, "--enable-mutations")
	command.Stderr = stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "pending-runtime-native", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	g02Check(t, err)
	defer session.Close()
	target := service.Target{Name: "fixture", RunnerID: entry.RunnerID}
	invocations := 0
	invoke := func(adapter, id string, input, output any) {
		t.Helper()
		data, e := json.Marshal(input)
		g02Check(t, e)
		invocations++
		base := filepath.Join(fixture.Root, fmt.Sprintf("call-%03d-%s", invocations, id))
		g02Check(t, os.WriteFile(base+"-request.json", data, 0600))
		t.Log("invoke", invocations, adapter, id)
		var body []byte
		if adapter == "cli" {
			cmd := exec.CommandContext(ctx, runtimeVMBinaries+"/candidate/omabit", "--config-dir", config, "call", id, "--input-json", string(data))
			body, e = cmd.Output()
			if exit, ok := e.(*exec.ExitError); ok {
				g02Check(t, os.WriteFile(base+"-stderr.log", exit.Stderr, 0600))
			}
		} else {
			result, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + id, Arguments: json.RawMessage(data)})
			g02Check(t, callErr)
			if len(result.Content) != 1 {
				t.Fatal("one typed response required")
			}
			text, ok := result.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("typed text response required")
			}
			body = []byte(text.Text)
		}
		g02Check(t, os.WriteFile(base+"-response.json", body, 0600))
		var envelope struct {
			Data  json.RawMessage  `json:"data"`
			Error *service.Problem `json:"error"`
		}
		g02Check(t, json.Unmarshal(body, &envelope))
		if envelope.Error != nil && envelope.Error.Code == "outcome_uncertain" {
			// Lookup only after a lost response; never resubmit a mutation or
			// substitute a new original/audit ID to make the gate progress.
			switch in := input.(type) {
			case service.AppSourcePhaseInput:
				if id != "app.advance-source.v1" && id != "app.reconcile-source.v1" {
					break
				}
				saved, lookupErr := registry.GrubberyAppOperation(ctx, "fixture", entry.RunnerID, in.OperationID)
				g02Check(t, lookupErr)
				if saved.Record.Plan.Digest != in.Request.PlanDigest {
					t.Fatal("lookup changed original plan")
				}
				envelope.Data, e = json.Marshal(saved)
				envelope.Error = nil
			case service.AppRuntimeApplyInput:
				if id != "app.apply-runtime-recovery.v1" {
					break
				}
				saved, lookupErr := registry.GrubberyRuntimeRecovery(ctx, "fixture", entry.RunnerID, in.Request.OriginalRequestID, in.Request.RequestID)
				g02Check(t, lookupErr)
				envelope.Data, e = json.Marshal(saved)
				envelope.Error = nil
			}
			if envelope.Error == nil {
				g02Check(t, os.WriteFile(base+"-saved-lookup.json", envelope.Data, 0600))
			}
		}
		if envelope.Error != nil {
			t.Fatal("typed native recovery call refused; private original response retained", id, envelope.Error.Code)
		}
		g02Check(t, e)
		g02Check(t, json.Unmarshal(envelope.Data, output))
	}
	var review protocol.GrubberyRuntimeReview
	invoke("cli", "app.review-runtime-recovery.v1", service.AppRuntimeReviewInput{Target: target, Request: protocol.GrubberyRuntimeReviewRequest{OriginalRequestID: original.Record.Plan.RequestID, Selection: protocol.DeskSelection{Target: b.Target, RuntimeID: current.Resources.ContainerID, ConfigRevision: b.ConfigRevision}}}, &review)
	g02Check(t, review.Validate())
	if !reflect.DeepEqual(review.Original, original.Record) {
		t.Fatal("review rewrote original uncertainty")
	}
	save("runtime-review", review)
	auditID := stopEvidenceDigest([]byte("runtime-recovery-original-audit-v1"))
	var audit protocol.GrubberyRuntimeRecoveryStatus
	invoke("mcp", "app.apply-runtime-recovery.v1", service.AppRuntimeApplyInput{Target: target, Request: protocol.GrubberyRuntimeApplyRequest{OriginalRequestID: original.Record.Plan.RequestID, RequestID: auditID, Review: review.Review, Confirm: true}}, &audit)
	g02Check(t, audit.Validate())
	if audit.Receipt == nil || !reflect.DeepEqual(audit.Original, original.Record) {
		t.Fatal("audit resolved or changed original uncertainty")
	}
	save("runtime-audit", audit)
	var reconciled protocol.GrubberyAppOperation
	unknownStep, done, err := original.Record.Current()
	g02Check(t, err)
	if done {
		t.Fatal("original must remain unfinished")
	}
	for deadline := time.Now().Add(5 * time.Minute); ; {
		invoke("cli", "app.reconcile-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: original.Record.Plan.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: original.Record.Plan.Digest, StepID: unknownStep.ID}}, &reconciled)
		g02Check(t, reconciled.Validate())
		if len(reconciled.Record.Events) < len(original.Record.Events) || !reflect.DeepEqual(reconciled.Record.Events[:len(original.Record.Events)], original.Record.Events) {
			t.Fatal("reconciliation changed original prefix")
		}
		state, _ := reconciled.Record.State(unknownStep.ID)
		if state == "observed" {
			break
		}
		if state == "refused" || time.Now().After(deadline) || ctx.Err() != nil {
			t.Fatal("original unknown effect remains unconfirmed", state)
		}
		time.Sleep(time.Second)
	}
	save("unknown-phase-observed", reconciled)
	finish := func(adapter string, p grubbery.Plan) grubbery.Record {
		t.Helper()
		for calls := 0; calls < 64; calls++ {
			saved, e := c.GrubberyAppOperation(ctx, entry.Identity(), p.RequestID)
			g02Check(t, e)
			step, done, e := saved.Record.Current()
			g02Check(t, e)
			if done {
				return saved.Record
			}
			state, e := saved.Record.State(step.ID)
			g02Check(t, e)
			action := "reconcile"
			if state == "pending" {
				action = "advance"
			}
			var reply protocol.GrubberyAppOperation
			invoke(adapter, "app."+action+"-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: p.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: p.Digest, StepID: step.ID}}, &reply)
			g02Check(t, reply.Validate())
			save("latest-source-"+p.RequestID, reply)
			if state, _ = reply.Record.State(step.ID); state == "refused" {
				t.Fatal("original source phase explicitly refused", step.Kind)
			}
		}
		t.Fatal("bounded source progression unresolved")
		return grubbery.Record{}
	}
	first := finish("cli", original.Record.Plan)
	save("original-source-completed", first)
	if appVMState(t, ctx, registry, entry, fixture, first, false, false, "initial") != "initial" {
		t.Fatal("initial recovered app state differs")
	}
	appVMState(t, ctx, registry, entry, fixture, first, true, false, "runtime-recovery-saved")
	sample, err := appdev.ReadSample("grubbery-state-v2")
	g02Check(t, err)
	source := filepath.Join(fixture.Root, "source-v2")
	g02Check(t, os.Mkdir(source, 0700))
	for _, file := range sample.Files {
		path := filepath.Join(source, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(path), 0700))
		g02Check(t, os.WriteFile(path, []byte(file.Content), 0600))
	}
	stage, err := registry.PlanDeskStage(ctx, "fixture", entry.RunnerID, fixture.Instance, b.Target.Desk, source)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "fixture", entry.RunnerID, stage, source)
	g02Check(t, err)
	fresh, err := registry.InspectDesk(ctx, "fixture", entry.RunnerID, fixture.Instance, b.Target.Desk)
	g02Check(t, err)
	var secondReview protocol.GrubberyAppReview
	invoke("mcp", "app.review-source.v1", service.AppSourceReviewInput{Target: target, Request: protocol.GrubberyAppReviewRequest{Selection: fresh.Observation.Selection, FrameworkReceiptID: b.FrameworkReceiptID, AppID: b.AppID, StageID: stage.Intent.ID, RequestID: stopEvidenceDigest([]byte("runtime-recovery-second-source-v2"))}}, &secondReview)
	g02Check(t, secondReview.Validate())
	var begun protocol.GrubberyAppOperation
	invoke("mcp", "app.begin-source.v1", service.AppSourceBeginInput{Target: target, Request: protocol.GrubberyAppBeginRequest{Plan: secondReview.Plan}}, &begun)
	second := finish("mcp", secondReview.Plan)
	if appVMState(t, ctx, registry, entry, fixture, second, false, false, "runtime-recovery-saved") != "runtime-recovery-saved" || appVMState(t, ctx, registry, entry, fixture, second, false, true, "public-v2") != "public-v2" {
		t.Fatal("second version lost state or new code")
	}
	save("second-source-completed", second)
	lifecycle("stop", "runtime-recovery-second-stop")
	lifecycle("start", "runtime-recovery-second-start")
	replacement := waitRunning()
	if replacement.Resources.ContainerID == current.Resources.ContainerID {
		t.Fatal("second supported restart did not replace process")
	}
	if appVMState(t, ctx, registry, entry, fixture, second, false, false, "runtime-recovery-saved") != "runtime-recovery-saved" || appVMState(t, ctx, registry, entry, fixture, second, false, true, "public-v2") != "public-v2" {
		t.Fatal("second same-pier restart lost state/code")
	}
	appVMState(t, ctx, registry, entry, fixture, second, true, false, "runtime-after-second-restart")
	g02Check(t, exec.CommandContext(ctx, "systemctl", "--user", "restart", "urbit-runner.service").Run())
	if waitRunning().Resources.ContainerID != replacement.Resources.ContainerID {
		t.Fatal("manager restart replaced the native process")
	}
	var auditReplay protocol.GrubberyRuntimeRecoveryStatus
	invoke("cli", "app.runtime-recovery.v1", service.AppRuntimeStatusInput{Target: target, OriginalRequestID: original.Record.Plan.RequestID, AuditRequestID: auditID}, &auditReplay)
	if auditReplay.Receipt == nil || !reflect.DeepEqual(auditReplay.Receipt, audit.Receipt) {
		t.Fatal("saved historical audit changed")
	}
	for _, record := range []grubbery.Record{first, second} {
		var replay protocol.GrubberyAppOperation
		invoke("cli", "app.advance-source.v1", service.AppSourcePhaseInput{Target: target, OperationID: record.Plan.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: record.Plan.Digest, StepID: record.Plan.Steps[0].ID}}, &replay)
		if !reflect.DeepEqual(replay.Record, record) {
			t.Fatal("original completed replay changed")
		}
		for _, step := range record.Plan.Steps {
			dispatches := 0
			for _, event := range record.Events {
				if event.StepID == step.ID && event.State == "dispatched" {
					dispatches++
				}
			}
			if dispatches != 1 {
				t.Fatal("not one original durable dispatch", step.Kind, dispatches)
			}
		}
	}
	if appVMState(t, ctx, registry, entry, fixture, second, false, false, "runtime-after-second-restart") != "runtime-after-second-restart" {
		t.Fatal("state changed after manager replacement")
	}
	save("qualification", map[string]any{"audited_replacement": true, "unknown_phase_observed_without_redispatch": true, "original_source_completed": true, "second_source_state_and_code": true, "second_same_pier_replacement_state": true, "manager_restart_audit_and_original_replay": true, "one_durable_dispatch_per_original_phase": true, "boundary": "fixed disposable pending app only; no full permits/routes/assets/peers, process-death auth cleanup, installed upgrade or DEV/production qualification"})
}
