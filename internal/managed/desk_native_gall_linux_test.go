//go:build linux && amd64

package managed

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/appdev"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/eyre"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

// A named, isolated native gate. Its lifecycle backend is synthetic; actual
// source commit, compilation, activation and observations use the owned kernel.
// App traffic is direct to Eyre. The supervisor owns only this capsule's restart.
func TestNativeGallSampleWorkflow(t *testing.T) {
	f := nativeDeskGuard(t)
	var guard struct {
		WorkflowCase string `json:"workflow_case"`
	}
	guardData, err := os.ReadFile("/guard.json")
	g02Check(t, err)
	g02Check(t, json.Unmarshal(guardData, &guard))
	if guard.WorkflowCase != "gall-state-v2" {
		t.Fatal("explicit named Gall capsule required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	var mountRequests atomic.Int32
	f.auditEffect = func(body []byte) error {
		if strings.Contains(string(body), "+hood/mount") || strings.Contains(string(body), "%kiln-mount") {
			mountRequests.Add(1)
			return errors.New("native Gall fixture refuses Unix mount effects")
		}
		digest := sha256.Sum256(body)
		entry, err := json.Marshal(map[string]any{"sha256": hex.EncodeToString(digest[:]), "bytes": len(body), "unix_mount": false})
		if err != nil {
			return err
		}
		file, err := os.OpenFile("/lab/gall-native-effect-inputs.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err = file.Write(append(entry, '\n')); err != nil {
			return err
		}
		return file.Sync()
	}
	g02Check(t, os.Mkdir("/lab/runner", 0700))
	s, err := testOpenStore("/lab/runner")
	g02Check(t, err)
	defer s.close()
	_, _, err = s.register("fixture", registration("alpha", "native-gall", 18081))
	g02Check(t, err)
	instance, err := loadInstance(s.db, "alpha")
	g02Check(t, err)
	instance.Public.ObservedState = "running"
	instance.Public.Resources = &protocol.ResourceMetrics{ContainerID: strings.Repeat("b", 64)}
	saveRecoveryFixture(t, s, instance)
	r := &Server{s: s, b: &nativeDeskBackend{&deskBackendFixture{}, f}, ctx: ctx, cancel: cancel, uid: uint32(os.Geteuid())}
	defer r.Close()
	socket := "/lab/runner.sock"
	listener, err := net.Listen("unix", socket)
	g02Check(t, err)
	g02Check(t, os.Chmod(socket, 0600))
	server := &http.Server{Handler: r, ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
		return context.WithValue(ctx, peerKey{}, uint32(os.Geteuid()))
	}}
	go server.Serve(listener)
	defer server.Close()
	registry, err := collection.Open("/lab/collector")
	g02Check(t, err)
	entry, err := registry.Enroll(ctx, "native", socket)
	g02Check(t, err)
	proof := map[string]any{"boundary": "native kernel and typed API/store/staging; synthetic lifecycle; no Unix mount"}
	save := func(name string, value any) {
		t.Helper()
		proof[name] = value
		data, err := json.MarshalIndent(proof, "", "  ")
		g02Check(t, err)
		g02Check(t, os.WriteFile("/lab/gall-sample-proof.json", data, 0600))
		t.Log(name)
	}
	wait := func(id, want string) deskwork.Receipt {
		t.Helper()
		r.wg.Wait()
		op, err := registry.DeskOperation(ctx, "native", entry.RunnerID, id)
		g02Check(t, err)
		save("operation-"+id, op)
		if got := op.Receipt.Steps[len(op.Receipt.Steps)-1].State; got != want {
			t.Fatalf("native operation: want %s got %s", want, got)
		}
		return op.Receipt
	}
	preflight, err := registry.AppDevelopmentPreflight(ctx, appdev.PreflightRequest{Name: "native", RunnerID: entry.RunnerID, Workflow: "gall", Effects: []string{"create-desk", "commit"}})
	g02Check(t, err)
	save("environment-preflight", preflight)
	if !preflight.EnvironmentCompatible {
		t.Fatal("native adapter capabilities unavailable")
	}
	create := func(want string) {
		plan, err := registry.PlanDeskCreation(ctx, "native", entry.RunnerID, "alpha", f.Desk)
		g02Check(t, err)
		_, err = registry.CreateDesk(ctx, "native", entry.RunnerID, plan)
		g02Check(t, err)
		wait(plan.Intent.ID, want)
	}
	create("blocked")
	if f.creates.Load() != 0 {
		t.Fatal("default creation block dispatched")
	}
	target := apppolicy.Target{InstanceID: "alpha", Desk: f.Desk}
	policy, err := s.putAppPolicy("fixture-user", apppolicy.Update{Scope: "ship", Target: target.Ship(), Gates: apppolicy.Gates{CreateDesk: apppolicy.Allow, Commit: apppolicy.Inherit, Publish: apppolicy.Inherit}})
	g02Check(t, err)
	create("observed")
	policy, err = s.putAppPolicy("fixture-user", apppolicy.Update{ExpectedRevision: policy.Revision, Scope: "desk", Target: target, Gates: apppolicy.Gates{CreateDesk: apppolicy.Inherit, Commit: apppolicy.Allow, Publish: apppolicy.Block}})
	g02Check(t, err)
	preflight, err = registry.AppDevelopmentPreflight(ctx, appdev.PreflightRequest{Name: "native", RunnerID: entry.RunnerID, Workflow: "gall", InstanceID: "alpha", Desk: f.Desk, Effects: []string{"commit"}})
	g02Check(t, err)
	save("target-preflight", preflight)
	if !preflight.Compatible() {
		t.Fatal("target policy incompatible")
	}
	sample, err := appdev.ReadSample("gall-state-v2")
	g02Check(t, err)
	save("sample", sample.Definition)
	save("source-manifest", sample.Source)
	// An ordinary source directory below 8KiB, never registered with Vere's Unix vane. This
	// matches the fixture's configured source alias without any live source mount.
	source := filepath.Join(f.pier(), f.Desk)
	g02Check(t, os.Mkdir(source, 0700))
	for _, file := range sample.Files {
		path := filepath.Join(source, file.Path)
		g02Check(t, os.MkdirAll(filepath.Dir(path), 0700))
		g02Check(t, os.WriteFile(path, []byte(file.Content), 0600))
	}
	stage, err := registry.PlanDeskStage(ctx, "native", entry.RunnerID, "alpha", f.Desk, source)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "native", entry.RunnerID, stage, source)
	g02Check(t, err)
	plan, err := registry.PlanDeskCommit(ctx, "native", entry.RunnerID, stage.Intent.ID)
	g02Check(t, err)
	save("commit-plan", plan)
	_, err = registry.CommitDesk(ctx, "native", entry.RunnerID, plan)
	g02Check(t, err)
	wait(plan.Intent.Intent.ID, "observed")
	active, err := f.AutocommitActive(ctx)
	g02Check(t, err)
	if active {
		t.Fatal("unexpected live Unix mount")
	}
	release, err := registry.PlanDeskRelease(ctx, "native", entry.RunnerID, "alpha", f.Desk, "activate-local", []deskwork.ReleaseCheck{})
	g02Check(t, err)
	save("activation-plan", release)
	// Compile/mark-convert the exact constructor as stdout before any install
	// dispatch. This separates a pure native contract failure from an uncertain
	// side effect, without introducing arbitrary source or a retry path.
	preview, err := deskwork.ActivateLocal(f.ship(), f.Desk, release.Intent.Intent.Baseline)
	g02Check(t, err)
	previewBody, err := preview.Body()
	g02Check(t, err)
	var previewRequest map[string]json.RawMessage
	g02Check(t, json.Unmarshal(previewBody, &previewRequest))
	previewRequest["sink"] = json.RawMessage(`{"stdout":null}`)
	previewBody, err = json.Marshal(previewRequest)
	g02Check(t, err)
	converted, err := f.post(ctx, previewBody)
	save("activation-payload-preview", map[string]any{"succeeded": err == nil, "value": converted})
	g02Check(t, err)
	_, err = registry.ReleaseDesk(ctx, "native", entry.RunnerID, release)
	g02Check(t, err)
	r.wg.Wait()
	originalActivation, err := registry.DeskOperation(ctx, "native", entry.RunnerID, release.Intent.Intent.ID)
	g02Check(t, err)
	if originalActivation.Receipt.Steps[len(originalActivation.Receipt.Steps)-1].State == "uncertain" {
		save("activation-original-uncertain", originalActivation)
		application, err := f.Application(ctx, f.Desk)
		g02Check(t, err)
		save("activation-uncertain-application", application)
		for deadline := time.Now().Add(60 * time.Second); ; {
			_, err = registry.ReconcileDeskOperation(ctx, "native", entry.RunnerID, release.Intent.Intent.ID)
			g02Check(t, err)
			r.wg.Wait()
			current, err := registry.DeskOperation(ctx, "native", entry.RunnerID, release.Intent.Intent.ID)
			g02Check(t, err)
			save("activation-read-only-reconciliation", current)
			if current.Receipt.Steps[len(current.Receipt.Steps)-1].State == "observed" {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("activation remains uncertain after bounded read-only reconciliation")
			}
			time.Sleep(time.Second)
		}
	}
	wait(release.Intent.Intent.ID, "observed")
	// These are fixed, read-only probes of the exact committed sample. Capture
	// native compiler diagnostics separately from the asynchronous install effect.
	// They deliberately run after cold activation; they cannot warm its cache.
	for _, probe := range []struct{ name, expression string }{
		{"agent", ".^(vase %ca /" + f.ship() + "/" + f.Desk + "/(scot %da now)/app/omabit-state/hoon)"},
		{"bill", ".^((list dude:gall) %cx /" + f.ship() + "/" + f.Desk + "/(scot %da now)/desk/bill)"},
		{"mark", ".^(* %cb /" + f.ship() + "/" + f.Desk + "/(scot %da now)/omabit-entry)"},
	} {
		code := "=/  result  (mule |.(" + probe.expression + "))\n?:(?=(%& -.result) %compiled [%compile-error p.result])"
		body, err := json.Marshal(map[string]any{"source": map[string]any{"dojo": code}, "sink": map[string]any{"stdout": nil}})
		g02Check(t, err)
		value, err := f.post(ctx, body)
		save("compile-"+probe.name, map[string]any{"transport_succeeded": err == nil, "value": value})
		g02Check(t, err)
		if strings.TrimSpace(value) != "%compiled" {
			t.Fatal("sample native compilation failed; retained separate compiler evidence", probe.name)
		}
	}
	app, err := registry.InspectDeskApplication(ctx, "native", entry.RunnerID, "alpha", f.Desk)
	g02Check(t, err)
	save("application", app)
	if len(app.Observation.Application.ExpectedAgents) != 1 || app.Observation.Application.ExpectedAgents[0] != "omabit-state" {
		t.Fatal("sample expected agent inventory changed")
	}
	live := false
	for _, agent := range app.Observation.Application.Agents {
		if agent.Name == "omabit-state" && agent.Live {
			live = true
		}
	}
	if !live {
		t.Fatal("sample agent not live")
	}
	owner := nativeGallOwner(t, ctx, f)
	defer owner.Close()
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 8*time.Second)
		defer stop()
		if err := owner.Logout(cleanup); err != nil {
			t.Error("sample owner session cleanup unconfirmed")
		}
	}()
	want := nativeGallState{Revision: 0, Entries: []string{}}
	nativeGallRead(t, ctx, owner, want)
	save("initial-state", want)
	// The watch is acknowledged before the one-shot poke. Both the actual Gall
	// acknowledgement and expected changed fact must arrive; HTTP acceptance alone
	// never counts as success. The client deletes only its own channel afterward.
	facts := make(chan nativeGallState, 2)
	watchDone := make(chan error, 1)
	watchAck := make(chan struct{}, 1)
	watchExited := make(chan struct{})
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer func() {
		watchCancel()
		select {
		case <-watchExited:
		case <-time.After(10 * time.Second):
			t.Error("sample watch cleanup deadline")
		}
	}()
	go func() {
		defer close(watchExited)
		watchDone <- owner.Watch(watchCtx, "omabit-state", "/state", 2, func(event eyre.Event) error {
			if event.Kind == "event" && event.Response == "subscribe" {
				select {
				case watchAck <- struct{}{}:
				default:
				}
			}
			if event.Kind == "event" && event.Response == "diff" {
				var state nativeGallState
				if json.Unmarshal(event.Data, &state) != nil {
					return errors.New("invalid sample watch state")
				}
				facts <- state
			}
			return nil
		})
	}()
	nextFact := func(expected nativeGallState) {
		t.Helper()
		deadline := time.NewTimer(30 * time.Second)
		defer deadline.Stop()
		select {
		case state := <-facts:
			if !reflect.DeepEqual(state, expected) {
				t.Fatal("watch state differs")
			}
		case <-deadline.C:
			select {
			case err := <-watchDone:
				t.Fatalf("watch ended before expected fact: %v", err)
			default:
				t.Fatal("sample watch fact deadline")
			}
		case <-ctx.Done():
			t.Fatal("watch deadline")
		}
	}
	nextFact(want)
	select {
	case <-watchAck:
	case <-time.After(30 * time.Second):
		t.Fatal("subscription acknowledgement absent")
	}
	text := "omabit-native-v2-first"
	payload, err := json.Marshal(text)
	g02Check(t, err)
	save("poke-first", map[string]any{"state": "dispatched", "entry": text})
	ack, err := owner.Poke(ctx, "omabit-state", sample.Definition.PokeMark, payload)
	g02Check(t, err)
	if !ack.Acknowledged || !ack.CleanupConfirmed {
		t.Fatal("sample Gall acknowledgement or poke-channel cleanup absent")
	}
	want = nativeGallState{Revision: 1, Entries: []string{text}}
	nextFact(want)
	select {
	case err := <-watchDone:
		g02Check(t, err)
	case <-ctx.Done():
		t.Fatal("watch cleanup deadline")
	}
	save("poke-first", map[string]any{"state": "acknowledged", "entry": text, "channel_cleanup": true})
	save("watch-state", want)
	nativeGallRead(t, ctx, owner, want)
	ack, err = owner.Poke(ctx, "omabit-state", sample.Definition.PokeMark, payload)
	if !errors.Is(err, eyre.ErrNack) || ack.Acknowledged || !ack.CleanupConfirmed {
		t.Fatal("duplicate poke not definitively rejected with cleanup")
	}
	nativeGallRead(t, ctx, owner, want)
	save("duplicate-rejected-state-preserved", true)
	ack, err = owner.Poke(ctx, "omabit-state", sample.Definition.PokeMark, json.RawMessage(`{"unexpected":"object"}`))
	if !errors.Is(err, eyre.ErrNack) || ack.Acknowledged || !ack.CleanupConfirmed {
		t.Fatal("invalid JSON mark payload not definitively rejected")
	}
	nativeGallRead(t, ctx, owner, want)
	save("mark-rejection-state-preserved", true)
	// A unique on-load marker proves that changed code actually loaded, rather
	// than only observing a Clay revision or an old still-live agent.
	appPath := filepath.Join(source, "app/omabit-state.hoon")
	code, err := os.ReadFile(appPath)
	g02Check(t, err)
	before := string(code)
	needle := "++  on-load\n  |=  old=vase\n"
	if strings.Count(before, needle) != 1 {
		t.Fatal("frozen on-load shape changed")
	}
	marker := "omabit-native-gall-reloaded-v2"
	log, err := os.ReadFile("/lab/vere.log")
	g02Check(t, err)
	if strings.Contains(string(log), marker) {
		t.Fatal("reload marker already present")
	}
	changed := strings.Replace(before, needle, needle+"  ~&  %"+marker+"\n", 1)
	g02Check(t, os.WriteFile(appPath, []byte(changed), 0600))
	updateStage, err := registry.PlanDeskStage(ctx, "native", entry.RunnerID, "alpha", f.Desk, source)
	g02Check(t, err)
	_, err = registry.StageDesk(ctx, "native", entry.RunnerID, updateStage, source)
	g02Check(t, err)
	update, err := registry.PlanDeskCommit(ctx, "native", entry.RunnerID, updateStage.Intent.ID)
	g02Check(t, err)
	save("reload-commit-plan", update)
	_, err = registry.CommitDesk(ctx, "native", entry.RunnerID, update)
	g02Check(t, err)
	wait(update.Intent.Intent.ID, "observed")
	for deadline := time.Now().Add(90 * time.Second); ; {
		log, err = os.ReadFile("/lab/vere.log")
		g02Check(t, err)
		if strings.Contains(string(log), marker) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("new native on-load marker absent")
		}
		time.Sleep(200 * time.Millisecond)
	}
	nativeGallRead(t, ctx, owner, want)
	save("native-code-reload-state-preserved", want)
	// The supervisor binds this one restart to the exact owned PID/start/hash
	// and same filesystem object. It accepts no command, path or ship selection.
	restart := nativeGallRestart(t, ctx, f)
	save("same-pier-restart", restart)
	nativeGallRead(t, ctx, owner, want)
	save("restart-state-preserved-existing-session", want)
	var resumedFact nativeGallState
	err = owner.Watch(ctx, "omabit-state", "/state", 1, func(event eyre.Event) error {
		if event.Kind == "event" && event.Response == "diff" {
			return json.Unmarshal(event.Data, &resumedFact)
		}
		return nil
	})
	g02Check(t, err)
	if !reflect.DeepEqual(resumedFact, want) {
		t.Fatal("restart subscription state differs")
	}
	save("restart-subscription-cleanup", true)
	second := "omabit-native-v2-after-restart"
	payload, err = json.Marshal(second)
	g02Check(t, err)
	save("poke-after-restart", map[string]any{"state": "dispatched", "entry": second})
	ack, err = owner.Poke(ctx, "omabit-state", sample.Definition.PokeMark, payload)
	g02Check(t, err)
	if !ack.Acknowledged || !ack.CleanupConfirmed {
		t.Fatal("resumed app poke/cleanup absent")
	}
	want = nativeGallState{Revision: 2, Entries: []string{text, second}}
	nativeGallRead(t, ctx, owner, want)
	save("poke-after-restart", map[string]any{"state": "acknowledged", "entry": second, "channel_cleanup": true})
	save("final-state", want)
	active, err = f.AutocommitActive(ctx)
	g02Check(t, err)
	if active {
		t.Fatal("unexpected source autocommit")
	}
	if mountRequests.Load() != 0 {
		t.Fatal("unexpected attempted Unix mount")
	}
	var sourceBytes int64
	for _, file := range sample.Source.Files {
		sourceBytes += file.Bytes
	}
	save("source-lifecycle", map[string]any{"unix_mount_requests": mountRequests.Load(), "autocommit": false, "source_bytes": sourceBytes, "scope": "fresh private kernel; native effect inputs audited before dispatch"})
	save("qualification", "passed-native-deploy-json-read-poke-watch-reload-same-pier-restart; synthetic lifecycle backend")
}

