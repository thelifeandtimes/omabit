//go:build linux && amd64

package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/grubbery"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const appVMRoot = "/home/review/omabit-app-product"
const appVMInstance = "grubbery-zod"

// This helper uses the actual installed runner, CLI, SDK and stdio server. It
// injects no backend and exposes no production diagnostic/command interface.
func TestIssuesGrubberyProductDisposableVM(t *testing.T) {
	if os.Getenv("OMABIT_APP_PRODUCT_VM") == "" {
		t.Skip("explicit disposable product VM only")
	}
	vendor, e1 := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	cmdline, e2 := os.ReadFile("/proc/cmdline")
	raw, e3 := os.ReadFile("/opt/omabit-release-input/scenario.json")
	var scenario struct {
		Kind        string `json:"kind"`
		Source      string `json:"source_commit"`
		Case        string `json:"case"`
		Predecessor string `json:"predecessor_root_sha256"`
	}
	decoded := json.Unmarshal(raw, &scenario) == nil
	fixture, caseAdmitted := appVMFixtureForCase(scenario.Case)
	if scenario.Case == "cow-fresh-nec-v1" && scenario.Predecessor != "a23f10ddc89657489d8e55eaf5e4362603848366a447dbf9f51ce192837a08f9" {
		caseAdmitted = false
	}
	if e1 != nil || e2 != nil || e3 != nil || !decoded || !caseAdmitted || scenario.Kind != "issues18-product-grubbery-v1" || len(issuesVMSourceRevision) != 40 || scenario.Source != issuesVMSourceRevision || strings.TrimSpace(string(vendor)) != "QEMU" || !strings.Contains(string(cmdline), "omabit.disposable-guest=release-upgrade-v1") || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/review" || os.Getenv("OMABIT_APP_PRODUCT_VM") != fixture.Environment {
		t.Fatal("refused outside fresh explicit product VM")
	}
	appVMRoot, appVMInstance := fixture.Root, fixture.Instance
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
	defer cancel()
	if _, err := os.Lstat(appVMRoot); !os.IsNotExist(err) {
		t.Fatal("fresh helper output required", err)
	}
	g02Check(t, os.Mkdir(appVMRoot, 0700))
	proof := map[string]any{"scope": "actual installed runner Docker/curl source transport, CLI/stdio/SDK; fixed test-only state probes; fresh single fake; no complete permits/routes/assets/peer/upgrade qualification"}
	save := func(name string, value any) {
		t.Helper()
		proof[name] = value
		data, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile(filepath.Join(appVMRoot, "proof.json"), data, 0600))
		t.Log(name)
	}
	config := filepath.Join(appVMRoot, "registry")
	registry, err := collection.Open(config)
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "fixture", "/run/user/1000/urbit-runner/runner.sock")
	g02Check(t, err)
	c := entry.Client(true)
	defer c.Close()
	hello, err := c.Hello(ctx)
	g02Check(t, err)
	if hello.AccessRole != "operator" || entry.RunnerID != hello.RunnerID {
		t.Fatal("exact operator runner required")
	}
	var predecessor *protocol.GrubberyAppOperation
	if fixture.Ship == "~nec" {
		prior, e := c.GrubberyAppOperation(ctx, entry.Identity(), stopEvidenceDigest([]byte("product-source-request-v1")))
		g02Check(t, e)
		if prior.Record.Plan.Binding.Target.InstanceID != "grubbery-zod" || len(prior.Record.Events) != 17 {
			t.Fatal("wrong preserved pending predecessor")
		}
		predecessor = &prior
		snapshot, e := c.Snapshot(ctx)
		g02Check(t, e)
		if len(snapshot.Instances) != 1 || snapshot.Instances[0].ID != "grubbery-zod" {
			t.Fatal("new fixture requires exact old disposable inventory")
		}
		_, e = c.CreateFake(ctx, protocol.CreateFakeRequest{ID: fixture.Instance, Label: fixture.Instance, FakeShip: fixture.Ship, Config: protocol.Configuration{HTTPPort: fixture.Port, AmesPort: 34343, Loom: 31}, RequestID: "product-nec-create", Bootstrap: &protocol.Bootstrap{Kind: "recommended", Pill: &protocol.BootPill{SHA256: "f8b08c53dd6d0aa9c4f7654f0a7bcd11e94089222888a3c99c0a10eedcbb377b", Bytes: 217421172}}})
		g02Check(t, e)
		op, e := c.Operate(ctx, fixture.Instance, protocol.OperationRequest{Kind: "start", RequestID: "product-nec-initial-start", ExpectedRevision: 1})
		g02Check(t, e)
		for deadline := time.Now().Add(10 * time.Minute); ; {
			snapshot, e := c.Snapshot(ctx)
			g02Check(t, e)
			done := false
			for _, o := range snapshot.Operations {
				if o.ID == op.ID {
					if o.Status == "succeeded" {
						done = true
					} else if o.Status != "queued" && o.Status != "running" && o.Status != "pending" {
						t.Fatal("new fake Start failed", o.Status)
					}
				}
			}
			if done {
				break
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("new fake Start unresolved")
			}
			time.Sleep(time.Second)
		}
	}
	waitRunning := func() protocol.Instance {
		t.Helper()
		for deadline := time.Now().Add(time.Minute); ; {
			snapshot, e := c.Snapshot(ctx)
			g02Check(t, e)
			if v, ready := appVMRunningInstanceFor(snapshot, fixture.Instance); ready {
				return v
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("running container selection unresolved; no native app mutation attempted")
			}
			time.Sleep(time.Second)
		}
	}
	save("initial-running-selection", waitRunning())
	target := apppolicy.Target{InstanceID: appVMInstance, Desk: grubbery.FrameworkDesk}
	policy, err := c.AppPolicy(ctx, entry.Identity())
	g02Check(t, err)
	shipPolicy, err := c.SetAppPolicy(ctx, entry.Identity(), appVMPolicy(target, policy.Policy.Revision, "ship"))
	g02Check(t, err)
	_, err = c.SetAppPolicy(ctx, entry.Identity(), appVMPolicy(target, shipPolicy.Revision, "desk"))
	g02Check(t, err)
	waitDesk := func(id string) {
		t.Helper()
		for deadline := time.Now().Add(6 * time.Minute); ; {
			reply, e := registry.DeskOperation(ctx, "fixture", entry.RunnerID, id)
			g02Check(t, e)
			state := reply.Receipt.Steps[len(reply.Receipt.Steps)-1].State
			if state == "observed" {
				return
			}
			if state == "uncertain" || state == "interrupted" {
				_, e = registry.ReconcileDeskOperation(ctx, "fixture", entry.RunnerID, id)
				g02Check(t, e)
			} else if state != "dispatched" && state != "acknowledged" && state != "planned" {
				t.Fatal("native desk phase refused", state)
			}
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("native desk phase unresolved; original receipt retained", id, state)
			}
			time.Sleep(time.Second)
		}
	}
	framework, err := appdev.ReadFramework(appdev.GrubberyFramework)
	g02Check(t, err)
	frameworkRoot, err := os.OpenRoot("/opt/omabit-release-input/framework-source")
	g02Check(t, err)
	defer frameworkRoot.Close()
	manifest, err := deskwork.SnapshotRoot(frameworkRoot)
	g02Check(t, err)
	g02Check(t, framework.VerifySource(manifest))
	create, err := registry.PlanDeskCreation(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk)
	g02Check(t, err)
	save("framework-create-plan", create)
	_, err = registry.CreateDesk(ctx, "fixture", entry.RunnerID, create)
	g02Check(t, err)
	waitDesk(create.Intent.ID)
	inspected, err := registry.InspectDesk(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk)
	g02Check(t, err)
	exported, err := registry.ExportDeskSource(ctx, "fixture", entry.RunnerID, inspected.Observation, []string{"mar/hoon.hoon", "sys.kelvin"})
	g02Check(t, err)
	adoptionDir := filepath.Join(appVMRoot, "original-scaffold")
	g02Check(t, os.Mkdir(adoptionDir, 0700))
	for _, file := range exported.Files {
		p := filepath.Join(adoptionDir, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(p), 0700))
		data, e := file.Decode()
		g02Check(t, e)
		g02Check(t, os.WriteFile(p, data, 0600))
	}
	stage, err := registry.PlanDeskStage(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk, adoptionDir)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "fixture", entry.RunnerID, stage, adoptionDir)
	g02Check(t, err)
	adoption, err := registry.PlanDeskAdoption(ctx, "fixture", entry.RunnerID, stage.Intent.ID)
	g02Check(t, err)
	if adoption.Review.NewCount != 2 || adoption.Review.ConflictCount != 0 {
		t.Fatal("original scaffold adoption changed")
	}
	save("framework-adoption-plan", adoption)
	_, err = registry.AdoptDeskSource(ctx, "fixture", entry.RunnerID, adoption)
	g02Check(t, err)
	waitDesk(adoption.Intent.Intent.ID)
	frameworkStage, err := registry.PlanDeskStageRoot(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk, frameworkRoot)
	g02Check(t, err)
	_, err = registry.StageDeskRoot(ctx, "fixture", entry.RunnerID, frameworkStage, frameworkRoot)
	g02Check(t, err)
	commit, err := registry.PlanDeskCommit(ctx, "fixture", entry.RunnerID, frameworkStage.Intent.ID)
	g02Check(t, err)
	save("framework-commit-plan", commit)
	_, err = registry.CommitDesk(ctx, "fixture", entry.RunnerID, commit)
	g02Check(t, err)
	waitDesk(commit.Intent.Intent.ID)
	release, err := registry.PlanDeskRelease(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk, "activate-local", []deskwork.ReleaseCheck{})
	g02Check(t, err)
	save("framework-activation-plan", release)
	_, err = registry.ReleaseDesk(ctx, "fixture", entry.RunnerID, release)
	g02Check(t, err)
	waitDesk(release.Intent.Intent.ID)
	save("framework-active", true)
	stderr, err := os.OpenFile(filepath.Join(appVMRoot, "mcp-stderr.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	g02Check(t, err)
	defer stderr.Close()
	serverCommand := exec.CommandContext(ctx, "/opt/omabit-release-input/candidate/omabit", "--config-dir", config, "mcp", "serve", "--allow-runner", "fixture:"+entry.RunnerID, "--enable-mutations")
	serverCommand.Stderr = stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "omabit-product-qualification", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: serverCommand}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	g02Check(t, err)
	defer session.Close()
	invocation := 0
	invoke := func(adapter, id string, input, output any) {
		t.Helper()
		data, e := json.Marshal(input)
		g02Check(t, e)
		invocation++
		base := filepath.Join(appVMRoot, fmt.Sprintf("invocation-%03d-%s", invocation, id))
		g02Check(t, os.WriteFile(base+"-request.json", data, 0600))
		var reply []byte
		failed := false
		if adapter == "cli" {
			command := exec.CommandContext(ctx, "/opt/omabit-release-input/candidate/omabit", "--config-dir", config, "call", id, "--input-json", string(data))
			reply, e = command.Output()
			failed = e != nil
			if exit, ok := e.(*exec.ExitError); ok {
				// Output's ExitError keeps a bounded stderr prefix/suffix. Retain
				// it privately, not in the public serial/qualification result.
				g02Check(t, os.WriteFile(base+"-stderr.log", exit.Stderr, 0600))
			}
		} else {
			tool, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + id, Arguments: json.RawMessage(data)})
			g02Check(t, e)
			if len(tool.Content) != 1 {
				t.Fatal("typed product MCP call refused", id)
			}
			text, ok := tool.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatal("unexpected MCP content")
			}
			reply = []byte(text.Text)
			failed = tool.IsError
		}
		g02Check(t, os.WriteFile(base+"-response.json", reply, 0600))
		var envelope struct {
			Data  json.RawMessage  `json:"data"`
			Error *service.Problem `json:"error"`
		}
		g02Check(t, json.Unmarshal(reply, &envelope))
		if envelope.Error != nil {
			t.Fatal("typed product core call refused", id, envelope.Error.Code)
		}
		if failed {
			t.Fatal("typed product call failed; private response retained", id)
		}
		g02Check(t, json.Unmarshal(envelope.Data, output))
	}
	selected := service.Target{Name: "fixture", RunnerID: entry.RunnerID}
	var records []grubbery.Record
	for number, adapter := range []string{"cli", "mcp"} {
		sample, e := appdev.ReadSample(fmt.Sprintf("grubbery-state-v%d", number+1))
		g02Check(t, e)
		dir := filepath.Join(appVMRoot, sample.Definition.Name)
		g02Check(t, os.Mkdir(dir, 0700))
		for _, file := range sample.Files {
			p := filepath.Join(dir, file.Path)
			g02Check(t, os.MkdirAll(filepath.Dir(p), 0700))
			g02Check(t, os.WriteFile(p, []byte(file.Content), 0600))
		}
		plan, e := registry.PlanDeskStage(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk, dir)
		g02Check(t, e)
		_, e = registry.StageDesk(ctx, "fixture", entry.RunnerID, plan, dir)
		g02Check(t, e)
		fresh, e := registry.InspectDesk(ctx, "fixture", entry.RunnerID, appVMInstance, grubbery.FrameworkDesk)
		g02Check(t, e)
		requestID := stopEvidenceDigest([]byte(fmt.Sprintf("product-%s-source-request-v%d", fixture.Instance, number+1)))
		var review protocol.GrubberyAppReview
		invoke(adapter, "app.review-source.v1", service.AppSourceReviewInput{Target: selected, Request: protocol.GrubberyAppReviewRequest{Selection: fresh.Observation.Selection, FrameworkReceiptID: commit.Intent.Intent.ID, AppID: "product-state", StageID: plan.Intent.ID, RequestID: requestID}}, &review)
		g02Check(t, review.Validate())
		save(fmt.Sprintf("source-%d-plan", number+1), review.Plan)
		var reply protocol.GrubberyAppOperation
		invoke(adapter, "app.begin-source.v1", service.AppSourceBeginInput{Target: selected, Request: protocol.GrubberyAppBeginRequest{Plan: review.Plan}}, &reply)
		g02Check(t, reply.Validate())
		for index, step := range review.Plan.Steps {
			input := service.AppSourcePhaseInput{Target: selected, OperationID: requestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: review.Plan.Digest, StepID: step.ID}}
			invoke(adapter, "app.advance-source.v1", input, &reply)
			g02Check(t, reply.Validate())
			save(fmt.Sprintf("source-%d-phase-%02d", number+1, index), reply.Record)
			for deadline := time.Now().Add(3 * time.Minute); ; {
				state, e := reply.Record.State(step.ID)
				g02Check(t, e)
				if state == "observed" {
					break
				}
				if state == "refused" || time.Now().After(deadline) {
					t.Fatal("product native phase remains unresolved", step.Kind, state)
				}
				invoke(adapter, "app.reconcile-source.v1", input, &reply)
				g02Check(t, reply.Validate())
				time.Sleep(time.Second)
			}
		}
		if _, done, e := reply.Record.Current(); e != nil || !done {
			t.Fatal("product source not followed", e)
		}
		records = append(records, reply.Record)
		save(fmt.Sprintf("source-%d-followed", number+1), true)
		var cli, mcpReply protocol.GrubberyAppOperation
		lookup := service.DeskOperationInput{Target: selected, OperationID: requestID}
		invoke("cli", "app.source-operation.v1", lookup, &cli)
		invoke("mcp", "app.source-operation.v1", lookup, &mcpReply)
		if !reflect.DeepEqual(cli.Record, reply.Record) || !reflect.DeepEqual(cli, mcpReply) {
			t.Fatal("actual native CLI/MCP receipt parity changed")
		}
		// State probes are a fixed test-only diagnostic using the existing
		// operator code endpoint and an enabled exact local route. No credential
		// response/cookie is written to proof or process arguments.
		if number == 0 {
			if appVMState(t, ctx, registry, entry, fixture, reply.Record, false, false, "initial") != "initial" {
				t.Fatal("initial product state differs")
			}
			appVMState(t, ctx, registry, entry, fixture, reply.Record, true, false, "product-saved-entry")
		} else if appVMState(t, ctx, registry, entry, fixture, reply.Record, false, false, "product-saved-entry") != "product-saved-entry" || appVMState(t, ctx, registry, entry, fixture, reply.Record, false, true, "public-v2") != "public-v2" {
			t.Fatal("product update lost state or code label")
		}
	}
	save("two-source-state-and-cli-mcp", true)
	before, err := c.Snapshot(ctx)
	g02Check(t, err)
	originalContainer := ""
	for _, v := range before.Instances {
		if v.ID == appVMInstance && v.Resources != nil {
			originalContainer = v.Resources.ContainerID
		}
	}
	if originalContainer == "" {
		t.Fatal("exact running container unavailable")
	}
	for _, action := range []string{"stop", "start"} {
		snapshot, e := c.Snapshot(ctx)
		g02Check(t, e)
		var revision uint64
		for _, v := range snapshot.Instances {
			if v.ID == appVMInstance {
				revision = v.ConfigRevision
			}
		}
		op, e := c.Operate(ctx, appVMInstance, protocol.OperationRequest{Kind: action, RequestID: "product-" + fixture.Instance + "-" + action + "-same-pier", ExpectedRevision: revision})
		g02Check(t, e)
		for deadline := time.Now().Add(5 * time.Minute); ; {
			snapshot, e = c.Snapshot(ctx)
			g02Check(t, e)
			done := false
			for _, current := range snapshot.Operations {
				if current.ID == op.ID {
					if current.Status == "succeeded" {
						done = true
					} else if current.Status != "queued" && current.Status != "running" && current.Status != "pending" {
						t.Fatal("owned lifecycle operation failed; original receipt retained", action, current.Status)
					}
				}
			}
			if done {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("owned lifecycle operation unresolved", action)
			}
			time.Sleep(time.Second)
		}
	}
	replacement := waitRunning().Resources.ContainerID
	if replacement == "" || replacement == originalContainer {
		t.Fatal("supported restart did not replace its exact container")
	}
	last := records[len(records)-1]
	if appVMState(t, ctx, registry, entry, fixture, last, false, false, "product-saved-entry") != "product-saved-entry" || appVMState(t, ctx, registry, entry, fixture, last, false, true, "public-v2") != "public-v2" {
		t.Fatal("product same-pier replacement lost state/code")
	}
	appVMState(t, ctx, registry, entry, fixture, last, true, false, "product-after-restart")
	save("same-pier-new-container-state", true)
	command := exec.CommandContext(ctx, "systemctl", "--user", "restart", "urbit-runner.service")
	g02Check(t, command.Run())
	for _, original := range records {
		var replay protocol.GrubberyAppOperation
		invoke("cli", "app.advance-source.v1", service.AppSourcePhaseInput{Target: selected, OperationID: original.Plan.RequestID, Request: protocol.GrubberyAppPhaseRequest{PlanDigest: original.Plan.Digest, StepID: original.Plan.Steps[0].ID}}, &replay)
		if !reflect.DeepEqual(replay.Record, original) {
			t.Fatal("completed replay changed after real process/store restart")
		}
	}
	if appVMState(t, ctx, registry, entry, fixture, last, false, false, "product-after-restart") != "product-after-restart" {
		t.Fatal("state write did not survive manager restart")
	}
	if predecessor != nil {
		original, e := c.GrubberyAppOperation(ctx, entry.Identity(), predecessor.Record.Plan.RequestID)
		g02Check(t, e)
		if !reflect.DeepEqual(original, *predecessor) {
			t.Fatal("fresh peer changed the original pending app receipt")
		}
		save("original-pending-app-receipt-unchanged", true)
	}
	save("qualification", map[string]any{"actual_product_docker_transport": true, "native_cli_and_stdio": true, "two_source_versions": true, "state_and_new_code": true, "same_pier_container_replacement": true, "state_after_replacement": true, "manager_restart_original_replay": true, "boundary": "single fresh ordinary fake; fixed test-only state probes; no pending-phase rebinding, process-death auth cleanup, permits/alias/routes/assets/peers or installed upgrade"})
}

