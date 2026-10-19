package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/hostcompat"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestHostCheckCollectorOnlyCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("DOCKER_HOST", "tcp://unused.invalid:2375")
	var scenario atomic.Int32
	var reports atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, req *http.Request) {
		s := scenario.Load()
		if req.Method != "GET" {
			t.Error("host check sent a mutation", req.Method)
		}
		if req.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", AccessRole: "observer", Features: []string{protocol.HostCompatibilityFeature}}
			if s == 1 {
				h.Features = nil
			}
			if s == 2 {
				h.RunnerID = "replacement"
			}
			reply(t, w, h)
			return
		}
		if req.URL.Path != protocol.HostCompatibilityPath {
			t.Error("host check queried inventory or mutation route", req.URL.Path)
		}
		reports.Add(1)
		if s == 3 {
			w.WriteHeader(503)
			reply(t, w, protocol.ErrorResponse{Error: protocol.APIError{Code: "secret-remote", Message: "secret-remote"}})
			return
		}
		r := hostcompat.Inspect(fstest.MapFS{}, "linux", "amd64")
		r.InstalledEngineVersion = "29.9.1"
		r.Engine = &hostcompat.Engine{Version: "29.7.2", APIVersion: "1.54", MinAPIVersion: "1.44", Rootless: true, CgroupDriver: "systemd", CgroupVersion: "2", DataRoot: "/private/engine-data"}
		r = protocol.PublicHostCompatibility(hostcompat.Evaluate(r, "/private/engine-data"))
		reply(t, w, protocol.HostCompatibilityReport{SchemaVersion: 1, ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.HostCompatibilityFeature}, Report: r, ExpiresAt: r.ObservedAt.Add(protocol.HostCompatibilityMaxAge)})
	})
	dir := filepath.Join(t.TempDir(), "collector")
	invoke := func(want int, args ...string) (string, string) {
		t.Helper()
		var out, errOut bytes.Buffer
		code := run(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &errOut)
		if code != want {
			t.Fatal(args, code, out.String(), errOut.String())
		}
		return out.String(), errOut.String()
	}
	invoke(0, "enroll", "remote", "--socket", socket)
	out, errOut := invoke(0, "--runner", "remote", "host-check", "--format", "json")
	var report protocol.HostCompatibilityReport
	if json.Unmarshal([]byte(out), &report) != nil || report.RunnerID != "runner" || report.Report.CanMutate || report.Report.Engine == nil || report.Report.Engine.Version != "29.7.2" || !report.Report.CanStop || !report.Report.RestartPending || errOut != "" {
		t.Fatal(out, errOut)
	}
	for _, args := range [][]string{{"--format", "text", "--runner", "remote", "host-check"}, {"--runner", "remote", "host-check", "--format", "text"}, {"--socket", socket, "host-check", "--format", "text"}} {
		out, errOut = invoke(0, args...)
		if !strings.Contains(out, "Runner: runner") || !strings.Contains(out, "Running engine: 29.7.2") || !strings.Contains(out, "Installed engine: 29.9.1; restart pending: true") || !strings.Contains(out, "stop=true; mutate=false") || errOut != "" {
			t.Fatal(out, errOut)
		}
	}
	before := reports.Load()
	_, errOut = invoke(1, "--runner", "remote", "--expected-runner-id", "wrong", "host-check")
	if !strings.Contains(errOut, "runner_identity_mismatch") || reports.Load() != before {
		t.Fatal(errOut)
	}
	for s, code := range map[int32]string{1: "host_report_unsupported", 2: "runner_identity_mismatch", 3: "host_report_unavailable"} {
		scenario.Store(s)
		before := reports.Load()
		out, errOut := invoke(1, "--runner", "remote", "host-check")
		if out != "" || !strings.Contains(errOut, code) || strings.Contains(errOut, "secret-remote") {
			t.Fatal(out, errOut)
		}
		if s != 3 && reports.Load() != before {
			t.Fatal("unsupported/replaced runner queried")
		}
	}
}

func TestHostCheckCollectorUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"host-check", "--help"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "no local Docker") {
		t.Fatal(code, out.String(), errOut.String())
	}
	for _, args := range [][]string{{"--help"}, {"host-check", "--help"}} {
		out.Reset()
		errOut.Reset()
		if code := run(context.Background(), args, &out, &errOut); code != 0 {
			t.Fatal(code, errOut.String())
		}
		for _, phrase := range []string{"host-check", "verify-host-packages", "repair-engine-guard", "before or after"} {
			if !strings.Contains(out.String(), phrase) {
				t.Fatal("missing documented interface", phrase, out.String())
			}
		}
	}
	for _, args := range [][]string{{"host-check"}, {"host-check", "extra"}, {"--runner", "one", "host-check", "--live"}, {"--runner", "one", "host-check", "--fetch-missing"}, {"--runner", "one", "--fetch-missing", "host-check"}, {"--runner", "one", "host-check", "--verify-host-packages"}, {"--runner", "one", "host-check", "--format", "xml"}, {"--runner", "one", "--socket", "/tmp/r.sock", "host-check"}} {
		out.Reset()
		errOut.Reset()
		if code := run(context.Background(), args, &out, &errOut); code != 2 || out.Len() != 0 {
			t.Fatal(args, code, out.String(), errOut.String())
		}
	}
}
