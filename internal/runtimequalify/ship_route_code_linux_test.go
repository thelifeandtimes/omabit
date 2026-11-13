//go:build linux && amd64

package runtimequalify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/shiproute"
)

func TestShipRouteCodeVerifiesIncarnationBeforeAndAfterPrivateRead(t *testing.T) {
	for _, kind := range []string{"ok", "wrong-binding", "wrong-ship", "code-error", "invalid-code", "replaced-during-code", "process-during-code", "port-during-code"} {
		t.Run(kind, func(t *testing.T) {
			f, state, persist := managedFixture(t)
			var err error
			state, _, err = applyManaged(context.Background(), "start", state, persist, f.d)
			if err != nil {
				t.Fatal(err)
			}
			id := state.Runtime.Containers[len(state.Runtime.Containers)-1].ID
			pier := managedPier(f, state)
			if err = os.WriteFile(filepath.Join(pier, ".http.ports"), []byte("8080 insecure public\n12322 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			codes := 0
			f.d.executeInput = func(_ context.Context, args, env []string, input []byte) (string, error) {
				if strings.Contains(string(input), `"dojo":"our"`) {
					if kind == "wrong-ship" {
						return `"~nec"`, nil
					}
					return `"~zod"`, nil
				}
				if string(input) != `{"source":{"dojo":"+code"},"sink":{"stdout":null}}` {
					t.Fatal("unexpected Lens expression", string(input))
				}
				codes++
				if kind == "code-error" {
					return "", errors.New("PRIVATE-CODE-ERROR")
				}
				if kind == "invalid-code" {
					return `"PRIVATE-CODE-ERROR"`, nil
				}
				if kind == "replaced-during-code" {
					if err := os.Rename(filepath.Join(pier, ".urb"), filepath.Join(pier, ".urb-old")); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(filepath.Join(pier, ".urb"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				for _, c := range f.containers {
					if kind == "process-during-code" {
						mapAt(c.doc, "State")["Pid"] = 99999
					}
					if kind == "port-during-code" {
						mapAt(c.doc, "NetworkSettings")["Ports"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "38099"}}}
					}
				}
				return `"lidlut-tabwed-pillex-ridrup\n"`, nil
			}
			runtime, err := openManagedDesk(context.Background(), state, id, f.d)
			if err != nil {
				t.Fatal(err)
			}
			continuity, err := runtime.SourceIdentity(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			expected := shiproute.Runtime{Ship: "~zod", Continuity: continuity, Origin: fmt.Sprintf("http://127.0.0.1:%d", state.Port)}
			if kind == "wrong-binding" {
				expected.Continuity = strings.Repeat("f", 64)
			}
			code, err := runtime.RouteCode(context.Background(), expected)
			if kind == "ok" {
				if err != nil || code != "lidlut-tabwed-pillex-ridrup" {
					t.Fatal("credential unavailable", err)
				}
			} else if err == nil || code != "" || strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "lidlut") {
				t.Fatal("unsafe credential result", kind)
			}
			want := 1
			if kind == "wrong-binding" || kind == "wrong-ship" {
				want = 0
			}
			if codes != want {
				t.Fatal("credential read count", codes, want)
			}
		})
	}
}