type nativeGallState struct {
	Revision uint64   `json:"revision"`
	Entries  []string `json:"entries"`
}

func nativeGallOwner(t *testing.T, ctx context.Context, f *nativeDeskFixture) *eyre.Client {
	t.Helper()
	g02Check(t, f.check())
	port := f.HTTPPort
	if port == 0 {
		port = 18081
	}
	client, err := eyre.New(fmt.Sprintf("http://127.0.0.1:%d", port), f.ship())
	g02Check(t, err)
	request := []byte(`{"source":{"dojo":"+code"},"sink":{"stdout":null}}`)
	code, err := f.post(ctx, request)
	g02Check(t, err)
	err = client.Login(ctx, []byte(strings.TrimPrefix(strings.TrimSpace(code), "~")))
	code = ""
	g02Check(t, err)
	return client
}

func nativeGallRead(t *testing.T, ctx context.Context, client *eyre.Client, want nativeGallState) {
	t.Helper()
	data, err := client.Scry(ctx, "omabit-state", "/state")
	g02Check(t, err)
	var got nativeGallState
	g02Check(t, json.Unmarshal(data, &got))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native sample state differs")
	}
}

type nativeGallRestartReceipt struct {
	Version       int    `json:"version"`
	ID            string `json:"id"`
	State         string `json:"state"`
	OriginalPID   int    `json:"original_pid"`
	OriginalStart string `json:"original_start"`
	PID           int    `json:"pid"`
	Start         string `json:"start"`
	Exit          int    `json:"exit"`
	Device        uint64 `json:"device"`
	Inode         uint64 `json:"inode"`
	VereSHA       string `json:"vere_sha256"`
	RequestSHA    string `json:"request_sha256"`
}

