//go:build linux && amd64 && omabit_private_gateway_vm

package managed

// This explicitly tagged fixture is compiled into a separate test executable.
// It is never part of the runner or an ordinary unit-test run. Every entry first
// requires the fresh no-NIC QEMU guest; no host path is accepted as a parameter.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimequalify"
)

const privateVMRoot = "/home/lab/.local/share/urbit-runner/managed"
const privateVMManifest = "fd4c0739be071ef2ad88fb10bcd17cbc93c712e6dc7544ca8a2c9f0232f7832a"
const privateVMFoundation = "17a7d5fc3fb8e9314f5d6e806cad3e4ef0654388c87920ff729b48624e846410"
const privateVMActor = "qualification:private-initializer"
const privateVMObserver = "qualification:private-observer"

func privateVMGuard() (string, error) {
	bad := errors.New("private managed gateway fixture requires its fresh no-NIC QEMU guest")
	if os.Getenv("OMABIT_PRIVATE_MANAGED_GATEWAY") != "guest-v1" || os.Getuid() != 1000 || os.Geteuid() != 1000 || os.Getenv("HOME") != "/home/lab" || os.Getenv("XDG_RUNTIME_DIR") != "/run/user/1000" {
		return "", bad
	}
	mode := os.Getenv("OMABIT_PRIVATE_MANAGED_STAGE")
	if mode != "initialize" && mode != "recover" && mode != "verify" {
		return "", bad
	}
	for path, expected := range map[string]string{
		"/opt/omabit-lab/guest-marker": "offline-import-guest-v1",
		"/sys/class/dmi/id/sys_vendor": "QEMU",
		"/proc/1/comm":                 "systemd",
	} {
		raw, err := os.ReadFile(path)
		if err != nil || strings.TrimSpace(string(raw)) != expected {
			return "", bad
		}
	}
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil || len(entries) != 1 || entries[0].Name() != "lo" {
		return "", bad
	}
	if home, err := os.UserHomeDir(); err != nil || home != "/home/lab" {
		return "", bad
	}
	return mode, nil
}

func privateVMCheck(t *testing.T, err error, stage string) {
	t.Helper()
	if err != nil {
		// Actual engine/program errors remain in the private guest's diagnostics.
		// Avoid emitting command arguments, bootstrap output or key material.
		t.Fatal("private managed gateway fixture failed at " + stage)
	}
}