// Start completion precedes optional telemetry. Desk selection needs an actual
// container ID; a running row with an unavailable stats sample is not ready.
// Repeat only snapshot reads and never infer identity from an old sample.
func appVMRunningInstance(snapshot protocol.Snapshot) (protocol.Instance, bool) {
	return appVMRunningInstanceFor(snapshot, appVMInstance)
}
func appVMRunningInstanceFor(snapshot protocol.Snapshot, instance string) (protocol.Instance, bool) {
	var out protocol.Instance
	count := 0
	for _, v := range snapshot.Instances {
		if v.ID == instance {
			out = v
			count++
		}
	}
	if count != 1 || out.ProjectID != "" || out.ObservedState != "running" || out.ActiveOperationID != "" || out.RecoveryRequired || out.Resources == nil {
		return out, false
	}
	selection := protocol.DeskSelection{Target: apppolicy.Target{InstanceID: instance, Desk: grubbery.FrameworkDesk}, RuntimeID: out.Resources.ContainerID, ConfigRevision: out.ConfigRevision}
	return out, selection.Validate() == nil
}

func TestProductVMRunningSelectionWaitsForActualContainer(t *testing.T) {
	v := protocol.Instance{ID: appVMInstance, ConfigRevision: 1, ObservedState: "running", Resources: &protocol.ResourceMetrics{Status: "unavailable"}}
	snapshot := protocol.Snapshot{Instances: []protocol.Instance{v}}
	if _, ready := appVMRunningInstance(snapshot); ready {
		t.Fatal("Start completion without a container observation became ready")
	}
	v.Resources = &protocol.ResourceMetrics{Status: "fresh", ContainerID: strings.Repeat("a", 64)}
	snapshot.Instances[0] = v
	if got, ready := appVMRunningInstance(snapshot); !ready || got.Resources.ContainerID != v.Resources.ContainerID {
		t.Fatal("actual running selection refused")
	}
	for _, change := range []func(*protocol.Instance){
		func(v *protocol.Instance) { v.RecoveryRequired = true },
		func(v *protocol.Instance) { v.ActiveOperationID = "active" },
		func(v *protocol.Instance) { v.ObservedState = "starting" },
		func(v *protocol.Instance) { v.ProjectID = "unrelated" },
		func(v *protocol.Instance) { v.ConfigRevision = 0 },
		func(v *protocol.Instance) { v.Resources = nil },
	} {
		changed := v
		change(&changed)
		if _, ready := appVMRunningInstance(protocol.Snapshot{Instances: []protocol.Instance{changed}}); ready {
			t.Fatal("unresolved or different selection became ready")
		}
	}
	if _, ready := appVMRunningInstance(protocol.Snapshot{Instances: []protocol.Instance{v, v}}); ready {
		t.Fatal("duplicate target became ready")
	}
}

