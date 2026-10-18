package main

import (
	"bytes"
	"context"
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeAllShipsConfigureCapability(t *testing.T) {
	for _, mode := range []string{"fake", "comet", "moon", "planet", "star", "galaxy", ""} {
		for _, updated := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/old", true: "/new"}[updated], func(t *testing.T) {
				writes, reads := 0, 0
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hello" {
						features := []string{"runtime-settings"}
						if updated {
							features = append(features, "native-control-all-ships")
						}
						reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: features})
						return
					}
					if r.URL.Path == "/v1/snapshot" {
						reads++
						reply(t, w, protocol.Snapshot{Instances: []protocol.Instance{{ID: "alpha", Mode: mode, ConfigRevision: 1}}})
						return
					}
					writes++
					reply(t, w, protocol.Instance{ID: "alpha"})
				})
				var out, errs bytes.Buffer
				code := run(context.Background(), []string{"--socket", socket, "configure", "alpha", "--revision=1", "--http-port=18080", "--ames-port=34343", "--loom=31", "--native-control=true"}, &out, &errs)
				allowed := updated || mode == "fake"
				if allowed {
					if code != 0 || writes != 1 {
						t.Fatal("supported configuration failed", code, writes, errs.String())
					}
				} else if code != 1 || writes != 0 || !strings.Contains(errs.String(), "native-control-all-ships") {
					t.Fatal("old runner not gated before mutation", code, writes, errs.String())
				}
				if !updated && reads != 1 {
					t.Fatal("old backend mode was not checked", reads)
				}
			})
		}
	}
}

func TestNativeAllShipsCreationCLI(t *testing.T) {
	for _, kind := range []string{"create-comet", "create-ship"} {
		for _, updated := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/old", true: "/new"}[updated], func(t *testing.T) {
				writes := 0
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hello" {
						features := []string{"runtime-settings", "create-comet", "key-boot-options", "bootstrap-selection"}
						if updated {
							features = append(features, "native-control-all-ships")
						}
						reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-all", Mode: "development-managed", AccessRole: "operator", Features: features})
						return
					}
					writes++
					var body struct {
						Config protocol.Configuration `json:"config"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body.Config.Loom != 34 || !body.Config.NativeControl {
						t.Error("explicit settings lost", body.Config)
					}
					id, operation := "alpha", "register-comet"
					if kind == "create-ship" {
						id, operation = "zod", "register-ship"
					}
					reply(t, w, protocol.Operation{ID: "op-one", InstanceID: id, RequestID: "one", Kind: operation, Status: "succeeded", ConfigRevision: 1})
				})
				dir := filepath.Join(t.TempDir(), "collector")
				invoke := func(args ...string) (int, string) {
					var out, errs bytes.Buffer
					code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
					return code, errs.String()
				}
				if code, err := invoke("enroll", "host", "--socket", socket); code != 0 {
					t.Fatal(err)
				}
				args := []string{"--runner", "host", "--expected-runner-id", "runner-all", kind}
				if kind == "create-ship" {
					keyfile := filepath.Join(t.TempDir(), "missing.key")
					if updated {
						if err := os.WriteFile(keyfile, []byte("0w"+strings.Repeat("a", 128)), 0600); err != nil {
							t.Fatal(err)
						}
					}
					args = append(args, "--ship=~zod", "--key-file="+keyfile)
				} else {
					args = append(args, "alpha", "--http-port=18080")
				}
				args = append(args, "--request-id=one", "--confirm-network", "--loom=34", "--native-control")
				if !updated {
					args = append(args, "--bootstrap=recommended", "--pill=/missing-sensitive-pill")
				}
				code, err := invoke(args...)
				if updated {
					if code != 0 || writes != 1 {
						t.Fatal("creation failed", code, writes, err)
					}
				} else if code != 1 || writes != 0 || !strings.Contains(err, "native-control-all-ships") {
					t.Fatal("must gate before key/pill read or upload", code, writes, err)
				}
			})
		}
	}
}
