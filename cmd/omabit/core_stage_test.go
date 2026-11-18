package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/apppolicy"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCLIPlanMCPStageLostReplyCLIReceipt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config := filepath.Join(t.TempDir(), "registry")
	r, err := collection.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	source, cache := t.TempDir(), t.TempDir()
	if err := os.Chmod(cache, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "hello.txt"), []byte("reviewed staged bytes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	target := apppolicy.Target{InstanceID: "publisher", Desk: "sample"}
	hello := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "operator", Actor: "fixture", Features: []string{protocol.AppPolicyFeature, protocol.DeskSettingsFeature, protocol.DeskStageFeature}}
	var uploads atomic.Int32
	var cacheMu sync.Mutex
	socket := filepath.Join(t.TempDir(), "r.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set(protocol.Header, "1")
		encode := func(value any) {
			if err := json.NewEncoder(w).Encode(value); err != nil {
				t.Error(err)
			}
		}
		switch {
		case q.URL.Path == "/hello":
			encode(hello)
		case q.URL.Path == protocol.AppPolicyPath && q.Method == "GET":
			encode(protocol.AppPolicyStatus{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, ReadAt: time.Now().UTC(), Policy: apppolicy.Default(), Targets: []protocol.AppPolicyTarget{{Target: target.Ship(), Ship: "~zod", Label: "publisher", Ready: true}}})
		case q.URL.Path == protocol.DeskSettingsPath && q.Method == "POST":
			var got apppolicy.Target
			if json.NewDecoder(q.Body).Decode(&got) != nil || got != target {
				t.Error("wrong source target")
				http.Error(w, "bad", 400)
				return
			}
			encode(protocol.DeskSetting{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Setting: deskwork.DefaultSetting(target), ReadAt: time.Now().UTC()})
		case q.URL.Path == protocol.DeskStagesPath && q.Method == "POST":
			count := uploads.Add(1)
			var intent deskwork.StageIntent
			header, err := base64.RawURLEncoding.DecodeString(q.Header.Get(protocol.DeskStageHeader))
			if err != nil || json.Unmarshal(header, &intent) != nil || intent.Target != target {
				t.Error("wrong staged intent")
				http.Error(w, "bad", 400)
				return
			}
			cacheMu.Lock()
			staged, err := deskwork.CacheStage(q.Context(), cache, intent, "fixture", q.Body)
			cacheMu.Unlock()
			if err != nil {
				t.Error(err)
				http.Error(w, "bad", 400)
				return
			}
			if count == 1 {
				_, _ = w.Write([]byte("{"))
				return
			}
			encode(protocol.DeskStage{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Receipt: staged.Receipt, ReadAt: time.Now().UTC()})
		case strings.HasPrefix(q.URL.Path, protocol.DeskStagesPath+"/") && q.Method == "GET":
			cacheMu.Lock()
			staged, _, err := deskwork.ReadStage(q.Context(), cache, strings.TrimPrefix(q.URL.Path, protocol.DeskStagesPath+"/"), "fixture")
			cacheMu.Unlock()
			if err != nil {
				t.Error(err)
				http.Error(w, "bad", 400)
				return
			}
			encode(protocol.DeskStage{Version: 1, RunnerID: hello.RunnerID, Mode: hello.Mode, Receipt: staged.Receipt, ReadAt: time.Now().UTC()})
		default:
			t.Error("unexpected effect or route", q.Method, q.URL.Path)
			http.NotFound(w, q)
		}
	})}
	go server.Serve(listener)
	defer server.Close()
	if _, err := r.Enroll(ctx, "dev", socket); err != nil {
		t.Fatal(err)
	}
	cli := func(id string, input any, approve bool) json.RawMessage {
		t.Helper()
		body, _ := json.Marshal(input)
		args := []string{"--config-dir", config, "call", id, "--input-json", string(body)}
		if approve {
			args = append(args, "--allow-source", "repo:"+source)
		}
		cmd := coreTestCommand(t, args...)
		var diagnostic bytes.Buffer
		cmd.Stderr = &diagnostic
		data, err := cmd.Output()
		if err != nil {
			t.Fatalf("CLI failed: %v %s %s", err, diagnostic.String(), data)
		}
		var envelope struct {
			Data  json.RawMessage  `json:"data"`
			Error *service.Problem `json:"error"`
		}
		if json.Unmarshal(data, &envelope) != nil || envelope.Error != nil {
			t.Fatalf("invalid CLI result %s", data)
		}
		return envelope.Data
	}
	selected := service.Target{Name: "dev", RunnerID: "runner"}
	var plan service.SourceStagePlan
	if err := json.Unmarshal(cli("desk.plan-stage.v1", service.SourceStageInput{DeskInput: service.DeskInput{Target: selected, InstanceID: target.InstanceID, Desk: target.Desk}, Source: service.SourceSelection{Root: "repo", Directory: "."}}, true), &plan); err != nil {
		t.Fatal(err)
	}
	if uploads.Load() != 0 {
		t.Fatal("planning transferred source")
	}
	transport := &mcp.CommandTransport{Command: coreTestCommand(t, "--config-dir", config, "mcp", "serve", "--allow-runner", "dev:runner", "--enable-mutations", "--allow-source", "repo:"+source)}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "stage-fixture", Version: "1"}, nil).Connect(ctx, transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.desk.stage.v1", Arguments: service.SourceStageSubmit{Target: selected, Plan: plan}})
	if err != nil || !result.IsError || uploads.Load() != 1 {
		t.Fatal("lost reply did not preserve uncertain single upload", result, err, uploads.Load())
	}
	var original protocol.DeskStage
	if err := json.Unmarshal(cli("desk.stage-status.v1", service.StageInput{Target: selected, StageID: plan.Plan.Intent.ID}, false), &original); err != nil {
		t.Fatal(err)
	}
	if original.Receipt.Intent != plan.Plan.Intent {
		t.Fatal("cross-interface receipt rebound")
	}
	if err := os.WriteFile(filepath.Join(source, "hello.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "omabit.desk.stage.v1", Arguments: service.SourceStageSubmit{Target: selected, Plan: plan}})
	if err != nil || !result.IsError || uploads.Load() != 1 {
		t.Fatal("changed source was retransferred", result, err)
	}
	var changed service.Result
	if json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &changed) != nil || changed.Error == nil || changed.Error.Dispatch != "not-dispatched" {
		t.Fatal("pre-upload refusal lost its definite dispatch boundary", result)
	}
	var still protocol.DeskStage
	if err := json.Unmarshal(cli("desk.stage-status.v1", service.StageInput{Target: selected, StageID: plan.Plan.Intent.ID}, false), &still); err != nil {
		t.Fatal(err)
	}
	if still.Receipt.Intent != original.Receipt.Intent || uploads.Load() != 1 {
		t.Fatal("saved status changed or repeated transfer")
	}
}
