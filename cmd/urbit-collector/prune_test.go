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

func TestPruneCLIRequiresConfirmationAndExactCapability(t *testing.T) {
	for _, mode := range []string{"supported", "unsupported", "unconfirmed"} {
		t.Run(mode, func(t *testing.T) {
			var requests, mutations atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path == "/hello" {
					features := []string{"epoch-inventory"}
					if mode == "supported" {
						features = append(features, "epoch-prune-genesis")
					}
					reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: features})
					return
				}
				mutations.Add(1)
				var req protocol.OperationRequest
				if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Kind != "prune-epochs" || req.EpochCutoff != "0i0" || req.InventoryDigest != strings.Repeat("d", 64) || !req.ConfirmHistoryLoss || req.ExpectedRevision != 2 || req.RequestID != "prune" {
					t.Errorf("wrong confirmation binding: %+v", req)
				}
				reply(t, w, protocol.Operation{ID: "prune-op", Kind: "prune-epochs", Status: "queued"})
			})
			args := []string{"--socket", socket, "epoch-prune", "alpha", "--revision", "2", "--expected-digest", strings.Repeat("d", 64), "--cutoff", "0i0", "--request-id", "prune"}
			if mode != "unconfirmed" {
				args = append(args, "--confirm-history-loss")
			}
			var out, errout bytes.Buffer
			code := run(context.Background(), args, &out, &errout)
			switch mode {
			case "supported":
				if code != 0 || requests.Load() != 2 || mutations.Load() != 1 {
					t.Fatalf("prune sent unexpected requests/retries: %d %s", code, &errout)
				}
			case "unsupported":
				if code != 1 || mutations.Load() != 0 {
					t.Fatal("unsupported prune submitted")
				}
			case "unconfirmed":
				if code != 2 || requests.Load() != 0 {
					t.Fatal("unconfirmed prune contacted runner")
				}
			}
		})
	}
}

func TestPreviewEnablesOnlyAdvertisedQualifiedGenesisScope(t *testing.T) {
	inv := &epochs.Inventory{Source: epochs.Source, PierID: strings.Repeat("a", 32), ObservedAt: time.Now().UTC(), Digest: strings.Repeat("d", 64), Entries: []epochs.Entry{
		{Name: "0i0", Digest: strings.Repeat("0", 64), FileCount: 4},
		{Name: "0i100", Digest: strings.Repeat("1", 64), FileCount: 5, SnapshotBytes: 100},
		{Name: "0i200", Digest: strings.Repeat("2", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
		{Name: "0i300", Digest: strings.Repeat("3", 64), FileCount: 5, SnapshotBytes: 100, Protected: true},
	}}
	socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("preview mutated runner")
			w.WriteHeader(405)
			return
		}
		if r.URL.Path == "/hello" {
			reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Features: []string{"epoch-inventory", "epoch-prune-genesis"}})
			return
		}
		reply(t, w, protocol.Snapshot{RunnerID: "runner-test", Instances: []protocol.Instance{{ID: "alpha", ConfigRevision: 2, DesiredState: "stopped", ObservedState: "stopped", Epochs: inv}}})
	})
	for _, cutoff := range []string{"0i0", "0i100"} {
		var out, errout bytes.Buffer
		code := run(context.Background(), []string{"--socket", socket, "epoch-preview", "alpha", "--revision", "2", "--expected-digest", inv.Digest, "--cutoff", cutoff}, &out, &errout)
		var p epochs.Preview
		if code != 0 || json.Unmarshal(out.Bytes(), &p) != nil || p.DeletionSupported != (cutoff == "0i0") {
			t.Fatalf("preview scope: %s %d %s %s", cutoff, code, &out, &errout)
		}
	}
}

func TestNonGenesisPruneRequiresPrefixCapability(t *testing.T) {
	for _, capability := range []string{"epoch-prune-genesis", "epoch-prune-prefix"} {
		t.Run(capability, func(t *testing.T) {
			var mutations atomic.Int32
			socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/hello" {
					reply(t, w, protocol.Hello{ProtocolVersion: 1, Features: []string{capability}})
					return
				}
				mutations.Add(1)
				var req protocol.OperationRequest
				if json.NewDecoder(r.Body).Decode(&req) != nil || req.EpochCutoff != "0i100" || !req.ConfirmHistoryLoss {
					t.Error("prefix confirmation/cutoff changed")
				}
				reply(t, w, protocol.Operation{ID: "prefix-op", Kind: "prune-epochs", Status: "queued"})
			})
			var out, errout bytes.Buffer
			code := run(context.Background(), []string{"--socket", socket, "epoch-prune", "alpha", "--revision", "2", "--cutoff", "0i100", "--expected-digest", strings.Repeat("d", 64), "--request-id", "prefix", "--confirm-history-loss"}, &out, &errout)
			if capability == "epoch-prune-prefix" {
				if code != 0 || mutations.Load() != 1 {
					t.Fatalf("supported prefix: %d %s", code, &errout)
				}
			} else if code != 1 || mutations.Load() != 0 {
				t.Fatal("genesis-only runner received a broader deletion")
			}
		})
	}
}
