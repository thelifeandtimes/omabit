package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestEpochPreviewIsReadOnlyBoundToSelectionAndProtectsSuffix(t *testing.T) {
	for _, fault := range []string{"", "digest", "revision", "running", "busy", "stale", "protected", "identity"} {
		t.Run("preview-"+fault, func(t *testing.T) {
			inv := &epochs.Inventory{Source: epochs.Source, PierID: strings.Repeat("a", 32), Digest: strings.Repeat("d", 64), ObservedAt: time.Now().UTC(), Entries: []epochs.Entry{
				{Name: "0i0", FileCount: 3, LogicalBytes: 100, AllocatedBytes: 4096},
				{Name: "0i10", FileCount: 4, LogicalBytes: 200, AllocatedBytes: 8192, Protected: true},
				{Name: "0i20", FileCount: 4, LogicalBytes: 300, AllocatedBytes: 12288, Protected: true},
			}}
			v := protocol.Instance{ID: "alpha", ConfigRevision: 2, DesiredState: "stopped", ObservedState: "stopped", Epochs: inv}
			cutoff, digest := "0i0", inv.Digest
			switch fault {
			case "digest":
				digest = strings.Repeat("e", 64)
			case "revision":
				v.ConfigRevision = 3
			case "running":
				v.ObservedState = "running"
			case "busy":
				v.ActiveOperationID = "operation"
			case "stale":
				inv.ObservedAt = time.Now().Add(-time.Hour)
			case "protected":
				cutoff = "0i10"
			}
			var mutations atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					mutations.Add(1)
					t.Error("preview submitted a mutation")
				}
				if r.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-one", Mode: "development-managed-fake", Features: []string{"epoch-inventory"}})
					return
				}
				id := "runner-one"
				if fault == "identity" {
					id = "runner-other"
				}
				reply(t, w, protocol.Snapshot{RunnerID: id, Mode: "development-managed-fake", Instances: []protocol.Instance{v}})
			})
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), []string{"--socket", socket, "epoch-preview", "alpha", "--revision", "2", "--expected-digest", digest, "--cutoff", cutoff}, &stdout, &stderr)
			if fault != "" {
				if code != 1 || stdout.Len() != 0 {
					t.Fatalf("unsafe preview returned: %d %s %s", code, &stdout, &stderr)
				}
				return
			}
			var p epochs.Preview
			if code != 0 || json.Unmarshal(stdout.Bytes(), &p) != nil || p.RunnerID != "runner-one" || p.ConfigRevision != 2 || len(p.Selected) != 1 || len(p.Retained) != 2 || p.DeletionSupported || !p.RequiresStoppedRevalidation || mutations.Load() != 0 {
				t.Fatalf("preview: %d %s %s", code, &stdout, &stderr)
			}
			var text bytes.Buffer
			if err := newOutput(&text, "text", false).Encode(p); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text.String(), "Selected: 0i0") || !strings.Contains(text.String(), "Deletion supported: false") {
				t.Fatalf("misleading text: %s", &text)
			}
		})
	}
}

func TestEpochCommandsRefuseMissingCapabilityAndExecutionFlags(t *testing.T) {
	var mutations atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			mutations.Add(1)
		}
		reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-one"})
	})
	for _, args := range [][]string{
		{"operate", "alpha", "inspect-epochs", "--revision", "1", "--request-id", "inspect"},
		{"epoch-preview", "alpha", "--revision", "1", "--expected-digest", strings.Repeat("d", 64), "--cutoff", "0i0"},
	} {
		var out, errout bytes.Buffer
		if code := run(context.Background(), append([]string{"--socket", socket}, args...), &out, &errout); code != 1 || mutations.Load() != 0 {
			t.Fatalf("unsupported command dispatched: %d %s", code, &errout)
		}
	}
	for _, args := range [][]string{
		{"epoch-preview", "alpha", "--revision", "1", "--expected-digest", strings.Repeat("d", 64), "--cutoff", "0i0", "--apply"},
		{"epoch-preview", "alpha", "--revision", "1", "--expected-digest", "bad", "--cutoff", "0i0"},
		{"epoch-preview", "alpha", "--revision", "1", "--expected-digest", strings.Repeat("d", 64), "--cutoff", "../log"},
	} {
		var out, errout bytes.Buffer
		if code := run(context.Background(), append([]string{"--socket", socket}, args...), &out, &errout); code != 2 {
			t.Fatalf("bad preview arguments accepted: %d %s", code, &errout)
		}
	}
}