func privateVMHash(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	privateVMCheck(t, err, "encode-public-binding")
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func privateVMWrite(t *testing.T, name string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	privateVMCheck(t, err, "encode-receipt")
	f, err := os.OpenFile(filepath.Join(privateVMRoot, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	privateVMCheck(t, err, "exclusive-receipt")
	_, err = f.Write(raw)
	err = errors.Join(err, f.Sync(), f.Close())
	privateVMCheck(t, err, "sync-receipt")
	d, err := os.Open(privateVMRoot)
	privateVMCheck(t, err, "open-receipt-parent")
	privateVMCheck(t, errors.Join(d.Sync(), d.Close()), "sync-receipt-parent")
}

type privateVMProof struct {
	Version      int                             `json:"version"`
	ProjectID    string                          `json:"project_id"`
	ProjectSHA   string                          `json:"project_sha256"`
	WorkspaceSHA string                          `json:"workspace_sha256"`
	GatewaySHA   string                          `json:"gateway_sha256"`
	OperationSHA string                          `json:"operation_sha256"`
	Observation  privatechain.GatewayObservation `json:"initial_observation"`
	RecoveryID   string                          `json:"recovery_id,omitempty"`
	RecoverySHA  string                          `json:"recovery_sha256,omitempty"`
	ReadySHA     string                          `json:"ready_sha256,omitempty"`
}

func privateVMReadProof(t *testing.T, mode string) privateVMProof {
	t.Helper()
	f, err := os.Open(filepath.Join(privateVMRoot, "qualification-private-gateway-"+mode+".json"))
	privateVMCheck(t, err, "open-proof")
	raw, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	privateVMCheck(t, errors.Join(err, f.Close()), "read-proof")
	if len(raw) > 256*1024 {
		t.Fatal("proof exceeds bound")
	}
	var proof privateVMProof
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	privateVMCheck(t, d.Decode(&proof), "decode-proof")
	if err := d.Decode(new(any)); err != io.EOF || proof.Version != 1 || !protocol.ValidID(proof.ProjectID) {
		t.Fatal("private proof differs")
	}
	return proof
}

// Only the observation reply is dropped. Every factory and runtime operation
// delegates to the real current-source backend without replacing its dependencies.
type privateVMLostReadinessBackend struct {
	realBackend
	observed chan privatechain.GatewayObservation
}

func (b *privateVMLostReadinessBackend) OpenPrivateGateway(ctx context.Context, root string, p privatechain.Plan, w privatechain.Workspace, g privatechain.Gateway) (privateGatewayDriver, error) {
	d, err := b.realBackend.OpenPrivateGateway(ctx, root, p, w, g)
	if err != nil {
		return nil, err
	}
	return &privateVMLostReadinessDriver{privateGatewayDriver: d, observed: b.observed}, nil
}

type privateVMLostReadinessDriver struct {
	privateGatewayDriver
	observed chan privatechain.GatewayObservation
}

func (d *privateVMLostReadinessDriver) ObserveGateway(ctx context.Context, g privatechain.Gateway) (privatechain.GatewayObservation, error) {
	o, err := d.privateGatewayDriver.ObserveGateway(ctx, g)
	if err != nil {
		return o, err
	}
	select {
	case d.observed <- o.Clone():
	default:
		return privatechain.GatewayObservation{}, errors.New("readiness was observed more than once")
	}
	return privatechain.GatewayObservation{}, errors.New("qualification dropped the successful readiness reply")
}

func privateVMWait(t *testing.T, r *Server, id string) protocol.Operation {
	t.Helper()
	deadline := time.NewTimer(12 * time.Minute)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		op, err := loadOperation(r.s.db, id)
		privateVMCheck(t, err, "operation-poll")
		r.mu.Lock()
		workers := r.updateWorkers
		r.mu.Unlock()
		if op.Status != "queued" && op.Status != "running" && workers == 0 {
			if r.failed.Load() {
				t.Fatal("managed storage failed")
			}
			return op
		}
		select {
		case <-deadline.C:
			t.Fatal("private operation deadline expired; preserve guest state")
		case <-ticker.C:
		}
	}
}

func privateVMInitialize(t *testing.T, r *Server, b *privateVMLostReadinessBackend) {
	t.Helper()
	for _, table := range []string{"instances", "projects", "private_projects", "private_project_workspaces", "private_project_gateways"} {
		var count int
		privateVMCheck(t, r.s.db.QueryRow("SELECT count(*) FROM "+table).Scan(&count), "fresh-store")
		if count != 0 {
			t.Fatal("initialization requires a fresh fixture store")
		}
	}
	path := filepath.Join(privateVMRoot, "private-runtime-inputs", privateVMManifest, "manifest.json")
	f, err := os.Open(path)
	privateVMCheck(t, err, "open-manifest")
	raw, err := io.ReadAll(io.LimitReader(f, privatechain.MaxRuntimeInputManifest+1))
	privateVMCheck(t, errors.Join(err, f.Close()), "read-manifest")
	m, err := privatechain.DecodeRuntimeInputs(raw, privateVMManifest, privateVMFoundation)
	privateVMCheck(t, err, "reviewed-manifest")
	plan, err := privatechain.Build(privatechain.Request{Label: "Managed private VM qualification", Foundation: m.Foundation, ConfirmSharedAdministration: true, Members: []privatechain.MemberRequest{{InstanceID: "managed-vm-zod", Ship: "~zod", HTTPPort: 18381}, {InstanceID: "managed-vm-nec", Ship: "~nec", HTTPPort: 18382}}}, privatechain.Reservations{})
	privateVMCheck(t, err, "plan")
	digest, err := plan.Digest()
	privateVMCheck(t, err, "plan-digest")
	registered, hit, err := r.s.registerPrivateProject(privateVMActor, privateProjectRegistration{RequestID: "managed-vm-register", Plan: plan, ReviewDigest: digest})
	privateVMCheck(t, err, "registration")
	if hit {
		t.Fatal("fresh registration replayed")
	}
	project := registered.ProjectID
	wr := privateWorkspaceRequest{RequestID: "managed-vm-workspace", ExpectedRevision: 1, ReviewDigest: digest}
	op, hit, err := r.queuePrivateWorkspace(privateVMActor, project, wr)
	privateVMCheck(t, err, "workspace-queue")
	if hit || privateVMWait(t, r, op.ID).Status != "succeeded" {
		t.Fatal("fresh workspace did not complete")
	}
	w, err := readyPrivateWorkspace(r.s.db, project)
	privateVMCheck(t, err, "workspace-ready")
	engine, inputs, err := b.realBackend.InspectPrivateGatewayInputs(r.ctx, privateVMRoot, plan, w, privateVMManifest)
	privateVMCheck(t, err, "gateway-preview-inputs")
	preview, err := privatechain.NewGateway(plan, w, "managed-vm-preview", engine, inputs)
	privateVMCheck(t, err, "gateway-preview")
	gr := privateGatewayRequest{RequestID: "managed-vm-gateway", ExpectedRevision: 1, ReviewDigest: digest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: preview.Selection.WorkspaceDigest, InputManifestSHA256: privateVMManifest}
	op, hit, err = r.queuePrivateGateway(privateVMActor, project, gr)
	privateVMCheck(t, err, "gateway-queue")
	if hit {
		t.Fatal("fresh gateway replayed")
	}
	op = privateVMWait(t, r, op.ID)
	g, err := loadPrivateGateway(r.s.db, project)
	privateVMCheck(t, err, "gateway-source")
	if op.Status != "interrupted" || g.Phase != "started" || g.NetworkID == "" || g.ContainerID == "" {
		t.Fatal("lost readiness did not retain exact started resources")
	}
	var observed privatechain.GatewayObservation
	select {
	case observed = <-b.observed:
	default:
		t.Fatal("real backend never reported successful readiness")
	}
	ready := g.Clone()
	ready.Phase, ready.Observation = "ready", &observed
	privateVMCheck(t, ready.Validate(plan, w), "real-readiness")
	p, err := loadPrivateProject(r.s.db, project)
	privateVMCheck(t, err, "project-source")
	proof := privateVMProof{Version: 1, ProjectID: project, ProjectSHA: privateVMHash(t, p), WorkspaceSHA: privateVMHash(t, w), GatewaySHA: privateVMHash(t, g), OperationSHA: privateVMHash(t, op), Observation: observed}
	privateVMReplay(t, r, p, w, g, op)
	privateVMWrite(t, "qualification-private-gateway-initialize.json", proof)
}

func privateVMReplay(t *testing.T, r *Server, p storedPrivateProject, w privatechain.Workspace, g privatechain.Gateway, source protocol.Operation) {
	t.Helper()
	for _, only := range []bool{false, true} {
		wr := privateWorkspaceRequest{RequestID: "managed-vm-workspace", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, ReplayOnly: only}
		op, hit, err := r.queuePrivateWorkspace(privateVMActor, p.Binding.ProjectID, wr)
		privateVMCheck(t, err, "workspace-replay")
		if !hit || op.ID != w.OperationID || op.Status != "succeeded" {
			t.Fatal("workspace replay differs")
		}
		gr := privateGatewayRequest{RequestID: "managed-vm-gateway", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, WorkspaceOperationID: w.OperationID, WorkspaceDigest: g.Selection.WorkspaceDigest, InputManifestSHA256: privateVMManifest, ReplayOnly: only}
		op, hit, err = r.queuePrivateGateway(privateVMActor, p.Binding.ProjectID, gr)
		privateVMCheck(t, err, "gateway-replay")
		if !hit || !reflect.DeepEqual(op, source) {
			t.Fatal("gateway replay changed original receipt")
		}
	}
	after, err := loadPrivateGateway(r.s.db, p.Binding.ProjectID)
	privateVMCheck(t, err, "after-replay")
	if !reflect.DeepEqual(after, g) {
		t.Fatal("replay changed gateway")
	}
}

func privateVMRecover(t *testing.T, r *Server, mode string) {
	t.Helper()
	before := privateVMReadProof(t, "initialize")
	p, err := loadPrivateProject(r.s.db, before.ProjectID)
	privateVMCheck(t, err, "retained-project")
	w, err := readyPrivateWorkspace(r.s.db, before.ProjectID)
	privateVMCheck(t, err, "retained-workspace")
	g, err := loadPrivateGateway(r.s.db, before.ProjectID)
	privateVMCheck(t, err, "retained-gateway")
	source, err := loadOperation(r.s.db, g.Selection.OperationID)
	privateVMCheck(t, err, "retained-operation")
	if privateVMHash(t, p) != before.ProjectSHA || privateVMHash(t, w) != before.WorkspaceSHA || privateVMHash(t, g) != before.GatewaySHA || privateVMHash(t, source) != before.OperationSHA {
		t.Fatal("manager reopen changed original records")
	}
	privateVMReplay(t, r, p, w, g, source)
	req := privateGatewayObservationRequest{RequestID: "managed-vm-recover", ExpectedRevision: 1, ReviewDigest: w.PlanDigest, SourceOperationID: g.Selection.OperationID, SourceDigest: privateGatewaySourceDigest(g), ReplayOnly: mode == "verify"}
	op, hit, err := r.queuePrivateGatewayObservation(privateVMObserver, before.ProjectID, req)
	privateVMCheck(t, err, "recovery-queue")
	if hit != (mode == "verify") {
		t.Fatal("recovery replay status differs")
	}
	if !hit {
		op = privateVMWait(t, r, op.ID)
	}
	ready, receipt, err := readyPrivateGateway(r.s.db, before.ProjectID)
	privateVMCheck(t, err, "recovery-ready")
	if op.Status != "succeeded" || receipt.ID != op.ID || receipt.Actor != privateVMObserver || ready.ContainerID != g.ContainerID || ready.NetworkID != g.NetworkID {
		t.Fatal("recovery did not preserve resources and actor")
	}
	observed := ready.Observation.Clone()
	initial := before.Observation.Clone()
	observed.ObservedAt, initial.ObservedAt = time.Time{}, time.Time{}
	if !reflect.DeepEqual(observed, initial) {
		t.Fatal("read-only recovery changed initialized runtime evidence")
	}
	privateVMReplay(t, r, p, w, g, source)
	req.ReplayOnly = true
	again, hit, err := r.queuePrivateGatewayObservation(privateVMObserver, before.ProjectID, req)
	privateVMCheck(t, err, "recovery-replay")
	if !hit || !reflect.DeepEqual(again, op) {
		t.Fatal("recovery replay changed receipt")
	}
	wrong, hit, err := r.queuePrivateGatewayObservation(privateVMActor, before.ProjectID, req)
	var fault *fault
	if err == nil || !errors.As(err, &fault) || fault.problem.Code != "replay_not_found" || hit || wrong.ID != "" {
		t.Fatal("recovery receipt crossed actors")
	}
	if mode == "verify" {
		retained := privateVMReadProof(t, "recover")
		if retained.RecoveryID != op.ID || retained.RecoverySHA != privateVMHash(t, op) || retained.ReadySHA != privateVMHash(t, ready) {
			t.Fatal("second manager reopen changed recovery")
		}
	}
	before.RecoveryID, before.RecoverySHA, before.ReadySHA = op.ID, privateVMHash(t, op), privateVMHash(t, ready)
	privateVMWrite(t, "qualification-private-gateway-"+mode+".json", before)
}

func TestPrivateManagedGatewayVM(t *testing.T) {
	mode, err := privateVMGuard()
	if err != nil {
		t.Fatal("private managed gateway fixture refused outside its fresh no-NIC QEMU guest")
	}
	t.Log("private managed gateway guest admitted: " + mode)
	host, err := runtimequalify.OpenManagedHost()
	privateVMCheck(t, err, "owned-host-lock")
	defer host.Close()
	if host.Directory != privateVMRoot {
		t.Fatal("managed root differs")
	}
	privateVMWrite(t, "qualification-private-gateway-"+mode+".attempt.json", map[string]any{"stage": mode, "one_shot": true})
	lost := &privateVMLostReadinessBackend{observed: make(chan privatechain.GatewayObservation, 1)}
	var b backend = realBackend{}
	if mode == "initialize" {
		b = lost
	}
	r, err := newServerMode(privateVMRoot, b, 10*time.Second, true)
	privateVMCheck(t, err, "open-current-manager")
	defer func() { privateVMCheck(t, r.Close(), "close-manager") }()
	if mode == "initialize" {
		privateVMInitialize(t, r, lost)
	} else {
		privateVMRecover(t, r, mode)
	}
	t.Log("private managed gateway stage passed: " + mode)
}
