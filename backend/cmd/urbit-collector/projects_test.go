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
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestProjectsCLIReadOnlyAndCapabilityBound(t *testing.T) {
	var older atomic.Bool
	var reads atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, q *http.Request) {
		if q.Method != "GET" {
			t.Error("mutation", q.Method)
		}
		if q.URL.Path == "/hello" {
			h := protocol.Hello{ProtocolVersion: 1, RunnerID: "runner", Mode: "simulation", Features: []string{protocol.ProjectInventoryFeature}}
			if older.Load() {
				h.Features = nil
			}
			reply(t, w, h)
			return
		}
		if q.URL.Path != protocol.ProjectInventoryPath {
			t.Error("unexpected route", q.URL.Path)
		}
		reads.Add(1)
		reply(t, w, protocol.ProjectInventory{SchemaVersion: 1, RunnerID: "runner", Mode: "simulation", ReadAt: time.Now().UTC(), Basis: "stored-reservations", RuntimeObservation: "not-performed", Projects: []protocol.ProjectReservation{}})
	})
	registry := filepath.Join(t.TempDir(), "collector")
	var enrollment, enrollmentErr bytes.Buffer
	if code := run(context.Background(), []string{"--config-dir", registry, "enroll", "fixture", "--socket", socket}, &enrollment, &enrollmentErr); code != 0 {
		t.Fatal(code, enrollmentErr.String())
	}
	for _, format := range []string{"json", "text"} {
		var out, errs bytes.Buffer
		code := run(context.Background(), []string{"--config-dir", registry, "--runner", "fixture", "--expected-runner-id", "runner", "--format", format, "projects"}, &out, &errs)
		if code != 0 || errs.Len() != 0 {
			t.Fatal(code, errs.String())
		}
		if format == "json" {
			var v protocol.ProjectInventory
			if json.Unmarshal(out.Bytes(), &v) != nil || v.RuntimeObservation != "not-performed" {
				t.Fatal(out.String())
			}
		} else if !strings.Contains(out.String(), "Saved reservations only.") {
			t.Fatal(out.String())
		}
	}
	older.Store(true)
	var out, errs bytes.Buffer
	if run(context.Background(), []string{"--config-dir", registry, "--runner", "fixture", "--expected-runner-id", "runner", "projects"}, &out, &errs) != 1 || reads.Load() != 2 || !strings.Contains(errs.String(), "project-reservations-v1") {
		t.Fatal("unsupported runner queried", out.String(), errs.String())
	}
}