func appVMPolicy(target apppolicy.Target, revision uint64, scope string) apppolicy.Update {
	if scope == "ship" {
		return apppolicy.Update{ExpectedRevision: revision, Scope: "ship", Target: target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}}
	}
	return apppolicy.Update{ExpectedRevision: revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}}
}

func TestProductVMPolicyAuthoringUsesExactShipAndInheritedDeskCreation(t *testing.T) {
	target := apppolicy.Target{InstanceID: appVMInstance, Desk: grubbery.FrameworkDesk}
	ship, desk := appVMPolicy(target, 0, "ship"), appVMPolicy(target, 1, "desk")
	if ship.Validate() != nil || desk.Validate() != nil || ship.Target != target.Ship() || desk.Target != target || ship.Scope != "ship" || desk.Scope != "desk" || ship.Gates.CreateDesk != apppolicy.Allow || desk.Gates.CreateDesk != apppolicy.Inherit || desk.Gates.Commit != apppolicy.Allow || desk.Gates.Publish != apppolicy.Block {
		t.Fatal("product fixture permission scope changed")
	}
}

func appVMState(t *testing.T, ctx context.Context, r *collection.Registry, entry collection.Connection, fixture appVMFixture, record grubbery.Record, write, label bool, value string) string {
	t.Helper()
	routes, err := r.ShipRoutes(ctx, "fixture", entry.RunnerID, fixture.Instance)
	g02Check(t, err)
	for _, route := range routes.Registry.Routes {
		if !route.Enabled || route.Network != "local" || route.Scope != entry.RunnerID || (route.Origin != fmt.Sprintf("http://127.0.0.1:%d", fixture.Port) && route.Origin != fmt.Sprintf("http://localhost:%d", fixture.Port)) {
			continue
		}
		c := entry.Client(true)
		defer c.Close()
		secret, err := c.ShipRouteCode(ctx, entry.Identity(), protocol.ShipRouteCodeRequest{Selection: routes.Selection, Binding: routes.Record.Binding, RegistryRevision: routes.Registry.Revision, Route: route})
		g02Check(t, err)
		owner, err := eyre.New(route.Origin, fixture.Ship)
		g02Check(t, err)
		defer owner.Close()
		code := []byte(secret.Code)
		secret.Code = ""
		defer clear(code)
		g02Check(t, owner.Login(ctx, code))
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if owner.Logout(cleanup) != nil {
				t.Error("fixed product state session cleanup unconfirmed")
			}
		}()
		encoded, err := owner.ExportSession()
		g02Check(t, err)
		defer clear(encoded)
		var cookie struct {
			Name  string `json:"cookie_name"`
			Value string `json:"cookie_value"`
		}
		g02Check(t, json.Unmarshal(encoded, &cookie))
		defer func() { cookie.Value = "" }()
		n, err := record.Plan.Binding.Namespace()
		g02Check(t, err)
		method, path, body := "GET", "/grubbery/api/peek"+n.DataRoot+"/state/state.json", []byte{}
		if label {
			path = "/grubbery/api/peek" + n.DataRoot + "/state/label.json"
		}
		if write {
			method = "POST"
			path = "/grubbery/api/poke" + n.DataRoot + "/state/state.json?blot=%2Fjson"
			body, _ = json.Marshal(value)
		}

		client := http.Client{Timeout: 35 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		deadline := time.Now().Add(40 * time.Second)
		for {
			request, e := http.NewRequestWithContext(ctx, method, route.Origin+path, bytes.NewReader(body))
			g02Check(t, e)
			request.GetBody = nil
			request.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
			if write {
				request.Header.Set("Content-Type", "application/json")
			}
			response, e := client.Do(request)
			g02Check(t, e)
			data, e := io.ReadAll(io.LimitReader(response.Body, 4097))
			response.Body.Close()
			g02Check(t, e)
			if len(data) > 4096 {
				t.Fatal("fixed product state response bound")
			}
			if response.StatusCode == 200 {
				if write {
					return ""
				}
				var got string
				g02Check(t, json.Unmarshal(data, &got))
				if got == value {
					return got
				}
			} else if write || response.StatusCode != 404 {
				t.Fatal("fixed product state request refused")
			}
			// A source-follow marker need not mean the nexus is loaded yet.
			// Observe only; the test's state write is never repeated.
			if time.Now().After(deadline) || ctx.Err() != nil {
				t.Fatal("fixed product state observation deadline")
			}
			time.Sleep(time.Second)
		}

	}
	t.Fatal("enabled exact private local route unavailable")
	return ""
}

