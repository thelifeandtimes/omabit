//go:build linux

package hostcompat

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func engineFixture(t *testing.T, version string, change func(string, map[string]any), response func(http.ResponseWriter, *http.Request) bool) (string, string, *[]string, *sync.Mutex) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("rootless peer fixture requires an ordinary test UID")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "engine.sock")
	dataRoot := filepath.Join(dir, "data")
	if err := os.Mkdir(dataRoot, 0700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	var mu sync.Mutex
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		if response != nil && response(w, r) {
			return
		}
		var body map[string]any
		switch r.URL.Path {
		case "/version":
			body = map[string]any{"Version": version, "ApiVersion": "1.54", "MinAPIVersion": "1.44"}
		case "/v1.47/info":
			body = map[string]any{"ServerVersion": version, "SecurityOptions": []string{"name=seccomp,profile=builtin", "name=rootless"}, "CgroupDriver": "systemd", "CgroupVersion": "2", "DockerRootDir": dataRoot}
		default:
			http.Error(w, "unexpected path", 400)
			return
		}
		if change != nil {
			change(r.URL.Path, body)
		}
		json.NewEncoder(w).Encode(body)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close(); listener.Close() })
	return socket, dataRoot, &paths, &mu
}

func TestProbePrivateUnixSocketNoEnvironmentProxyAndRequiredAPI(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "http://127.0.0.1:1")
	for _, version := range []string{"29.9.1", "30.0.0"} {
		socket, data, paths, mu := engineFixture(t, version, nil, nil)
		e, err := ProbeEngine(context.Background(), socket, data)
		if err != nil || e.Version != version || !e.Rootless || e.DataRoot != data {
			t.Fatalf("private probe failed: %+v %v", e, err)
		}
		mu.Lock()
		got := strings.Join(*paths, ",")
		mu.Unlock()
		if got != "/version,/v1.47/info" {
			t.Fatalf("unexpected calls %q", got)
		}
	}
}

func TestProbeRefusesUnsafeAPIAndEngineInvariants(t *testing.T) {
	for _, tc := range []struct {
		field string
		value any
	}{
		{"MinAPIVersion", "1.48"}, {"ApiVersion", "2.54"}, {"Version", "credential\nleak"},
		{"ServerVersion", "29.8.0"}, {"SecurityOptions", []string{"name=apparmor"}},
		{"CgroupDriver", "cgroupfs"}, {"CgroupVersion", "1"}, {"DockerRootDir", "/wrong/private/path"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			socket, data, _, _ := engineFixture(t, "29.9.1", func(_ string, b map[string]any) {
				if _, ok := b[tc.field]; ok {
					b[tc.field] = tc.value
				}
			}, nil)
			if _, err := ProbeEngine(context.Background(), socket, data); !errors.Is(err, errProbe) {
				t.Fatalf("unsafe response accepted: %v", err)
			}
		})
	}
}

func TestProbeRejectsRedirectOversizedResponseAndCancellation(t *testing.T) {
	for _, mode := range []string{"redirect", "oversized", "bad-json", "credential-error"} {
		t.Run(mode, func(t *testing.T) {
			socket, data, _, _ := engineFixture(t, "29.9.1", nil, func(w http.ResponseWriter, r *http.Request) bool {
				switch mode {
				case "redirect":
					http.Redirect(w, r, "http://credential:secret@127.0.0.1:1/", http.StatusFound)
				case "oversized":
					w.Write([]byte(strings.Repeat("x", (1<<20)+1)))
				case "bad-json":
					w.Write([]byte("{ malformed"))
				case "credential-error":
					http.Error(w, "secret repository credential", 500)
				}
				return true
			})
			if _, err := ProbeEngine(context.Background(), socket, data); err != errProbe {
				t.Fatalf("response escaped fixed diagnostic: %v", err)
			}
		})
	}
	socket, data, _, _ := engineFixture(t, "29.9.1", nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ProbeEngine(ctx, socket, data); err == nil {
		t.Fatal("canceled probe succeeded")
	}
}

func TestProbeRejectsSymlinksAndWritableBoundaries(t *testing.T) {
	socket, data, _, _ := engineFixture(t, "29.9.1", nil, nil)
	link := filepath.Join(filepath.Dir(socket), "alias.sock")
	if err := os.Symlink(socket, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeEngine(context.Background(), link, data); err == nil {
		t.Fatal("symlink socket accepted")
	}
	if err := os.Chmod(filepath.Dir(socket), 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeEngine(context.Background(), socket, data); err == nil {
		t.Fatal("writable ancestor accepted")
	}
	if err := os.Chmod(filepath.Dir(socket), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(data, 0777); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeEngine(context.Background(), socket, data); err == nil {
		t.Fatal("writable data root accepted")
	}
}
