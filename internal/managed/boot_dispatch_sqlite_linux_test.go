//go:build linux && amd64

package managed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

// A test-only child hosts the real runtime driver and its existing isolated
// engine fixtures. Each callback synchronously runs this package's runtimeUpdate
// and each returned result goes through finish. This avoids production test hooks
// and the managed -> runtimequalify -> managed test import cycle.
type bootDispatchWire struct {
	Kind        string                            `json:"kind"`
	Action      string                            `json:"action,omitempty"`
	Cut         string                            `json:"cut,omitempty"`
	Directory   string                            `json:"directory,omitempty"`
	State       runtimequalify.ManagedState       `json:"state"`
	Observation runtimequalify.ManagedObservation `json:"observation"`
	Error       string                            `json:"error,omitempty"`
	Events      []string                          `json:"events,omitempty"`
}

type bootDispatchProcess struct {
	t         *testing.T
	encode    *json.Encoder
	decode    *json.Decoder
	directory string
}

func buildBootDispatchWorker(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "runtime-driver.test")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-c", "-race", "-o", binary, "../runtimequalify")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile isolated runtime fixture: %v\n%s", err, out)
	}
	return binary
}

func startBootDispatchWorker(t *testing.T, binary, mode string) *bootDispatchProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cmd := exec.CommandContext(ctx, binary, "-test.run=^TestBootSQLiteDriverWorker$", "-test.timeout=55s")
	cmd.Env = append(os.Environ(), "OMABIT_TEST_BOOT_SQLITE_WORKER="+mode)
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	p := &bootDispatchProcess{t: t, encode: json.NewEncoder(input), decode: json.NewDecoder(output)}
	t.Cleanup(func() {
		if t.Failed() {
			cancel()
		} else {
			_ = p.encode.Encode(bootDispatchWire{Kind: "close"})
		}
		_ = input.Close()
		if err := cmd.Wait(); err != nil && !t.Failed() {
			t.Error("isolated runtime fixture failed", err)
		}
		cancel()
	})
	ready := p.receive()
	if ready.Kind != "ready" || ready.Directory == "" {
		t.Fatal("runtime fixture did not initialize")
	}
	p.directory = ready.Directory
	return p
}

func (p *bootDispatchProcess) receive() bootDispatchWire {
	p.t.Helper()
	var message bootDispatchWire
	if err := p.decode.Decode(&message); err != nil {
		p.t.Fatal("runtime fixture protocol", err)
	}
	return message
}

func (p *bootDispatchProcess) run(request bootDispatchWire, persist func(runtimequalify.ManagedState) string) bootDispatchWire {
	p.t.Helper()
	if err := p.encode.Encode(request); err != nil {
		p.t.Fatal(err)
	}
	for {
		message := p.receive()
		if message.Kind != "persist" {
			return message
		}
		if persist == nil {
			p.t.Fatal("unexpected runtime persistence callback")
		}
		if err := p.encode.Encode(bootDispatchWire{Kind: persist(message.State)}); err != nil {
			p.t.Fatal(err)
		}
	}
}

func bootDispatchRecord(state runtimequalify.ManagedState) (runtimequalify.ContainerRecord, bool) {
	for _, c := range state.Runtime.Containers {
		if c.Kind == "http-boot" || c.Kind == "comet-boot" {
			return c, true
		}
	}
	return runtimequalify.ContainerRecord{}, false
}

