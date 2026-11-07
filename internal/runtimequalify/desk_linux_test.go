//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeskRuntimeUsesPinnedPrivateProcessAndStdin(t *testing.T) {
	for _, variant := range []string{"ok", "wrong-container", "quarantined", "wrong-identity", "port-changed-before", "port-changed-after", "process-changed", "compiler-error", "transport-error", "missing-input-driver"} {
		t.Run(variant, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			var err error
			state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			ports := filepath.Join(managedPier(f, state), ".http.ports")
			if err = os.WriteFile(ports, []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			count, effects := 0, 0
			f.d.executeInput = func(ctx context.Context, args, env []string, input []byte) (string, error) {
				count++
				joined := strings.Join(args, " ")
				for _, required := range []string{"container exec --interactive " + id, "--disable", "--noproxy *", "--proxy ", "--data-binary @-", "http://127.0.0.1:12322"} {
					if !strings.Contains(joined, required) {
						t.Fatal("unsafe desk command", required)
					}
				}
				if strings.Contains(joined, "kiln-commit") || strings.Contains(joined, "base64") || strings.Contains(joined, "conn.sock") {
					t.Fatal("payload leaked into argv")
				}
				var request struct {
					Source map[string]json.RawMessage `json:"source"`
					Sink   map[string]json.RawMessage `json:"sink"`
				}
				if json.Unmarshal(input, &request) != nil {
					t.Fatal("invalid request")
				}
				if raw, ok := request.Source["dojo"]; ok && string(raw) == `"our"` {
					if variant == "wrong-identity" {
						return `"~nec"`, nil
					}
					if variant == "port-changed-before" {
						os.WriteFile(ports, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600)
					}
					return `"~zod\n"`, nil
				}
				effects++
				switch variant {
				case "port-changed-after":
					os.WriteFile(ports, []byte("8080 insecure public\n12323 insecure loopback\n"), 0600)
				case "process-changed":
					for _, c := range f.containers {
						mapAt(c.doc, "State")["Pid"] = 99999
					}
				case "compiler-error":
					return `"PRIVATE compiler detail"`, nil
				case "transport-error":
					return "PRIVATE", errors.New("PRIVATE")
				}
				return `">="`, nil
			}
			if variant == "missing-input-driver" {
				f.d.executeInput = nil
			}
			selected := id
			if variant == "wrong-container" {
				selected = strings.Repeat("f", 64)
			}
			if variant == "quarantined" {
				state.MaintenanceUncertain = true
			}
			runtime, err := openManagedDesk(context.Background(), state, selected, f.d)
			if err == nil {
				err = runtime.Commit(context.Background(), "omabit-test")
			}
			if variant == "ok" {
				if err != nil || count != 3 || effects != 1 {
					t.Fatal(err, count, effects)
				}
			} else {
				if err == nil || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("unsafe error/result", err)
				}
				if effects > 1 {
					t.Fatal("request replayed")
				}
				switch variant {
				case "wrong-container", "quarantined", "wrong-identity", "port-changed-before", "missing-input-driver":
					if effects != 0 {
						t.Fatal("effect after failed binding")
					}
				}
			}
		})
	}
}

func TestDeskRuntimeDiscoveryRejectsUnknownAutocommitAndBadInventories(t *testing.T) {
	f, state, persist := managedFixture(t)
	state, _, err := applyManaged(context.Background(), "start", state, persist, f.d)
	if err != nil {
		t.Fatal(err)
	}
	id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
	if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var response string
	f.d.executeInput = func(_ context.Context, _, _ []string, input []byte) (string, error) {
		var req struct {
			Source map[string]json.RawMessage `json:"source"`
		}
		if json.Unmarshal(input, &req) != nil {
			t.Fatal("invalid request")
		}
		if string(req.Source["dojo"]) == `"our"` {
			return `"~zod\n"`, nil
		}
		b, _ := json.Marshal(response)
		return string(b), nil
	}
	runtime, err := openManagedDesk(context.Background(), state, id, f.d)
	if err != nil {
		t.Fatal(err)
	}
	for _, reply := range []string{"%.y", "%.n", "compiler failed", "", "~", "%.n extra"} {
		response = reply
		active, err := runtime.AutocommitActive(context.Background())
		if reply == "%.y" || reply == "%.n" {
			if err != nil || active != (reply == "%.y") {
				t.Fatal(reply, active, err)
			}
		} else if err == nil {
			t.Fatal("unknown autocommit treated as inactive", reply)
		}
	}
	atom := func(s string) string {
		b := []byte(s)
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}
		return "0x" + hex.EncodeToString(b)
	}
	response = atom(`["zeta","base"]`)
	names, err := runtime.ListDesks(context.Background())
	if err != nil || len(names) != 2 || names[0] != "base" {
		t.Fatal(names, err)
	}
	response = atom(`["base","base"]`)
	if _, err = runtime.ListDesks(context.Background()); err == nil {
		t.Fatal("duplicate accepted")
	}
	response = atom(`[["app","example","hoon"],["sys","kelvin"]]`)
	paths, err := runtime.FilePaths(context.Background(), "base", 2)
	if err != nil || len(paths) != 2 || paths[0] != "app/example.hoon" {
		t.Fatal(paths, err)
	}
	response = atom(`[["..","escape","hoon"]]`)
	if _, err = runtime.FilePaths(context.Background(), "base", 2); err == nil {
		t.Fatal("unsafe path accepted")
	}
}
