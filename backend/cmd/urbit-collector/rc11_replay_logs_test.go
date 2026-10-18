package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/client"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestRC11HistoricalCreationCLIReplayContract(t *testing.T) {
	for _, command := range []string{"create-fake", "create-comet", "create-ship"} {
		t.Run(command, func(t *testing.T) {
			ctx := context.Background()
			upgraded, mutations, replayCalls := false, 0, 0
			var accepted map[string]any
			kind := map[string]string{"create-fake": "register-fake", "create-comet": "register-comet", "create-ship": "register-ship"}[command]
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					features := []string{"create-fake", "create-comet", "key-boot-options", "runtime-settings", "bootstrap-selection", "pier-storage"}
					if upgraded {
						features = append(features, protocol.CreationReplayFeature)
					}
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "replay-fixture", Mode: "development-managed", AccessRole: "operator", Features: features})
					return
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if !upgraded {
					mutations++
					accepted = body // model the old server's accepted historical body
				} else {
					replayCalls++
					if r.Header.Get(protocol.CreationReplayHeader) != "true" {
						t.Error("small loom creation lacked replay-only guard")
					}
					if !reflect.DeepEqual(accepted, body) {
						w.WriteHeader(http.StatusConflict)
						reply(t, w, protocol.ErrorResponse{Error: protocol.APIError{Code: "request_conflict", Message: "no exact historical receipt"}})
						return
					}
				}
				reply(t, w, protocol.Operation{ID: "old-receipt", InstanceID: body["id"].(string), RequestID: body["request_id"].(string), Kind: kind, Status: "succeeded"})
			})
			c := client.New(socket)
			defer c.Close()
			cfg := protocol.Configuration{HTTPPort: 18080, AmesPort: 34343, Loom: 29}
			req := protocol.CreateFakeRequest{ID: "alpha", RequestID: "old-request", Config: cfg, Bootstrap: &protocol.Bootstrap{Kind: "bare"}}
			args := []string{command, "alpha", "--http-port=18080", "--request-id=old-request", "--loom=29"}
			var err error
			switch command {
			case "create-fake":
				_, err = c.CreateFake(ctx, req)
			case "create-comet":
				_, err = c.CreateComet(ctx, protocol.CreateCometRequest{CreateFakeRequest: req, ConfirmNetwork: true})
				args = append(args, "--confirm-network")
			case "create-ship":
				keyFile := filepath.Join(t.TempDir(), "fixture.key")
				if err := os.WriteFile(keyFile, []byte("0wfixture.synthetic.only"), 0600); err != nil {
					t.Fatal(err)
				}
				cfg.AmesPort = 0
				_, err = c.CreateShip(ctx, protocol.CreateShipRequest{ID: "sampel-palnet", Ship: "~sampel-palnet", PierPath: "livenet/planets/sampel-palnet/sampel-palnet", RequestID: "old-request", Key: "0wfixture.synthetic.only", Config: cfg, ConfirmNetwork: true})
				args = []string{command, "--ship=~sampel-palnet", "--key-file=" + keyFile, "--http-port=18080", "--request-id=old-request", "--loom=29", "--confirm-network"}
			}
			if err != nil || mutations != 1 {
				t.Fatal("old acceptance failed", err, mutations)
			}
			dir := filepath.Join(t.TempDir(), "collector")
			var out, errs bytes.Buffer
			if code := run(ctx, []string{"--config-dir", dir, "enroll", "fixture", "--socket", socket}, &out, &errs); code != 0 {
				t.Fatal(errs.String())
			}
			prefix := []string{"--config-dir", dir, "--runner", "fixture", "--expected-runner-id", "replay-fixture"}
			invoke := func(want int) {
				t.Helper()
				out.Reset()
				errs.Reset()
				if code := run(ctx, append(prefix, args...), &out, &errs); code != want {
					t.Fatal(code, out.String(), errs.String())
				}
			}
			invoke(1) // old server must never receive a fresh small-loom request
			if mutations != 1 || !strings.Contains(errs.String(), protocol.CreationReplayFeature) {
				t.Fatal("old server was not gated", mutations, errs.String())
			}
			upgraded = true
			invoke(0)
			if !strings.Contains(out.String(), "old-receipt") {
				t.Fatal("historical receipt lost", out.String())
			}
			for i, arg := range args {
				if arg == "--request-id=old-request" {
					args[i] = "--request-id=fresh-request"
				}
			}
			invoke(1)
			if mutations != 1 || replayCalls != 2 {
				t.Fatal("replay mutated historical state", mutations, replayCalls)
			}
		})
	}
}

func TestRC11SmallLoomConfigurationStillRejectedLocally(t *testing.T) {
	for _, loom := range []string{"20", "29"} {
		var errs bytes.Buffer
		if code := run(context.Background(), []string{"--socket", "/missing", "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=" + loom}, io.Discard, &errs); code != 2 || !strings.Contains(errs.String(), "30..34") {
			t.Fatal(code, errs.String())
		}
	}
}

func TestRC11TextLogsSanitizeRemoteControls(t *testing.T) {
	payload := "visible\x1b[2J\x1b]52;c;Zml4dHVyZQ==\a\rspoof\u202ehidden\u2066text\x9b"
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "log-fixture", Mode: "simulation", AccessRole: "operator"})
			return
		}
		reply(t, w, protocol.RuntimeLogs{RunnerID: "log-fixture", InstanceID: "alpha", ConfigRevision: 1, Source: "docker", ContainerID: strings.Repeat("a", 64), ObservedAt: time.Now().UTC(), Lines: []string{payload, "ordinary log line"}})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", dir, "enroll", "fixture", "--socket", socket}, io.Discard, &errs); code != 0 {
		t.Fatal(errs.String())
	}
	if code := run(context.Background(), []string{"--config-dir", dir, "--format=text", "--runner=fixture", "--expected-runner-id=log-fixture", "logs", "alpha", "--revision=1"}, &out, &errs); code != 0 {
		t.Fatal(errs.String())
	}
	for _, r := range out.String() {
		if r != '\n' && (unicode.IsControl(r) || unicode.Is(unicode.Cf, r)) {
			t.Fatal("terminal control escaped local boundary")
		}
	}
	if !strings.Contains(out.String(), "ordinary log line\n") || !strings.Contains(out.String(), "visible") {
		t.Fatal("log content lost")
	}
}
