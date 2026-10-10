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

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func TestOfflineMaintenanceCapabilityRequiredBeforeMutation(t *testing.T) {
	for _, kind := range []string{"pack", "meld", "roll"} {
		for _, advertised := range []bool{false, true} {
			name := kind + "/unsupported"
			if advertised {
				name = kind + "/supported"
			}
			t.Run(name, func(t *testing.T) {
				var mutations atomic.Int32
				socket := fakeRunner(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/hello" {
						features := []string{"snapshot", "offline-something-else"}
						if advertised {
							features = append(features, "offline-"+kind)
						}
						reply(t, w, protocol.Hello{ProtocolVersion: protocol.Version, RunnerID: "runner-test", Features: features})
						return
					}
					mutations.Add(1)
					var req protocol.OperationRequest
					if r.Method != "POST" || r.URL.Path != "/v1/instances/alpha/operations" || json.NewDecoder(r.Body).Decode(&req) != nil || req.Kind != kind || req.ExpectedRevision != 2 || req.RequestID != "maintenance" {
						t.Errorf("incorrect maintenance request: %+v", req)
					}
					reply(t, w, protocol.Operation{ID: "op-maintenance", Kind: kind, Status: "queued"})
				})
				var stdout, stderr bytes.Buffer
				code := run(context.Background(), []string{"--socket", socket, "--format", "text", "operate", "alpha", kind, "--revision", "2", "--request-id", "maintenance"}, &stdout, &stderr)
				if advertised {
					if code != 0 || mutations.Load() != 1 || !strings.Contains(stdout.String(), "op-maintenance") {
						t.Fatalf("qualified request: exit=%d %s %s", code, &stdout, &stderr)
					}
				} else if code != 1 || mutations.Load() != 0 || !strings.Contains(stderr.String(), "does not support offline "+kind) {
					t.Fatalf("unqualified request dispatched: exit=%d %s", code, &stderr)
				}
			})
		}
	}
}

func TestMaintenanceTextDistinguishesUnknownAfterMeasurement(t *testing.T) {
	start := time.Date(2026, 9, 25, 17, 0, 0, 0, time.UTC)
	finish := start.Add(17 * time.Second)
	m := &protocol.MaintenanceReport{Kind: "meld", BeforePierBytes: 1000, StartedAt: start, MemAvailableBytes: 6 << 30, DiskAvailableBytes: 10 << 30}
	var text bytes.Buffer
	out := newOutput(&text, "text", false)
	if err := out.Encode(protocol.Instance{ID: "alpha", Maintenance: m}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "after=unknown") || !strings.Contains(text.String(), "6.00 GiB") {
		t.Fatalf("missing freshness/evidence distinction: %s", &text)
	}
	m.FinishedAt, m.AfterPierBytes = &finish, 500
	text.Reset()
	if err := out.Encode(protocol.Instance{ID: "alpha", Maintenance: m}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "after=500") || !strings.Contains(text.String(), "17.00 seconds") {
		t.Fatalf("completed measurement missing: %s", &text)
	}
}
