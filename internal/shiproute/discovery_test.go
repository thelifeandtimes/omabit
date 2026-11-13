package shiproute

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTailscaleDiscoveryUsesOnlyBoundedReadsAndRejectsProfileChange(t *testing.T) {
	for _, kind := range []string{"ok", "status-unavailable", "serve-unavailable", "not-matched", "profile-change", "invalid-json", "case-alias"} {
		t.Run(kind, func(t *testing.T) {
			var calls [][]string
			read := func(ctx context.Context, args ...string) ([]byte, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("unbounded discovery")
				}
				calls = append(calls, append([]string{}, args...))
				if reflect.DeepEqual(args, []string{"status", "--json", "--peers=false"}) {
					if kind == "status-unavailable" {
						return nil, errors.New("PRIVATE")
					}
					if kind == "profile-change" && len(calls) == 3 {
						return []byte(strings.ReplaceAll(serveStatus, "box.", "other.")), nil
					}
					if kind == "case-alias" {
						return []byte(strings.Replace(serveStatus, `"TUN":true`, `"TUN":true,"tun":false`, 1)), nil
					}
					return []byte(serveStatus), nil
				}
				if !reflect.DeepEqual(args, []string{"serve", "status", "--json"}) {
					t.Fatal("unexpected Tailscale command", args)
				}
				if kind == "serve-unavailable" {
					return nil, errors.New("PRIVATE")
				}
				if kind == "not-matched" {
					return []byte(`{}`), nil
				}
				if kind == "invalid-json" {
					return []byte(`PRIVATE`), nil
				}
				body, _ := json.Marshal(serveFixture())
				return body, nil
			}
			out := discoverTailscale(context.Background(), "http://127.0.0.1:8888", read)
			if err := out.Validate(); err != nil {
				t.Fatal(out, err)
			}
			if kind == "ok" {
				if out.Route == nil || out.Network.Serve != "tailnet" || len(calls) != 3 {
					t.Fatal(out, calls)
				}
			} else if out.Route != nil {
				t.Fatal("unsafe route", out)
			}
			body, _ := json.Marshal(out)
			if strings.Contains(string(body), "PRIVATE") {
				t.Fatal("private diagnostic leaked")
			}
			if kind == "profile-change" && out.Network.Tailscale != "unavailable" {
				t.Fatal("profile switch accepted", out)
			}
		})
	}
}
func TestTailscaleCommandReaderHasByteTimeAndEnvironmentBounds(t *testing.T) {
	t.Setenv("TS_SOCKET", "PRIVATE")
	t.Setenv("HTTP_PROXY", "PRIVATE")
	for _, kind := range []string{"ok", "oversized", "error", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			script := "#!/bin/sh\n[ -z \"${TS_SOCKET-}${HTTP_PROXY-}\" ] || exit 81\n[ \"$*\" = 'status --json --peers=false' ] || exit 82\n"
			switch kind {
			case "ok":
				script += "printf '%s' '{\"ok\":true}'\n"
			case "oversized":
				script += "head -c 262145 /dev/zero\n"
			case "error":
				script += "printf PRIVATE >&2\nexit 1\n"
			case "cancel":
				script += "exec sleep 10\n"
			}
			binary := filepath.Join(t.TempDir(), "tailscale-fixture")
			if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			start := time.Now()
			out, err := readTailscale(ctx, binary, "status", "--json", "--peers=false")
			if kind == "ok" {
				if err != nil || string(out) != `{"ok":true}` {
					t.Fatal(string(out), err)
				}
			} else if err == nil || len(out) != 0 || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unsafe observation", len(out), err)
			}
			if time.Since(start) > 3*time.Second {
				t.Fatal("reader exceeded deadline")
			}
		})
	}
}
