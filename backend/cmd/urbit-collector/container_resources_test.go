package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerResourcesCLI(t *testing.T) {
	for _, supported := range []bool{false, true} {
		calls := 0
		socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/hello" {
				features := []string{"runtime-settings"}
				if supported {
					features = append(features, protocol.ContainerResourcesFeature)
				}
				reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: features})
				return
			}
			calls++
			var req protocol.ConfigureRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Config.MemoryGiB != 16 || req.Config.CPUCores != 8 || !req.PreserveNativeControl {
				t.Error("limits or native preservation lost", req)
			}
			reply(t, w, protocol.Instance{ID: "alpha"})
		})
		var out, errs bytes.Buffer
		code := run(context.Background(), []string{"--socket", socket, "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=31", "--memory-gib=16", "--cpus=8"}, &out, &errs)
		if supported && (code != 0 || calls != 1) || !supported && (code != 1 || calls != 0) {
			t.Fatal(supported, code, calls, errs.String())
		}
	}
	for _, flag := range []string{"--memory-gib", "--cpus"} {
		for _, value := range []string{"0", "17", "256", "-1", "1.5"} {
			var out, errs bytes.Buffer
			code := run(context.Background(), []string{"--socket", "/nonexistent.sock", "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=31", flag + "=" + value}, &out, &errs)
			if code != 2 {
				t.Fatal("invalid limit reached transport", flag, value, code, errs.String())
			}
		}
	}
}

func TestCreationResourceLimitsCLI(t *testing.T) {
	for _, command := range []string{"create-fake", "create-comet", "create-ship"} {
		for _, supported := range []bool{false, true} {
			t.Run(fmt.Sprint(command, supported), func(t *testing.T) {
				posts := 0
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hello" {
						features := []string{"create-fake", "create-comet", "key-boot-options"}
						if supported {
							features = append(features, protocol.ContainerResourcesFeature)
						}
						reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Mode: "development-managed", AccessRole: "operator", Features: features})
						return
					}
					posts++
					var body struct {
						Config protocol.Configuration `json:"config"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body.Config.MemoryGiB != 16 || body.Config.CPUCores != 8 || body.Config.Loom != 31 || body.Config.NativeControl {
						t.Fatal("creation settings lost", body)
					}
					instance, kind := "alpha", "register-fake"
					if command == "create-comet" {
						kind = "register-comet"
					}
					if command == "create-ship" {
						instance, kind = "zod", "register-ship"
					}
					reply(t, w, protocol.Operation{ID: "op-test", InstanceID: instance, RequestID: "one", Kind: kind})
				})
				dir := filepath.Join(t.TempDir(), "collector")
				var out, errs bytes.Buffer
				invoke := func(args ...string) int {
					return run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
				}
				if code := invoke("enroll", "review", "--socket", socket); code != 0 {
					t.Fatal(errs.String())
				}
				args := []string{"--runner", "review", "--expected-runner-id", "runner-test", command}
				if command == "create-ship" {
					key := filepath.Join(t.TempDir(), "key")
					if err := os.WriteFile(key, []byte("0w"+strings.Repeat("a", 128)), 0600); err != nil {
						t.Fatal(err)
					}
					args = append(args, "--ship=~zod", "--key-file="+key, "--confirm-network")
				} else {
					args = append(args, "alpha", "--http-port=18080")
					if command == "create-comet" {
						args = append(args, "--confirm-network")
					}
				}
				args = append(args, "--request-id=one", "--memory-gib=16", "--cpus=8")
				code := invoke(args...)
				if supported && (code != 0 || posts != 1) || !supported && (code != 1 || posts != 0) {
					t.Fatal(code, posts, errs.String())
				}
			})
		}
	}
	for _, command := range []string{"create-fake", "create-comet", "create-ship"} {
		for _, flag := range []string{"--memory-gib", "--cpus"} {
			for _, value := range []string{"0", "17", "256", "-1", "1.5"} {
				var out, errs bytes.Buffer
				args := []string{"--socket", "/nonexistent.sock", command}
				if command != "create-ship" {
					args = append(args, "alpha")
				}
				args = append(args, flag+"="+value)
				if code := run(context.Background(), args, &out, &errs); code != 2 {
					t.Fatal(command, flag, value, code, errs.String())
				}
			}
		}
	}
}