type appVMFixture struct {
	Root, Instance, Ship, Environment string
	Port                              uint16
}

func appVMFixtureForCase(name string) (appVMFixture, bool) {
	switch name {
	case "source-state-restart-v1":
		return appVMFixture{appVMRoot, appVMInstance, "~zod", "source-state-restart-v1", 19380}, true
	case "cow-fresh-nec-v1":
		return appVMFixture{"/home/review/omabit-app-product-nec", "grubbery-nec", "~nec", "source-state-restart-nec-v1", 19381}, true
	}
	return appVMFixture{}, false
}
func TestProductVMFixedCasesCannotReusePendingIdentityOrRequest(t *testing.T) {
	old, ok := appVMFixtureForCase("source-state-restart-v1")
	if !ok {
		t.Fatal("original fixture missing")
	}
	fresh, ok := appVMFixtureForCase("cow-fresh-nec-v1")
	if !ok || fresh.Root == old.Root || fresh.Instance == old.Instance || fresh.Ship == old.Ship || fresh.Port == old.Port || fresh.Environment == old.Environment {
		t.Fatal("fresh fixture aliases original pending work")
	}
	if _, ok = appVMFixtureForCase("custom-host-path"); ok {
		t.Fatal("arbitrary fixture override admitted")
	}
	v := protocol.Instance{ID: old.Instance, ConfigRevision: 1, ObservedState: "running", Resources: &protocol.ResourceMetrics{Status: "fresh", ContainerID: strings.Repeat("a", 64)}}
	if _, ok = appVMRunningInstanceFor(protocol.Snapshot{Instances: []protocol.Instance{v}}, fresh.Instance); ok {
		t.Fatal("old running identity selected for new fixture")
	}
}