func nativeGallRestart(t *testing.T, ctx context.Context, f *nativeDeskFixture) nativeGallRestartReceipt {
	t.Helper()
	g02Check(t, f.check())
	var nonce [16]byte
	_, err := rand.Read(nonce[:])
	g02Check(t, err)
	info, err := os.Lstat(f.pier())
	g02Check(t, err)
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("owned pier no longer an ordinary directory")
	}
	identity := info.Sys().(*syscall.Stat_t)
	id := hex.EncodeToString(nonce[:])
	request := map[string]any{"version": 1, "id": id, "action": "restart", "pid": f.PID, "start": f.Start, "vere_sha256": f.VereSHA, "device": uint64(identity.Dev), "inode": identity.Ino}
	body, err := json.Marshal(request)
	g02Check(t, err)
	digest := sha256.Sum256(body)
	requestSHA := hex.EncodeToString(digest[:])
	g02Check(t, os.WriteFile("/lab/gall-control-request.tmp", body, 0600))
	g02Check(t, os.Rename("/lab/gall-control-request.tmp", "/lab/gall-control-request.json"))
	end := time.Now().Add(190 * time.Second)
	for {
		data, err := os.ReadFile("/lab/gall-control-receipt.json")
		if err == nil {
			var receipt nativeGallRestartReceipt
			g02Check(t, json.Unmarshal(data, &receipt))
			if receipt.ID != id || receipt.Version != 1 || receipt.RequestSHA != requestSHA {
				t.Fatal("owned restart receipt identity differs")
			}
			if receipt.State == "succeeded" {
				if receipt.OriginalPID != f.PID || receipt.OriginalStart != f.Start || receipt.PID == f.PID || receipt.PID < 2 || receipt.Start == "" || receipt.Exit != 0 || receipt.Device != uint64(identity.Dev) || receipt.Inode != identity.Ino || receipt.VereSHA != f.VereSHA {
					t.Fatal("owned restart changed its binding")
				}
				after, err := os.Stat(f.pier())
				g02Check(t, err)
				current := after.Sys().(*syscall.Stat_t)
				if current.Dev != identity.Dev || current.Ino != identity.Ino {
					t.Fatal("same-pier restart replaced storage")
				}
				// No native operation or guard check remains in flight here. Only
				// this fixture binding follows the supervisor's new owned process;
				// production container/lifecycle behavior is not established by it.
				f.PID, f.Start = receipt.PID, receipt.Start
				g02Check(t, f.check())
				return receipt
			}
			if receipt.State != "stop-dispatched" && receipt.State != "restart-dispatched" {
				t.Fatal("owned restart did not complete cleanly")
			}
		} else if !os.IsNotExist(err) {
			g02Check(t, err)
		}
		if time.Now().After(end) || ctx.Err() != nil {
			t.Fatal("owned restart receipt deadline")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
