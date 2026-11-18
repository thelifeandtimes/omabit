package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectflow"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/projectnetwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/runtimecandidate"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIBeginMCPAdvanceLostReplyAndCLIStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	registry, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := projectnetwork.Build(projectnetwork.Request{Label: "empty bounded", Members: []projectnetwork.MemberRequest{}, ConfirmSharedAdministration: true}, projectnetwork.Reservations{})
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := plan.Digest()
	created := time.Now().UTC().Add(-time.Second)
	features := []string{protocol.ProjectRegistrationFeature, protocol.ProjectStatusFeature, protocol.CreationReplayFeature, protocol.ProjectDependenciesFeature}
	for _, step := range projectflow.Steps(plan) {
		feature, _, _ := protocol.ProjectAction(step.Kind)
		features = append(features, feature)
	}
	hello := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Actor: "actor", Features: features}
	reservation := protocol.ProjectReservation{ID: "project", Revision: 1, Label: plan.Label, CreatedAt: created, ReviewDigest: digest, Members: []protocol.ProjectMemberReservation{}}
	var mu sync.Mutex
	operations := []protocol.Operation{}
	registrations, actions, downloads := 0, 0, 0
	socket := filepath.Join(t.TempDir(), "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set(protocol.Header, "1")
		encode := func(value any) {
			if err := json.NewEncoder(w).Encode(value); err != nil {
				t.Error(err)
			}
		}
		switch {
		case q.URL.Path == "/hello":
			encode(hello)
		case q.URL.Path == protocol.ProjectInventoryPath && q.Method == "POST":
			var request protocol.ProjectRegisterRequest
			if json.NewDecoder(q.Body).Decode(&request) != nil || request.RequestID != "parent" || request.ReviewDigest != digest {
				t.Error("registration changed")
				http.Error(w, "bad", 400)
				return
			}
			if len(operations) == 0 {
				if q.Header.Get(protocol.CreationReplayHeader) == "true" {
					t.Error("lookup created reservation")
					http.Error(w, "absent", 404)
					return
				}
				registrations++
				operations = append(operations, protocol.Operation{ID: "reservation", ProjectID: "project", RequestID: "parent", Actor: "actor", Kind: "register-project", Status: "succeeded", ConfigRevision: 1, CreatedAt: created, FinishedAt: &created})
			} else if q.Header.Get(protocol.CreationReplayHeader) != "true" {
				t.Error("continuation lost reservation lookup guard")
			}
			encode(protocol.ProjectRegistrationResult{SchemaVersion: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, ReviewDigest: digest, ReservationOnly: true, Operation: operations[0]})
		case q.URL.Path == protocol.ProjectStatusPath("project") && q.Method == "GET":
			s := protocol.ProjectStatus{SchemaVersion: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, ReadAt: time.Now().UTC(), Basis: "stored-operation-history", RuntimeObservation: "not-performed", Project: reservation, Operations: []protocol.ProjectOperationSummary{}}
			for i := len(operations) - 1; i >= 0; i-- {
				op := operations[i]
				s.Operations = append(s.Operations, protocol.ProjectOperationSummary{ID: op.ID, RequestID: op.RequestID, Actor: op.Actor, Kind: op.Kind, Status: op.Status, InstanceID: op.InstanceID, ConfigRevision: 1, CreatedAt: op.CreatedAt, FinishedAt: op.FinishedAt})
			}
			encode(s)
		case q.URL.Path == protocol.ProjectDependenciesPath && q.Method == "POST":
			downloads++
			encode(protocol.ProjectDependenciesResult{SchemaVersion: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Image: runtimecandidate.Image, Pill: protocol.BootPill{SHA256: runtimecandidate.PillSHA256, Bytes: runtimecandidate.PillBytes}})
		case q.URL.Path == protocol.ProjectActionPath("project") && q.Method == "POST":
			var request protocol.ProjectActionRequest
			if json.NewDecoder(q.Body).Decode(&request) != nil || !protocol.ValidProjectActionRequest(request) {
				t.Error("invalid phase")
				http.Error(w, "bad", 400)
				return
			}
			for _, prior := range operations {
				if prior.RequestID == request.RequestID {
					t.Error("phase resubmitted")
					http.Error(w, "repeated", 409)
					return
				}
			}
			actions++
			_, kind, _ := protocol.ProjectAction(request.Kind)
			op := protocol.Operation{ID: request.RequestID, ProjectID: "project", RequestID: request.RequestID, Actor: "actor", Kind: kind, Status: "queued", ConfigRevision: 1, CreatedAt: time.Now().UTC()}
			operations = append(operations, op)
			if actions == 1 {
				_, _ = w.Write([]byte("{"))
				return
			}
			encode(protocol.ProjectActionResult{SchemaVersion: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, ReviewDigest: digest, Operation: op})
		default:
			t.Error("unexpected route/effect", q.Method, q.URL.Path)
			http.NotFound(w, q)
		}
	})}
	go server.Serve(listener)
	defer server.Close()
	if _, err := registry.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	target := service.Target{Name: "dev", RunnerID: "runner"}
	selection := service.CreationSelection{Target: target, Plan: plan, RequestID: "parent"}
	input := service.CreationInput{CreationSelection: selection, ConfirmStart: true}
	decode := func(data []byte) (projectflow.Handle, *service.Problem) {
		t.Helper()
		var out struct {
			Data  projectflow.Handle `json:"data"`
			Error *service.Problem   `json:"error"`
		}
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err, string(data))
		}
		return out.Data, out.Error
	}
	cli := func(id string, input any) (projectflow.Handle, *service.Problem) {
		t.Helper()
		body, _ := json.Marshal(input)
		cmd := coreTestCommand(t, "--config-dir", config, "call", id, "--input-json", string(body))
		var diagnostic bytes.Buffer
		cmd.Stderr = &diagnostic
		data, err := cmd.Output()
		handle, problem := decode(data)
		if (err != nil) != (problem != nil) {
			t.Fatal(err, problem, diagnostic.String())
		}
		return handle, problem
	}
	app := input
	app.Workflow = "gall"
	app.AppEffects = []string{"commit"}
	if _, problem := cli("project.create-begin.v1", app); problem == nil || problem.Code != "application_preflight_failed" {
		t.Fatal("missing app API reached creation", problem)
	}
	mu.Lock()
	if registrations != 0 || actions != 0 || downloads != 0 {
		t.Fatal("preflight had effects")
	}
	mu.Unlock()
	h, problem := cli("project.create-begin.v1", input)
	if problem != nil || h.Phase != "reserved" {
		t.Fatal(h, problem)
	}
	advanceInput := service.CreationAdvanceInput{CreationInput: input, NativeRequestID: h.NativeRequestID}
	if _, problem := cli("project.create-advance.v1", input); problem == nil || problem.Code != "invalid_input" {
		t.Fatal("unbound phase reached dispatch", problem)
	}
	connect := func() *mcp.ClientSession {
		t.Helper()
		s, err := mcp.NewClient(&mcp.Implementation{Name: "creation-fixture", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations")}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	session := connect()
	call := func(id string, input any) (projectflow.Handle, *service.Problem) {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit." + id, Arguments: input})
		if err != nil {
			t.Fatal(err)
		}
		return decode([]byte(result.Content[0].(*mcp.TextContent).Text))
	}
	if _, problem := call("project.create-advance.v1", advanceInput); problem == nil || problem.Code != "outcome_uncertain" {
		t.Fatal("lost phase response not uncertain", problem)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	h, problem = cli("project.create-status.v1", selection)
	if problem != nil || h.Phase != "waiting" || h.NativeRequestID != projectflow.RequestID("parent", projectflow.Steps(plan)[0]) {
		t.Fatal("CLI rebound pending native phase", h, problem)
	}
	session = connect()
	defer session.Close()
	waitResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.project.create-wait.v1", Arguments: service.CreationWaitInput{CreationSelection: selection, TimeoutMillis: 0}})
	if err != nil {
		t.Fatal(err)
	}
	var waited struct {
		Data  projectflow.WaitResult `json:"data"`
		Error *service.Problem       `json:"error"`
	}
	if err := json.Unmarshal([]byte(waitResult.Content[0].(*mcp.TextContent).Text), &waited); err != nil || waited.Error != nil || !waited.Data.WaitExpired || waited.Data.Handle.OperationID != h.OperationID {
		t.Fatal("MCP wait rebound or advanced pending work", waited, err)
	}
	h, problem = call("project.create-advance.v1", advanceInput)
	if problem != nil || h.Phase != "waiting" {
		t.Fatal(h, problem)
	}
	mu.Lock()
	if actions != 1 || downloads != 1 || registrations != 1 {
		t.Fatal("disconnect/observe repeated effects", actions, downloads, registrations)
	}
	finished := time.Now().UTC()
	operations[1].Status = "succeeded"
	operations[1].FinishedAt = &finished
	mu.Unlock()
	h, problem = cli("project.create-advance.v1", advanceInput)
	if problem != nil || h.Phase != "completed" {
		t.Fatal("completed old phase was not returned", h, problem)
	}
	mu.Lock()
	if actions != 1 || downloads != 1 {
		t.Fatal("old phase retry advanced another phase", actions, downloads)
	}
	mu.Unlock()
	h, problem = cli("project.create-status.v1", selection)
	if problem != nil || h.Phase != "next-phase" {
		t.Fatal(h, problem)
	}
	advanceInput.NativeRequestID = h.NativeRequestID
	h, problem = cli("project.create-advance.v1", advanceInput)
	if problem != nil || h.Phase != "submitted" || h.Step != "start-anchor" {
		t.Fatal(h, problem)
	}
	mu.Lock()
	finished = time.Now().UTC()
	operations[2].Status = "interrupted"
	operations[2].FinishedAt = &finished
	mu.Unlock()
	h, problem = call("project.create-advance.v1", advanceInput)
	if problem != nil || h.Phase != "needs-recovery" {
		t.Fatal("interruption bypassed", h, problem)
	}
	mu.Lock()
	defer mu.Unlock()
	if actions != 2 || downloads != 1 || registrations != 1 {
		t.Fatal("terminal phase repeated", actions, downloads, registrations)
	}
}
