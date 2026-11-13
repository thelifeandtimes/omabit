//go:build linux && amd64

package runtimequalify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShipRouteIdentityReadsVerifiedNativeIdentityAndHostPort(t *testing.T) {
	for _, kind := range []string{"ok", "wrong-identity", "changed-pier", "changed-process", "changed-port"} {
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
			reads := 0
			f.d.executeInput = func(_ context.Context, args, env []string, input []byte) (string, error) {
				reads++
				if string(input) != `{"source":{"dojo":"our"},"sink":{"stdout":null}}` {
					t.Fatal("route inspection sent an effect", string(input))
				}
				switch kind {
				case "wrong-identity":
					return `"~nec"`, nil
				case "changed-pier":
					if err := os.Rename(filepath.Join(pier, ".urb"), filepath.Join(pier, ".urb-retained")); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(filepath.Join(pier, ".urb"), 0700); err != nil {
						t.Fatal(err)
					}
				case "changed-process":
					for _, c := range f.containers {
						mapAt(c.doc, "State")["Pid"] = 99999
					}
				case "changed-port":
					for _, c := range f.containers {
						mapAt(c.doc, "NetworkSettings")["Ports"] = map[string]any{"8080/tcp": []any{map[string]any{"HostIp": "127.0.0.1", "HostPort": "38099"}}}
					}
				}
				return `"~zod\n"`, nil
			}
			runtime, err := openManagedDesk(context.Background(), state, id, f.d)
			if err != nil {
				t.Fatal(err)
			}
			out, err := runtime.RouteIdentity(context.Background())
			if kind == "ok" {
				if err != nil || out.Validate() != nil || out.Ship != "~zod" || out.Origin != fmt.Sprintf("http://127.0.0.1:%d", state.Port) {
					t.Fatal(out, err)
				}
			} else if err == nil {
				t.Fatal("changed identity accepted", kind, out)
			}
			if reads != 1 || strings.Contains(out.Origin, "12322") {
				t.Fatal("private Lens leaked or identity replayed", reads, out)
			}
		})
	}
}
