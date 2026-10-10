package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCodeClipboardHandoffKeepsSecretOutOfArgumentsAndOutput(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "runner.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(protocol.Header, "1")
		if r.URL.Path == "/hello" {
			json.NewEncoder(w).Encode(protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-code", Mode: "simulation", AccessRole: "operator", Features: []string{"login-code"}})
			return
		}
		json.NewEncoder(w).Encode(protocol.LoginCode{RunnerID: "runner-code", InstanceID: "alpha", ConfigRevision: 1, ContainerID: strings.Repeat("a", 64), Code: "aaaaaa-bbbbbb-cccccc-dddddd"})
	})}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	defer func() {
		server.Close()
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	}()
	r, err := collection.Open(filepath.Join(dir, "registry"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Enroll(context.Background(), "one", socket); err != nil {
		t.Fatal(err)
	}
	helper := `#!/bin/sh
test "$#" = 4 && test "$1" = --foreground && test "$2" = --sensitive && test "$3" = --type && test "$4" = text/plain || exit 40
IFS= read -r value || true
test "$value" = aaaaaa-bbbbbb-cccccc-dddddd || exit 41
`
	if err := os.WriteFile(filepath.Join(dir, "wl-copy"), []byte(helper), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WAYLAND_DISPLAY", "fixture-no-real-clipboard")
	reported := false
	output := func(v any) int {
		data, _ := json.Marshal(v)
		if strings.Contains(string(data), "aaaaaa") {
			t.Fatal("code entered output")
		}
		reported = true
		return 0
	}
	fail := func(err error) int { t.Error(err); return 1 }
	if code := copyCode(context.Background(), r, []string{"--runner", "one", "--expected-runner-id", "runner-code", "--instance", "alpha", "--revision", "1", "--container-id", strings.Repeat("a", 64)}, output, fail); code != 0 || !reported {
		t.Fatal("clipboard handoff failed")
	}
	// Exercise the actual child lifecycle with a provider that would otherwise
	// outlive the helper. Parent cancellation must release only this provider.
	if err := os.WriteFile(filepath.Join(dir, "wl-copy"), []byte(helper+"exec /usr/bin/sleep 120\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := time.Now()
	notify := func(v any) int { result := output(v); time.AfterFunc(100*time.Millisecond, cancel); return result }
	if code := copyCode(ctx, r, []string{"--runner", "one", "--expected-runner-id", "runner-code", "--instance", "alpha", "--revision", "1", "--container-id", strings.Repeat("a", 64)}, notify, fail); code != 0 || time.Since(started) > 2*time.Second {
		t.Fatal("clipboard provider did not terminate on cancellation")
	}
}
