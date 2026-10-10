package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestConnectionEditCLIAndExplicitCodeReveal(t *testing.T) {
	posts := 0
	role := "operator"
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-code", Mode: "simulation", AccessRole: role, Features: []string{"login-code"}})
			return
		}
		if r.URL.Path != "/v1/instances/alpha/login-code" {
			t.Error("unexpected endpoint", r.URL.Path)
		}
		posts++
		var request protocol.LoginCodeRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("invalid code request")
		}
		reply(t, w, protocol.LoginCode{RunnerID: "runner-code", InstanceID: "alpha", ConfigRevision: request.ExpectedRevision, ContainerID: request.ContainerID, Code: "aaaaaa-bbbbbb-cccccc-dddddd"})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(expected int, args ...string) string {
		t.Helper()
		var out, errs bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errs)
		if code != expected {
			t.Fatalf("code=%d want=%d error=%s", code, expected, &errs)
		}
		return out.String()
	}
	invoke(0, "enroll", "old", "--socket", socket)
	updated := invoke(0, "runner-edit", "old", "--runner-id", "runner-code", "--revision", "1", "--name", "new", "--socket", socket)
	if !strings.Contains(updated, `"revision": 2`) {
		t.Fatal("revision missing")
	}
	invoke(1, "runner-edit", "new", "--runner-id", "runner-code", "--revision", "1", "--name", "third", "--socket", socket)
	args := []string{"--runner", "new", "--expected-runner-id", "runner-code", "login-code", "alpha", "--revision", "7", "--container-id", strings.Repeat("a", 64)}
	if out := invoke(2, args...); out != "" || posts != 0 {
		t.Fatal("code disclosed without acknowledgement")
	}
	args = append(args, "--reveal")
	if out := invoke(0, args...); !strings.Contains(out, "aaaaaa-bbbbbb") || posts != 1 {
		t.Fatal("explicit reveal failed")
	}
	role = "observer"
	if out := invoke(1, args...); out != "" || posts != 1 {
		t.Fatal("observer reached secret endpoint")
	}
}
