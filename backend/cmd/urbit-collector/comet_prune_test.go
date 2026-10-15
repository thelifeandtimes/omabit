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

func TestCometPrunePreviewRequiresItsCapabilityAndQualifiedLayout(t *testing.T) {
	for _, kind := range []string{"comet", "fake", "shared-comet"} {
		t.Run(kind, func(t *testing.T) {
			mode := "comet"
			if kind == "fake" {
				mode = "fake"
			}
			inv := &epochs.Inventory{Source: epochs.Source, PierID: strings.Repeat("a", 32), ObservedAt: time.Now().UTC(), Digest: strings.Repeat("d", 64), Entries: []epochs.Entry{
				{Name: "0i0", Digest: strings.Repeat("0", 64), FileCount: 4},
				{Name: "0i100", Digest: strings.Repeat("1", 64), FileCount: 5, SnapshotBytes: 100},
				{Name: "0i200", Digest: strings.Repeat("2", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
				{Name: "0i300", Digest: strings.Repeat("3", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
			}}
			if kind == "shared-comet" {
				inv.Entries[1].SharedFiles = true
			}
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Error("preview mutated runner")
					w.WriteHeader(405)
					return
				}
				if r.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Mode: "development-managed", Features: []string{"epoch-inventory", "comet-epochs", "comet-prune-prefix"}})
					return
				}
				reply(t, w, protocol.Snapshot{RunnerID: "runner-test", Mode: "development-managed", Instances: []protocol.Instance{{ID: "selected", Mode: mode, ConfigRevision: 1, DesiredState: "stopped", ObservedState: "stopped", Epochs: inv}}})
			})
			var out, errs bytes.Buffer
			code := run(context.Background(), []string{"--socket", socket, "epoch-preview", "selected", "--revision", "1", "--expected-digest", inv.Digest, "--cutoff", "0i100"}, &out, &errs)
			var p epochs.Preview
			if code != 0 || json.Unmarshal(out.Bytes(), &p) != nil || p.DeletionSupported != (kind == "comet") || len(p.Selected) != 2 || len(p.Retained) != 2 {
				t.Fatalf("preview capability/layout: %d %s %s", code, &out, &errs)
			}
		})
	}
}

func TestCometOnlyPruningCapabilitySubmitsOneBoundRequest(t *testing.T) {
	var mutations atomic.Int32
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Features: []string{"comet-prune-prefix"}})
			return
		}
		mutations.Add(1)
		var req protocol.OperationRequest
		if r.Method != "POST" || r.URL.Path != "/v1/instances/comet/operations" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Kind != "prune-epochs" || req.EpochCutoff != "0i100" || req.InventoryDigest != strings.Repeat("d", 64) || !req.ConfirmHistoryLoss || req.ExpectedRevision != 1 || req.RequestID != "confirmed-prune" {
			t.Error("prune request binding changed", req)
		}
		reply(t, w, protocol.Operation{ID: "prune-op", Kind: "prune-epochs", Status: "queued"})
	})
	var out, errs bytes.Buffer
	code := run(context.Background(), []string{"--socket", socket, "epoch-prune", "comet", "--revision", "1", "--expected-digest", strings.Repeat("d", 64), "--cutoff", "0i100", "--request-id", "confirmed-prune", "--confirm-history-loss"}, &out, &errs)
	if code != 0 || mutations.Load() != 1 {
		t.Fatalf("comet-only prune: %d %s", code, &errs)
	}
}
