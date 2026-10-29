//go:build linux && amd64

package runtimequalify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyedBootGracefulExitUsesHoodAndNeverForceKills(t *testing.T) {
	for _, mode := range []string{"clean", "lost-response", "wrong-identity", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			f, state, journal := bootAdmissionFixture(t, true)
			state, _, err := applyManaged(context.Background(), "start", state, journal.save, f.d)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(managedPier(f, state), ".http.ports"), []byte("8080 insecure public\n12321 insecure loopback\n"), 0600); err != nil {
				t.Fatal(err)
			}
			execute := f.d.execute
			calls := 0
			f.d.execute = func(ctx context.Context, args, env []string) (string, error) {
				joined := strings.Join(args, " ")
				if strings.Contains(joined, "container stop") || strings.Contains(joined, "container kill") {
					t.Fatal("keyed bootstrap used signal or automatic kill")
				}
				if strings.Contains(joined, "--data-binary") {
					if strings.Contains(joined, `"dojo":"our"`) {
						ship := state.NativeShip
						if mode == "wrong-identity" {
							ship = "~nec"
						}
						b, _ := json.Marshal(ship)
						return string(b), nil
					}
					if !strings.Contains(joined, `"dojo":"+hood/exit"`) || !strings.Contains(joined, `"app":"hood"`) {
						t.Fatal("unqualified Lens payload")
					}
					calls++
					if mode != "timeout" {
						for _, c := range f.containers {
							mapAt(c.doc, "State")["Running"] = false
							mapAt(c.doc, "State")["ExitCode"] = 0
						}
					}
					if mode == "lost-response" {
						return "", errors.New("reply lost")
					}
					return "", nil
				}
				return execute(ctx, args, env)
			}
			_, obs, err := applyManaged(context.Background(), "stop", state, journal.save, f.d)
			if mode == "clean" || mode == "lost-response" {
				if err != nil || obs.State != "stopped" || calls != 1 {
					t.Fatal(obs, err, calls)
				}
			} else {
				if err == nil {
					t.Fatal("false stop success")
				}
				if mode == "wrong-identity" && calls != 0 {
					t.Fatal("unverified identity stopped")
				}
			}
		})
	}
}
