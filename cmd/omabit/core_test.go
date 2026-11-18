package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run the actual argument dispatcher with private stdio, without the Go test
// framework writing PASS lines into an MCP protocol stream.
func TestCoreAdapterProcess(t *testing.T) {
	if os.Getenv("OMABIT_CORE_TEST_PROCESS") != "1" {
		return
	}
	var args []string
	if json.Unmarshal([]byte(os.Getenv("OMABIT_CORE_TEST_ARGS")), &args) != nil {
		os.Exit(99)
	}
	os.Exit(run(context.Background(), args))
}
func coreTestCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-test.run=^TestCoreAdapterProcess$")
	cmd.Env = []string{"OMABIT_CORE_TEST_PROCESS=1", "OMABIT_CORE_TEST_ARGS=" + string(encoded), "HOME=" + t.TempDir(), "PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	return cmd
}
func TestCLIAndMCPUseSamePolicyAndRefusalContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	configDir := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(configDir)
	if err != nil {
		t.Fatal(err)
	}
	var observer, missing atomic.Bool
	var writes atomic.Int32
	h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Actor: "actor", Features: []string{protocol.AppPolicyFeature, protocol.ProjectRegistrationFeature, protocol.ProjectStopRecoveryFeature}}
	policy := protocol.AppPolicyStatus{Version: 1, RunnerID: h.RunnerID, Mode: h.Mode, ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{}}
	diagnosis := protocol.ProjectMemberDiagnosis{SchemaVersion: 1, RunnerID: h.RunnerID, Mode: h.Mode, Actor: h.Actor, ProjectID: "project", InstanceID: "member", Operation: protocol.Operation{ID: "operation", ProjectID: "project", InstanceID: "member", Kind: "stop-project-ship", RequestID: "private-request", Actor: "private cookie", Status: "interrupted", ConfigRevision: 1, CreatedAt: policy.ReadAt, Message: "private cookie must not escape"}, Phase: "stop-dispatched", Fence: "original-interrupted-operation", ReasonCode: "stop_needs_observation", Message: "private cookie must not escape", RuntimeObservation: "not-performed", ReadAt: policy.ReadAt}
	socket := filepath.Join(t.TempDir(), "r.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if q.URL.Path == "/hello" {
			hello := h
			if observer.Load() {
				hello.AccessRole = "observer"
			}
			if missing.Load() {
				hello.Features = []string{}
			}
			_ = json.NewEncoder(w).Encode(hello)
			return
		}
		if q.Method != "GET" {
			writes.Add(1)
			w.WriteHeader(403)
			_ = json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: protocol.APIError{Code: "policy_denied", Message: "private cookie must not escape"}})
			return
		}
		if q.URL.RequestURI() == protocol.ProjectMemberDiagnosisPath("project", "member", "operation", false) {
			_ = json.NewEncoder(w).Encode(diagnosis)
			return
		}
		if q.URL.Path != protocol.AppPolicyPath {
			http.NotFound(w, q)
			return
		}
		_ = json.NewEncoder(w).Encode(policy)
	})}
	go server.Serve(listener)
	defer server.Close()
	if _, err = registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	transport := &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", configDir, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations")}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "fixture", Version: "1"}, nil).Connect(ctx, transport, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "empty", Members: []projectnetwork.MemberRequest{}, ConfirmSharedAdministration: true}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	register := service.ProjectRegisterInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, Request: protocol.ProjectRegisterRequest{Label: "empty", Members: []protocol.ProjectMemberInput{}, ConfirmSharedAdministration: true, ReviewDigest: digest, RequestID: "original-request", ConfirmReservation: true}}
	for _, tc := range []struct {
		name, id          string
		input             any
		observer, missing bool
		code              string
	}{
		{name: "frozen-sample", id: "app.sample.v1", input: service.SampleInput{Name: "gall-state-v1"}},
		{name: "json-sample", id: "app.sample.v1", input: service.SampleInput{Name: "gall-state-v2"}},
		{name: "grubbery-first-source", id: "app.sample.v1", input: service.SampleInput{Name: "grubbery-state-v1"}},
		{name: "grubbery-next-source", id: "app.sample.v1", input: service.SampleInput{Name: "grubbery-state-v2"}},
		{name: "policy-read", id: "app.policy.v1", input: service.Target{Name: "dev", RunnerID: "runner"}},
		{name: "shareable-stop-diagnosis", id: "project.stop-diagnostic-bundle.v1", input: service.StopBundleInput{CheckpointInput: service.CheckpointInput{ProjectInput: service.ProjectInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, ProjectID: "project"}, OperationID: "operation"}, InstanceID: "member"}},
		{name: "shareable-stop-diagnosis-role", id: "project.stop-diagnostic-bundle.v1", input: service.StopBundleInput{CheckpointInput: service.CheckpointInput{ProjectInput: service.ProjectInput{Target: service.Target{Name: "dev", RunnerID: "runner"}, ProjectID: "project"}, OperationID: "operation"}, InstanceID: "member"}, observer: true, code: "operator_required"},
		{name: "capability", id: "app.policy.v1", input: service.Target{Name: "dev", RunnerID: "runner"}, missing: true, code: "capability_unavailable"},
		{name: "role", id: "project.register.v1", input: register, observer: true, code: "operator_required"},
		{name: "runner-policy", id: "project.register.v1", input: register, code: "runner_rejected"},
		{name: "bad-input", id: "app.policy.v1", input: map[string]any{"name": "dev", "runner_id": "runner", "unknown": "field"}, code: "invalid_input"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observer.Store(tc.observer)
			missing.Store(tc.missing)
			input, _ := json.Marshal(tc.input)
			cmd := coreTestCommand(t, "--config-dir", configDir, "call", tc.id, "--input-json", string(input))
			var diagnostic bytes.Buffer
			cmd.Stderr = &diagnostic
			cli, err := cmd.Output()
			if (err != nil) != (tc.code != "") {
				t.Fatalf("CLI error %v %s", err, diagnostic.String())
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + tc.id, Arguments: json.RawMessage(input)})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bytes.TrimSpace(cli), []byte(result.Content[0].(*mcp.TextContent).Text)) {
				t.Fatalf("CLI/MCP diverged:\n%s\n%+v", cli, result)
			}
			var envelope service.Result
			if err = json.Unmarshal(cli, &envelope); err != nil {
				t.Fatal(err)
			}
			if tc.code != "" && (envelope.Error == nil || envelope.Error.Code != tc.code) {
				t.Fatalf("error code %+v", envelope)
			}
			if bytes.Contains(cli, []byte("private cookie")) {
				t.Fatal("diagnostic leaked")
			}
		})
	}
	if writes.Load() != 2 {
		t.Fatalf("unexpected mutations %d", writes.Load())
	}
}