func TestBootDispatchSQLiteCompletionAndInterruptedReopen(t *testing.T) {
	binary := buildBootDispatchWorker(t)
	for _, mode := range []string{"fake", "comet", "owned"} {
		for _, cut := range []string{"journal-before", "journal-after", "crash-intent", "crash-dispatch", "ack-lost-crash", "create-response", "legacy-intent", "checkpoint-mismatch"} {
			if mode == "fake" && cut == "create-response" {
				continue
			}
			t.Run(mode+"/"+cut, func(t *testing.T) {
				worker := startBootDispatchWorker(t, binary, mode)
				s, err := testOpenStore(worker.directory)
				if err != nil {
					t.Fatal(err)
				}
				s.mode = DevelopmentMode
				t.Cleanup(func() { _ = s.close() })
				id := "fixture"
				if mode == "owned" {
					_, _, err = s.registerShip("actor", protocol.CreateShipRequest{Ship: "~sampel-palnet", RequestID: "register", Key: "0w" + strings.Repeat("a", 128), ConfirmNetwork: true})
					id = "sampel-palnet"
				} else {
					_, _, err = s.registerKind("actor", registration(id, "register", 18080), nil, mode == "comet")
				}
				if err != nil {
					t.Fatal(err)
				}
				load := func() storedInstance {
					t.Helper()
					v, err := loadInstance(s.db, id)
					if err != nil {
						t.Fatal(err)
					}
					return v
				}
				original := load()
				reopen := func() {
					t.Helper()
					if err := s.close(); err != nil {
						t.Fatal(err)
					}
					s, err = testOpenStore(worker.directory)
					if err != nil {
						t.Fatal(err)
					}
					s.mode = DevelopmentMode
					if err := s.reconcileInterrupted(); err != nil {
						t.Fatal(err)
					}
				}
				begin := func(action, request string) (protocol.Operation, storedInstance) {
					t.Helper()
					op, replayed, err := s.accept("actor", id, protocol.OperationRequest{Kind: action, RequestID: request, ExpectedRevision: 1})
					if err != nil || replayed {
						t.Fatal("operation admission", action, err)
					}
					op, v, err := s.beginOperation(op.ID)
					if err != nil {
						t.Fatal(err)
					}
					return op, v
				}
				update := func(op protocol.Operation, state runtimequalify.ManagedState) {
					t.Helper()
					if err := s.runtimeUpdate(id, op.ID, state); err != nil {
						t.Fatal("actual runtimeUpdate", err)
					}
				}
				finish := func(op protocol.Operation, result bootDispatchWire) {
					t.Helper()
					if result.Kind != "result" {
						t.Fatal("unexpected runtime death")
					}
					var cause error
					if result.Error != "" {
						cause = errors.New(result.Error)
					}
					if err := s.finish(op, result.State, result.Observation, cause, false); err != nil {
						t.Fatal("actual finish", err)
					}
				}
				apply := func(action, request string) bootDispatchWire {
					t.Helper()
					op, v := begin(action, request)
					result := worker.run(bootDispatchWire{Kind: "apply", Action: action, State: v.Runtime}, func(state runtimequalify.ManagedState) string { update(op, state); return "ok" })
					finish(op, result)
					return result
				}
				observe := func() bootDispatchWire {
					t.Helper()
					v := load()
					result := worker.run(bootDispatchWire{Kind: "observe", State: v.Runtime}, nil)
					state, observation := result.State, result.Observation
					if result.Error != "" {
						state = v.Runtime
						observation = runtimequalify.ManagedObservation{State: "unknown", Health: "unknown", RecoveryRequired: true}
					}
					if err := s.observe(v, state, observation); err != nil {
						t.Fatal("actual observation write", err)
					}
					return result
				}
				op, v := begin("start", "first-start")
				injected := false
				result := worker.run(bootDispatchWire{Kind: "apply", Action: "start", State: v.Runtime, Cut: cut}, func(state runtimequalify.ManagedState) string {
					record, exists := bootDispatchRecord(state)
					atIntent := exists && !record.CreateDispatched
					atDispatch := exists && record.CreateDispatched && record.CreateUncertain
					if !injected && atDispatch && cut == "journal-before" {
						injected = true
						return "error"
					}
					update(op, state)
					if !injected && (atIntent && (cut == "crash-intent" || cut == "legacy-intent" || cut == "checkpoint-mismatch") || atDispatch && cut == "crash-dispatch") {
						injected = true
						return "crash"
					}
					if !injected && atDispatch && (cut == "journal-after" || cut == "ack-lost-crash") {
						injected = true
						return "error"
					}
					return "ok"
				})
				if cut == "create-response" {
					if result.Error == "" {
						t.Fatal("actual create failure not reached")
					}
				} else if !injected {
					t.Fatal("journal boundary was not reached")
				}
				cleanReturn := cut == "journal-before" || cut == "journal-after" || cut == "ack-lost-crash"
				if cleanReturn {
					record, ok := bootDispatchRecord(result.State)
					if result.Kind != "result" || result.Error == "" || !ok || record.CreateDispatched || record.CreateUncertain || result.State.UncertainProvisioning() ||
						result.State.Comet != nil && result.State.Comet.StartedAt != nil || !result.State.UndispatchedInitialBoot() {
						t.Fatal("driver returned an inconsistent non-dispatch tuple for finish", result.Error)
					}
				}
				if cut == "journal-before" || cut == "journal-after" || cut == "create-response" {
					finish(op, result) // critical: persist RETURNED state, not the last callback
				}
				if cut == "legacy-intent" {
					changed := load().Runtime
					changed.Runtime.Containers[len(changed.Runtime.Containers)-1].EngineVersion = ""
					update(op, changed)
				}
				if cut == "checkpoint-mismatch" {
					if _, err := s.db.Exec("UPDATE operations SET runtime=? WHERE id=?", []byte(`{}`), op.ID); err != nil {
						t.Fatal(err)
					}
				}
				reopen()
				v = load()
				retryable := cut == "journal-before" || cut == "journal-after" || cut == "crash-intent"
				if retryable {
					if v.Runtime.UncertainProvisioning() || v.Runtime.Storage.Uncertain || !v.Public.RecoveryRequired || v.Public.ActiveOperationID != "" || !v.Runtime.UndispatchedInitialBoot() {
						t.Fatalf("finish/reconciliation invented a private fence: %+v", v.Runtime.Comet)
					}
					if observed := observe(); observed.Error == "" || !load().Public.RecoveryRequired {
						t.Fatal("observation bypassed explicit Stop of retained intent")
					}
					_, _, err := s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "blocked-before-stop", ExpectedRevision: 1})
					requireFault(t, err, "invalid_state")
					for round, request := range []string{"stop-preparation", "stop-preparation-again"} {
						if result := apply("stop", request); result.Error != "" {
							t.Fatal(result.Error)
						}
						reopen()
						if load().Runtime.UncertainProvisioning() {
							t.Fatal("Stop invented a provisioning fence")
						}
						if round == 0 {
							_, _, err := s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "blocked-before-observe", ExpectedRevision: 1})
							requireFault(t, err, "invalid_state")
						}
					}
					if result := observe(); result.Error != "" || result.Observation.RecoveryRequired || result.Observation.State != "uninitialized" {
						t.Fatal("fresh observation did not readmit preparation", result.Error)
					}
					if result := apply("start", "first-real-boot"); result.Error != "" || result.Observation.Health != "healthy" {
						t.Fatal("first real boot", result.Error)
					}
					reopen()
					booted := load()
					if booted.Runtime.ID != original.Runtime.ID || booted.Runtime.Storage.Path != original.Runtime.Storage.Path || booted.Runtime.NativeShip != original.Runtime.NativeShip || booted.Public.Ship == "" {
						t.Fatal("boot changed enrollment identity")
					}
					for _, request := range []string{"stop-ready", "stop-ready-again"} {
						if result := apply("stop", request); result.Error != "" {
							t.Fatal(result.Error)
						}
						reopen()
					}
					result := apply("start", "resume-without-bootstrap")
					if result.Error != "" || result.Observation.Health != "healthy" || load().Public.Ship != booted.Public.Ship || load().Runtime.Storage.Identity != booted.Runtime.Storage.Identity {
						t.Fatal("resume changed pier/ship identity", result.Error)
					}
					boots := 0
					for _, event := range result.Events {
						if event == "start-comet-boot" || event == "start-http-boot" {
							boots++
						}
					}
					if boots != 1 {
						t.Fatal("duplicate bootstrap after recovery", boots)
					}
				} else {
					if !v.Runtime.UncertainProvisioning() || !v.Public.RecoveryRequired {
						t.Fatal("ambiguous/dispatched/legacy evidence lost its fence")
					}
					for _, request := range []string{"stop-uncertain", "stop-uncertain-again"} {
						apply("stop", request)
						reopen()
						if !load().Runtime.UncertainProvisioning() || !load().Public.RecoveryRequired {
							t.Fatal("daemon absence or Stop cleared uncertain provisioning")
						}
					}
					observe()
					_, _, err := s.accept("actor", id, protocol.OperationRequest{Kind: "start", RequestID: "no-duplicate-bootstrap", ExpectedRevision: 1})
					requireFault(t, err, "recovery_required")
				}
			})
		}
	}
}
