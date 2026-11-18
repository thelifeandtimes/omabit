//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGrubberyProcessObservesExactEngineProcessNamespaceAndLaunchContext(t *testing.T) {
	for _, scenario := range []string{"ok", "pid", "start", "engine", "boot", "namespace", "namespace-during-read", "launch", "ports", "invalid-pid", "missing-engine", "pid-during-read", "launch-during-read", "ports-during-read", "engine-during-read"} {
		t.Run(scenario, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			f.engine.ID = "fixture-process-daemon"
			state, _, err := applyManaged(t.Context(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			for _, c := range f.containers {
				mapAt(c.doc, "State")["Pid"] = 4242
				mapAt(c.doc, "State")["StartedAt"] = "2026-01-01T00:00:00Z"
			}
			ports := filepath.Join(managedPier(f, state), ".http.ports")
			if err = os.WriteFile(ports, []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			boot := "11111111-2222-3333-4444-555555555555"
			namespace := "4:1001"
			reads, changeDuring := 0, false
			f.d.readBootID = func() (string, error) { return boot, nil }
			execute := f.d.execute
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				if len(args) >= 4 && reflect.DeepEqual(args[4:], []string{"container", "exec", id, "/bin/stat", "-L", "-c", "%d:%i", "/proc/self/ns/net"}) {
					reads++
					// The second observation has already captured its first full
					// snapshot. Drift here must be caught by its final snapshot or
					// engine bracket, without relying on extra process guards.
					if reads == 4 {
						switch scenario {
						case "pid-during-read", "launch-during-read":
							for _, c := range f.containers {
								if scenario == "pid-during-read" {
									mapAt(c.doc, "State")["Pid"] = 4243
								} else {
									mapAt(c.doc, "HostConfig")["Memory"] = 1
								}
							}
						case "ports-during-read":
							if err := os.WriteFile(ports, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600); err != nil {
								t.Fatal(err)
							}
						case "engine-during-read":
							f.engine.ID = "other-process-daemon"
						}
					}
					if changeDuring && reads%2 == 0 {
						return "4:9999", nil
					}
					return namespace, nil
				}
				return execute(ctx, args, env)
			}
			runtime, err := openManagedDesk(t.Context(), state, id, f.d)
			if err != nil {
				t.Fatal(err)
			}
			original, err := runtime.GrubberyProcess(t.Context())
			if err != nil || original.Validate() != nil || original.ContainerID != id || original.EngineID != f.engine.ID || original.HostBootID != boot || reads != 2 {
				t.Fatal("missing exact trusted observation", original, err, reads)
			}
			raw, _ := json.Marshal(original)
			if strings.Contains(string(raw), f.d.home) || strings.Contains(string(raw), "Entrypoint") || strings.Contains(string(raw), "Env") {
				t.Fatal("private launch context leaked")
			}
			switch scenario {
			case "pid", "invalid-pid":
				for _, c := range f.containers {
					mapAt(c.doc, "State")["Pid"] = 4243
					if scenario == "invalid-pid" {
						mapAt(c.doc, "State")["Pid"] = 1
					}
				}
			case "start":
				for _, c := range f.containers {
					mapAt(c.doc, "State")["StartedAt"] = "2026-11-19T00:00:00Z"
				}
			case "engine":
				f.engine.ID = "other-process-daemon"
			case "missing-engine":
				f.engine.ID = ""
			case "boot":
				boot = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			case "namespace":
				namespace = "4:1002"
			case "namespace-during-read":
				changeDuring = true
			case "launch":
				for _, c := range f.containers {
					mapAt(c.doc, "HostConfig")["Memory"] = 1
				}
			case "ports":
				if err = os.WriteFile(ports, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			fresh, err := runtime.GrubberyProcess(t.Context())
			if scenario == "ok" {
				if err != nil || fresh != original {
					t.Fatal("unchanged process drifted", fresh, err)
				}
			} else if scenario == "boot" || scenario == "namespace" {
				if err != nil || fresh == original || fresh.ContextSHA256 == original.ContextSHA256 {
					t.Fatal("fresh changed namespace/boot was not distinct", fresh, err)
				}
			} else if !errors.Is(err, errDeskProcess) {
				t.Fatal("changed exact selection admitted", scenario, fresh, err)
			}
		})
	}
}
