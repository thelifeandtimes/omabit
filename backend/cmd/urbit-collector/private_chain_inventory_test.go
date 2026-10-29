package main

import (
	"bytes"
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/privatechain"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestPrivateChainProjectsCLIReadOnly(t *testing.T) {
	reads := 0
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		if q.Method != "GET" {
			t.Error("inventory mutated", q.Method)
		}
		if q.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Actor: "observer", AccessRole: "observer", Features: []string{protocol.PrivateChainInventoryFeature}})
			return
		}
		if q.URL.Path != protocol.PrivateChainProjectsPath {
			t.Error(q.URL.Path)
		}
		reads++
		reply(t, w, privatechain.Inventory{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: time.Now().UTC(), Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []privatechain.Reservation{}})
	})
	registry := filepath.Join(t.TempDir(), "registry")
	var out, errs bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "lab", "--socket", socket}, &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	for _, format := range []string{"json", "text"} {
		out.Reset()
		errs.Reset()
		if code := run(context.Background(), []string{"--config-dir", registry, "--runner", "lab", "--expected-runner-id", "runner", "--format", format, "private-chain-projects"}, &out, &errs); code != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "text" && !strings.Contains(out.String(), "No private-chain projects reserved.") {
			t.Fatal(out.String())
		}
	}
	if reads != 2 {
		t.Fatal(reads)
	}
	for _, args := range [][]string{{"private-chain-projects"}, {"--runner", "lab", "private-chain-projects"}, {"--runner", "lab", "--expected-runner-id", "runner", "private-chain-projects", "start"}} {
		if code := run(context.Background(), args, &out, &errs); code != 2 {
			t.Fatal("invalid selection accepted", code)
		}
	}
	out.Reset()
	errs.Reset()
	if code := run(context.Background(), []string{"private-chain-projects", "--help"}, &out, &errs); code != 0 || !strings.Contains(out.String(), "Observer access") {
		t.Fatal(code, errs.String())
	}
}
