package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/epochs"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestCometEpochPreviewCannotInheritFakePruningCapability(t *testing.T) {
	for _, capability := range []string{"epoch-prune-genesis", "epoch-prune-prefix"} {
		for _, supported := range []bool{false, true} {
			t.Run(capability+map[bool]string{false: "/old", true: "/new"}[supported], func(t *testing.T) {
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
						features := []string{"epoch-inventory", capability}
						if supported {
							features = append(features, "comet-epochs")
						}
						reply(t, w, protocol.Hello{ProtocolVersion: 1, RunnerID: "runner-test", Mode: "development-managed", Features: features})
						return
					}
					reply(t, w, protocol.Snapshot{RunnerID: "runner-test", Mode: "development-managed", Instances: []protocol.Instance{{ID: "comet", Mode: "comet", ConfigRevision: 1, DesiredState: "stopped", ObservedState: "stopped", Epochs: inv}}})
				})
				var out, errs bytes.Buffer
				code := run(context.Background(), []string{"--socket", socket, "epoch-preview", "comet", "--revision", "1", "--expected-digest", inv.Digest, "--cutoff", "0i0"}, &out, &errs)
				if !supported {
					if code != 1 || out.Len() != 0 {
						t.Fatal("unqualified comet preview available")
					}
					return
				}
				var p epochs.Preview
				if code != 0 || json.Unmarshal(out.Bytes(), &p) != nil || p.DeletionSupported || len(p.Selected) != 1 || len(p.Retained) != 3 {
					t.Fatalf("comet preview: %d %s %s", code, &out, &errs)
				}
			})
		}
	}
}
